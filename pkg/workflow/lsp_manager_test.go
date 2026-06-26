//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLSPManagerValidate(t *testing.T) {
	manager := NewLSPManager(map[string]LSPServerConfig{
		"typescript": {
			Command: "typescript-language-server",
			Args:    []string{"--stdio"},
			FileExtensions: map[string]string{
				".ts": "typescript",
			},
		},
	})
	require.NoError(t, manager.Validate())

	invalid := NewLSPManager(map[string]LSPServerConfig{
		"python": {
			Command: "pyright-langserver",
		},
	})
	require.Error(t, invalid.Validate())
}

func TestLSPManagerDuplicateKeyNormalization(t *testing.T) {
	// "TypeScript" and "typescript" both normalize to "typescript".
	// Sorted order puts "TypeScript" first (uppercase < lowercase in ASCII),
	// so the "TypeScript" entry should win deterministically.
	manager := NewLSPManager(map[string]LSPServerConfig{
		"TypeScript": {
			Command: "first-server",
			FileExtensions: map[string]string{
				".ts": "typescript",
			},
		},
		"typescript": {
			Command: "second-server",
			FileExtensions: map[string]string{
				".ts": "typescript",
			},
		},
	})

	servers := manager.CopilotLSPServers()
	require.Len(t, servers, 1)
	assert.Equal(t, "first-server", servers["typescript"].Command)
}

func TestLSPManagerGenerateInstallSteps(t *testing.T) {
	manager := NewLSPManager(map[string]LSPServerConfig{
		"typescript": {
			Command: "typescript-language-server",
			FileExtensions: map[string]string{
				".ts": "typescript",
			},
		},
		"unknown": {
			Command: "my-lsp",
			FileExtensions: map[string]string{
				".foo": "foo",
			},
		},
	})

	steps := manager.GenerateInstallSteps()
	require.Len(t, steps, 1)
	content := strings.Join(steps[0], "\n")
	assert.Contains(t, content, "Install TypeScript LSP dependencies")
	assert.Contains(t, content, "npm install -g typescript typescript-language-server")
}
