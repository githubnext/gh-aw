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
	gitRoot, scanPath, err := runRunnerGuardOnDirectoryPaths(workflowDir)
	if err != nil {
		return err
	}
	cmd := runRunnerGuardOnDirectoryCommand(gitRoot, scanPath)
	runRunnerGuardOnDirectoryPrint(gitRoot, scanPath, verbose)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	totalFindings, parseErr := parseAndDisplayRunnerGuardOutput(stdout.String(), verbose, gitRoot)
	if parseErr != nil {
		runRunnerGuardOnDirectoryRawOutput(stdout, stderr, parseErr)
	}
	return runRunnerGuardOnDirectoryHandleError(err, parseErr, totalFindings, strict)
}

func runRunnerGuardOnDirectoryPaths(workflowDir string) (string, string, error) {
	gitRoot, err := gitutil.FindGitRoot()
	if err != nil {
		return "", "", fmt.Errorf("failed to find git root: %w", err)
	}
	if !filepath.IsAbs(gitRoot) {
		return "", "", fmt.Errorf("git root is not an absolute path: %s", gitRoot)
	}
	scanPath := "."
	if workflowDir != "" {
		relDir, relErr := filepath.Rel(gitRoot, workflowDir)
		if relErr == nil && relDir != ".." && !strings.HasPrefix(relDir, ".."+string(filepath.Separator)) {
			scanPath = relDir
		}
	}
	return gitRoot, scanPath, nil
}

func runRunnerGuardOnDirectoryCommand(gitRoot, scanPath string) *exec.Cmd {
	return exec.Command("docker", "run", "--rm", "-v", gitRoot+":/workdir", "-w", "/workdir", RunnerGuardImage, "scan", scanPath, "--format", "json")
}

func runRunnerGuardOnDirectoryPrint(gitRoot, scanPath string, verbose bool) {
	fmt.Fprintf(os.Stderr, "%s\n", console.FormatInfoMessage("Running runner-guard taint analysis scanner"))
	if verbose {
		dockerCmd := fmt.Sprintf("docker run --rm -v \"%s:/workdir\" -w /workdir %s scan %s --format json", gitRoot, RunnerGuardImage, scanPath)
		fmt.Fprintf(os.Stderr, "%s\n", console.FormatInfoMessage("Run runner-guard directly: "+dockerCmd))
	}
}

func runRunnerGuardOnDirectoryRawOutput(stdout, stderr bytes.Buffer, parseErr error) {
	runnerGuardLog.Printf("Failed to parse runner-guard output: %v", parseErr)
	if stdout.Len() > 0 {
		fmt.Fprint(os.Stderr, stdout.String())
	}
	if stderr.Len() > 0 {
		fmt.Fprint(os.Stderr, stderr.String())
	}
}

func runRunnerGuardOnDirectoryHandleError(err error, parseErr error, totalFindings int, strict bool) error {
	if err == nil {
		return nil
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return fmt.Errorf("runner-guard failed: %w", err)
	}
	exitCode := exitErr.ExitCode()
	runnerGuardLog.Printf("runner-guard exited with code %d (findings=%d)", exitCode, totalFindings)
	if exitCode != 1 {
		return fmt.Errorf("runner-guard failed with exit code %d", exitCode)
	}
	if !strict {
		return nil
	}
	if parseErr != nil {
		return fmt.Errorf("strict mode: runner-guard exited with code 1 (findings present) and output could not be parsed: %w", parseErr)
	}
	if totalFindings > 0 {
		return fmt.Errorf("strict mode: runner-guard found %d security findings - workflows must have no runner-guard findings in strict mode", totalFindings)
	}
	return errors.New("strict mode: runner-guard exited with code 1 indicating findings are present")
}

