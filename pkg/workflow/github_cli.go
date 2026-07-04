//go:build !js && !wasm

package workflow

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/tty"
)

var githubCLILog = logger.New("workflow:github_cli")
var defaultGHHost struct {
	mu   sync.RWMutex
	host string
}

// SetDefaultGHHost sets the default host used by gh CLI helper commands when GH_HOST
// is not set in the process environment.
func SetDefaultGHHost(host string) {
	defaultGHHost.mu.Lock()
	defer defaultGHHost.mu.Unlock()
	defaultGHHost.host = host
}

func getDefaultGHHost() string {
	defaultGHHost.mu.RLock()
	defer defaultGHHost.mu.RUnlock()
	return defaultGHHost.host
}

// setupGHCommand creates an exec.Cmd for gh CLI with proper token configuration.
// This is the core implementation shared by ExecGH and ExecGHContext.
// When ctx is nil, it falls back to context.TODO().
func setupGHCommand(ctx context.Context, args ...string) *exec.Cmd {
	// Check if GH_TOKEN or GITHUB_TOKEN is available
	ghToken := lookupProcessEnv("GH_TOKEN")
	githubToken := lookupProcessEnv("GITHUB_TOKEN")

	if ctx == nil {
		ctx = context.TODO()
	}
	cmd := exec.CommandContext(ctx, "gh", args...)

	if ghToken != "" || githubToken != "" {
		githubCLILog.Printf("Token detected, using gh CLI for command: gh %v", args)
	} else {
		githubCLILog.Printf("No token available, using default gh CLI for command: gh %v", args)
	}

	// Set up environment to ensure token is available
	// Only add GH_TOKEN if it's not set but GITHUB_TOKEN is available
	if ghToken == "" && githubToken != "" {
		githubCLILog.Printf("GH_TOKEN not set, using GITHUB_TOKEN for gh CLI")
		cmd.Env = append(os.Environ(), "GH_TOKEN="+githubToken)
	}
	if lookupProcessEnv("GH_HOST") == "" {
		host := getDefaultGHHost()
		if host != "" && host != "github.com" {
			if cmd.Env == nil {
				// Build the base env from os.Environ(), filtering out GH_TOKEN when
				// the processEnvLookup says it is absent. This prevents the real
				// runner GH_TOKEN from leaking into commands that should not have it
				// (e.g., under test mocks or when only GH_HOST must be injected).
				cmd.Env = filterTokenFromEnv(os.Environ(), ghToken)
			}
			cmd.Env = append(cmd.Env, "GH_HOST="+host)
		}
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
// and returns the output. This is the core implementation for all RunGH* functions.
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

// RunGH executes a gh CLI command with a spinner and returns the stdout output.
// The spinner is shown in interactive terminals to provide feedback during network operations.
// The spinnerMessage parameter describes what operation is being performed.
//
// Usage:
//
//	output, err := RunGH("Fetching user info...", "api", "/user")
func RunGH(spinnerMessage string, args ...string) ([]byte, error) {
	return RunGHContext(context.Background(), spinnerMessage, args...)
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
	return RunGHCombinedContext(context.Background(), spinnerMessage, args...)
}

// RunGHCombinedContext executes a gh CLI command with context support (for cancellation/timeout),
// a spinner, and returns combined stdout+stderr output. The spinner is shown in interactive
// terminals to provide feedback during network operations.
//
// Usage:
//
//	output, err := RunGHCombinedContext(ctx, "Fetching releases...", "api", "/repos/owner/repo/releases")
func RunGHCombinedContext(ctx context.Context, spinnerMessage string, args ...string) ([]byte, error) {
	return runGHWithSpinnerContext(ctx, spinnerMessage, true, args...)
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

// RunGHContextWithHost executes a gh CLI command with context support, a spinner,
// and an explicit GitHub host.
func RunGHContextWithHost(ctx context.Context, spinnerMessage string, host string, args ...string) ([]byte, error) {
	cmd := ExecGHContext(ctx, args...)
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

// filterTokenFromEnv returns env unchanged when knownToken is non-empty (the token is
// legitimately present), or returns a copy of env with all "GH_TOKEN=…" entries removed
// when knownToken is empty. This prevents the real runner GH_TOKEN from leaking into a
// command's environment when the processEnvLookup indicates no token should be present
// (e.g., under test mocks that override the env lookup).
func filterTokenFromEnv(env []string, knownToken string) []string {
	if knownToken != "" {
		return env
	}
	filtered := make([]string, 0, len(env))
	for _, e := range env {
		if !strings.HasPrefix(e, "GH_TOKEN=") {
			filtered = append(filtered, e)
		}
	}
	return filtered
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

// ForceGHHostEnv forces GH_HOST=<host> on the command's environment, overriding
// any GH_HOST already present in the process environment or cmd.Env.
// Unlike SetGHHostEnv, this always sets GH_HOST — including for "github.com" —
// so that a GHE host in the process environment cannot be inherited by the subprocess.
func ForceGHHostEnv(cmd *exec.Cmd, host string) {
	if host == "" {
		return
	}
	base := cmd.Env
	if base == nil {
		base = os.Environ()
	}
	filtered := make([]string, 0, len(base)+1)
	for _, e := range base {
		if !strings.HasPrefix(e, "GH_HOST=") {
			filtered = append(filtered, e)
		}
	}
	cmd.Env = append(filtered, "GH_HOST="+host)
}
