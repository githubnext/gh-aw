package cli

import (
	"strings"
	"testing"
)

func TestRemoveFieldFromOnTrigger(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		fieldName   string
		shouldMatch string // We'll check if the result contains this string instead of exact match
		expectError bool
	}{
		{
			name: "remove stop-after from on trigger",
			content: `---
on:
  issues:
    types: [opened]
  stop-after: "+48h"
permissions:
  contents: read
---

# Test Workflow

This is a test workflow.`,
			fieldName:   "stop-after",
			shouldMatch: "# Test Workflow",
			expectError: false,
		},
		{
			name: "no stop-after field to remove",
			content: `---
on:
  issues:
    types: [opened]
permissions:
  contents: read
---

# Test Workflow`,
			fieldName:   "stop-after",
			shouldMatch: "# Test Workflow",
			expectError: false,
		},
		{
			name: "on field is a string not a map",
			content: `---
on: push
permissions:
  contents: read
---

# Test Workflow`,
			fieldName:   "stop-after",
			shouldMatch: "on: push",
			expectError: false,
		},
		{
			name: "no on field at all",
			content: `---
permissions:
  contents: read
---

# Test Workflow`,
			fieldName:   "stop-after",
			shouldMatch: "# Test Workflow",
			expectError: false,
		},
		{
			name: "remove reaction field from on trigger",
			content: `---
on:
  issues:
    types: [opened]
  reaction: "+1"
  stop-after: "+48h"
permissions:
  contents: read
---

# Test Workflow`,
			fieldName:   "stop-after",
			shouldMatch: "reaction:",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := RemoveFieldFromOnTrigger(tt.content, tt.fieldName)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Check that result doesn't contain the field to be removed
			if strings.Contains(result, tt.fieldName+":") && tt.fieldName == "stop-after" {
				t.Errorf("Result still contains '%s' field:\n%s", tt.fieldName, result)
			}

			// Check that expected content is present
			if !strings.Contains(result, tt.shouldMatch) {
				t.Errorf("Result doesn't contain expected string '%s':\n%s", tt.shouldMatch, result)
			}
		})
	}
}
