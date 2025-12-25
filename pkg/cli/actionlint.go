package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var actionlintLog = logger.New("cli:actionlint")

// actionlintVersion caches the actionlint version to avoid repeated Docker calls
var actionlintVersion string

// actionlintStats tracks aggregate statistics across all actionlint validations
var actionlintStats *ActionlintStats

// ActionlintStats tracks actionlint validation statistics across all files
type ActionlintStats struct {
	TotalWorkflows int
	TotalErrors    int
	TotalWarnings  int
	ErrorsByKind   map[string]int
}

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

// initActionlintStats initializes the global actionlint statistics tracker
func initActionlintStats() {
	actionlintStats = &ActionlintStats{
		ErrorsByKind: make(map[string]int),
	}
}

// displayActionlintSummary displays aggregate statistics for all actionlint validations
func displayActionlintSummary() {
	if actionlintStats == nil || actionlintStats.TotalWorkflows == 0 {
		return
	}

	// Create visual separator
	separator := strings.Repeat("━", 60)

	fmt.Fprintf(os.Stderr, "\n%s\n", separator)
	fmt.Fprintf(os.Stderr, "%s\n", console.FormatInfoMessage("Actionlint Summary"))
	fmt.Fprintf(os.Stderr, "%s\n\n", separator)

	// Show total workflows checked
	fmt.Fprintf(os.Stderr, "%s\n",
		console.FormatSuccessMessage(fmt.Sprintf("Checked %d workflow(s)", actionlintStats.TotalWorkflows)))

	// Show total issues found
	totalIssues := actionlintStats.TotalErrors + actionlintStats.TotalWarnings
	if totalIssues > 0 {
		issueText := fmt.Sprintf("Found %d issue(s)", totalIssues)
		if actionlintStats.TotalErrors > 0 && actionlintStats.TotalWarnings > 0 {
			issueText += fmt.Sprintf(" (%d error(s), %d warning(s))", actionlintStats.TotalErrors, actionlintStats.TotalWarnings)
		} else if actionlintStats.TotalErrors > 0 {
			issueText += fmt.Sprintf(" (%d error(s))", actionlintStats.TotalErrors)
		} else if actionlintStats.TotalWarnings > 0 {
			issueText += fmt.Sprintf(" (%d warning(s))", actionlintStats.TotalWarnings)
		}
		fmt.Fprintf(os.Stderr, "%s\n", console.FormatWarningMessage(issueText))

		// Break down by error kind if we have multiple kinds
		if len(actionlintStats.ErrorsByKind) > 0 {
			fmt.Fprintf(os.Stderr, "\n%s\n", console.FormatInfoMessage("Issues by type:"))
			for kind, count := range actionlintStats.ErrorsByKind {
				fmt.Fprintf(os.Stderr, "  • %s: %d\n", kind, count)
			}
		}
	} else {
		fmt.Fprintf(os.Stderr, "%s\n",
			console.FormatSuccessMessage("No issues found"))
	}

	fmt.Fprintf(os.Stderr, "\n%s\n", separator)
}

// getActionlintVersion fetches and caches the actionlint version from Docker
func getActionlintVersion() (string, error) {
	// Return cached version if already fetched
	if actionlintVersion != "" {
		return actionlintVersion, nil
	}

	actionlintLog.Print("Fetching actionlint version from Docker")

	// Run docker command to get version with a 30 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(
		ctx,
		"docker",
		"run",
		"--rm",
		"rhysd/actionlint:latest",
		"--version",
	)

	output, err := cmd.Output()
	if err != nil {
		actionlintLog.Printf("Failed to get actionlint version: %v", err)
		return "", fmt.Errorf("failed to get actionlint version: %w", err)
	}

	// Parse version from output (format: "1.7.9\ninstalled by...\nbuilt with...")
	// We only want the first line which contains the version number
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 {
		return "", fmt.Errorf("no version output from actionlint")
	}
	version := strings.TrimSpace(lines[0])
	actionlintVersion = version
	actionlintLog.Printf("Cached actionlint version: %s", version)

	return version, nil
}

