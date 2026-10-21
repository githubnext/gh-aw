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

func TestScannerFindingIdentityResolvesPathsFromGitRoot(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, initTestGitRepo(root))
	lockDir := filepath.Join(root, ".github", "workflows")
	nestedDir := filepath.Join(lockDir, "nested")
	require.NoError(t, os.MkdirAll(nestedDir, 0755))
	lock := filepath.Join(lockDir, "test.lock.yml")
	content := "jobs:\n  agent:\n    steps:\n      - id: check\n        run: echo unsafe\n"
	require.NoError(t, os.WriteFile(lock, []byte(content), 0600))

	oldWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(nestedDir))
	t.Cleanup(func() { require.NoError(t, os.Chdir(oldWD)) })

	relative := scanfindings.Finding{RuleID: "test", Message: "unsafe", File: ".github/workflows/test.lock.yml", Line: 6}
	absolute := relative
	absolute.File = lock
	assert.Equal(t, scannerFindingIdentity(absolute), scannerFindingIdentity(relative))
}

func TestActionlintShellcheckIdentitySurvivesLeadingScriptComment(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "shellcheck.lock.yml")
	before := `jobs:
  agent:
    runs-on: ubuntu-latest
    steps:
      - id: check
        run: |
          echo "$value"
`
	require.NoError(t, os.WriteFile(path, []byte(before), 0600))
	lineFor := func(content, text string) int {
		offset := strings.Index(content, text)
		require.NotEqual(t, -1, offset)
		return strings.Count(content[:offset], "\n") + 1
	}
	issue := actionlintError{
		Kind: "shellcheck", Filepath: path, Line: lineFor(before, `echo "$value"`),
		Message: "SC2086:warning:1:6: quote this",
	}
	beforeIdentity, ok := actionlintShellcheckIdentity(issue)
	require.True(t, ok)

	after := strings.Replace(before, "          echo \"$value\"", "          # harmless comment\n          echo \"$value\"", 1)
	require.NoError(t, os.WriteFile(path, []byte(after), 0600))
	issue.Line = lineFor(after, `echo "$value"`)
	issue.Message = "SC2086:warning:2:6: quote this"
	afterIdentity, ok := actionlintShellcheckIdentity(issue)
	require.True(t, ok)
	assert.Equal(t, beforeIdentity, afterIdentity)
}

