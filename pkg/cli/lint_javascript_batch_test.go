//go:build !integration

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

type lintPrimingParser struct {
	batchError error
	onBatch    func()
	batches    int
	parses     int
}

func (p *lintPrimingParser) Parse(ctx context.Context, _ string) (*taint.JavaScriptNode, error) {
	p.parses++
	return &taint.JavaScriptNode{Kind: "SourceFile"}, ctx.Err()
}

func (p *lintPrimingParser) ParseBatch(context.Context, []string) ([]taint.JavaScriptParseResult, error) {
	p.batches++
	if p.onBatch != nil {
		p.onBatch()
	}
	return nil, p.batchError
}

func TestJavaScriptLintPrimingErrorsAreBestEffort(t *testing.T) {
	for _, batchError := range []error{context.DeadlineExceeded, context.Canceled, errors.New("parser unavailable")} {
		t.Run(batchError.Error(), func(t *testing.T) {
			dir := t.TempDir()
			file := filepath.Join(dir, "workflow.lock.yml")
			require.NoError(t, os.WriteFile(file, []byte("jobs:\n  test:\n    runs-on: ubuntu-latest\n    steps:\n      - run: echo '${{ secrets.TOKEN }}'\n"), 0o600))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "workflow.md"), []byte("---\nstrict: false\ncheck-for-updates: false\nmcp-scripts:\n  custom:\n    script: console.log('public');\n---\n"), 0o600))
			parser := &lintPrimingParser{batchError: batchError}
			runner, err := workflowlint.Default().Select("secret-leak", "version-check")
			require.NoError(t, err)
			var output bytes.Buffer
			err = runWorkflowLint(t.Context(), []string{file}, runner, &output, true, workflowLintOptions{
				JavaScript: &taint.JavaScriptOptions{Parser: parser, Mode: taint.JavaScriptFast},
			})
			require.ErrorContains(t, err, "workflow lint found 1 error(s)")
			require.NotErrorIs(t, err, batchError)
			assert.NotContains(t, err.Error(), batchError.Error())
			require.NoError(t, t.Context().Err(), "the parser's internal failure did not cancel the caller")
			assert.Equal(t, 1, parser.batches)
			assert.Equal(t, 1, parser.parses, "uncached sources are parsed during per-file analysis")
			var diagnostics []workflowLintDiagnostic
			require.NoError(t, json.Unmarshal(output.Bytes(), &diagnostics))
			var ruleIDs []string
			for _, diagnostic := range diagnostics {
				ruleIDs = append(ruleIDs, diagnostic.RuleID)
				assert.Equal(t, file, diagnostic.File)
			}
			assert.Contains(t, ruleIDs, "secret-leak")
			assert.Contains(t, ruleIDs, "version-check")
		})
	}
}

func TestJavaScriptLintPrimingReportsValidationErrorsPerFile(t *testing.T) {
	parser := &lintPrimingParser{}
	var files []string
	for _, name := range []string{"first", "second"} {
		file := filepath.Join(t.TempDir(), name+".md")
		require.NoError(t, os.WriteFile(file, []byte("---\nstrict: false\ncheck-for-updates: false\nmcp-scripts:\n  custom:\n    script: 42\n---\n"), 0o600))
		files = append(files, file)
	}
	runner, err := workflowlint.Default().Select("secret-leak", "version-check")
	require.NoError(t, err)
	var output bytes.Buffer
	err = runWorkflowLint(t.Context(), files, runner, &output, true, workflowLintOptions{
		JavaScript: &taint.JavaScriptOptions{Parser: parser, Mode: taint.JavaScriptFast},
	})
	require.Error(t, err)
	assert.Equal(t, len(files), strings.Count(err.Error(), "AW script must be a string"))
	for _, file := range files {
		assert.Contains(t, err.Error(), file+": lint rule secret-leak: AW script must be a string")
	}
	var diagnostics []workflowLintDiagnostic
	require.NoError(t, json.Unmarshal(output.Bytes(), &diagnostics))
	var checkedFiles []string
	for _, diagnostic := range diagnostics {
		assert.Contains(t, files, diagnostic.File)
		if diagnostic.RuleID == "version-check" {
			checkedFiles = append(checkedFiles, diagnostic.File)
		}
	}
	assert.Equal(t, files, checkedFiles, "independent diagnostics survive per-file validation errors")
}

func TestJavaScriptLintPrimingPropagatesCallerCancellation(t *testing.T) {
	for _, duringBatch := range []bool{false, true} {
		t.Run(map[bool]string{false: "before preparation", true: "during batch"}[duringBatch], func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			parser := &lintPrimingParser{onBatch: cancel}
			file := filepath.Join(t.TempDir(), "workflow.md")
			require.NoError(t, os.WriteFile(file, []byte("---\nmcp-scripts:\n  custom:\n    script: console.log('public');\n---\n"), 0o600))
			runner, err := workflowlint.Default().Select("secret-leak")
			require.NoError(t, err)
			if !duringBatch {
				cancel()
			}
			var output bytes.Buffer
			err = runWorkflowLint(ctx, []string{file}, runner, &output, true, workflowLintOptions{
				JavaScript: &taint.JavaScriptOptions{Parser: parser, Mode: taint.JavaScriptFast},
			})
			require.ErrorIs(t, err, context.Canceled)
			assert.Zero(t, parser.parses)
			if duringBatch {
				assert.Equal(t, 1, parser.batches)
			} else {
				assert.Zero(t, parser.batches)
			}
			assert.Empty(t, output.String())
		})
	}
}

func TestJavaScriptLintPrimingPropagatesCallerDeadline(t *testing.T) {
	ctx, cancel := context.WithDeadline(t.Context(), time.Now().Add(-time.Second))
	defer cancel()
	parser := &lintPrimingParser{}
	var output bytes.Buffer
	err := runWorkflowLint(ctx, nil, workflowlint.Default(), &output, true, workflowLintOptions{
		JavaScript: &taint.JavaScriptOptions{Parser: parser, Mode: taint.JavaScriptFast},
	})
	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Zero(t, parser.batches)
	assert.Empty(t, output.String())
}
