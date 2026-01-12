//go:build integration

package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompletionCommandIntegration tests the completion command via the CLI binary
func TestCompletionCommandIntegration(t *testing.T) {
	setup := setupIntegrationTest(t)
	defer setup.cleanup()

	tests := []struct {
		name     string
		args     []string
		wantExit int
		wantIn   []string
		wantOut  []string
	}{
		{
			name:     "completion bash generates script",
			args:     []string{"completion", "bash"},
			wantExit: 0,
			wantIn:   []string{"# bash completion", "__start_gh", "complete -o default"},
		},
		{
			name:     "completion zsh generates script",
			args:     []string{"completion", "zsh"},
			wantExit: 0,
			wantIn:   []string{"#compdef gh", "_gh"},
		},
		{
			name:     "completion fish generates script",
			args:     []string{"completion", "fish"},
			wantExit: 0,
			wantIn:   []string{"# fish completion", "complete -c gh"},
		},
		{
			name:     "completion powershell generates script",
			args:     []string{"completion", "powershell"},
			wantExit: 0,
			wantIn:   []string{"# powershell completion", "Register-ArgumentCompleter"},
		},
		{
			name:     "completion with invalid shell fails",
			args:     []string{"completion", "invalid"},
			wantExit: 1,
			wantOut:  []string{"invalid argument"},
		},
		{
			name:     "completion without args fails",
			args:     []string{"completion"},
			wantExit: 1,
			wantOut:  []string{"accepts 1 arg(s)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(setup.binaryPath, tt.args...)
			output, err := cmd.CombinedOutput()
			outputStr := string(output)

			// Check exit code
			if tt.wantExit == 0 {
				require.NoError(t, err, "Command should succeed: %s", outputStr)
			} else {
				require.Error(t, err, "Command should fail")
			}

			// Check for expected strings in output
			for _, want := range tt.wantIn {
				assert.Contains(t, outputStr, want, "Output should contain: %s", want)
			}

			// Check for expected strings in stderr/error output
			for _, want := range tt.wantOut {
				assert.Contains(t, outputStr, want, "Output should contain: %s", want)
			}
		})
	}
}

// TestCompletionCommandSubcommandsIntegration tests install and uninstall subcommands
func TestCompletionCommandSubcommandsIntegration(t *testing.T) {
	setup := setupIntegrationTest(t)
	defer setup.cleanup()

	// Test that install subcommand exists and shows help
	t.Run("install subcommand help", func(t *testing.T) {
		cmd := exec.Command(setup.binaryPath, "completion", "install", "--help")
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "install --help should succeed: %s", string(output))

		outputStr := string(output)
		assert.Contains(t, outputStr, "Automatically install shell completion")
		assert.Contains(t, outputStr, "Auto-detect and install")
	})

	// Test that uninstall subcommand exists and shows help
	t.Run("uninstall subcommand help", func(t *testing.T) {
		cmd := exec.Command(setup.binaryPath, "completion", "uninstall", "--help")
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "uninstall --help should succeed: %s", string(output))

		outputStr := string(output)
		assert.Contains(t, outputStr, "Automatically uninstall shell completion")
		assert.Contains(t, outputStr, "Auto-detect and uninstall")
	})
}