func TestScannerFindingIdentityAST(t *testing.T) {
	t.Parallel()
	wrap := func(steps string) string {
		return "on: workflow_dispatch\njobs:\n  agent:\n    runs-on: ubuntu-latest\n    steps:\n" + steps
	}
	for _, test := range []struct {
		name, extension, before, after, beforeTarget, afterTarget string
		same                                                      bool
	}{
		{
			name: "quotes comments and indentation", same: true,
			before:       wrap("      - id: check\n        run: echo unsafe\n"),
			after:        wrap("      - id: check\n        run: 'echo unsafe' # comment\n"),
			beforeTarget: "echo unsafe", afterTarget: "'echo unsafe'",
		},
		{
			name: "flow mapping columns distinguish fields", same: false,
			before:       wrap("      - id: check\n        env: {FIRST: unsafe, SECOND: unsafe}\n        run: echo ok\n"),
			after:        wrap("      - id: check\n        env: {FIRST: unsafe, SECOND: unsafe}\n        run: echo ok\n"),
			beforeTarget: "FIRST", afterTarget: "SECOND",
		},
		{
			name: "flow mapping reordering", same: true,
			before:       wrap("      - id: check\n        env: {FIRST: unsafe, SECOND: safe}\n        run: echo ok\n"),
			after:        wrap("      - id: check\n        env: {SECOND: safe, FIRST: unsafe}\n        run: echo ok\n"),
			beforeTarget: "FIRST", afterTarget: "FIRST",
		},
		{
			name: "changed value at unchanged key", same: false,
			before:       wrap("      - id: check\n        env:\n          TOKEN: before\n        run: echo ok\n"),
			after:        wrap("      - id: check\n        env:\n          TOKEN: after\n        run: echo ok\n"),
			beforeTarget: "TOKEN:", afterTarget: "TOKEN:",
		},
		{
			name: "duplicate names do not conflate steps", same: false,
			before:       wrap("      - name: duplicate\n        env: {TOKEN: unsafe}\n        run: echo first\n      - name: duplicate\n        env: {TOKEN: safe}\n        run: echo second\n"),
			after:        wrap("      - name: duplicate\n        env: {TOKEN: safe}\n        run: echo first\n      - name: duplicate\n        env: {TOKEN: unsafe}\n        run: echo second\n"),
			beforeTarget: "TOKEN: unsafe", afterTarget: "TOKEN: unsafe",
		},
		{
			name: "anonymous action inputs do not conflate steps", same: false,
			before:       wrap("      - uses: actions/checkout@v4\n        with: {path: first}\n"),
			after:        wrap("      - uses: actions/checkout@v4\n        with: {path: second}\n"),
			beforeTarget: "uses:", afterTarget: "uses:",
		},
		{
			name: "anonymous step shifts", same: true,
			before:       wrap("      - run: echo unsafe\n"),
			after:        wrap("      - run: echo safe\n      - run: echo unsafe\n"),
			beforeTarget: "echo unsafe", afterTarget: "echo unsafe",
		},
		{
			name: "block script comment insertion", same: true,
			before:       wrap("      - id: check\n        run: |\n          echo unsafe\n"),
			after:        wrap("      - id: check\n        run: |\n          # harmless comment\n          echo unsafe\n"),
			beforeTarget: "echo unsafe", afterTarget: "echo unsafe",
		},
		{
			name: "run key includes script value", same: false,
			before:       wrap("      - id: check\n        run: |\n          echo before\n"),
			after:        wrap("      - id: check\n        run: |\n          echo after\n"),
			beforeTarget: "run:", afterTarget: "run:",
		},
		{
			name: "quoted dotted keys are distinct", same: false,
			before:       wrap("      - id: check\n        env: {'a.b': unsafe, a: unsafe}\n        run: echo ok\n"),
			after:        wrap("      - id: check\n        env: {'a.b': unsafe, a: unsafe}\n        run: echo ok\n"),
			beforeTarget: "'a.b'", afterTarget: "a: unsafe",
		},
		{
			name: "alias values come from the Actions AST", same: false,
			before:       "env: &shared\n  TOKEN: before\n" + wrap("      - id: check\n        env: *shared\n        run: echo ok\n"),
			after:        "env: &shared\n  TOKEN: after\n" + wrap("      - id: check\n        env: *shared\n        run: echo ok\n"),
			beforeTarget: "*shared", afterTarget: "*shared",
		},
		{
			name: "alias renaming preserves semantic value", same: true,
			before:       "env: &before\n  TOKEN: unsafe\n" + wrap("      - id: check\n        env: *before\n        run: echo ok\n"),
			after:        "env: &after\n  TOKEN: unsafe\n" + wrap("      - id: check\n        env: *after\n        run: echo ok\n"),
			beforeTarget: "*before", afterTarget: "*after",
		},
		{
			name: "uppercase YAML extension", extension: ".YAML", same: true,
			before:       wrap("      - id: check\n        run: echo unsafe\n"),
			after:        "# header\n" + wrap("      - run: echo safe\n      - id: check\n        run: echo unsafe\n"),
			beforeTarget: "echo unsafe", afterTarget: "echo unsafe",
		},
		{
			name: "AW post steps", extension: ".md", same: true,
			before:       "---\non: workflow_dispatch\nsafe-outputs:\n  post-steps:\n    - id: check\n      run: echo unsafe\n---\nPrompt\n",
			after:        "---\n# header\non: workflow_dispatch\nsafe-outputs:\n  post-steps:\n    - run: echo safe\n    - id: check\n      run: 'echo unsafe'\n---\nPrompt\n",
			beforeTarget: "echo unsafe", afterTarget: "'echo unsafe'",
		},
		{
			name: "AW dotted job ID", extension: ".md", same: true,
			before:       "---\nsafe-jobs:\n  'a.b':\n    steps:\n      - id: check\n        run: echo unsafe\n---\nPrompt\n",
			after:        "---\nsafe-jobs:\n  'a.b':\n    steps:\n      - run: echo safe\n      - id: check\n        run: echo unsafe\n---\nPrompt\n",
			beforeTarget: "echo unsafe", afterTarget: "echo unsafe",
		},
		{
			name: "AW changed run value", extension: ".md", same: false,
			before:       "---\nsteps:\n  - id: check\n    run: echo before\n---\nPrompt\n",
			after:        "---\nsteps:\n  - id: check\n    run: echo after\n---\nPrompt\n",
			beforeTarget: "run:", afterTarget: "run:",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			extension := test.extension
			if extension == "" {
				extension = ".yml"
			}
			path := filepath.Join(t.TempDir(), "workflow"+extension)
			identity := func(content, target string) string {
				require.NoError(t, os.WriteFile(path, []byte(content), 0600))
				offset := strings.Index(content, target)
				require.NotEqual(t, -1, offset)
				lineStart := strings.LastIndex(content[:offset], "\n") + 1
				finding := scanfindings.Finding{
					File: path, Line: strings.Count(content[:offset], "\n") + 1,
					Column: offset - lineStart + 1, RuleID: "test",
					Severity: scanfindings.SeverityMedium, Message: "unsafe",
				}
				source := make(scannerSources).load(normalizeScannerFile(path))
				require.NotNil(t, source)
				require.NotEmpty(t, source.locations, "must use a parsed AST, not coordinate fallback")
				location := source.locate(finding.Line, finding.Column)
				require.NotNil(t, location, "finding must have an exact source anchor")
				if strings.Contains(target, "echo") || target == "run:" {
					require.True(t, location.run, "script findings must anchor to run, not a neighboring field")
				}
				return scannerFindingIdentity(finding)
			}
			before := identity(test.before, test.beforeTarget)
			after := identity(test.after, test.afterTarget)
			assert.Equal(t, test.same, before == after)
		})
	}
}

