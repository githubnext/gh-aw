//go:build !integration

package cli

import (
	"testing"

	"github.com/github/gh-aw/pkg/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInlineAgentsFeatureRemovalCodemod(t *testing.T) {
	codemod := getInlineAgentsFeatureRemovalCodemod()

	tests := []struct {
		name        string
		input       string
		expectApply bool
	}{
		{
			name: "removes inline-agents when true",
			input: `---
name: Test Workflow
features:
  inline-agents: true
  mcp-gateway: true
---
# Test workflow`,
			expectApply: true,
		},
		{
			name: "removes inline-agents when false",
			input: `---
name: Test Workflow
features:
  inline-agents: false
---
# Test workflow`,
			expectApply: true,
		},
		{
			name: "does not modify when inline-agents is absent",
			input: `---
name: Test Workflow
features:
  mcp-gateway: true
---
# Test workflow`,
			expectApply: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.ExtractFrontmatterFromContent(tt.input)
			require.NoError(t, err, "Failed to parse test input frontmatter")

			output, applied, err := codemod.Apply(tt.input, result.Frontmatter)
			require.NoError(t, err, "Codemod apply should not error")
			assert.Equal(t, tt.expectApply, applied, "Applied status mismatch")

			if tt.expectApply {
				assert.NotContains(t, output, "inline-agents:", "Codemod should remove deprecated inline-agents flag")
			} else {
				assert.Equal(t, tt.input, output, "Output should be unchanged when codemod does not apply")
			}
		})
	}
}
