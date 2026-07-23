//go:build !integration && !windows

package fileutil

import (
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateExecutablePathRejectsNonRegularFile(t *testing.T) {
	pipePath := filepath.Join(t.TempDir(), "tool.pipe")
	require.NoError(t, syscall.Mkfifo(pipePath, 0o755))

	_, err := ValidateExecutablePath(pipePath)
	require.Error(t, err)
	require.ErrorContains(t, err, "is not a regular file")
}
