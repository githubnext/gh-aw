//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractDependenciesFromFrontmatter(t *testing.T) {
	tests := []struct {
		name         string
		frontmatter  map[string]any
		expectedDeps []string
	}{
		{
			name: "No dependencies field",
			frontmatter: map[string]any{
				"engine": "copilot",
			},
			expectedDeps: nil,
		},
		{
			name: "Single dependency",
			frontmatter: map[string]any{
				"dependencies": []any{"microsoft/apm-sample-package"},
			},
			expectedDeps: []string{"microsoft/apm-sample-package"},
		},
		{
			name: "Multiple dependencies",
			frontmatter: map[string]any{
				"dependencies": []any{
					"microsoft/apm-sample-package",
					"github/awesome-copilot/skills/review-and-refactor",
					"anthropics/skills/skills/frontend-design",
				},
			},
			expectedDeps: []string{
				"microsoft/apm-sample-package",
				"github/awesome-copilot/skills/review-and-refactor",
				"anthropics/skills/skills/frontend-design",
			},
		},
		{
			name: "Empty array",
			frontmatter: map[string]any{
				"dependencies": []any{},
			},
			expectedDeps: nil,
		},
		{
			name: "Non-array value is ignored",
			frontmatter: map[string]any{
				"dependencies": "microsoft/apm-sample-package",
			},
			expectedDeps: nil,
		},
		{
			name: "Empty string items are skipped",
			frontmatter: map[string]any{
				"dependencies": []any{"microsoft/apm-sample-package", "", "github/awesome-copilot"},
			},
			expectedDeps: []string{"microsoft/apm-sample-package", "github/awesome-copilot"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractDependenciesFromFrontmatter(tt.frontmatter)
			assert.Equal(t, tt.expectedDeps, result, "Extracted dependencies should match expected")
		})
	}
}

func TestGenerateAPMDependenciesStep(t *testing.T) {
	tests := []struct {
		name             string
		dependencies     []string
		expectedContains []string
		expectedEmpty    bool
	}{
		{
			name:          "Empty dependencies returns empty step",
			dependencies:  []string{},
			expectedEmpty: true,
		},
		{
			name:          "Nil dependencies returns empty step",
			dependencies:  nil,
			expectedEmpty: true,
		},
		{
			name:         "Single dependency",
			dependencies: []string{"microsoft/apm-sample-package"},
			expectedContains: []string{
				"Install APM dependencies",
				"microsoft/apm-setup",
				"dependencies: |",
				"- microsoft/apm-sample-package",
			},
		},
		{
			name: "Multiple dependencies",
			dependencies: []string{
				"microsoft/apm-sample-package",
				"github/awesome-copilot/skills/review-and-refactor",
			},
			expectedContains: []string{
				"Install APM dependencies",
				"microsoft/apm-setup",
				"dependencies: |",
				"- microsoft/apm-sample-package",
				"- github/awesome-copilot/skills/review-and-refactor",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := &WorkflowData{Name: "test-workflow"}
			step := GenerateAPMDependenciesStep(tt.dependencies, data)

			if tt.expectedEmpty {
				assert.Empty(t, step, "Step should be empty for empty/nil dependencies")
				return
			}

			require.NotEmpty(t, step, "Step should not be empty")

			// Combine all lines for easier assertion
			var sb strings.Builder
			for _, line := range step {
				sb.WriteString(line + "\n")
			}
			combined := sb.String()

			for _, expected := range tt.expectedContains {
				assert.Contains(t, combined, expected, "Step should contain: %s", expected)
			}
		})
	}
}

func TestAPMDependenciesStepFormat(t *testing.T) {
	deps := []string{"microsoft/apm-sample-package", "github/awesome-copilot/skills/review-and-refactor"}
	data := &WorkflowData{Name: "test-workflow"}
	step := GenerateAPMDependenciesStep(deps, data)

	require.NotEmpty(t, step, "Step should not be empty")

	// Combine all lines for easy assertion
	var sb strings.Builder
	for _, line := range step {
		sb.WriteString(line + "\n")
	}
	combined := sb.String()

	// Verify the step has the correct structure
	assert.Contains(t, combined, "- name: Install APM dependencies", "Should have correct step name")
	assert.Contains(t, combined, "uses:", "Should have uses line")
	assert.Contains(t, combined, "microsoft/apm-setup", "Should reference microsoft/apm-setup action")
	assert.Contains(t, combined, "dependencies: |", "Should use YAML block scalar for dependencies")
	assert.Contains(t, combined, "            - microsoft/apm-sample-package", "First dep should be properly indented")
	assert.Contains(t, combined, "            - github/awesome-copilot/skills/review-and-refactor", "Second dep should be properly indented")
}
