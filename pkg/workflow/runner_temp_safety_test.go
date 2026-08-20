package workflow

import (
	"strings"
	"testing"
)

func TestRewriteRunnerTempInGithubScriptRequire(t *testing.T) {
	input := `steps:
  - uses: actions/github-script@v7
    with:
      script: |
        const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
        setupGlobals(core, github, context, exec, io, getOctokit);
        const { main } = require('${{ runner.temp }}/gh-aw/actions/generate_aw_info.cjs');
        await main(core, context);
`

	got := rewriteRunnerTempInExecutableBodies(input)

	if strings.Contains(got, "require('${{ runner.temp }}/gh-aw/actions/") {
		t.Fatalf("expected github-script require paths to avoid runner.temp expression:\n%s", got)
	}
	if !strings.Contains(got, "const path = require('path');") {
		t.Fatalf("expected path module prelude:\n%s", got)
	}
	if !strings.Contains(got, "const actionsDir = path.join(process.env.RUNNER_TEMP, 'gh-aw', 'actions');") {
		t.Fatalf("expected safe actionsDir prelude:\n%s", got)
	}
	if !strings.Contains(got, "const { setupGlobals } = require(path.join(actionsDir, 'setup_globals.cjs'));") {
		t.Fatalf("expected setup_globals require to use path.join:\n%s", got)
	}
	if !strings.Contains(got, "const { main } = require(path.join(actionsDir, 'generate_aw_info.cjs'));") {
		t.Fatalf("expected generated script require to use path.join:\n%s", got)
	}
	if err := validateNoRunnerTempInExecutableBodies(got); err != nil {
		t.Fatalf("rewritten output should pass executable body validation: %v", err)
	}
}

func TestRewriteRunnerTempInShellRunCommand(t *testing.T) {
	input := `steps:
  - run: node ${{ runner.temp }}/gh-aw/actions/generate_usage_activity_summary.cjs --json
`

	got := rewriteRunnerTempInExecutableBodies(input)

	if strings.Contains(got, "node ${{ runner.temp }}/gh-aw/actions/") {
		t.Fatalf("expected shell command to avoid runner.temp expression:\n%s", got)
	}
	if !strings.Contains(got, `run: node "${RUNNER_TEMP}/gh-aw/actions/generate_usage_activity_summary.cjs" --json`) {
		t.Fatalf("expected shell command to use quoted RUNNER_TEMP path:\n%s", got)
	}
	if err := validateNoRunnerTempInExecutableBodies(got); err != nil {
		t.Fatalf("rewritten output should pass executable body validation: %v", err)
	}
}

func TestRewriteRunnerTempOnlyRewritesRunCommands(t *testing.T) {
	input := `steps:
  - uses: github/gh-aw/actions/setup@v1
    with:
      command: node ${{ runner.temp }}/gh-aw/actions/generate_aw_info.cjs
  - run: node ${{ runner.temp }}/gh-aw/actions/generate_aw_info.cjs
`

	got := rewriteRunnerTempInExecutableBodies(input)

	if !strings.Contains(got, "command: node ${{ runner.temp }}/gh-aw/actions/generate_aw_info.cjs") {
		t.Fatalf("expected non-executable YAML field to remain unchanged:\n%s", got)
	}
	if !strings.Contains(got, `run: node "${RUNNER_TEMP}/gh-aw/actions/generate_aw_info.cjs"`) {
		t.Fatalf("expected run command to use quoted RUNNER_TEMP path:\n%s", got)
	}
}

func TestRewriteRunnerTempIsIdempotent(t *testing.T) {
	input := `steps:
  - uses: actions/github-script@v7
    with:
      script: |
        const path = require('path');
        const actionsDir = path.join(process.env.RUNNER_TEMP, 'gh-aw', 'actions');
        const { main } = require(path.join(actionsDir, 'generate_aw_info.cjs'));
`

	once := rewriteRunnerTempInExecutableBodies(input)
	twice := rewriteRunnerTempInExecutableBodies(once)
	if once != twice {
		t.Fatalf("rewrite is not idempotent\nfirst:\n%s\nsecond:\n%s", once, twice)
	}
}

func TestRewriteRunnerTempPreservesCRLF(t *testing.T) {
	input := "steps:\r\n  - run: node ${{ runner.temp }}/gh-aw/actions/generate_aw_info.cjs\r\n"

	got := rewriteRunnerTempInExecutableBodies(input)

	if strings.Contains(got, "${{ runner.temp }}") {
		t.Fatalf("expected CRLF run command to avoid runner.temp expression:\n%s", got)
	}
	if !strings.Contains(got, "\r\n") {
		t.Fatalf("expected CRLF line endings to be preserved:\n%s", got)
	}
}

func TestValidateNoRunnerTempInExecutableBodiesAllowsYamlFields(t *testing.T) {
	input := `steps:
  - uses: github/gh-aw/actions/setup@v1
    with:
      destination: ${{ runner.temp }}/gh-aw/actions
`

	if err := validateNoRunnerTempInExecutableBodies(input); err != nil {
		t.Fatalf("non-executable YAML fields should be allowed: %v", err)
	}
}

func TestValidateNoRunnerTempInExecutableBodiesRejectsScriptRegression(t *testing.T) {
	input := `steps:
  - uses: actions/github-script@v7
    with:
      script: |
        const { main } = require('${{ runner.temp }}/gh-aw/actions/generate_aw_info.cjs');
`

	if err := validateNoRunnerTempInExecutableBodies(input); err == nil {
		t.Fatal("expected unsafe runner.temp expression in script body to be rejected")
	}
}
