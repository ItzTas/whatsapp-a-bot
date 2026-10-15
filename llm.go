package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

type chatModel interface {
	reply(ctx context.Context, system, prompt string, maxTokens int) (string, error)
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

func (m anthropicModel) reply(ctx context.Context, system, prompt string, maxTokens int) (string, error) {
	resp, err := m.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     m.model,
		MaxTokens: int64(maxTokens),
		System:    []anthropic.TextBlockParam{{Text: system}},
		Messages:  []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock(prompt))},
	})
	if err != nil {
		return "", err
	}
	if resp.StopReason == anthropic.StopReasonRefusal {
		return "", fmt.Errorf("modelo recusou responder (%s)", resp.StopDetails.Category)
	}
	var sb strings.Builder
	for _, block := range resp.Content {
		if text, ok := block.AsAny().(anthropic.TextBlock); ok {
			sb.WriteString(text.Text)
		}
	}
	return strings.TrimSpace(sb.String()), nil
}

type openAIModel struct {
	endpoint string
	key      string
	model    string
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func newOpenAIModel(s modelSettings) (chatModel, error) {
	if s.url == "" {
		return nil, errors.New("a API openai precisa da URL base, ex.: https://api.deepseek.com/v1")
	}
	return openAIModel{endpoint: strings.TrimSuffix(s.url, "/") + "/chat/completions", key: s.key, model: s.model}, nil
}

func (m openAIModel) reply(ctx context.Context, system, prompt string, maxTokens int) (answer string, err error) {
	body, err := json.Marshal(map[string]any{
		"model":      m.model,
		"max_tokens": maxTokens,
		"messages":   []openAIMessage{{Role: "system", Content: system}, {Role: "user", Content: prompt}},
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if m.key != "" {
		req.Header.Set("Authorization", "Bearer "+m.key)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { err = errors.Join(err, resp.Body.Close()) }()

	if resp.StatusCode != http.StatusOK {
		detail, readErr := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", errors.Join(fmt.Errorf("%s: %s", resp.Status, bytes.TrimSpace(detail)), readErr)
	}
	var out struct {
		Choices []struct {
			Message openAIMessage `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("ler resposta: %w", err)
	}
	if len(out.Choices) == 0 {
		return "", errors.New("resposta sem choices")
	}
	return strings.TrimSpace(out.Choices[0].Message.Content), nil
}
