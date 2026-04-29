//go:build !integration

package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeRunDir creates a run-{id} directory with an optional run_summary.json.
func makeRunDir(t *testing.T, parent string, id int64, createdAt time.Time, writeSummary bool) string {
	t.Helper()
	dir := filepath.Join(parent, "run-"+strconv.FormatInt(id, 10))
	require.NoError(t, os.MkdirAll(dir, 0755), "create run dir")

	if writeSummary {
		summary := RunSummary{
			CLIVersion:  "test",
			RunID:       id,
			ProcessedAt: time.Now(),
			Run: WorkflowRun{
				DatabaseID: id,
				CreatedAt:  createdAt,
			},
		}
		data, err := json.Marshal(summary)
		require.NoError(t, err, "marshal run summary")
		require.NoError(t, os.WriteFile(filepath.Join(dir, runSummaryFileName), data, 0644), "write run summary")
	}

	return dir
}

func TestCleanupOldRunFolders(t *testing.T) {
	now := time.Now()
	cutoff := now.Add(-7 * 24 * time.Hour) // 1 week ago

	tests := []struct {
		name            string
		setup           func(t *testing.T, dir string)
		wantRemoved     int
		wantDirsLeft    []string
		wantDirsRemoved []string
	}{
		{
			name: "removes folders older than cutoff",
			setup: func(t *testing.T, dir string) {
				makeRunDir(t, dir, 1, now.Add(-14*24*time.Hour), true) // 2 weeks old - should be removed
				makeRunDir(t, dir, 2, now.Add(-3*24*time.Hour), true)  // 3 days old - should be kept
			},
			wantRemoved:     1,
			wantDirsLeft:    []string{"run-2"},
			wantDirsRemoved: []string{"run-1"},
		},
		{
			name: "keeps folders newer than cutoff",
			setup: func(t *testing.T, dir string) {
				makeRunDir(t, dir, 10, now.Add(-1*24*time.Hour), true) // 1 day old - kept
				makeRunDir(t, dir, 11, now.Add(-2*24*time.Hour), true) // 2 days old - kept
			},
			wantRemoved:  0,
			wantDirsLeft: []string{"run-10", "run-11"},
		},
		{
			name: "removes all old folders",
			setup: func(t *testing.T, dir string) {
				makeRunDir(t, dir, 20, now.Add(-30*24*time.Hour), true) // 30 days old - removed
				makeRunDir(t, dir, 21, now.Add(-14*24*time.Hour), true) // 14 days old - removed
			},
			wantRemoved:     2,
			wantDirsRemoved: []string{"run-20", "run-21"},
		},
		{
			name: "ignores non run- directories",
			setup: func(t *testing.T, dir string) {
				// A directory not following the run-{ID} pattern should not be touched
				nonRunDir := filepath.Join(dir, "summary")
				require.NoError(t, os.MkdirAll(nonRunDir, 0755))

				makeRunDir(t, dir, 30, now.Add(-30*24*time.Hour), true) // old - removed
			},
			wantRemoved:     1,
			wantDirsLeft:    []string{"summary"},
			wantDirsRemoved: []string{"run-30"},
		},
		{
			name: "ignores run- directories with non-integer suffix",
			setup: func(t *testing.T, dir string) {
				// Directories like "run-backup" or "run-temp" must not be removed
				for _, name := range []string{"run-backup", "run-temp", "run-old"} {
					require.NoError(t, os.MkdirAll(filepath.Join(dir, name), 0755))
				}
				makeRunDir(t, dir, 50, now.Add(-30*24*time.Hour), true) // old with numeric ID - removed
			},
			wantRemoved:     1,
			wantDirsLeft:    []string{"run-backup", "run-temp", "run-old"},
			wantDirsRemoved: []string{"run-50"},
		},
		{
			name: "falls back to dir mtime when no run_summary.json",
			setup: func(t *testing.T, dir string) {
				// Create a run dir without a summary file; its mtime will be recent
				makeRunDir(t, dir, 40, time.Time{}, false)              // no summary; mtime is now
				makeRunDir(t, dir, 41, now.Add(-30*24*time.Hour), true) // old with summary - removed
			},
			wantRemoved:     1,
			wantDirsLeft:    []string{"run-40"},
			wantDirsRemoved: []string{"run-41"},
		},
		{
			name: "empty output directory returns zero removed",
			setup: func(t *testing.T, dir string) {
				// nothing to do
			},
			wantRemoved: 0,
		},
		{
			name: "non-existent output directory returns zero removed",
			setup: func(t *testing.T, dir string) {
				// Remove the directory so it doesn't exist
				require.NoError(t, os.RemoveAll(dir))
			},
			wantRemoved: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tt.setup(t, tmpDir)

			removed, err := cleanupOldRunFolders(tmpDir, cutoff, false)

			require.NoError(t, err, "cleanupOldRunFolders should not return an error")
			assert.Equal(t, tt.wantRemoved, removed, "number of removed folders should match")

			for _, name := range tt.wantDirsLeft {
				assert.DirExists(t, filepath.Join(tmpDir, name), "directory should still exist: %s", name)
			}
			for _, name := range tt.wantDirsRemoved {
				assert.NoDirExists(t, filepath.Join(tmpDir, name), "directory should have been removed: %s", name)
			}
		})
	}
}

func TestCleanupOldRunFoldersVerbose(t *testing.T) {
	now := time.Now()
	cutoff := now.Add(-7 * 24 * time.Hour)
	tmpDir := t.TempDir()

	makeRunDir(t, tmpDir, 99, now.Add(-30*24*time.Hour), true)

	// Should work identically in verbose mode without panicking
	removed, err := cleanupOldRunFolders(tmpDir, cutoff, true)
	require.NoError(t, err, "verbose cleanup should not error")
	assert.Equal(t, 1, removed, "one folder should be removed in verbose mode")
}
