//go:build !integration

package testutil_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/github/gh-aw/pkg/testutil"
)

// TestSpec_PublicAPI_GetTestRunDir_Idempotent validates that GetTestRunDir
// returns the same path on every call within a process as described in the
// package README.md.
//
// Specification: "GetTestRunDir uses sync.Once so the directory is created
// exactly once per process even when multiple test packages run concurrently."
func TestSpec_PublicAPI_GetTestRunDir_Idempotent(t *testing.T) {
	dir1 := testutil.GetTestRunDir()
	dir2 := testutil.GetTestRunDir()
	dir3 := testutil.GetTestRunDir()

	assert.Equal(t, dir1, dir2,
		"GetTestRunDir should return the same path on every call within a process")
	assert.Equal(t, dir2, dir3,
		"GetTestRunDir should return the same path on every call within a process")
}

// TestSpec_PublicAPI_GetTestRunDir_PathLocation validates that GetTestRunDir
// creates its directory outside the git repository as described in the README.md.
//
// Specification: "It is created once per process under $TMPDIR/gh-aw-test-runs/<timestamp>-<pid>.
// Using a directory outside the repository prevents git commands from
// interfering with test artifacts."
func TestSpec_PublicAPI_GetTestRunDir_PathLocation(t *testing.T) {
	dir := testutil.GetTestRunDir()

	assert.Contains(t, dir, "gh-aw-test-runs",
		"GetTestRunDir path should be under the gh-aw-test-runs directory as documented")

	_, err := os.Stat(dir)
	assert.NoError(t, err,
		"GetTestRunDir should return a path to an existing directory")
}

// TestSpec_PublicAPI_TempDir_CreatesSubdirectory validates that TempDir creates
// a unique subdirectory inside the test run directory as described in the README.md.
//
// Specification: "TempDir creates a temporary subdirectory inside the test run
// directory matching pattern."
func TestSpec_PublicAPI_TempDir_CreatesSubdirectory(t *testing.T) {
	dir := testutil.TempDir(t, "spec-test-*")

	assert.NotEmpty(t, dir, "TempDir should return a non-empty path")

	info, err := os.Stat(dir)
	require.NoError(t, err, "TempDir should create an actual directory on disk")
	assert.True(t, info.IsDir(), "TempDir should create a directory, not a file")

	runDir := testutil.GetTestRunDir()
	assert.True(t, strings.HasPrefix(dir, runDir),
		"TempDir should create a subdirectory inside the test run directory")
}

// TestSpec_PublicAPI_TempDir_CleanupOnTestCompletion validates that the
// directory created by TempDir is removed when the test completes, as described
// in the README.md.
//
// Specification: "The directory is automatically removed when the test
// completes via t.Cleanup."
func TestSpec_PublicAPI_TempDir_CleanupOnTestCompletion(t *testing.T) {
	var capturedPath string

	t.Run("subtest creates TempDir", func(t *testing.T) {
		capturedPath = testutil.TempDir(t, "spec-cleanup-*")

		_, err := os.Stat(capturedPath)
		assert.NoError(t, err,
			"TempDir should create a real directory during the test")
	})

	_, err := os.Stat(capturedPath)
	assert.ErrorIs(t, err, os.ErrNotExist,
		"TempDir directory should be removed automatically after the test completes via t.Cleanup")
}

// TestSpec_PublicAPI_CaptureStderr_ReturnsOutput validates that CaptureStderr
// captures and returns everything written to os.Stderr during fn as described
// in the README.md.
//
// Specification: "CaptureStderr runs fn and returns everything written to
// os.Stderr during its execution."
func TestSpec_PublicAPI_CaptureStderr_ReturnsOutput(t *testing.T) {
	const marker = "spec-test-stderr-marker"

	output := testutil.CaptureStderr(t, func() {
		fmt.Fprint(os.Stderr, marker)
	})

	assert.Contains(t, output, marker,
		"CaptureStderr should return the text written to os.Stderr inside fn")
}

// TestSpec_PublicAPI_CaptureStderr_RestoresAfterCapture validates that
// os.Stderr is restored after CaptureStderr completes, as described in the README.md.
//
// Specification: "os.Stderr is restored automatically via t.Cleanup."
func TestSpec_PublicAPI_CaptureStderr_RestoresAfterCapture(t *testing.T) {
	originalStderr := os.Stderr

	testutil.CaptureStderr(t, func() {
		// Stderr is being captured here; we don't verify the captured fd itself,
		// but we verify restoration after CaptureStderr returns.
	})

	assert.Equal(t, originalStderr, os.Stderr,
		"os.Stderr should be restored to its original value after CaptureStderr returns")
}

// TestSpec_PublicAPI_StripYAMLCommentHeader validates that StripYAMLCommentHeader
// removes the leading comment block and returns only the non-comment content,
// as described in the package README.md.
//
// Specification: "Removes the leading comment block from a generated YAML file
// and returns only the non-comment content."
func TestSpec_PublicAPI_StripYAMLCommentHeader(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "strips leading comment block before YAML content",
			input:    "# Auto-generated by gh-aw\n# Do not edit manually\nruns-on: ubuntu-latest\n",
			expected: "runs-on: ubuntu-latest\n",
		},
		{
			name:     "returns content unchanged when no comment header is present",
			input:    "runs-on: ubuntu-latest\n",
			expected: "runs-on: ubuntu-latest\n",
		},
		{
			name:     "strips multi-line comment block before YAML document separator",
			input:    "# Header\n# More header\n---\njobs:\n  build:\n    runs-on: ubuntu-latest\n",
			expected: "---\njobs:\n  build:\n    runs-on: ubuntu-latest\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testutil.StripYAMLCommentHeader(tt.input)
			assert.Equal(t, tt.expected, result,
				"StripYAMLCommentHeader should remove only the leading comment block")
		})
	}
}
