//go:build !integration

package cli

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/scanfindings"
	"github.com/github/gh-aw/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDryRunScannerDifferential(t *testing.T) {
	t.Parallel()
	finding := func(rule string, severity scanfindings.SeverityLevel, line int) scanfindings.Finding {
		return scanfindings.Finding{RuleID: rule, Severity: severity, File: "missing.lock.yml", Line: line, Message: rule}
	}
	medium := finding("medium", scanfindings.SeverityMedium, 1)
	high := finding("high", scanfindings.SeverityHigh, 2)
	low := finding("low", scanfindings.SeverityLow, 3)
	info := finding("info", scanfindings.SeverityInfo, 4)
	unknown := finding("unknown", scanfindings.SeverityUnknown, 5)
	toError := func(findings []scanfindings.Finding) error {
		if len(findings) == 0 {
			return nil
		}
		return withScannerFindings(errors.New("scanner findings"), findings)
	}
	for _, test := range []struct {
		name                     string
		before, after            []scanfindings.Finding
		existing, added, ignored int
	}{
		{"unchanged", []scanfindings.Finding{medium, high}, []scanfindings.Finding{medium, high}, 2, 0, 0},
		{"new medium", nil, []scanfindings.Finding{medium}, 0, 1, 0},
		{"new high", nil, []scanfindings.Finding{high}, 0, 1, 0},
		{"new unknown is not low", nil, []scanfindings.Finding{unknown}, 0, 1, 0},
		{"low and info remain nonblocking", nil, []scanfindings.Finding{low, info}, 0, 0, 2},
		{"removed", []scanfindings.Finding{medium}, nil, 0, 0, 0},
		{"same total different finding", []scanfindings.Finding{medium}, []scanfindings.Finding{high}, 0, 1, 0},
		{"duplicate occurrence added", []scanfindings.Finding{medium}, []scanfindings.Finding{medium, medium}, 1, 1, 0},
		{"severity escalation", []scanfindings.Finding{low}, []scanfindings.Finding{
			finding("low", scanfindings.SeverityHigh, 3),
		}, 0, 1, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			baseline := &dryRunScannerBaseline{results: map[string]error{"zizmor": toError(test.before)}}
			diff, err := baseline.filter("zizmor", toError(test.after))
			assert.Equal(t, test.added > 0, err != nil)
			require.NotNil(t, diff)
			assert.Equal(t, &DryRunScannerDifferential{
				Baseline: len(test.before), Existing: test.existing, Added: test.added, Ignored: test.ignored,
			}, diff)
		})
	}
	for _, tool := range []string{"zizmor", "poutine", "actionlint", "runner-guard", "grype", "grant", "yamllint", "shellcheck"} {
		t.Run(tool+" operational failures remain fatal", func(t *testing.T) {
			t.Parallel()
			failure := errors.New("scanner unavailable or malformed output")
			baseline := &dryRunScannerBaseline{results: map[string]error{tool: failure}}
			_, err := baseline.filter(tool, nil)
			require.ErrorIs(t, err, failure)
			baseline.results[tool] = nil
			_, err = baseline.filter(tool, failure)
			require.ErrorIs(t, err, failure)
		})
	}
}

func TestScannerFindingIdentitySurvivesLineAndStepShifts(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "test.lock.yml")
	workflow := `on: workflow_dispatch
jobs:
  agent:
    runs-on: ubuntu-latest
    steps:
      - name: Existing step
        id: existing
        run: echo unsafe
`
	require.NoError(t, os.WriteFile(path, []byte(workflow), 0600))
	finding := scanfindings.Finding{
		RuleID: "test", Severity: scanfindings.SeverityMedium, Message: "unsafe",
		File: path, Line: 8, Column: 9,
	}
	before := scannerFindingIdentity(finding)
	changed := "# Added header\n" + strings.Replace(workflow, "    steps:\n", "    steps:\n      - name: New step\n        run: echo safe\n", 1)
	require.NoError(t, os.WriteFile(path, []byte(changed), 0600))
	finding.Line += 3
	assert.Equal(t, before, scannerFindingIdentity(finding), "line and step index changes are not new findings")
	changed = strings.Replace(changed, "id: existing", "id: different", 1)
	require.NoError(t, os.WriteFile(path, []byte(changed), 0600))
	assert.NotEqual(t, before, scannerFindingIdentity(finding), "a finding in another logical step is new")
}

