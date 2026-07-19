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

	"github.com/github/gh-aw/pkg/constants"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/fileutil"
	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/logger"
)

var poutineLog = logger.New("cli:poutine")

// poutineFinding represents a single finding from poutine JSON output
type poutineFinding struct {
	RuleID string `json:"rule_id"`
	Purl   string `json:"purl"`
	Meta   struct {
		Path    string `json:"path"`
		Line    int    `json:"line"`
		Details string `json:"details"`
	} `json:"meta"`
}

// poutineOutput represents the complete JSON output from poutine
type poutineOutput struct {
	Findings []poutineFinding `json:"findings"`
	Rules    map[string]struct {
		ID          string `json:"id"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Level       string `json:"level"` // error, warning, note
	} `json:"rules"`
}

// ensurePoutineConfig creates .poutine.yml to configure allowed runners if it doesn't exist
func ensurePoutineConfig(gitRoot string) error {
	configPath := filepath.Join(gitRoot, ".poutine.yml")

	// Check if config already exists
	if fileutil.FileExists(configPath) {
		// Config exists, do not update it
		poutineLog.Print(".poutine.yml already exists, skipping creation")
		return nil
	}

	// Create the config file
	configContent := `# Configure poutine security scanner
# See: https://github.com/boostsecurityio/poutine

# Set rule configuration options
rulesConfig:
  pr_runs_on_self_hosted:
    allowed_runners:
      - ubuntu-slim  # GitHub's new built-in runner (not self-hosted)
`

	// Write the config file
	if err := os.WriteFile(configPath, []byte(configContent), constants.FilePermPublic); err != nil {
		return fmt.Errorf("failed to write .poutine.yml: %w", err)
	}

	poutineLog.Printf("Created .poutine.yml at %s", configPath)
	return nil
}

// runPoutineOnDirectory runs the poutine security scanner on a directory containing workflows
func runPoutineOnDirectory(workflowDir string, verbose bool, strict bool) error {
	poutineLog.Printf("Running poutine security scanner on directory: %s", workflowDir)

	gitRoot, cmd, err := runPoutineOnDirectoryCommand()
	if err != nil {
		return err
	}

	// Always show that poutine is running (regular verbosity)
	fmt.Fprintf(os.Stderr, "%s\n", console.FormatInfoMessage("Running poutine security scanner"))

	// In verbose mode, also show the command that users can run directly
	runPoutineOnDirectoryVerboseCommand(verbose, gitRoot)

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run the command
	err = cmd.Run()

	// Parse and display output for all files (no filtering)
	totalWarnings, parseErr := parseAndDisplayPoutineOutputForDirectory(stdout.String(), verbose, gitRoot)
	if parseErr != nil {
		poutineLog.Printf("Failed to parse poutine output: %v", parseErr)
		// Fall back to showing raw output
		if stdout.Len() > 0 {
			fmt.Fprint(os.Stderr, stdout.String())
		}
		if stderr.Len() > 0 {
			fmt.Fprint(os.Stderr, stderr.String())
		}
	}

	// Check if the error is due to findings or actual failure
	if err := runPoutineOnDirectoryHandleError(err, totalWarnings, strict); err != nil {
		return err
	}

	return nil
}

func runPoutineOnDirectoryCommand() (string, *exec.Cmd, error) {
	gitRoot, err := gitutil.FindGitRoot()
	if err != nil {
		return "", nil, fmt.Errorf("failed to find git root: %w", err)
	}
	if !filepath.IsAbs(gitRoot) {
		return "", nil, fmt.Errorf("git root is not an absolute path: %s", gitRoot)
	}
	if err := ensurePoutineConfig(gitRoot); err != nil {
		return "", nil, fmt.Errorf("failed to ensure poutine config: %w", err)
	}
	// #nosec G204 -- gitRoot comes from git rev-parse (trusted source) and is validated as absolute path
	cmd := exec.Command("docker", "run", "--rm", "-v", gitRoot+":/workdir", "-w", "/workdir", "ghcr.io/boostsecurityio/poutine:latest", "analyze_local", ".", "--format", "json", "--quiet")
	return gitRoot, cmd, nil
}

func runPoutineOnDirectoryVerboseCommand(verbose bool, gitRoot string) {
	if !verbose {
		return
	}
	dockerCmd := fmt.Sprintf("docker run --rm -v \"%s:/workdir\" -w /workdir ghcr.io/boostsecurityio/poutine:latest analyze_local . --format json --quiet", gitRoot)
	fmt.Fprintf(os.Stderr, "%s\n", console.FormatInfoMessage("Run poutine directly: "+dockerCmd))
}

func runPoutineOnDirectoryHandleError(err error, totalWarnings int, strict bool) error {
	if err == nil {
		return nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		exitCode := exitErr.ExitCode()
		poutineLog.Printf("Poutine exited with code %d (warnings=%d)", exitCode, totalWarnings)
		if exitCode == 1 {
			if strict && totalWarnings > 0 {
				return fmt.Errorf("strict mode: poutine found %d security warnings/errors - workflows must have no poutine findings in strict mode", totalWarnings)
			}
			return nil
		}
		return fmt.Errorf("poutine failed with exit code %d", exitCode)
	}
	return fmt.Errorf("poutine failed: %w", err)
}

// runPoutineOnFile runs the poutine security scanner on a single .lock.yml file using Docker
// This is a wrapper that filters the directory scan results to a single file for backward compatibility
func runPoutineOnFile(lockFile string, verbose bool, strict bool) error {
	poutineLog.Printf("Running poutine security scanner: file=%s, strict=%v", lockFile, strict)

	gitRoot, err := runPoutineOnFileGitRoot()
	if err != nil {
		return err
	}
	// Get the relative path from git root
	relPath, err := filepath.Rel(gitRoot, lockFile)
	if err != nil {
		return fmt.Errorf("failed to get relative path: %w", err)
	}

	cmd := runPoutineOnFileCommand(gitRoot)

	// Always show that poutine is running (regular verbosity)
	fmt.Fprintf(os.Stderr, "%s\n", console.FormatInfoMessage("Running poutine security scanner"))

	// In verbose mode, also show the command that users can run directly
	runPoutineOnFileVerboseCommand(verbose, gitRoot)

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run the command
	err = cmd.Run()

	// Parse and reformat the output, get total warning count
	totalWarnings, parseErr := parseAndDisplayPoutineOutput(stdout.String(), relPath, verbose)
	if parseErr != nil {
		poutineLog.Printf("Failed to parse poutine output: %v", parseErr)
		// Fall back to showing raw output
		if stdout.Len() > 0 {
			fmt.Fprint(os.Stderr, stdout.String())
		}
		if stderr.Len() > 0 {
			fmt.Fprint(os.Stderr, stderr.String())
		}
	}

	// Check if the error is due to findings or actual failure
	if err := runPoutineOnFileHandleError(err, totalWarnings, strict, lockFile); err != nil {
		return err
	}

	return nil
}

func runPoutineOnFileGitRoot() (string, error) {
	gitRoot, err := gitutil.FindGitRoot()
	if err != nil {
		return "", fmt.Errorf("failed to find git root: %w", err)
	}
	if !filepath.IsAbs(gitRoot) {
		return "", fmt.Errorf("git root is not an absolute path: %s", gitRoot)
	}
	if err := ensurePoutineConfig(gitRoot); err != nil {
		return "", fmt.Errorf("failed to ensure poutine config: %w", err)
	}
	return gitRoot, nil
}

func runPoutineOnFileCommand(gitRoot string) *exec.Cmd {
	// #nosec G204 -- gitRoot comes from git rev-parse (trusted source) and is validated as absolute path
	return exec.Command("docker", "run", "--rm", "-v", gitRoot+":/workdir", "-w", "/workdir", "ghcr.io/boostsecurityio/poutine:latest", "analyze_local", ".", "--format", "json", "--quiet")
}

func runPoutineOnFileVerboseCommand(verbose bool, gitRoot string) {
	if !verbose {
		return
	}
	dockerCmd := fmt.Sprintf("docker run --rm -v \"%s:/workdir\" -w /workdir ghcr.io/boostsecurityio/poutine:latest analyze_local . --format json --quiet", gitRoot)
	fmt.Fprintf(os.Stderr, "%s\n", console.FormatInfoMessage("Run poutine directly: "+dockerCmd))
}

func runPoutineOnFileHandleError(err error, totalWarnings int, strict bool, lockFile string) error {
	if err == nil {
		return nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		exitCode := exitErr.ExitCode()
		poutineLog.Printf("Poutine exited with code %d (warnings=%d)", exitCode, totalWarnings)
		if exitCode == 1 {
			if strict && totalWarnings > 0 {
				return fmt.Errorf("strict mode: poutine found %d security warnings/errors in %s - workflows must have no poutine findings in strict mode", totalWarnings, filepath.Base(lockFile))
			}
			return nil
		}
		return fmt.Errorf("poutine failed with exit code %d on %s", exitCode, filepath.Base(lockFile))
	}
	return fmt.Errorf("poutine failed on %s: %w", filepath.Base(lockFile), err)
}

// parseAndDisplayPoutineOutput parses poutine JSON output and displays it in the desired format
// Returns the total number of warnings found for the specific file
func parseAndDisplayPoutineOutput(stdout, targetFile string, verbose bool) (int, error) {
	// Parse JSON output from stdout
	output, ok, err := parseAndDisplayPoutineOutputParse(stdout)
	if err != nil || !ok {
		return 0, err
	}

	// Filter findings to only those relevant to the target file
	relevantFindings := parseAndDisplayPoutineOutputRelevant(output.Findings, targetFile)
	totalWarnings := len(relevantFindings)

	// Skip files with 0 warnings
	if totalWarnings == 0 {
		return 0, nil
	}

	// Read file content for context display
	fileLines := parseAndDisplayPoutineOutputFileLines(targetFile)

	// Display detailed findings using CompilerError format
	for _, finding := range relevantFindings {
		parseAndDisplayPoutineOutputFinding(finding, output.Rules, fileLines)
	}

	return totalWarnings, nil
}

func parseAndDisplayPoutineOutputParse(stdout string) (poutineOutput, bool, error) {
	var output poutineOutput
	if stdout == "" {
		return output, false, nil
	}
	trimmed := strings.TrimSpace(stdout)
	if !strings.HasPrefix(trimmed, "{") {
		if trimmed != "" {
			return output, false, fmt.Errorf("unexpected poutine output format: %s", trimmed)
		}
		return output, false, nil
	}
	if err := json.Unmarshal([]byte(stdout), &output); err != nil {
		return output, false, fmt.Errorf("failed to parse poutine JSON output: %w", err)
	}
	return output, true, nil
}

func parseAndDisplayPoutineOutputRelevant(findings []poutineFinding, targetFile string) []poutineFinding {
	var relevantFindings []poutineFinding
	for _, finding := range findings {
		if finding.Meta.Path == targetFile {
			relevantFindings = append(relevantFindings, finding)
		}
	}
	return relevantFindings
}

func parseAndDisplayPoutineOutputFileLines(targetFile string) []string {
	fileContent, err := os.ReadFile(targetFile)
	if err != nil {
		return nil
	}
	return strings.Split(string(fileContent), "\n")
}

func parseAndDisplayPoutineOutputFinding(finding poutineFinding, rules map[string]struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Level       string `json:"level"`
}, fileLines []string) {
	ruleInfo := rules[finding.RuleID]
	severity, title := parseAndDisplayPoutineOutputRule(finding, ruleInfo.Level, ruleInfo.Title)
	lineNum := parseAndDisplayPoutineOutputLine(finding)
	context := parseAndDisplayPoutineOutputContext(fileLines, lineNum)
	compilerErr := console.CompilerError{
		Position: console.ErrorPosition{File: finding.Meta.Path, Line: lineNum, Column: 1},
		Type:     parseAndDisplayPoutineOutputErrorType(severity),
		Message:  parseAndDisplayPoutineOutputMessage(finding, severity, title),
		Context:  context,
	}
	fmt.Fprint(os.Stderr, console.FormatError(compilerErr))
}

func parseAndDisplayPoutineOutputRule(finding poutineFinding, severity string, title string) (string, string) {
	if severity == "" {
		severity = "warning"
	}
	if title == "" {
		title = finding.RuleID
	}
	return severity, title
}

func parseAndDisplayPoutineOutputLine(finding poutineFinding) int {
	if finding.Meta.Line == 0 {
		return 1
	}
	return finding.Meta.Line
}

func parseAndDisplayPoutineOutputContext(fileLines []string, lineNum int) []string {
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
	return context
}

func parseAndDisplayPoutineOutputErrorType(severity string) string {
	switch severity {
	case "error":
		return "error"
	case "note":
		return "info"
	default:
		return "warning"
	}
}

func parseAndDisplayPoutineOutputMessage(finding poutineFinding, severity string, title string) string {
	message := fmt.Sprintf("[%s] %s: %s", severity, finding.RuleID, title)
	if finding.Meta.Details != "" {
		message = fmt.Sprintf("%s - %s", message, finding.Meta.Details)
	}
	return message
}

// parseAndDisplayPoutineOutputForDirectory parses poutine JSON output and displays all findings
// Returns the total number of warnings found across all files
func parseAndDisplayPoutineOutputForDirectory(stdout string, verbose bool, gitRoot string) (int, error) {
	// Parse JSON output from stdout
	output, ok, err := parseAndDisplayPoutineOutputParse(stdout)
	if err != nil || !ok {
		return 0, err
	}

	// Display all findings (no filtering by file)
	totalWarnings := len(output.Findings)

	// Skip if no warnings
	if totalWarnings == 0 {
		return 0, nil
	}

	// Group findings by file for better readability
	findingsByFile := parseAndDisplayPoutineOutputForDirectoryByFile(output.Findings)

	// Display findings for each file
	for filePath, findings := range findingsByFile {
		parseAndDisplayPoutineOutputForDirectoryFile(filePath, findings, output.Rules, gitRoot)
	}

	return totalWarnings, nil
}

func parseAndDisplayPoutineOutputForDirectoryByFile(findings []poutineFinding) map[string][]poutineFinding {
	findingsByFile := make(map[string][]poutineFinding)
	for _, finding := range findings {
		findingsByFile[finding.Meta.Path] = append(findingsByFile[finding.Meta.Path], finding)
	}
	return findingsByFile
}

func parseAndDisplayPoutineOutputForDirectoryFile(filePath string, findings []poutineFinding, rules map[string]struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Level       string `json:"level"`
}, gitRoot string) {
	absPath, ok := parseAndDisplayPoutineOutputForDirectoryPath(filePath, gitRoot)
	if !ok {
		return
	}
	fileLines := parseAndDisplayPoutineOutputForDirectoryFileLines(absPath)
	for _, finding := range findings {
		parseAndDisplayPoutineOutputForDirectoryFinding(finding, rules, fileLines)
	}
}

func parseAndDisplayPoutineOutputForDirectoryPath(filePath string, gitRoot string) (string, bool) {
	cleanPath := filepath.Clean(filePath)
	absPath := cleanPath
	if !filepath.IsAbs(cleanPath) {
		absPath = filepath.Join(gitRoot, cleanPath)
	}
	absGitRoot, err := filepath.Abs(gitRoot)
	if err != nil {
		poutineLog.Printf("Failed to get absolute path for git root: %v", err)
		return "", false
	}
	absPath, err = filepath.Abs(absPath)
	if err != nil {
		poutineLog.Printf("Failed to get absolute path for %s: %v", filePath, err)
		return "", false
	}
	relPath, err := filepath.Rel(absGitRoot, absPath)
	if err != nil || strings.HasPrefix(relPath, "..") {
		poutineLog.Printf("Skipping file outside git root: %s", filePath)
		return "", false
	}
	return absPath, true
}

func parseAndDisplayPoutineOutputForDirectoryFileLines(absPath string) []string {
	// #nosec G304 -- absPath is validated through filepath.Clean(), absolute path resolution, and filepath.Rel() boundary checks.
	fileContent, err := os.ReadFile(absPath)
	if err != nil {
		return nil
	}
	return strings.Split(string(fileContent), "\n")
}

func parseAndDisplayPoutineOutputForDirectoryFinding(finding poutineFinding, rules map[string]struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Level       string `json:"level"`
}, fileLines []string) {
	ruleInfo := rules[finding.RuleID]
	severity, title := parseAndDisplayPoutineOutputRule(finding, ruleInfo.Level, ruleInfo.Title)
	lineNum := parseAndDisplayPoutineOutputLine(finding)
	compilerErr := console.CompilerError{
		Position: console.ErrorPosition{File: finding.Meta.Path, Line: lineNum, Column: 1},
		Type:     parseAndDisplayPoutineOutputErrorType(severity),
		Message:  parseAndDisplayPoutineOutputMessage(finding, severity, title),
		Context:  parseAndDisplayPoutineOutputContext(fileLines, lineNum),
	}
	fmt.Fprint(os.Stderr, console.FormatError(compilerErr))
}
