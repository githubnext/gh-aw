// This file provides shellcheck integration for workflow run step linting.
//
// It extracts run: step scripts from compiled lock files and runs shellcheck
// on each shell snippet, reporting issues and ignoring known false positives
// introduced by GitHub Actions expression syntax.
//
// # Key Functions
//
//   - runShellcheckOnLockFiles() - Run shellcheck on run steps in multiple lock files
//   - extractRunStepsFromLockFile() - Parse a lock file and extract run step info
//   - isShellcheckAvailable() - Check whether the shellcheck binary is in PATH
//   - isShellcheckableShell() - True for bash/sh steps; false for pwsh/python/etc.

package cli

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/goccy/go-yaml"
)

var shellcheckLog = logger.New("cli:shellcheck")

// shellcheckDefaultIgnoreCodes lists SC error codes that are false positives
// in GitHub Actions run: scripts and are always suppressed.
//
// Rationale for each code:
//
//	SC2016: "${{ }}" GitHub Actions expression syntax appears in single-quoted
//	        strings which shellcheck flags as unexpanded variable references.
//	SC1090: "Can't follow non-constant source" – scripts are downloaded and
//	        sourced dynamically at runtime; the source path is not resolvable at
//	        lint time.
//	SC1091: "Not following: shell file doesn't exist" – same reason as SC1090.
var shellcheckDefaultIgnoreCodes = []string{"SC2016", "SC1090", "SC1091"}

// runStepInfo captures the information from a single run: step in a lock file
// that is needed to run shellcheck on the script snippet.
type runStepInfo struct {
	// Name is the step's "name" field, used only for diagnostic messages.
	Name string
	// Script is the raw content of the run: field.
	Script string
	// Shell is the value of the step's "shell" field, or "" when unset
	// (GitHub Actions defaults to bash in that case).
	Shell string
	// LockFile is the absolute path of the lock file that contains this step.
	LockFile string
}

// isShellcheckAvailable returns true when the shellcheck binary can be found in PATH.
func isShellcheckAvailable() bool {
	_, err := exec.LookPath("shellcheck")
	return err == nil
}

// isShellcheckableShell returns true for shell values that shellcheck can lint.
// GitHub Actions supports bash (default), sh, pwsh, powershell, python, and
// custom shells. Only bash and sh are valid targets for shellcheck.
func isShellcheckableShell(shell string) bool {
	switch strings.ToLower(shell) {
	case "", "bash":
		// Empty means GitHub Actions default (bash).
		return true
	case "sh":
		return true
	default:
		return false
	}
}

// shellcheckShell returns the value to pass to shellcheck's --shell flag.
// When shell is empty the GitHub Actions default (bash) is used.
func shellcheckShell(shell string) string {
	if strings.ToLower(shell) == "sh" {
		return "sh"
	}
	return "bash"
}

// extractRunStepsFromLockFile parses a compiled lock file and returns all
// run: steps whose shell is lintable by shellcheck.
func extractRunStepsFromLockFile(lockFile string) ([]runStepInfo, error) {
	shellcheckLog.Printf("Extracting run steps from %s", lockFile)

	content, err := os.ReadFile(lockFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read lock file %s: %w", lockFile, err)
	}

	var workflowYAML map[string]any
	if err := yaml.Unmarshal(content, &workflowYAML); err != nil {
		return nil, fmt.Errorf("failed to parse YAML in %s: %w", lockFile, err)
	}

	var steps []runStepInfo

	jobs, ok := workflowYAML["jobs"].(map[string]any)
	if !ok {
		return steps, nil
	}

	for _, jobData := range jobs {
		job, ok := jobData.(map[string]any)
		if !ok {
			continue
		}
		rawSteps, ok := job["steps"].([]any)
		if !ok {
			continue
		}
		for _, stepData := range rawSteps {
			step, ok := stepData.(map[string]any)
			if !ok {
				continue
			}
			runScript, ok := step["run"].(string)
			if !ok || runScript == "" {
				continue
			}
			shell, _ := step["shell"].(string)
			if !isShellcheckableShell(shell) {
				continue
			}
			name, _ := step["name"].(string)
			steps = append(steps, runStepInfo{
				Name:     name,
				Script:   runScript,
				Shell:    shell,
				LockFile: lockFile,
			})
		}
	}

	shellcheckLog.Printf("Found %d shellcheckable run steps in %s", len(steps), lockFile)
	return steps, nil
}

