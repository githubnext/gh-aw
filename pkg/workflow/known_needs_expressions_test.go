//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateKnownNeedsExpressions(t *testing.T) {
	tests := []struct {
		name               string
		data               *WorkflowData
		expectedMinCount   int
		expectedExprPrefix string
		checkExpressions   []string
	}{
		{
			name:             "basic activation and agent jobs",
			data:             &WorkflowData{},
			expectedMinCount: 10, // At least activation, pre_activation, detection, and agent outputs
			checkExpressions: []string{
				"needs.activation.outputs.text",
				"needs.activation.outputs.title",
				"needs.activation.outputs.body",
				"needs.pre_activation.outputs.activated",
				"needs.detection.outputs.success",
				"needs.agent.outputs.output",
				"needs.agent.outputs.output_types",
			},
		},
		{
			name: "with safe outputs",
			data: &WorkflowData{
				SafeOutputs: &SafeOutputsConfig{
					CreateIssues: &CreateIssuesConfig{},
				},
			},
			checkExpressions: []string{
				"needs.activation.outputs.text",
				"needs.create_issue.outputs.issue_url",
				"needs.create_issue.outputs.issue_number",
			},
		},
		{
			name: "with custom jobs",
			data: &WorkflowData{
				Jobs: map[string]any{
					"custom_job": map[string]any{
						"runs-on": "ubuntu-latest",
					},
				},
			},
			checkExpressions: []string{
				"needs.activation.outputs.text",
				"needs.custom_job.outputs.output",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mappings := generateKnownNeedsExpressions(tt.data)

			// Check minimum count
			assert.GreaterOrEqual(t, len(mappings), tt.expectedMinCount,
				"Should generate at least %d expressions", tt.expectedMinCount)

			// Build a map for easy lookup
			exprMap := make(map[string]*ExpressionMapping)
			for _, mapping := range mappings {
				exprMap[mapping.Content] = mapping
			}

			// Check specific expressions
			for _, expr := range tt.checkExpressions {
				mapping, found := exprMap[expr]
				assert.True(t, found, "Expected expression %s to be generated", expr)
				if found {
					assert.NotEmpty(t, mapping.EnvVar, "EnvVar should not be empty for %s", expr)
					assert.Contains(t, mapping.EnvVar, "GH_AW_NEEDS_", "EnvVar should have GH_AW_NEEDS_ prefix")
					assert.Equal(t, expr, mapping.Content, "Content should match expression")
				}
			}
		})
	}
}

func TestNormalizeJobNameForEnvVar(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"activation", "ACTIVATION"},
		{"pre_activation", "PRE_ACTIVATION"},
		{"agent", "AGENT"},
		{"my-custom-job", "MY_CUSTOM_JOB"},
		{"job_with_numbers_123", "JOB_WITH_NUMBERS_123"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeJobNameForEnvVar(tt.input)
			assert.Equal(t, tt.expected, result, "Job name normalization failed")
		})
	}
}

func TestNormalizeOutputNameForEnvVar(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"text", "TEXT"},
		{"comment_id", "COMMENT_ID"},
		{"issue_url", "ISSUE_URL"},
		{"output_types", "OUTPUT_TYPES"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeOutputNameForEnvVar(tt.input)
			assert.Equal(t, tt.expected, result, "Output name normalization failed")
		})
	}
}

func TestGetSafeOutputJobNames(t *testing.T) {
	tests := []struct {
		name         string
		data         *WorkflowData
		expectedJobs []string
	}{
		{
			name:         "no safe outputs",
			data:         &WorkflowData{},
			expectedJobs: []string{},
		},
		{
			name: "single create-issues",
			data: &WorkflowData{
				SafeOutputs: &SafeOutputsConfig{
					CreateIssues: &CreateIssuesConfig{},
				},
			},
			expectedJobs: []string{"create_issue"},
		},
		{
			name: "multiple safe output types",
			data: &WorkflowData{
				SafeOutputs: &SafeOutputsConfig{
					CreateIssues:      &CreateIssuesConfig{},
					CreateDiscussions: &CreateDiscussionsConfig{},
				},
			},
			expectedJobs: []string{"create_discussion", "create_issue", "safe_outputs"},
		},
		{
			name: "with custom safe-jobs",
			data: &WorkflowData{
				SafeOutputs: &SafeOutputsConfig{
					CreateIssues: &CreateIssuesConfig{},
					Jobs: map[string]*SafeJobConfig{
						"my_custom_job": {},
					},
				},
			},
			expectedJobs: []string{"create_issue", "my_custom_job"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jobNames := getSafeOutputJobNames(tt.data)
			assert.ElementsMatch(t, tt.expectedJobs, jobNames,
				"Safe output job names mismatch")
		})
	}
}

func TestGetCustomJobNames(t *testing.T) {
	tests := []struct {
		name         string
		data         *WorkflowData
		expectedJobs []string
	}{
		{
			name:         "no custom jobs",
			data:         &WorkflowData{},
			expectedJobs: []string{},
		},
		{
			name: "single custom job",
			data: &WorkflowData{
				Jobs: map[string]any{
					"custom_job": map[string]any{
						"runs-on": "ubuntu-latest",
					},
				},
			},
			expectedJobs: []string{"custom_job"},
		},
		{
			name: "multiple custom jobs",
			data: &WorkflowData{
				Jobs: map[string]any{
					"job_a": map[string]any{},
					"job_b": map[string]any{},
					"job_c": map[string]any{},
				},
			},
			expectedJobs: []string{"job_a", "job_b", "job_c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jobNames := getCustomJobNames(tt.data)
			assert.ElementsMatch(t, tt.expectedJobs, jobNames,
				"Custom job names mismatch")
		})
	}
}

func TestGenerateKnownNeedsExpressions_EnvVarFormat(t *testing.T) {
	data := &WorkflowData{}
	mappings := generateKnownNeedsExpressions(data)

	require.NotEmpty(t, mappings, "Should generate at least some mappings")

	// Check that all env vars follow the correct format
	for _, mapping := range mappings {
		assert.Contains(t, mapping.EnvVar, "GH_AW_NEEDS_",
			"EnvVar should start with GH_AW_NEEDS_: %s", mapping.EnvVar)
		assert.Contains(t, mapping.EnvVar, "_OUTPUTS_",
			"EnvVar should contain _OUTPUTS_: %s", mapping.EnvVar)

		// Verify the expression content matches the expected format
		assert.Contains(t, mapping.Content, "needs.",
			"Content should contain 'needs.': %s", mapping.Content)
		assert.Contains(t, mapping.Content, ".outputs.",
			"Content should contain '.outputs.': %s", mapping.Content)
	}
}
