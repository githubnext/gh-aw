package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPullRequestForksSchema(t *testing.T) {
	tests := []struct {
		name  string
		forks any
		valid bool
	}{
		{"string pattern", "trusted/*", true},
		{"pattern list", []any{"trusted/repo", "org/*"}, true},
		{"wildcard", "*", true},
		{"empty pattern list", []any{}, true},
		{"trusted associations", map[string]any{"author-association": []any{"OWNER", "MEMBER", "COLLABORATOR"}}, true},
		{"all association values", map[string]any{"author-association": []any{"COLLABORATOR", "CONTRIBUTOR", "FIRST_TIMER", "FIRST_TIME_CONTRIBUTOR", "MANNEQUIN", "MEMBER", "NONE", "OWNER"}}, true},
		{"empty associations", map[string]any{"author-association": []any{}}, true},
		{"missing associations", map[string]any{}, false},
		{"misspelled key", map[string]any{"author-associations": []any{"OWNER"}}, false},
		{"unknown property", map[string]any{"author-association": []any{"OWNER"}, "unknown": true}, false},
		{"scalar association", map[string]any{"author-association": "OWNER"}, false},
		{"unknown association", map[string]any{"author-association": []any{"ADMIN"}}, false},
		{"lowercase association", map[string]any{"author-association": []any{"owner"}}, false},
		{"wildcard association", map[string]any{"author-association": []any{"*"}}, false},
		{"non-string association", map[string]any{"author-association": []any{true}}, false},
		{"null associations", map[string]any{"author-association": nil}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMainWorkflowFrontmatterWithSchemaAndLocation(map[string]any{
				"on": map[string]any{
					"pull_request": map[string]any{"forks": tt.forks},
				},
			}, "forks.md")
			if tt.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
