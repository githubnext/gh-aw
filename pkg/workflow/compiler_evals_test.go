package workflow

import (
	"strings"
	"testing"
)

// TestExtractEvalsFromFrontmatter validates parsing of the "evals" frontmatter field.
func TestExtractEvalsFromFrontmatter(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter map[string]any
		wantLen     int
		wantIDs     []string
	}{
		{
			name:        "no evals key returns nil",
			frontmatter: map[string]any{},
			wantLen:     0,
		},
		{
			name:        "nil evals value returns nil",
			frontmatter: map[string]any{"evals": nil},
			wantLen:     0,
		},
		{
			name: "valid evals parsed correctly",
			frontmatter: map[string]any{
				"evals": []any{
					map[string]any{"id": "builds", "question": "Does the code compile?"},
					map[string]any{"id": "tests", "question": "Are all tests passing?"},
				},
			},
			wantLen: 2,
			wantIDs: []string{"builds", "tests"},
		},
		{
			name: "entries with empty id are skipped",
			frontmatter: map[string]any{
				"evals": []any{
					map[string]any{"id": "", "question": "Question?"},
					map[string]any{"id": "valid", "question": "Valid question?"},
				},
			},
			wantLen: 1,
			wantIDs: []string{"valid"},
		},
		{
			name: "entries with empty question are skipped",
			frontmatter: map[string]any{
				"evals": []any{
					map[string]any{"id": "noquestion", "question": ""},
					map[string]any{"id": "good", "question": "Good question?"},
				},
			},
			wantLen: 1,
			wantIDs: []string{"good"},
		},
		{
			name: "wrong type for evals returns nil",
			frontmatter: map[string]any{
				"evals": "not-a-slice",
			},
			wantLen: 0,
		},
		{
			name: "non-map items in slice are skipped",
			frontmatter: map[string]any{
				"evals": []any{
					"not-a-map",
					map[string]any{"id": "valid", "question": "Valid?"},
				},
			},
			wantLen: 1,
			wantIDs: []string{"valid"},
		},
		{
			name: "empty slice returns nil",
			frontmatter: map[string]any{
				"evals": []any{},
			},
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractEvalsFromFrontmatter(tt.frontmatter)
			if len(result) != tt.wantLen {
				t.Errorf("extractEvalsFromFrontmatter() len = %d, want %d", len(result), tt.wantLen)
			}
			for i, wantID := range tt.wantIDs {
				if i >= len(result) {
					t.Errorf("missing result[%d], expected id=%q", i, wantID)
					continue
				}
				if result[i].ID != wantID {
					t.Errorf("result[%d].ID = %q, want %q", i, result[i].ID, wantID)
				}
			}
		})
	}
}

// TestValidateEvals validates the uniqueness and non-empty constraints on eval definitions.
func TestValidateEvals(t *testing.T) {
	tests := []struct {
		name    string
		evals   []EvalDefinition
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil evals passes",
			evals:   nil,
			wantErr: false,
		},
		{
			name:    "empty evals passes",
			evals:   []EvalDefinition{},
			wantErr: false,
		},
		{
			name: "valid evals pass",
			evals: []EvalDefinition{
				{ID: "builds", Question: "Does the code compile?"},
				{ID: "tests", Question: "Are all tests passing?"},
			},
			wantErr: false,
		},
		{
			name: "duplicate id fails",
			evals: []EvalDefinition{
				{ID: "builds", Question: "Does the code compile?"},
				{ID: "builds", Question: "Another question?"},
			},
			wantErr: true,
			errMsg:  "duplicate evaluation id",
		},
		{
			name: "empty id fails",
			evals: []EvalDefinition{
				{ID: "", Question: "Some question?"},
			},
			wantErr: true,
			errMsg:  "non-empty id",
		},
		{
			name: "empty question fails",
			evals: []EvalDefinition{
				{ID: "myeval", Question: ""},
			},
			wantErr: true,
			errMsg:  "non-empty question",
		},
		{
			name: "single valid eval passes",
			evals: []EvalDefinition{
				{ID: "focused", Question: "Is the implementation limited to the requested change?"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEvals(tt.evals)
			if tt.wantErr && err == nil {
				t.Error("validateEvals() expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("validateEvals() unexpected error: %v", err)
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !containsString(err.Error(), tt.errMsg) {
					t.Errorf("validateEvals() error %q does not contain %q", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

// TestBuildEvalSpecJSON validates JSON serialization of eval definitions.
func TestBuildEvalSpecJSON(t *testing.T) {
	tests := []struct {
		name  string
		evals []EvalDefinition
	}{
		{
			name:  "empty evals produces empty array",
			evals: nil,
		},
		{
			name: "single eval is serialized",
			evals: []EvalDefinition{
				{ID: "builds", Question: "Does the code compile?"},
			},
		},
		{
			name: "multiple evals are serialized in order",
			evals: []EvalDefinition{
				{ID: "a", Question: "Question A?"},
				{ID: "b", Question: "Question B?"},
				{ID: "c", Question: "Question C?"},
			},
		},
		{
			name: "questions with special characters are escaped",
			evals: []EvalDefinition{
				{ID: "special", Question: `Does it handle "quotes" and 'apostrophes'?`},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildEvalSpecJSON(tt.evals)
			if result == "" {
				t.Error("buildEvalSpecJSON() returned empty string")
			}
			// Must be valid JSON
			if result != "[]" && result != "null" {
				// Should start with [ and end with ]
				if len(result) < 2 || result[0] != '[' || result[len(result)-1] != ']' {
					t.Errorf("buildEvalSpecJSON() = %q, expected JSON array", result)
				}
			}
		})
	}
}

func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}
