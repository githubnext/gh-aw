package taint

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNativeJavaScriptBatchIsolation(t *testing.T) {
	options := nativeJavaScriptOptions(t)
	parser := options.Parser.(*TypeScriptParser)
	sources := []string{`console.log("first")`, `const broken =`, `console.log("first")`}
	for i := range 65 {
		sources = append(sources, fmt.Sprintf(`const sameName = %d; console.log(sameName)`, i))
	}
	sources = append(sources, strings.Repeat(" ", 1024*1024+1))
	results, err := parser.ParseBatch(t.Context(), sources)
	require.NoError(t, err)
	require.Len(t, results, len(sources))
	assert.Same(t, results[0].Node, results[2].Node)
	require.Error(t, results[1].Err)
	assert.NotContains(t, results[1].Err.Error(), "broken")
	require.ErrorContains(t, results[len(results)-1].Err, "size limit")
	for i, result := range results {
		if i == 1 || i == len(results)-1 {
			continue
		}
		require.NoError(t, result.Err)
		require.NotNil(t, result.Node)
		assert.Equal(t, "SourceFile", result.Node.Kind)
	}
	assert.Len(t, parser.cache, 66)
	parser.NodeExecutable = "gh-aw-missing-test-node"
	cached, err := parser.Parse(t.Context(), sources[0])
	require.NoError(t, err)
	assert.Same(t, results[0].Node, cached)
	_, err = parser.Parse(t.Context(), sources[1])
	require.Error(t, err)
	assert.Equal(t, results[1].Err, err, "syntax failures are cached, too")
}

func TestNativeJavaScriptBatchCancellation(t *testing.T) {
	parser := &TypeScriptParser{}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := parser.ParseBatch(ctx, []string{"console.log(1)"})
	require.ErrorIs(t, err, context.Canceled)
}

func TestNativeJavaScriptBatchUnavailable(t *testing.T) {
	parser := &TypeScriptParser{NodeExecutable: "gh-aw-missing-test-node", ModuleDirectory: t.TempDir()}
	results, err := parser.ParseBatch(t.Context(), []string{"const first = 1", "const second = 2"})
	require.NoError(t, err)
	require.Len(t, results, 2)
	for _, result := range results {
		require.Error(t, result.Err)
		assert.Nil(t, result.Node)
	}
	assert.Len(t, parser.failures, 2)
}

func BenchmarkNativeJavaScriptBatch(b *testing.B) {
	options := nativeJavaScriptOptions(b)
	// The integration dependency is explicit, just as it is for lint callers.
	module := options.Parser.(*TypeScriptParser).ModuleDirectory
	sources := make([]string, 50)
	for i := range sources {
		sources[i] = fmt.Sprintf("const value = %d; console.log(value)", i)
	}
	b.ResetTimer()
	for b.Loop() {
		parser := &TypeScriptParser{ModuleDirectory: module}
		results, err := parser.ParseBatch(b.Context(), sources)
		require.NoError(b, err)
		for _, result := range results {
			require.NoError(b, result.Err)
		}
	}
}
