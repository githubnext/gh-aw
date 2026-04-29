//go:build !integration

package fileutil

import (
	"archive/tar"
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateAbsolutePath(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "valid absolute Unix path",
			path:        "/home/user/file.txt",
			shouldError: false,
		},
		{
			name:        "valid absolute path with cleaned components",
			path:        "/home/user/../user/file.txt",
			shouldError: false,
		},
		{
			name:        "empty path",
			path:        "",
			shouldError: true,
			errorMsg:    "path cannot be empty",
		},
		{
			name:        "relative path",
			path:        "relative/path.txt",
			shouldError: true,
			errorMsg:    "path must be absolute",
		},
		{
			name:        "relative path with dot",
			path:        "./file.txt",
			shouldError: true,
			errorMsg:    "path must be absolute",
		},
		{
			name:        "relative path with double dot",
			path:        "../file.txt",
			shouldError: true,
			errorMsg:    "path must be absolute",
		},
		{
			name:        "path traversal attempt",
			path:        "../../../etc/passwd",
			shouldError: true,
			errorMsg:    "path must be absolute",
		},
		{
			name:        "single dot",
			path:        ".",
			shouldError: true,
			errorMsg:    "path must be absolute",
		},
		{
			name:        "double dot",
			path:        "..",
			shouldError: true,
			errorMsg:    "path must be absolute",
		},
	}

	// Add Windows-specific tests only on Windows
	if runtime.GOOS == "windows" {
		tests = append(tests, []struct {
			name        string
			path        string
			shouldError bool
			errorMsg    string
		}{
			{
				name:        "valid absolute Windows path",
				path:        "C:\\Users\\user\\file.txt",
				shouldError: false,
			},
			{
				name:        "valid absolute Windows UNC path",
				path:        "\\\\server\\share\\file.txt",
				shouldError: false,
			},
		}...)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateAbsolutePath(tt.path)

			if tt.shouldError {
				require.Error(t, err, "Expected error for path: %s", tt.path)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg, "Error message should contain expected text")
				}
				assert.Empty(t, result, "Result should be empty on error")
			} else {
				require.NoError(t, err, "Should not error for valid absolute path: %s", tt.path)
				assert.NotEmpty(t, result, "Result should not be empty")
				assert.True(t, filepath.IsAbs(result), "Result should be an absolute path: %s", result)
				// Verify path is cleaned (no .. components)
				assert.NotContains(t, result, "..", "Cleaned path should not contain .. components")
			}
		})
	}
}

func TestValidateAbsolutePath_Cleaning(t *testing.T) {
	// Test that paths are properly cleaned
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "path with redundant separators",
			path:     "/home//user///file.txt",
			expected: "/home/user/file.txt",
		},
		{
			name:     "path with trailing separator",
			path:     "/home/user/",
			expected: "/home/user",
		},
		{
			name:     "path with . components",
			path:     "/home/./user/./file.txt",
			expected: "/home/user/file.txt",
		},
		{
			name:     "path with .. components",
			path:     "/home/user/../user/file.txt",
			expected: "/home/user/file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Only run on Unix systems for consistent path separators
			if runtime.GOOS != "windows" {
				result, err := ValidateAbsolutePath(tt.path)
				require.NoError(t, err, "Should not error for valid absolute path")
				assert.Equal(t, tt.expected, result, "Path should be cleaned correctly")
			}
		})
	}
}

func TestValidateAbsolutePath_SecurityScenarios(t *testing.T) {
	// Test common path traversal attack patterns
	traversalPatterns := []string{
		"../../etc/passwd",
		"../../../etc/passwd",
		"../../../../etc/passwd",
		"..\\..\\windows\\system32\\config\\sam",
		"./../../../etc/passwd",
		"./../../etc/passwd",
	}

	for _, pattern := range traversalPatterns {
		t.Run("blocks_"+strings.ReplaceAll(pattern, "/", "_"), func(t *testing.T) {
			result, err := ValidateAbsolutePath(pattern)
			require.Error(t, err, "Should reject path traversal pattern: %s", pattern)
			assert.Contains(t, err.Error(), "path must be absolute", "Error should mention absolute path requirement")
			assert.Empty(t, result, "Result should be empty for invalid path")
		})
	}
}

