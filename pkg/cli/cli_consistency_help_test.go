//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuditCommandDescriptionsAreConsistent(t *testing.T) {
	cmd := NewAuditCommand()

	assert.Contains(t, cmd.Short, "workflow runs", "audit short description should describe multiple run inputs")
	assert.Contains(t, cmd.Long, "Audit one or more workflow runs", "audit long description should describe multiple run inputs")
	assert.Contains(t, cmd.Long, "remaining runs are compared against it", "audit help should document multi-run analysis mode")
}

func TestTrialCommandUsesStandardExamplesHeading(t *testing.T) {
	cmd := NewTrialCommand(func(string) error { return nil })

	assert.NotEmpty(t, cmd.Example, "trial command should use cobra's Example field for examples")
	assert.NotContains(t, cmd.Long, "Single workflow:", "trial long help should avoid custom example section headings")
	assert.NotContains(t, cmd.Long, "Multiple workflows (for comparison):", "trial long help should avoid custom example section headings")
	assert.NotContains(t, cmd.Long, "Workflows from different repositories:", "trial long help should avoid custom example section headings")
	assert.NotContains(t, cmd.Long, "Repository mode examples:", "trial long help should avoid custom example section headings")
	assert.NotContains(t, cmd.Long, "Repeat and cleanup examples:", "trial long help should avoid custom example section headings")
	assert.NotContains(t, cmd.Long, "Auto-merge examples:", "trial long help should avoid custom example section headings")
	assert.NotContains(t, cmd.Long, "Advanced examples:", "trial long help should avoid custom example section headings")
}

func TestUpdateDocsIncludeCoolDownOption(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "should resolve current test file path")

	docsPath := filepath.Join(filepath.Dir(currentFile), "..", "..", "docs", "src", "content", "docs", "setup", "cli.md")
	content, err := os.ReadFile(docsPath)
	require.NoError(t, err, "should read CLI setup docs")

	text := string(content)
	updateIndex := strings.Index(text, "#### `update`")
	require.NotEqual(t, -1, updateIndex, "CLI setup docs should contain the update section")

	updateSection := text[updateIndex:]
	assert.Contains(t, updateSection, "`--cool-down`", "update docs options should include --cool-down")
}

func TestCompileDocsIncludeNoModelsDevLookupOption(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "should resolve current test file path")

	docsPath := filepath.Join(filepath.Dir(currentFile), "..", "..", "docs", "src", "content", "docs", "setup", "cli.md")
	content, err := os.ReadFile(docsPath)
	require.NoError(t, err, "should read CLI setup docs")

	text := string(content)
	compileIndex := strings.Index(text, "#### `compile`")
	require.NotEqual(t, -1, compileIndex, "CLI setup docs should contain the compile section")

	compileSection := text[compileIndex:]
	assert.Contains(t, compileSection, "`--no-models-dev-lookup`", "compile docs options should include --no-models-dev-lookup")
	assert.Contains(t, compileSection, "does not run codemods unless you pass `--fix`", "compile docs should explain --fix opt-in behavior")
}

func TestCLIDocsReflectStatusAuditAndExperimentsCommands(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "should resolve current test file path")

	docsPath := filepath.Join(filepath.Dir(currentFile), "..", "..", "docs", "src", "content", "docs", "setup", "cli.md")
	content, err := os.ReadFile(docsPath)
	require.NoError(t, err, "should read CLI setup docs")

	text := string(content)
	assert.Contains(t, text, "#### `experiments`", "CLI setup docs should include the experiments command")
	assert.Contains(t, text, "#### `doctor`", "CLI setup docs should include the doctor command")
	assert.Contains(t, text, "The `audit` command has two modes", "audit docs should describe the current two-mode behavior")
	assert.NotContains(t, text, "enabled/disabled status, schedules, and labels", "status docs should not promise schedule output in console mode")
	assert.Contains(t, text, "Use `--json` to inspect the raw `on` data, including schedules", "status docs should direct schedule inspection to JSON output")
	assert.Contains(t, text, "runs codemods, action version updates, and workflow compilation by default and uses `--no-fix` to skip all three steps", "upgrade docs should explain the inverse --fix/--no-fix behavior")
	assert.Contains(t, text, "Print the current version and build information for the gh aw CLI extension.", "version docs should match the command help text")
	assert.Contains(t, text, "**Options:** `--repo/-r`, `--dir/-d`, `--require-owner-type`, `--json/-j`", "doctor docs should include the --dir shorthand")
	assert.Contains(t, text, "`--require-owner-type` accepts `any`, `user`, or `org` and defaults to `any`", "doctor docs should document the full owner type set and default")
	assert.Contains(t, text, "`--dir` and `--require-owner-type` require `--repo`", "doctor docs should document the repo requirement for repository-only flags")
}

func TestSubcommandListingsUseHyphenBullets(t *testing.T) {
	tests := []struct {
		name    string
		longDoc string
	}{
		{name: "mcp", longDoc: NewMCPCommand().Long},
		{name: "project", longDoc: NewProjectCommand().Long},
		{name: "secrets", longDoc: NewSecretsCommand().Long},
		{name: "experiments", longDoc: NewExperimentsCommand().Long},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Contains(t, tt.longDoc, "Available subcommands:", "command should document available subcommands")
			assert.NotContains(t, tt.longDoc, "  • ", "subcommand list should use '-' bullet style consistently")
		})
	}
}

func TestHelpTextUsesStandardEgPunctuation(t *testing.T) {
	assert.Contains(t, coolDownFlagUsage, "(e.g., 7d", "--cool-down help should use e.g., punctuation")
	assert.Contains(t, NewEnvCommand().Long, "(e.g., default_max_turns)", "env help should use e.g., punctuation")
	assert.Contains(t, NewDomainsCommand().Long, "(e.g., \"node\", \"python\", \"github\")", "domains help should use e.g., punctuation")
	assert.Contains(t, NewChecksCommand().Long, "(e.g., Vercel,", "checks help should use e.g., punctuation")
	assert.Contains(t, NewViewCommand().Long, "(e.g., issues,", "view help should use e.g., punctuation")
	assert.Contains(t, NewExperimentsAnalyzeSubcommand().Long, "e.g., \"my-workflow\"", "experiments analyze help should use e.g., punctuation")
}
