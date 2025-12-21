package workflow

import (
	"strings"
	"testing"
)

func TestParseLabelTriggerShorthand(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		wantEntityType   string
		wantLabelNames   []string
		wantIsLabel      bool
		wantErr          bool
		wantErrContains  string
	}{
		{
			name:           "simple labeled with single label",
			input:          "labeled bug",
			wantEntityType: "issues",
			wantLabelNames: []string{"bug"},
			wantIsLabel:    true,
			wantErr:        false,
		},
		{
			name:           "simple labeled with multiple labels",
			input:          "labeled bug enhancement priority-high",
			wantEntityType: "issues",
			wantLabelNames: []string{"bug", "enhancement", "priority-high"},
			wantIsLabel:    true,
			wantErr:        false,
		},
		{
			name:           "issue labeled with single label",
			input:          "issue labeled bug",
			wantEntityType: "issues",
			wantLabelNames: []string{"bug"},
			wantIsLabel:    true,
			wantErr:        false,
		},
		{
			name:           "issue labeled with multiple labels",
			input:          "issue labeled bug enhancement",
			wantEntityType: "issues",
			wantLabelNames: []string{"bug", "enhancement"},
			wantIsLabel:    true,
			wantErr:        false,
		},
		{
			name:           "pull_request labeled with single label",
			input:          "pull_request labeled needs-review",
			wantEntityType: "pull_request",
			wantLabelNames: []string{"needs-review"},
			wantIsLabel:    true,
			wantErr:        false,
		},
		{
			name:           "pull_request labeled with multiple labels",
			input:          "pull_request labeled needs-review approved",
			wantEntityType: "pull_request",
			wantLabelNames: []string{"needs-review", "approved"},
			wantIsLabel:    true,
			wantErr:        false,
		},
		{
			name:           "discussion labeled with single label",
			input:          "discussion labeled question",
			wantEntityType: "discussion",
			wantLabelNames: []string{"question"},
			wantIsLabel:    true,
			wantErr:        false,
		},
		{
			name:           "discussion labeled with multiple labels",
			input:          "discussion labeled question announcement",
			wantEntityType: "discussion",
			wantLabelNames: []string{"question", "announcement"},
			wantIsLabel:    true,
			wantErr:        false,
		},
		{
			name:           "with extra whitespace",
			input:          "  labeled   bug   enhancement  ",
			wantEntityType: "issues",
			wantLabelNames: []string{"bug", "enhancement"},
			wantIsLabel:    true,
			wantErr:        false,
		},
		{
			name:            "labeled without label names",
			input:           "labeled",
			wantEntityType:  "",
			wantLabelNames:  nil,
			wantIsLabel:     true,
			wantErr:         true,
			wantErrContains: "requires at least one label name",
		},
		{
			name:            "issue labeled without label names",
			input:           "issue labeled",
			wantEntityType:  "",
			wantLabelNames:  nil,
			wantIsLabel:     true,
			wantErr:         true,
			wantErrContains: "requires at least one label name",
		},
		{
			name:           "not a label trigger - just 'issue'",
			input:          "issue",
			wantEntityType: "",
			wantLabelNames: nil,
			wantIsLabel:    false,
			wantErr:        false,
		},
		{
			name:           "not a label trigger - random text",
			input:          "push",
			wantEntityType: "",
			wantLabelNames: nil,
			wantIsLabel:    false,
			wantErr:        false,
		},
		{
			name:           "not a label trigger - schedule",
			input:          "daily at 10:00",
			wantEntityType: "",
			wantLabelNames: nil,
			wantIsLabel:    false,
			wantErr:        false,
		},
		{
			name:           "not a label trigger - slash command",
			input:          "/test",
			wantEntityType: "",
			wantLabelNames: nil,
			wantIsLabel:    false,
			wantErr:        false,
		},
		{
			name:           "labels with hyphens and underscores",
			input:          "labeled priority-high bug_fix needs_review",
			wantEntityType: "issues",
			wantLabelNames: []string{"priority-high", "bug_fix", "needs_review"},
			wantIsLabel:    true,
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entityType, labelNames, isLabel, err := parseLabelTriggerShorthand(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("parseLabelTriggerShorthand() expected error but got none")
				} else if tt.wantErrContains != "" && !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Errorf("parseLabelTriggerShorthand() error = %v, want error containing %q", err, tt.wantErrContains)
				}
				return
			}

			if err != nil {
				t.Errorf("parseLabelTriggerShorthand() unexpected error = %v", err)
				return
			}

			if isLabel != tt.wantIsLabel {
				t.Errorf("parseLabelTriggerShorthand() isLabel = %v, want %v", isLabel, tt.wantIsLabel)
			}

			if entityType != tt.wantEntityType {
				t.Errorf("parseLabelTriggerShorthand() entityType = %v, want %v", entityType, tt.wantEntityType)
			}

			if !slicesEqual(labelNames, tt.wantLabelNames) {
				t.Errorf("parseLabelTriggerShorthand() labelNames = %v, want %v", labelNames, tt.wantLabelNames)
			}
		})
	}
}

