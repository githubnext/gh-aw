//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewVerifyCommand tests that the verify command is created correctly
func TestNewVerifyCommand(t *testing.T) {
	cmd := NewVerifyCommand()

	require.NotNil(t, cmd, "NewVerifyCommand should return a non-nil command")
	assert.Equal(t, "verify", cmd.Name(), "Command name should be 'verify'")
	assert.NotEmpty(t, cmd.Short, "Command should have a short description")
	assert.NotEmpty(t, cmd.Long, "Command should have a long description")

	// Verify the --dir flag exists
	dirFlag := cmd.Flags().Lookup("dir")
	require.NotNil(t, dirFlag, "verify command should have a --dir flag")
	assert.Equal(t, "d", dirFlag.Shorthand, "--dir flag should have -d shorthand")
}
