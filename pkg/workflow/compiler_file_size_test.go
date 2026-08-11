//go:build integration

package workflow

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/github/gh-aw/pkg/stringutil"

	"github.com/github/gh-aw/pkg/testutil"
)

// TestMaxLockFileSizeConstant verifies the documented value of the size limit
// constant (500KB) used to warn about oversized generated lock files.
func TestMaxLockFileSizeConstant(t *testing.T) {
	assert.EqualValues(t, 512000, MaxLockFileSize, "MaxLockFileSize should be 500KB (512000 bytes)")
}

func TestCompileWorkflowFileSizeValidation(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "file-size-test")

	t.Run("workflow under 500KB should compile successfully", func(t *testing.T) {
		// Create a normal workflow that should be well under 500KB
		testContent := `---
on: push
timeout-minutes: 10
permissions:
  contents: read
  issues: read
  pull-requests: read
strict: false
tools:
  github:
    allowed: [list_issues, create_issue]
---

# Normal Test Workflow

This is a normal workflow that should compile successfully.
`

		testFile := filepath.Join(tmpDir, "normal-workflow.md")
		require.NoError(t, os.WriteFile(testFile, []byte(testContent), 0644))

		compiler := NewCompiler()
		err := compiler.CompileWorkflow(testFile)
		require.NoError(t, err, "expected no error for normal workflow")

		// Verify lock file was created and is under 500KB
		lockFile := stringutil.MarkdownToLockFile(testFile)
		info, err := os.Stat(lockFile)
		require.NoError(t, err, "lock file was not created")
		assert.LessOrEqual(t, info.Size(), int64(MaxLockFileSize), "lock file size should not exceed max size")
	})

	t.Run("noEmit mode skips size validation and does not write lock file", func(t *testing.T) {
		testContent := `---
on: push
timeout-minutes: 10
permissions:
  contents: read
strict: false
---

# No-Emit Test Workflow

This workflow validates that noEmit mode does not write or size-check a lock file.
`

		testFile := filepath.Join(tmpDir, "no-emit-workflow.md")
		require.NoError(t, os.WriteFile(testFile, []byte(testContent), 0644))

		lockFile := stringutil.MarkdownToLockFile(testFile)
		oversizedContent := strings.Repeat("x", MaxLockFileSize+1)
		require.NoError(t, os.WriteFile(lockFile, []byte(oversizedContent), 0644))

		compiler := NewCompiler()
		compiler.SetNoEmit(true)

		stderrOutput := captureStderr(t, func() {
			err := compiler.CompileWorkflow(testFile)
			require.NoError(t, err, "expected no error for noEmit compilation")
		})

		lockFileContent, err := os.ReadFile(lockFile)
		require.NoError(t, err, "pre-existing lock file should not be removed in noEmit mode")
		assert.Equal(t, oversizedContent, string(lockFileContent), "noEmit mode should not write the lock file")
		assert.NotContains(t, stderrOutput, "exceeds recommended maximum size", "noEmit mode should not perform size validation")
	})

	t.Run("writeWorkflowOutput warns when generated content exceeds MaxLockFileSize", func(t *testing.T) {
		markdownPath := filepath.Join(tmpDir, "over-limit-source.md")
		// mockLockFile is a synthetic lock file path used to exercise writeWorkflowOutput
		// directly; it is not produced by an actual compile of markdownPath.
		mockLockFile := stringutil.MarkdownToLockFile(markdownPath)
		t.Cleanup(func() { _ = os.Remove(mockLockFile) })

		oversizedContent := strings.Repeat("x", MaxLockFileSize+100000) // 100KB over the limit

		compiler := NewCompiler()
		stderrOutput := captureStderr(t, func() {
			err := compiler.writeWorkflowOutput(mockLockFile, oversizedContent, markdownPath)
			require.NoError(t, err, "writeWorkflowOutput should not error even when the file exceeds the recommended size")
		})

		info, err := os.Stat(mockLockFile)
		require.NoError(t, err, "lock file should have been written")
		require.Greater(t, info.Size(), int64(MaxLockFileSize), "mock file size should exceed limit")

		assert.Contains(t, stderrOutput, "exceeds recommended maximum size", "should emit size-exceeded warning")
		assert.Contains(t, stderrOutput, "KB", "warning message should contain size in KB")
	})

	t.Run("writeWorkflowOutput does not warn at or under MaxLockFileSize boundary", func(t *testing.T) {
		tests := []struct {
			name string
			size int
		}{
			{name: "exactly at MaxLockFileSize", size: MaxLockFileSize},
			{name: "one byte under MaxLockFileSize", size: MaxLockFileSize - 1},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				safeName := strings.ReplaceAll(tt.name, " ", "-")
				markdownPath := filepath.Join(tmpDir, "boundary-"+safeName+"-source.md")
				mockLockFile := stringutil.MarkdownToLockFile(markdownPath)
				t.Cleanup(func() { _ = os.Remove(mockLockFile) })

				content := strings.Repeat("x", tt.size)

				compiler := NewCompiler()
				stderrOutput := captureStderr(t, func() {
					err := compiler.writeWorkflowOutput(mockLockFile, content, markdownPath)
					require.NoError(t, err)
				})

				info, err := os.Stat(mockLockFile)
				require.NoError(t, err, "lock file should have been written")
				assert.LessOrEqual(t, info.Size(), int64(MaxLockFileSize))
				assert.NotContains(t, stderrOutput, "exceeds recommended maximum size", "should not warn when size is at or under the limit")
			})
		}
	})
}

// captureStderr redirects os.Stderr during fn and returns everything written to it.
// os.Stderr is always restored, even if fn panics or fails a require assertion.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()

	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err, "should create stderr capture pipe")
	defer func() { _ = r.Close() }()
	os.Stderr = w
	defer func() {
		os.Stderr = oldStderr
		_ = w.Close()
	}()

	fn()

	require.NoError(t, w.Close(), "should close stderr capture writer")

	var buf bytes.Buffer
	_, copyErr := io.Copy(&buf, r)
	require.NoError(t, copyErr, "should copy stderr output")
	return buf.String()
}
