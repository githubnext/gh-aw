//go:build integration

package main

import (
	"regexp"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/campaign"
	"github.com/githubnext/gh-aw/pkg/cli"
	"github.com/spf13/cobra"
)

// TestArgumentSyntaxConsistency verifies that command argument syntax is consistent with validators
func TestArgumentSyntaxConsistency(t *testing.T) {
	tests := []struct {
		name           string
		command        *cobra.Command
		expectedUse    string
		argsValidator  string // Description of the Args validator
		shouldValidate func(*cobra.Command) error
	}{
		// Commands with required arguments (using angle brackets <>)
		{
			name:           "run command requires workflow",
			command:        runCmd,
			expectedUse:    "run <workflow>...",
			argsValidator:  "MinimumNArgs(1)",
			shouldValidate: func(cmd *cobra.Command) error { return cmd.Args(cmd, []string{"test"}) },
		},
		{
			name:           "audit command requires run-id",
			command:        cli.NewAuditCommand(),
			expectedUse:    "audit <run-id>",
			argsValidator:  "ExactArgs(1)",
			shouldValidate: func(cmd *cobra.Command) error { return cmd.Args(cmd, []string{"123456"}) },
		},
		{
			name:           "trial command requires workflow-spec",
			command:        cli.NewTrialCommand(validateEngine),
			expectedUse:    "trial <workflow-spec>...",
			argsValidator:  "MinimumNArgs(1)",
			shouldValidate: func(cmd *cobra.Command) error { return cmd.Args(cmd, []string{"test"}) },
		},
		{
			name:           "add command requires workflow",
			command:        cli.NewAddCommand(validateEngine),
			expectedUse:    "add <workflow>...",
			argsValidator:  "MinimumNArgs(1)",
			shouldValidate: func(cmd *cobra.Command) error { return cmd.Args(cmd, []string{"test"}) },
		},
		{
			name:           "campaign new requires campaign-id",
			command:        findSubcommand(campaign.NewCommand(), "new"),
			expectedUse:    "new <campaign-id>",
			argsValidator:  "MaximumNArgs(1) with custom validation",
			shouldValidate: nil, // Skip validation as it has custom error handling
		},

		// Commands with optional arguments (using square brackets [])
		{
			name:           "new command has optional workflow",
			command:        newCmd,
			expectedUse:    "new [workflow]",
			argsValidator:  "MaximumNArgs(1)",
			shouldValidate: func(cmd *cobra.Command) error { return cmd.Args(cmd, []string{}) },
		},
		{
			name:           "remove command has optional pattern",
			command:        removeCmd,
			expectedUse:    "remove [pattern]",
			argsValidator:  "no validator (all optional)",
			shouldValidate: func(cmd *cobra.Command) error { return nil },
		},
		{
			name:           "enable command has optional workflow",
			command:        enableCmd,
			expectedUse:    "enable [workflow]...",
			argsValidator:  "no validator (all optional)",
			shouldValidate: func(cmd *cobra.Command) error { return nil },
		},
		{
			name:           "disable command has optional workflow",
			command:        disableCmd,
			expectedUse:    "disable [workflow]...",
			argsValidator:  "no validator (all optional)",
			shouldValidate: func(cmd *cobra.Command) error { return nil },
		},
		{
			name:           "compile command has optional workflow",
			command:        compileCmd,
			expectedUse:    "compile [workflow]...",
			argsValidator:  "no validator (all optional)",
			shouldValidate: func(cmd *cobra.Command) error { return nil },
		},
		{
			name:           "logs command has optional workflow",
			command:        cli.NewLogsCommand(),
			expectedUse:    "logs [workflow]",
			argsValidator:  "no validator (all optional)",
			shouldValidate: func(cmd *cobra.Command) error { return nil },
		},
		{
			name:           "fix command has optional workflow",
			command:        cli.NewFixCommand(),
			expectedUse:    "fix [workflow]...",
			argsValidator:  "no validator (all optional)",
			shouldValidate: func(cmd *cobra.Command) error { return nil },
		},
		{
			name:           "update command has optional workflow",
			command:        cli.NewUpdateCommand(validateEngine),
			expectedUse:    "update [workflow]...",
			argsValidator:  "no validator (all optional)",
			shouldValidate: func(cmd *cobra.Command) error { return nil },
		},
		{
			name:           "status command has optional pattern",
			command:        cli.NewStatusCommand(),
			expectedUse:    "status [pattern]",
			argsValidator:  "no validator (all optional)",
			shouldValidate: func(cmd *cobra.Command) error { return nil },
		},
		{
			name:           "campaign command has optional filter",
			command:        campaign.NewCommand(),
			expectedUse:    "campaign [filter]",
			argsValidator:  "MaximumNArgs(1)",
			shouldValidate: func(cmd *cobra.Command) error { return cmd.Args(cmd, []string{}) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Check Use pattern
			use := tt.command.Use
			if use != tt.expectedUse {
				t.Errorf("Command Use=%q, expected %q", use, tt.expectedUse)
			}

			// Validate the Use pattern format
			if !isValidUseSyntax(use) {
				t.Errorf("Command Use=%q has invalid syntax", use)
			}

			// Skip validation check if not provided
			if tt.shouldValidate == nil {
				return
			}

			// Validate that the Args validator works as expected
			if err := tt.shouldValidate(tt.command); err != nil {
				t.Errorf("Args validator failed for command %q: %v", tt.command.Name(), err)
			}
		})
	}
}

