package cli

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// checkLockfileDrift fails if any .lock.yml files in the workflow directory
// are modified/added/deleted after compilation.
//
// This is intended for CI/PR checks: run `gh-aw compile --check` and require that
// the working tree remains clean for lockfiles.
func checkLockfileDrift(workflowDir string) error {
	if !isGitRepo() {
		return fmt.Errorf("--check requires running inside a git repository")
	}

	gitRoot, err := findGitRoot()
	if err != nil {
		return fmt.Errorf("--check requires running inside a git repository: %w", err)
	}

	relDir := workflowDir
	if relDir == "" {
		relDir = ".github/workflows"
	}
	relDir = filepath.Clean(relDir)

	// Use porcelain output and restrict to workflow directory.
	cmd := exec.Command("git", "-C", gitRoot, "status", "--porcelain", "--", relDir)
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check git status for %s: %w", relDir, err)
	}

	var changedLockfiles []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Porcelain: XY <path> (or XY <old> -> <new>).
		// We only care about paths mentioning .lock.yml.
		if strings.Contains(line, ".lock.yml") {
			changedLockfiles = append(changedLockfiles, line)
		}
	}

	if len(changedLockfiles) == 0 {
		return nil
	}

	var b strings.Builder
	b.WriteString("Lockfile drift detected: compilation produced changes to .lock.yml files.\n\n")
	b.WriteString("To fix:\n")
	b.WriteString("  1) Run: gh-aw compile\n")
	b.WriteString("  2) Commit the updated .lock.yml files\n\n")
	b.WriteString("Changed entries (git status --porcelain):\n")
	for _, l := range changedLockfiles {
		b.WriteString("  ")
		b.WriteString(l)
		b.WriteString("\n")
	}

	return fmt.Errorf("%s", b.String())
}