func TestScannerSourcesAreScopedToOneResult(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "workflow.yml")
	before := "jobs:\n  agent:\n    runs-on: ubuntu-latest\n    steps:\n      - id: check\n        run: echo before\n"
	require.NoError(t, os.WriteFile(path, []byte(before), 0600))
	finding := scanfindings.Finding{File: path, Line: 6, Column: 9, RuleID: "test", Severity: scanfindings.SeverityHigh}
	sources := make(scannerSources)
	identity := sources.findingIdentity(finding)
	baseline := withScannerFindings(errors.New("findings"), []scanfindings.Finding{finding})
	require.NoError(t, os.WriteFile(path, []byte(strings.ReplaceAll(before, "echo before", "echo after")), 0600))
	assert.Equal(t, identity, sources.findingIdentity(finding), "a scanner result uses one immutable source snapshot")
	assert.NotEqual(t, identity, scannerFindingIdentity(finding), "a new result reparses the file")
	diff, err := (&dryRunScannerBaseline{results: map[string]error{"test": baseline}}).filter(
		"test", withScannerFindings(errors.New("findings"), []scanfindings.Finding{finding}),
	)
	require.Error(t, err)
	assert.Equal(t, 1, diff.Added)
	assert.Zero(t, diff.Existing)
}

func TestScannerSourceUnmappedLocationsRemainConservative(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "workflow.yml")
	content := "jobs:\n  agent:\n    runs-on: ubuntu-latest\n    steps:\n      - run: |\n          echo unsafe\n        env:\n          TOKEN: unsafe\n# outside\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))
	source := make(scannerSources).load(path)
	require.NotNil(t, source)
	require.NotEmpty(t, source.locations)
	assert.Nil(t, source.locate(9, 1), "an outside comment must not inherit a preceding scalar")
	assert.Nil(t, source.locate(99, 1), "out-of-bounds findings must not inherit a preceding scalar")
	location := source.locate(8, 11)
	require.NotNil(t, location)
	assert.False(t, location.run, "a following env field must not be part of a dash-prefixed run scalar")
	assert.Equal(t, "$.jobs.agent.steps[0].env.TOKEN", location.path)
	for _, line := range []int{0, 9, 99} {
		_, ok := actionlintShellcheckIdentity(actionlintError{
			Filepath: path, Line: line, Column: 1, Message: "SC2086:warning:1:1: quote this",
		})
		assert.False(t, ok)
	}
}

