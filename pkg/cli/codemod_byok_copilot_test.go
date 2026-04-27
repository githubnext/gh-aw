//go:build !integration

package cli

import (
	"testing"

	"github.com/github/gh-aw/pkg/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestByokCopilotFeatureRemovalCodemod(t *testing.T) {
	codemod := getByokCopilotFeatureRemovalCodemod()

	tests := []struct {
		name        string
		input       string
		expectApply bool
	}{
		{
			name: "removes byok-copilot when true",
			input: `---
name: Test Workflow
engine: copilot
features:
  byok-copilot: true
  mcp-gateway: true
---
# Test workflow`,
			expectApply: true,
		},
		{
			name: "removes byok-copilot when false",
			input: `---
name: Test Workflow
engine: copilot
features:
  byok-copilot: false
---
# Test workflow`,
			expectApply: true,
		},
		{
			name: "does not modify when byok-copilot is absent",
			input: `---
name: Test Workflow
engine: copilot
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
				assert.NotContains(t, output, "byok-copilot:", "Codemod should remove deprecated byok-copilot flag")
			} else {
				assert.Equal(t, tt.input, output, "Output should be unchanged when codemod does not apply")
			}
		})
	}
}