func TestFileExists(t *testing.T) {
	dir := t.TempDir()

	// Create a real file to test against
	filePath := filepath.Join(dir, "test.txt")
	require.NoError(t, os.WriteFile(filePath, []byte("hello"), 0600), "Should create temp file")

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "existing file returns true",
			path:     filePath,
			expected: true,
		},
		{
			name:     "non-existent path returns false",
			path:     filepath.Join(dir, "does_not_exist.txt"),
			expected: false,
		},
		{
			name:     "directory path returns false",
			path:     dir,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FileExists(tt.path)
			assert.Equal(t, tt.expected, result, "FileExists(%q) should return %v", tt.path, tt.expected)
		})
	}
}

func TestDirExists(t *testing.T) {
	dir := t.TempDir()

	// Create a real file to use as a non-directory path
	filePath := filepath.Join(dir, "test.txt")
	require.NoError(t, os.WriteFile(filePath, []byte("hello"), 0600), "Should create temp file")

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "existing directory returns true",
			path:     dir,
			expected: true,
		},
		{
			name:     "non-existent path returns false",
			path:     filepath.Join(dir, "does_not_exist"),
			expected: false,
		},
		{
			name:     "file path returns false",
			path:     filePath,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DirExists(tt.path)
			assert.Equal(t, tt.expected, result, "DirExists(%q) should return %v", tt.path, tt.expected)
		})
	}
}

func TestIsDirEmpty(t *testing.T) {
	t.Run("empty directory returns true", func(t *testing.T) {
		dir := t.TempDir()
		assert.True(t, IsDirEmpty(dir), "Newly created temp dir should be empty")
	})

	t.Run("non-empty directory returns false", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "file.txt"), []byte("data"), 0600), "Should create file in dir")
		assert.False(t, IsDirEmpty(dir), "Dir with a file should not be empty")
	})

	t.Run("unreadable directory returns true", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Permission-based test not applicable on Windows")
		}
		dir := t.TempDir()
		unreadable := filepath.Join(dir, "unreadable")
		require.NoError(t, os.Mkdir(unreadable, 0000), "Should create unreadable dir")
		t.Cleanup(func() { _ = os.Chmod(unreadable, 0700) })
		assert.True(t, IsDirEmpty(unreadable), "Unreadable directory should be treated as empty")
	})
}

func TestCopyFile(t *testing.T) {
	t.Run("successful copy", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "src.txt")
		dst := filepath.Join(dir, "dst.txt")
		content := []byte("file content")

		require.NoError(t, os.WriteFile(src, content, 0600), "Should create source file")

		err := CopyFile(src, dst)
		require.NoError(t, err, "CopyFile should succeed for valid src and dst")

		got, readErr := os.ReadFile(dst)
		require.NoError(t, readErr, "Should be able to read copied file")
		assert.Equal(t, content, got, "Copied file content should match source")
	})

	t.Run("missing source file returns error", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "nonexistent.txt")
		dst := filepath.Join(dir, "dst.txt")

		err := CopyFile(src, dst)
		require.Error(t, err, "CopyFile should return error when source does not exist")
	})

	t.Run("missing destination directory returns error", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "src.txt")
		dst := filepath.Join(dir, "missing_dir", "dst.txt")

		require.NoError(t, os.WriteFile(src, []byte("data"), 0600), "Should create source file")

		err := CopyFile(src, dst)
		require.Error(t, err, "CopyFile should return error when destination directory does not exist")
	})

	t.Run("destination file is removed on io.Copy write failure", func(t *testing.T) {
		// /dev/full is a Linux special device that always returns ENOSPC on
		// writes, making it the most reliable way to inject an io.Copy error
		// without modifying CopyFile's signature.
		if runtime.GOOS != "linux" {
			t.Skip("requires /dev/full (Linux only)")
		}
		if _, err := os.Stat("/dev/full"); err != nil {
			t.Skip("/dev/full not available")
		}

		dir := t.TempDir()
		src := filepath.Join(dir, "src.txt")
		dst := filepath.Join(dir, "dst.txt")

		require.NoError(t, os.WriteFile(src, []byte("hello"), 0600), "Should create source file")

		// Point dst at /dev/full via a symlink so that:
		//   - os.Create(dst) succeeds (opens /dev/full for writing)
		//   - io.Copy fails with ENOSPC (every write to /dev/full fails)
		//   - os.Remove(dst) removes the local symlink, not /dev/full itself
		require.NoError(t, os.Symlink("/dev/full", dst), "Should create symlink to /dev/full")

		err := CopyFile(src, dst)
		require.Error(t, err, "CopyFile should return an error when the write fails")
		require.False(t, FileExists(dst), "Destination symlink should be removed after io.Copy failure")
	})
}

