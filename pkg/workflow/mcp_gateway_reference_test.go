//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMCPGatewayReferenceExamplesCompile(t *testing.T) {
	pagePath := filepath.Join("..", "..", "docs", "src", "content", "docs", "reference", "mcp-gateway.md")
	page, err := os.ReadFile(pagePath)
	require.NoError(t, err)

	for _, test := range []struct {
		name string
		want []string
	}{
		{
			name: "trusted-bots",
			want: []string{`"trustedBots": ["github-actions[bot]", "copilot-swe-agent[bot]"]`},
		},
		{
			name: "oidc-mcp",
			want: []string{
				`"url": "https://my-server.example.com/mcp"`,
				`"type": "github-oidc"`,
				`"audience": "https://my-server.example.com"`,
				"id-token: write",
				"-e ACTIONS_ID_TOKEN_REQUEST_URL",
				"-e ACTIONS_ID_TOKEN_REQUEST_TOKEN",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			fence := "```aw wrap title=\".github/workflows/" + test.name + ".md\"\n"
			require.Equal(t, 1, strings.Count(string(page), fence))
			_, rest, found := strings.Cut(string(page), fence)
			require.True(t, found)
			example, _, found := strings.Cut(rest, "\n```")
			require.True(t, found, "reference example must have a closing fence")

			dir := t.TempDir()
			path := filepath.Join(dir, test.name+".md")
			require.NoError(t, os.WriteFile(path, []byte(example+"\n"), 0o600))
			compiler := NewCompiler()
			require.NoError(t, compiler.CompileWorkflow(path))

			lock, err := os.ReadFile(filepath.Join(dir, test.name+".lock.yml"))
			require.NoError(t, err)
			for _, want := range test.want {
				require.Contains(t, string(lock), want)
			}
		})
	}
}
