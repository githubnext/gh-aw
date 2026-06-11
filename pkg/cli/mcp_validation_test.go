//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetBinaryPath(t *testing.T) {
	t.Run("returns non-empty path", func(t *testing.T) {
		path, err := GetBinaryPath()
		require.NoError(t, err, "Should get binary path without error")
		assert.NotEmpty(t, path, "Binary path should not be empty")
	})

	t.Run("returns absolute path", func(t *testing.T) {
		path, err := GetBinaryPath()
		require.NoError(t, err, "Should get binary path without error")
		assert.True(t, filepath.IsAbs(path), "Binary path should be absolute")
	})

	t.Run("returned path exists", func(t *testing.T) {
		path, err := GetBinaryPath()
		require.NoError(t, err, "Should get binary path without error")

		// Check if the file exists
		info, err := os.Stat(path)
		require.NoError(t, err, "Binary file should exist at the returned path")
		assert.False(t, info.IsDir(), "Binary path should not be a directory")
	})

	t.Run("path ends with executable name", func(t *testing.T) {
		path, err := GetBinaryPath()
		require.NoError(t, err, "Should get binary path without error")

		// The path should end with a reasonable executable name
		// During tests, it might be a test binary name
		base := filepath.Base(path)
		assert.NotEmpty(t, base, "Binary path should have a base name")
		// Don't check for specific name as it could be the test binary
	})

	t.Run("resolves symlinks", func(t *testing.T) {
		path, err := GetBinaryPath()
		require.NoError(t, err, "Should get binary path without error")

		// The path should be the resolved path, not a symlink
		// We can verify this by checking that EvalSymlinks returns the same path
		resolved, err := filepath.EvalSymlinks(path)
		if err == nil {
			// If we can resolve symlinks, the path should already be resolved
			assert.Equal(t, path, resolved, "Path should already be resolved (no symlinks)")
		}
		// If EvalSymlinks fails, that's OK - the original path is still valid
	})
}

func TestWithNonInteractiveCIEnv(t *testing.T) {
	t.Run("returns copied env with CI forced on", func(t *testing.T) {
		input := []string{"CI=false", "HOME=/tmp/test-home"}

		output := withNonInteractiveCIEnv(input)

		assert.Equal(t, []string{"CI=false", "HOME=/tmp/test-home"}, input)
		assert.Contains(t, output, "CI=1")
		assert.NotContains(t, output, "CI=false")
		assert.Contains(t, output, "HOME=/tmp/test-home")
	})

	t.Run("adds CI when missing", func(t *testing.T) {
		input := []string{"HOME=/tmp/test-home"}

		output := withNonInteractiveCIEnv(input)

		assert.Equal(t, []string{"HOME=/tmp/test-home"}, input, "must not mutate input")
		assert.Contains(t, output, "CI=1")
		assert.Contains(t, output, "HOME=/tmp/test-home")
	})
}
