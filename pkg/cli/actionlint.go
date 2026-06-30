package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/logger"
)

var actionlintLog = logger.New("cli:actionlint")

// actionlintVersion caches the actionlint version to avoid repeated Docker calls
var actionlintVersion string

// actionlintRunOptions configures optional actionlint integrations and ignores.
type actionlintRunOptions struct {
	IncludeShellcheck bool
	IncludePyflakes   bool
	// IgnorePatterns contains regular expressions passed to actionlint via
	// repeated -ignore flags to suppress known false positives.
	IgnorePatterns []string
}

// buildActionlintIntegrationStatus returns a human-readable description of the
// shellcheck/pyflakes integration state for actionlint execution messages.
func buildActionlintIntegrationStatus(includeShellcheck bool, includePyflakes bool) string {
	switch {
	case includeShellcheck && includePyflakes:
		return "with shellcheck/pyflakes"
	case includeShellcheck:
		return "with shellcheck only"
	case includePyflakes:
		return "with pyflakes only"
	default:
		return "without shellcheck/pyflakes"
	}
}

// getActionlintDocsURL returns the documentation URL for a given actionlint error kind
// Error kinds map to documentation anchors at https://github.com/rhysd/actionlint/blob/main/docs/checks.md
func getActionlintDocsURL(kind string) string {
	if kind == "" {
		return "https://github.com/rhysd/actionlint/blob/main/docs/checks.md"
	}

	// Map error kind to documentation anchor
	// Most kinds follow the pattern "check-{kind}" as the anchor
	anchor := kind

	// Special case mappings for kinds that don't follow the standard pattern
	switch kind {
	case "runner-label":
		anchor = "check-runner-labels"
	case "pyflakes":
		anchor = "check-pyflakes-integ"
	case "shellcheck":
		anchor = "check-shellcheck-integ"
	case "expression":
		anchor = "check-syntax-expression"
	case "syntax-check":
		anchor = "check-unexpected-keys"
	default:
		// For other kinds, try the standard "check-{kind}" pattern
		if !strings.HasPrefix(anchor, "check-") {
			anchor = "check-" + anchor
		}
	}

	return "https://github.com/rhysd/actionlint/blob/main/docs/checks.md#" + anchor
}

// actionlintStats tracks aggregate statistics across all actionlint validations
var actionlintStats *ActionlintStats

// ActionlintStats tracks actionlint validation statistics across all files
type ActionlintStats struct {
	TotalWorkflows    int
	TotalErrors       int
	TotalWarnings     int
	IntegrationErrors int // counts tooling/subprocess failures, not lint findings
	ErrorsByKind      map[string]int
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
	} else if actionlintStats.IntegrationErrors > 0 {
		// Integration failures occurred but no lint issues were parsed.
		// Explicitly distinguish this from a clean run so users are not misled.
		msg := fmt.Sprintf("No lint issues found, but %d actionlint invocation(s) failed. "+
			"This likely indicates a tooling or integration error, not a workflow problem.",
			actionlintStats.IntegrationErrors)
		fmt.Fprintf(os.Stderr, "%s\n", console.FormatWarningMessage(msg))
	} else {
		fmt.Fprintf(os.Stderr, "%s\n",
			console.FormatSuccessMessage("No issues found"))
	}

	// Report any integration failures alongside lint findings
	if totalIssues > 0 && actionlintStats.IntegrationErrors > 0 {
		msg := fmt.Sprintf("%d actionlint invocation(s) also failed with tooling errors (not workflow validation failures)",
			actionlintStats.IntegrationErrors)
		fmt.Fprintf(os.Stderr, "\n%s\n", console.FormatWarningMessage(msg))
	}

	fmt.Fprintf(os.Stderr, "\n%s\n", separator)
}

// getActionlintVersion fetches and caches the actionlint version from Docker.
// The provided context allows caller-driven cancellation.
func getActionlintVersion(ctx context.Context) (string, error) {
	// Return cached version if already fetched
	if actionlintVersion != "" {
		return actionlintVersion, nil
	}

	actionlintLog.Print("Fetching actionlint version from Docker")

	// Run docker command to get version with a 30 second timeout
	versionCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(
		versionCtx,
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
		return "", errors.New("no version output from actionlint")
	}
	version := strings.TrimSpace(lines[0])
	actionlintVersion = version
	actionlintLog.Printf("Cached actionlint version: %s", version)

	return version, nil
}

// runActionlintOnFiles runs the actionlint linter on one or more .lock.yml files using Docker.
// The provided context allows caller-driven cancellation.
func runActionlintOnFiles(ctx context.Context, lockFiles []string, verbose bool, strict bool) error {
	return runActionlintOnFilesWithOptions(ctx, lockFiles, verbose, strict, actionlintRunOptions{
		IncludeShellcheck: true,
		IncludePyflakes:   true,
	})
}

