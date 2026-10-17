package ai

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/llm"
)

const (
	iaDiceMax      = 20
	iaDiceSidesMax = 1000
	iaDrawMax      = 50
)

func (k *iaToolkit) randomTools() []llm.Tool {
	list := func(description string) map[string]any {
		return map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": description}
	}
	return []llm.Tool{
		{
			Name:        "rolar_dados",
			Description: "Rola dados de verdade e devolve cada resultado e a soma. Use sempre que precisar de um dado, de uma rolagem de RPG ou de um resultado na sorte, em vez de inventar o número.",
			Properties: map[string]any{
				"dados": map[string]any{"type": "integer", "description": fmt.Sprintf("quantos dados rolar (padrão 1, máximo %d)", iaDiceMax)},
				"lados": map[string]any{"type": "integer", "description": fmt.Sprintf("quantos lados cada dado tem (padrão 6, máximo %d)", iaDiceSidesMax)},
			},
			Run: rollDiceTool,
		},
		{
			Name:        "numero_aleatorio",
			Description: "Sorteia um número inteiro entre o mínimo e o máximo, contando os dois. Use quando pedirem um número na sorte ou uma nota, porcentagem ou chance aleatória.",
			Properties: map[string]any{
				"minimo": map[string]any{"type": "integer", "description": "menor número possível"},
				"maximo": map[string]any{"type": "integer", "description": "maior número possível"},
			},
			Required: []string{"minimo", "maximo"},
			Run:      randomNumberTool,
		},
		{
			Name:        "cara_ou_coroa",
			Description: "Joga uma moeda e diz se deu cara ou coroa.",
			Run: func(context.Context, json.RawMessage) (string, error) {
				return core.PickOne([]string{"Deu cara.", "Deu coroa."}), nil
			},
		},
		{
			Name:        "sortear",
			Description: "Sorteia uma ou mais opções de uma lista, sem repetir. Use pra escolher na sorte entre opções, tipo onde comer, qual filme ver ou quem paga a conta.",
			Properties: map[string]any{
				"opcoes":     list("as opções do sorteio"),
				"quantidade": map[string]any{"type": "integer", "description": "quantas opções sortear (padrão 1)"},
			},
			Required: []string{"opcoes"},
			Run:      drawTool,
		},
		{
			Name:        "embaralhar",
			Description: "Embaralha uma lista e devolve numa ordem aleatória. Use pra montar ordem de jogadores, fila, times ou ranking na sorte.",
			Properties: map[string]any{
				"itens": list("o que embaralhar"),
			},
			Required: []string{"itens"},
			Run:      shuffleTool,
		},
		{
			Name:        "sortear_membros",
			Description: "Sorteia pessoas do grupo, sem repetir. Use quando pedirem pra escolher alguém do grupo na sorte. Pra marcar quem saiu, chame marcar_pessoas com os nomes depois.",
			Properties: map[string]any{
				"quantidade": map[string]any{"type": "integer", "description": "quantas pessoas sortear (padrão 1)"},
			},
			Run: k.drawMembers,
		},
	}
}

func rollDiceTool(_ context.Context, input json.RawMessage) (string, error) {
	args, err := decodeToolInput[struct {
		Dice  int `json:"dados"`
		Sides int `json:"lados"`
	}](input)
	if err != nil {
		return "", err
	}
	dice, sides := cmp.Or(args.Dice, 1), cmp.Or(args.Sides, 6)
	if dice < 1 || dice > iaDiceMax {
		return "", fmt.Errorf("dá pra rolar de 1 a %d dados", iaDiceMax)
	}
	if sides < 2 || sides > iaDiceSidesMax {
		return "", fmt.Errorf("o dado precisa ter de 2 a %d lados", iaDiceSidesMax)
	}
	rolls := make([]string, dice)
	total := 0
	for i := range rolls {
		roll := 1 + rand.IntN(sides)
		total += roll
		rolls[i] = strconv.Itoa(roll)
	}
	if dice == 1 {
		return fmt.Sprintf("1d%d: deu %d.", sides, total), nil
	}
	return fmt.Sprintf("%dd%d: %s. Soma: %d.", dice, sides, strings.Join(rolls, ", "), total), nil
}

func randomNumberTool(_ context.Context, input json.RawMessage) (string, error) {
	args, err := decodeToolInput[struct {
		Min int64 `json:"minimo"`
		Max int64 `json:"maximo"`
	}](input)
	if err != nil {
		return "", err
	}
	low, high := min(args.Min, args.Max), max(args.Min, args.Max)
	span := high - low + 1
	if span <= 0 {
		return "", errors.New("o intervalo é grande demais")
	}
	return fmt.Sprintf("Sorteado entre %d e %d: %d.", low, high, low+rand.Int64N(span)), nil
}

func drawTool(_ context.Context, input json.RawMessage) (string, error) {
	args, err := decodeToolInput[struct {
		Options  []string `json:"opcoes"`
		Quantity int      `json:"quantidade"`
	}](input)
	if err != nil {
		return "", err
	}
	options := cleanToolList(args.Options)
	if len(options) < 2 {
		return "", errors.New("manda pelo menos duas opções pro sorteio")
	}
	picked, err := drawFrom(options, args.Quantity)
	if err != nil {
		return "", err
	}
	return "Sorteado: " + strings.Join(picked, ", ") + ".", nil
}

func shuffleTool(_ context.Context, input json.RawMessage) (string, error) {
	args, err := decodeToolInput[struct {
		Items []string `json:"itens"`
	}](input)
	if err != nil {
		return "", err
	}
	items := cleanToolList(args.Items)
	if len(items) < 2 {
		return "", errors.New("manda pelo menos dois itens pra embaralhar")
	}
	rand.Shuffle(len(items), func(i, j int) { items[i], items[j] = items[j], items[i] })
	lines := make([]string, len(items))
	for i, item := range items {
		lines[i] = fmt.Sprintf("%d. %s", i+1, item)
	}
	return strings.Join(lines, "\n"), nil
}

func (k *iaToolkit) drawMembers(_ context.Context, input json.RawMessage) (string, error) {
	args, err := decodeToolInput[struct {
		Quantity int `json:"quantidade"`
	}](input)
	if err != nil {
		return "", err
	}
	members, err := k.members()
	if err != nil {
		return "", err
	}
	names := make([]string, len(members))
	for i, m := range members {
		names[i] = m.name
	}
	picked, err := drawFrom(names, args.Quantity)
	if err != nil {
		return "", err
	}
	return "Sorteado: " + strings.Join(picked, ", ") + ".", nil
}

func drawFrom(items []string, quantity int) ([]string, error) {
	quantity = cmp.Or(quantity, 1)
	if quantity < 1 || quantity > min(len(items), iaDrawMax) {
		return nil, fmt.Errorf("dá pra sortear de 1 a %d", min(len(items), iaDrawMax))
	}
	picked := make([]string, quantity)
	for i, j := range rand.Perm(len(items))[:quantity] {
		picked[i] = items[j]
	}
	return picked, nil
}

func cleanToolList(items []string) []string {
	cleaned := make([]string, 0, len(items))
	for _, item := range items {
		if item = strings.TrimSpace(item); item != "" {
			cleaned = append(cleaned, item)
		}
	}
	return slices.Clip(cleaned)
}
