//go:build !integration

package cli

import (
	"testing"

	"github.com/github/gh-aw/pkg/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetBashSingleQuotedArgsCodemod(t *testing.T) {
	codemod := getBashSingleQuotedArgsCodemod()

	assert.Equal(t, "bash-single-quoted-args-rewrite", codemod.ID)
	assert.Equal(t, "Rewrite single-quoted bash tool args", codemod.Name)
	assert.NotEmpty(t, codemod.Description)
	assert.NotEmpty(t, codemod.IntroducedIn)
	require.NotNil(t, codemod.Apply)
}

func TestBashSingleQuotedArgsCodemod_RewritesSimpleSingleQuotedArg(t *testing.T) {
	codemod := getBashSingleQuotedArgsCodemod()
	content := `---
name: test
tools:
  bash:
    - grep -rn 'pattern'
---
Test workflow body`

	frontmatter := map[string]any{
		"name": "test",
		"tools": map[string]any{
			"bash": []any{"grep -rn 'pattern'"},
		},
	}

	out, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.True(t, applied)

	parsed, err := parser.ExtractFrontmatterFromContent(out)
	require.NoError(t, err)
	tools := parsed.Frontmatter["tools"].(map[string]any)
	bash := tools["bash"].([]any)
	assert.Equal(t, `grep -rn "pattern"`, bash[0])
	assert.Equal(t, "Test workflow body", parsed.Markdown)
}

func TestBashSingleQuotedArgsCodemod_RewritesGlobPatterns(t *testing.T) {
	codemod := getBashSingleQuotedArgsCodemod()
	content := `---
name: test
tools:
  bash:
    - grep -rn 'pattern' --include='*.lua'
---
body`

	frontmatter := map[string]any{
		"name": "test",
		"tools": map[string]any{
			"bash": []any{"grep -rn 'pattern' --include='*.lua'"},
		},
	}

	out, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.True(t, applied)

	parsed, err := parser.ExtractFrontmatterFromContent(out)
	require.NoError(t, err)
	tools := parsed.Frontmatter["tools"].(map[string]any)
	bash := tools["bash"].([]any)
	assert.Equal(t, `grep -rn "pattern" --include="*.lua"`, bash[0])
}

func TestBashSingleQuotedArgsCodemod_NoOpForAlreadySafeEntry(t *testing.T) {
	codemod := getBashSingleQuotedArgsCodemod()
	content := `---
name: test
tools:
  bash:
    - grep -rn "pattern"
---
body`

	frontmatter := map[string]any{
		"name": "test",
		"tools": map[string]any{
			"bash": []any{`grep -rn "pattern"`},
		},
	}

	out, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, out)
}

func TestBashSingleQuotedArgsCodemod_UnmatchedQuoteLeftUnchanged(t *testing.T) {
	codemod := getBashSingleQuotedArgsCodemod()
	content := `---
name: test
tools:
  bash:
    - grep -rn 'pattern
---
body`

	frontmatter := map[string]any{
		"name": "test",
		"tools": map[string]any{
			"bash": []any{"grep -rn 'pattern"},
		},
	}

	out, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, out)
}

func TestRewriteSingleQuotedBashArgs(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      string
		wantSafe  bool
		wantApply bool
	}{
		{
			name:      "simple quoted segment",
			input:     "grep -rn 'pattern'",
			want:      `grep -rn "pattern"`,
			wantSafe:  true,
			wantApply: true,
		},
		{
			name:      "escaped dollars stay literal",
			input:     "echo '$HOME'",
			want:      `echo "\$HOME"`,
			wantSafe:  true,
			wantApply: true,
		},
		{
			name:      "no single quotes",
			input:     `grep -rn "pattern"`,
			want:      `grep -rn "pattern"`,
			wantSafe:  true,
			wantApply: false,
		},
		{
			name:      "unmatched single quote",
			input:     "grep -rn 'pattern",
			want:      "grep -rn 'pattern",
			wantSafe:  false,
			wantApply: false,
		},
		{
			name:      "apostrophe inside double quotes remains unchanged",
			input:     `echo "it's fine"`,
			want:      `echo "it's fine"`,
			wantSafe:  true,
			wantApply: false,
		},
		{
			name:      "escaped apostrophe outside quotes remains unchanged",
			input:     `echo it\'s fine`,
			want:      `echo it\'s fine`,
			wantSafe:  true,
			wantApply: false,
		},
		{
			name:      "embedded quote pattern rewrites segments and preserves escaped apostrophe between them",
			input:     `'foo'\''bar'`,
			want:      `"foo"\'"bar"`,
			wantSafe:  true,
			wantApply: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, safe, changed := rewriteSingleQuotedBashArgs(tt.input)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantSafe, safe)
			assert.Equal(t, tt.wantApply, changed)
		})
	}
}
