//go:build !integration

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunCommandHelpTextConsistency(t *testing.T) {
	assert.Contains(t, runCmd.Long, "this command enters interactive mode and shows", "run command interactive mode text should be explicit")

	runApprove := runCmd.Flags().Lookup("approve")
	compileApprove := compileCmd.Flags().Lookup("approve")
	require.NotNil(t, runApprove, "run command should define --approve")
	require.NotNil(t, compileApprove, "compile command should define --approve")
	assert.Contains(t, compileApprove.Usage, "safe update changes", "compile --approve should describe compiler safe update approval")
	assert.Equal(t, "Approve safe update manifest changes when --push triggers an automatic recompile step", runApprove.Usage, "run --approve should explain the --push-triggered recompile behavior")
}

func TestCompileScheduleSeedHelpUsesConsistentQuotes(t *testing.T) {
	scheduleSeedFlag := compileCmd.Flags().Lookup("schedule-seed")
	require.NotNil(t, scheduleSeedFlag, "compile command should define --schedule-seed")
	assert.Contains(t, scheduleSeedFlag.Usage, "\"github/gh-aw\"", "--schedule-seed example should use double quotes")
	assert.Contains(t, scheduleSeedFlag.Usage, "\"origin\"", "--schedule-seed remote example should use double quotes")
	assert.Contains(t, scheduleSeedFlag.Usage, "(e.g.,", "--schedule-seed example should use standard e.g., punctuation")
}

func TestCompileStagedFlagHelpText(t *testing.T) {
	stagedFlag := compileCmd.Flags().Lookup("staged")
	require.NotNil(t, stagedFlag, "compile command should define --staged")
	assert.Equal(t, "Force all safe-outputs into staged mode", stagedFlag.Usage)
}

func TestCompileShowAllFlagHelpText(t *testing.T) {
	showAllFlag := compileCmd.Flags().Lookup("show-all")
	require.NotNil(t, showAllFlag, "compile command should define --show-all")
	assert.Equal(t, "Display all compilation errors instead of only the highest-priority subset (default: top 5)", showAllFlag.Usage)
}

func TestCompileStrictFlagHelpText(t *testing.T) {
	strictFlag := compileCmd.Flags().Lookup("strict")
	require.NotNil(t, strictFlag, "compile command should define --strict")
	assert.Contains(t, strictFlag.Usage, "disallows write permissions and deprecated fields")
}

func TestCompileGhAwRefMutuallyExclusiveFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "gh-aw-ref with action-tag",
			args: []string{"compile", "--gh-aw-ref", "main", "--action-tag", "v1.2.3"},
		},
		{
			name: "gh-aw-ref with action-mode",
			args: []string{"compile", "--gh-aw-ref", "main", "--action-mode", "dev"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd.SetArgs(tt.args)
			t.Cleanup(func() {
				rootCmd.SetArgs([]string{})
			})

			err := rootCmd.Execute()
			require.Error(t, err)
			assert.Contains(t, err.Error(), "if any flags in the group", "expected mutually exclusive flag-group error")
		})
	}
}
