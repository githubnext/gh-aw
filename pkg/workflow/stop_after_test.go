package workflow

import (
	"os"
	"path/filepath"
	"testing"
)

// TestExtractStopTimeFromLockFile tests the ExtractStopTimeFromLockFile function
func TestExtractStopTimeFromLockFile(t *testing.T) {
	tests := []struct {
		name         string
		lockContent  string
		expectedTime string
	}{
		{
			name: "valid stop-time in lock file",
			lockContent: `name: Test Workflow
on:
  workflow_dispatch:
jobs:
  safety_checks:
    runs-on: ubuntu-latest
    steps:
      - name: Safety checks
        run: |
          STOP_TIME="2025-12-31 23:59:59"
          echo "Checking stop-time limit: $STOP_TIME"`,
			expectedTime: "2025-12-31 23:59:59",
		},
		{
			name: "no stop-time in lock file",
			lockContent: `name: Test Workflow
on:
  workflow_dispatch:
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Test step
        run: echo "No stop time here"`,
			expectedTime: "",
		},
		{
			name: "malformed stop-time line",
			lockContent: `name: Test Workflow
on:
  workflow_dispatch:
jobs:
  safety_checks:
    runs-on: ubuntu-latest
    steps:
      - name: Safety checks
        run: |
          STOP_TIME=malformed-no-quotes`,
			expectedTime: "",
		},
		{
			name: "multiple stop-time lines (should get first)",
			lockContent: `name: Test Workflow
on:
  workflow_dispatch:
jobs:
  safety_checks:
    runs-on: ubuntu-latest
    steps:
      - name: Safety checks
        run: |
          STOP_TIME="2025-06-01 12:00:00"
          echo "Checking stop-time limit: $STOP_TIME"
          STOP_TIME="2025-07-01 12:00:00"`,
			expectedTime: "2025-06-01 12:00:00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpDir, err := os.MkdirTemp("", "lock-file-test")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			lockFile := filepath.Join(tmpDir, "test.lock.yml")
			err = os.WriteFile(lockFile, []byte(tt.lockContent), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Test extraction
			result := ExtractStopTimeFromLockFile(lockFile)
			if result != tt.expectedTime {
				t.Errorf("ExtractStopTimeFromLockFile() = %q, want %q", result, tt.expectedTime)
			}
		})
	}

	// Test non-existent file
	t.Run("non-existent file", func(t *testing.T) {
		result := ExtractStopTimeFromLockFile("/non/existent/file.lock.yml")
		if result != "" {
			t.Errorf("ExtractStopTimeFromLockFile() for non-existent file = %q, want empty string", result)
		}
	})
}