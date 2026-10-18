package taint

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNativeRepositoryJavaScriptHelpers(t *testing.T) {
	options := nativeJavaScriptOptions(t)
	root, err := os.OpenRoot(filepath.Join("..", "..", ".."))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, root.Close()) })
	options.Files = root.FS()
	entry := "actions/setup/js/start_mcp_gateway.cjs"
	source, err := javascriptReadSource(options.Files, entry)
	require.NoError(t, err)
	result, err := AnalyzeJavaScript(JavaScriptInput{
		Name: entry, Source: string(source),
		Stdin: []Source{{Reference: "secrets.test_token", Path: "workflow.stdin"}},
	}, SecretLeakRule(), *options)
	require.NoError(t, err)
	// The gateway is not certified safe: subprocesses, configuration converters,
	// dynamic callbacks and other unsupported flows must stay incomplete.
	require.NotEmpty(t, result.Gaps)
	parser, ok := options.Parser.(*TypeScriptParser)
	require.True(t, ok)
	for _, file := range []string{entry, "actions/setup/js/mcp_configuration.cjs"} {
		contents, err := javascriptReadSource(options.Files, file)
		require.NoError(t, err)
		assert.Contains(t, parser.cache, sha256.Sum256(contents), "actual helper AST must be reached, not allowlisted")
	}

}

func TestNativeRepositoryJavaScriptMaskRegistration(t *testing.T) {
	options := nativeJavaScriptOptions(t)
	root, err := os.OpenRoot(filepath.Join("..", "..", ".."))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, root.Close()) })
	options.Files = root.FS()
	secret := []Source{{Reference: "secrets.token", Path: "env.TOKEN"}}
	result, err := AnalyzeJavaScript(JavaScriptInput{
		Name:   "entry.cjs",
		Source: `require("./actions/setup/js/actions_secret_masking.cjs").maskSecret(process.env.TOKEN);`,
		Env:    map[string][]Source{"TOKEN": secret},
	}, SecretLeakRule(), *options)
	require.NoError(t, err)
	assert.Empty(t, result.Findings)
	assert.NotEmpty(t, result.Gaps, "unresolved globals and optional fallback machinery remain incomplete")
}

func TestNativeRepositoryJavaScriptGitOptions(t *testing.T) {
	options := nativeJavaScriptOptions(t)
	root, err := os.OpenRoot(filepath.Join("..", "..", ".."))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, root.Close()) })
	options.Files = root.FS()
	secret := []Source{{Reference: "github.token", Path: "env.GH_TOKEN"}}
	for _, tt := range []struct {
		name, timeout string
		disclose      bool
	}{
		{"public timeout", "60000", false},
		{"secret timeout", "process.env.GH_TOKEN", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result, err := AnalyzeJavaScript(JavaScriptInput{
				Name: "entry.cjs",
				Source: `require("./actions/setup/js/git_helpers.cjs").execGitSync(["fetch","origin"],{
					env:{GH_TOKEN:process.env.GH_TOKEN},timeout:` + tt.timeout + `});`,
				Env: map[string][]Source{"GH_TOKEN": secret},
			}, SecretLeakRule(), *options)
			require.NoError(t, err)
			require.NotEmpty(t, result.Gaps, "child-process results remain explicitly unmodeled")
			assert.Equal(t, tt.disclose, len(result.Findings) != 0, "%+v", result)
		})
	}
}
