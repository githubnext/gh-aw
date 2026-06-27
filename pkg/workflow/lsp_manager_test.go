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
	assert.Contains(t, content, "npm install -g --ignore-scripts typescript typescript-language-server")
}

func TestLSPManagerRuntimeRequirements_NodeBased(t *testing.T) {
	// Node.js-based LSP servers (typescript, python/pyright, bash, php, yaml) should all
	// resolve to the "node" runtime — deduplicated to a single requirement.
	manager := NewLSPManager(map[string]LSPServerConfig{
		"typescript": {Command: "typescript-language-server", FileExtensions: map[string]string{".ts": "typescript"}},
		"python":     {Command: "pyright-langserver", FileExtensions: map[string]string{".py": "python"}},
	})
	reqs := manager.RuntimeRequirements()
	require.Len(t, reqs, 1, "typescript and python both need node; expect exactly one node requirement")
	assert.Equal(t, "node", reqs[0].Runtime.ID)
}

func TestLSPManagerRuntimeRequirements_GoLSP(t *testing.T) {
	// gopls requires the Go runtime.
	manager := NewLSPManager(map[string]LSPServerConfig{
		"go": {Command: "gopls", FileExtensions: map[string]string{".go": "go"}},
	})
	reqs := manager.RuntimeRequirements()
	require.Len(t, reqs, 1)
	assert.Equal(t, "go", reqs[0].Runtime.ID)
}

func TestLSPManagerRuntimeRequirements_RubyLSP(t *testing.T) {
	// solargraph requires the Ruby runtime.
	manager := NewLSPManager(map[string]LSPServerConfig{
		"ruby": {Command: "solargraph", FileExtensions: map[string]string{".rb": "ruby"}},
	})
	reqs := manager.RuntimeRequirements()
	require.Len(t, reqs, 1)
	assert.Equal(t, "ruby", reqs[0].Runtime.ID)
}

func TestLSPManagerRuntimeRequirements_RustLSP(t *testing.T) {
	// rust-analyzer uses rustup; Rust is not in knownRuntimes so no runtime requirement is returned.
	manager := NewLSPManager(map[string]LSPServerConfig{
		"rust": {Command: "rust-analyzer", FileExtensions: map[string]string{".rs": "rust"}},
	})
	reqs := manager.RuntimeRequirements()
	assert.Empty(t, reqs, "Rust is not in knownRuntimes; expect no runtime requirement")
}

func TestLSPManagerRuntimeRequirements_UnknownLanguage(t *testing.T) {
	// A language not in lspInstallSpecs contributes no runtime requirement.
	manager := NewLSPManager(map[string]LSPServerConfig{
		"cobol": {Command: "cobol-lsp", FileExtensions: map[string]string{".cbl": "cobol"}},
	})
	reqs := manager.RuntimeRequirements()
	assert.Empty(t, reqs)
}

func TestLSPManagerRuntimeRequirements_MultipleRuntimes(t *testing.T) {
	// A workflow with both a Go LSP and a TypeScript LSP needs both Go and Node.js.
	manager := NewLSPManager(map[string]LSPServerConfig{
		"go":         {Command: "gopls", FileExtensions: map[string]string{".go": "go"}},
		"typescript": {Command: "typescript-language-server", FileExtensions: map[string]string{".ts": "typescript"}},
	})
	reqs := manager.RuntimeRequirements()
	require.Len(t, reqs, 2)
	ids := map[string]bool{}
	for _, r := range reqs {
		ids[r.Runtime.ID] = true
	}
	assert.True(t, ids["go"], "expected go runtime requirement")
	assert.True(t, ids["node"], "expected node runtime requirement")
}

func TestDetectRuntimeRequirements_LSPServers(t *testing.T) {
	// DetectRuntimeRequirements should pick up runtime requirements from LSP config.
	data := &WorkflowData{
		LSP: map[string]LSPServerConfig{
			"go": {Command: "gopls", FileExtensions: map[string]string{".go": "go"}},
		},
	}
	reqs := DetectRuntimeRequirements(data)
	found := false
	for _, r := range reqs {
		if r.Runtime.ID == "go" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected Go runtime requirement from LSP config")
}
