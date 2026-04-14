//go:build !js && !wasm

package workflow

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/tty"
)

var githubCLILog = logger.New("workflow:github_cli")

// setupGHCommand creates an exec.Cmd for gh CLI with proper token configuration.
// This is the core implementation shared by ExecGH and ExecGHContext.
// When ctx is nil, it uses exec.Command; when ctx is provided, it uses exec.CommandContext.
func setupGHCommand(ctx context.Context, args ...string) *exec.Cmd {
	// Check if GH_TOKEN or GITHUB_TOKEN is available
	ghToken := os.Getenv("GH_TOKEN")
	githubToken := os.Getenv("GITHUB_TOKEN")

	var cmd *exec.Cmd
	if ctx != nil {
		cmd = exec.CommandContext(ctx, "gh", args...)
		if ghToken != "" || githubToken != "" {
			githubCLILog.Printf("Using gh CLI via go-gh/v2 for command with context: gh %v", args)
		} else {
			githubCLILog.Printf("No token available, using default gh CLI with context for command: gh %v", args)
		}
	} else {
		cmd = exec.Command("gh", args...)
		if ghToken != "" || githubToken != "" {
			githubCLILog.Printf("Using gh CLI via go-gh/v2 for command: gh %v", args)
		} else {
			githubCLILog.Printf("No token available, using default gh CLI for command: gh %v", args)
		}
	}

	// Set up environment to ensure token is available
	// Only add GH_TOKEN if it's not set but GITHUB_TOKEN is available
	if ghToken == "" && githubToken != "" {
		githubCLILog.Printf("GH_TOKEN not set, using GITHUB_TOKEN for gh CLI")
		cmd.Env = append(os.Environ(), "GH_TOKEN="+githubToken)
	}

	return cmd
}

// ExecGH wraps gh CLI calls and ensures proper token configuration.
// It uses go-gh/v2 to execute gh commands when GH_TOKEN or GITHUB_TOKEN is available,
// otherwise falls back to direct exec.Command for backward compatibility.
//
// Usage:
//
//	cmd := ExecGH("api", "/user")
//	output, err := cmd.Output()
func ExecGH(args ...string) *exec.Cmd {
	//nolint:staticcheck // Passing nil context to use exec.Command instead of exec.CommandContext
	return setupGHCommand(nil, args...)
}

// ExecGHContext wraps gh CLI calls with context support and ensures proper token configuration.
// Similar to ExecGH but accepts a context for cancellation and timeout support.
//
// Usage:
//
//	cmd := ExecGHContext(ctx, "api", "/user")
//	output, err := cmd.Output()
func ExecGHContext(ctx context.Context, args ...string) *exec.Cmd {
	return setupGHCommand(ctx, args...)
}

