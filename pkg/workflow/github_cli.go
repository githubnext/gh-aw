package workflow

import (
	"bytes"
	"context"
	"os"
	"os/exec"

	"github.com/cli/go-gh/v2"
	"github.com/cli/go-gh/v2/pkg/auth"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var githubCLILog = logger.New("workflow:github_cli")

// Token source constants from auth.TokenForHost()
const (
	// tokenSourceGHToken indicates the token came from GH_TOKEN environment variable
	tokenSourceGHToken = "GH_TOKEN"
	// tokenSourceGitHubToken indicates the token came from GITHUB_TOKEN environment variable
	tokenSourceGitHubToken = "GITHUB_TOKEN"
	// tokenSourceOAuthToken indicates the token came from gh config file
	tokenSourceOAuthToken = "oauth_token"
	// tokenSourceGH indicates the token came from system keyring via 'gh auth token'
	tokenSourceGH = "gh"
	// tokenSourceDefault indicates no token was found
	tokenSourceDefault = "default"
)

// ExecGH wraps gh CLI calls and ensures proper token configuration.
// It uses go-gh/v2 to execute gh commands with proper authentication using pkg/auth.
//
// Usage:
//
//	cmd := ExecGH("api", "/user")
//	output, err := cmd.Output()
func ExecGH(args ...string) *exec.Cmd {
	// Use pkg/auth to get token for github.com
	token, tokenSource := auth.TokenForHost("github.com")

	// If we have a token, ensure it's available to gh CLI
	if token != "" {
		githubCLILog.Printf("Using gh CLI with token from %s for command: gh %v", tokenSource, args)

		// Create a command that will execute via go-gh
		// We return an exec.Cmd for backward compatibility with existing code
		cmd := exec.Command("gh", args...)

		// Set up environment to ensure token is available
		// If token source is not GH_TOKEN, we need to set it explicitly
		// This ensures the token is available even if it came from GITHUB_TOKEN or config
		if tokenSource != tokenSourceGHToken {
			githubCLILog.Printf("Setting GH_TOKEN for gh CLI from %s", tokenSource)
			cmd.Env = append(os.Environ(), "GH_TOKEN="+token)
		}

		return cmd
	}

	// If no token is available, use default gh CLI behavior
	githubCLILog.Printf("No token available, using default gh CLI for command: gh %v", args)
	return exec.Command("gh", args...)
}

// ExecGHContext wraps gh CLI calls with context support and ensures proper token configuration.
// Similar to ExecGH but accepts a context for cancellation and timeout support.
//
// Usage:
//
//	cmd := ExecGHContext(ctx, "api", "/user")
//	output, err := cmd.Output()
func ExecGHContext(ctx context.Context, args ...string) *exec.Cmd {
	// Use pkg/auth to get token for github.com
	token, tokenSource := auth.TokenForHost("github.com")

	// If we have a token, ensure it's available to gh CLI
	if token != "" {
		githubCLILog.Printf("Using gh CLI with token from %s for command with context: gh %v", tokenSource, args)

		// Create a command that will execute via go-gh with context
		cmd := exec.CommandContext(ctx, "gh", args...)

		// Set up environment to ensure token is available
		// If token source is not GH_TOKEN, we need to set it explicitly
		// This ensures the token is available even if it came from GITHUB_TOKEN or config
		if tokenSource != tokenSourceGHToken {
			githubCLILog.Printf("Setting GH_TOKEN for gh CLI from %s", tokenSource)
			cmd.Env = append(os.Environ(), "GH_TOKEN="+token)
		}

		return cmd
	}

	// If no token is available, use default gh CLI behavior
	githubCLILog.Printf("No token available, using default gh CLI with context for command: gh %v", args)
	return exec.CommandContext(ctx, "gh", args...)
}

// ExecGHWithOutput executes a gh CLI command using go-gh/v2 and returns stdout, stderr, and error.
// This is a convenience wrapper that directly uses go-gh/v2's Exec function.
//
// Usage:
//
//	stdout, stderr, err := ExecGHWithOutput("api", "/user")
func ExecGHWithOutput(args ...string) (stdout, stderr bytes.Buffer, err error) {
	githubCLILog.Printf("Executing gh CLI command via go-gh/v2: gh %v", args)
	return gh.Exec(args...)
}
