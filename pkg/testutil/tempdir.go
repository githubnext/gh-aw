package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

var (
	testRunDir     string
	testRunDirOnce sync.Once
)

// GetTestRunDir returns the unique directory for this test run.
// It creates a "test-runs" directory in the repo root with a unique subdirectory
// based on the current timestamp and process ID.
func GetTestRunDir() string {
	testRunDirOnce.Do(func() {
		// Get the repository root (assuming we're in pkg/testutil)
		wd, err := os.Getwd()
		if err != nil {
			panic(fmt.Sprintf("failed to get working directory: %v", err))
		}

		// Navigate up to find the repo root (contains go.mod)
		repoRoot := wd
		for {
			if _, err := os.Stat(filepath.Join(repoRoot, "go.mod")); err == nil {
				break
			}
			parent := filepath.Dir(repoRoot)
			if parent == repoRoot {
				// Reached filesystem root without finding go.mod
				panic("failed to find repository root (go.mod)")
			}
			repoRoot = parent
		}

		// Create test-runs directory in repo root
		testRunsDir := filepath.Join(repoRoot, "test-runs")
		if err := os.MkdirAll(testRunsDir, 0755); err != nil {
			panic(fmt.Sprintf("failed to create test-runs directory: %v", err))
		}

		// Create unique subdirectory for this test run
		timestamp := time.Now().Format("20060102-150405")
		pid := os.Getpid()
		testRunDir = filepath.Join(testRunsDir, fmt.Sprintf("%s-%d", timestamp, pid))

		if err := os.MkdirAll(testRunDir, 0755); err != nil {
			panic(fmt.Sprintf("failed to create test run directory: %v", err))
		}
	})

	return testRunDir
}

// TempDir creates a temporary directory for testing within the test run directory.
// It automatically cleans up the directory when the test completes.
// This replaces the use of os.MkdirTemp or t.TempDir() to ensure all test
// artifacts are isolated in a known location.
func TempDir(t *testing.T, pattern string) string {
	t.Helper()

	baseDir := GetTestRunDir()

	// Create a unique subdirectory within the test run directory
	tempDir, err := os.MkdirTemp(baseDir, pattern)
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}

	// Register cleanup to remove the directory after test completes
	t.Cleanup(func() {
		os.RemoveAll(tempDir)
	})

	return tempDir
}
