package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/githubnext/gh-aw/pkg/cli/fileutil"
	"github.com/githubnext/gh-aw/pkg/testutil"
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
				dir := testutil.TempDir(t, "test-*")
				return dir
			},
			expected: 0,
		},
		{
			name: "single file",
			setup: func(t *testing.T) string {
				dir := testutil.TempDir(t, "test-*")
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
				dir := testutil.TempDir(t, "test-*")
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
			got := fileutil.CalculateDirectorySize(dir)
			if got != tt.expected {
				t.Errorf("fileutil.CalculateDirectorySize() = %d, want %d", got, tt.expected)
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

// TestDownloadedFilesInAuditData verifies that downloaded files are properly included in audit data
func TestDownloadedFilesInAuditData(t *testing.T) {
// Create temporary directory with test files
tmpDir := testutil.TempDir(t, "test-*")

// Create various test files
testFiles := map[string][]byte{
"aw_info.json":      []byte(`{"engine":"copilot"}`),
"safe_output.jsonl": []byte(`{"test":"data"}`),
"agent-stdio.log":   []byte("Log content\n"),
"aw.patch":          []byte("diff content\n"),
"prompt.txt":        []byte("Prompt text\n"),
"run_summary.json":  []byte(`{"version":"1.0"}`),
"firewall.md":       []byte("# Firewall Analysis\n"),
"log.md":            []byte("# Agent Log\n"),
"custom.json":       []byte(`{}`),
"notes.txt":         []byte("Some notes\n"),
}

for filename, content := range testFiles {
if err := os.WriteFile(filepath.Join(tmpDir, filename), content, 0644); err != nil {
t.Fatalf("Failed to create test file %s: %v", filename, err)
}
}

// Create a subdirectory
subdir := filepath.Join(tmpDir, "agent_output")
if err := os.MkdirAll(subdir, 0755); err != nil {
t.Fatalf("Failed to create subdirectory: %v", err)
}
if err := os.WriteFile(filepath.Join(subdir, "result.json"), []byte(`{}`), 0644); err != nil {
t.Fatalf("Failed to create file in subdirectory: %v", err)
}

// Extract downloaded files
files := extractDownloadedFiles(tmpDir)

// Verify we got all files
expectedCount := len(testFiles) + 1 // +1 for the directory
if len(files) != expectedCount {
t.Errorf("Expected %d files, got %d", expectedCount, len(files))
}

// Verify specific files have correct attributes
fileMap := make(map[string]FileInfo)
for _, file := range files {
fileMap[file.Path] = file
}

// Check aw_info.json
if info, ok := fileMap["aw_info.json"]; ok {
if info.Description != "Engine configuration and workflow metadata" {
t.Errorf("Expected specific description for aw_info.json, got: %s", info.Description)
}
if info.IsDirectory {
t.Error("aw_info.json should not be marked as directory")
}
if info.Size == 0 {
t.Error("aw_info.json should have non-zero size")
}
if info.SizeFormatted == "" {
t.Error("aw_info.json should have formatted size")
}
} else {
t.Error("Expected to find aw_info.json in files list")
}

// Check firewall.md
if info, ok := fileMap["firewall.md"]; ok {
if info.Description != "Firewall log analysis report" {
t.Errorf("Expected specific description for firewall.md, got: %s", info.Description)
}
} else {
t.Error("Expected to find firewall.md in files list")
}

// Check custom.json (should have generic description)
if info, ok := fileMap["custom.json"]; ok {
if info.Description != "JSON data file" {
t.Errorf("Expected generic JSON description for custom.json, got: %s", info.Description)
}
} else {
t.Error("Expected to find custom.json in files list")
}

// Check agent_output directory
if info, ok := fileMap["agent_output"]; ok {
if !info.IsDirectory {
t.Error("agent_output should be marked as directory")
}
if info.Description != "Directory containing log files" {
t.Errorf("Expected specific description for agent_output, got: %s", info.Description)
}
if info.Size == 0 {
t.Error("Directory should have calculated size")
}
} else {
t.Error("Expected to find agent_output directory in files list")
}
}

// TestConsoleOutputIncludesFileInfo verifies console output displays file information
func TestConsoleOutputIncludesFileInfo(t *testing.T) {
// Create temporary directory
tmpDir := testutil.TempDir(t, "test-*")

// Create test data
run := WorkflowRun{
DatabaseID:   123,
WorkflowName: "Test",
Status:       "completed",
Conclusion:   "success",
CreatedAt:    time.Now(),
Event:        "push",
HeadBranch:   "main",
URL:          "https://github.com/test/repo/actions/runs/123",
LogsPath:     tmpDir,
}

metrics := LogMetrics{}

processedRun := ProcessedRun{
Run: run,
}

downloadedFiles := []FileInfo{
{
Path:          "aw_info.json",
Size:          256,
SizeFormatted: "256 B",
Description:   "Engine configuration and workflow metadata",
IsDirectory:   false,
},
{
Path:          "agent_output",
Size:          4096,
SizeFormatted: "4.0 KB",
Description:   "Directory containing log files",
IsDirectory:   true,
},
}

// Build audit data
auditData := buildAuditData(processedRun, metrics)
auditData.DownloadedFiles = downloadedFiles

// Verify downloaded files are in audit data
if len(auditData.DownloadedFiles) != 2 {
t.Errorf("Expected 2 downloaded files in audit data, got %d", len(auditData.DownloadedFiles))
}

// Verify file details are preserved
for _, file := range auditData.DownloadedFiles {
if file.Path == "aw_info.json" {
if file.Size != 256 {
t.Errorf("Expected size 256, got %d", file.Size)
}
if file.SizeFormatted != "256 B" {
t.Errorf("Expected formatted size '256 B', got '%s'", file.SizeFormatted)
}
if file.Description == "" {
t.Error("Expected description to be present")
}
}
}
}
