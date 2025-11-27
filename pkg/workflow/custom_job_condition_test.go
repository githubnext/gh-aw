package workflow

import "testing"

func TestReferencesCustomJobOutputs(t *testing.T) {
	c := &Compiler{}

	tests := []struct {
		name       string
		condition  string
		customJobs map[string]any
		expected   bool
	}{
		{
			name:       "empty condition",
			condition:  "",
			customJobs: map[string]any{"ast_grep": nil},
			expected:   false,
		},
		{
			name:       "no custom jobs",
			condition:  "needs.ast_grep.outputs.found_patterns == 'true'",
			customJobs: nil,
			expected:   false,
		},
		{
			name:       "references custom job output",
			condition:  "needs.ast_grep.outputs.found_patterns == 'true'",
			customJobs: map[string]any{"ast_grep": nil},
			expected:   true,
		},
		{
			name:       "references custom job result",
			condition:  "needs.my_job.result == 'success'",
			customJobs: map[string]any{"my_job": nil},
			expected:   true,
		},
		{
			name:       "does not reference custom job",
			condition:  "github.event.action == 'opened'",
			customJobs: map[string]any{"ast_grep": nil},
			expected:   false,
		},
		{
			name:       "references standard job not custom",
			condition:  "needs.activation.outputs.text != ''",
			customJobs: map[string]any{"ast_grep": nil},
			expected:   false,
		},
		{
			name:       "complex condition with custom job",
			condition:  "(needs.pre_activation.outputs.activated == 'true') && (needs.ast_grep.outputs.found_patterns == 'true')",
			customJobs: map[string]any{"ast_grep": nil},
			expected:   true,
		},
		{
			name:       "multiple custom jobs but only one referenced",
			condition:  "needs.job_a.outputs.done == 'true'",
			customJobs: map[string]any{"job_a": nil, "job_b": nil, "job_c": nil},
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.referencesCustomJobOutputs(tt.condition, tt.customJobs)
			if result != tt.expected {
				t.Errorf("referencesCustomJobOutputs(%q, %v) = %v, want %v", tt.condition, tt.customJobs, result, tt.expected)
			}
		})
	}
}
