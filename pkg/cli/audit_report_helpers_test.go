package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCalculateDirectorySize(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) string
		expected int64
	}{
		{
			name: "empty directory",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				return dir
			},
			expected: 0,
		},
		{
			name: "single file",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				err := os.WriteFile(filepath.Join(dir, "test.txt"), []byte("hello"), 0644)
				if err != nil {
					t.Fatal(err)
				}
				return dir
			},
			expected: 5,
		},
		{
			name: "multiple files in nested directories",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				// File 1: 10 bytes
				err := os.WriteFile(filepath.Join(dir, "file1.txt"), []byte("0123456789"), 0644)
				if err != nil {
					t.Fatal(err)
				}
				// Create subdirectory
				subdir := filepath.Join(dir, "subdir")
				err = os.Mkdir(subdir, 0755)
				if err != nil {
					t.Fatal(err)
				}
				// File 2: 5 bytes
				err = os.WriteFile(filepath.Join(subdir, "file2.txt"), []byte("hello"), 0644)
				if err != nil {
					t.Fatal(err)
				}
				return dir
			},
			expected: 15,
		},
		{
			name: "nonexistent directory",
			setup: func(t *testing.T) string {
				return "/nonexistent/path/that/does/not/exist"
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.setup(t)
			got := calculateDirectorySize(dir)
			if got != tt.expected {
				t.Errorf("calculateDirectorySize() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestParseDurationString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Duration
	}{
		{
			name:     "valid duration - seconds",
			input:    "5s",
			expected: 5 * time.Second,
		},
		{
			name:     "valid duration - minutes",
			input:    "2m",
			expected: 2 * time.Minute,
		},
		{
			name:     "valid duration - hours",
			input:    "1h",
			expected: 1 * time.Hour,
		},
		{
			name:     "valid duration - complex",
			input:    "1h30m45s",
			expected: 1*time.Hour + 30*time.Minute + 45*time.Second,
		},
		{
			name:     "invalid duration",
			input:    "not a duration",
			expected: 0,
		},
		{
			name:     "empty string",
			input:    "",
			expected: 0,
		},
		{
			name:     "zero duration",
			input:    "0s",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseDurationString(tt.input)
			if got != tt.expected {
				t.Errorf("parseDurationString(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "string shorter than max length",
			input:    "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "string equal to max length",
			input:    "hello",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "string longer than max length",
			input:    "hello world",
			maxLen:   8,
			expected: "hello...",
		},
		{
			name:     "max length 3 or less",
			input:    "hello",
			maxLen:   3,
			expected: "hel",
		},
		{
			name:     "max length 2",
			input:    "hello",
			maxLen:   2,
			expected: "he",
		},
		{
			name:     "max length 1",
			input:    "hello",
			maxLen:   1,
			expected: "h",
		},
		{
			name:     "empty string",
			input:    "",
			maxLen:   10,
			expected: "",
		},
		{
			name:     "unicode string",
			input:    "Hello 世界",
			maxLen:   9,
			expected: "Hello ...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateString(tt.input, tt.maxLen)
			if got != tt.expected {
				t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.expected)
			}
		})
	}
}
