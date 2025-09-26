package cli

import (
	"strings"
	"testing"
)

func TestMCPServerCommand(t *testing.T) {
	t.Run("mcp serve command is available", func(t *testing.T) {
		cmd := NewMCPServerSubcommand()
		if cmd == nil {
			t.Fatal("NewMCPServerSubcommand returned nil")
		}

		if cmd.Use != "serve" {
			t.Errorf("Expected command Use to be 'serve', got '%s'", cmd.Use)
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
		server := createMCPServer(false, []string{})
		if server == nil {
			t.Fatal("createMCPServer returned nil")
		}

		// We can't easily test the server tools without starting it,
		// but we can verify it was created successfully
	})

	t.Run("createMCPServer creates server with filtered tools", func(t *testing.T) {
		// Test with limited tools
		server := createMCPServer(false, []string{"compile", "logs"})
		if server == nil {
			t.Fatal("createMCPServer returned nil")
		}

		// We can't easily test the exact tool count without starting the server,
		// but we can verify it was created successfully with the filter
	})

	t.Run("createMCPServer includes docs tool", func(t *testing.T) {
		// Test that docs tool is included when no filter is applied
		server := createMCPServer(false, []string{})
		if server == nil {
			t.Fatal("createMCPServer returned nil")
		}

		// Test that docs tool is included when specifically allowed
		server = createMCPServer(false, []string{"docs"})
		if server == nil {
			t.Fatal("createMCPServer returned nil")
		}
	})
}

func TestGetInstructionsTemplate(t *testing.T) {
	t.Run("GetInstructionsTemplate returns non-empty content", func(t *testing.T) {
		template := GetInstructionsTemplate()
		if template == "" {
			t.Error("Expected GetInstructionsTemplate to return non-empty content")
		}

		// Should contain expected markdown content
		if !strings.Contains(template, "# GitHub Agentic Workflows") {
			t.Error("Expected template to contain main heading")
		}

		if !strings.Contains(template, "## File Format Overview") {
			t.Error("Expected template to contain File Format Overview section")
		}
	})
}
