package fileutil

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// ValidateExecutablePath validates that an executable path is absolute, resolves
// symlinks when possible, points to a file, and is executable on non-Windows platforms.
func ValidateExecutablePath(path string) (string, error) {
	cleanPath, err := ValidateAbsolutePath(path)
	if err != nil {
		return "", err
	}

	resolvedPath, err := filepath.EvalSymlinks(cleanPath)
	if err == nil {
		cleanPath = resolvedPath
	} else if os.IsNotExist(err) {
		return "", fmt.Errorf("executable path %q does not exist", path)
	} else {
		return "", fmt.Errorf("failed to resolve executable path %q: %w", path, err)
	}

	info, err := os.Stat(cleanPath)
	if err != nil {
		return "", fmt.Errorf("failed to stat executable path %q: %w", cleanPath, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("executable path %q is a directory", cleanPath)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("executable path %q is not a regular file", cleanPath)
	}
	if runtime.GOOS != "windows" && info.Mode()&0o111 == 0 {
		return "", fmt.Errorf("executable path %q is not executable", cleanPath)
	}

	return cleanPath, nil
}

// ResolveExecutablePath resolves an executable from PATH and validates the resulting path.
func ResolveExecutablePath(name string) (string, error) {
	if name == "" {
		return "", errors.New("executable name cannot be empty")
	}

	path, err := exec.LookPath(name)
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(path) {
		path, err = filepath.Abs(path)
		if err != nil {
			return "", fmt.Errorf("failed to make executable path absolute for %q: %w", name, err)
		}
	}

	return ValidateExecutablePath(path)
}
