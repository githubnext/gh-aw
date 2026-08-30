//go:build !integration

package workflow

import (
	"testing"

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

	t.Run("claude engine accepts lsp", func(t *testing.T) {
		err := compiler.validateLSPSupport(&WorkflowData{
			AI:  "claude",
			LSP: validLSP,
		})
		require.NoError(t, err)
	})

	t.Run("claude bare mode rejects lsp", func(t *testing.T) {
		err := compiler.validateLSPSupport(&WorkflowData{
			AI:           "claude",
			EngineConfig: &EngineConfig{Bare: true},
			LSP:          validLSP,
		})
		require.ErrorContains(t, err, "Claude Code disables LSP in bare mode")
	})

	t.Run("unsupported engine rejects lsp", func(t *testing.T) {
		err := compiler.validateLSPSupport(&WorkflowData{
			AI:  "codex",
			LSP: validLSP,
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "only supported for engines: copilot and claude")
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
		require.ErrorContains(t, err, "lsp.python.command is required")
	})
}
