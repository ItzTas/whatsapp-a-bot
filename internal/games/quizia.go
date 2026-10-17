package games

import (
	"errors"
	"fmt"
	"slices"
)

const (
	QuizOptionCount  = 4
	quizMaxQuestion  = 250
	quizMaxOptionLen = 100
)

func (q QuizQuestion) Validate() error {
	if q.Question == "" || len(q.Question) > quizMaxQuestion {
		return errors.New("pergunta vazia ou grande demais")
	}
	if len(q.Options) != QuizOptionCount {
		return fmt.Errorf("%d alternativas em vez de %d", len(q.Options), QuizOptionCount)
	}
	if q.Correct < 0 || q.Correct >= len(q.Options) {
		return fmt.Errorf("índice da certa fora das alternativas: %d", q.Correct)
	}
	if slices.ContainsFunc(q.Options, func(o string) bool { return o == "" || len(o) > quizMaxOptionLen }) {
		return errors.New("alternativa vazia ou grande demais")
	}
	unique := slices.Compact(slices.Sorted(slices.Values(q.Options)))
	if len(unique) != len(q.Options) {
		return errors.New("alternativas repetidas")
	}
	return nil
}
