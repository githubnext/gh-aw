package taint

import (
	"crypto/sha256"
	"testing"
	"testing/fstest"

	"github.com/github/gh-aw/pkg/workflow/actions"
	"github.com/github/gh-aw/pkg/workflow/aw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJavaScriptFastPairedGraph(t *testing.T) {
	options := nativeJavaScriptOptions(t)
	options.Mode = JavaScriptFast
	document, err := aw.Parse(`---
env:
  TOKEN: '${{ secrets.TOKEN }}'
steps:
  - shell: bash
    run: echo "$TOKEN"
  - uses: actions/github-script@v7
    with:
      script: console.log(process.env.TOKEN);
mcp-scripts:
  custom:
    script: return process.env.TOKEN;
---
Prompt
`)
	require.NoError(t, err)
	generated := `unknownGeneratedRuntime(process.env.TOKEN);`
	compiled, err := actions.Parse([]byte(`jobs:
  agent:
    runs-on: ubuntu-latest
    env:
      TOKEN: '${{ secrets.TOKEN }}'
    steps:
      - shell: bash
        run: echo "$TOKEN"
      - uses: actions/github-script@v7
        with:
          script: ` + generated + `
  producer:
    runs-on: ubuntu-latest
    outputs:
      token: '${{ secrets.JOB_TOKEN }}'
    steps:
      - run: "true"
  consumer:
    runs-on: ubuntu-latest
    needs: producer
    steps:
      - shell: bash
        run: echo '${{ needs.producer.outputs.token }}'
`))
	require.NoError(t, err)
	rule := SecretLeakRule()
	rule.JavaScript = options
	require.NoError(t, PrimeAuthoredJavaScript(t.Context(), []*aw.Document{document}, rule, *options))
	result, err := AnalyzeDocument(document, compiled, rule)
	require.NoError(t, err)
	var paths []string
	for _, finding := range result.Findings {
		paths = append(paths, finding.Site.Path)
	}
	assert.Contains(t, paths, "jobs.agent.steps[0].run", "compiled shell checking remains enabled")
	assert.Contains(t, paths, "jobs.consumer.steps[0].run", "compiled cross-job flows remain enabled")
	assert.Contains(t, paths, "steps[1].with.script.javascript")
	assert.Contains(t, paths, "mcp-scripts.custom.script.javascript")
	assert.NotContains(t, paths, "steps[0].run", "ordinary source findings do not duplicate compiled findings")
	parser := options.Parser.(*TypeScriptParser)
	assert.NotContains(t, parser.cache, sha256.Sum256([]byte(generated)))
	repeated, err := AnalyzeDocument(document, compiled, rule)
	require.NoError(t, err)
	assert.Equal(t, result, repeated)
	options.Mode = JavaScriptDeep
	_, err = AnalyzeDocument(document, compiled, rule)
	require.NoError(t, err)
	assert.Contains(t, parser.cache, sha256.Sum256([]byte(generated)), "explicit deep mode still checks generated JavaScript")
}

func TestJavaScriptFastAuthoredShellDelta(t *testing.T) {
	options := nativeJavaScriptOptions(t)
	options.Mode = JavaScriptFast
	script := `value=$(node -e 'process.stdout.write(process.env.TOKEN)'); echo "$value"`
	document := &aw.Document{Frontmatter: map[string]any{
		"env":   map[string]any{"TOKEN": "${{ secrets.TOKEN }}"},
		"steps": []any{map[string]any{"shell": "bash", "run": script}},
	}}
	compiled, err := actions.Parse([]byte("jobs:\n  agent:\n    runs-on: ubuntu-latest\n    env:\n      TOKEN: '${{ secrets.TOKEN }}'\n    steps:\n      - shell: bash\n        run: |\n          " + script + "\n"))
	require.NoError(t, err)
	rule := SecretLeakRule()
	rule.JavaScript = options
	result, err := AnalyzeDocument(document, compiled, rule)
	require.NoError(t, err)
	require.Len(t, result.Findings, 1)
	assert.Equal(t, "steps[0].run", result.Findings[0].Site.Path)
}

func TestJavaScriptAuthoredDriverSource(t *testing.T) {
	options := nativeJavaScriptOptions(t)
	options.Mode = JavaScriptFast
	options.Files = fstest.MapFS{
		"drivers/main.cjs":   {Data: []byte(`require("./helper.cjs").consume(process.env.TOKEN)`)},
		"drivers/helper.cjs": {Data: []byte(`module.exports = {consume: token => console.log(token)}`)},
	}
	document := &aw.Document{Frontmatter: map[string]any{
		"env":    map[string]any{"TOKEN": "${{ secrets.TOKEN }}"},
		"engine": map[string]any{"driver": "drivers/main.cjs"},
	}}
	rule := SecretLeakRule()
	rule.JavaScript = options
	result, err := AnalyzeAuthoredScripts(t.Context(), document, rule)
	require.NoError(t, err)
	require.Len(t, result.Findings, 1)
	assert.Equal(t, "engine.driver.javascript[drivers/helper.cjs]", result.Findings[0].Site.Path)
}

func TestAuthoredOperationalGraderEncoding(t *testing.T) {
	document, err := aw.Parse(`---
graders:
  operational-value:
    script: |
      echo '${{ secrets.NOT_INTERPOLATED }}'
      echo "$GH_TOKEN"
---
`)
	require.NoError(t, err)
	result, err := AnalyzeAuthoredScripts(t.Context(), document, SecretLeakRule())
	require.NoError(t, err)
	require.Len(t, result.Findings, 1)
	require.Len(t, result.Findings[0].Sources, 1)
	assert.Equal(t, "github.token", result.Findings[0].Sources[0].Reference)
}
