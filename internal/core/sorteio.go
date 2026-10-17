package core

import (
	"math/rand/v2"
)

func PickOne[T any](items []T) T {
	return items[rand.IntN(len(items))]
}

func PickSome[T any](items []T, n int) []T {
	picked := make([]T, 0, n)
	for _, i := range rand.Perm(len(items))[:min(n, len(items))] {
		picked = append(picked, items[i])
	}
	return picked
}
