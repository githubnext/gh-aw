//go:build !integration

package stringutil

import (
	"os"
	"testing"
)

func TestNormalizeWorkflowName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "name without extension",
			input:    "weekly-research",
			expected: "weekly-research",
		},
		{
			name:     "name with .md extension",
			input:    "weekly-research.md",
			expected: "weekly-research",
		},
		{
			name:     "name with .lock.yml extension",
			input:    "weekly-research.lock.yml",
			expected: "weekly-research",
		},
		{
			name:     "name with .lock.yaml extension",
			input:    "weekly-research.lock.yaml",
			expected: "weekly-research",
		},
		{
			name:     "name with dots in filename",
			input:    "my.workflow.md",
			expected: "my.workflow",
		},
		{
			name:     "name with dots and lock.yml",
			input:    "my.workflow.lock.yml",
			expected: "my.workflow",
		},
		{
			name:     "name with dots and lock.yaml",
			input:    "my.workflow.lock.yaml",
			expected: "my.workflow",
		},
		{
			name:     "name with other extension",
			input:    "workflow.yaml",
			expected: "workflow.yaml",
		},
		{
			name:     "simple name",
			input:    "agent",
			expected: "agent",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "just .md",
			input:    ".md",
			expected: "",
		},
		{
			name:     "just .lock.yml",
			input:    ".lock.yml",
			expected: "",
		},
		{
			name:     "just .lock.yaml",
			input:    ".lock.yaml",
			expected: "",
		},
		{
			name:     "multiple extensions priority",
			input:    "workflow.md.lock.yml",
			expected: "workflow.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeWorkflowName(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeWorkflowName(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNormalizeSafeOutputIdentifier(t *testing.T) {
	tests := []struct {
		name       string
		identifier string
		expected   string
	}{
		{
			name:       "dash-separated to underscore",
			identifier: "create-issue",
			expected:   "create_issue",
		},
		{
			name:       "already underscore-separated",
			identifier: "create_issue",
			expected:   "create_issue",
		},
		{
			name:       "multiple dashes",
			identifier: "add-comment-to-issue",
			expected:   "add_comment_to_issue",
		},
		{
			name:       "mixed dashes and underscores",
			identifier: "update-pr_status",
			expected:   "update_pr_status",
		},
		{
			name:       "no dashes or underscores",
			identifier: "createissue",
			expected:   "createissue",
		},
		{
			name:       "single dash",
			identifier: "add-comment",
			expected:   "add_comment",
		},
		{
			name:       "trailing dash",
			identifier: "update-",
			expected:   "update_",
		},
		{
			name:       "leading dash",
			identifier: "-create",
			expected:   "_create",
		},
		{
			name:       "consecutive dashes",
			identifier: "create--issue",
			expected:   "create__issue",
		},
		{
			name:       "empty string",
			identifier: "",
			expected:   "",
		},
		{
			name:       "only dashes",
			identifier: "---",
			expected:   "___",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeSafeOutputIdentifier(tt.identifier)
			if result != tt.expected {
				t.Errorf("NormalizeSafeOutputIdentifier(%q) = %q, want %q", tt.identifier, result, tt.expected)
			}
		})
	}
}

func BenchmarkNormalizeWorkflowName(b *testing.B) {
	name := "weekly-research-workflow.lock.yml"
	for b.Loop() {
		NormalizeWorkflowName(name)
	}
}

func BenchmarkNormalizeSafeOutputIdentifier(b *testing.B) {
	identifier := "create-pull-request-review-comment"
	for b.Loop() {
		NormalizeSafeOutputIdentifier(identifier)
	}
}

