package parser

import (
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractLinearBuiltinMCPTool(t *testing.T) {
	required := false
	frontmatter := map[string]any{
		"tools": map[string]any{
			"linear": map[string]any{
				"token":     "${{ secrets.LINEAR_API_KEY }}",
				"read-only": false,
				"allowed":   []any{"get_issue", "list_issues"},
				"required":  required,
			},
		},
	}

	configs, err := ExtractMCPConfigurations(frontmatter, "linear")
	require.NoError(t, err)
	require.Len(t, configs, 1)
	config := configs[0]
	assert.Equal(t, "linear", config.Name)
	assert.Equal(t, "http", config.Type)
	assert.Equal(t, constants.LinearMCPURL, config.URL)
	assert.Equal(t, "Bearer "+"${{ secrets.LINEAR_API_KEY }}", config.Headers["Authorization"])
	assert.Equal(t, []string{"get_issue", "list_issues"}, config.Allowed)
	require.NotNil(t, config.Required)
	assert.False(t, *config.Required)
}

func TestExtractLinearBuiltinMCPToolDefaultsReadOnly(t *testing.T) {
	frontmatter := map[string]any{
		"tools": map[string]any{
			"linear": map[string]any{
				"token": "${{ secrets.LINEAR_API_KEY }}",
			},
		},
	}

	configs, err := ExtractMCPConfigurations(frontmatter, "")
	require.NoError(t, err)
	require.Len(t, configs, 1)
	assert.Equal(t, constants.LinearMCPReadOnlyURL, configs[0].URL)
	assert.Nil(t, configs[0].Required)
}

func TestLinearToolSchema(t *testing.T) {
	valid := map[string]any{
		"on":     "workflow_dispatch",
		"engine": "copilot",
		"tools": map[string]any{
			"linear": map[string]any{
				"token":     "${{ secrets.LINEAR_API_KEY }}",
				"read-only": true,
				"allowed":   []any{"*"},
				"required":  true,
			},
		},
	}
	require.NoError(t, ValidateMainWorkflowFrontmatterWithSchemaAndLocation(valid, "/tmp/linear-valid.md"))

	tests := []map[string]any{
		{},
		{"token": "literal-token"},
		{"token": "${{ secrets.LINEAR_API_KEY }}", "read-only": "yes"},
		{"token": "${{ secrets.LINEAR_API_KEY }}", "allowed": []any{}},
		{"token": "${{ secrets.LINEAR_API_KEY }}", "unknown": true},
	}
	for _, linear := range tests {
		frontmatter := map[string]any{
			"on":     "workflow_dispatch",
			"engine": "copilot",
			"tools":  map[string]any{"linear": linear},
		}
		require.Error(t, ValidateMainWorkflowFrontmatterWithSchemaAndLocation(frontmatter, "/tmp/linear-invalid.md"))
	}
}
