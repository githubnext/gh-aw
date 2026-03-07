//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractAPMDependenciesFromFrontmatter(t *testing.T) {
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
			name: "Single dependency (array format)",
			frontmatter: map[string]any{
				"dependencies": []any{"microsoft/apm-sample-package"},
			},
			expectedDeps: []string{"microsoft/apm-sample-package"},
		},
		{
			name: "Multiple dependencies (array format)",
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
			result := extractAPMDependenciesFromFrontmatter(tt.frontmatter)
			if tt.expectedDeps == nil {
				assert.Nil(t, result, "Should return nil for no dependencies")
			} else {
				require.NotNil(t, result, "Should return non-nil APMDependenciesInfo")
				assert.Equal(t, tt.expectedDeps, result.Packages, "Extracted packages should match expected")
			}
		})
	}
}

func TestGenerateAPMDependenciesStep(t *testing.T) {
	tests := []struct {
		name             string
		apmDeps          *APMDependenciesInfo
		expectedContains []string
		expectedEmpty    bool
	}{
		{
			name:          "Nil deps returns empty step",
			apmDeps:       nil,
			expectedEmpty: true,
		},
		{
			name:          "Empty packages returns empty step",
			apmDeps:       &APMDependenciesInfo{Packages: []string{}},
			expectedEmpty: true,
		},
		{
			name:    "Single dependency",
			apmDeps: &APMDependenciesInfo{Packages: []string{"microsoft/apm-sample-package"}},
			expectedContains: []string{
				"Install APM dependencies",
				"microsoft/apm-action",
				"dependencies: |",
				"- microsoft/apm-sample-package",
			},
		},
		{
			name: "Multiple dependencies",
			apmDeps: &APMDependenciesInfo{
				Packages: []string{
					"microsoft/apm-sample-package",
					"github/awesome-copilot/skills/review-and-refactor",
				},
			},
			expectedContains: []string{
				"Install APM dependencies",
				"microsoft/apm-action",
				"dependencies: |",
				"- microsoft/apm-sample-package",
				"- github/awesome-copilot/skills/review-and-refactor",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := &WorkflowData{Name: "test-workflow"}
			step := GenerateAPMDependenciesStep(tt.apmDeps, data)

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
	deps := &APMDependenciesInfo{
		Packages: []string{
			"microsoft/apm-sample-package",
			"github/awesome-copilot/skills/review-and-refactor",
		},
	}
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
	assert.Contains(t, combined, "microsoft/apm-action", "Should reference microsoft/apm-action action")
	assert.Contains(t, combined, "dependencies: |", "Should use YAML block scalar for dependencies")
	assert.Contains(t, combined, "            - microsoft/apm-sample-package", "First dep should be properly indented")
	assert.Contains(t, combined, "            - github/awesome-copilot/skills/review-and-refactor", "Second dep should be properly indented")
}
