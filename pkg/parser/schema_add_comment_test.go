//go:build !integration

package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAddCommentMaxLinksSchema(t *testing.T) {
	for _, tt := range []struct {
		name  string
		value any
		valid bool
	}{
		{"lower limit", 1, true},
		{"default limit", 50, true},
		{"raised limit", 100, true},
		{"zero", 0, false},
		{"negative", -1, false},
		{"fraction", 1.5, false},
		{"string", "100", false},
		{"expression", "${{ inputs.max-links }}", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMainWorkflowFrontmatterWithSchemaAndLocation(map[string]any{
				"on": "workflow_dispatch",
				"safe-outputs": map[string]any{
					"add-comment": map[string]any{"max-links": tt.value},
				},
			}, "workflow.md")
			if tt.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
