package workflow

import "testing"

func TestGetWorkflowIDFromPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "simple filename",
			path:     "ai-moderator.md",
			expected: "ai-moderator",
		},
		{
			name:     "full path",
			path:     "/home/user/workflows/test-workflow.md",
			expected: "test-workflow",
		},
		{
			name:     "filename with multiple dots",
			path:     "/path/to/workflow.test.md",
			expected: "workflow.test",
		},
		{
			name:     "relative path",
			path:     ".github/workflows/daily-fact.md",
			expected: "daily-fact",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetWorkflowIDFromPath(tt.path)
			if result != tt.expected {
				t.Errorf("GetWorkflowIDFromPath(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}
