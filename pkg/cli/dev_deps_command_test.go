package cli

import (
	"testing"
)

func TestDevDepsCommand(t *testing.T) {
	cmd := NewDevDepsCommand()

	// Test that command is created correctly
	if cmd.Use != "dev-deps" {
		t.Errorf("Expected Use to be 'dev-deps', got '%s'", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Expected Short description to be set")
	}

	if cmd.Long == "" {
		t.Error("Expected Long description to be set")
	}

	// Test that verbose flag exists
	if cmd.Flags().Lookup("verbose") != nil {
		t.Error("dev-deps command should not have its own verbose flag (it's inherited from root)")
	}
}

func TestGetCommandVersion(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		versionFlag string
		expectError bool
	}{
		{
			name:        "node version",
			command:     "node",
			versionFlag: "--version",
			expectError: false,
		},
		{
			name:        "npm version",
			command:     "npm",
			versionFlag: "--version",
			expectError: false,
		},
		{
			name:        "nonexistent command",
			command:     "this-command-does-not-exist-12345",
			versionFlag: "--version",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version, err := getCommandVersion(tt.command, tt.versionFlag)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if version == "" {
					t.Error("Expected version string but got empty")
				}
			}
		})
	}
}