func runActionlintOnFilesWithOptions(ctx context.Context, lockFiles []string, verbose bool, strict bool, options actionlintRunOptions) error {
	if len(lockFiles) == 0 {
		return nil
	}
	actionlintLog.Printf("Running actionlint on %d file(s): %v (verbose=%t, strict=%t)", len(lockFiles), lockFiles, verbose, strict)

	if actionlintVersion == "" {
		version, err := getActionlintVersion(ctx)
		if err != nil {
			actionlintLog.Printf("Could not fetch actionlint version: %v", err)
		} else {
			fmt.Fprintf(os.Stderr, "%s\n", console.FormatInfoMessage("Using actionlint "+version))
		}
	}

	gitRoot, err := gitutil.FindGitRoot()
	if err != nil {
		return fmt.Errorf("failed to find git root: %w", err)
	}

	relPaths, err := getActionlintRelPaths(gitRoot, lockFiles)
	if err != nil {
		return err
	}

	timeoutDuration := time.Duration(max(5, len(lockFiles))) * time.Minute
	runCtx, cancel := context.WithTimeout(ctx, timeoutDuration)
	defer cancel()

	dockerArgs := buildActionlintDockerArgs(gitRoot, relPaths, options)
	cmd := exec.CommandContext(runCtx, "docker", dockerArgs...)

	logActionlintRunStatus(verbose, gitRoot, relPaths, lockFiles, options)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
		fileList := "files"
		if len(lockFiles) == 1 {
			fileList = filepath.Base(lockFiles[0])
		}
		if actionlintStats != nil {
			actionlintStats.IntegrationErrors++
		}
		return fmt.Errorf("actionlint timed out after %d minutes on %s - this may indicate a Docker or network issue", int(timeoutDuration.Minutes()), fileList)
	}
	if errors.Is(runCtx.Err(), context.Canceled) {
		return errors.New("actionlint was canceled before completion (for example by Ctrl+C or caller cancellation)")
	}

	if actionlintStats != nil {
		actionlintStats.TotalWorkflows += len(lockFiles)
	}

	totalErrors, errorsByKind, parseErr := parseAndDisplayActionlintOutput(stdout.String(), verbose)
	trackActionlintParseResult(parseErr, totalErrors, errorsByKind, &stdout, &stderr)

	return handleActionlintCommandError(err, strict, lockFiles, totalErrors, parseErr)
}

// getActionlintRelPaths returns relative paths from gitRoot for each lock file.
func getActionlintRelPaths(gitRoot string, lockFiles []string) ([]string, error) {
	var relPaths []string
	for _, lockFile := range lockFiles {
		relPath, err := filepath.Rel(gitRoot, lockFile)
		if err != nil {
			return nil, fmt.Errorf("failed to get relative path for %s: %w", lockFile, err)
		}
		relPaths = append(relPaths, relPath)
	}
	return relPaths, nil
}

// buildActionlintDockerArgs assembles the docker run arguments for actionlint.
func buildActionlintDockerArgs(gitRoot string, relPaths []string, options actionlintRunOptions) []string {
	dockerArgs := []string{
		"run",
		"--rm",
		"-v", gitRoot + ":/workdir",
		"-w", "/workdir",
		"rhysd/actionlint:latest",
		"-format", "{{json .}}",
	}
	if !options.IncludeShellcheck {
		dockerArgs = append(dockerArgs, "-shellcheck=")
	}
	if !options.IncludePyflakes {
		dockerArgs = append(dockerArgs, "-pyflakes=")
	}
	for _, ignorePattern := range options.IgnorePatterns {
		dockerArgs = append(dockerArgs, "-ignore", ignorePattern)
	}
	return append(dockerArgs, relPaths...)
}

// logActionlintRunStatus prints run status and (in verbose mode) the full docker command.
func logActionlintRunStatus(verbose bool, gitRoot string, relPaths []string, lockFiles []string, options actionlintRunOptions) {
	integrationStatus := buildActionlintIntegrationStatus(options.IncludeShellcheck, options.IncludePyflakes)
	if len(lockFiles) == 1 {
		fmt.Fprintf(os.Stderr, "%s\n", console.FormatInfoMessage("Running actionlint ("+integrationStatus+") on "+relPaths[0]))
	} else {
		fmt.Fprintf(os.Stderr, "%s\n", console.FormatInfoMessage(fmt.Sprintf("Running actionlint (%s) on %d files", integrationStatus, len(lockFiles))))
	}
	if verbose {
		dockerCmd := fmt.Sprintf("docker run --rm -v \"%s:/workdir\" -w /workdir rhysd/actionlint:latest -format '{{json .}}' %s",
			gitRoot, strings.Join(relPaths, " "))
		fmt.Fprintf(os.Stderr, "%s\n", console.FormatInfoMessage("Run actionlint directly: "+dockerCmd))
	}
}

