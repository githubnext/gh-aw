package workflow

import (
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
)

// TestYAMLBooleanKeywordNotInterpreted tests that YAML boolean keywords
// like "on" are correctly quoted in generated YAML and not interpreted as booleans
func TestYAMLBooleanKeywordNotInterpreted(t *testing.T) {
	compiler := NewCompiler(false, "", "test")
	compiler.SetSkipValidation(true) // Skip schema validation for this test

	// Create a simple workflow with an "on" section
	frontmatter := map[string]any{
		"on": map[string]any{
			"push": map[string]any{
				"branches": []string{"main"},
			},
		},
		"permissions": map[string]any{
			"contents": "read",
		},
		"tools": map[string]any{
			"github": map[string]any{
				"allowed": []any{"list_issues"},
			},
		},
	}

	// Extract the on section
	onSection := compiler.extractTopLevelYAMLSection(frontmatter, "on")

	// Verify that the YAML contains quoted "on"
	if !strings.Contains(onSection, `"on":`) {
		t.Errorf("Expected on section to contain quoted 'on' key, got:\n%s", onSection)
	}

	// Verify that when parsed, "on" is interpreted as a string key, not a boolean
	var parsed map[string]any
	if err := yaml.Unmarshal([]byte(onSection), &parsed); err != nil {
		t.Fatalf("Failed to parse generated YAML: %v", err)
	}

	// Check that the key "on" exists and is a string
	keys := make([]string, 0, len(parsed))
	for k := range parsed {
		keys = append(keys, k)
	}

	if len(keys) != 1 {
		t.Fatalf("Expected exactly one key in parsed YAML, got %d: %v", len(keys), keys)
	}

	if keys[0] != "on" {
		t.Errorf("Expected key to be 'on', got '%s' (type: %T)", keys[0], keys[0])
	}

	// Ensure it's a string, not a boolean
	if _, ok := parsed["on"]; !ok {
		t.Error("Key 'on' not found in parsed YAML")
	}
}

// TestAllYAMLBooleanKeywordsArePreserved tests that all YAML boolean keywords
// remain quoted when used as keys
func TestAllYAMLBooleanKeywordsArePreserved(t *testing.T) {
	boolKeywords := []string{"on", "off", "yes", "no", "true", "false"}

	for _, keyword := range boolKeywords {
		t.Run(keyword, func(t *testing.T) {
			// Test that the keyword is NOT unquoted
			input := `"` + keyword + `":
  value: test`
			result := UnquoteYAMLKey(input, keyword)

			// Should remain quoted
			if !strings.Contains(result, `"`+keyword+`":`) {
				t.Errorf("YAML boolean keyword '%s' should remain quoted, got:\n%s", keyword, result)
			}

			// Verify that when parsed, it's interpreted as a string key
			var parsed map[string]any
			if err := yaml.Unmarshal([]byte(result), &parsed); err != nil {
				t.Fatalf("Failed to parse YAML for keyword '%s': %v", keyword, err)
			}

			// Check that the key exists and is the string keyword
			if _, ok := parsed[keyword]; !ok {
				t.Errorf("Expected key '%s' in parsed YAML, got keys: %v", keyword, getKeys(parsed))
			}
		})
	}
}

// TestNonBooleanKeywordsAreUnquoted tests that non-boolean keywords
// are still unquoted as expected
func TestNonBooleanKeywordsAreUnquoted(t *testing.T) {
	nonBoolKeywords := []string{"if", "workflow_dispatch", "permissions", "runs-on"}

	for _, keyword := range nonBoolKeywords {
		t.Run(keyword, func(t *testing.T) {
			input := `"` + keyword + `":
  value: test`
			result := UnquoteYAMLKey(input, keyword)

			// Should be unquoted
			if strings.Contains(result, `"`+keyword+`":`) {
				t.Errorf("Non-boolean keyword '%s' should be unquoted, got:\n%s", keyword, result)
			}

			// Should contain unquoted version
			if !strings.Contains(result, keyword+":") {
				t.Errorf("Expected keyword '%s' to be unquoted, got:\n%s", keyword, result)
			}
		})
	}
}

// getKeys returns the keys from a map for error reporting
func getKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
