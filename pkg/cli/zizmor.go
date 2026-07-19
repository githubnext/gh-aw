package cli

import (
	"bufio"
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
	"github.com/github/gh-aw/pkg/setutil"
)

var zizmorLog = logger.New("cli:zizmor")

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

// runZizmorOnFiles runs the zizmor security scanner on one or more .lock.yml files using Docker
func runZizmorOnFiles(lockFiles []string, verbose bool, strict bool) error {
	if len(lockFiles) == 0 {
		return nil
	}

	zizmorLog.Printf("Running zizmor security scanner on %d file(s): %v (verbose=%t, strict=%t)", len(lockFiles), lockFiles, verbose, strict)

	gitRoot, relPaths, err := runZizmorOnFilesPaths(lockFiles)
	if err != nil {
		return err
	}
	cmd := runZizmorOnFilesCommand(gitRoot, relPaths)
	runZizmorOnFilesAnnounce(lockFiles, relPaths, gitRoot, verbose)

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run the command
	err = cmd.Run()
	return runZizmorOnFilesResult(err, lockFiles, stdout, stderr, verbose, strict)
}

func runZizmorOnFilesPaths(lockFiles []string) (string, []string, error) {
	gitRoot, err := gitutil.FindGitRoot()
	if err != nil {
		return "", nil, fmt.Errorf("failed to find git root: %w", err)
	}
	if !filepath.IsAbs(gitRoot) {
		return "", nil, fmt.Errorf("git root must be an absolute path, got: %s", gitRoot)
	}
	var relPaths []string
	for _, lockFile := range lockFiles {
		relPath, err := filepath.Rel(gitRoot, lockFile)
		if err != nil {
			return "", nil, fmt.Errorf("failed to get relative path for %s: %w", lockFile, err)
		}
		relPaths = append(relPaths, relPath)
	}
	return gitRoot, relPaths, nil
}

func runZizmorOnFilesCommand(gitRoot string, relPaths []string) *exec.Cmd {
	dockerArgs := []string{"run", "--rm", "-v", gitRoot + ":/workdir", "-w", "/workdir", "ghcr.io/zizmorcore/zizmor:latest", "--format", "json"}
	dockerArgs = append(dockerArgs, relPaths...)
	// #nosec G204 -- exec.Command is used with separate args (not shell execution) to prevent shell injection.
	// The gitRoot path is validated to be absolute, and relPaths are validated through filepath.Rel.
	return exec.Command("docker", dockerArgs...)
}

func runZizmorOnFilesAnnounce(lockFiles, relPaths []string, gitRoot string, verbose bool) {
	if len(lockFiles) == 1 {
		fmt.Fprintf(os.Stderr, "%s\n", console.FormatInfoMessage("Running zizmor security scanner on "+relPaths[0]))
	} else {
		fmt.Fprintf(os.Stderr, "%s\n", console.FormatInfoMessage(fmt.Sprintf("Running zizmor security scanner on %d files", len(lockFiles))))
	}
	if verbose {
		dockerCmd := fmt.Sprintf("docker run --rm -v \"%s:/workdir\" -w /workdir ghcr.io/zizmorcore/zizmor:latest --format json %s", gitRoot, strings.Join(relPaths, " "))
		fmt.Fprintf(os.Stderr, "%s\n", console.FormatInfoMessage("Run zizmor directly: "+dockerCmd))
	}
}

func runZizmorOnFilesResult(err error, lockFiles []string, stdout, stderr bytes.Buffer, verbose bool, strict bool) error {
	// Parse and reformat the output, get total warning count
	totalWarnings, parseErr := parseAndDisplayZizmorOutput(stdout.String(), stderr.String(), verbose)
	if parseErr != nil {
		zizmorLog.Printf("Failed to parse zizmor output: %v", parseErr)
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
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode := exitErr.ExitCode()
			zizmorLog.Printf("Zizmor exited with code %d (warnings=%d)", exitCode, totalWarnings)
			// Exit codes 10-14 indicate findings
			if exitCode >= 10 && exitCode <= 14 {
				// In strict mode, findings are treated as errors
				if strict {
					fileDescription := "workflows"
					if len(lockFiles) == 1 {
						fileDescription = filepath.Base(lockFiles[0])
					}
					return fmt.Errorf("strict mode: zizmor found %d security warnings/errors in %s - workflows must have no zizmor findings in strict mode", totalWarnings, fileDescription)
				}
				// In non-strict mode, findings are logged but not treated as errors
				return nil
			}
			// Other exit codes are actual errors
			fileDescription := "workflows"
			if len(lockFiles) == 1 {
				fileDescription = filepath.Base(lockFiles[0])
			}
			return fmt.Errorf("zizmor failed with exit code %d on %s", exitCode, fileDescription)
		}
		// Non-ExitError errors (e.g., command not found)
		return fmt.Errorf("zizmor failed: %w", err)
	}

	return nil
}

