//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBashAllowlistUnsupportedEngineCodemod_Metadata(t *testing.T) {
	codemod := getBashAllowlistUnsupportedEngineCodemod()

	assert.Equal(t, "bash-allowlist-unsupported-engine-guided-error", codemod.ID)
	assert.NotEmpty(t, codemod.Name)
	assert.NotEmpty(t, codemod.Description)
	assert.Equal(t, "0.78.0", codemod.IntroducedIn)
	assert.True(t, codemod.Guided, "codemod must be guided since the fix changes semantics")
	assert.NotNil(t, codemod.Apply)
}

func TestBashAllowlistUnsupportedEngineCodemod_Apply(t *testing.T) {
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
			errContains: []string{"engine 'codex' does not support bash command allow-listing", "'bash: [git, npm]'", "copilot, claude, or gemini", `bash: ["*"]`},
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
			name: "default engine with restricted bash allow-list is a no-op",
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