func TestExpandLabelTriggerShorthand(t *testing.T) {
	tests := []struct {
		name             string
		entityType       string
		labelNames       []string
		wantTriggerKey   string
		wantItemTypeName string
	}{
		{
			name:             "issues with single label",
			entityType:       "issues",
			labelNames:       []string{"bug"},
			wantTriggerKey:   "issues",
			wantItemTypeName: "issue",
		},
		{
			name:             "issues with multiple labels",
			entityType:       "issues",
			labelNames:       []string{"bug", "enhancement"},
			wantTriggerKey:   "issues",
			wantItemTypeName: "issue",
		},
		{
			name:             "pull_request with single label",
			entityType:       "pull_request",
			labelNames:       []string{"needs-review"},
			wantTriggerKey:   "pull_request",
			wantItemTypeName: "pull request",
		},
		{
			name:             "pull_request with multiple labels",
			entityType:       "pull_request",
			labelNames:       []string{"needs-review", "approved"},
			wantTriggerKey:   "pull_request",
			wantItemTypeName: "pull request",
		},
		{
			name:             "discussion with single label",
			entityType:       "discussion",
			labelNames:       []string{"question"},
			wantTriggerKey:   "discussion",
			wantItemTypeName: "discussion",
		},
		{
			name:             "discussion with multiple labels",
			entityType:       "discussion",
			labelNames:       []string{"question", "announcement"},
			wantTriggerKey:   "discussion",
			wantItemTypeName: "discussion",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandLabelTriggerShorthand(tt.entityType, tt.labelNames)

			// Check that the trigger key exists
			if _, exists := result[tt.wantTriggerKey]; !exists {
				t.Errorf("expandLabelTriggerShorthand() missing trigger key %q", tt.wantTriggerKey)
			}

			// Check trigger configuration
			triggerConfig, ok := result[tt.wantTriggerKey].(map[string]any)
			if !ok {
				t.Fatalf("expandLabelTriggerShorthand() trigger config is not a map")
			}

			// Check types field
			types, ok := triggerConfig["types"].([]any)
			if !ok {
				t.Fatalf("expandLabelTriggerShorthand() types is not an array")
			}
			if len(types) != 1 || types[0] != "labeled" {
				t.Errorf("expandLabelTriggerShorthand() types = %v, want [labeled]", types)
			}

			// Check names field
			names, ok := triggerConfig["names"].([]string)
			if !ok {
				t.Fatalf("expandLabelTriggerShorthand() names is not a string array")
			}
			if !slicesEqual(names, tt.labelNames) {
				t.Errorf("expandLabelTriggerShorthand() names = %v, want %v", names, tt.labelNames)
			}

			// Check workflow_dispatch
			if _, exists := result["workflow_dispatch"]; !exists {
				t.Errorf("expandLabelTriggerShorthand() missing workflow_dispatch")
			}

			dispatchConfig, ok := result["workflow_dispatch"].(map[string]any)
			if !ok {
				t.Fatalf("expandLabelTriggerShorthand() workflow_dispatch is not a map")
			}

			// Check inputs
			inputs, ok := dispatchConfig["inputs"].(map[string]any)
			if !ok {
				t.Fatalf("expandLabelTriggerShorthand() inputs is not a map")
			}

			// Check item_number input
			itemNumber, ok := inputs["item_number"].(map[string]any)
			if !ok {
				t.Fatalf("expandLabelTriggerShorthand() item_number is not a map")
			}

			description, ok := itemNumber["description"].(string)
			if !ok {
				t.Fatalf("expandLabelTriggerShorthand() description is not a string")
			}

			if !strings.Contains(description, tt.wantItemTypeName) {
				t.Errorf("expandLabelTriggerShorthand() description = %q, want to contain %q", description, tt.wantItemTypeName)
			}

			required, ok := itemNumber["required"].(bool)
			if !ok || !required {
				t.Errorf("expandLabelTriggerShorthand() required = %v, want true", required)
			}

			inputType, ok := itemNumber["type"].(string)
			if !ok || inputType != "string" {
				t.Errorf("expandLabelTriggerShorthand() type = %v, want 'string'", inputType)
			}
		})
	}
}

func TestGetItemTypeName(t *testing.T) {
	tests := []struct {
		entityType string
		want       string
	}{
		{"issues", "issue"},
		{"pull_request", "pull request"},
		{"discussion", "discussion"},
		{"unknown", "item"},
	}

	for _, tt := range tests {
		t.Run(tt.entityType, func(t *testing.T) {
			got := getItemTypeName(tt.entityType)
			if got != tt.want {
				t.Errorf("getItemTypeName(%q) = %q, want %q", tt.entityType, got, tt.want)
			}
		})
	}
}

// Helper function to compare string slices
func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
