// This file provides disk-usage limiting for the logs command.
package cli

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
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
	mu        sync.Mutex
	reached   atomic.Bool
	initErr   error
	usedBytes int64
	completed map[string]struct{}
}

type logsFolderSize struct {
	name string
	size int64
}

type logsPruneCandidate struct {
	path string
	size int64
}

var prunableLogsCacheDirectories = map[string]struct{}{
	"agent":         {},
	"aw-mcp":        {},
	"mcp-logs":      {},
	"mcp-scripts":   {},
	"pi-agent-dir":  {},
	"proxy-logs":    {},
	"sandbox":       {},
	"workflow-logs": {},
}

var essentialLogsCacheFiles = map[string]struct{}{
	constants.AgentOutputFilename.String():      {},
	constants.TokenUsageFilename.String():       {},
	"aw_info.json":                              {},
	constants.EvalsResultFilename.String():      {},
	constants.GithubRateLimitsFilename.String(): {},
	constants.GraderManifestFilename.String():   {},
	constants.GraderResultsFilename.String():    {},
	jobsAPIResponseFileName:                     {},
	runSummaryFileName:                          {},
	"safe-output-items.jsonl":                   {},
	constants.SafeOutputsFilename.String():      {},
	constants.TemporaryIdMapFilename.String():   {},
}

func newLogsStorageLimit(outputDir string, maxStorageMB int) *logsStorageLimit {
	if maxStorageMB <= 0 {
		return nil
	}
	limit := &logsStorageLimit{
		outputDir: outputDir,
		maxBytes:  int64(maxStorageMB) * bytesPerMegabyte,
		completed: make(map[string]struct{}),
	}
	limit.initErr = limit.initialize()
	return limit
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

func logsFolderSizes(path string) ([]logsFolderSize, error) {
	entries, err := os.ReadDir(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	folders := make([]logsFolderSize, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		size, err := logsDirectorySize(filepath.Join(path, entry.Name()))
		if err != nil {
			return nil, err
		}
		folders = append(folders, logsFolderSize{name: entry.Name(), size: size})
	}
	slices.SortFunc(folders, func(a, b logsFolderSize) int {
		if a.size == b.size {
			return cmp.Compare(a.name, b.name)
		}
		return cmp.Compare(b.size, a.size)
	})
	return folders, nil
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
	pruneErr := l.recordUsage(storagePath, sizeAfter-sizeBefore)
	return errors.Join(downloadErr, pruneErr)
}

// reserve checks (and lazily initializes) the shared budget state under a short-lived
// lock. It never holds the lock across the actual download, so concurrent downloads
// keep running in parallel; only the shared usedBytes bookkeeping is serialized.
func (l *logsStorageLimit) reserve() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.initErr != nil {
		return l.initErr
	}
	if l.reached.Load() {
		return errLogsStorageLimitReached
	}
	if l.usedBytes >= l.maxBytes {
		l.markReached(l.usedBytes)
		return errLogsStorageLimitReached
	}
	return nil
}

func (l *logsStorageLimit) initialize() error {
	size, err := logsDirectorySize(l.outputDir)
	if err != nil {
		return fmt.Errorf("failed to measure logs storage: %w", err)
	}
	folders, err := logsFolderSizes(l.outputDir)
	if err != nil {
		return fmt.Errorf("failed to measure logs cache folders: %w", err)
	}
	l.reportStartingUsage(size, folders)
	l.usedBytes = size
	for _, folder := range folders {
		l.completed[filepath.Join(l.outputDir, folder.name)] = struct{}{}
	}
	if l.usedBytes >= l.maxBytes {
		freed, err := pruneLogsCache(l.outputDir, l.usedBytes-l.maxBytes+1, nil)
		if err != nil {
			return fmt.Errorf("failed to prune logs cache: %w", err)
		}
		l.recordPrunedUsage(freed)
	}
	if l.usedBytes >= l.maxBytes {
		l.markReached(l.usedBytes)
	}
	return nil
}

// recordUsage applies a completed download's byte delta and selectively removes
// non-essential agent data when the cache would otherwise exceed the budget.
func (l *logsStorageLimit) recordUsage(storagePath string, delta int64) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.usedBytes += delta
	l.completed[filepath.Clean(storagePath)] = struct{}{}
	if l.usedBytes >= l.maxBytes {
		freed, err := pruneLogsCache(l.outputDir, l.usedBytes-l.maxBytes+1, l.completed)
		if err != nil {
			return fmt.Errorf("failed to prune logs cache: %w", err)
		}
		l.recordPrunedUsage(freed)
	}
	if l.usedBytes >= l.maxBytes {
		l.markReached(l.usedBytes)
	}
	return nil
}

