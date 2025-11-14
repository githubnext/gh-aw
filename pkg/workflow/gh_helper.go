package workflow

import (
	"os/exec"

	"github.com/githubnext/gh-aw/pkg/ghcli"
)

// ExecGH wraps exec.Command for "gh" CLI calls and ensures proper token configuration.
// It sets GH_TOKEN from GITHUB_TOKEN if GH_TOKEN is not already set.
// This ensures gh CLI commands work in environments where GITHUB_TOKEN is set but GH_TOKEN is not.
//
// Usage:
//
//	cmd := ExecGH("api", "/user")
//	output, err := cmd.Output()
//
// Deprecated: Use ghcli.ExecGH directly instead.
func ExecGH(args ...string) *exec.Cmd {
	return ghcli.ExecGH(args...)
}