func TestMustBeWithin(t *testing.T) {
	base := t.TempDir()

	tests := []struct {
		name      string
		candidate string
		shouldErr bool
	}{
		{
			name:      "file directly inside base",
			candidate: filepath.Join(base, "file.txt"),
			shouldErr: false,
		},
		{
			name:      "file in subdirectory",
			candidate: filepath.Join(base, "sub", "file.txt"),
			shouldErr: false,
		},
		{
			name:      "base directory itself",
			candidate: base,
			shouldErr: false,
		},
		{
			name:      "path traversal with ..",
			candidate: filepath.Join(base, "..", "escape.txt"),
			shouldErr: true,
		},
		{
			name:      "deeply nested traversal",
			candidate: filepath.Join(base, "a", "b", "..", "..", "..", "escape.txt"),
			shouldErr: true,
		},
		{
			name:      "absolute path outside base",
			candidate: "/etc/passwd",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := MustBeWithin(base, tt.candidate)
			if tt.shouldErr {
				require.Error(t, err, "MustBeWithin should reject path %q relative to %q", tt.candidate, base)
				assert.Contains(t, err.Error(), "escapes base directory", "Error should describe the escape")
			} else {
				require.NoError(t, err, "MustBeWithin should accept path %q within %q", tt.candidate, base)
			}
		})
	}

	t.Run("symlink escape", func(t *testing.T) {
		// Create a real file outside the base directory.
		outsideFile, err := os.CreateTemp("", "mustbewithin-outside-*")
		require.NoError(t, err, "failed to create outside file")
		t.Cleanup(func() { _ = os.Remove(outsideFile.Name()) })
		outsidePath := outsideFile.Name()
		require.NoError(t, outsideFile.Close())

		// Create a symlink inside base that points to the outside file.
		linkPath := filepath.Join(base, "link-to-outside")
		if err := os.Symlink(outsidePath, linkPath); err != nil {
			t.Skipf("symlinks not supported: %v", err)
		}
		t.Cleanup(func() { _ = os.Remove(linkPath) })

		err = MustBeWithin(base, linkPath)
		require.Error(t, err, "MustBeWithin should reject symlink that points outside base")
		assert.Contains(t, err.Error(), "escapes base directory", "Error should describe the symlink escape")
	})
}