// ensureActionlintConfig creates .github/actionlint.yaml to configure custom runner labels if it doesn't exist
func ensureActionlintConfig(gitRoot string) error {
	configPath := filepath.Join(gitRoot, ".github", "actionlint.yaml")
	actionlintLog.Printf("Ensuring actionlint config at: %s", configPath)

	// Check if config already exists
	if _, err := os.Stat(configPath); err == nil {
		// Config exists, do not update it
		actionlintLog.Print("actionlint.yaml already exists, skipping creation")
		return nil
	}

	// Create the config file
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

	actionlintLog.Printf("Created actionlint.yaml at %s", configPath)
	return nil
}

// runActionlintOnFile runs the actionlint linter on a single .lock.yml file using Docker
func runActionlintOnFile(lockFile string, verbose bool, strict bool) error {
	actionlintLog.Printf("Running actionlint on file: %s (verbose=%t, strict=%t)", lockFile, verbose, strict)

	// Display actionlint version on first use
	if actionlintVersion == "" {
		version, err := getActionlintVersion()
		if err != nil {
			// Log error but continue - version display is not critical
			actionlintLog.Printf("Could not fetch actionlint version: %v", err)
		} else {
			fmt.Fprintf(os.Stderr, "%s\n", console.FormatInfoMessage(fmt.Sprintf("Using actionlint %s", version)))
		}
	}

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
	// Set a timeout context to prevent Docker from hanging indefinitely (5 minutes should be sufficient)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(
		ctx,
		"docker",
		"run",
		"--rm",
		"-v", fmt.Sprintf("%s:/workdir", gitRoot),
		"-w", "/workdir",
		"rhysd/actionlint:latest",
		"-format", "{{json .}}",
		relPath,
	)

	// Always show that actionlint is running (regular verbosity)
	fmt.Fprintf(os.Stderr, "%s\n", console.FormatInfoMessage(fmt.Sprintf("Running actionlint (includes shellcheck & pyflakes) on %s", relPath)))

	// In verbose mode, also show the command that users can run directly
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

	// Check for timeout
	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("actionlint timed out after 5 minutes on %s - this may indicate a Docker or network issue", filepath.Base(lockFile))
	}

	// Track this workflow in statistics
	if actionlintStats != nil {
		actionlintStats.TotalWorkflows++
	}

	// Parse and reformat the output, get total error count and error details
	totalErrors, errorsByKind, parseErr := parseAndDisplayActionlintOutput(stdout.String(), verbose)
	if parseErr != nil {
		actionlintLog.Printf("Failed to parse actionlint output: %v", parseErr)
		// Fall back to showing raw output
		if stdout.Len() > 0 {
			fmt.Fprint(os.Stderr, stdout.String())
		}
		if stderr.Len() > 0 {
			fmt.Fprint(os.Stderr, stderr.String())
		}
	} else {
		// Track error statistics
		if actionlintStats != nil {
			actionlintStats.TotalErrors += totalErrors
			for kind, count := range errorsByKind {
				actionlintStats.ErrorsByKind[kind] += count
			}
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
			actionlintLog.Printf("Actionlint exited with code %d, found %d errors", exitCode, totalErrors)
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
// Returns the total number of errors found and a breakdown by kind
func parseAndDisplayActionlintOutput(stdout string, verbose bool) (int, map[string]int, error) {
	// Skip if no output
	if stdout == "" || strings.TrimSpace(stdout) == "" {
		actionlintLog.Print("No actionlint output to parse")
		return 0, make(map[string]int), nil
	}

	// Parse JSON errors from stdout - actionlint outputs a single JSON array
	var errors []actionlintError
	if err := json.Unmarshal([]byte(stdout), &errors); err != nil {
		return 0, nil, fmt.Errorf("failed to parse actionlint JSON output: %w", err)
	}

	totalErrors := len(errors)
	actionlintLog.Printf("Parsed %d actionlint errors from output", totalErrors)

	// Track errors by kind
	errorsByKind := make(map[string]int)

	// Display errors using CompilerError format
	for _, err := range errors {
		// Track error kind
		if err.Kind != "" {
			errorsByKind[err.Kind]++
		}

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

	return totalErrors, errorsByKind, nil
}
