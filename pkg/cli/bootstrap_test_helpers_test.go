//go:build !integration

package cli

import (
	"os/exec"
	"testing"
)

func initBootstrapGitRepo(t *testing.T) string {
	t.Helper()

	repoDir := t.TempDir()
	cmd := exec.Command("git", "init")
	cmd.Dir = repoDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, string(output))
	}

	return repoDir
}
