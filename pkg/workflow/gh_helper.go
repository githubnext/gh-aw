package workflow

import (
	"os"
	"os/exec"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var ghHelperLog = logger.New("workflow:gh_helper")

// ExecGH wraps exec.Command for "gh" CLI calls and ensures proper token configuration.
// It sets GH_TOKEN from GITHUB_TOKEN if GH_TOKEN is not already set.
// This ensures gh CLI commands work in environments where GITHUB_TOKEN is set but GH_TOKEN is not.
//
// Usage:
//
//	cmd := ExecGH("api", "/user")
//	output, err := cmd.Output()
func ExecGH(args ...string) *exec.Cmd {
	cmd := exec.Command("gh", args...)

	// Check if GH_TOKEN is already set
	ghToken := os.Getenv("GH_TOKEN")
	if ghToken != "" {
		ghHelperLog.Printf("GH_TOKEN is set, using it for gh CLI")
		return cmd
	}

	// Fall back to GITHUB_TOKEN if available
	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken != "" {
		ghHelperLog.Printf("GH_TOKEN not set, using GITHUB_TOKEN as fallback for gh CLI")
		// Set GH_TOKEN in the command's environment
		cmd.Env = append(os.Environ(), "GH_TOKEN="+githubToken)
	} else {
		ghHelperLog.Printf("Neither GH_TOKEN nor GITHUB_TOKEN is set, gh CLI will use default authentication")
	}

	return cmd
}
