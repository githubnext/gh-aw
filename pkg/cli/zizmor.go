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
	Desc           string `json:"desc"`
	URL            string `json:"url"`
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
			Annotation string `json:"annotation"`
		} `json:"symbolic"`
		Concrete struct {
			Location struct {
				StartPoint struct {
					Row    int `json:"row"`
					Column int `json:"column"`
				} `json:"start_point"`
			} `json:"location"`
		} `json:"concrete"`
	} `json:"locations"`
}

// runZizmorOnFile runs the zizmor security scanner on a single .lock.yml file using Docker
func runZizmorOnFile(lockFile string, verbose bool, strict bool) error {
	compileLog.Printf("Running zizmor security scanner on %s", lockFile)

	// Find git root to get the absolute path for Docker volume mount
	gitRoot, err := findGitRoot()
	if err != nil {
		return fmt.Errorf("failed to find git root: %w", err)
	}

	// Get the relative path from git root
	relPath, err := filepath.Rel(gitRoot, lockFile)
	if err != nil {
		return fmt.Errorf("failed to get relative path: %w", err)
	}

	// Build the Docker command with JSON output for easier parsing
	// docker run --rm -v "$(pwd)":/workdir -w /workdir ghcr.io/zizmorcore/zizmor:latest --format json <file>
	cmd := exec.Command(
		"docker",
		"run",
		"--rm",
		"-v", fmt.Sprintf("%s:/workdir", gitRoot),
		"-w", "/workdir",
		"ghcr.io/zizmorcore/zizmor:latest",
		"--format", "json",
		relPath,
	)

	// In verbose mode, show the command that users can run directly
	if verbose {
		dockerCmd := fmt.Sprintf("docker run --rm -v \"%s:/workdir\" -w /workdir ghcr.io/zizmorcore/zizmor:latest --format json %s",
			gitRoot, relPath)
		fmt.Fprintf(os.Stderr, "%s\n", console.FormatInfoMessage("Run zizmor directly: "+dockerCmd))
	}

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
					return fmt.Errorf("strict mode: zizmor found %d security warnings/errors in %s - workflows must have no zizmor findings in strict mode", totalWarnings, filepath.Base(lockFile))
				}
				// In non-strict mode, findings are logged but not treated as errors
				return nil
			}
			// Other exit codes are actual errors
			return fmt.Errorf("zizmor failed with exit code %d on %s", exitCode, filepath.Base(lockFile))
		}
		// Non-ExitError errors (e.g., command not found)
		return fmt.Errorf("zizmor failed on %s: %w", filepath.Base(lockFile), err)
	}

	return nil
}

// parseAndDisplayZizmorOutput parses zizmor JSON output and displays it in the desired format
// Returns the total number of warnings found
func parseAndDisplayZizmorOutput(stdout, stderr string, verbose bool) (int, error) {
	// Map findings to files for detailed display
	fileFindings := make(map[string][]zizmorFinding)

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
				// Initialize empty findings slice
				if _, exists := fileFindings[filePath]; !exists {
					fileFindings[filePath] = []zizmorFinding{}
				}
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

		// Organize findings by file
		for _, finding := range findings {
			// Track which files this finding affects (avoid duplicates)
			affectedFiles := make(map[string]bool)
			for _, location := range finding.Locations {
				filePath := location.Symbolic.Key.Local.GivenPath
				if filePath != "" && !affectedFiles[filePath] {
					affectedFiles[filePath] = true
					fileFindings[filePath] = append(fileFindings[filePath], finding)
					totalWarnings++
				}
			}
		}
	}

	// Display reformatted output for each completed file
	for _, filePath := range completedFiles {
		findings := fileFindings[filePath]
		count := len(findings)

		// Skip files with 0 warnings
		if count == 0 {
			continue
		}

		// Read file content for context display
		fileContent, err := os.ReadFile(filePath)
		var fileLines []string
		if err == nil {
			fileLines = strings.Split(string(fileContent), "\n")
		}

		// Display detailed findings using CompilerError format
		for _, finding := range findings {
			severity := finding.Determinations.Severity
			ident := finding.Ident
			desc := finding.Desc
			url := finding.URL

			// Find the primary location (first location in the list)
			if len(finding.Locations) > 0 {
				loc := finding.Locations[0]
				row := loc.Concrete.Location.StartPoint.Row
				col := loc.Concrete.Location.StartPoint.Column
				// Zizmor uses 0-based indexing, convert to 1-based for user display
				lineNum := row + 1
				colNum := col + 1

				// Create context lines around the error
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
				if severity == "High" || severity == "Critical" {
					errorType = "error"
				}

				// Build message with URL link if available
				message := fmt.Sprintf("[%s] %s: %s", severity, ident, desc)
				if url != "" {
					message = fmt.Sprintf("%s (%s)", message, url)
				}

				// Create and format CompilerError
				compilerErr := console.CompilerError{
					Position: console.ErrorPosition{
						File:   filePath,
						Line:   lineNum,
						Column: colNum,
					},
					Type:    errorType,
					Message: message,
					Context: context,
				}

				fmt.Fprint(os.Stderr, console.FormatError(compilerErr))
			}
		}
	}

	return totalWarnings, nil
}
