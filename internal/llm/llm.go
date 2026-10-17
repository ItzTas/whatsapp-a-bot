package llm

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

const maxToolRounds = 6

type chatModel interface {
	reply(ctx context.Context, system, prompt string, maxTokens int, tools []Tool) (string, error)
}

type Tool struct {
	Name        string
	Description string
	Properties  map[string]any
	Required    []string
	Run         func(ctx context.Context, input json.RawMessage) (string, error)
}

func callTool(ctx context.Context, tools []Tool, name string, input json.RawMessage) string {
	i := slices.IndexFunc(tools, func(t Tool) bool { return t.Name == name })
	if i < 0 {
		return fmt.Sprintf("erro: a ferramenta %q não existe", name)
	}
	out, err := tools[i].Run(ctx, input)
	if err != nil {
		return "erro: " + err.Error()
	}
	return out
}

func toolSchema(t Tool) map[string]any {
	return map[string]any{"type": "object", "properties": toolProperties(t), "required": t.Required}
}

func toolProperties(t Tool) map[string]any {
	if t.Properties == nil {
		return map[string]any{}
	}
	return t.Properties
}

type modelSettings struct {
	api   string
	url   string
	key   string
	model string
}

func modelFromEnv(prefix string, defaults modelSettings) (chatModel, error) {
	s := modelSettings{
		api:   envOr(prefix+"_API", defaults.api),
		url:   envOr(prefix+"_URL", defaults.url),
		key:   os.Getenv(prefix + "_KEY"),
		model: os.Getenv(prefix + "_MODEL"),
	}
	if s.model == "" {
		return nil, nil
	}
	builders := map[string]func(modelSettings) (chatModel, error){
		"anthropic": newAnthropicModel,
		"openai":    newOpenAIModel,
	}
	build, ok := builders[s.api]
	if !ok {
		return nil, fmt.Errorf("%s_API=%q inválido, use anthropic ou openai", prefix, s.api)
	}
	return build(s)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

type anthropicModel struct {
	client anthropic.Client
	model  string
}

func newAnthropicModel(s modelSettings) (chatModel, error) {
	opts := []option.RequestOption{option.WithAPIKey(s.key)}
	if s.url != "" {
		opts = append(opts, option.WithBaseURL(s.url))
	}
	return anthropicModel{client: anthropic.NewClient(opts...), model: s.model}, nil
}

func (m anthropicModel) reply(ctx context.Context, system, prompt string, maxTokens int, tools []Tool) (string, error) {
	params := anthropic.MessageNewParams{
		Model:     m.model,
		MaxTokens: int64(maxTokens),
		System:    []anthropic.TextBlockParam{{Text: system}},
		Messages:  []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock(prompt))},
		Tools:     anthropicTools(tools),
	}
	for round := range maxToolRounds {
		if round == maxToolRounds-1 && len(tools) > 0 {
			params.ToolChoice = anthropic.ToolChoiceUnionParam{OfNone: &anthropic.ToolChoiceNoneParam{}}
		}
		resp, err := m.client.Messages.New(ctx, params)
		if err != nil {
			return "", err
		}
		if resp.StopReason == anthropic.StopReasonRefusal {
			return "", fmt.Errorf("modelo recusou responder (%s)", resp.StopDetails.Category)
		}
		if resp.StopReason != anthropic.StopReasonToolUse {
			return anthropicText(resp), nil
		}
		params.Messages = append(params.Messages, resp.ToParam(), anthropic.NewUserMessage(m.runTools(ctx, resp, tools)...))
	}
	return "", errors.New("o modelo não parou de chamar ferramentas")
}

func (m anthropicModel) runTools(ctx context.Context, resp *anthropic.Message, tools []Tool) []anthropic.ContentBlockParamUnion {
	var results []anthropic.ContentBlockParamUnion
	for _, block := range resp.Content {
		use, ok := block.AsAny().(anthropic.ToolUseBlock)
		if !ok {
			continue
		}
		results = append(results, anthropic.NewToolResultBlock(use.ID, callTool(ctx, tools, use.Name, use.Input), false))
	}
	return results
}