// TestCompletionInstallUninstallIntegration tests actual install and uninstall functionality
func TestCompletionInstallUninstallIntegration(t *testing.T) {
	setup := setupIntegrationTest(t)
	defer setup.cleanup()

	// Create a temporary home directory for testing
	tmpHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	// Set HOME to temp directory
	os.Setenv("HOME", tmpHome)

	// Set SHELL to bash for predictable testing
	originalShell := os.Getenv("SHELL")
	os.Setenv("SHELL", "/bin/bash")
	defer os.Setenv("SHELL", originalShell)

	t.Run("install creates completion file", func(t *testing.T) {
		cmd := exec.Command(setup.binaryPath, "completion", "install")
		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		// Should succeed
		require.NoError(t, err, "install should succeed: %s", outputStr)

		// Should indicate bash was detected
		assert.Contains(t, outputStr, "Detected shell: bash")

		// Should indicate where file was installed
		assert.Contains(t, outputStr, "Installed bash completion")

		// Verify the completion file was created
		completionPath := filepath.Join(tmpHome, ".bash_completion.d", "gh-aw")
		_, err = os.Stat(completionPath)
		assert.NoError(t, err, "Completion file should exist at: %s", completionPath)

		// Verify the file contains bash completion content
		if err == nil {
			content, readErr := os.ReadFile(completionPath)
			require.NoError(t, readErr, "Should be able to read completion file")
			assert.Contains(t, string(content), "# bash completion")
			assert.Contains(t, string(content), "__start_gh")
		}
	})

	t.Run("uninstall removes completion file", func(t *testing.T) {
		// First, ensure completion file exists
		completionPath := filepath.Join(tmpHome, ".bash_completion.d", "gh-aw")
		_, err := os.Stat(completionPath)
		require.NoError(t, err, "Completion file should exist before uninstall")

		// Run uninstall
		cmd := exec.Command(setup.binaryPath, "completion", "uninstall")
		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		// Should succeed
		require.NoError(t, err, "uninstall should succeed: %s", outputStr)

		// Should indicate bash was detected
		assert.Contains(t, outputStr, "Detected shell: bash")

		// Should indicate file was removed
		assert.Contains(t, outputStr, "Removed bash completion")

		// Verify the completion file was removed
		_, err = os.Stat(completionPath)
		assert.True(t, os.IsNotExist(err), "Completion file should be removed")
	})

	t.Run("uninstall when no file exists shows error", func(t *testing.T) {
		// Completion file should already be removed from previous test
		cmd := exec.Command(setup.binaryPath, "completion", "uninstall")
		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		// Should fail
		require.Error(t, err, "uninstall should fail when no file exists")

		// Should indicate no file was found
		assert.Contains(t, outputStr, "no bash completion file found")
	})
}

// TestCompletionCommandCompleteIntegration tests that the completion command itself can be completed
func TestCompletionCommandCompleteIntegration(t *testing.T) {
	setup := setupIntegrationTest(t)
	defer setup.cleanup()

	// Test completion of the completion command
	cmd := exec.Command(setup.binaryPath, "__complete", "completion", "")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "CLI __complete for completion command failed: %s", string(output))

	outputStr := string(output)

	// Should suggest shell types and subcommands
	assert.Contains(t, outputStr, "bash")
	assert.Contains(t, outputStr, "zsh")
	assert.Contains(t, outputStr, "fish")
	assert.Contains(t, outputStr, "powershell")
	assert.Contains(t, outputStr, "install")
	assert.Contains(t, outputStr, "uninstall")
}

// TestCompletionScriptContentIntegration validates that generated completion scripts are well-formed
func TestCompletionScriptContentIntegration(t *testing.T) {
	setup := setupIntegrationTest(t)
	defer setup.cleanup()

	tests := []struct {
		shell            string
		requiredContents []string
		description      string
	}{
		{
			shell: "bash",
			requiredContents: []string{
				"__gh_handle_go_custom_completion",
				"__start_gh",
				"_gh_completion",
				"_gh_completion_install",
				"_gh_completion_uninstall",
			},
			description: "bash script should contain completion functions for main command and subcommands",
		},
		{
			shell: "zsh",
			requiredContents: []string{
				"#compdef gh",
				"_gh()",
				"shellCompDirective",
			},
			description: "zsh script should be properly formatted",
		},
		{
			shell: "fish",
			requiredContents: []string{
				"function __gh_perform_completion",
				"complete -c gh",
			},
			description: "fish script should contain completion functions",
		},
		{
			shell: "powershell",
			requiredContents: []string{
				"Register-ArgumentCompleter",
				"-CommandName 'gh'",
				"${__ghCompleterBlock}",
			},
			description: "powershell script should register argument completer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			cmd := exec.Command(setup.binaryPath, "completion", tt.shell)
			output, err := cmd.CombinedOutput()
			require.NoError(t, err, "Completion script generation for %s failed: %s", tt.shell, string(output))

			outputStr := string(output)

			// Verify all required contents are present
			for _, required := range tt.requiredContents {
				assert.Contains(t, outputStr, required,
					"%s: %s should contain '%s'", tt.shell, tt.description, required)
			}

			// Verify the script is not empty
			assert.Greater(t, len(strings.TrimSpace(outputStr)), 100,
				"%s completion script should be substantial", tt.shell)
		})
	}
}

// TestCompletionVisibilityIntegration verifies that the completion command is visible in help
func TestCompletionVisibilityIntegration(t *testing.T) {
	setup := setupIntegrationTest(t)
	defer setup.cleanup()

	// Test that completion appears in main help
	cmd := exec.Command(setup.binaryPath, "--help")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Help command should succeed: %s", string(output))

	outputStr := string(output)

	// Should show completion command in Utilities section
	assert.Contains(t, outputStr, "Utilities:")
	assert.Contains(t, outputStr, "completion")
	assert.Contains(t, outputStr, "Generate shell completion scripts")
}