func TestMarkdownToLockFile(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple markdown file",
			input:    "weekly-research.md",
			expected: "weekly-research.lock.yml",
		},
		{
			name:     "markdown file with path",
			input:    ".github/workflows/test.md",
			expected: ".github/workflows/test.lock.yml",
		},
		{
			name:     "already a lock file .yml",
			input:    "workflow.lock.yml",
			expected: "workflow.lock.yml",
		},
		{
			name:     "already a lock file .yaml",
			input:    "workflow.lock.yaml",
			expected: "workflow.lock.yaml",
		},
		{
			name:     "file with dots in name",
			input:    "my.workflow.md",
			expected: "my.workflow.lock.yml",
		},
		{
			name:     "file without extension",
			input:    "workflow",
			expected: "workflow.lock.yml",
		},
		{
			name:     "absolute path",
			input:    "/home/user/.github/workflows/daily.md",
			expected: "/home/user/.github/workflows/daily.lock.yml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MarkdownToLockFile(tt.input)
			if result != tt.expected {
				t.Errorf("MarkdownToLockFile(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestLockFileToMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple lock file .yml",
			input:    "weekly-research.lock.yml",
			expected: "weekly-research.md",
		},
		{
			name:     "simple lock file .yaml",
			input:    "weekly-research.lock.yaml",
			expected: "weekly-research.md",
		},
		{
			name:     "lock file with path",
			input:    ".github/workflows/test.lock.yml",
			expected: ".github/workflows/test.md",
		},
		{
			name:     "lock file .yaml with path",
			input:    ".github/workflows/test.lock.yaml",
			expected: ".github/workflows/test.md",
		},
		{
			name:     "already a markdown file",
			input:    "workflow.md",
			expected: "workflow.md",
		},
		{
			name:     "file with dots in name",
			input:    "my.workflow.lock.yml",
			expected: "my.workflow.md",
		},
		{
			name:     "file with dots in name .yaml",
			input:    "my.workflow.lock.yaml",
			expected: "my.workflow.md",
		},
		{
			name:     "absolute path",
			input:    "/home/user/.github/workflows/daily.lock.yml",
			expected: "/home/user/.github/workflows/daily.md",
		},
		{
			name:     "agentic-campaign-generator in workflows directory",
			input:    ".github/workflows/agentic-campaign-generator.lock.yml",
			expected: ".github/workflows/agentic-campaign-generator.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := LockFileToMarkdown(tt.input)
			if result != tt.expected {
				t.Errorf("LockFileToMarkdown(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestRoundTripConversions(t *testing.T) {
	// Test that converting back and forth preserves the base name
	t.Run("markdown to lock and back", func(t *testing.T) {
		original := "workflow.md"
		lockFile := MarkdownToLockFile(original)
		backToMd := LockFileToMarkdown(lockFile)
		if backToMd != original {
			t.Errorf("Round trip failed: %q -> %q -> %q", original, lockFile, backToMd)
		}
	})

	t.Run("lock to markdown and back", func(t *testing.T) {
		original := "workflow.lock.yml"
		mdFile := LockFileToMarkdown(original)
		backToLock := MarkdownToLockFile(mdFile)
		if backToLock != original {
			t.Errorf("Round trip failed: %q -> %q -> %q", original, mdFile, backToLock)
		}
	})

	t.Run("lock.yaml to markdown and back", func(t *testing.T) {
		original := "workflow.lock.yaml"
		mdFile := LockFileToMarkdown(original)
		if mdFile != "workflow.md" {
			t.Errorf("LockFileToMarkdown(%q) = %q, expected %q", original, mdFile, "workflow.md")
		}
	})
}

func TestMarkdownToLockFileOnDisk(t *testing.T) {
	t.Run("no existing lock file defaults to .lock.yml", func(t *testing.T) {
		tmpDir := t.TempDir()
		mdPath := tmpDir + "/workflow.md"
		result := MarkdownToLockFileOnDisk(mdPath)
		expected := tmpDir + "/workflow.lock.yml"
		if result != expected {
			t.Errorf("MarkdownToLockFileOnDisk(%q) = %q, expected %q", mdPath, result, expected)
		}
	})

	t.Run("existing .lock.yaml is preferred", func(t *testing.T) {
		tmpDir := t.TempDir()
		mdPath := tmpDir + "/workflow.md"
		lockYamlPath := tmpDir + "/workflow.lock.yaml"
		// Create the .lock.yaml file
		if err := writeFile(lockYamlPath, ""); err != nil {
			t.Fatalf("failed to create .lock.yaml: %v", err)
		}
		result := MarkdownToLockFileOnDisk(mdPath)
		if result != lockYamlPath {
			t.Errorf("MarkdownToLockFileOnDisk(%q) = %q, expected %q", mdPath, result, lockYamlPath)
		}
	})

	t.Run("existing .lock.yml is not overridden by .lock.yaml when only .lock.yml present", func(t *testing.T) {
		tmpDir := t.TempDir()
		mdPath := tmpDir + "/workflow.md"
		lockYmlPath := tmpDir + "/workflow.lock.yml"
		// Create the .lock.yml file (but not .lock.yaml)
		if err := writeFile(lockYmlPath, ""); err != nil {
			t.Fatalf("failed to create .lock.yml: %v", err)
		}
		result := MarkdownToLockFileOnDisk(mdPath)
		if result != lockYmlPath {
			t.Errorf("MarkdownToLockFileOnDisk(%q) = %q, expected %q", mdPath, result, lockYmlPath)
		}
	})

	t.Run("already a .lock.yml returns unchanged", func(t *testing.T) {
		result := MarkdownToLockFileOnDisk("workflow.lock.yml")
		if result != "workflow.lock.yml" {
			t.Errorf("MarkdownToLockFileOnDisk(%q) = %q, expected %q", "workflow.lock.yml", result, "workflow.lock.yml")
		}
	})

	t.Run("already a .lock.yaml returns unchanged", func(t *testing.T) {
		result := MarkdownToLockFileOnDisk("workflow.lock.yaml")
		if result != "workflow.lock.yaml" {
			t.Errorf("MarkdownToLockFileOnDisk(%q) = %q, expected %q", "workflow.lock.yaml", result, "workflow.lock.yaml")
		}
	})
}

func TestStripLockExtension(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     ".lock.yml extension",
			input:    "workflow.lock.yml",
			expected: "workflow",
		},
		{
			name:     ".lock.yaml extension",
			input:    "workflow.lock.yaml",
			expected: "workflow",
		},
		{
			name:     "path with .lock.yml",
			input:    "/path/to/workflow.lock.yml",
			expected: "/path/to/workflow",
		},
		{
			name:     "path with .lock.yaml",
			input:    "/path/to/workflow.lock.yaml",
			expected: "/path/to/workflow",
		},
		{
			name:     "no lock extension unchanged",
			input:    "workflow.md",
			expected: "workflow.md",
		},
		{
			name:     "dots in name .lock.yml",
			input:    "my.workflow.lock.yml",
			expected: "my.workflow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StripLockExtension(tt.input)
			if result != tt.expected {
				t.Errorf("StripLockExtension(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

// writeFile is a test helper to create a file with given content
func writeFile(path, content string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(content)
	return err
}
