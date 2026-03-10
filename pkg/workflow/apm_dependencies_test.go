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
		name             string
		frontmatter      map[string]any
		expectedDeps     []string
		expectedIsolated bool
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
			name: "Non-array, non-object value is ignored",
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
		{
			name: "Object format with packages only",
			frontmatter: map[string]any{
				"dependencies": map[string]any{
					"packages": []any{
						"microsoft/apm-sample-package",
						"github/awesome-copilot",
					},
				},
			},
			expectedDeps:     []string{"microsoft/apm-sample-package", "github/awesome-copilot"},
			expectedIsolated: false,
		},
		{
			name: "Object format with isolated true",
			frontmatter: map[string]any{
				"dependencies": map[string]any{
					"packages": []any{"microsoft/apm-sample-package"},
					"isolated": true,
				},
			},
			expectedDeps:     []string{"microsoft/apm-sample-package"},
			expectedIsolated: true,
		},
		{
			name: "Object format with isolated false",
			frontmatter: map[string]any{
				"dependencies": map[string]any{
					"packages": []any{"microsoft/apm-sample-package"},
					"isolated": false,
				},
			},
			expectedDeps:     []string{"microsoft/apm-sample-package"},
			expectedIsolated: false,
		},
		{
			name: "Object format with empty packages",
			frontmatter: map[string]any{
				"dependencies": map[string]any{
					"packages": []any{},
				},
			},
			expectedDeps: nil,
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
				assert.Equal(t, tt.expectedIsolated, result.Isolated, "Isolated flag should match expected")
			}
		})
	}
}

func TestEngineGetAPMTarget(t *testing.T) {
	tests := []struct {
		name     string
		engine   CodingAgentEngine
		expected string
	}{
		{name: "copilot engine returns copilot", engine: NewCopilotEngine(), expected: "copilot"},
		{name: "claude engine returns claude", engine: NewClaudeEngine(), expected: "claude"},
		{name: "codex engine returns all", engine: NewCodexEngine(), expected: "all"},
		{name: "gemini engine returns all", engine: NewGeminiEngine(), expected: "all"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.engine.GetAPMTarget()
			assert.Equal(t, tt.expected, result, "APM target should match for engine %s", tt.engine.GetID())
		})
	}
}

func TestGenerateAPMPackStep(t *testing.T) {
	tests := []struct {
		name             string
		apmDeps          *APMDependenciesInfo
		target           string
		expectedContains []string
		expectedEmpty    bool
	}{
		{
			name:          "Nil deps returns empty step",
			apmDeps:       nil,
			target:        "copilot",
			expectedEmpty: true,
		},
		{
			name:          "Empty packages returns empty step",
			apmDeps:       &APMDependenciesInfo{Packages: []string{}},
			target:        "copilot",
			expectedEmpty: true,
		},
		{
			name:    "Single dependency with copilot target",
			apmDeps: &APMDependenciesInfo{Packages: []string{"microsoft/apm-sample-package"}},
			target:  "copilot",
			expectedContains: []string{
				"Install and pack APM dependencies",
				"id: apm_pack",
				"microsoft/apm-action",
				"dependencies: |",
				"- microsoft/apm-sample-package",
				"isolated: 'true'",
				"pack: 'true'",
				"archive: 'true'",
				"target: copilot",
				"working-directory: /tmp/gh-aw/apm-workspace",
			},
		},
		{
			name:    "Multiple dependencies with claude target",
			apmDeps: &APMDependenciesInfo{Packages: []string{"microsoft/apm-sample-package", "github/skills/review"}},
			target:  "claude",
			expectedContains: []string{
				"Install and pack APM dependencies",
				"id: apm_pack",
				"microsoft/apm-action",
				"- microsoft/apm-sample-package",
				"- github/skills/review",
				"target: claude",
			},
		},
		{
			name:    "All target for non-copilot/claude engine",
			apmDeps: &APMDependenciesInfo{Packages: []string{"microsoft/apm-sample-package"}},
			target:  "all",
			expectedContains: []string{
				"target: all",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := &WorkflowData{Name: "test-workflow"}
			step := GenerateAPMPackStep(tt.apmDeps, tt.target, data)

			if tt.expectedEmpty {
				assert.Empty(t, step, "Step should be empty for empty/nil dependencies")
				return
			}

			require.NotEmpty(t, step, "Step should not be empty")

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

func TestGenerateAPMRestoreStep(t *testing.T) {
	tests := []struct {
		name                string
		apmDeps             *APMDependenciesInfo
		expectedContains    []string
		expectedNotContains []string
		expectedEmpty       bool
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
			name:    "Non-isolated restore step",
			apmDeps: &APMDependenciesInfo{Packages: []string{"microsoft/apm-sample-package"}, Isolated: false},
			expectedContains: []string{
				"Restore APM dependencies",
				"microsoft/apm-action",
				"bundle: /tmp/gh-aw/apm-bundle/*.tar.gz",
			},
			expectedNotContains: []string{"isolated"},
		},
		{
			name:    "Isolated restore step",
			apmDeps: &APMDependenciesInfo{Packages: []string{"microsoft/apm-sample-package"}, Isolated: true},
			expectedContains: []string{
				"Restore APM dependencies",
				"microsoft/apm-action",
				"bundle: /tmp/gh-aw/apm-bundle/*.tar.gz",
				"isolated: 'true'",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := &WorkflowData{Name: "test-workflow"}
			step := GenerateAPMRestoreStep(tt.apmDeps, data)

			if tt.expectedEmpty {
				assert.Empty(t, step, "Step should be empty for empty/nil dependencies")
				return
			}

			require.NotEmpty(t, step, "Step should not be empty")

			var sb strings.Builder
			for _, line := range step {
				sb.WriteString(line + "\n")
			}
			combined := sb.String()

			for _, expected := range tt.expectedContains {
				assert.Contains(t, combined, expected, "Step should contain: %s", expected)
			}
			for _, notExpected := range tt.expectedNotContains {
				assert.NotContains(t, combined, notExpected, "Step should not contain: %s", notExpected)
			}
		})
	}
}
