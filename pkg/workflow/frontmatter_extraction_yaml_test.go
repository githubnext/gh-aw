//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsGitHubAppNestedField(t *testing.T) {
	tests := []struct {
		name string
		line string
		want bool
	}{
		{name: "app id field", line: "app-id: 123", want: true},
		{name: "client id field", line: "client-id: abc", want: true},
		{name: "private key field", line: "private-key: ${{ secrets.KEY }}", want: true},
		{name: "ignore-if-missing field", line: "ignore-if-missing: true", want: true},
		{name: "owner field", line: "owner: octocat", want: true},
		{name: "repositories field", line: "repositories:", want: true},
		{name: "array item", line: "- gh-aw", want: true},
		{name: "non-matching field", line: "ignored-field: value", want: false},
		{name: "partial field name", line: "app-idd: 123", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isGitHubAppNestedField(tt.line)
			assert.Equal(t, tt.want, got, "isGitHubAppNestedField(%q) should return %t", tt.line, tt.want)
		})
	}
}

func TestIsValidWorkflowRunConclusion(t *testing.T) {
	tests := []struct {
		name       string
		conclusion string
		want       bool
	}{
		{name: "success", conclusion: "success", want: true},
		{name: "failure", conclusion: "failure", want: true},
		{name: "neutral", conclusion: "neutral", want: true},
		{name: "cancelled", conclusion: "cancelled", want: true},
		{name: "skipped", conclusion: "skipped", want: true},
		{name: "timed_out", conclusion: "timed_out", want: true},
		{name: "action_required", conclusion: "action_required", want: true},
		{name: "stale", conclusion: "stale", want: true},
		{name: "unknown value", conclusion: "done", want: false},
		{name: "expression injection attempt", conclusion: "success' || contains(github.event.comment.body, 'x') || '", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidWorkflowRunConclusion(tt.conclusion)
			assert.Equal(t, tt.want, got, "isValidWorkflowRunConclusion(%q) should return %t", tt.conclusion, tt.want)
		})
	}
}

func TestIndentYAMLLines(t *testing.T) {
	compiler := NewCompiler()

	tests := []struct {
		name    string
		input   string
		indent  string
		wantOut string
	}{
		{
			name:    "empty input",
			input:   "",
			indent:  "  ",
			wantOut: "",
		},
		{
			name:    "single line",
			input:   "name: test",
			indent:  "  ",
			wantOut: "name: test",
		},
		{
			name:    "multi line",
			input:   "name: test\nruns-on: ubuntu-latest\nsteps:",
			indent:  "  ",
			wantOut: "name: test\n  runs-on: ubuntu-latest\n  steps:",
		},
		{
			name:    "blank lines are preserved without extra indentation",
			input:   "first: value\n\nsecond: value\n  \nthird: value",
			indent:  "  ",
			wantOut: "first: value\n\n  second: value\n  \n  third: value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compiler.indentYAMLLines(tt.input, tt.indent)
			assert.Equal(t, tt.wantOut, got, "indentYAMLLines should preserve formatting for %q", tt.name)
		})
	}
}

func TestExtractDeploymentStatusStateCondition(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter map[string]any
		want        string
	}{
		{
			name:        "missing on section",
			frontmatter: map[string]any{},
			want:        "",
		},
		{
			name: "single state",
			frontmatter: map[string]any{
				"on": map[string]any{
					"deployment_status": map[string]any{
						"state": "success",
					},
				},
			},
			want: "github.event_name != 'deployment_status' || (github.event.deployment_status.state == 'success')",
		},
		{
			name: "multiple states",
			frontmatter: map[string]any{
				"on": map[string]any{
					"deployment_status": map[string]any{
						"state": []any{"success", "failure"},
					},
				},
			},
			want: "github.event_name != 'deployment_status' || (github.event.deployment_status.state == 'success' || github.event.deployment_status.state == 'failure')",
		},
		{
			name: "mixed array keeps only strings",
			frontmatter: map[string]any{
				"on": map[string]any{
					"deployment_status": map[string]any{
						"state": []any{"success", 42},
					},
				},
			},
			want: "github.event_name != 'deployment_status' || (github.event.deployment_status.state == 'success')",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractDeploymentStatusStateCondition(tt.frontmatter)
			assert.Equal(t, tt.want, got, "extractDeploymentStatusStateCondition should match expected expression for %q", tt.name)
		})
	}
}

func TestExtractWorkflowRunConclusionConditionHelper(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter map[string]any
		want        string
		wantErr     bool
	}{
		{
			name:        "missing on section",
			frontmatter: map[string]any{},
			want:        "",
			wantErr:     false,
		},
		{
			name: "single valid conclusion",
			frontmatter: map[string]any{
				"on": map[string]any{
					"workflow_run": map[string]any{
						"conclusion": "failure",
					},
				},
			},
			want:    "github.event_name != 'workflow_run' || (github.event.workflow_run.conclusion == 'failure')",
			wantErr: false,
		},
		{
			name: "multiple valid conclusions",
			frontmatter: map[string]any{
				"on": map[string]any{
					"workflow_run": map[string]any{
						"conclusion": []any{"failure", "timed_out"},
					},
				},
			},
			want:    "github.event_name != 'workflow_run' || (github.event.workflow_run.conclusion == 'failure' || github.event.workflow_run.conclusion == 'timed_out')",
			wantErr: false,
		},
		{
			name: "invalid conclusion rejects injection attempt",
			frontmatter: map[string]any{
				"on": map[string]any{
					"workflow_run": map[string]any{
						"conclusion": "failure' || github.actor == 'attacker' || '",
					},
				},
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractWorkflowRunConclusionCondition(tt.frontmatter)
			if tt.wantErr {
				require.Error(t, err, "extractWorkflowRunConclusionCondition should reject invalid conclusion for %q", tt.name)
			} else {
				require.NoError(t, err, "extractWorkflowRunConclusionCondition should not return error for %q", tt.name)
			}
			assert.Equal(t, tt.want, got, "extractWorkflowRunConclusionCondition should return expected expression for %q", tt.name)
		})
	}
}
