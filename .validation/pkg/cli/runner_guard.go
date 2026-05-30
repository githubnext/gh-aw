package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/logger"
)

var runnerGuardLog = logger.New("cli:runner_guard")

// runnerGuardFinding represents a single finding from runner-guard JSON output
type runnerGuardFinding struct {
	RuleID      string `json:"rule_id"`
	Name        string `json:"name"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Remediation string `json:"remediation"`
	File        string `json:"file"`
	Line        int    `json:"line"`
}

// runnerGuardOutput represents the complete JSON output from runner-guard
type runnerGuardOutput struct {
	Findings []runnerGuardFinding `json:"findings"`
	Score    int                  `json:"score,omitempty"`
	Grade    string               `json:"grade,omitempty"`
}

// runRunnerGuardOnDirectory runs the runner-guard taint analysis scanner on a directory
// containing workflows using the Docker image.
func runRunnerGuardOnDirectory(workflowDir string, verbose bool, strict bool) error {
	runnerGuardLog.Printf("Running runner-guard taint analysis on directory: %s", workflowDir)

	// Find git root to get the absolute path for Docker volume mount
	gitRoot, err := gitutil.FindGitRoot()
	if err != nil {
		return fmt.Errorf("failed to find git root: %w", err)
	}

	// Validate gitRoot is an absolute path (security: ensure trusted path from git)
	if !filepath.IsAbs(gitRoot) {
		return fmt.Errorf("git root is not an absolute path: %s", gitRoot)
	}

	// Determine the scan path: use workflowDir relative to gitRoot when possible,
	// so the scan is scoped to the compiled workflows directory.
	scanPath := "."
	if workflowDir != "" {
		relDir, relErr := filepath.Rel(gitRoot, workflowDir)
		if relErr == nil && relDir != ".." && !strings.HasPrefix(relDir, ".."+string(filepath.Separator)) {
			scanPath = relDir
		}
	}

	// Build the Docker command
	// docker run --rm -v "$gitRoot:/workdir" -w /workdir ghcr.io/vigilant-llc/runner-guard:latest scan <path> --format json
	// #nosec G204 -- gitRoot comes from git rev-parse (trusted source) and is validated as absolute path.
	// exec.Command with separate args (not shell execution) prevents command injection.
	cmd := exec.Command(
		"docker",
		"run",
		"--rm",
		"-v", gitRoot+":/workdir",
		"-w", "/workdir",
		RunnerGuardImage,
		"scan",
		scanPath,
		"--format", "json",
	)

	// Always show that runner-guard is running (regular verbosity)
	fmt.Fprintf(os.Stderr, "%s\n", console.FormatInfoMessage("Running runner-guard taint analysis scanner"))

	// In verbose mode, also show the command that users can run directly
	if verbose {
		dockerCmd := fmt.Sprintf("docker run --rm -v \"%s:/workdir\" -w /workdir %s scan %s --format json",
			gitRoot, RunnerGuardImage, scanPath)
		fmt.Fprintf(os.Stderr, "%s\n", console.FormatInfoMessage("Run runner-guard directly: "+dockerCmd))
	}

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run the command
	err = cmd.Run()

	// Parse and display output
	totalFindings, parseErr := parseAndDisplayRunnerGuardOutput(stdout.String(), verbose, gitRoot)
	if parseErr != nil {
		runnerGuardLog.Printf("Failed to parse runner-guard output: %v", parseErr)
		// Fall back to showing raw output
		if stdout.Len() > 0 {
			fmt.Fprint(os.Stderr, stdout.String())
		}
		if stderr.Len() > 0 {
			fmt.Fprint(os.Stderr, stderr.String())
		}
	}

	// Check if the error is due to findings or actual failure
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode := exitErr.ExitCode()
			runnerGuardLog.Printf("runner-guard exited with code %d (findings=%d)", exitCode, totalFindings)
			// Exit code 1 typically indicates findings in the repository
			if exitCode == 1 {
				if strict {
					if parseErr != nil {
						// JSON parsing failed but exit code confirms findings exist
						return fmt.Errorf("strict mode: runner-guard exited with code 1 (findings present) and output could not be parsed: %w", parseErr)
					}
					if totalFindings > 0 {
						return fmt.Errorf("strict mode: runner-guard found %d security findings - workflows must have no runner-guard findings in strict mode", totalFindings)
					}
					// Exit code 1 with no parseable findings is still a failure in strict mode
					return errors.New("strict mode: runner-guard exited with code 1 indicating findings are present")
				}
				// In non-strict mode, findings are logged but not treated as errors
				return nil
			}
			// Other exit codes are actual errors
			return fmt.Errorf("runner-guard failed with exit code %d", exitCode)
		}
		// Non-ExitError errors (e.g., command not found)
		return fmt.Errorf("runner-guard failed: %w", err)
	}

	return nil
}

// parseAndDisplayRunnerGuardOutput parses runner-guard JSON output and displays findings.
// Returns the total number of findings found.
func parseAndDisplayRunnerGuardOutput(stdout string, verbose bool, gitRoot string) (int, error) {
	if stdout == "" {
		return 0, nil // No output means no findings
	}

	trimmed := strings.TrimSpace(stdout)
	if !strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "[") {
		if len(trimmed) > 0 {
			return 0, fmt.Errorf("unexpected runner-guard output format: %s", trimmed)
		}
		return 0, nil
	}

	var output runnerGuardOutput
	if err := json.Unmarshal([]byte(stdout), &output); err != nil {
		return 0, fmt.Errorf("failed to parse runner-guard JSON output: %w", err)
	}

	totalFindings := len(output.Findings)
	if totalFindings == 0 {
		return 0, nil
	}

	// Display score/grade if present
	if output.Score > 0 || output.Grade != "" {
		fmt.Fprintf(os.Stderr, "%s\n", console.FormatInfoMessage(
			fmt.Sprintf("Runner-Guard Score: %d/100 (Grade: %s)", output.Score, output.Grade),
		))
	}

	// Group findings by file for better readability
	findingsByFile := make(map[string][]runnerGuardFinding)
	for _, finding := range output.Findings {
		findingsByFile[finding.File] = append(findingsByFile[finding.File], finding)
	}

	// Display findings for each file
	for filePath, findings := range findingsByFile {
		// Validate and sanitize file path to prevent path traversal
		cleanPath := filepath.Clean(filePath)

		absPath := cleanPath
		if !filepath.IsAbs(cleanPath) {
			absPath = filepath.Join(gitRoot, cleanPath)
		}

		absGitRoot, err := filepath.Abs(gitRoot)
		if err != nil {
			runnerGuardLog.Printf("Failed to get absolute path for git root: %v", err)
			continue
		}

		absPath, err = filepath.Abs(absPath)
		if err != nil {
			runnerGuardLog.Printf("Failed to get absolute path for %s: %v", filePath, err)
			continue
		}

		// Check if the resolved path is within gitRoot to prevent path traversal
		relPath, err := filepath.Rel(absGitRoot, absPath)
		if err != nil || relPath == ".." || strings.HasPrefix(relPath, ".."+string(filepath.Separator)) {
			runnerGuardLog.Printf("Skipping file outside git root: %s", filePath)
			continue
		}

		// Read file content for context display
		// #nosec G304 -- absPath is validated through: 1) filepath.Clean() normalization,
		// 2) absolute path resolution, and 3) filepath.Rel() check ensuring it's within gitRoot.
		// Path traversal attacks are prevented by the boundary validation above.
		fileContent, err := os.ReadFile(absPath)
		var fileLines []string
		if err == nil {
			fileLines = strings.Split(string(fileContent), "\n")
		}

		for _, finding := range findings {
			lineNum := finding.Line
			if lineNum == 0 {
				lineNum = 1
			}

			// Create context lines around the finding
			var context []string
			if len(fileLines) > 0 && lineNum > 0 && lineNum <= len(fileLines) {
				startLine := max(1, lineNum-2)
				endLine := min(len(fileLines), lineNum+2)
				for i := startLine; i <= endLine; i++ {
					if i-1 < len(fileLines) {
						context = append(context, fileLines[i-1])
					}
				}
			}

			// Map severity to error type
			errorType := "warning"
			switch strings.ToLower(finding.Severity) {
			case "critical", "high", "error":
				errorType = "error"
			case "note", "info":
				errorType = "info"
			}

			// Build message
			message := fmt.Sprintf("[%s] %s: %s", finding.Severity, finding.RuleID, finding.Name)
			if finding.Description != "" {
				message = fmt.Sprintf("%s - %s", message, finding.Description)
			}

			compilerErr := console.CompilerError{
				Position: console.ErrorPosition{
					File:   finding.File,
					Line:   lineNum,
					Column: 1,
				},
				Type:    errorType,
				Message: message,
				Context: context,
			}

			fmt.Fprint(os.Stderr, console.FormatError(compilerErr))
		}
	}

	return totalFindings, nil
}