func TestExtractFileFromTar_UnsafePaths(t *testing.T) {
	buildTar := func(files map[string][]byte) []byte {
		var buf bytes.Buffer
		tw := tar.NewWriter(&buf)
		for name, content := range files {
			hdr := &tar.Header{
				Name: name,
				Mode: 0600,
				Size: int64(len(content)),
			}
			if err := tw.WriteHeader(hdr); err != nil {
				t.Fatalf("buildTar: WriteHeader: %v", err)
			}
			if _, err := tw.Write(content); err != nil {
				t.Fatalf("buildTar: Write: %v", err)
			}
		}
		if err := tw.Close(); err != nil {
			t.Fatalf("buildTar: Close: %v", err)
		}
		return buf.Bytes()
	}

	t.Run("rejects absolute path as search target", func(t *testing.T) {
		archive := buildTar(map[string][]byte{"file.txt": []byte("data")})
		got, err := ExtractFileFromTar(archive, "/etc/passwd")
		require.Error(t, err, "Should reject absolute path as search target")
		assert.Contains(t, err.Error(), "unsafe path", "Error should mention unsafe path")
		assert.Nil(t, got, "Result should be nil for unsafe path")
	})

	t.Run("rejects dotdot in search target", func(t *testing.T) {
		archive := buildTar(map[string][]byte{"file.txt": []byte("data")})
		got, err := ExtractFileFromTar(archive, "../escape.txt")
		require.Error(t, err, "Should reject .. in search target")
		assert.Contains(t, err.Error(), "unsafe path", "Error should mention unsafe path")
		assert.Nil(t, got, "Result should be nil for unsafe path")
	})

	t.Run("allows filename containing dotdot as substring", func(t *testing.T) {
		want := []byte("not a traversal")
		archive := buildTar(map[string][]byte{"file..backup.txt": want})
		got, err := ExtractFileFromTar(archive, "file..backup.txt")
		require.NoError(t, err, "Should allow filename with dotdot as substring, not path component")
		assert.Equal(t, want, got, "Should return correct content for file..backup.txt")
	})

	t.Run("skips tar entry with absolute name, does not match", func(t *testing.T) {
		// Build archive with an absolute-named entry; it should be skipped even
		// if the caller searches for the same name.
		archive := buildTar(map[string][]byte{"/etc/passwd": []byte("root")})
		// Searching for the same absolute path must be rejected by the safe-target check.
		got, err := ExtractFileFromTar(archive, "/etc/passwd")
		require.Error(t, err, "Should reject absolute search target")
		assert.Nil(t, got)
	})

	t.Run("skips tar entry with dotdot name and returns not found", func(t *testing.T) {
		archive := buildTar(map[string][]byte{"../escape.txt": []byte("bad")})
		// Searching for the relative form that doesn't start with .. is fine as
		// a target; the archive entry is just silently skipped.
		got, err := ExtractFileFromTar(archive, "escape.txt")
		require.Error(t, err, "File should not be found because dotdot entry was skipped")
		assert.Contains(t, err.Error(), "not found", "Error should indicate file not found")
		assert.Nil(t, got)
	})
}

func TestExtractFileFromTar(t *testing.T) {
	// Helper to build an in-memory tar archive
	buildTar := func(files map[string][]byte) []byte {
		var buf bytes.Buffer
		tw := tar.NewWriter(&buf)
		for name, content := range files {
			hdr := &tar.Header{
				Name: name,
				Mode: 0600,
				Size: int64(len(content)),
			}
			if err := tw.WriteHeader(hdr); err != nil {
				t.Fatalf("buildTar: WriteHeader: %v", err)
			}
			if _, err := tw.Write(content); err != nil {
				t.Fatalf("buildTar: Write: %v", err)
			}
		}
		if err := tw.Close(); err != nil {
			t.Fatalf("buildTar: Close: %v", err)
		}
		return buf.Bytes()
	}

	t.Run("found file returns its content", func(t *testing.T) {
		want := []byte("hello from tar")
		archive := buildTar(map[string][]byte{"subdir/file.txt": want})

		got, err := ExtractFileFromTar(archive, "subdir/file.txt")
		require.NoError(t, err, "ExtractFileFromTar should succeed when file is present")
		assert.Equal(t, want, got, "Extracted content should match original")
	})

	t.Run("file not found returns error", func(t *testing.T) {
		archive := buildTar(map[string][]byte{"other.txt": []byte("data")})

		got, err := ExtractFileFromTar(archive, "missing.txt")
		require.Error(t, err, "ExtractFileFromTar should return error when file is absent")
		assert.Contains(t, err.Error(), "missing.txt", "Error should mention the missing filename")
		assert.Nil(t, got, "Result should be nil when file is not found")
	})

	t.Run("corrupted archive returns error", func(t *testing.T) {
		corrupted := []byte("this is not a valid tar archive")

		got, err := ExtractFileFromTar(corrupted, "any.txt")
		require.Error(t, err, "ExtractFileFromTar should return error for corrupted archive")
		assert.Nil(t, got, "Result should be nil for corrupted archive")
	})
}
