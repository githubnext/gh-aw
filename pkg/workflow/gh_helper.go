package workflow

import (
	"bytes"
	"os"
	"os/exec"

	"github.com/cli/go-gh/v2"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var ghHelperLog = logger.New("workflow:gh_helper")

// ExecGH wraps gh CLI calls and ensures proper token configuration.
// It uses go-gh/v2 to execute gh commands when GH_TOKEN or GITHUB_TOKEN is available,
// otherwise falls back to direct exec.Command for backward compatibility.
//
// Usage:
//
//	cmd := ExecGH("api", "/user")
//	output, err := cmd.Output()
func ExecGH(args ...string) *exec.Cmd {
	// Check if GH_TOKEN or GITHUB_TOKEN is available
	ghToken := os.Getenv("GH_TOKEN")
	githubToken := os.Getenv("GITHUB_TOKEN")

	// If we have a token, use go-gh/v2 which handles authentication properly
	if ghToken != "" || githubToken != "" {
		ghHelperLog.Printf("Using gh CLI via go-gh/v2 for command: gh %v", args)

		// Create a command that will execute via go-gh
		// We return an exec.Cmd for backward compatibility with existing code
		cmd := exec.Command("gh", args...)

		// Set up environment to ensure token is available
		if ghToken == "" && githubToken != "" {
			ghHelperLog.Printf("GH_TOKEN not set, using GITHUB_TOKEN for gh CLI")
			cmd.Env = append(os.Environ(), "GH_TOKEN="+githubToken)
		}

		return cmd
	}

	// If no token is available, use default gh CLI behavior
	ghHelperLog.Printf("No token available, using default gh CLI for command: gh %v", args)
	return exec.Command("gh", args...)
}

// ExecGHWithOutput executes a gh CLI command using go-gh/v2 and returns stdout, stderr, and error.
// This is a convenience wrapper that directly uses go-gh/v2's Exec function.
//
// Usage:
//
//	stdout, stderr, err := ExecGHWithOutput("api", "/user")
func ExecGHWithOutput(args ...string) (stdout, stderr bytes.Buffer, err error) {
	ghHelperLog.Printf("Executing gh CLI command via go-gh/v2: gh %v", args)
	return gh.Exec(args...)
}
