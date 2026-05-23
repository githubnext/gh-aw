//go:build !js && !wasm

// This file provides Git repository utilities for workflow compilation.
//
// This file contains the small set of Git helpers still needed by workflow
// compilation: locating the repository root and running git commands with
// consistent spinner behavior in interactive terminals.
//
// # Organization Rationale
//
// These Git utilities are grouped in a helper file because they:
//   - Provide Git-specific functionality
//   - Are used by multiple workflow compilation modules
//   - Encapsulate Git command execution and error handling
//   - Have a clear domain focus (repository discovery and command execution)
//
// This follows the helper file conventions documented in the developer instructions.
// See skills/developer/SKILL.md#helper-file-conventions for details.
//
// # Key Functions
//
// Command Execution with Spinner:
//   - RunGitCombined() - Execute git command with spinner, returning combined stdout+stderr
//
// # Usage Patterns
//
// These functions are primarily used during workflow compilation to:
//   - Discover the active repository root when workflows need repository context
//   - Execute git commands with spinner feedback in interactive terminals

package workflow

import (
	"os/exec"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/tty"
)

var gitHelpersLog = logger.New("workflow:git_helpers")

// findGitRoot attempts to find the git repository root directory.
// Returns empty string if not in a git repository or if git command fails.
// This function is safe to call from any context and won't cause errors if git is not available.
func findGitRoot() string {
	gitHelpersLog.Print("Attempting to find git root directory")
	gitRoot, err := gitutil.FindGitRoot()
	if err != nil {
		gitHelpersLog.Printf("Could not find git root (not a git repo or git not available): %v", err)
		return ""
	}
	gitHelpersLog.Printf("Found git root: %s", gitRoot)
	return gitRoot
}

// runGitWithSpinner executes a git command with an optional spinner.
// If stderr is a terminal, a spinner is shown while the command runs.
func runGitWithSpinner(spinnerMessage string, combined bool, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	gitHelpersLog.Printf("Running git command: git %s", strings.Join(args, " "))

	if tty.IsStderrTerminal() {
		spinner := console.NewSpinner(spinnerMessage)
		spinner.Start()
		var output []byte
		var err error
		if combined {
			output, err = cmd.CombinedOutput()
		} else {
			output, err = cmd.Output()
		}
		spinner.Stop()
		return output, err
	}

	if combined {
		return cmd.CombinedOutput()
	}
	return cmd.Output()
}

// RunGitCombined executes a git command with an optional spinner, returning combined stdout+stderr.
// If stderr is a terminal, a spinner with the given message is shown.
func RunGitCombined(spinnerMessage string, args ...string) ([]byte, error) {
	return runGitWithSpinner(spinnerMessage, true, args...)
}
