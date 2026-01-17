package parser

import (
"strings"
"testing"
)

// TestForbiddenFieldsInSharedWorkflows verifies each forbidden field is properly rejected
func TestForbiddenFieldsInSharedWorkflows(t *testing.T) {
forbiddenFields := []string{
"on", "bots", "cache", "command", "concurrency", "container",
"env", "environment", "features", "github-token", "if", "imports",
"labels", "name", "post-steps", "roles", "run-name", "runs-on",
"sandbox", "source", "strict", "timeout-minutes", "timeout_minutes",
"tracker-id",
}

for _, field := range forbiddenFields {
t.Run("reject_"+field, func(t *testing.T) {
frontmatter := map[string]any{
field:  "test-value",
"tools": map[string]any{"bash": true},
}

err := ValidateIncludedFileFrontmatterWithSchema(frontmatter)
if err == nil {
t.Errorf("Expected error for forbidden field '%s', got nil", field)
}

if err != nil && !strings.Contains(err.Error(), "cannot be used in shared workflows") {
t.Errorf("Error message should mention shared workflows, got: %v", err)
}
})
}
}

// TestAllowedFieldsInSharedWorkflows verifies allowed fields work correctly
func TestAllowedFieldsInSharedWorkflows(t *testing.T) {
allowedFields := map[string]any{
"tools":          map[string]any{"bash": true},
"engine":         "copilot",
"network":        map[string]any{"allowed": []string{"defaults"}},
"mcp-servers":    map[string]any{},
"permissions":    "read-all",
"runtimes":       map[string]any{"node": map[string]any{"version": "20"}},
"safe-outputs":   map[string]any{},
"safe-inputs":    map[string]any{},
"services":       map[string]any{},
"steps":          []any{},
"secret-masking": true,
"jobs":           map[string]any{"test": map[string]any{"runs-on": "ubuntu-latest", "steps": []any{map[string]any{"run": "echo test"}}}},
"description":    "test",
"metadata":       map[string]any{},
"inputs":         map[string]any{},
}

for field, value := range allowedFields {
t.Run("allow_"+field, func(t *testing.T) {
frontmatter := map[string]any{
field: value,
}

err := ValidateIncludedFileFrontmatterWithSchema(frontmatter)
if err != nil && strings.Contains(err.Error(), "cannot be used in shared workflows") {
t.Errorf("Field '%s' should be allowed in shared workflows, got error: %v", field, err)
}
})
}
}
