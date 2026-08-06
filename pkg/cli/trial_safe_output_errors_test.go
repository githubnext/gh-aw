package cli

import "testing"

func TestExtractSafeOutputErrors(t *testing.T) {
	tests := []struct {
		name        string
		safeOutputs map[string]any
		want        []string
	}{
		{
			name:        "nil safe outputs",
			safeOutputs: nil,
			want:        nil,
		},
		{
			name:        "no errors key",
			safeOutputs: map[string]any{"items": []any{}},
			want:        nil,
		},
		{
			name:        "empty errors array",
			safeOutputs: map[string]any{"items": []any{}, "errors": []any{}},
			want:        nil,
		},
		{
			name: "non-empty errors array",
			safeOutputs: map[string]any{
				"items":  []any{},
				"errors": []any{"Line 1: set_issue_field requires at least one of: 'field_name', 'field_node_id' fields"},
			},
			want: []string{"Line 1: set_issue_field requires at least one of: 'field_name', 'field_node_id' fields"},
		},
		{
			name: "multiple errors",
			safeOutputs: map[string]any{
				"errors": []any{"error one", "error two"},
			},
			want: []string{"error one", "error two"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractSafeOutputErrors(tt.safeOutputs)
			if len(got) != len(tt.want) {
				t.Fatalf("extractSafeOutputErrors() = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("extractSafeOutputErrors()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestWorkflowTrialResultSuccessField(t *testing.T) {
	result := WorkflowTrialResult{
		WorkflowName: "test-workflow",
		SafeOutputErrors: []string{
			"Line 1: some validation error",
		},
		Success: false,
	}
	if result.Success {
		t.Error("expected Success to be false when SafeOutputErrors is non-empty")
	}

	successResult := WorkflowTrialResult{
		WorkflowName: "test-workflow",
		Success:      true,
	}
	if !successResult.Success {
		t.Error("expected Success to be true when there are no safe-output errors")
	}
}

func TestAggregateTrialResults(t *testing.T) {
	tests := []struct {
		name              string
		results           []WorkflowTrialResult
		wantSuccess       bool
		wantTotalRejected int
		wantFirstError    string
	}{
		{
			name:              "no results",
			results:           nil,
			wantSuccess:       true,
			wantTotalRejected: 0,
			wantFirstError:    "",
		},
		{
			name: "all successful",
			results: []WorkflowTrialResult{
				{WorkflowName: "a", Success: true},
				{WorkflowName: "b", Success: true},
			},
			wantSuccess:       true,
			wantTotalRejected: 0,
			wantFirstError:    "",
		},
		{
			name: "one failure with rejected messages",
			results: []WorkflowTrialResult{
				{WorkflowName: "a", Success: true},
				{
					WorkflowName:     "b",
					Success:          false,
					SafeOutputErrors: []string{"first error", "second error"},
				},
			},
			wantSuccess:       false,
			wantTotalRejected: 2,
			wantFirstError:    "first error",
		},
		{
			name: "multiple failures aggregate total and keep first error in order",
			results: []WorkflowTrialResult{
				{
					WorkflowName:     "a",
					Success:          false,
					SafeOutputErrors: []string{"error from a"},
				},
				{
					WorkflowName:     "b",
					Success:          false,
					SafeOutputErrors: []string{"error from b1", "error from b2"},
				},
			},
			wantSuccess:       false,
			wantTotalRejected: 3,
			wantFirstError:    "error from a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSuccess, gotTotalRejected, gotFirstError := aggregateTrialResults(tt.results)
			if gotSuccess != tt.wantSuccess {
				t.Errorf("aggregateTrialResults() success = %v, want %v", gotSuccess, tt.wantSuccess)
			}
			if gotTotalRejected != tt.wantTotalRejected {
				t.Errorf("aggregateTrialResults() totalRejected = %d, want %d", gotTotalRejected, tt.wantTotalRejected)
			}
			if gotFirstError != tt.wantFirstError {
				t.Errorf("aggregateTrialResults() firstErrorMessage = %q, want %q", gotFirstError, tt.wantFirstError)
			}
		})
	}
}

func TestSanitizeControlChars(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty string", in: "", want: ""},
		{name: "plain text unchanged", in: "plain error message", want: "plain error message"},
		{
			name: "escapes ANSI escape sequence",
			in:   "before\x1b[31mred\x1b[0mafter",
			want: `before'\x1b'[31mred'\x1b'[0mafter`,
		},
		{
			name: "escapes newline and tab",
			in:   "line1\nline2\ttabbed",
			want: `line1'\n'line2'\t'tabbed`,
		},
		{
			name: "escapes carriage return",
			in:   "before\rafter",
			want: `before'\r'after`,
		},
		{
			name: "escapes C1 control character",
			in:   "before\u009bafter",
			want: `before'\u009b'after`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeControlChars(tt.in)
			if got != tt.want {
				t.Errorf("sanitizeControlChars(%q) = %q, want %q", tt.in, got, tt.want)
			}
			for _, r := range got {
				if isControlRune(r) {
					t.Errorf("sanitizeControlChars(%q) result %q still contains raw control character", tt.in, got)
				}
			}
		})
	}
}
