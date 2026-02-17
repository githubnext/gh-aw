//go:build !integration

package cli

import (
	"testing"
)

func TestIsRepoOnlySpec(t *testing.T) {
	tests := []struct {
		name     string
		spec     string
		expected bool
	}{
		{
			name:     "repo only without version",
			spec:     "githubnext/agentics",
			expected: true,
		},
		{
			name:     "repo only with version",
			spec:     "githubnext/agentics@v1.0.0",
			expected: true,
		},
		{
			name:     "full spec with workflow",
			spec:     "githubnext/agentics/ci-doctor",
			expected: false,
		},
		{
			name:     "full spec with workflow and version",
			spec:     "githubnext/agentics/ci-doctor@main",
			expected: false,
		},
		{
			name:     "full spec with path",
			spec:     "githubnext/agentics/workflows/ci-doctor.md",
			expected: false,
		},
		{
			name:     "GitHub URL",
			spec:     "https://github.com/githubnext/agentics/blob/main/workflows/ci-doctor.md",
			expected: false,
		},
		{
			name:     "local path",
			spec:     "./workflows/my-workflow.md",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRepoOnlySpec(tt.spec)
			if result != tt.expected {
				t.Errorf("isRepoOnlySpec(%q) = %v, want %v", tt.spec, result, tt.expected)
			}
		})
	}
}
