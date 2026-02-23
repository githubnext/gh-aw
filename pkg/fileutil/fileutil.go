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
		return err
	}
	log.Printf("File copied successfully: src=%s, dst=%s", src, dst)
	return out.Sync()
}

// CalculateDirectorySize recursively calculates the total size of files in a directory.
func CalculateDirectorySize(dirPath string) int64 {
	log.Printf("Calculating directory size: %s", dirPath)
	var totalSize int64

	_ = filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	log.Printf("Directory size: path=%s, size=%d bytes", dirPath, totalSize)
	return totalSize
}
