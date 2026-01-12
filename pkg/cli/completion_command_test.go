package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCompletionCommand(t *testing.T) {
	cmd := NewCompletionCommand()

	assert.Equal(t, "completion", cmd.Name())
	assert.Equal(t, "Generate shell completion scripts for gh aw commands", cmd.Short)
	assert.Contains(t, cmd.Long, "Tab completion provides")
	assert.Equal(t, []string{"bash", "zsh", "fish", "powershell"}, cmd.ValidArgs)
}

func TestCompletionCommand_Bash(t *testing.T) {
	rootCmd := &cobra.Command{Use: "gh"}
	completionCmd := NewCompletionCommand()
	rootCmd.AddCommand(completionCmd)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)

	rootCmd.SetArgs([]string{"completion", "bash"})
	err := rootCmd.Execute()

	require.NoError(t, err)
	output := buf.String()

	// Verify bash completion script contains expected markers
	assert.Contains(t, output, "# bash completion")
	assert.Contains(t, output, "__start_gh")
	assert.Contains(t, output, "complete -o default -F __start_gh gh")
}

func TestCompletionCommand_Zsh(t *testing.T) {
	rootCmd := &cobra.Command{Use: "gh"}
	completionCmd := NewCompletionCommand()
	rootCmd.AddCommand(completionCmd)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)

	rootCmd.SetArgs([]string{"completion", "zsh"})
	err := rootCmd.Execute()

	require.NoError(t, err)
	output := buf.String()

	// Verify zsh completion script contains expected markers
	assert.Contains(t, output, "#compdef gh")
}

func TestCompletionCommand_Fish(t *testing.T) {
	rootCmd := &cobra.Command{Use: "gh"}
	completionCmd := NewCompletionCommand()
	rootCmd.AddCommand(completionCmd)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)

	rootCmd.SetArgs([]string{"completion", "fish"})
	err := rootCmd.Execute()

	require.NoError(t, err)
	output := buf.String()

	// Verify fish completion script contains expected markers
	assert.Contains(t, output, "complete -c gh")
}

func TestCompletionCommand_PowerShell(t *testing.T) {
	rootCmd := &cobra.Command{Use: "gh"}
	completionCmd := NewCompletionCommand()
	rootCmd.AddCommand(completionCmd)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)

	rootCmd.SetArgs([]string{"completion", "powershell"})
	err := rootCmd.Execute()

	require.NoError(t, err)
	output := buf.String()

	// Verify PowerShell completion script contains expected markers
	assert.Contains(t, output, "Register-ArgumentCompleter")
}

func TestCompletionCommand_InvalidShell(t *testing.T) {
	rootCmd := &cobra.Command{Use: "gh"}
	completionCmd := NewCompletionCommand()
	rootCmd.AddCommand(completionCmd)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)

	rootCmd.SetArgs([]string{"completion", "invalid"})
	err := rootCmd.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid argument")
}

func TestCompletionCommand_NoArgs(t *testing.T) {
	rootCmd := &cobra.Command{Use: "gh"}
	completionCmd := NewCompletionCommand()
	rootCmd.AddCommand(completionCmd)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)

	rootCmd.SetArgs([]string{"completion"})
	err := rootCmd.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg(s)")
}

func TestCompletionCommand_InstallSubcommand(t *testing.T) {
	cmd := NewCompletionCommand()

	// Verify install subcommand exists
	installCmd := findSubcommand(cmd, "install")
	require.NotNil(t, installCmd, "install subcommand should exist")

	assert.Equal(t, "install", installCmd.Name())
	assert.Equal(t, "Install shell completion for the detected shell", installCmd.Short)
	assert.Contains(t, installCmd.Long, "Automatically install shell completion")
}

func TestCompletionCommand_LongHelp(t *testing.T) {
	cmd := NewCompletionCommand()

	assert.Contains(t, cmd.Long, "Tab completion provides")
	assert.Contains(t, cmd.Long, "Command name completion")
	assert.Contains(t, cmd.Long, "Workflow name completion")
	assert.Contains(t, cmd.Long, "Engine name completion")
	assert.Contains(t, cmd.Long, "bash, zsh, fish, powershell")
}

