package strategy

import (
	"voyage/internal/domain"
	"voyage/internal/ports"
)

type Outgoing struct{}

func (Outgoing) Related(note *domain.Note, index *domain.GraphIndex) []ports.Relation {
	out := make([]ports.Relation, 0, len(note.ResolvedLinks)+len(note.DanglingLinks))
	for _, id := range note.ResolvedLinks {
		out = append(out, ports.Relation{Kind: "resolved", ID: id})
	}
	for _, raw := range note.DanglingLinks {
		out = append(out, ports.Relation{Kind: "dangling", Raw: raw})
	}
	return out
}
