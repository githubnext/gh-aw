//go:build !integration

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	workflowlint "github.com/github/gh-aw/pkg/workflow/lint"
	"github.com/github/gh-aw/pkg/workflow/taint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type lintBatchParser struct {
	parser  *taint.TypeScriptParser
	batches int
	sources int
}

func (p *lintBatchParser) Parse(ctx context.Context, source string) (*taint.JavaScriptNode, error) {
	return p.parser.Parse(ctx, source)
}

func (p *lintBatchParser) ParseBatch(ctx context.Context, sources []string) ([]taint.JavaScriptParseResult, error) {
	p.batches++
	p.sources += len(sources)
	results, err := p.parser.ParseBatch(ctx, sources)
	p.parser.NodeExecutable = "gh-aw-missing-test-node"
	return results, err
}

func TestJavaScriptLintBatchesAcrossDocuments(t *testing.T) {
	module, err := filepath.Abs("../workflow/taint/javascript")
	require.NoError(t, err)
	if _, err := os.Stat(filepath.Join(module, "node_modules", "@typescript", "native-preview", "package.json")); err != nil {
		t.Skip("requires npm ci --prefix pkg/workflow/taint/javascript")
	}
	parser := &lintBatchParser{parser: &taint.TypeScriptParser{ModuleDirectory: module}}
	options := &taint.JavaScriptOptions{Context: t.Context(), Parser: parser, Mode: taint.JavaScriptFast}
	var files []string
	for _, name := range []string{"first", "second"} {
		file := filepath.Join(t.TempDir(), name+".md")
		content := "---\nenv:\n  TOKEN: '${{ secrets.TOKEN }}'\nmcp-scripts:\n  custom:\n    script: console.log('" + name + "', process.env.TOKEN);\n---\n"
		require.NoError(t, os.WriteFile(file, []byte(content), 0o600))
		files = append(files, file)
	}
	runner, err := workflowlint.New(workflowlint.SecretLeakRule())
	require.NoError(t, err)
	var output bytes.Buffer
	err = runWorkflowLint(t.Context(), files, runner, &output, true, workflowLintOptions{JavaScript: options})
	require.ErrorContains(t, err, "found 2 error(s)")
	assert.Equal(t, 1, parser.batches)
	assert.Equal(t, 2, parser.sources)
	var diagnostics []workflowLintDiagnostic
	require.NoError(t, json.Unmarshal(output.Bytes(), &diagnostics))
	count := 0
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == workflowlint.Error {
			count++
			assert.Equal(t, "mcp-scripts.custom.script.javascript", diagnostic.Path)
		}
	}
	assert.Equal(t, 2, count, "both native ASTs were warmed before Node was made unavailable")
}

func TestJavaScriptLintSkipsUnselectedRulePreparation(t *testing.T) {
	parser := &lintBatchParser{parser: &taint.TypeScriptParser{}}
	file := filepath.Join(t.TempDir(), "workflow.md")
	require.NoError(t, os.WriteFile(file, []byte("---\nmcp-scripts:\n  custom:\n    script: 42\n---\n"), 0o600))
	runner, err := workflowlint.Default().Select("action-pin")
	require.NoError(t, err)
	var output bytes.Buffer
	err = runWorkflowLint(t.Context(), []string{file}, runner, &output, true, workflowLintOptions{
		JavaScript: &taint.JavaScriptOptions{Parser: parser, Mode: taint.JavaScriptFast},
	})
	require.NoError(t, err)
	assert.Zero(t, parser.batches)
}
