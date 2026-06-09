// Package fileutil provides utility functions for working with file paths and file operations.
package fileutil

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/github/gh-aw/pkg/logger"
)

var fileutilLog = logger.New("fileutil:fileutil")

// ValidateAbsolutePath validates that a file path is absolute and safe to use.
// It performs the following security checks:
//   - Cleans the path using filepath.Clean to normalize . and .. components
//   - Verifies the path is absolute to prevent relative path traversal attacks
//
// Returns the cleaned absolute path if validation succeeds, or an error if:
//   - The path is empty
//   - The path is relative (not absolute)
//
// This function should be used before any file operations (read, write, stat, etc.)
// to ensure defense-in-depth security against path traversal vulnerabilities.
//
// Example:
//
// cleanPath, err := fileutil.ValidateAbsolutePath(userInputPath)
//
//	if err != nil {
//	   return fmt.Errorf("invalid path: %w", err)
//	}
//
// content, err := os.ReadFile(cleanPath)
func ValidateAbsolutePath(path string) (string, error) {
	// Check for empty path
	if path == "" {
		fileutilLog.Print("ValidateAbsolutePath: rejected empty path")
		return "", errors.New("path cannot be empty")
	}

	// Sanitize the filepath to prevent path traversal attacks
	cleanPath := filepath.Clean(path)

	// Verify the path is absolute to prevent relative path traversal
	if !filepath.IsAbs(cleanPath) {
		fileutilLog.Printf("ValidateAbsolutePath: rejected relative path: %s", path)
		return "", fmt.Errorf("path must be absolute, got: %s", path)
	}

	fileutilLog.Printf("ValidateAbsolutePath: validated path: %s", cleanPath)
	return cleanPath, nil
}

// ValidatePathWithinBase checks that candidate is located within the base directory tree.
// Both paths are resolved via filepath.EvalSymlinks (with filepath.Abs as
// fallback when a path does not yet exist) before comparison, so neither ".."
// components nor symlinks pointing outside base can be used to escape.
//
// Returns an error when:
//   - Either path cannot be resolved to an absolute form.
//   - The resolved candidate path starts outside the resolved base directory.
func ValidatePathWithinBase(base, candidate string) error {
	fileutilLog.Printf("ValidatePathWithinBase: checking candidate=%q within base=%q", candidate, base)
	// EvalSymlinks resolves both symlinks and ".." components.
	// Fall back to Abs when a path does not exist on disk yet.
	absBase, err := filepath.EvalSymlinks(base)
	if err != nil {
		absBase, err = filepath.Abs(base)
		if err != nil {
			return fmt.Errorf("failed to resolve base path %q: %w", base, err)
		}
	}
	absCand, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		absCand, err = filepath.Abs(candidate)
		if err != nil {
			return fmt.Errorf("failed to resolve candidate path %q: %w", candidate, err)
		}
	}
	rel, err := filepath.Rel(absBase, absCand)
	if err != nil || !filepath.IsLocal(rel) {
		fileutilLog.Printf("ValidatePathWithinBase: path escape detected: candidate=%q base=%q", candidate, base)
		return fmt.Errorf("path %q escapes base directory %q", candidate, base)
	}
	fileutilLog.Printf("ValidatePathWithinBase: path is safe: candidate=%q (rel=%s) within base=%q", candidate, rel, base)
	return nil
}

// EnsureParentDir ensures the parent directory for path exists, creating it recursively when needed.
func EnsureParentDir(path string, perm os.FileMode) error {
	if path == "" {
		fileutilLog.Print("EnsureParentDir: rejected empty path")
		return errors.New("path cannot be empty")
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, perm); err != nil {
		fileutilLog.Printf("EnsureParentDir: failed for %s: %v", dir, err)
		return fmt.Errorf("failed to create parent directory %s: %w", dir, err)
	}
	fileutilLog.Printf("EnsureParentDir: ensured %s", dir)
	return nil
}

// FileExists checks if a file exists and is not a directory.
func FileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// DirExists checks if a directory exists.
func DirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// IsDirEmpty checks if a directory is empty.
func IsDirEmpty(path string) bool {
	files, err := os.ReadDir(path)
	if err != nil {
		return true // Consider it empty if we can't read it
	}
	return len(files) == 0
}

type syncWriteCloser interface {
	io.Writer
	Sync() error
	Close() error
}

func copyFileContents(in io.Reader, out syncWriteCloser, dst string) (err error) {
	removePartial := false

	defer func() {
		if closeErr := out.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
		if removePartial {
			if removeErr := os.Remove(dst); removeErr != nil {
				fileutilLog.Printf("Failed to remove partial destination file during cleanup: %s", removeErr)
			}
		}
	}()

	if _, err = io.Copy(out, in); err != nil {
		removePartial = true
		return err
	}

	return out.Sync()
}

// CopyFile copies a file from src to dst using buffered IO.
func CopyFile(src, dst string) error {
	fileutilLog.Printf("Copying file: src=%s, dst=%s", src, dst)
	in, err := os.Open(src)
	if err != nil {
		fileutilLog.Printf("Failed to open source file: %s", err)
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		fileutilLog.Printf("Failed to create destination file: %s", err)
		return err
	}
	err = copyFileContents(in, out, dst)
	if err != nil {
		return err
	}

	fileutilLog.Printf("File copied successfully: src=%s, dst=%s", src, dst)
	return nil
}
