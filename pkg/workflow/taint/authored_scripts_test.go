package taint

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/workflow/aw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthoredScriptEntryPoints(t *testing.T) {
	options := nativeJavaScriptOptions(t)
	options.Mode = JavaScriptFast
	document, err := aw.Parse(`---
env:
  TOKEN: '${{ secrets.TOKEN }}'
  HOME: '${{ secrets.HOME }}'
steps:
  - uses: actions/github-script@v7
    with:
      script: return process.env.TOKEN;
post-steps:
  - shell: node
    run: console.log(process.env.TOKEN);
jobs:
  custom:
    steps:
      - shell: bash
        run: node -e 'console.log(process.env.TOKEN)'
safe-outputs:
  jobs:
    custom:
      steps:
        - shell: node
          run: console.log(process.env.TOKEN);
  scripts:
    custom:
      script: return process.env.TOKEN;
mcp-scripts:
  js:
    env:
      LOCAL: '${{ secrets.LOCAL }}'
    script: return process.env.LOCAL;
  shell:
    run: echo "$TOKEN"
  python:
    py: print("not JavaScript")
  go:
    go: package main
graders:
  custom:
    script: return trace.token;
  operational-value:
    script: echo "$TOKEN"
tools:
  cache-memory:
    validation:
      script: console.log(process.env.HOME);
  drive-memory:
    - validation: console.log(process.env.HOME);
  repo-memory:
    - validation-script: console.log(process.env.HOME);
    - custom-validation: console.log(process.env.HOME);
engine:
  id: test
  driver:
    node: console.log(process.env.TOKEN);
  behavior:
    harness-script: console.log(process.env.TOKEN);
    log-parser: 'function parseLog(content) { return { markdown: process.env.TOKEN }; }'
    mcp:
      config-adapter: console.log(process.env.TOKEN);
---
Prompt
`)
	require.NoError(t, err)
	before, err := json.Marshal(document)
	require.NoError(t, err)
	rule := SecretLeakRule()
	isSource := rule.IsSource
	rule.IsSource = func(ref string) bool { return ref == "trace.token" || isSource(ref) }
	require.NoError(t, PrimeAuthoredJavaScript(t.Context(), []*aw.Document{document}, rule, *options))
	parser := options.Parser.(*TypeScriptParser)
	parsed := len(parser.cache)
	rule.JavaScript = options
	result, err := AnalyzeAuthoredDocument(t.Context(), document, rule)
	require.NoError(t, err)
	var paths []string
	for _, finding := range result.Findings {
		paths = append(paths, finding.Site.Path)
	}
	assert.ElementsMatch(t, []string{
		"steps[0].with.script.javascript",
		"post-steps[0].run.javascript",
		"jobs.custom.steps[0].run.javascript",
		"safe-outputs.jobs.custom.steps[0].run.javascript",
		"safe-outputs.scripts.custom.script.javascript",
		"mcp-scripts.js.script.javascript",
		"mcp-scripts.shell.run",
		"graders.custom.script.javascript",
		"graders.operational-value.script",
		"tools.cache-memory.validation.script.javascript",
		"tools.drive-memory[0].validation.javascript",
		"tools.repo-memory[0].validation-script.javascript",
		"tools.repo-memory[1].custom-validation.javascript",
		"engine.driver.node.javascript",
		"engine.behavior.harness-script.javascript",
		"engine.behavior.log-parser.javascript",
		"engine.behavior.mcp.config-adapter.javascript",
	}, paths)
	assert.Len(t, parser.cache, parsed, "all entry scripts were warmed, including Node eval sources")
	for _, language := range []string{"python.py", "go.go"} {
		assert.Contains(t, result.Gaps, CoverageGap{
			Path: "mcp-scripts." + language, Reason: "authored script language is not modeled",
		})
	}
	after, err := json.Marshal(document)
	require.NoError(t, err)
	assert.Equal(t, string(before), string(after), "source AST remains read-only")
}

func TestAuthoredScriptSandboxAndPayloadBoundaries(t *testing.T) {
	options := nativeJavaScriptOptions(t)
	document, err := aw.Parse(`---
env:
  TOKEN: '${{ secrets.TOKEN }}'
graders:
  custom:
    script: 'console.log(process.env.TOKEN); return "${{ secrets.TOKEN }}";'
tools:
  cache-memory:
    validation:
      script: 'console.log(process.env.TOKEN); console.log("${{ secrets.TOKEN }}");'
safe-outputs:
  scripts:
    custom:
      script: 'console.log("${{ secrets.TOKEN }}");'
mcp-scripts:
  custom:
    script: 'console.log("${{ secrets.TOKEN }}");'
---
`)
	require.NoError(t, err)
	rule := SecretLeakRule()
	rule.JavaScript = options
	require.NoError(t, PrimeAuthoredJavaScript(t.Context(), []*aw.Document{document}, rule, *options))
	result, err := AnalyzeAuthoredScripts(t.Context(), document, rule)
	require.NoError(t, err)
	require.Len(t, result.Findings, 2)
	for _, finding := range result.Findings {
		assert.True(t, strings.HasPrefix(finding.Site.Path, "mcp-scripts.") || strings.HasPrefix(finding.Site.Path, "safe-outputs.scripts."))
	}
}

func TestAuthoredScriptSchemaLocations(t *testing.T) {
	document, err := aw.Parse(`---
mcp-scripts:
  custom:
    script: console.log(1)
    inputs:
      data:
        schema:
          properties:
            script:
              default: not code
enclaves:
  - script:
      max-script-bytes: 100
tools:
  unrelated:
    script: not code
---
` + "```javascript\nnot code\n```\n")
	require.NoError(t, err)
	scripts, err := authoredScripts(document)
	require.NoError(t, err)
	require.Len(t, scripts, 1)
	assert.Equal(t, "mcp-scripts.custom.script", scripts[0].path)
	document.Frontmatter["mcp-scripts"] = map[string]any{"custom": map[string]any{"script": 42}}
	_, err = authoredScripts(document)
	require.ErrorContains(t, err, "script must be a string")
	assert.NotContains(t, err.Error(), "42")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = AnalyzeAuthoredScripts(ctx, document, SecretLeakRule())
	require.ErrorIs(t, err, context.Canceled)
}
