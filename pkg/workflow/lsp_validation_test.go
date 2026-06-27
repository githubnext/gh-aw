//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateLSPSupport(t *testing.T) {
	compiler := NewCompiler()

	validLSP := map[string]LSPServerConfig{
		"typescript": {
			Command: "typescript-language-server",
			Args:    []string{"--stdio"},
			FileExtensions: map[string]string{
				".ts": "typescript",
			},
		},
	}

	t.Run("copilot engine accepts lsp", func(t *testing.T) {
		err := compiler.validateLSPSupport(&WorkflowData{
			AI:  "copilot",
			LSP: validLSP,
		})
		require.NoError(t, err)
	})

	t.Run("non-copilot engine rejects lsp", func(t *testing.T) {
		err := compiler.validateLSPSupport(&WorkflowData{
			AI:  "codex",
			LSP: validLSP,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "only supported for engine: copilot")
	})

	t.Run("invalid lsp config fails validation", func(t *testing.T) {
		err := compiler.validateLSPSupport(&WorkflowData{
			AI: "copilot",
			LSP: map[string]LSPServerConfig{
				"python": {
					Command: "",
					FileExtensions: map[string]string{
						".py": "python",
					},
				},
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "lsp.python.command is required")
	})
}