func TestDryRunZizmorUsesSharedResultFilter(t *testing.T) {
	stdout := `[
{"ident":"secrets-outside-env","desc":"secret outside environment","determinations":{"severity":"Medium"},"locations":[{"symbolic":{"key":{"Local":{"verbatim_path":"test.lock.yml"}}},"concrete":{"location":{"start_point":{"row":1}}}}]},
{"ident":"self-repository","desc":"self reference","determinations":{"severity":"Low"},"locations":[{"symbolic":{"key":{"Local":{"verbatim_path":"test.lock.yml"}}},"concrete":{"location":{"start_point":{"row":2}}}}]},
{"ident":"template-injection","desc":"unsafe input","determinations":{"severity":"High"},"locations":[{"symbolic":{"key":{"Local":{"verbatim_path":"test.lock.yml"}}},"concrete":{"location":{"start_point":{"row":3}}}}]}
]`
	var findings []scanfindings.Finding
	var count, high int
	var err error
	output := captureStderrForGuardPolicyReportTest(func() {
		count, high, err = parseAndDisplayZizmorOutputWithOptions(stdout, "", false, zizmorResultOptions{dryRun: true, findings: &findings})
	})
	require.NoError(t, err)
	assert.Equal(t, 2, count)
	assert.Equal(t, 1, high)
	require.Len(t, findings, 2)
	assert.NotContains(t, output, "secrets-outside-env")
	assert.Contains(t, output, "test.lock.yml:3:1: info: [Low] self-repository")
	assert.Contains(t, output, "test.lock.yml:4:1: error: [High] template-injection")
	captureStderrForGuardPolicyReportTest(func() {
		count, high, err = parseAndDisplayZizmorOutput(stdout, "", false)
	})
	require.NoError(t, err)
	assert.Equal(t, 3, count, "ordinary compilation must not ignore secrets-outside-env")
	assert.Equal(t, 1, high)
	_, _, err = parseAndDisplayZizmorOutputWithOptions("invalid JSON", "", false, zizmorResultOptions{dryRun: true})
	require.Error(t, err)
}

func TestDevelopmentCompileScannerBaselineBeforeEmission(t *testing.T) {
	source := filepath.Join(t.TempDir(), "differential.md")
	lock := strings.TrimSuffix(source, ".md") + ".lock.yml"
	require.NoError(t, os.WriteFile(source, []byte(`---
on: workflow_dispatch
engine: copilot
safe-outputs:
  noop:
---
Say Done.
`), 0600))
	normal := "name: baseline\non: workflow_dispatch\njobs: {}\n"
	require.NoError(t, os.WriteFile(lock, []byte(normal), 0600))
	originalZizmor := runBatchDryRunZizmorOnFiles
	originalShellcheck := runBatchShellcheckOnLockFilesAndResources
	t.Cleanup(func() {
		runBatchDryRunZizmorOnFiles = originalZizmor
		runBatchShellcheckOnLockFilesAndResources = originalShellcheck
	})
	calls := 0
	runBatchDryRunZizmorOnFiles = func(files []string, _, strict bool) error {
		require.True(t, strict)
		require.Equal(t, []string{lock}, files)
		calls++
		content, err := os.ReadFile(lock)
		require.NoError(t, err)
		if calls == 1 {
			assert.Equal(t, normal, string(content))
		} else {
			assert.Contains(t, string(content), "dry_run")
		}
		return withScannerFindings(errors.New("existing finding"), []scanfindings.Finding{{
			RuleID: "existing", Severity: scanfindings.SeverityMedium, Message: "existing",
			File: "unchanged-image@sha256:123", Line: 1,
		}})
	}
	runBatchShellcheckOnLockFilesAndResources = func(context.Context, []string, []workflow.ShellScriptResource, bool, bool) error { return nil }
	stdout, _ := captureOutput(t, func() error {
		_, err := CompileWorkflows(context.Background(), CompileConfig{
			MarkdownFiles: []string{source}, DryRun: true, Zizmor: true,
			JSONOutput: true, activeModels: &activeModelInventory{},
		})
		require.NoError(t, err)
		return err
	})
	assert.Equal(t, 2, calls)
	var results []ValidationResult
	require.NoError(t, json.Unmarshal([]byte(stdout), &results), stdout)
	require.NotEmpty(t, results)
	report := results[len(results)-1].DryRun
	require.NotNil(t, report)
	assert.Equal(t, "passed", report.Gate)
	assert.Equal(t, "pre_compile_lock_files", report.WarningBaseline)
	assert.Equal(t, &DryRunScannerDifferential{Baseline: 1, Existing: 1}, report.Scanners["zizmor"].Differential)
	assert.Equal(t, "passed", report.Scanners["zizmor"].Status)
}