func (l *logsStorageLimit) reportStartingUsage(size int64, folders []logsFolderSize) {
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf(
		"Logs cache starting size: %s (maximum %s)",
		console.FormatFileSize(size), console.FormatFileSize(l.maxBytes),
	)))
	logsOrchestratorLog.Printf("Logs cache starting size: used=%d maximum=%d", size, l.maxBytes)
	for _, folder := range folders {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf(
			"Logs cache folder %s: %s",
			strconv.Quote(folder.name), console.FormatFileSize(folder.size),
		)))
		logsOrchestratorLog.Printf("Logs cache folder size: folder=%q size=%d", folder.name, folder.size)
	}
}

func (l *logsStorageLimit) recordPrunedUsage(freed int64) {
	if freed <= 0 {
		return
	}
	l.usedBytes -= freed
	if l.usedBytes < 0 {
		l.usedBytes = 0
	}
	if l.usedBytes < l.maxBytes {
		l.reached.Store(false)
	}
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf(
		"Pruned %s of non-essential logs cache data", console.FormatFileSize(freed),
	)))
	logsOrchestratorLog.Printf("Pruned non-essential logs cache data: freed=%d remaining=%d", freed, l.usedBytes)
}

func pruneLogsCache(path string, bytesToFree int64, completedPaths map[string]struct{}) (int64, error) {
	if bytesToFree <= 0 {
		return 0, nil
	}

	var candidates []logsPruneCandidate
	err := filepath.Walk(path, func(candidatePath string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			if os.IsNotExist(walkErr) {
				return nil
			}
			return walkErr
		}
		if info == nil || !info.Mode().IsRegular() {
			return nil
		}
		if !isInCompletedLogsCachePath(candidatePath, completedPaths) {
			return nil
		}
		relativePath, err := filepath.Rel(path, candidatePath)
		if err != nil {
			return err
		}
		if isPrunableLogsCacheFile(relativePath) {
			candidates = append(candidates, logsPruneCandidate{path: candidatePath, size: info.Size()})
		}
		return nil
	})
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	slices.SortFunc(candidates, func(a, b logsPruneCandidate) int {
		if a.size == b.size {
			return cmp.Compare(a.path, b.path)
		}
		return cmp.Compare(b.size, a.size)
	})
	var freed int64
	for _, candidate := range candidates {
		if freed >= bytesToFree {
			break
		}
		if err := os.Remove(candidate.path); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return freed, err
		}
		freed += candidate.size
		if err := invalidatePrunedArtifactMarkers(candidate.path, path); err != nil {
			return freed, err
		}
		removeEmptyLogsCacheParents(filepath.Dir(candidate.path), path)
	}
	return freed, nil
}

func isInCompletedLogsCachePath(path string, completedPaths map[string]struct{}) bool {
	if len(completedPaths) == 0 {
		return true
	}
	for completedPath := range completedPaths {
		relativePath, err := filepath.Rel(completedPath, path)
		if err == nil && relativePath != ".." && !filepath.IsAbs(relativePath) &&
			!strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func isPrunableLogsCacheFile(relativePath string) bool {
	base := filepath.Base(relativePath)
	if _, essential := essentialLogsCacheFiles[base]; essential {
		return false
	}
	parts := splitLogsCachePath(relativePath)
	for _, part := range parts {
		if _, prunable := prunableLogsCacheDirectories[part]; prunable {
			return true
		}
	}
	return false
}

func invalidatePrunedArtifactMarkers(prunedPath, root string) error {
	root = filepath.Clean(root)
	for dir := filepath.Dir(prunedPath); ; dir = filepath.Dir(dir) {
		markerDir := filepath.Join(dir, downloadedArtifactsMarkerDir)
		entries, err := os.ReadDir(markerDir)
		if err == nil {
			removed := false
			for _, entry := range entries {
				if entry.IsDir() || !isAgentArtifactMarker(entry.Name()) {
					continue
				}
				if err := os.Remove(filepath.Join(markerDir, entry.Name())); err != nil && !os.IsNotExist(err) {
					return err
				}
				removed = true
			}
			if removed {
				return nil
			}
		} else if !os.IsNotExist(err) {
			return err
		}
		if dir == root || dir == "." || dir == filepath.Dir(dir) {
			return nil
		}
	}
}

func isAgentArtifactMarker(name string) bool {
	return name == string(ArtifactSetAll) ||
		artifactNameMatchesBase(name, constants.AgentArtifactName.String()) ||
		artifactNameMatchesBase(name, constants.AgentOutputFallbackArtifactName.String())
}

func splitLogsCachePath(path string) []string {
	var parts []string
	for path != "." && path != string(filepath.Separator) {
		dir, base := filepath.Split(path)
		if base != "" {
			parts = append(parts, base)
		}
		path = filepath.Clean(dir)
	}
	return parts
}

func removeEmptyLogsCacheParents(path, root string) {
	root = filepath.Clean(root)
	for path = filepath.Clean(path); path != root && path != "."; path = filepath.Dir(path) {
		if err := os.Remove(path); err != nil {
			return
		}
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
