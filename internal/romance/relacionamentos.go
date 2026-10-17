package romance

import (
	"slices"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
)

const (
	kindDating core.RelationshipKind = "namoro"
	kindLover  core.RelationshipKind = "amante"
)

func ofKind(relationships []core.Relationship, kind core.RelationshipKind) []core.Relationship {
	return slices.DeleteFunc(slices.Clone(relationships), func(r core.Relationship) bool { return r.Kind != kind })
}
