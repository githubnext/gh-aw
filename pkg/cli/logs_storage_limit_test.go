//go:build !integration

package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogsDirectorySize(t *testing.T) {
	outputDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(outputDir, "run-1", "usage"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(outputDir, "run-1", "usage", "a.json"), make([]byte, 128), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(outputDir, "summary.json"), make([]byte, 64), 0o644))

	size, err := logsDirectorySize(outputDir)

	require.NoError(t, err)
	assert.Equal(t, int64(192), size)
}

func TestLogsDirectorySizeMissingDirectory(t *testing.T) {
	size, err := logsDirectorySize(filepath.Join(t.TempDir(), "missing"))

	require.NoError(t, err)
	assert.Zero(t, size)
}

func TestLogsStorageLimitStopsNewDownloadsAtExistingLimit(t *testing.T) {
	outputDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(outputDir, "existing.bin"), make([]byte, bytesPerMegabyte), 0o644))
	limit := newLogsStorageLimit(outputDir, 1)
	called := false

	err := limit.runDownload(context.Background(), filepath.Join(outputDir, "run-1"), func() error {
		called = true
		return nil
	})

	require.ErrorIs(t, err, errLogsStorageLimitReached)
	assert.False(t, called)
	assert.True(t, limit.isReached())
}

func TestLogsStorageLimitAllowsFinalDownloadThenStops(t *testing.T) {
	outputDir := t.TempDir()
	limit := newLogsStorageLimit(outputDir, 1)

	runDir := filepath.Join(outputDir, "run-1")
	err := limit.runDownload(context.Background(), runDir, func() error {
		require.NoError(t, os.MkdirAll(runDir, 0o755))
		return os.WriteFile(filepath.Join(runDir, "download.bin"), make([]byte, bytesPerMegabyte), 0o644)
	})

	require.NoError(t, err)
	assert.True(t, limit.isReached())
	secondCalled := false
	err = limit.runDownload(context.Background(), filepath.Join(outputDir, "run-2"), func() error {
		secondCalled = true
		return nil
	})
	require.ErrorIs(t, err, errLogsStorageLimitReached)
	assert.False(t, secondCalled)
}

func TestLogsStorageLimitDisabled(t *testing.T) {
	var limit *logsStorageLimit
	expected := errors.New("download failed")

	err := limit.runDownload(context.Background(), "", func() error {
		return expected
	})

	require.ErrorIs(t, err, expected)
	assert.False(t, limit.isReached())
}

func TestValidateMaxStorageMB(t *testing.T) {
	assert.NoError(t, validateMaxStorageMB(0))
	assert.NoError(t, validateMaxStorageMB(10240))
	assert.Error(t, validateMaxStorageMB(-1))
}
