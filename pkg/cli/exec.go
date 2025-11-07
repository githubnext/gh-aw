package cli

import (
	"bytes"
	"os"
	"os/exec"
	"strings"

	"github.com/cli/go-gh/v2"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var execLog = logger.New("cli:exec")

// ghExecOrFallback executes a gh CLI command if GH_TOKEN is available,
// otherwise falls back to an alternative command.
// Returns the stdout, stderr, and error from whichever command was executed.
func ghExecOrFallback(ghArgs []string, fallbackCmd string, fallbackArgs []string, fallbackEnv []string) (string, string, error) {
	ghToken := os.Getenv("GH_TOKEN")

	if ghToken != "" {
		// Use gh CLI when GH_TOKEN is available
		execLog.Printf("Using gh CLI: gh %s", strings.Join(ghArgs, " "))
		stdout, stderr, err := gh.Exec(ghArgs...)
		return stdout.String(), stderr.String(), err
	}

	// Fall back to alternative command when GH_TOKEN is not available
	execLog.Printf("Using fallback command: %s %s", fallbackCmd, strings.Join(fallbackArgs, " "))
	cmd := exec.Command(fallbackCmd, fallbackArgs...)

	// Add custom environment variables if provided
	if len(fallbackEnv) > 0 {
		cmd.Env = append(os.Environ(), fallbackEnv...)
	}

	// Capture stdout and stderr separately like gh.Exec
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}
