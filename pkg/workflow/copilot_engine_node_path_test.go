//go:build !integration && !windows

package workflow

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestNodePathSetupCommandWithoutNpmBinary verifies that nodePathSetupCommand still
// resolves NODE_PATH correctly when the npm binary is absent from PATH (e.g. a sandbox
// that mounts only node and its global node_modules directory, as avenger.md does after
// removing the invalid npm symlink bind mount). Without this fallback, `npm root -g`
// would silently fail (swallowed by `|| true`) and NODE_PATH would remain unset even
// though the global node_modules directory is present on disk.
func TestNodePathSetupCommandWithoutNpmBinary(t *testing.T) {
	root := t.TempDir()
	nodeBin := filepath.Join(root, "bin", "node")
	globalNodeModules := filepath.Join(root, "lib", "node_modules")

	if err := os.MkdirAll(filepath.Dir(nodeBin), 0o755); err != nil {
		t.Fatalf("failed to create bin dir: %v", err)
	}
	if err := os.WriteFile(nodeBin, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("failed to create fake node binary: %v", err)
	}
	if err := os.MkdirAll(globalNodeModules, 0o755); err != nil {
		t.Fatalf("failed to create global node_modules dir: %v", err)
	}

	// Use a PATH that provides standard shell utilities (dirname, etc.) but no npm
	// binary anywhere on it, simulating a sandbox where only node (via GH_AW_NODE_EXEC)
	// is mounted and npm is absent.
	script := nodePathSetupCommand + `; printf '%s' "$NODE_PATH"`
	cmd := exec.Command("bash", "-c", script)
	cmd.Env = []string{
		"PATH=/usr/bin:/bin",
		"GH_AW_NODE_EXEC=" + nodeBin,
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("nodePathSetupCommand failed: %v\noutput: %s", err, out)
	}

	got := strings.TrimSpace(string(out))
	if got != globalNodeModules {
		t.Errorf("expected NODE_PATH to fall back to %q when npm is unavailable, got %q", globalNodeModules, got)
	}
}

// TestNodePathSetupCommandPrefersNpmRootWhenAvailable verifies that nodePathSetupCommand
// still prefers `npm root -g` when npm is present on PATH, preserving prior behavior for
// sandboxes/runners that do mount or install npm.
func TestNodePathSetupCommandPrefersNpmRootWhenAvailable(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("failed to create bin dir: %v", err)
	}

	npmGlobalRoot := filepath.Join(root, "npm-root", "node_modules")
	if err := os.MkdirAll(npmGlobalRoot, 0o755); err != nil {
		t.Fatalf("failed to create npm global root dir: %v", err)
	}
	fakeNpm := filepath.Join(binDir, "npm")
	fakeNpmScript := "#!/bin/sh\nif [ \"$1 $2\" = \"root -g\" ]; then printf '%s' " + shellQuote(npmGlobalRoot) + "; fi\n"
	if err := os.WriteFile(fakeNpm, []byte(fakeNpmScript), 0o755); err != nil {
		t.Fatalf("failed to create fake npm binary: %v", err)
	}

	// A fallback node path that must NOT be used when npm resolves successfully; it is
	// intentionally left non-existent to prove the fallback branch was skipped.
	unusedFallback := filepath.Join(root, "should-not-be-used", "bin", "node")

	script := nodePathSetupCommand + `; printf '%s' "$NODE_PATH"`
	cmd := exec.Command("bash", "-c", script)
	cmd.Env = []string{
		"PATH=" + binDir + ":/usr/bin:/bin",
		"GH_AW_NODE_EXEC=" + unusedFallback,
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("nodePathSetupCommand failed: %v\noutput: %s", err, out)
	}

	got := strings.TrimSpace(string(out))
	if got != npmGlobalRoot {
		t.Errorf("expected NODE_PATH to use `npm root -g` output %q when npm is available, got %q", npmGlobalRoot, got)
	}
}

// shellQuote wraps s in single quotes for embedding as a literal argument inside a
// generated shell script, escaping any embedded single quotes.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