// trackActionlintParseResult updates stats and falls back to raw output when parse fails.
func trackActionlintParseResult(parseErr error, totalErrors int, errorsByKind map[string]int, stdout *bytes.Buffer, stderr *bytes.Buffer) {
	if parseErr != nil {
		actionlintLog.Printf("Failed to parse actionlint output: %v", parseErr)
		if actionlintStats != nil {
			actionlintStats.IntegrationErrors++
		}
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(
			"actionlint output could not be parsed — this is a tooling error, not a workflow validation failure: "+parseErr.Error()))
		if stdout.Len() > 0 {
			fmt.Fprint(os.Stderr, stdout.String())
		}
		if stderr.Len() > 0 {
			fmt.Fprint(os.Stderr, stderr.String())
		}
		return
	}
	if actionlintStats != nil {
		actionlintStats.TotalErrors += totalErrors
		for kind, count := range errorsByKind {
			actionlintStats.ErrorsByKind[kind] += count
		}
	}
}

// handleActionlintCommandError translates a command execution error into a caller error
// based on the actionlint exit code, strict mode setting, and parse outcome.
func handleActionlintCommandError(err error, strict bool, lockFiles []string, totalErrors int, parseErr error) error {
	if err == nil {
		return nil
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		if actionlintStats != nil {
			actionlintStats.IntegrationErrors++
		}
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(
			"actionlint could not be invoked — this is a tooling error, not a workflow validation failure: "+err.Error()))
		return fmt.Errorf("actionlint failed: %w", err)
	}
	exitCode := exitErr.ExitCode()
	actionlintLog.Printf("Actionlint exited with code %d, found %d errors", exitCode, totalErrors)
	if exitCode == 1 {
		if strict {
			fileDescription := "workflows"
			if len(lockFiles) == 1 {
				fileDescription = filepath.Base(lockFiles[0])
			}
			if parseErr != nil {
				return fmt.Errorf("strict mode: actionlint exited with errors on %s but output could not be parsed — this is likely a tooling or integration error", fileDescription)
			}
			return fmt.Errorf("strict mode: actionlint found %d errors in %s - workflows must have no actionlint errors in strict mode", totalErrors, fileDescription)
		}
		return nil
	}
	fileDescription := "workflows"
	if len(lockFiles) == 1 {
		fileDescription = filepath.Base(lockFiles[0])
	}
	if actionlintStats != nil {
		actionlintStats.IntegrationErrors++
	}
	fmt.Fprintln(os.Stderr, console.FormatWarningMessage(
		fmt.Sprintf("actionlint failed with exit code %d on %s — this is a tooling error, not a workflow validation failure", exitCode, fileDescription)))
	return fmt.Errorf("actionlint failed with exit code %d on %s", exitCode, fileDescription)
}

// parseAndDisplayActionlintOutput parses actionlint JSON output and displays it in the desired format
// Returns the total number of errors found and a breakdown by kind
func parseAndDisplayActionlintOutput(stdout string, verbose bool) (int, map[string]int, error) {
	if stdout == "" || strings.TrimSpace(stdout) == "" {
		actionlintLog.Print("No actionlint output to parse")
		return 0, make(map[string]int), nil
	}

	var errs []actionlintError
	if err := json.Unmarshal([]byte(stdout), &errs); err != nil {
		return 0, nil, fmt.Errorf("failed to parse actionlint JSON output: %w", err)
	}

	totalErrors := len(errs)
	actionlintLog.Printf("Parsed %d actionlint errors from output", totalErrors)

	errorsByKind := make(map[string]int)
	for _, err := range errs {
		if err.Kind != "" {
			errorsByKind[err.Kind]++
		}
		fmt.Fprint(os.Stderr, formatActionlintError(err))
	}

	return totalErrors, errorsByKind, nil
}

// formatActionlintError formats a single actionlint error as a console CompilerError string.
func formatActionlintError(err actionlintError) string {
	var context []string
	if err.Snippet != "" {
		lines := strings.Split(err.Snippet, "\n")
		if len(lines) > 0 {
			context = []string{lines[0]}
		}
	}

	errorType := "error"
	if strings.Contains(strings.ToLower(err.Kind), "warning") {
		errorType = "warning"
	}

	message := err.Message
	if err.Kind != "" {
		docsURL := getActionlintDocsURL(err.Kind)
		message = fmt.Sprintf("[%s] %s\n\n  📖 %s", err.Kind, err.Message, docsURL)
	}

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
	return console.FormatError(compilerErr)
}
