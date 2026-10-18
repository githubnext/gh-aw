package taint

import (
	"context"
	"crypto/sha256"
	"errors"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/github/gh-aw/pkg/workflow/actions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"mvdan.cc/sh/v3/syntax"
)

func TestJavaScriptLocatorIdentity(t *testing.T) {
	for _, tt := range []struct {
		code string
		want bool
	}{
		{`NODE=$(which node 2>/dev/null || command -v node 2>/dev/null || echo node)`, true},
		{`NODE=/untrusted/node`, false},
		{`NODE=$(echo /untrusted/node)`, false},
		{`NODE=$(which node || echo node); NODE=/untrusted/node`, false},
	} {
		tree, err := syntax.NewParser().Parse(strings.NewReader(tt.code), "")
		require.NoError(t, err)
		_, found := javascriptExecutables(tree)["NODE"]
		assert.Equal(t, tt.want, found, tt.code)
	}
}

func TestJavaScriptRuntimePathConfinement(t *testing.T) {
	mapping := map[string]string{"${RUNNER_TEMP}/actions": "assets"}
	for _, name := range []string{
		"${RUNNER_TEMP}/actions/../outside.cjs",
		"${RUNNER_TEMP}/actions/nested/../../outside.cjs",
		"${RUNNER_TEMP}/actions//outside.cjs",
	} {
		target, matched := javascriptRuntimePath(mapping, name)
		assert.True(t, matched)
		assert.Empty(t, target)
	}
	options := nativeJavaScriptOptions(t)
	options.RuntimePaths = mapping
	options.Files = fstest.MapFS{
		"outside.cjs": {Data: []byte(`console.log(process.env.TOKEN)`)},
	}
	result, err := AnalyzeJavaScript(JavaScriptInput{
		Name: "entry.cjs", Source: `require(process.env.RUNNER_TEMP+"/actions/../outside.cjs")`,
		Env: map[string][]Source{"TOKEN": {{Reference: "secrets.token"}}},
	}, SecretLeakRule(), *options)
	require.NoError(t, err)
	assert.Empty(t, result.Findings)
	require.NotEmpty(t, result.Gaps)
}

func TestJavaScriptCyclicModules(t *testing.T) {
	options := nativeJavaScriptOptions(t)
	options.Files = fstest.MapFS{
		"a.js": {Data: []byte(`require("./b")`)},
		"b.js": {Data: []byte(`require("./a")`)},
	}
	result, err := AnalyzeJavaScript(JavaScriptInput{Name: "entry.cjs", Source: `require("./a")`}, SecretLeakRule(), *options)
	require.NoError(t, err)
	require.NotEmpty(t, result.Gaps)
	assert.Contains(t, result.Gaps[0].Reason, "cyclic")
}

func TestNativeJavaScriptCancellation(t *testing.T) {
	parser := &TypeScriptParser{cache: map[[32]byte]*JavaScriptNode{
		sha256.Sum256([]byte("safe")): {Kind: "SourceFile"},
	}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := parser.Parse(ctx, "safe")
	require.ErrorIs(t, err, context.Canceled)
	parser.once.Do(func() { parser.gate = make(chan struct{}, 1) })
	parser.gate <- struct{}{}
	ctx, cancel = context.WithCancel(context.Background())
	finished := make(chan error, 1)
	go func() {
		_, err := parser.Parse(ctx, "safe")
		finished <- err
	}()
	cancel()
	select {
	case err := <-finished:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("canceled parser waited for another compilation")
	}
	<-parser.gate
}

type javascriptParserFunc func(context.Context, string) (*JavaScriptNode, error)

func (f javascriptParserFunc) Parse(ctx context.Context, text string) (*JavaScriptNode, error) {
	return f(ctx, text)
}

func TestJavaScriptEvaluationCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	parser := javascriptParserFunc(func(context.Context, string) (*JavaScriptNode, error) {
		cancel()
		return &JavaScriptNode{Kind: "SourceFile"}, nil
	})
	_, err := AnalyzeJavaScript(JavaScriptInput{Name: "entry.cjs"}, SecretLeakRule(), JavaScriptOptions{Context: ctx, Parser: parser})
	require.ErrorIs(t, err, context.Canceled)
}

func TestUnavailableJavaScriptDoesNotInventOutputLeaks(t *testing.T) {
	workflow, err := actions.Parse([]byte(`jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - env:
          TOKEN: ${{ secrets.TOKEN }}
        run: |
          output=$(printf '%s' "$TOKEN" | node unavailable.cjs)
          echo "$output"
`))
	require.NoError(t, err)
	for _, options := range []*JavaScriptOptions{
		nil,
		{Parser: javascriptParserFunc(func(context.Context, string) (*JavaScriptNode, error) {
			return nil, errors.New("unavailable")
		})},
	} {
		rule := SecretLeakRule()
		rule.JavaScript = options
		result, err := Analyze(workflow, rule)
		require.NoError(t, err)
		assert.Empty(t, result.Findings)
		assert.NotEmpty(t, result.Gaps)
	}
}
