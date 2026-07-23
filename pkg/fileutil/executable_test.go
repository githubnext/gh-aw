//go:build !integration

package fileutil

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateExecutablePath(t *testing.T) {
	t.Run("accepts executable file", func(t *testing.T) {
		exe := filepath.Join(t.TempDir(), "tool")
		require.NoError(t, os.WriteFile(exe, []byte("#!/bin/sh\nexit 0\n"), 0o755))

		got, err := ValidateExecutablePath(exe)
		require.NoError(t, err)
		assert.Equal(t, exe, got)
	})

	t.Run("rejects directory", func(t *testing.T) {
		_, err := ValidateExecutablePath(t.TempDir())
		require.Error(t, err)
		require.ErrorContains(t, err, "is a directory")
	})

	t.Run("rejects non-executable file on unix", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("executable bit semantics differ on Windows")
		}

		exe := filepath.Join(t.TempDir(), "tool")
		require.NoError(t, os.WriteFile(exe, []byte("echo hi\n"), 0o644))

		_, err := ValidateExecutablePath(exe)
		require.Error(t, err)
		require.ErrorContains(t, err, "is not executable")
	})
}

func TestResolveExecutablePath(t *testing.T) {
	t.Run("rejects empty name", func(t *testing.T) {
		_, err := ResolveExecutablePath("")
		require.Error(t, err)
		require.ErrorContains(t, err, "executable name cannot be empty")
	})

	t.Run("resolves executable on PATH", func(t *testing.T) {
		binDir := t.TempDir()
		exeName := "fake-tool"
		if runtime.GOOS == "windows" {
			exeName += ".bat"
		}
		exePath := filepath.Join(binDir, exeName)
		require.NoError(t, os.WriteFile(exePath, []byte("@echo off\r\n"), 0o755))

		originalPath := os.Getenv("PATH")
		t.Cleanup(func() {
			require.NoError(t, os.Setenv("PATH", originalPath))
		})
		require.NoError(t, os.Setenv("PATH", binDir+string(os.PathListSeparator)+originalPath))

		got, err := ResolveExecutablePath(exeName)
		require.NoError(t, err)
		assert.Equal(t, exePath, got)
	})
}
