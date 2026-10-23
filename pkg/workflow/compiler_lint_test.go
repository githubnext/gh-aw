package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCompilerLintEnabledAndOptOut(t *testing.T) {
	t.Parallel()
	content := `name: Haiku
on: workflow_dispatch
permissions:
  contents: read
jobs:
  print-message:
    runs-on: ubuntu-latest
    timeout-minutes: 1
    steps:
      - run: echo "${{ inputs.message }}"
`
	data := &WorkflowData{}
	compiler := NewCompiler(WithLint(true))
	err := compiler.lintCompiledWorkflow(data, content, "haiku.md")
	require.ErrorContains(t, err, "[template-injection]")
	require.ErrorContains(t, err, "CTR-006")
	require.NoError(t, NewCompiler(WithLint(false)).lintCompiledWorkflow(data, content, "haiku.md"))

	safe := `name: Haiku
on: workflow_dispatch
permissions:
  contents: read
jobs:
  print-message:
    runs-on: ubuntu-latest
    timeout-minutes: 1
    steps:
      - env:
          MESSAGE: ${{ inputs.message }}
        run: printf '%s\n' "$MESSAGE"
`
	require.NoError(t, compiler.lintCompiledWorkflow(data, safe, "haiku.md"))
}

func TestCompilerLintRejectsInvalidGeneratedAST(t *testing.T) {
	t.Parallel()
	require.ErrorContains(t, NewCompiler(WithLint(true)).lintCompiledWorkflow(&WorkflowData{}, "jobs: [", "test.md"), "cannot parse generated Actions AST")
}

func TestCompilerLintPreservesThreatSuppressions(t *testing.T) {
	t.Parallel()
	content := "jobs:\n  test:\n    runs-on: ubuntu-latest\n    steps:\n      - run: echo '${{ inputs.message }}'\n"
	for _, active := range []bool{false, true} {
		expires := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
		if active {
			expires = time.Now().UTC().AddDate(0, 0, 1).Format("2006-01-02")
		}
		data := &WorkflowData{ParsedFrontmatter: &FrontmatterConfig{
			ThreatDetectionSuppressions: []ThreatDetectionSuppression{{
				Rule: "CTR-006", Reason: "Test suppression", Expires: expires,
			}},
		}}
		err := NewCompiler(WithLint(true)).lintCompiledWorkflow(data, content, "test.md")
		if active {
			require.NoError(t, err)
		} else {
			require.ErrorContains(t, err, "[template-injection]")
		}
	}
}

func TestCompilerLintWarningsRemainAdvisory(t *testing.T) {
	t.Parallel()
	content := "jobs:\n  test:\n    runs-on: ubuntu-latest\n    steps:\n      - uses: example/action@main\n"
	compiler := NewCompiler(WithLint(true))
	require.NoError(t, compiler.lintCompiledWorkflow(&WorkflowData{}, content, "test.md"))
	require.Positive(t, compiler.GetWarningCount())
}

func TestCompilerLintRunsBeforeEmission(t *testing.T) {
	t.Parallel()
	for _, noEmit := range []bool{false, true} {
		for _, skipLint := range []bool{false, true} {
			t.Run(fmt.Sprintf("noEmit=%t/skipLint=%t", noEmit, skipLint), func(t *testing.T) {
				t.Parallel()
				dir := t.TempDir()
				path := filepath.Join(dir, "test.md")
				require.NoError(t, os.WriteFile(path, []byte(`---
on: workflow_dispatch
engine: copilot
permissions:
  contents: read
jobs:
  print-message:
    runs-on: ubuntu-latest
    timeout-minutes: 1
    steps:
      - run: echo "${{ secrets.TEST_SECRET }}"
---
Test.
`), 0o600))
				compiler := NewCompiler(WithNoEmit(noEmit), WithLint(!skipLint))
				compiler.SetReadOnly(true)
				compiler.SetSkipValidation(true)
				compiler.SetStrictMode(false)
				data, err := compiler.ParseWorkflowFile(path)
				require.NoError(t, err)
				err = compiler.CompileWorkflowData(data, path)
				if !skipLint {
					require.ErrorContains(t, err, "[secret-leak]")
					require.NoFileExists(t, filepath.Join(dir, "test.lock.yml"))
				} else {
					require.NoError(t, err)
					if noEmit {
						require.NoFileExists(t, filepath.Join(dir, "test.lock.yml"))
					} else {
						require.FileExists(t, filepath.Join(dir, "test.lock.yml"))
					}
				}
			})
		}
	}
}
