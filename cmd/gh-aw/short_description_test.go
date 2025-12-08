package main

import (
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/cli"
	"github.com/spf13/cobra"
)

// TestShortDescriptionConsistency verifies that all command Short descriptions
// follow CLI conventions:
// - No trailing punctuation (periods, exclamation marks, question marks)
// - This is a common convention for CLI tools (e.g., Git, kubectl, gh)
func TestShortDescriptionConsistency(t *testing.T) {
	// Collect all commands to test
	allCommands := []*cobra.Command{
		rootCmd,
		newCmd,
		removeCmd,
		enableCmd,
		disableCmd,
		compileCmd,
		runCmd,
		versionCmd,
		cli.NewStatusCommand(),
		cli.NewInitCommand(),
		cli.NewLogsCommand(),
		cli.NewTrialCommand(validateEngine),
		cli.NewAddCommand(validateEngine),
		cli.NewUpdateCommand(validateEngine),
		cli.NewAuditCommand(),
		cli.NewMCPCommand(),
		cli.NewMCPServerCommand(),
		cli.NewMCPGatewayCommand(),
		cli.NewPRCommand(),
	}

	// Also include MCP subcommands
	mcpCmd := cli.NewMCPCommand()
	allCommands = append(allCommands, mcpCmd.Commands()...)

	// Also include PR subcommands
	prCmd := cli.NewPRCommand()
	allCommands = append(allCommands, prCmd.Commands()...)

	// Check each command's Short description
	for _, cmd := range allCommands {
		t.Run("command "+cmd.Name()+" has no trailing punctuation", func(t *testing.T) {
			short := cmd.Short
			if short == "" {
				t.Skip("Command has no Short description")
			}

			// Check for trailing punctuation
			lastChar := short[len(short)-1:]
			if lastChar == "." || lastChar == "!" || lastChar == "?" {
				t.Errorf("Command '%s' Short description should not end with punctuation. Got: %q", cmd.Name(), short)
			}
		})
	}
}

// TestLongDescriptionHasSentences verifies that Long descriptions use proper
// sentences with punctuation, in contrast to Short descriptions
func TestLongDescriptionHasSentences(t *testing.T) {
	// Sample commands that have Long descriptions
	commandsWithLong := []*cobra.Command{
		rootCmd,
		newCmd,
		removeCmd,
		enableCmd,
		disableCmd,
		compileCmd,
		runCmd,
		cli.NewMCPCommand(),
	}

	for _, cmd := range commandsWithLong {
		t.Run("command "+cmd.Name()+" Long description uses sentences", func(t *testing.T) {
			long := strings.TrimSpace(cmd.Long)
			if long == "" {
				t.Skip("Command has no Long description")
			}

			// Long descriptions should contain at least one sentence ending with a period
			// This is just a sanity check, not a strict requirement
			if !strings.Contains(long, ".") {
				t.Logf("Note: Command '%s' Long description may benefit from proper sentence punctuation", cmd.Name())
			}
		})
	}
}