func TestCompletionCommand_Examples(t *testing.T) {
	cmd := NewCompletionCommand()

	// Verify examples are present for all shells
	assert.Contains(t, cmd.Long, "gh aw completion install")
	assert.Contains(t, cmd.Long, "gh aw completion bash")
	assert.Contains(t, cmd.Long, "gh aw completion zsh")
	assert.Contains(t, cmd.Long, "gh aw completion fish")
	assert.Contains(t, cmd.Long, "gh aw completion powershell")
}

func TestCompletionCommand_ValidArgs(t *testing.T) {
	cmd := NewCompletionCommand()

	expectedArgs := []string{"bash", "zsh", "fish", "powershell"}
	assert.Equal(t, expectedArgs, cmd.ValidArgs)
}

// Helper function to find a subcommand by name
func findSubcommand(cmd *cobra.Command, name string) *cobra.Command {
	for _, subCmd := range cmd.Commands() {
		if subCmd.Name() == name {
			return subCmd
		}
	}
	return nil
}

// TestCompletionCommand_BashScriptFormat verifies the bash completion script
// has the correct format and essential functions
func TestCompletionCommand_BashScriptFormat(t *testing.T) {
	rootCmd := &cobra.Command{Use: "gh"}
	completionCmd := NewCompletionCommand()
	rootCmd.AddCommand(completionCmd)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)

	rootCmd.SetArgs([]string{"completion", "bash"})
	err := rootCmd.Execute()

	require.NoError(t, err)
	output := buf.String()

	// Check for essential bash completion functions
	essentialFunctions := []string{
		"__gh_debug",
		"__gh_handle_go_custom_completion",
		"__start_gh",
	}

	for _, fn := range essentialFunctions {
		assert.Contains(t, output, fn, "Bash completion script should contain function: %s", fn)
	}
}

// TestCompletionCommand_ZshScriptFormat verifies the zsh completion script
// has the correct format
func TestCompletionCommand_ZshScriptFormat(t *testing.T) {
	rootCmd := &cobra.Command{Use: "gh"}
	completionCmd := NewCompletionCommand()
	rootCmd.AddCommand(completionCmd)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)

	rootCmd.SetArgs([]string{"completion", "zsh"})
	err := rootCmd.Execute()

	require.NoError(t, err)
	output := buf.String()

	// Zsh completion scripts should start with #compdef
	assert.True(t, strings.HasPrefix(output, "#compdef"), "Zsh script should start with #compdef")
}

// TestCompletionCommand_FishScriptFormat verifies the fish completion script
// has the correct format
func TestCompletionCommand_FishScriptFormat(t *testing.T) {
	rootCmd := &cobra.Command{Use: "gh"}
	completionCmd := NewCompletionCommand()
	rootCmd.AddCommand(completionCmd)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)

	rootCmd.SetArgs([]string{"completion", "fish"})
	err := rootCmd.Execute()

	require.NoError(t, err)
	output := buf.String()

	// Fish uses "complete -c" commands
	assert.Contains(t, output, "complete -c gh")
	assert.Contains(t, output, "function __gh_")
}

// TestCompletionCommand_PowerShellScriptFormat verifies the PowerShell completion script
// has the correct format
func TestCompletionCommand_PowerShellScriptFormat(t *testing.T) {
	rootCmd := &cobra.Command{Use: "gh"}
	completionCmd := NewCompletionCommand()
	rootCmd.AddCommand(completionCmd)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)

	rootCmd.SetArgs([]string{"completion", "powershell"})
	err := rootCmd.Execute()

	require.NoError(t, err)
	output := buf.String()

	// PowerShell uses Register-ArgumentCompleter
	assert.Contains(t, output, "Register-ArgumentCompleter")
	assert.Contains(t, output, "-CommandName")
}