func TestDryRunLowSeverityWorkflowWarnings(t *testing.T) {
	t.Parallel()
	for _, compilerWarning := range []bool{false, true} {
		t.Run(strconv.FormatBool(compilerWarning), func(t *testing.T) {
			t.Parallel()
			config := CompileConfig{DryRun: true}
			config.dryRunReport = newDryRunCompileReport(config)
			compiler := workflow.NewCompiler()
			if compilerWarning {
				compiler.IncrementWarningCount()
			}
			stats := &CompilationStats{Succeeded: 1, Warnings: 1}
			results := []ValidationResult{{
				Workflow: "low.md", Valid: true,
				Warnings: []ValidationIssue{{Type: "low", Message: "advice", Severity: scanfindings.SeverityLow}},
			}}
			err := enforceDevelopmentDiagnostics(config, compiler, stats, &results)
			assert.Equal(t, compilerWarning, err != nil, "low workflow warnings must not hide unclassified compiler warnings")
			appendDryRunCompileReport(config, stats, &results)
			assert.Equal(t, !compilerWarning, results[len(results)-1].Valid)
			assert.True(t, results[0].Valid)
			require.Len(t, results[0].Warnings, 1)
			output, outputErr := formatValidationOutput(results)
			require.NoError(t, outputErr)
			assert.Contains(t, output, `"severity": "low"`)
		})
	}
}

func TestShellcheckDifferentialUsesScriptLocations(t *testing.T) {
	t.Parallel()
	step := runStepInfo{
		LockFile: "test.lock.yml", JobID: "agent", Name: "Check",
		Script: "echo $value\n",
	}
	before := withShellcheckFindings(errors.New("findings"), step, "script:1:6: warning: quote this [SC2086]\n")
	baseline := &dryRunScannerBaseline{results: map[string]error{"shellcheck": before}}
	step.Script = "# added line\n" + step.Script
	after := withShellcheckFindings(errors.New("findings"), step, "script:2:6: warning: quote this [SC2086]\n")
	diff, err := baseline.filter("shellcheck", after)
	require.NoError(t, err)
	assert.Equal(t, 1, diff.Existing)
	for _, output := range []string{"unparseable", "script:999:1: warning: quote this [SC2086]\n"} {
		_, err := baseline.filter("shellcheck", withShellcheckFindings(errors.New("bad output"), step, output))
		require.Error(t, err)
	}
}

func TestShellcheckDifferentialNativeFindings(t *testing.T) {
	if !isShellcheckAvailable() {
		t.Skip("native ShellCheck is not installed")
	}
	step := runStepInfo{
		Name: "Quote value", JobID: "agent", LockFile: "native.lock.yml",
		Script: "#!/bin/bash\necho \"$missing\"\n",
	}
	output, err := runShellcheckOnScript(step, nil, false)
	require.Error(t, err, string(output))
	var findings *scannerFindingsError
	require.ErrorAs(t, err, &findings, string(output))
	require.NotEmpty(t, findings.diagnostics)
	assert.Equal(t, "SC2154", findings.diagnostics[0].Finding.RuleID)
	baseline := &dryRunScannerBaseline{results: map[string]error{"shellcheck": err}}
	step.Script = "#!/bin/bash\n# leading comment\necho \"$missing\"\n"
	_, currentErr := runShellcheckOnScript(step, nil, false)
	diff, err := baseline.filter("shellcheck", currentErr)
	require.NoError(t, err)
	assert.Positive(t, diff.Existing)
}
