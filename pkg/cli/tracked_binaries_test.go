//go:build !integration

package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// toolBinaryNames lists build-time tools that are installed via `go install` or
// `go tool` (see the `tools` target in the Makefile). Their compiled binaries
// must never be committed to the repository.
var toolBinaryNames = []string{
	"actionlint",
	"gosec",
	"govulncheck",
	"golangci-lint",
	"gopls",
}

// TestNoToolBinariesTracked ensures pre-built third-party tool binaries are not
// committed to the repository. Committing them adds opaque compiled code to git
// history and bypasses code review.
func TestNoToolBinariesTracked(t *testing.T) {
	repoRoot := repositoryRootForTest(t)

	cmd := exec.Command("git", "ls-files", "-z")
	cmd.Dir = repoRoot
	output, err := cmd.Output()
	if err != nil {
		t.Skipf("Skipping test - unable to list tracked files: %v", err)
	}

	// git ls-files -z NUL-terminates every record, so drop the trailing separator.
	byName := make(map[string]string)
	for file := range strings.SplitSeq(strings.TrimSuffix(string(output), "\x00"), "\x00") {
		if file != "" {
			byName[filepath.Base(file)] = file
		}
	}
	require.NotEmpty(t, byName, "Should find tracked files in the repository")

	for _, name := range toolBinaryNames {
		path, found := byName[name]
		assert.Falsef(t, found, "Tool binary %q must not be committed (found at %q); it is installed via `go install`/`go tool`", name, path)
	}
}

// TestToolBinariesGitignored ensures the tool binaries that were previously
// committed by accident stay ignored so they cannot be committed again. Only
// `actionlint` and `gosec` are checked because those are the tools whose binaries
// end up in the repository root when built locally.
func TestToolBinariesGitignored(t *testing.T) {
	repoRoot := repositoryRootForTest(t)

	content, err := os.ReadFile(filepath.Join(repoRoot, ".gitignore"))
	require.NoError(t, err, "Should read .gitignore")

	entries := make(map[string]bool)
	for line := range strings.SplitSeq(string(content), "\n") {
		entries[strings.TrimSpace(line)] = true
	}

	for _, name := range []string{"/actionlint", "/gosec"} {
		assert.Truef(t, entries[name], ".gitignore should contain %q", name)
	}
}

// repositoryRootForTest returns the root directory of this repository checkout.
func repositoryRootForTest(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	require.NoError(t, err, "Should get current directory")

	root, err := filepath.Abs(filepath.Join(wd, "..", ".."))
	require.NoError(t, err, "Should resolve repository root")

	if _, err := os.Stat(filepath.Join(root, ".gitignore")); err != nil {
		t.Skipf("Skipping test - repository root not found: %v", err)
	}
	return root
}