// TestMCPSubcommandArgumentSyntax verifies MCP subcommands have consistent syntax
func TestMCPSubcommandArgumentSyntax(t *testing.T) {
	mcpCmd := cli.NewMCPCommand()

	tests := []struct {
		name        string
		subcommand  string
		expectedUse string
	}{
		{
			name:        "mcp list has optional workflow",
			subcommand:  "list",
			expectedUse: "list [workflow]",
		},
		{
			name:        "mcp inspect has optional workflow",
			subcommand:  "inspect",
			expectedUse: "inspect [workflow]",
		},
		{
			name:        "mcp add has optional workflow and server",
			subcommand:  "add",
			expectedUse: "add [workflow] [server]",
		},
		{
			name:        "mcp list-tools requires server with optional workflow",
			subcommand:  "list-tools",
			expectedUse: "list-tools <server> [workflow]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Find the subcommand
			var foundCmd *cobra.Command
			for _, cmd := range mcpCmd.Commands() {
				if cmd.Name() == tt.subcommand {
					foundCmd = cmd
					break
				}
			}

			if foundCmd == nil {
				t.Fatalf("MCP subcommand %q not found", tt.subcommand)
			}

			use := foundCmd.Use
			if use != tt.expectedUse {
				t.Errorf("MCP subcommand Use=%q, expected %q", use, tt.expectedUse)
			}

			// Validate the Use pattern format
			if !isValidUseSyntax(use) {
				t.Errorf("MCP subcommand Use=%q has invalid syntax", use)
			}
		})
	}
}

// TestPRSubcommandArgumentSyntax verifies PR subcommands have consistent syntax
func TestPRSubcommandArgumentSyntax(t *testing.T) {
	prCmd := cli.NewPRCommand()

	tests := []struct {
		name        string
		subcommand  string
		expectedUse string
	}{
		{
			name:        "pr transfer requires pr-url",
			subcommand:  "transfer",
			expectedUse: "transfer <pr-url>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Find the subcommand
			var foundCmd *cobra.Command
			for _, cmd := range prCmd.Commands() {
				if cmd.Name() == tt.subcommand {
					foundCmd = cmd
					break
				}
			}

			if foundCmd == nil {
				t.Fatalf("PR subcommand %q not found", tt.subcommand)
			}

			use := foundCmd.Use
			if use != tt.expectedUse {
				t.Errorf("PR subcommand Use=%q, expected %q", use, tt.expectedUse)
			}

			// Validate the Use pattern format
			if !isValidUseSyntax(use) {
				t.Errorf("PR subcommand Use=%q has invalid syntax", use)
			}
		})
	}
}

// TestCampaignSubcommandArgumentSyntax verifies campaign subcommands have consistent syntax
func TestCampaignSubcommandArgumentSyntax(t *testing.T) {
	campaignCmd := campaign.NewCommand()

	tests := []struct {
		name        string
		subcommand  string
		expectedUse string
	}{
		{
			name:        "campaign status has optional filter",
			subcommand:  "status",
			expectedUse: "status [filter]",
		},
		{
			name:        "campaign new requires campaign-id",
			subcommand:  "new",
			expectedUse: "new <campaign-id>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Find the subcommand
			var foundCmd *cobra.Command
			for _, cmd := range campaignCmd.Commands() {
				if cmd.Name() == tt.subcommand {
					foundCmd = cmd
					break
				}
			}

			if foundCmd == nil {
				t.Fatalf("Campaign subcommand %q not found", tt.subcommand)
			}

			use := foundCmd.Use
			if use != tt.expectedUse {
				t.Errorf("Campaign subcommand Use=%q, expected %q", use, tt.expectedUse)
			}

			// Validate the Use pattern format
			if !isValidUseSyntax(use) {
				t.Errorf("Campaign subcommand Use=%q has invalid syntax", use)
			}
		})
	}
}

