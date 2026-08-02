//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIsShellcheckableShell tests the shell filter.
func TestIsShellcheckableShell(t *testing.T) {
	t.Run("empty shell defaults to bash", func(t *testing.T) {
		assert.True(t, isShellcheckableShell(""))
	})
	t.Run("bash is checkable", func(t *testing.T) {
		assert.True(t, isShellcheckableShell("bash"))
	})
	t.Run("BASH is checkable (case insensitive)", func(t *testing.T) {
		assert.True(t, isShellcheckableShell("BASH"))
	})
	t.Run("sh is checkable", func(t *testing.T) {
		assert.True(t, isShellcheckableShell("sh"))
	})
	t.Run("pwsh is not checkable", func(t *testing.T) {
		assert.False(t, isShellcheckableShell("pwsh"))
	})
	t.Run("powershell is not checkable", func(t *testing.T) {
		assert.False(t, isShellcheckableShell("powershell"))
	})
	t.Run("python is not checkable", func(t *testing.T) {
		assert.False(t, isShellcheckableShell("python"))
	})
	t.Run("cmd is not checkable", func(t *testing.T) {
		assert.False(t, isShellcheckableShell("cmd"))
	})
}

// TestShellcheckShell verifies the --shell= argument selection.
func TestShellcheckShell(t *testing.T) {
	assert.Equal(t, "bash", shellcheckShell(""))
	assert.Equal(t, "bash", shellcheckShell("bash"))
	assert.Equal(t, "sh", shellcheckShell("sh"))
	// Any other value should fall back to bash.
	assert.Equal(t, "bash", shellcheckShell("zsh"))
}

// TestExtractRunStepsFromLockFile tests YAML parsing and step extraction.
func TestExtractRunStepsFromLockFile(t *testing.T) {
	t.Run("extracts bash and sh steps", func(t *testing.T) {
		content := `
jobs:
  build:
    steps:
      - name: bash step
        run: echo hello
      - name: sh step
        shell: sh
        run: echo sh
      - name: pwsh step
        shell: pwsh
        run: Write-Host hi
      - name: uses step
        uses: actions/checkout@v4
`
		tmpFile := writeTempLockFile(t, content)
		steps, err := extractRunStepsFromLockFile(tmpFile)
		require.NoError(t, err)
		require.Len(t, steps, 2)
		assert.Equal(t, "bash step", steps[0].Name)
		assert.Equal(t, "echo hello", steps[0].Script)
		assert.Empty(t, steps[0].Shell) // no shell field → default bash
		assert.Equal(t, "sh step", steps[1].Name)
		assert.Equal(t, "sh", steps[1].Shell)
	})

	t.Run("returns empty slice when no run steps", func(t *testing.T) {
		content := `
jobs:
  build:
    steps:
      - uses: actions/checkout@v4
`
		tmpFile := writeTempLockFile(t, content)
		steps, err := extractRunStepsFromLockFile(tmpFile)
		require.NoError(t, err)
		assert.Empty(t, steps)
	})

	t.Run("returns empty slice when no jobs", func(t *testing.T) {
		content := `name: empty workflow`
		tmpFile := writeTempLockFile(t, content)
		steps, err := extractRunStepsFromLockFile(tmpFile)
		require.NoError(t, err)
		assert.Empty(t, steps)
	})

	t.Run("handles multiple jobs", func(t *testing.T) {
		content := `
jobs:
  job1:
    steps:
      - name: step1
        run: echo job1
  job2:
    steps:
      - name: step2
        run: echo job2
`
		tmpFile := writeTempLockFile(t, content)
		steps, err := extractRunStepsFromLockFile(tmpFile)
		require.NoError(t, err)
		assert.Len(t, steps, 2)
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		_, err := extractRunStepsFromLockFile("/nonexistent/file.lock.yml")
		require.Error(t, err)
	})

	t.Run("returns error for invalid YAML", func(t *testing.T) {
		tmpFile := writeTempLockFile(t, "{{invalid yaml: [")
		_, err := extractRunStepsFromLockFile(tmpFile)
		require.Error(t, err)
	})
}

// TestStepLabel tests the diagnostic label helper.
func TestStepLabel(t *testing.T) {
	t.Run("includes step name when set", func(t *testing.T) {
		info := runStepInfo{Name: "my step", LockFile: "/a/b/foo.lock.yml"}
		label := stepLabel(info)
		assert.Contains(t, label, "foo.lock.yml")
		assert.Contains(t, label, "my step")
	})
	t.Run("uses lock file basename when name is empty", func(t *testing.T) {
		info := runStepInfo{Name: "", LockFile: "/a/b/foo.lock.yml"}
		label := stepLabel(info)
		assert.Equal(t, "foo.lock.yml", label)
	})
}

// TestDefaultIgnoreCodes verifies the well-known false-positive codes are present.
func TestDefaultIgnoreCodes(t *testing.T) {
	assert.Contains(t, shellcheckDefaultIgnoreCodes, "SC2016")
	assert.Contains(t, shellcheckDefaultIgnoreCodes, "SC1090")
	assert.Contains(t, shellcheckDefaultIgnoreCodes, "SC1091")
}

// TestRunShellcheckOnLockFilesSkipsWhenUnavailable verifies that the function
// returns nil (no error) and does not panic when shellcheck is not in PATH.
// We use a PATH override to hide any real shellcheck binary.
func TestRunShellcheckOnLockFilesSkipsWhenUnavailable(t *testing.T) {
	orig := os.Getenv("PATH")
	t.Cleanup(func() { os.Setenv("PATH", orig) })
	os.Setenv("PATH", "") // ensure shellcheck cannot be found

	err := runShellcheckOnLockFiles([]string{"/fake/file.lock.yml"}, false, false)
	assert.NoError(t, err)
}

// TestRunShellcheckOnLockFilesEmpty returns nil for an empty list.
func TestRunShellcheckOnLockFilesEmpty(t *testing.T) {
	err := runShellcheckOnLockFiles(nil, false, false)
	assert.NoError(t, err)
}

// writeTempLockFile writes content to a temporary *.lock.yml file and returns its path.
func writeTempLockFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.lock.yml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}
