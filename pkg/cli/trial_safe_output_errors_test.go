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
