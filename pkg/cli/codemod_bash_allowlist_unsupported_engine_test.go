//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBashAllowlistUnsupportedEngineCodemod_Metadata(t *testing.T) {
	t.Parallel()
	codemod := getBashAllowlistUnsupportedEngineCodemod()

	assert.Equal(t, "bash-allowlist-unsupported-engine-guided-error", codemod.ID)
	assert.NotEmpty(t, codemod.Name)
	assert.NotEmpty(t, codemod.Description)
	assert.Equal(t, "0.78.0", codemod.IntroducedIn)
	assert.True(t, codemod.Guided, "codemod must be guided since the fix changes semantics")
	assert.NotNil(t, codemod.Apply)
	assert.NotNil(t, codemod.ApplyWithContext, "codemod must expose ApplyWithContext to resolve effective tools from imports")
}

func TestBashAllowlistUnsupportedEngineCodemod_Apply(t *testing.T) {
	t.Parallel()
	codemod := getBashAllowlistUnsupportedEngineCodemod()

	content := `---
on: workflow_dispatch
engine:
  id: codex
tools:
  bash: ["git", "npm"]
---

# Agent
`

	tests := []struct {
		name        string
		frontmatter map[string]any
		wantErr     bool
		errContains []string
	}{
		{
			name: "codex with restricted bash allow-list returns guided error",
			frontmatter: map[string]any{
				"engine": map[string]any{"id": "codex"},
				"tools":  map[string]any{"bash": []any{"git", "npm"}},
			},
			wantErr:     true,
			errContains: []string{"engine 'codex' does not support bash command allow-listing", `'bash: ["git", "npm"]'`, "copilot", "claude", "gemini", `bash: ["*"]`},
		},
		{
			name: "codex as engine string returns guided error",
			frontmatter: map[string]any{
				"engine": "codex",
				"tools":  map[string]any{"bash": []any{"git"}},
			},
			wantErr:     true,
			errContains: []string{"engine 'codex' does not support bash command allow-listing"},
		},
		{
			name: "command with control characters is quoted and does not spoof output",
			frontmatter: map[string]any{
				"engine": "codex",
				"tools":  map[string]any{"bash": []any{"git\x1b[31m status"}},
			},
			wantErr: true,
			// The ANSI escape sequence must be rendered as \x1b in the error, not raw bytes.
			errContains: []string{`"git\x1b[31m status"`},
		},
		{
			name: "command with embedded newline is quoted and does not spoof output",
			frontmatter: map[string]any{
				"engine": "codex",
				"tools":  map[string]any{"bash": []any{"git\nstatus"}},
			},
			wantErr:     true,
			errContains: []string{`"git\nstatus"`},
		},
		{
			name: "codex with bash: false returns guided error",
			frontmatter: map[string]any{
				"engine": "codex",
				"tools":  map[string]any{"bash": false},
			},
			wantErr:     true,
			errContains: []string{"'bash: false'"},
		},
		{
			name: "codex with empty bash list returns guided error",
			frontmatter: map[string]any{
				"engine": "codex",
				"tools":  map[string]any{"bash": []any{}},
			},
			wantErr:     true,
			errContains: []string{"'bash: []'"},
		},
		{
			name: "codex with wildcard bash is a no-op",
			frontmatter: map[string]any{
				"engine": "codex",
				"tools":  map[string]any{"bash": []any{"*"}},
			},
		},
		{
			name: "codex with bash: true is a no-op",
			frontmatter: map[string]any{
				"engine": "codex",
				"tools":  map[string]any{"bash": true},
			},
		},
		{
			name: "codex without tools.bash is a no-op",
			frontmatter: map[string]any{
				"engine": "codex",
				"tools":  map[string]any{"edit": nil},
			},
		},
		{
			name: "copilot with restricted bash allow-list is a no-op",
			frontmatter: map[string]any{
				"engine": "copilot",
				"tools":  map[string]any{"bash": []any{"git", "npm"}},
			},
		},
		{
			// No engine key → extractEngineIDFromFrontmatter returns "copilot", which does
			// support BashCommandAllowlist, so no guided error is emitted. This test will
			// need to be revisited if the default engine changes or loses the capability.
			name: "default engine (copilot) with restricted bash allow-list is a no-op because copilot supports the capability",
			frontmatter: map[string]any{
				"tools": map[string]any{"bash": []any{"git"}},
			},
		},
		{
			name: "unknown engine is a no-op",
			frontmatter: map[string]any{
				"engine": "not-a-real-engine",
				"tools":  map[string]any{"bash": []any{"git"}},
			},
		},
		{
			name:        "workflow without tools is a no-op",
			frontmatter: map[string]any{"engine": "codex"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			newContent, applied, err := codemod.Apply(content, tt.frontmatter)
			assert.False(t, applied, "guided codemod never modifies the workflow")
			assert.Equal(t, content, newContent, "content must be preserved")
			if !tt.wantErr {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			for _, expected := range tt.errContains {
				assert.Contains(t, err.Error(), expected)
			}
		})
	}
}

// TestBashAllowlistUnsupportedEngineCodemod_ApplyWithContext_ImportedRestriction verifies that
// ApplyWithContext detects a bash restriction that originates solely from an imported file (not
// the top-level workflow), which Apply cannot detect because it only sees raw frontmatter.
func TestBashAllowlistUnsupportedEngineCodemod_ApplyWithContext_ImportedRestriction(t *testing.T) {
	t.Parallel()
	codemod := getBashAllowlistUnsupportedEngineCodemod()

	dir := t.TempDir()

	// Import file that declares a restricted bash allow-list.
	importContent := `---
tools:
  bash: ["git", "npm ci"]
---
`
	importPath := filepath.Join(dir, "tools-import.md")
	require.NoError(t, os.WriteFile(importPath, []byte(importContent), 0o644))

	// Main workflow file: codex engine, no top-level tools.bash, but imports the restriction.
	mainContent := `---
engine:
  id: codex
imports:
  - tools-import.md
---

# Agent
`
	mainPath := filepath.Join(dir, "workflow.md")
	require.NoError(t, os.WriteFile(mainPath, []byte(mainContent), 0o644))

	frontmatter := map[string]any{
		"engine":  map[string]any{"id": "codex"},
		"imports": []any{"tools-import.md"},
	}

	newContent, applied, err := codemod.ApplyWithContext(mainContent, frontmatter, mainPath)
	assert.False(t, applied, "guided codemod never modifies the workflow")
	assert.Equal(t, mainContent, newContent, "content must be preserved")
	require.Error(t, err, "should detect bash restriction imported from shared file")
	assert.Contains(t, err.Error(), "engine 'codex' does not support bash command allow-listing")
}

// TestBashAllowlistUnsupportedEngineCodemod_ApplyWithContext_NoImportedRestriction verifies that
// ApplyWithContext is a no-op when no bash restriction exists in the top-level or imported tools.
func TestBashAllowlistUnsupportedEngineCodemod_ApplyWithContext_NoImportedRestriction(t *testing.T) {
	t.Parallel()
	codemod := getBashAllowlistUnsupportedEngineCodemod()

	content := `---
engine:
  id: codex
---

# Agent
`
	frontmatter := map[string]any{
		"engine": map[string]any{"id": "codex"},
	}

	newContent, applied, err := codemod.ApplyWithContext(content, frontmatter, "")
	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, newContent)
}
