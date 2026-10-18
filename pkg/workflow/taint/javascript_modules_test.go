package taint

import (
	"crypto/sha256"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJavaScriptFastRepositoryModuleProvenance(t *testing.T) {
	for _, name := range []string{"unreviewed.cjs", "error_helpers.cjs"} {
		t.Run(name, func(t *testing.T) {
			options := nativeJavaScriptOptions(t)
			options.Mode = JavaScriptFast
			target := "actions/setup/js/" + name
			options.Files = fstest.MapFS{
				target: {Data: []byte(`module.exports.consume = token => console.error(token)`)},
			}
			result, err := AnalyzeJavaScript(JavaScriptInput{
				Name: "entry.cjs", Source: `require("./` + target + `").consume(process.env.TOKEN)`,
				Env: map[string][]Source{"TOKEN": {{Reference: "secrets.token", Path: "env.TOKEN"}}},
			}, SecretLeakRule(), *options)
			require.NoError(t, err)
			require.Len(t, result.Findings, 1)
			assert.Equal(t, "entry.cjs.javascript["+target+"]", result.Findings[0].Site.Path)
			assert.Empty(t, result.Gaps)
		})
	}
}

func TestJavaScriptSummaryUsesResolvedModule(t *testing.T) {
	for _, mapped := range []bool{false, true} {
		name := "repository"
		if mapped {
			name = "mapped runtime"
		}
		t.Run(name, func(t *testing.T) {
			options := summaryTestOptions(t)
			if !mapped {
				options.RuntimePaths = nil
			}
			files := summaryTestFiles(t)
			source := []byte(`module.exports.getErrorMessage = error => {console.error(process.env.TOKEN); return "error"}`)
			files["actions/setup/js/error_helpers"] = &fstest.MapFile{Data: source}
			options.Files = files
			result, err := AnalyzeJavaScript(JavaScriptInput{
				Name:   "entry.cjs",
				Source: `require("./actions/setup/js/error_helpers").getErrorMessage("public")`,
				Env:    map[string][]Source{"TOKEN": {{Reference: "secrets.token", Path: "env.TOKEN"}}},
			}, SecretLeakRule(), *options)
			require.NoError(t, err)
			if mapped {
				assert.Empty(t, result.Findings)
				assert.NotEmpty(t, result.Gaps, "an exact unreviewed file cannot use its reviewed .cjs sibling's summary")
				assert.NotContains(t, options.Parser.(*TypeScriptParser).cache, sha256.Sum256(source))
			} else {
				require.Len(t, result.Findings, 1)
				assert.Equal(t, "entry.cjs.javascript[actions/setup/js/error_helpers]", result.Findings[0].Site.Path)
				assert.Empty(t, result.Gaps)
			}
		})
	}
}

func TestJavaScriptSummaryExtensionlessReviewedModule(t *testing.T) {
	options := summaryTestOptions(t)
	result, err := AnalyzeJavaScript(JavaScriptInput{
		Name:   "entry.cjs",
		Source: `console.error(require("./actions/setup/js/error_helpers").getErrorMessage(process.env.TOKEN))`,
		Env:    map[string][]Source{"TOKEN": {{Reference: "secrets.token", Path: "env.TOKEN"}}},
	}, SecretLeakRule(), *options)
	require.NoError(t, err)
	require.Len(t, result.Findings, 1)
	assert.Empty(t, result.Gaps)
}
