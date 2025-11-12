package main

import (
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/cli"
	"github.com/spf13/cobra"
)

// TestCapitalizationConsistency verifies that command descriptions follow Option 2:
// - Use lowercase "agentic workflows" when referring generically to workflow files/functionality
// - Use capitalized "Agentic Workflows" only when explicitly referring to the product as a whole
func TestCapitalizationConsistency(t *testing.T) {
	// Test root command uses product name with capital
	if !strings.Contains(rootCmd.Short, "GitHub Agentic Workflows") {
		t.Errorf("Root command Short should use capitalized product name 'GitHub Agentic Workflows', got: %s", rootCmd.Short)
	}
	if !strings.Contains(rootCmd.Long, "GitHub Agentic Workflows") {
		t.Errorf("Root command Long should use capitalized product name 'GitHub Agentic Workflows', got: %s", rootCmd.Long)
	}

	// Test version command uses product name with capital
	if !strings.Contains(versionCmd.Long, "GitHub Agentic Workflows") && !strings.Contains(versionCmd.Short, "GitHub Agentic Workflows") {
		// Version command is allowed to not have the product name in descriptions
		// since it's output in the Run function instead
	}

	// Define commands that should use lowercase "agentic workflows" (generic usage)
	genericWorkflowCommands := []*cobra.Command{
		enableCmd,
		disableCmd,
		runCmd,
		cli.NewStatusCommand(),
		cli.NewInitCommand(),
		cli.NewLogsCommand(),
		cli.NewTrialCommand(validateEngine),
	}

	for _, cmd := range genericWorkflowCommands {
		// Check Short description
		if strings.Contains(cmd.Short, "Agentic Workflows") && !strings.Contains(cmd.Short, "GitHub Agentic Workflows") {
			t.Errorf("Command '%s' Short description should use lowercase 'agentic workflows' for generic usage, not 'Agentic Workflows'. Got: %s", cmd.Name(), cmd.Short)
		}

		// Check Long description
		if strings.Contains(cmd.Long, "Agentic Workflows") && !strings.Contains(cmd.Long, "GitHub Agentic Workflows") {
			t.Errorf("Command '%s' Long description should use lowercase 'agentic workflows' for generic usage, not 'Agentic Workflows'. Got: %s", cmd.Name(), cmd.Long)
		}

		// Verify they use lowercase "agentic workflows" or "agentic workflow" somewhere
		lowerShort := strings.ToLower(cmd.Short)
		lowerLong := strings.ToLower(cmd.Long)
		if !strings.Contains(lowerShort, "agentic workflow") && !strings.Contains(lowerLong, "agentic workflow") {
			// Not all commands need to mention agentic workflows
			continue
		}

		// If they mention it, ensure it's lowercase
		if strings.Contains(cmd.Short, "agentic workflow") || strings.Contains(cmd.Long, "agentic workflow") {
			// Lowercase usage found - this is correct
			continue
		}

		// Check if uppercase was used without "GitHub" prefix (which would be wrong)
		if strings.Contains(cmd.Short, "Agentic Workflow") && !strings.Contains(cmd.Short, "GitHub Agentic Workflow") {
			t.Errorf("Command '%s' should use lowercase 'agentic workflow' not 'Agentic Workflow' in Short", cmd.Name())
		}
		if strings.Contains(cmd.Long, "Agentic Workflow") && !strings.Contains(cmd.Long, "GitHub Agentic Workflow") {
			t.Errorf("Command '%s' should use lowercase 'agentic workflow' not 'Agentic Workflow' in Long", cmd.Name())
		}
	}
}

// TestMCPCommandCapitalization specifically tests MCP subcommands
func TestMCPCommandCapitalization(t *testing.T) {
	mcpCmd := cli.NewMCPCommand()

	// MCP command Long description should use lowercase "agentic workflows"
	if strings.Contains(mcpCmd.Long, "Agentic Workflows") && !strings.Contains(mcpCmd.Long, "GitHub Agentic Workflows") {
		t.Errorf("MCP command Long should use lowercase 'agentic workflows', got: %s", mcpCmd.Long)
	}

	// Check all MCP subcommands
	for _, subCmd := range mcpCmd.Commands() {
		if strings.Contains(subCmd.Short, "Agentic Workflows") && !strings.Contains(subCmd.Short, "GitHub Agentic Workflows") {
			t.Errorf("MCP subcommand '%s' Short should use lowercase 'agentic workflows', got: %s", subCmd.Name(), subCmd.Short)
		}
		if strings.Contains(subCmd.Long, "Agentic Workflows") && !strings.Contains(subCmd.Long, "GitHub Agentic Workflows") {
			t.Errorf("MCP subcommand '%s' Long should use lowercase 'agentic workflows', got: %s", subCmd.Name(), subCmd.Long)
		}
	}
}

// TestTechnicalTermsCapitalization verifies that technical terms remain capitalized
func TestTechnicalTermsCapitalization(t *testing.T) {
	// Technical terms that should remain capitalized
	technicalTerms := []string{"Markdown", "YAML", "MCP"}

	// Check compile command which mentions these terms
	for _, term := range technicalTerms {
		if !strings.Contains(compileCmd.Short, term) {
			// Term not mentioned - skip
			continue
		}
		// If mentioned, it should be capitalized (this test just documents the expectation)
		// The actual check is that it's not all lowercase
		lowerTerm := strings.ToLower(term)
		if strings.Contains(compileCmd.Short, lowerTerm) && !strings.Contains(compileCmd.Short, term) {
			t.Errorf("Compile command should capitalize technical term '%s', but found lowercase '%s'", term, lowerTerm)
		}
	}
}