// findSubcommand finds a subcommand by name in a command
func findSubcommand(cmd *cobra.Command, name string) *cobra.Command {
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == name {
			return subcmd
		}
	}
	return nil
}

// isValidUseSyntax validates the Use syntax pattern
func isValidUseSyntax(use string) bool {
	// Pattern: command-name [<required>|[optional]] [...]
	// Required arguments use angle brackets: <arg>
	// Optional arguments use square brackets: [arg]
	// Multiple arguments indicated with ellipsis: ...

	parts := strings.Fields(use)
	if len(parts) == 0 {
		return false
	}

	// First part should be the command name (no brackets or special chars except hyphen)
	commandName := parts[0]
	if !regexp.MustCompile(`^[a-z][a-z0-9-]*$`).MatchString(commandName) {
		return false
	}

	// Check argument patterns
	for i := 1; i < len(parts); i++ {
		arg := parts[i]

		// Check for valid patterns:
		// - <arg>     (required)
		// - <arg>...  (required multiple)
		// - [arg]     (optional)
		// - [arg]...  (optional multiple)

		validPatterns := []string{
			`^<[a-z][a-z0-9-]*>$`,      // <required>
			`^<[a-z][a-z0-9-]*>\.\.\.$`, // <required>...
			`^\[[a-z][a-z0-9-]*\]$`,     // [optional]
			`^\[[a-z][a-z0-9-]*\]\.\.\.$`, // [optional]...
		}

		isValid := false
		for _, pattern := range validPatterns {
			if regexp.MustCompile(pattern).MatchString(arg) {
				isValid = true
				break
			}
		}

		if !isValid {
			return false
		}
	}

	return true
}

// TestArgumentNamingConventions verifies that argument names follow conventions
func TestArgumentNamingConventions(t *testing.T) {
	// Collect all commands
	commands := []*cobra.Command{
		newCmd,
		removeCmd,
		enableCmd,
		disableCmd,
		compileCmd,
		runCmd,
		cli.NewAddCommand(validateEngine),
		cli.NewUpdateCommand(validateEngine),
		cli.NewTrialCommand(validateEngine),
		cli.NewLogsCommand(),
		cli.NewAuditCommand(),
		cli.NewFixCommand(),
		cli.NewStatusCommand(),
		cli.NewMCPCommand(),
		cli.NewPRCommand(),
		campaign.NewCommand(),
	}

	// Also collect subcommands
	for _, cmd := range commands {
		commands = append(commands, cmd.Commands()...)
	}

	// Define naming conventions
	conventions := map[string]string{
		"workflow":    "Workflow-related commands should use 'workflow' for consistency",
		"pattern":     "Filter/search commands should use 'pattern' or 'filter'",
		"run-id":      "Audit command should use 'run-id' for clarity",
		"workflow-spec": "Trial command should use 'workflow-spec' to indicate special format",
		"campaign-id": "Campaign new should use 'campaign-id' for clarity",
		"pr-url":      "PR transfer should use 'pr-url' for clarity",
		"server":      "MCP commands should use 'server' for MCP server names",
	}

	for _, cmd := range commands {
		use := cmd.Use
		parts := strings.Fields(use)

		for i := 1; i < len(parts); i++ {
			arg := parts[i]

			// Extract the argument name (remove brackets and ellipsis)
			argName := arg
			argName = strings.TrimPrefix(argName, "<")
			argName = strings.TrimPrefix(argName, "[")
			argName = strings.TrimSuffix(argName, "...")
			argName = strings.TrimSuffix(argName, ">")
			argName = strings.TrimSuffix(argName, "]")

			// Verify argument name follows conventions
			if reason, exists := conventions[argName]; exists {
				t.Logf("✓ Command %q uses conventional argument name %q: %s", cmd.Name(), argName, reason)
			}

			// Argument names should be lowercase with hyphens
			if !regexp.MustCompile(`^[a-z][a-z0-9-]*$`).MatchString(argName) {
				t.Errorf("Command %q has argument %q with invalid naming (should be lowercase with hyphens)", cmd.Name(), argName)
			}
		}
	}
}
