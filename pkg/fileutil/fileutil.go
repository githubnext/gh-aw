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

var log = logger.New("fileutil:fileutil")

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
		return "", errors.New("path cannot be empty")
	}

	// Sanitize the filepath to prevent path traversal attacks
	cleanPath := filepath.Clean(path)

	// Verify the path is absolute to prevent relative path traversal
	if !filepath.IsAbs(cleanPath) {
		return "", fmt.Errorf("path must be absolute, got: %s", path)
	}

	return cleanPath, nil
}

// MustBeWithin checks that candidate is located within the base directory tree.
// Both paths are resolved via filepath.EvalSymlinks (with filepath.Abs as
// fallback when a path does not yet exist) before comparison, so neither ".."
// components nor symlinks pointing outside base can be used to escape.
//
// Returns an error when:
//   - Either path cannot be resolved to an absolute form.
//   - The resolved candidate path starts outside the resolved base directory.
func MustBeWithin(base, candidate string) error {
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
		return fmt.Errorf("path %q escapes base directory %q", candidate, base)
	}
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

// CopyFile copies a file from src to dst using buffered IO.
func CopyFile(src, dst string) error {
	log.Printf("Copying file: src=%s, dst=%s", src, dst)
	in, err := os.Open(src)
	if err != nil {
		log.Printf("Failed to open source file: %s", err)
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		log.Printf("Failed to create destination file: %s", err)
		return err
	}
	defer func() { _ = out.Close() }()

	if _, err = io.Copy(out, in); err != nil {
		if closeErr := out.Close(); closeErr != nil {
			log.Printf("Failed to close destination file during cleanup: %s", closeErr)
		}
		if removeErr := os.Remove(dst); removeErr != nil {
			log.Printf("Failed to remove partial destination file during cleanup: %s", removeErr)
		}
		return err
	}
	log.Printf("File copied successfully: src=%s, dst=%s", src, dst)
	return out.Sync()
}
