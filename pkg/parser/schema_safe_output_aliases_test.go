//go:build !integration

package parser

import (
	"os"
	"testing"
)

func TestSafeOutputAliasSuggestion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		errorMessage string
		jsonPath     string
		want         string
	}{
		{
			name:         "create-issue-comment maps to add-comment",
			errorMessage: "additional properties 'create-issue-comment' not allowed",
			jsonPath:     "/safe-outputs",
			want:         "Did you mean 'add-comment'?",
		},
		{
			name:         "create_issue_comment maps to add-comment",
			errorMessage: "additional properties 'create_issue_comment' not allowed",
			jsonPath:     "/safe-outputs",
			want:         "Did you mean 'add-comment'?",
		},
		{
			name:         "add-issue-comment maps to add-comment",
			errorMessage: "additional properties 'add-issue-comment' not allowed",
			jsonPath:     "/safe-outputs",
			want:         "Did you mean 'add-comment'?",
		},
		{
			name:         "post-comment maps to add-comment",
			errorMessage: "additional properties 'post-comment' not allowed",
			jsonPath:     "/safe-outputs",
			want:         "Did you mean 'add-comment'?",
		},
		{
			name:         "truly unknown field returns empty (not an alias)",
			errorMessage: "additional properties 'totally-unknown-field' not allowed",
			jsonPath:     "/safe-outputs",
			want:         "",
		},
		{
			name:         "non safe-outputs path returns empty",
			errorMessage: "additional properties 'add_comment' not allowed",
			jsonPath:     "/on",
			want:         "",
		},
		{
			name:         "empty path returns empty",
			errorMessage: "additional properties 'add_comment' not allowed",
			jsonPath:     "",
			want:         "",
		},
		{
			name:         "non additional properties error returns empty",
			errorMessage: "value must be one of 'true', 'false'",
			jsonPath:     "/safe-outputs",
			want:         "",
		},
		{
			name:         "two aliases that map to same canonical deduplicates",
			errorMessage: "additional properties 'create-issue-comment', 'create_issue_comment' not allowed",
			jsonPath:     "/safe-outputs",
			want:         "Did you mean 'add-comment'?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := safeOutputAliasSuggestion(tt.errorMessage, tt.jsonPath)
			if got != tt.want {
				t.Errorf("safeOutputAliasSuggestion(%q, %q) = %q, want %q", tt.errorMessage, tt.jsonPath, got, tt.want)
			}
		})
	}
}

// TestSafeOutputAliasSuggestion_Integration verifies that the alias suggestion is
// surfaced through the full schema validation pipeline when an agent uses a wrong
// safe-output field name. It writes a real workflow file so the frontmatter context
// is available and the error path is resolved to /safe-outputs.
func TestSafeOutputAliasSuggestion_Integration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		yamlContent string
		safeOutputs map[string]any
		wantInErr   string
	}{
		{
			name:        "create-issue-comment rejected with add-comment suggestion",
			yamlContent: "---\non:\n  issues:\n    types: [opened]\nengine: copilot\nsafe-outputs:\n  create-issue-comment:\n    max: 5\n---\n",
			safeOutputs: map[string]any{
				"create-issue-comment": map[string]any{"max": 5},
			},
			wantInErr: "add-comment",
		},
		{
			name:        "add_comment rejected with add-comment suggestion",
			yamlContent: "---\non:\n  issues:\n    types: [opened]\nengine: copilot\nsafe-outputs:\n  add_comment:\n    max: 5\n---\n",
			safeOutputs: map[string]any{
				"add_comment": map[string]any{"max": 5},
			},
			wantInErr: "add-comment",
		},
		{
			name:        "update_issue rejected with update-issue suggestion",
			yamlContent: "---\non:\n  issues:\n    types: [opened]\nengine: copilot\nsafe-outputs:\n  update_issue:\n    body: true\n---\n",
			safeOutputs: map[string]any{
				"update_issue": map[string]any{"body": true},
			},
			wantInErr: "update-issue",
		},
		{
			name:        "create_pull_request rejected with create-pull-request suggestion",
			yamlContent: "---\non:\n  issues:\n    types: [opened]\nengine: copilot\nsafe-outputs:\n  create_pull_request:\n    max: 1\n---\n",
			safeOutputs: map[string]any{
				"create_pull_request": map[string]any{"max": 1},
			},
			wantInErr: "create-pull-request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Write a real file so readFrontmatterContext can extract the YAML for
			// precise error-location detection (enabling the /safe-outputs path).
			dir := t.TempDir()
			filePath := dir + "/workflow.md"
			if err := os.WriteFile(filePath, []byte(tt.yamlContent), 0o600); err != nil {
				t.Fatalf("failed to write test workflow file: %v", err)
			}

			frontmatter := map[string]any{
				"on":           map[string]any{"issues": map[string]any{"types": []any{"opened"}}},
				"engine":       "copilot",
				"safe-outputs": tt.safeOutputs,
			}
			validationErr := ValidateMainWorkflowFrontmatterWithSchemaAndLocation(frontmatter, filePath)
			if validationErr == nil {
				t.Fatal("expected validation to fail for unknown safe-output field")
			}
			if !contains(validationErr.Error(), tt.wantInErr) {
				t.Errorf("expected error to contain %q as alias suggestion, got: %v", tt.wantInErr, validationErr)
			}
		})
	}
}
