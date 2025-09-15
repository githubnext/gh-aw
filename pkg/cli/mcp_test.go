package cli

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestNewMCPCommand(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "mcp command structure",
			test: func(t *testing.T) {
				cmd := NewMCPCommand()

				if cmd.Use != "mcp" {
					t.Errorf("Expected Use to be 'mcp', got '%s'", cmd.Use)
				}

				if cmd.Short == "" {
					t.Error("Expected Short description to be set")
				}

				if !strings.Contains(cmd.Long, "Model Context Protocol") {
					t.Error("Expected Long description to mention Model Context Protocol")
				}
			},
		},
		{
			name: "mcp command has inspect subcommand",
			test: func(t *testing.T) {
				cmd := NewMCPCommand()

				var inspectCmd *cobra.Command
				for _, subCmd := range cmd.Commands() {
					if subCmd.Use == "inspect [workflow-file]" {
						inspectCmd = subCmd
						break
					}
				}

				if inspectCmd == nil {
					t.Error("Expected 'inspect' subcommand to be present")
				}

				if inspectCmd != nil && inspectCmd.Short == "" {
					t.Error("Expected inspect subcommand to have Short description")
				}
			},
		},
		{
			name: "mcp inspect has required flags",
			test: func(t *testing.T) {
				cmd := NewMCPInspectSubCommand()

				expectedFlags := []string{"server", "tool", "verbose", "inspector", "generate-config", "launch-servers"}

				for _, flagName := range expectedFlags {
					flag := cmd.Flags().Lookup(flagName)
					if flag == nil {
						t.Errorf("Expected flag '%s' to be present", flagName)
					}
				}
			},
		},
		{
			name: "legacy mcp-inspect command",
			test: func(t *testing.T) {
				cmd := NewMCPInspectCommand()

				if cmd.Use != "mcp-inspect [workflow-file]" {
					t.Errorf("Expected legacy command Use to be 'mcp-inspect [workflow-file]', got '%s'", cmd.Use)
				}

				if !strings.Contains(cmd.Long, "deprecated") {
					t.Error("Expected legacy command Long description to mention it's deprecated")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func TestMCPInspectSubCommand(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "inspect subcommand structure",
			test: func(t *testing.T) {
				cmd := NewMCPInspectSubCommand()

				if cmd.Use != "inspect [workflow-file]" {
					t.Errorf("Expected Use to be 'inspect [workflow-file]', got '%s'", cmd.Use)
				}

				if cmd.Short == "" {
					t.Error("Expected Short description to be set")
				}

				expectedFeatures := []string{"generate MCP configurations", "Claude agentic engine", "github, playwright, and safe-outputs"}
				for _, feature := range expectedFeatures {
					if !strings.Contains(cmd.Long, feature) {
						t.Errorf("Expected Long description to mention '%s'", feature)
					}
				}
			},
		},
		{
			name: "inspect subcommand examples",
			test: func(t *testing.T) {
				cmd := NewMCPInspectSubCommand()

				expectedExamples := []string{"--generate-config", "--launch-servers", "--inspector"}
				for _, example := range expectedExamples {
					if !strings.Contains(cmd.Long, example) {
						t.Errorf("Expected Long description to include example with '%s'", example)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}