func anthropicText(resp *anthropic.Message) string {
	var sb strings.Builder
	for _, block := range resp.Content {
		if text, ok := block.AsAny().(anthropic.TextBlock); ok {
			sb.WriteString(text.Text)
		}
	}
	return strings.TrimSpace(sb.String())
}

func anthropicTools(tools []Tool) []anthropic.ToolUnionParam {
	params := make([]anthropic.ToolUnionParam, 0, len(tools))
	for _, t := range tools {
		params = append(params, anthropic.ToolUnionParam{OfTool: &anthropic.ToolParam{
			Name:        t.Name,
			Description: anthropic.String(t.Description),
			InputSchema: anthropic.ToolInputSchemaParam{Properties: toolProperties(t), Required: t.Required},
		}})
	}
	return params
}

type openAIModel struct {
	endpoint string
	key      string
	model    string
}

type openAIMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type openAIToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	} `json:"function"`
}

func (c openAIToolCall) input() json.RawMessage {
	var encoded string
	if err := json.Unmarshal(c.Function.Arguments, &encoded); err == nil {
		return json.RawMessage(cmp.Or(encoded, "{}"))
	}
	if len(c.Function.Arguments) == 0 {
		return json.RawMessage("{}")
	}
	return c.Function.Arguments
}

func newOpenAIModel(s modelSettings) (chatModel, error) {
	if s.url == "" {
		return nil, errors.New("a API openai precisa da URL base, ex.: https://api.deepseek.com/v1")
	}
	return openAIModel{endpoint: strings.TrimSuffix(s.url, "/") + "/chat/completions", key: s.key, model: s.model}, nil
}

func (m openAIModel) reply(ctx context.Context, system, prompt string, maxTokens int, tools []Tool) (string, error) {
	messages := []openAIMessage{{Role: "system", Content: system}, {Role: "user", Content: prompt}}
	for round := range maxToolRounds {
		offered := tools
		if round == maxToolRounds-1 {
			offered = nil
		}
		msg, err := m.complete(ctx, messages, maxTokens, offered)
		if err != nil {
			return "", err
		}
		if len(msg.ToolCalls) == 0 {
			return strings.TrimSpace(msg.Content), nil
		}
		messages = append(messages, msg)
		for _, call := range msg.ToolCalls {
			messages = append(messages, openAIMessage{Role: "tool", ToolCallID: call.ID, Content: callTool(ctx, tools, call.Function.Name, call.input())})
		}
	}
	return "", errors.New("o modelo não parou de chamar ferramentas")
}

func (m openAIModel) complete(ctx context.Context, messages []openAIMessage, maxTokens int, tools []Tool) (msg openAIMessage, err error) {
	request := map[string]any{"model": m.model, "max_tokens": maxTokens, "messages": messages}
	if len(tools) > 0 {
		request["tools"] = openAITools(tools)
	}
	body, err := json.Marshal(request)
	if err != nil {
		return msg, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.endpoint, bytes.NewReader(body))
	if err != nil {
		return msg, err
	}
	req.Header.Set("Content-Type", "application/json")
	if m.key != "" {
		req.Header.Set("Authorization", "Bearer "+m.key)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return msg, err
	}
	defer func() { err = errors.Join(err, resp.Body.Close()) }()

	if resp.StatusCode != http.StatusOK {
		detail, readErr := io.ReadAll(io.LimitReader(resp.Body, 512))
		return msg, errors.Join(fmt.Errorf("%s: %s", resp.Status, bytes.TrimSpace(detail)), readErr)
	}
	var out struct {
		Choices []struct {
			Message openAIMessage `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return msg, fmt.Errorf("ler resposta: %w", err)
	}
	if len(out.Choices) == 0 {
		return msg, errors.New("resposta sem choices")
	}
	return out.Choices[0].Message, nil
}

func openAITools(tools []Tool) []map[string]any {
	specs := make([]map[string]any, 0, len(tools))
	for _, t := range tools {
		specs = append(specs, map[string]any{
			"type":     "function",
			"function": map[string]any{"name": t.Name, "description": t.Description, "parameters": toolSchema(t)},
		})
	}
	return specs
}
