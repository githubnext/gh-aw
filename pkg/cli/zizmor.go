package cli

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
)

// zizmorFinding represents a single finding from zizmor JSON output
type zizmorFinding struct {
	Ident          string `json:"ident"`
	Determinations struct {
		Severity string `json:"severity"`
	} `json:"determinations"`
	Locations []struct {
		Symbolic struct {
			Key struct {
				Local struct {
					GivenPath string `json:"given_path"`
				} `json:"Local"`
			} `json:"key"`
		} `json:"symbolic"`
	} `json:"locations"`
}

// runZizmor runs the zizmor security scanner on generated .lock.yml files using Docker
func runZizmor(workflowsDir string, verbose bool, strict bool) error {
	compileLog.Print("Running zizmor security scanner")

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Running zizmor security scanner on generated .lock.yml files..."))
	}

	// Find git root to get the absolute path for Docker volume mount
	gitRoot, err := findGitRoot()
	if err != nil {
		return fmt.Errorf("failed to find git root: %w", err)
	}

	// Get the absolute path of the workflows directory
	var absWorkflowsDir string
	if filepath.IsAbs(workflowsDir) {
		absWorkflowsDir = workflowsDir
	} else {
		absWorkflowsDir = filepath.Join(gitRoot, workflowsDir)
	}

	compileLog.Printf("Running zizmor on directory: %s", absWorkflowsDir)

	// Build the Docker command with JSON output for easier parsing
	// docker run --rm -v "$(pwd)":/workdir -w /workdir ghcr.io/zizmorcore/zizmor:latest --format json .
	cmd := exec.Command(
		"docker",
		"run",
		"--rm",
		"-v", fmt.Sprintf("%s:/workdir", gitRoot),
		"-w", "/workdir",
		"ghcr.io/zizmorcore/zizmor:latest",
		"--format", "json",
		".",
	)

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run the command
	err = cmd.Run()

	// Parse and reformat the output, get total warning count
	totalWarnings, parseErr := parseAndDisplayZizmorOutput(stdout.String(), stderr.String(), verbose)
	if parseErr != nil {
		compileLog.Printf("Failed to parse zizmor output: %v", parseErr)
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
		// zizmor uses exit codes to indicate findings:
		// 0 = no findings
		// 10-13 = findings at different severity levels
		// 14 = findings with mixed severities
		// Other codes = actual errors
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode := exitErr.ExitCode()
			compileLog.Printf("Zizmor exited with code %d", exitCode)
			// Exit codes 10-14 indicate findings
			if exitCode >= 10 && exitCode <= 14 {
				// In strict mode, findings are treated as errors
				if strict {
					return fmt.Errorf("strict mode: zizmor found %d security warnings/errors - workflows must have no zizmor findings in strict mode", totalWarnings)
				}
				// In non-strict mode, findings are logged but not treated as errors
				return nil
			}
			// Other exit codes are actual errors
			return fmt.Errorf("zizmor failed with exit code %d", exitCode)
		}
		// Non-ExitError errors (e.g., command not found)
		return fmt.Errorf("zizmor failed: %w", err)
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Zizmor security scan completed - no findings"))
	}

	return nil
}

// parseAndDisplayZizmorOutput parses zizmor JSON output and displays it in the desired format
// Returns the total number of warnings found
func parseAndDisplayZizmorOutput(stdout, stderr string, verbose bool) (int, error) {
	// Count findings per file
	fileFindings := make(map[string]int)

	// Parse stderr for "completed" messages to get list of files
	completedFiles := []string{}
	scanner := bufio.NewScanner(strings.NewReader(stderr))
	for scanner.Scan() {
		line := scanner.Text()
		// Look for lines like: " INFO audit: zizmor: 🌈 completed ./.github/workflows/pdf-summary.lock.yml"
		if strings.Contains(line, "INFO audit: zizmor: 🌈 completed") {
			parts := strings.Split(line, "completed ")
			if len(parts) == 2 {
				filePath := strings.TrimSpace(parts[1])
				completedFiles = append(completedFiles, filePath)
				// Initialize count to 0
				fileFindings[filePath] = 0
			}
		}
	}

	// Parse JSON findings from stdout
	var findings []zizmorFinding
	totalWarnings := 0
	if stdout != "" && strings.HasPrefix(strings.TrimSpace(stdout), "[") {
		if err := json.Unmarshal([]byte(stdout), &findings); err != nil {
			return 0, fmt.Errorf("failed to parse zizmor JSON output: %w", err)
		}

		// Count findings per file - each finding counts as 1 regardless of how many locations it has
		for _, finding := range findings {
			// Track which files this finding affects
			affectedFiles := make(map[string]bool)
			for _, location := range finding.Locations {
				filePath := location.Symbolic.Key.Local.GivenPath
				if filePath != "" && !affectedFiles[filePath] {
					affectedFiles[filePath] = true
					fileFindings[filePath]++
					totalWarnings++
				}
			}
		}
	}

	// Display reformatted output for each completed file
	for _, filePath := range completedFiles {
		count := fileFindings[filePath]
		// Format: 🌈 zizmor xx warnings in <filepath>
		warningText := "warnings"
		if count == 1 {
			warningText = "warning"
		}
		fmt.Fprintf(os.Stderr, "🌈 zizmor %d %s in %s\n", count, warningText, filePath)
	}

	return totalWarnings, nil
}
