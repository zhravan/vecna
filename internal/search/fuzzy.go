package search

import "strings"

type Result[T any] struct {
	Value T
	Score int
}

// Match returns whether query fuzzily matches candidate and a score where a
// higher score means a tighter match. Matching is case-insensitive and rewards
// consecutive characters and word starts.
func Match(candidate, query string) (bool, int) {
	candidate = strings.ToLower(candidate)
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return true, 0
	}
	ci, qi, score := 0, 0, 0
	consecutive := 0
	for ci < len(candidate) && qi < len(query) {
		if candidate[ci] == query[qi] {
			score += 10
			if ci == 0 || candidate[ci-1] == ' ' || candidate[ci-1] == '-' || candidate[ci-1] == '_' || candidate[ci-1] == '/' {
				score += 8
			}
			consecutive++
			score += consecutive * 3
			qi++
		} else {
			consecutive = 0
		}
		ci++
	}
	if qi != len(query) {
		return false, 0
	}
	score -= len(candidate) - len(query)
	return true, score
}

func Rank[T any](items []T, query string, text func(T) string) []Result[T] {
	out := make([]Result[T], 0, len(items))
	for _, item := range items {
		if ok, score := Match(text(item), query); ok {
			out = append(out, Result[T]{Value: item, Score: score})
		}
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].Score > out[j-1].Score; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

type Command struct {
	ID          string
	Label       string
	Description string
	Run         func() error
}
type Palette struct{ commands []Command }

func NewPalette(commands []Command) *Palette {
	return &Palette{commands: append([]Command(nil), commands...)}
}
func (p *Palette) Search(query string) []Result[Command] {
	return Rank(p.commands, query, func(c Command) string { return c.Label + " " + c.Description })
}
