//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewValidateCommand tests that the validate command is created correctly
func TestNewValidateCommand(t *testing.T) {
	cmd := NewValidateCommand(func(string) error { return nil })

	require.NotNil(t, cmd, "NewValidateCommand should return a non-nil command")
	assert.Equal(t, "validate", cmd.Name(), "Command name should be 'validate'")
	assert.NotEmpty(t, cmd.Short, "Command should have a short description")
	assert.NotEmpty(t, cmd.Long, "Command should have a long description")

	// Verify key flags exist
	require.NotNil(t, cmd.Flags().Lookup("dir"), "validate command should have a --dir flag")
	assert.Equal(t, "d", cmd.Flags().Lookup("dir").Shorthand, "--dir flag should have -d shorthand")
	require.NotNil(t, cmd.Flags().Lookup("json"), "validate command should have a --json flag")
	assert.Equal(t, "j", cmd.Flags().Lookup("json").Shorthand, "--json flag should have -j shorthand")
	require.NotNil(t, cmd.Flags().Lookup("engine"), "validate command should have a --engine flag")
	require.NotNil(t, cmd.Flags().Lookup("strict"), "validate command should have a --strict flag")
	require.NotNil(t, cmd.Flags().Lookup("fail-fast"), "validate command should have a --fail-fast flag")
	require.NotNil(t, cmd.Flags().Lookup("stats"), "validate command should have a --stats flag")
	require.NotNil(t, cmd.Flags().Lookup("no-check-update"), "validate command should have a --no-check-update flag")
}