// runZizmorOnFile runs the zizmor security scanner on a single .lock.yml file using Docker
// This is a wrapper around runZizmorOnFiles for backward compatibility
func runZizmorOnFile(lockFile string, verbose bool, strict bool) error {
	zizmorLog.Printf("Running zizmor security scanner: file=%s, strict=%v", lockFile, strict)
	return runZizmorOnFiles([]string{lockFile}, verbose, strict)
}

// parseAndDisplayZizmorOutput parses zizmor JSON output and displays it in the desired format
// Returns the total number of warnings found
func parseAndDisplayZizmorOutput(stdout, stderr string, verbose bool) (int, error) {
	// Map findings to files for detailed display
	fileFindings := make(map[string][]zizmorFinding)

	// Parse stderr for "completed" messages to get list of files
	completedFiles := parseAndDisplayZizmorOutputCompletedFiles(stderr, fileFindings)

	// Parse JSON findings from stdout
	totalWarnings, err := parseAndDisplayZizmorOutputFindings(stdout, fileFindings)
	if err != nil {
		return 0, err
	}

	// Display reformatted output for each completed file
	for _, filePath := range completedFiles {
		parseAndDisplayZizmorOutputFile(filePath, fileFindings[filePath])
	}

	return totalWarnings, nil
}

func parseAndDisplayZizmorOutputCompletedFiles(stderr string, fileFindings map[string][]zizmorFinding) []string {
	completedFiles := []string{}
	scanner := bufio.NewScanner(strings.NewReader(stderr))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "INFO audit: zizmor: 🌈 completed") {
			parts := strings.Split(line, "completed ")
			if len(parts) == 2 {
				filePath := strings.TrimSpace(parts[1])
				completedFiles = append(completedFiles, filePath)
				if _, exists := fileFindings[filePath]; !exists {
					fileFindings[filePath] = []zizmorFinding{}
				}
			}
		}
	}
	return completedFiles
}

func parseAndDisplayZizmorOutputFindings(stdout string, fileFindings map[string][]zizmorFinding) (int, error) {
	var findings []zizmorFinding
	totalWarnings := 0
	if stdout == "" || !strings.HasPrefix(strings.TrimSpace(stdout), "[") {
		return totalWarnings, nil
	}
	if err := json.Unmarshal([]byte(stdout), &findings); err != nil {
		return 0, fmt.Errorf("failed to parse zizmor JSON output: %w", err)
	}
	for _, finding := range findings {
		affectedFiles := make(map[string]struct{})
		for _, location := range finding.Locations {
			filePath := location.Symbolic.Key.Local.GivenPath
			if filePath != "" && !setutil.Contains(affectedFiles, filePath) {
				affectedFiles[filePath] = struct{}{}
				fileFindings[filePath] = append(fileFindings[filePath], finding)
				totalWarnings++
			}
		}
	}
	return totalWarnings, nil
}

func parseAndDisplayZizmorOutputFile(filePath string, findings []zizmorFinding) {
	if len(findings) == 0 {
		return
	}
	fileContent, err := os.ReadFile(filePath)
	var fileLines []string
	if err == nil {
		fileLines = strings.Split(string(fileContent), "\n")
	}
	for _, finding := range findings {
		parseAndDisplayZizmorOutputFinding(filePath, fileLines, finding)
	}
}

func parseAndDisplayZizmorOutputFinding(filePath string, fileLines []string, finding zizmorFinding) {
	if len(finding.Locations) == 0 {
		return
	}
	loc := finding.Locations[0]
	lineNum := loc.Concrete.Location.StartPoint.Row + 1
	colNum := loc.Concrete.Location.StartPoint.Column + 1
	context := parseAndDisplayZizmorOutputContext(fileLines, lineNum)
	errorType := "warning"
	if finding.Determinations.Severity == "High" || finding.Determinations.Severity == "Critical" {
		errorType = "error"
	}
	message := fmt.Sprintf("[%s] %s: %s", finding.Determinations.Severity, finding.Ident, finding.Desc)
	if finding.URL != "" {
		message = fmt.Sprintf("%s (%s)", message, finding.URL)
	}
	compilerErr := console.CompilerError{
		Position: console.ErrorPosition{File: filePath, Line: lineNum, Column: colNum},
		Type:     errorType,
		Message:  message,
		Context:  context,
	}
	fmt.Fprint(os.Stderr, console.FormatError(compilerErr))
}

func parseAndDisplayZizmorOutputContext(fileLines []string, lineNum int) []string {
	var context []string
	if len(fileLines) == 0 || lineNum <= 0 || lineNum > len(fileLines) {
		return context
	}
	startLine := max(1, lineNum-2)
	endLine := min(len(fileLines), lineNum+2)
	for i := startLine; i <= endLine; i++ {
		if i-1 < len(fileLines) {
			context = append(context, fileLines[i-1])
		}
	}
	return context
}
