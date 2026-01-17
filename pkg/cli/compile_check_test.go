package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/githubnext/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/require"
)

func TestCheckLockfileDrift_Clean(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	tmpDir := testutil.TempDir(t, "lockfile-drift-clean")
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".github", "workflows"), 0755))

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(oldWd) })
	require.NoError(t, os.Chdir(tmpDir))

	require.NoError(t, exec.Command("git", "init").Run())
	require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
	require.NoError(t, exec.Command("git", "config", "user.name", "Test").Run())

	lockPath := filepath.Join(tmpDir, ".github", "workflows", "a.lock.yml")
	require.NoError(t, os.WriteFile(lockPath, []byte("name: test\n"), 0644))
	require.NoError(t, exec.Command("git", "add", ".").Run())
	require.NoError(t, exec.Command("git", "commit", "-m", "init").Run())

	require.NoError(t, checkLockfileDrift(".github/workflows"))
}

func TestCheckLockfileDrift_Dirty(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	tmpDir := testutil.TempDir(t, "lockfile-drift-dirty")
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".github", "workflows"), 0755))

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(oldWd) })
	require.NoError(t, os.Chdir(tmpDir))

	require.NoError(t, exec.Command("git", "init").Run())
	require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
	require.NoError(t, exec.Command("git", "config", "user.name", "Test").Run())

	lockPath := filepath.Join(tmpDir, ".github", "workflows", "a.lock.yml")
	require.NoError(t, os.WriteFile(lockPath, []byte("name: test\n"), 0644))
	require.NoError(t, exec.Command("git", "add", ".").Run())
	require.NoError(t, exec.Command("git", "commit", "-m", "init").Run())

	// Modify lockfile to simulate drift
	require.NoError(t, os.WriteFile(lockPath, []byte("name: test\n# changed\n"), 0644))

	err := checkLockfileDrift(".github/workflows")
	require.Error(t, err)
	require.Contains(t, err.Error(), "Lockfile drift detected")
	require.Contains(t, err.Error(), "a.lock.yml")
}
