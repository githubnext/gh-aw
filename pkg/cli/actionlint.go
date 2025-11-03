package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
)

// actionlintError represents a single error from actionlint JSON output
type actionlintError struct {
	Message   string `json:"message"`
	Filepath  string `json:"filepath"`
	Line      int    `json:"line"`
	Column    int    `json:"column"`
	Kind      string `json:"kind"`
	Snippet   string `json:"snippet"`
	EndColumn int    `json:"end_column"`
}

// ensureActionlintConfig creates or updates .github/actionlint.yaml to configure custom runner labels
func ensureActionlintConfig(gitRoot string) error {
	configPath := filepath.Join(gitRoot, ".github", "actionlint.yaml")

	// Check if config already exists
	if _, err := os.Stat(configPath); err == nil {
		// Config exists, check if it already has ubuntu-slim
		content, readErr := os.ReadFile(configPath)
		if readErr != nil {
			return fmt.Errorf("failed to read existing actionlint.yaml: %w", readErr)
		}

		// If ubuntu-slim is already in the config, no need to update
		if strings.Contains(string(content), "ubuntu-slim") {
			compileLog.Print("actionlint.yaml already contains ubuntu-slim configuration")
			return nil
		}

		compileLog.Print("actionlint.yaml exists but doesn't contain ubuntu-slim, updating...")
	}

	// Create or update the config file
	configContent := `# Configuration for actionlint
# See https://github.com/rhysd/actionlint/blob/main/docs/config.md

self-hosted-runner:
  # Labels of self-hosted runner in array of strings
  labels:
    - ubuntu-slim
`

	// Ensure .github directory exists
	githubDir := filepath.Join(gitRoot, ".github")
	if err := os.MkdirAll(githubDir, 0755); err != nil {
		return fmt.Errorf("failed to create .github directory: %w", err)
	}

	// Write the config file
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to write actionlint.yaml: %w", err)
	}

	compileLog.Printf("Created/updated actionlint.yaml at %s", configPath)
	return nil
}

// runActionlintOnFile runs the actionlint linter on a single .lock.yml file using Docker
func runActionlintOnFile(lockFile string, verbose bool, strict bool) error {
	compileLog.Printf("Running actionlint linter on %s", lockFile)

	// Find git root to get the absolute path for Docker volume mount
	gitRoot, err := findGitRoot()
	if err != nil {
		return fmt.Errorf("failed to find git root: %w", err)
	}

	// Ensure actionlint config exists with custom runner labels
	if err := ensureActionlintConfig(gitRoot); err != nil {
		return fmt.Errorf("failed to ensure actionlint config: %w", err)
	}

	// Get the relative path from git root
	relPath, err := filepath.Rel(gitRoot, lockFile)
	if err != nil {
		return fmt.Errorf("failed to get relative path: %w", err)
	}

	// Build the Docker command with JSON output for easier parsing
	// docker run --rm -v "$(pwd)":/workdir -w /workdir rhysd/actionlint:latest -format '{{json .}}' <file>
	cmd := exec.Command(
		"docker",
		"run",
		"--rm",
		"-v", fmt.Sprintf("%s:/workdir", gitRoot),
		"-w", "/workdir",
		"rhysd/actionlint:latest",
		"-format", "{{json .}}",
		relPath,
	)

	// In verbose mode, show the command that users can run directly
	if verbose {
		dockerCmd := fmt.Sprintf("docker run --rm -v \"%s:/workdir\" -w /workdir rhysd/actionlint:latest -format '{{json .}}' %s",
			gitRoot, relPath)
		fmt.Fprintf(os.Stderr, "%s\n", console.FormatInfoMessage("Run actionlint directly: "+dockerCmd))
	}

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run the command
	err = cmd.Run()

	// Parse and reformat the output, get total error count
	totalErrors, parseErr := parseAndDisplayActionlintOutput(stdout.String(), verbose)
	if parseErr != nil {
		compileLog.Printf("Failed to parse actionlint output: %v", parseErr)
		// Fall back to showing raw output
		if stdout.Len() > 0 {
			fmt.Fprint(os.Stderr, stdout.String())
		}
		if stderr.Len() > 0 {
			fmt.Fprint(os.Stderr, stderr.String())
		}
	}

	// Check if the error is due to findings (expected) or actual failure
	if err != nil {
		// actionlint uses exit code 1 when errors are found
		// Exit code 0 = no errors
		// Exit code 1 = errors found
		// Other codes = actual errors
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode := exitErr.ExitCode()
			compileLog.Printf("Actionlint exited with code %d", exitCode)
			// Exit code 1 indicates errors were found
			if exitCode == 1 {
				// In strict mode, errors are treated as compilation failures
				if strict {
					return fmt.Errorf("strict mode: actionlint found %d errors in %s - workflows must have no actionlint errors in strict mode", totalErrors, filepath.Base(lockFile))
				}
				// In non-strict mode, errors are logged but not treated as failures
				return nil
			}
			// Other exit codes are actual errors
			return fmt.Errorf("actionlint failed with exit code %d on %s", exitCode, filepath.Base(lockFile))
		}
		// Non-ExitError errors (e.g., command not found)
		return fmt.Errorf("actionlint failed on %s: %w", filepath.Base(lockFile), err)
	}

	return nil
}

// parseAndDisplayActionlintOutput parses actionlint JSON output and displays it in the desired format
// Returns the total number of errors found
func parseAndDisplayActionlintOutput(stdout string, verbose bool) (int, error) {
	// Skip if no output
	if stdout == "" || strings.TrimSpace(stdout) == "" {
		return 0, nil
	}

	// Parse JSON errors from stdout - actionlint outputs a single JSON array
	var errors []actionlintError
	if err := json.Unmarshal([]byte(stdout), &errors); err != nil {
		return 0, fmt.Errorf("failed to parse actionlint JSON output: %w", err)
	}

	totalErrors := len(errors)

	// Display errors using CompilerError format
	for _, err := range errors {
		// Read file content for context display
		fileContent, readErr := os.ReadFile(err.Filepath)
		var fileLines []string
		if readErr == nil {
			fileLines = strings.Split(string(fileContent), "\n")
		}

		// Create context lines around the error
		var context []string
		if len(fileLines) > 0 && err.Line > 0 && err.Line <= len(fileLines) {
			startLine := max(1, err.Line-2)
			endLine := min(len(fileLines), err.Line+2)

			for i := startLine; i <= endLine; i++ {
				if i-1 < len(fileLines) {
					context = append(context, fileLines[i-1])
				}
			}
		}

		// Map kind to error type
		// Most actionlint errors are actual errors, not warnings
		errorType := "error"
		if strings.Contains(strings.ToLower(err.Kind), "warning") {
			errorType = "warning"
		}

		// Build message with kind if available
		message := err.Message
		if err.Kind != "" {
			message = fmt.Sprintf("[%s] %s", err.Kind, err.Message)
		}

		// Create and format CompilerError
		compilerErr := console.CompilerError{
			Position: console.ErrorPosition{
				File:   err.Filepath,
				Line:   err.Line,
				Column: err.Column,
			},
			Type:    errorType,
			Message: message,
			Context: context,
		}

		fmt.Fprint(os.Stderr, console.FormatError(compilerErr))
	}

	return totalErrors, nil
}
