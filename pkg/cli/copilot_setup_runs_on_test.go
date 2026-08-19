//go:build !integration

package cli

import "testing"

// TestValidateCopilotSetupStepsRunsOn exercises all branches of
// validateCopilotSetupStepsRunsOn: missing key, string variants, slice
// variants, map (group/labels) variants, and unsupported types.
func TestValidateCopilotSetupStepsRunsOn(t *testing.T) {
	tests := []struct {
		name    string
		job     map[string]any
		wantErr bool
	}{
		{
			name:    "missing runs-on key",
			job:     map[string]any{},
			wantErr: true,
		},
		{
			name:    "non-empty string",
			job:     map[string]any{"runs-on": "ubuntu-latest"},
			wantErr: false,
		},
		{
			name:    "empty string",
			job:     map[string]any{"runs-on": ""},
			wantErr: true,
		},
		{
			name:    "whitespace-only string",
			job:     map[string]any{"runs-on": "   "},
			wantErr: true,
		},
		{
			name:    "slice with a non-empty label",
			job:     map[string]any{"runs-on": []any{"self-hosted", "linux"}},
			wantErr: false,
		},
		{
			name:    "slice with only empty/whitespace labels",
			job:     map[string]any{"runs-on": []any{"", "   "}},
			wantErr: true,
		},
		{
			name:    "slice with non-string items only",
			job:     map[string]any{"runs-on": []any{1, true, nil}},
			wantErr: true,
		},
		{
			name:    "empty slice",
			job:     map[string]any{"runs-on": []any{}},
			wantErr: true,
		},
		{
			name:    "map with non-empty group",
			job:     map[string]any{"runs-on": map[string]any{"group": "my-group"}},
			wantErr: false,
		},
		{
			name:    "map with empty group falls through to labels string",
			job:     map[string]any{"runs-on": map[string]any{"group": "", "labels": "linux"}},
			wantErr: false,
		},
		{
			name:    "map with empty group and empty labels string",
			job:     map[string]any{"runs-on": map[string]any{"group": "", "labels": ""}},
			wantErr: true,
		},
		{
			name:    "map with labels as non-empty slice",
			job:     map[string]any{"runs-on": map[string]any{"labels": []any{"linux", "x64"}}},
			wantErr: false,
		},
		{
			name:    "map with labels as slice of only empty strings",
			job:     map[string]any{"runs-on": map[string]any{"labels": []any{"", "  "}}},
			wantErr: true,
		},
		{
			name:    "map with labels as slice of non-string items",
			job:     map[string]any{"runs-on": map[string]any{"labels": []any{1, false}}},
			wantErr: true,
		},
		{
			name:    "map with no group and no labels key",
			job:     map[string]any{"runs-on": map[string]any{}},
			wantErr: true,
		},
		{
			name:    "map with labels of unsupported type",
			job:     map[string]any{"runs-on": map[string]any{"labels": 42}},
			wantErr: true,
		},
		{
			name:    "unsupported type (int)",
			job:     map[string]any{"runs-on": 42},
			wantErr: true,
		},
		{
			name:    "unsupported type (nil)",
			job:     map[string]any{"runs-on": nil},
			wantErr: true,
		},
		{
			name:    "unsupported type (bool)",
			job:     map[string]any{"runs-on": true},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCopilotSetupStepsRunsOn(tt.job)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}
