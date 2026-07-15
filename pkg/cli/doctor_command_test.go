//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDoctorCommand(t *testing.T) {
	cmd := NewDoctorCommand()

	require.NotNil(t, cmd)
	assert.Equal(t, "doctor", cmd.Use)
	assert.Equal(t, "Run diagnostics to verify CLI authentication and repository setup", cmd.Short)
	assert.False(t, cmd.Hidden)
	assert.NotNil(t, cmd.Flags().Lookup("json"), "should expose --json flag")
	assert.NotNil(t, cmd.Flags().Lookup("repo"), "should expose --repo flag")
	assert.NotNil(t, cmd.Flags().Lookup("dir"), "should expose --dir flag")
	assert.NotNil(t, cmd.Flags().Lookup("require-owner-type"), "should expose --require-owner-type flag")
	assert.False(t, cmd.HasSubCommands())
}

func TestDoctorCommandUsesNoArgs(t *testing.T) {
	cmd := NewDoctorCommand()
	require.NotNil(t, cmd.Args)
	require.NoError(t, cmd.Args(cmd, []string{}))
	assert.Error(t, cmd.Args(cmd, []string{"extra"}))
}

func TestDoctorCommandAdvertisesJSONExample(t *testing.T) {
	cmd := NewDoctorCommand()
	assert.Contains(t, cmd.Example, "gh aw doctor --json")
	assert.Contains(t, cmd.Example, "gh aw doctor --repo github/gh-aw --json")
}

func TestDoctorCommandExampleHasNoTabs(t *testing.T) {
	cmd := NewDoctorCommand()
	assert.NotContains(t, cmd.Example, "\t")
}
