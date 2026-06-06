//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShouldGeneratePRCheckoutStep(t *testing.T) {
	tests := []struct {
		name        string
		permissions string
		expected    bool
	}{
		{
			name:        "with contents read permission",
			permissions: "contents: read",
			expected:    true,
		},
		{
			name:        "with contents write permission",
			permissions: "contents: write",
			expected:    true,
		},
		{
			name:        "without contents permission",
			permissions: "issues: read",
			expected:    false,
		},
		{
			name:        "with read-all shorthand",
			permissions: "read-all",
			expected:    true,
		},
		{
			name:        "with write-all shorthand",
			permissions: "write-all",
			expected:    true,
		},
		{
			name:        "with none shorthand",
			permissions: "none",
			expected:    false,
		},
		{
			name:        "with all: read",
			permissions: `all: read`,
			expected:    true,
		},
		{
			name: "multiple permissions including contents",
			permissions: `contents: read
issues: write
pull-requests: read`,
			expected: true,
		},
		{
			name: "multiple permissions without contents",
			permissions: `issues: write
pull-requests: read`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := &WorkflowData{
				Permissions: tt.permissions,
			}
			result := ShouldGeneratePRCheckoutStep(data)
			assert.Equal(t, tt.expected, result, "ShouldGeneratePRCheckoutStep() result mismatch")
		})
	}
}

func TestGeneratePRReadyForReviewCheckout_IncludesWorkflowDispatchIssueCommentContext(t *testing.T) {
	compiler := NewCompiler()
	var yaml strings.Builder
	data := &WorkflowData{
		Permissions: "contents: read",
	}

	compiler.generatePRReadyForReviewCheckout(&yaml, data)
	rendered := yaml.String()

	assert.Contains(t, rendered, "github.event.pull_request")
	assert.Contains(t, rendered, "github.event.issue.pull_request")
	assert.Contains(t, rendered, "github.event_name == 'workflow_dispatch'")
	assert.Contains(t, rendered, "fromJSON(github.event.inputs.aw_context || '{}').item_type == 'pull_request'")
	assert.NotContains(t, rendered, "fromJSON(github.event.inputs.aw_context || '{}').event_type")
}
