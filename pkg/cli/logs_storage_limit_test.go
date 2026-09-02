//go:build !integration

package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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

// TestLogsStorageLimitConcurrentDownloadsRunInParallel guards against a regression
// where the storage limiter's internal lock was held across the entire download()
// call, silently serializing every download whenever --max-storage was configured
// (even though the concurrent download pool itself allowed parallelism). Run with
// `go test -race` to also catch data races on the shared usedBytes/reached state.
func TestLogsStorageLimitConcurrentDownloadsRunInParallel(t *testing.T) {
	outputDir := t.TempDir()
	limit := newLogsStorageLimit(outputDir, 10240) // large budget: never reached

	const numDownloads = 8
	var inFlight atomic.Int32
	var maxObservedConcurrency atomic.Int32
	release := make(chan struct{})

	var wg sync.WaitGroup
	for i := range numDownloads {
		wg.Go(func() {
			runDir := filepath.Join(outputDir, fmt.Sprintf("run-%d", i))
			err := limit.runDownload(context.Background(), runDir, func() error {
				current := inFlight.Add(1)
				defer inFlight.Add(-1)
				for {
					observedMax := maxObservedConcurrency.Load()
					if current <= observedMax || maxObservedConcurrency.CompareAndSwap(observedMax, current) {
						break
					}
				}
				// Block until every goroutine has entered its download closure so
				// we can observe genuine overlap rather than accidental interleaving.
				<-release
				return os.MkdirAll(runDir, 0o755)
			})
			assert.NoError(t, err)
		})
	}

	require.Eventually(t, func() bool { return inFlight.Load() == numDownloads }, 2*time.Second, time.Millisecond,
		"expected all downloads to run concurrently, not serialized by the storage limiter")
	close(release)
	wg.Wait()

	assert.EqualValues(t, numDownloads, maxObservedConcurrency.Load(), "storage limiter must not serialize concurrent downloads")
}
