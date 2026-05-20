//go:build !integration

package stringutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindClosestMatches(t *testing.T) {
	tests := []struct {
		name       string
		target     string
		candidates []string
		maxResults int
		want       []string
	}{
		{
			name:       "typo copiliot suggests copilot",
			target:     "copiliot",
			candidates: []string{"copilot", "claude", "codex", "custom"},
			maxResults: 3,
			want:       []string{"copilot"},
		},
		{
			name:       "case insensitive Copilot is exact match of copilot, no suggestions",
			target:     "Copilot",
			candidates: []string{"copilot", "claude", "codex", "custom"},
			maxResults: 3,
			want:       nil,
		},
		{
			name:       "partial match cop suggests codex within distance 3",
			target:     "cop",
			candidates: []string{"copilot", "claude", "codex", "custom"},
			maxResults: 3,
			want:       []string{"codex"},
		},
		{
			name:       "too different xyz no suggestions",
			target:     "xyz",
			candidates: []string{"copilot", "claude", "codex", "custom"},
			maxResults: 3,
			want:       nil,
		},
		{
			name:       "exact match skipped",
			target:     "copilot",
			candidates: []string{"copilot", "claude", "codex"},
			maxResults: 3,
			want:       nil,
		},
		{
			name:       "respects maxResults limit",
			target:     "cont",
			candidates: []string{"contents", "content", "context", "controls"},
			maxResults: 2,
			want:       []string{"content", "context"},
		},
		{
			name:       "push typo pus suggests push",
			target:     "pus",
			candidates: []string{"push", "pull_request", "issues"},
			maxResults: 1,
			want:       []string{"push"},
		},
		{
			name:       "typo in scope suggests contents",
			target:     "cntents",
			candidates: []string{"contents", "checks", "issues", "actions"},
			maxResults: 1,
			want:       []string{"contents"},
		},
		{
			name:       "empty target returns nothing",
			target:     "",
			candidates: []string{"copilot", "claude"},
			maxResults: 3,
			want:       nil,
		},
		{
			name:       "empty candidates returns nothing",
			target:     "copilot",
			candidates: []string{},
			maxResults: 3,
			want:       nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FindClosestMatches(tt.target, tt.candidates, tt.maxResults)
			assert.Equal(t, tt.want, got, "FindClosestMatches(%q, %v, %d)", tt.target, tt.candidates, tt.maxResults)
		})
	}
}

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want int
	}{
		{name: "identical strings", a: "copilot", b: "copilot", want: 0},
		{name: "one substitution", a: "copilot", b: "copiliot", want: 1},
		{name: "one deletion", a: "copilot", b: "copliot", want: 2},
		{name: "empty a", a: "", b: "abc", want: 3},
		{name: "empty b", a: "abc", b: "", want: 3},
		{name: "both empty", a: "", b: "", want: 0},
		{name: "push vs pus", a: "push", b: "pus", want: 1},
		{name: "contents vs scope typo", a: "contents", b: "cntents", want: 1},
		{name: "completely different", a: "xyz", b: "abc", want: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LevenshteinDistance(tt.a, tt.b)
			assert.Equal(t, tt.want, got, "LevenshteinDistance(%q, %q)", tt.a, tt.b)
		})
	}
}