func TestActionlintShellcheckIdentityKeepsDiagnosticText(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "workflow.yml")
	require.NoError(t, os.WriteFile(path, []byte("jobs:\n  agent:\n    runs-on: ubuntu-latest\n    steps:\n      - id: check\n        run: echo unsafe\n"), 0600))
	issue := actionlintError{Filepath: path, Line: 6, Column: 9, Message: "SC2086:warning:1:1: first message"}
	before, ok := actionlintShellcheckIdentity(issue)
	require.True(t, ok)
	issue.Message = "SC2086:warning:1:1: second message"
	after, ok := actionlintShellcheckIdentity(issue)
	require.True(t, ok)
	assert.NotEqual(t, before, after)
	for _, message := range []string{"SC2086:warning:0:1: invalid", "SC2086:warning:1:0: invalid", "SC2086:warning:-1:1: invalid"} {
		_, _, _, _, ok := parseActionlintShellcheckMessage(message)
		assert.False(t, ok)
	}
}

func TestDryRunZizmorUsesSharedResultFilter(t *testing.T) {
	stdout := `[
{"ident":"secrets-outside-env","desc":"secret outside environment","determinations":{"severity":"Medium"},"locations":[{"symbolic":{"key":{"Local":{"verbatim_path":"test.lock.yml"}}},"concrete":{"location":{"start_point":{"row":1}}}}]},
{"ident":"self-repository","desc":"self reference","determinations":{"severity":"Low"},"locations":[{"symbolic":{"key":{"Local":{"verbatim_path":"test.lock.yml"}}},"concrete":{"location":{"start_point":{"row":2}}}}]},
{"ident":"template-injection","desc":"unsafe input","determinations":{"severity":"High"},"locations":[{"symbolic":{"key":{"Local":{"verbatim_path":"test.lock.yml"}}},"concrete":{"location":{"start_point":{"row":3}}}}]}
]`
	var findings []scanfindings.Finding
	var filteredWarnings int
	var count, high int
	var err error
	output := captureStderrForGuardPolicyReportTest(func() {
		count, high, err = parseAndDisplayZizmorOutputWithOptions(stdout, "", false, zizmorResultOptions{
			dryRun: true, findings: &findings, filteredWarnings: &filteredWarnings,
		})
	})
	require.NoError(t, err)
	assert.Equal(t, 2, count)
	assert.Equal(t, 1, high)
	require.Len(t, findings, 2)
	assert.Equal(t, 1, filteredWarnings)
	assert.NotContains(t, output, "secrets-outside-env")
	assert.Contains(t, output, "test.lock.yml:3:1: info: [Low] self-repository")
	assert.Contains(t, output, "test.lock.yml:4:1: error: [High] template-injection")
	captureStderrForGuardPolicyReportTest(func() {
		count, high, err = parseAndDisplayZizmorOutputWithOptions(stdout, "", false, zizmorResultOptions{})
	})
	require.NoError(t, err)
	assert.Equal(t, 3, count, "ordinary compilation must not ignore secrets-outside-env")
	assert.Equal(t, 1, high)
	_, _, err = parseAndDisplayZizmorOutputWithOptions("invalid JSON", "", false, zizmorResultOptions{dryRun: true})
	require.Error(t, err)
}

func TestZizmorEmptyDiagnosticsOnlyAllowsFilteredFindings(t *testing.T) {
	require.Error(t, zizmorEmptyDiagnosticsError(10, 0))
	require.Error(t, zizmorEmptyDiagnosticsError(14, 0))
	assert.NoError(t, zizmorEmptyDiagnosticsError(10, 1))
	assert.NoError(t, zizmorEmptyDiagnosticsError(0, 0))

	for _, stdout := range []string{
		"null",
		`[{"ident":"template-injection","locations":[]}]`,
		`[{"ident":"secrets-outside-env","locations":[]}]`,
	} {
		var filteredWarnings int
		_, _, err := parseAndDisplayZizmorOutputWithOptions(stdout, "", false, zizmorResultOptions{
			dryRun: true, filteredWarnings: &filteredWarnings,
		})
		if stdout == "null" {
			require.NoError(t, err)
			assert.Error(t, zizmorEmptyDiagnosticsError(10, filteredWarnings))
		} else {
			require.Error(t, err, "unusable findings must fail even alongside usable diagnostics")
		}
	}
}