// parseAndDisplayRunnerGuardOutput parses runner-guard JSON output and displays findings.
// Returns the total number of findings found.
func parseAndDisplayRunnerGuardOutput(stdout string, verbose bool, gitRoot string) (int, error) {
	if stdout == "" {
		return 0, nil
	}
	trimmed := strings.TrimSpace(stdout)
	if !strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "[") {
		if trimmed != "" {
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
	parseAndDisplayRunnerGuardOutputScore(output)
	for filePath, findings := range parseAndDisplayRunnerGuardOutputByFile(output.Findings) {
		parseAndDisplayRunnerGuardOutputFile(filePath, findings, gitRoot)
	}
	return totalFindings, nil
}

func parseAndDisplayRunnerGuardOutputScore(output runnerGuardOutput) {
	if output.Score > 0 || output.Grade != "" {
		fmt.Fprintf(os.Stderr, "%s\n", console.FormatInfoMessage(fmt.Sprintf("Runner-Guard Score: %d/100 (Grade: %s)", output.Score, output.Grade)))
	}
}

func parseAndDisplayRunnerGuardOutputByFile(findings []runnerGuardFinding) map[string][]runnerGuardFinding {
	findingsByFile := make(map[string][]runnerGuardFinding)
	for _, finding := range findings {
		findingsByFile[finding.File] = append(findingsByFile[finding.File], finding)
	}
	return findingsByFile
}

func parseAndDisplayRunnerGuardOutputFile(filePath string, findings []runnerGuardFinding, gitRoot string) {
	absPath, ok := parseAndDisplayRunnerGuardOutputSafePath(filePath, gitRoot)
	if !ok {
		return
	}
	// #nosec G304 -- absPath is validated to be within gitRoot.
	fileContent, err := os.ReadFile(absPath)
	var fileLines []string
	if err == nil {
		fileLines = strings.Split(string(fileContent), "\n")
	}
	for _, finding := range findings {
		parseAndDisplayRunnerGuardOutputFinding(finding, fileLines)
	}
}

func parseAndDisplayRunnerGuardOutputSafePath(filePath, gitRoot string) (string, bool) {
	cleanPath := filepath.Clean(filePath)
	absPath := cleanPath
	if !filepath.IsAbs(cleanPath) {
		absPath = filepath.Join(gitRoot, cleanPath)
	}
	absGitRoot, err := filepath.Abs(gitRoot)
	if err != nil {
		runnerGuardLog.Printf("Failed to get absolute path for git root: %v", err)
		return "", false
	}
	absPath, err = filepath.Abs(absPath)
	if err != nil {
		runnerGuardLog.Printf("Failed to get absolute path for %s: %v", filePath, err)
		return "", false
	}
	relPath, err := filepath.Rel(absGitRoot, absPath)
	if err != nil || relPath == ".." || strings.HasPrefix(relPath, ".."+string(filepath.Separator)) {
		runnerGuardLog.Printf("Skipping file outside git root: %s", filePath)
		return "", false
	}
	return absPath, true
}

func parseAndDisplayRunnerGuardOutputFinding(finding runnerGuardFinding, fileLines []string) {
	lineNum := finding.Line
	if lineNum == 0 {
		lineNum = 1
	}
	compilerErr := console.CompilerError{
		Position: console.ErrorPosition{File: finding.File, Line: lineNum, Column: 1},
		Type:     parseAndDisplayRunnerGuardOutputSeverity(finding.Severity),
		Message:  parseAndDisplayRunnerGuardOutputMessage(finding),
		Context:  parseAndDisplayRunnerGuardOutputContext(fileLines, lineNum),
	}
	fmt.Fprint(os.Stderr, console.FormatError(compilerErr))
}

func parseAndDisplayRunnerGuardOutputContext(fileLines []string, lineNum int) []string {
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

func parseAndDisplayRunnerGuardOutputSeverity(severity string) string {
	switch strings.ToLower(severity) {
	case "critical", "high", "error":
		return "error"
	case "note", "info":
		return "info"
	default:
		return "warning"
	}
}

func parseAndDisplayRunnerGuardOutputMessage(finding runnerGuardFinding) string {
	message := fmt.Sprintf("[%s] %s: %s", finding.Severity, finding.RuleID, finding.Name)
	if finding.Description != "" {
		message = fmt.Sprintf("%s - %s", message, finding.Description)
	}
	return message
}
