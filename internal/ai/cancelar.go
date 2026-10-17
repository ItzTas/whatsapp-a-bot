package ai

import (
	"context"
	"fmt"
	"strings"

	"go.mau.fi/whatsmeow/types/events"
)

func (b *bot) cancelar(ctx context.Context, evt *events.Message) error {
	return b.askAboutPerson(ctx, evt, "cancelar", func(name string) string {
		return fmt.Sprintf(`Escreve uma thread de cancelamento de %s no estilo Twitter/X, como se a internet inteira tivesse decidido cancelar a pessoa hoje.
Faça de 3 a 5 tweets numerados (1/, 2/...), cada um de um @usuário fictício diferente com nome de perfil engraçado.
Use a hashtag #%sIsOverParty em pelo menos dois tweets (o nome vai sem espaço nem acento, com a primeira letra maiúscula).
Inclua "prints recuperados": trechos reais das mensagens acima, entre aspas, tirados de contexto e tratados como prova de um crime gravíssimo.
Feche com uma nota de repúdio oficial curtinha, assinada por alguma entidade inventada.
É zoeira entre amigos: pode ser ácido e exagerado, mas nada de preconceito (raça, religião, orientação sexual, gênero, deficiência, corpo) nem acusação de crime de verdade.`, name, strings.Join(strings.Fields(name), ""))
	})
}
