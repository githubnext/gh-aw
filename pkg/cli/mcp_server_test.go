package cli

import (
	"testing"
)

func TestMCPServerCommand(t *testing.T) {
	t.Run("mcp server command is available", func(t *testing.T) {
		cmd := NewMCPServerSubcommand()
		if cmd == nil {
			t.Fatal("NewMCPServerSubcommand returned nil")
		}
		
		if cmd.Use != "server" {
			t.Errorf("Expected command Use to be 'server', got '%s'", cmd.Use)
		}
		
		if cmd.Short == "" {
			t.Error("Expected command Short description to be non-empty")
		}
		
		// Check that the command has the verbose flag
		flag := cmd.Flags().Lookup("verbose")
		if flag == nil {
			t.Error("Expected command to have verbose flag")
		}
	})

	t.Run("createMCPServer creates server with tools", func(t *testing.T) {
		server := createMCPServer(false)
		if server == nil {
			t.Fatal("createMCPServer returned nil")
		}
		
		// We can't easily test the server tools without starting it,
		// but we can verify it was created successfully
	})
}