func TestDevelopmentCompileScannerBaselineBeforeEmission(t *testing.T) {
	source := filepath.Join(t.TempDir(), "differential.md")
	lock := strings.TrimSuffix(source, ".md") + ".lock.yml"
	require.NoError(t, os.WriteFile(source, []byte(`---
on:
  schedule: daily
  workflow_dispatch:
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
			MarkdownFiles: []string{source}, DryRun: true, Strict: true, Zizmor: true,
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

func TestCollectDryRunBaselineInputsSetsWorkflowIdentifierPerFile(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	require.NoError(t, initTestGitRepo(root))
	gitConfig := filepath.Join(root, ".git", "config")
	config, err := os.ReadFile(gitConfig)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(gitConfig, append(config, []byte("\n[remote \"origin\"]\n\turl = https://github.com/test-owner/test-repo.git\n")...), 0600))
	dir := filepath.Join(root, ".github", "workflows")
	require.NoError(t, os.MkdirAll(dir, 0755))
	var sources, locks []string
	for _, name := range []string{"first", "second"} {
		source := filepath.Join(dir, name+".md")
		lock := filepath.Join(dir, name+".lock.yml")
		content := `---
on:
  schedule: daily
mcp-scripts:
  inspect:
    description: Inspect repository state
    run: echo "baseline"
---
Check the workflow.
`
		require.NoError(t, os.WriteFile(source, []byte(content), 0600))
		require.NoError(t, os.WriteFile(lock, []byte("jobs: {}\n"), 0600))
		sources = append(sources, source)
		locks = append(locks, lock)
	}

	compiler := workflow.NewCompiler()
	compiler.SetRepositorySlug("original/repository")
	gotLocks, resources, err := collectDryRunBaselineInputs(sources,
		&CompileConfig{DryRun: true, Strict: true, Shellcheck: true}, compiler)
	require.NoError(t, err)
	assert.Equal(t, locks, gotLocks)
	assert.Equal(t, "original/repository", compiler.GetRepositorySlug(), "baseline collection must restore the compiler's repository context")
	require.Len(t, resources, 2)
	for i, resource := range resources {
		assert.Equal(t, strings.TrimSuffix(filepath.Base(sources[i]), ".md"), resource.Source)
		assert.Contains(t, resource.Script, `echo "baseline"`)
	}

	expectedCompiler := workflow.NewCompiler(
		workflow.WithWorkflowIdentifier(".github/workflows/second.md"))
	expectedCompiler.SetRepositorySlug("original/repository")
	expected, err := expectedCompiler.ParseWorkflowFile(sources[1])
	require.NoError(t, err)
	actual, err := compiler.ParseWorkflowFile(sources[1])
	require.NoError(t, err)
	assert.Equal(t, expected.On, actual.On, "the batch must not retain the first workflow's schedule seed")
}

func TestSetWorkflowFileContextUsesRemoteAndRespectsLockedSeed(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	require.NoError(t, initTestGitRepo(root))
	gitConfig := filepath.Join(root, ".git", "config")
	config, err := os.ReadFile(gitConfig)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(gitConfig, append(config, []byte("\n[remote \"origin\"]\n\turl = https://github.com/test-owner/test-repo.git\n")...), 0600))
	file := filepath.Join(root, ".github", "workflows", "daily.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(file), 0755))
	require.NoError(t, os.WriteFile(file, []byte("---\non:\n  schedule: daily\n---\nCheck the workflow.\n"), 0600))

	compiler := workflow.NewCompiler()
	compiler.SetRepositorySlug("original/repository")
	setWorkflowFileContext(compiler, file)
	assert.Equal(t, "test-owner/test-repo", compiler.GetRepositorySlug())
	expectedCompiler := workflow.NewCompiler(workflow.WithWorkflowIdentifier(".github/workflows/daily.md"))
	expectedCompiler.SetRepositorySlug("test-owner/test-repo")
	expectedData, err := expectedCompiler.ParseWorkflowFile(file)
	require.NoError(t, err)
	actualData, err := compiler.ParseWorkflowFile(file)
	require.NoError(t, err)
	assert.Equal(t, expectedData.On, actualData.On, "the workflow identifier and repository slug must come from this file")

	lockedCompiler := workflow.NewCompiler()
	lockedCompiler.SetRepositorySlug("explicit/seed")
	lockedCompiler.LockRepositorySlug()
	setWorkflowFileContext(lockedCompiler, file)
	assert.Equal(t, "explicit/seed", lockedCompiler.GetRepositorySlug())
	expectedLockedCompiler := workflow.NewCompiler(workflow.WithWorkflowIdentifier(".github/workflows/daily.md"))
	expectedLockedCompiler.SetRepositorySlug("explicit/seed")
	expectedLockedData, err := expectedLockedCompiler.ParseWorkflowFile(file)
	require.NoError(t, err)
	actualLockedData, err := lockedCompiler.ParseWorkflowFile(file)
	require.NoError(t, err)
	assert.Equal(t, expectedLockedData.On, actualLockedData.On, "an explicit seed must not be overridden by the file remote")
}

func TestPrepareDryRunScannerBaselineRejectsMalformedShellcheckLock(t *testing.T) {
	source := filepath.Join(t.TempDir(), "malformed.md")
	lock := strings.TrimSuffix(source, ".md") + ".lock.yml"
	require.NoError(t, os.WriteFile(source, []byte("---\non: workflow_dispatch\n---\n"), 0600))
	require.NoError(t, os.WriteFile(lock, []byte("jobs: [\n"), 0600))

	err := prepareDryRunScannerBaseline(context.Background(), &CompileConfig{
		DryRun: true, Shellcheck: true,
	}, []string{source}, "", workflow.NewCompiler())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot extract ShellCheck baseline")
}

func TestPrepareDryRunScannerBaselineIncludesShellcheckResources(t *testing.T) {
	source := filepath.Join(t.TempDir(), "resource.md")
	content := `---
name: Resource test
on: workflow_dispatch
mcp-scripts:
  inspect:
    description: Inspect repository state
    run: |
      echo "$value"
---
Check the workflow.
`
	require.NoError(t, os.WriteFile(source, []byte(content), 0600))
	originalShellcheck := runBatchShellcheckOnLockFilesAndResources
	t.Cleanup(func() { runBatchShellcheckOnLockFilesAndResources = originalShellcheck })

	var baselineResources []workflow.ShellScriptResource
	runBatchShellcheckOnLockFilesAndResources = func(_ context.Context, lockFiles []string, resources []workflow.ShellScriptResource, _, _ bool) error {
		assert.Empty(t, lockFiles)
		baselineResources = resources
		require.Len(t, resources, 1)
		step := runStepInfo{Source: resources[0].Source, Name: resources[0].Name, Script: resources[0].Script}
		return withShellcheckFindings(errors.New("existing resource finding"), step,
			"script:1:6: warning: quote this [SC2086]\n")
	}

	config := &CompileConfig{DryRun: true, Shellcheck: true}
	require.NoError(t, prepareDryRunScannerBaseline(context.Background(), config, []string{source}, "", workflow.NewCompiler()))
	require.Len(t, baselineResources, 1)

	data, err := workflow.NewCompiler().ParseWorkflowFile(source)
	require.NoError(t, err)
	require.Len(t, data.ShellScriptResources(), 1)
	resource := data.ShellScriptResources()[0]
	current := withShellcheckFindings(errors.New("current resource finding"),
		runStepInfo{Source: resource.Source, Name: resource.Name, Script: resource.Script},
		"script:1:6: warning: quote this [SC2086]\n")
	diff, err := config.dryRunBaseline.filter("shellcheck", current)
	require.NoError(t, err)
	assert.Equal(t, 1, diff.Existing)
}

func TestPrepareDryRunScannerBaselineKeepsDirectoryScopeWithoutTargetLock(t *testing.T) {
	root := t.TempDir()
	newWorkflowDir := filepath.Join(root, "new")
	existingWorkflowDir := filepath.Join(root, "existing")
	require.NoError(t, os.MkdirAll(newWorkflowDir, 0755))
	require.NoError(t, os.MkdirAll(existingWorkflowDir, 0755))
	newSource := filepath.Join(newWorkflowDir, "new.md")
	existingSource := filepath.Join(existingWorkflowDir, "existing.md")
	require.NoError(t, os.WriteFile(newSource, []byte("---\non: workflow_dispatch\n---\n"), 0600))
	require.NoError(t, os.WriteFile(existingSource, []byte("---\non: workflow_dispatch\n---\n"), 0600))
	require.NoError(t, os.WriteFile(strings.TrimSuffix(existingSource, ".md")+".lock.yml", []byte("jobs: {}\n"), 0600))

	originalPoutine := runBatchPoutineOnDirectory
	t.Cleanup(func() { runBatchPoutineOnDirectory = originalPoutine })
	var scannedDirs []string
	runBatchPoutineOnDirectory = func(dir string, _, _ bool) error {
		scannedDirs = append(scannedDirs, dir)
		return nil
	}
	config := CompileConfig{DryRun: true, Poutine: true}
	require.NoError(t, prepareDryRunScannerBaseline(context.Background(), &config,
		[]string{newSource, existingSource}, "", workflow.NewCompiler()))
	require.Equal(t, newWorkflowDir, config.dryRunWorkflowDir)

	_, err := runBatchExternalTools(context.Background(), config, batchToolsOptions{
		workflowDir: config.dryRunWorkflowDir,
	}, &CompilationStats{}, &[]ValidationResult{})
	require.NoError(t, err)
	require.Equal(t, []string{newWorkflowDir, newWorkflowDir}, scannedDirs)
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

func TestShellcheckOutputParsingFailsClosedOnUnmatchedLines(t *testing.T) {
	t.Parallel()
	info := runStepInfo{Script: "echo unsafe\n"}
	scannerErr := errors.New("shellcheck findings")
	err := withShellcheckFindings(scannerErr, info,
		"script:1:1: note: informational [SC1000]\ntruncated warning")
	require.ErrorIs(t, err, scannerErr)
	var findings *scannerFindingsError
	assert.NotErrorAs(t, err, &findings, "partial diagnostics must remain an operational failure")
}

func TestShellcheckIdentityPrefersStepID(t *testing.T) {
	t.Parallel()
	lock := filepath.Join(t.TempDir(), "steps.lock.yml")
	content := `jobs:
  agent:
    steps:
      - name: Check
        id: first
        run: echo "$value"
      - name: Check
        id: second
        run: echo "$value"
      - id: named-by-id
        run: echo "$other"
`
	require.NoError(t, os.WriteFile(lock, []byte(content), 0600))
	steps, err := extractRunStepsFromLockFile(lock)
	require.NoError(t, err)
	require.Len(t, steps, 3)
	assert.Equal(t, "first", steps[0].ID)
	assert.Equal(t, "second", steps[1].ID)
	assert.Equal(t, "named-by-id", steps[2].ID)

	output := "script:1:6: warning: quote this [SC2086]\n"
	first := withShellcheckFindings(errors.New("findings"), steps[0], output)
	second := withShellcheckFindings(errors.New("findings"), steps[1], output)
	var firstFindings, secondFindings *scannerFindingsError
	require.ErrorAs(t, first, &firstFindings)
	require.ErrorAs(t, second, &secondFindings)
	require.Len(t, firstFindings.diagnostics, 1)
	require.Len(t, secondFindings.diagnostics, 1)
	assert.NotEqual(t, firstFindings.diagnostics[0].Identity, secondFindings.diagnostics[0].Identity)

	before := withShellcheckFindings(errors.New("findings"), steps[2], output)
	steps[2].Script = "# harmless comment\necho \"$other\"\n"
	after := withShellcheckFindings(errors.New("findings"), steps[2],
		"script:2:6: warning: quote this [SC2086]\n")
	var beforeFindings, afterFindings *scannerFindingsError
	require.ErrorAs(t, before, &beforeFindings)
	require.ErrorAs(t, after, &afterFindings)
	assert.Equal(t, beforeFindings.diagnostics[0].Identity, afterFindings.diagnostics[0].Identity)
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
