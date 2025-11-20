package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMCPBridgeCommand(t *testing.T) {
	cmd := NewMCPBridgeCommand()

	require.NotNil(t, cmd)
	assert.Equal(t, "bridge", cmd.Use)
	assert.Contains(t, cmd.Short, "Bridge a stdio MCP server to HTTP transport")

	// Verify required flags
	portFlag := cmd.Flags().Lookup("port")
	require.NotNil(t, portFlag)
	assert.Equal(t, "int", portFlag.Value.Type())

	commandFlag := cmd.Flags().Lookup("command")
	require.NotNil(t, commandFlag)
	assert.Equal(t, "string", commandFlag.Value.Type())

	argsFlag := cmd.Flags().Lookup("args")
	require.NotNil(t, argsFlag)
	assert.Equal(t, "stringSlice", argsFlag.Value.Type())
}

func TestMCPBridgeValidation(t *testing.T) {
	tests := []struct {
		name        string
		port        int
		command     string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "missing command",
			port:        8080,
			command:     "",
			expectError: true,
			errorMsg:    "--command is required",
		},
		{
			name:        "invalid port",
			port:        0,
			command:     "echo",
			expectError: true,
			errorMsg:    "--port must be a positive number",
		},
		{
			name:        "negative port",
			port:        -1,
			command:     "echo",
			expectError: true,
			errorMsg:    "--port must be a positive number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test validation logic
			cmd := NewMCPBridgeCommand()

			// Set flags
			cmd.Flags().Set("port", string(rune(tt.port)))
			cmd.Flags().Set("command", tt.command)

			// Execute command (should fail in validation)
			err := cmd.RunE(cmd, []string{})

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				// We expect an error for valid cases too since we're not actually running a server
				// But it should not be a validation error
				if err != nil {
					assert.NotContains(t, err.Error(), tt.errorMsg)
				}
			}
		})
	}
}

func TestMCPBridgeCommandHelp(t *testing.T) {
	cmd := NewMCPBridgeCommand()

	// Verify help text contains key information
	assert.Contains(t, cmd.Long, "Bridge converts a stdio-based MCP server to HTTP transport")
	assert.Contains(t, cmd.Long, "SSE (Server-Sent Events)")
	assert.Contains(t, cmd.Long, "process isolation")
	assert.Contains(t, cmd.Long, "Examples:")
}
