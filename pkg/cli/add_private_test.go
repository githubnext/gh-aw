//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"testing"
)

// TestExtractWorkflowPrivate tests the ExtractWorkflowPrivate function
func TestExtractWorkflowPrivate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name: "workflow with private: true",
			content: `---
name: Test Workflow
private: true
on: push
---

# Test Workflow`,
			expected: true,
		},
		{
			name: "workflow with private: false",
			content: `---
name: Test Workflow
private: false
on: push
---

# Test Workflow`,
			expected: false,
		},
		{
			name: "workflow without private field",
			content: `---
name: Test Workflow
on: push
---

# Test Workflow`,
			expected: false,
		},
		{
			name:     "workflow without frontmatter",
			content:  "# Test Workflow\n\nThis is the workflow content.",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := ExtractWorkflowPrivate(tt.content)
			if result != tt.expected {
				t.Errorf("ExtractWorkflowPrivate() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestAIModeratorWorkflowIsShareable(t *testing.T) {
	t.Parallel()

	content, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "ai-moderator.md"))
	if err != nil {
		t.Fatalf("failed to read AI Moderator workflow: %v", err)
	}

	if ExtractWorkflowPrivate(string(content)) {
		t.Fatal("AI Moderator workflow must remain shareable for gh aw add")
	}
}
