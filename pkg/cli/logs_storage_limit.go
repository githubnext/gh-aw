// This file provides disk-usage limiting for the logs command.
package cli

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/github/gh-aw/pkg/console"
)

const bytesPerMegabyte int64 = 1024 * 1024

var errLogsStorageLimitReached = errors.New("logs storage limit reached")

type logsStorageLimit struct {
	outputDir   string
	maxBytes    int64
	gate        chan struct{}
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
		gate:      make(chan struct{}, 1),
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
	case l.gate <- struct{}{}:
		defer func() { <-l.gate }()
	case <-ctx.Done():
		return contextCause(ctx)
	}

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

	sizeBefore, err := logsDirectorySize(storagePath)
	if err != nil {
		return fmt.Errorf("failed to measure logs storage path %q: %w", storagePath, err)
	}
	downloadErr := download()
	sizeAfter, sizeErr := logsDirectorySize(storagePath)
	if sizeErr != nil {
		return errors.Join(downloadErr, fmt.Errorf("failed to measure logs storage: %w", sizeErr))
	}
	l.usedBytes += sizeAfter - sizeBefore
	if l.usedBytes >= l.maxBytes {
		l.markReached(l.usedBytes)
	}
	return downloadErr
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