// runShellcheckOnScript writes script to a temporary file and invokes shellcheck.
// It prints any findings to stderr and returns a non-nil error when shellcheck
// reports one or more issues.
func runShellcheckOnScript(info runStepInfo, ignoreCodes []string, verbose bool) error {
	shellcheckLog.Printf("Running shellcheck on step %q (shell=%s)", info.Name, info.Shell)

	// Write script to a temp file so shellcheck can lint it.
	tmpFile, err := os.CreateTemp("", "gh-aw-shellcheck-*.sh")
	if err != nil {
		return fmt.Errorf("failed to create temp file for shellcheck: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(info.Script); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write shellcheck temp file: %w", err)
	}
	tmpFile.Close()

	args := []string{
		"--shell=" + shellcheckShell(info.Shell),
		"--format=gcc",
	}
	for _, code := range ignoreCodes {
		args = append(args, "--exclude="+code)
	}
	args = append(args, tmpFile.Name())

	// #nosec G204 -- shellcheck is a trusted system binary; args are built
	// from controlled values (shell name and SC codes). The temp file path is
	// OS-generated and not user-controlled.
	cmd := exec.Command("shellcheck", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	// Display findings – replace the temp file path with the lock file for clarity.
	output := strings.ReplaceAll(stdout.String(), tmpFile.Name(), filepath.Base(info.LockFile))
	if stderr.Len() > 0 {
		output += strings.ReplaceAll(stderr.String(), tmpFile.Name(), filepath.Base(info.LockFile))
	}

	if output != "" {
		stepLabel := filepath.Base(info.LockFile)
		if info.Name != "" {
			stepLabel += " (step: " + info.Name + ")"
		}
		fmt.Fprintf(os.Stderr, "%s\n", console.FormatWarningMessage("shellcheck findings in "+stepLabel+":"))
		fmt.Fprint(os.Stderr, output)
	}

	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if exitErr.ExitCode() == 1 {
				// Exit code 1 means shellcheck found issues; already printed above.
				return fmt.Errorf("shellcheck found issues in %s", stepLabel(info))
			}
		}
		return fmt.Errorf("shellcheck failed: %w", err)
	}

	return nil
}

func stepLabel(info runStepInfo) string {
	if info.Name != "" {
		return filepath.Base(info.LockFile) + " (step: " + info.Name + ")"
	}
	return filepath.Base(info.LockFile)
}

// runShellcheckOnLockFiles extracts run: steps from each lock file and runs
// shellcheck on the shell snippets. It uses shellcheckDefaultIgnoreCodes to
// suppress known false positives from GitHub Actions expression syntax.
//
// When strict is false, individual step failures are printed as warnings and
// the function returns nil. When strict is true, the first step failure causes
// an error to be returned.
func runShellcheckOnLockFiles(lockFiles []string, verbose bool, strict bool) error {
	if len(lockFiles) == 0 {
		return nil
	}

	// Silently skip when shellcheck is not installed. The orchestrator is responsible
	// for warning the user in --validate mode.
	if !isShellcheckAvailable() {
		shellcheckLog.Print("shellcheck binary not found in PATH; skipping run step linting")
		return nil
	}

	shellcheckLog.Printf("Running shellcheck on run steps in %d lock file(s) (strict=%t)", len(lockFiles), strict)

	if len(lockFiles) == 1 {
		fmt.Fprintf(os.Stderr, "%s\n", console.FormatInfoMessage("Running shellcheck on run steps in "+filepath.Base(lockFiles[0])))
	} else {
		fmt.Fprintf(os.Stderr, "%s\n", console.FormatInfoMessage(fmt.Sprintf("Running shellcheck on run steps in %d files", len(lockFiles))))
	}

	var totalSteps, totalIssues int
	var firstErr error

	for _, lockFile := range lockFiles {
		steps, err := extractRunStepsFromLockFile(lockFile)
		if err != nil {
			shellcheckLog.Printf("Failed to extract run steps from %s: %v", lockFile, err)
			fmt.Fprintf(os.Stderr, "%s\n", console.FormatWarningMessage("shellcheck: could not parse "+filepath.Base(lockFile)+": "+err.Error()))
			continue
		}

		for _, step := range steps {
			totalSteps++
			if err := runShellcheckOnScript(step, shellcheckDefaultIgnoreCodes, verbose); err != nil {
				totalIssues++
				shellcheckLog.Printf("shellcheck issue in %s step %q: %v", lockFile, step.Name, err)
				if strict && firstErr == nil {
					firstErr = err
				}
			}
		}
	}

	shellcheckLog.Printf("shellcheck complete: steps=%d, issues=%d", totalSteps, totalIssues)

	if firstErr != nil {
		return fmt.Errorf("strict mode: shellcheck found issues in run steps: %w", firstErr)
	}
	return nil
}
