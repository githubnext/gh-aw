//go:build !integration

package cli

import (
	"testing"
)

// TestExtractWorkflowPrivate tests the ExtractWorkflowPrivate function
func TestExtractWorkflowPrivate(t *testing.T) {
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
			result := ExtractWorkflowPrivate(tt.content)
			if result != tt.expected {
				t.Errorf("ExtractWorkflowPrivate() = %v, want %v", result, tt.expected)
			}
		})
	}
}
