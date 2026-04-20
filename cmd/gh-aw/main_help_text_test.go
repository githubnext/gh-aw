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
	assert.Equal(t, compileApprove.Usage, runApprove.Usage, "run and compile should share the same --approve description")
}

func TestCompileScheduleSeedHelpUsesConsistentQuotes(t *testing.T) {
	scheduleSeedFlag := compileCmd.Flags().Lookup("schedule-seed")
	require.NotNil(t, scheduleSeedFlag, "compile command should define --schedule-seed")
	assert.Contains(t, scheduleSeedFlag.Usage, "\"github/gh-aw\"", "--schedule-seed example should use double quotes")
	assert.Contains(t, scheduleSeedFlag.Usage, "\"origin\"", "--schedule-seed remote example should use double quotes")
}