// enrichGHError enriches an error returned from a gh CLI command with the
// stderr output captured in *exec.ExitError. When cmd.Output() (stdout-only
// capture) fails, Go populates ExitError.Stderr with the command's stderr,
// which typically contains the human-readable error message from gh.
// This function appends that message to the error so callers see useful
// diagnostics instead of a bare "exit status 1".
func enrichGHError(err error) error {
	if err == nil {
		return nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && len(exitErr.Stderr) > 0 {
		stderr := strings.TrimSpace(string(exitErr.Stderr))
		if stderr != "" {
			return fmt.Errorf("%w: %s", err, stderr)
		}
	}
	return err
}

// runGHWithSpinnerContext executes a gh CLI command with context support, a spinner,
// and returns the output. This is the core implementation for RunGHContext.
func runGHWithSpinnerContext(ctx context.Context, spinnerMessage string, combined bool, args ...string) ([]byte, error) {
	cmd := ExecGHContext(ctx, args...)

	// Show spinner in interactive terminals
	if tty.IsStderrTerminal() {
		spinner := console.NewSpinner(spinnerMessage)
		spinner.Start()
		var output []byte
		var err error
		if combined {
			output, err = cmd.CombinedOutput()
		} else {
			output, err = cmd.Output()
			err = enrichGHError(err)
		}
		spinner.Stop()
		return output, err
	}

	if combined {
		return cmd.CombinedOutput()
	}
	output, err := cmd.Output()
	return output, enrichGHError(err)
}

// runGHWithSpinner executes a gh CLI command with a spinner and returns the output.
// This is the core implementation shared by RunGH and RunGHCombined.
func runGHWithSpinner(spinnerMessage string, combined bool, args ...string) ([]byte, error) {
	cmd := ExecGH(args...)

	// Show spinner in interactive terminals
	if tty.IsStderrTerminal() {
		spinner := console.NewSpinner(spinnerMessage)
		spinner.Start()
		var output []byte
		var err error
		if combined {
			output, err = cmd.CombinedOutput()
		} else {
			output, err = cmd.Output()
			err = enrichGHError(err)
		}
		spinner.Stop()
		return output, err
	}

	if combined {
		return cmd.CombinedOutput()
	}
	output, err := cmd.Output()
	return output, enrichGHError(err)
}

// RunGH executes a gh CLI command with a spinner and returns the stdout output.
// The spinner is shown in interactive terminals to provide feedback during network operations.
// The spinnerMessage parameter describes what operation is being performed.
//
// Usage:
//
//	output, err := RunGH("Fetching user info...", "api", "/user")
func RunGH(spinnerMessage string, args ...string) ([]byte, error) {
	return runGHWithSpinner(spinnerMessage, false, args...)
}

// RunGHContext executes a gh CLI command with context support (for cancellation/timeout), a
// spinner, and returns the stdout output. The spinner is shown in interactive terminals to
// provide feedback during network operations.
//
// Usage:
//
//	output, err := RunGHContext(ctx, "Fetching user info...", "api", "/user")
func RunGHContext(ctx context.Context, spinnerMessage string, args ...string) ([]byte, error) {
	return runGHWithSpinnerContext(ctx, spinnerMessage, false, args...)
}

// RunGHCombined executes a gh CLI command with a spinner and returns combined stdout+stderr output.
// The spinner is shown in interactive terminals to provide feedback during network operations.
// Use this when you need to capture error messages from stderr.
//
// Usage:
//
//	output, err := RunGHCombined("Creating repository...", "repo", "create", "myrepo")
func RunGHCombined(spinnerMessage string, args ...string) ([]byte, error) {
	return runGHWithSpinner(spinnerMessage, true, args...)
}

// RunGHWithHost executes a gh CLI command with a spinner, targeting a specific GitHub host.
// For non-github.com hosts (GHES, Proxima/data residency), the GH_HOST environment variable
// is set on the command. This is necessary because most gh subcommands (repo, pr, run, etc.)
// do not accept a --hostname flag — only `gh api` does.
//
// Usage:
//
//	output, err := RunGHWithHost("Fetching repo info...", "myorg.ghe.com", "repo", "view", "--json", "owner,name")
func RunGHWithHost(spinnerMessage string, host string, args ...string) ([]byte, error) {
	cmd := ExecGH(args...)
	SetGHHostEnv(cmd, host)

	if tty.IsStderrTerminal() {
		spinner := console.NewSpinner(spinnerMessage)
		spinner.Start()
		output, err := cmd.Output()
		err = enrichGHError(err)
		spinner.Stop()
		return output, err
	}

	output, err := cmd.Output()
	return output, enrichGHError(err)
}

// SetGHHostEnv sets the GH_HOST environment variable on the command for non-github.com hosts.
// This is needed for GitHub Enterprise Server (GHES) and Proxima (data residency) instances
// because commands like `gh repo view`, `gh pr create`, and `gh run view` do not accept a
// --hostname flag (unlike `gh api` which does).
func SetGHHostEnv(cmd *exec.Cmd, host string) {
	if host == "" || host == "github.com" {
		return
	}
	if cmd.Env == nil {
		cmd.Env = append(os.Environ(), "GH_HOST="+host)
	} else {
		cmd.Env = append(cmd.Env, "GH_HOST="+host)
	}
}
