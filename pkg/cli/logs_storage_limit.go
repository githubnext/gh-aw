// This file provides disk-usage limiting for the logs command.
package cli

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/github/gh-aw/pkg/console"
)

const bytesPerMegabyte int64 = 1024 * 1024

var errLogsStorageLimitReached = errors.New("logs storage limit reached")

type logsStorageLimit struct {
	outputDir string
	maxBytes  int64
	// mu guards usedBytes/initialized bookkeeping only. It is held briefly around
	// the pre-download budget check and the post-download usage update, never
	// across the download itself, so concurrent downloads (e.g. from
	// downloadRunArtifactsConcurrent's pool) can still run in parallel; only the
	// shared byte counter is serialized. This means the budget can overshoot by up
	// to the combined size of the in-flight downloads that were admitted before
	// the limit was reached, which is an accepted trade-off for a soft cap.
	mu          sync.Mutex
	reached     atomic.Bool
	initialized bool
	usedBytes   int64
}

func newLogsStorageLimit(outputDir string, maxStorageMB int) *logsStorageLimit {
	if maxStorageMB <= 0 {
		return nil
	}
	return &logsStorageLimit{
		outputDir: outputDir,
		maxBytes:  int64(maxStorageMB) * bytesPerMegabyte,
	}
}

func validateMaxStorageMB(maxStorageMB int) error {
	if maxStorageMB < 0 || int64(maxStorageMB) > math.MaxInt64/bytesPerMegabyte {
		return fmt.Errorf("invalid --max-storage value %d: expected a non-negative number of MB", maxStorageMB)
	}
	return nil
}

func logsDirectorySize(path string) (int64, error) {
	var total int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			if os.IsNotExist(walkErr) {
				return nil
			}
			return walkErr
		}
		if info == nil || !info.Mode().IsRegular() {
			return nil
		}
		if info.Size() > math.MaxInt64-total {
			return errors.New("logs directory size exceeds supported maximum")
		}
		total += info.Size()
		return nil
	})
	if os.IsNotExist(err) {
		return 0, nil
	}
	return total, err
}

func (l *logsStorageLimit) runDownload(ctx context.Context, storagePath string, download func() error) error {
	if l == nil {
		return download()
	}

	select {
	case <-ctx.Done():
		return contextCause(ctx)
	default:
	}

	if err := l.reserve(); err != nil {
		return err
	}

	sizeBefore, err := logsDirectorySize(storagePath)
	if err != nil {
		return fmt.Errorf("failed to measure logs storage path %q: %w", storagePath, err)
	}
	downloadErr := download()
	sizeAfter, sizeErr := logsDirectorySize(storagePath)
	if sizeErr != nil {
		return errors.Join(downloadErr, fmt.Errorf("failed to measure logs storage: %w", sizeErr))
	}
	l.recordUsage(sizeAfter - sizeBefore)
	return downloadErr
}

// reserve checks (and lazily initializes) the shared budget state under a short-lived
// lock. It never holds the lock across the actual download, so concurrent downloads
// keep running in parallel; only the shared usedBytes bookkeeping is serialized.
func (l *logsStorageLimit) reserve() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.reached.Load() {
		return errLogsStorageLimitReached
	}
	if !l.initialized {
		size, err := logsDirectorySize(l.outputDir)
		if err != nil {
			return fmt.Errorf("failed to measure logs storage: %w", err)
		}
		l.usedBytes = size
		l.initialized = true
	}
	if l.usedBytes >= l.maxBytes {
		l.markReached(l.usedBytes)
		return errLogsStorageLimitReached
	}
	return nil
}

// recordUsage applies a completed download's byte delta to the shared counter under
// a short-lived lock and marks the limit reached if the new total meets or exceeds it.
func (l *logsStorageLimit) recordUsage(delta int64) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.usedBytes += delta
	if l.usedBytes >= l.maxBytes {
		l.markReached(l.usedBytes)
	}
}

func (l *logsStorageLimit) markReached(size int64) {
	if !l.reached.CompareAndSwap(false, true) {
		return
	}
	message := fmt.Sprintf(
		"Logs storage limit reached (%s used; maximum %s). Stopping new downloads.",
		console.FormatFileSize(size), console.FormatFileSize(l.maxBytes),
	)
	fmt.Fprintln(os.Stderr, console.FormatWarningMessage(message))
	logsOrchestratorLog.Printf("Logs storage limit reached: used=%d, maximum=%d", size, l.maxBytes)
}

func (l *logsStorageLimit) isReached() bool {
	return l != nil && l.reached.Load()
}
