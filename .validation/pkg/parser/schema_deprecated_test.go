//go:build !integration

package parser

import (
	"testing"
)

func TestGetMainWorkflowDeprecatedFields(t *testing.T) {
	deprecatedFields, err := GetMainWorkflowDeprecatedFields()
	if err != nil {
		t.Fatalf("GetMainWorkflowDeprecatedFields() error = %v", err)
	}

	// Check that timeout_minutes is NOT in the list (it has been removed from the schema)
	// This field was fully removed per https://github.com/github/gh-aw/issues/14736
	for _, field := range deprecatedFields {
		if field.Name == "timeout_minutes" {
			t.Errorf("timeout_minutes should not be in the deprecated fields list - it has been completely removed from the schema")
		}
	}

	// The list can be empty or contain other deprecated fields, but timeout_minutes should not be present
	t.Logf("Found %d deprecated fields in schema (timeout_minutes correctly removed)", len(deprecatedFields))
}

func TestFindDeprecatedFieldsInFrontmatter(t *testing.T) {
	deprecatedFields := []DeprecatedField{
		{
			Name:        "timeout_minutes",
			Replacement: "timeout-minutes",
			Description: "Deprecated: Use 'timeout-minutes' instead",
		},
		{
			Name:        "old_field",
			Replacement: "new_field",
			Description: "Deprecated: Use 'new_field' instead",
		},
	}

	tests := []struct {
		name        string
		frontmatter map[string]any
		want        []string // field names that should be found
	}{
		{
			name: "no deprecated fields",
			frontmatter: map[string]any{
				"timeout-minutes": 10,
				"engine":          "copilot",
			},
			want: []string{},
		},
		{
			name: "one deprecated field",
			frontmatter: map[string]any{
				"timeout_minutes": 10,
				"engine":          "copilot",
			},
			want: []string{"timeout_minutes"},
		},
		{
			name: "multiple deprecated fields",
			frontmatter: map[string]any{
				"timeout_minutes": 10,
				"old_field":       "value",
				"engine":          "copilot",
			},
			want: []string{"timeout_minutes", "old_field"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found := FindDeprecatedFieldsInFrontmatter(tt.frontmatter, deprecatedFields)

			if len(found) != len(tt.want) {
				t.Errorf("FindDeprecatedFieldsInFrontmatter() found %d fields, want %d", len(found), len(tt.want))
			}

			// Check that all expected fields were found
			foundMap := make(map[string]bool)
			for _, field := range found {
				foundMap[field.Name] = true
			}

			for _, wantField := range tt.want {
				if !foundMap[wantField] {
					t.Errorf("Expected to find deprecated field '%s', but it was not found", wantField)
				}
			}
		})
	}
}

func TestExtractReplacementFromDescription(t *testing.T) {
	tests := []struct {
		name        string
		description string
		want        string
	}{
		{
			name:        "single quote pattern",
			description: "Deprecated: Use 'timeout-minutes' instead.",
			want:        "timeout-minutes",
		},
		{
			name:        "double quote pattern",
			description: "Deprecated: Use \"timeout-minutes\" instead.",
			want:        "timeout-minutes",
		},
		{
			name:        "backtick pattern",
			description: "Deprecated: Use `timeout-minutes` instead.",
			want:        "timeout-minutes",
		},
		{
			name:        "replaced with pattern",
			description: "This field is replaced with 'new-field'.",
			want:        "new-field",
		},
		{
			name:        "no replacement pattern",
			description: "This field is deprecated.",
			want:        "",
		},
		{
			name:        "complex description with replacement",
			description: "This is a long description explaining why this field is deprecated. Use 'new-field' instead for better compatibility.",
			want:        "new-field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractReplacementFromDescription(tt.description)
			if got != tt.want {
				t.Errorf("extractReplacementFromDescription() = %v, want %v", got, tt.want)
			}
		})
	}
}

// --- Tests for deep walker -------------------------------------------------------

func TestGetMainWorkflowDeprecatedFieldsDeep(t *testing.T) {
	fields, err := GetMainWorkflowDeprecatedFieldsDeep()
	if err != nil {
		t.Fatalf("GetMainWorkflowDeprecatedFieldsDeep() error = %v", err)
	}

	// Build path→field map for easy lookup.
	byPath := make(map[string]DeprecatedField, len(fields))
	for _, f := range fields {
		byPath[f.Path] = f
	}

	// tools.grep must be detected with its x-deprecation-message.
	grep, ok := byPath["tools.grep"]
	if !ok {
		t.Error("expected 'tools.grep' in deep deprecated fields, not found")
	} else {
		if grep.DeprecationMessage == "" {
			t.Error("tools.grep: DeprecationMessage should not be empty")
		}
	}

	// tools.serena must be detected.
	if _, ok := byPath["tools.serena"]; !ok {
		t.Error("expected 'tools.serena' in deep deprecated fields, not found")
	}

	// tools.github.repos must be detected with its x-deprecation-message.
	repos, ok := byPath["tools.github.repos"]
	if !ok {
		t.Error("expected 'tools.github.repos' in deep deprecated fields, not found")
	} else {
		if repos.DeprecationMessage == "" {
			t.Error("tools.github.repos: DeprecationMessage should not be empty")
		}
	}

	t.Logf("Found %d deep deprecated fields in schema", len(fields))
}

// TestAllDeprecatedFieldsHaveXDeprecationMessage is a reference test that ensures every
// deprecated field detected by the deep properties walker carries an x-deprecation-message.
// The walker traverses nested "properties" (plus oneOf/anyOf/allOf sub-schemas for property
// discovery) but does not resolve $ref or inspect $defs directly, and does not catch
// deprecated:true set on non-property subschemas (e.g. a deprecated oneOf variant).
// This test therefore enforces the invariant for the set of fields the walker actually emits
// warnings for — fields in $defs or deprecated oneOf variants must be kept consistent manually.
func TestAllDeprecatedFieldsHaveXDeprecationMessage(t *testing.T) {
	fields, err := GetMainWorkflowDeprecatedFieldsDeep()
	if err != nil {
		t.Fatalf("GetMainWorkflowDeprecatedFieldsDeep() error = %v", err)
	}

	for _, f := range fields {
		if f.DeprecationMessage == "" {
			t.Errorf("schema field %q has deprecated:true but no x-deprecation-message — "+
				"add an x-deprecation-message to the schema entry so users receive an "+
				"actionable migration hint", f.Path)
		}
	}

	t.Logf("Verified %d deprecated fields all have x-deprecation-message", len(fields))
}

func TestCollectDeprecatedDeep(t *testing.T) {
	// Build a minimal schema that exercises nesting and oneOf traversal.
	schema := map[string]any{
		"properties": map[string]any{
			"tools": map[string]any{
				"properties": map[string]any{
					"grep": map[string]any{
						"deprecated":            true,
						"description":           "DEPRECATED: grep is always available.",
						"x-deprecation-message": "Use bash instead.",
					},
					"github": map[string]any{
						"oneOf": []any{
							map[string]any{"type": "boolean"},
							map[string]any{
								"type": "object",
								"properties": map[string]any{
									"repos": map[string]any{
										"deprecated":            true,
										"description":           "Deprecated. Use 'allowed-repos' instead.",
										"x-deprecation-message": "'tools.github.repos' is deprecated. Use 'tools.github.allowed-repos' instead.",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	var results []DeprecatedField
	collectDeprecatedDeep(schema, "", &results)

	byPath := make(map[string]DeprecatedField)
	for _, f := range results {
		byPath[f.Path] = f
	}

	if _, ok := byPath["tools.grep"]; !ok {
		t.Error("expected tools.grep to be found")
	}
	if _, ok := byPath["tools.github.repos"]; !ok {
		t.Error("expected tools.github.repos to be found")
	}
	// Non-deprecated fields must not appear.
	if _, ok := byPath["tools"]; ok {
		t.Error("tools (non-deprecated) should not appear")
	}
	if _, ok := byPath["tools.github"]; ok {
		t.Error("tools.github (non-deprecated) should not appear")
	}
}

func TestFindDeprecatedFieldsInFrontmatterDeep(t *testing.T) {
	fields := []DeprecatedField{
		{Name: "grep", Path: "tools.grep", DeprecationMessage: "Use bash instead."},
		{Name: "repos", Path: "tools.github.repos", DeprecationMessage: "Use allowed-repos."},
		{Name: "old", Path: "old", DeprecationMessage: "Field removed."},
	}

	tests := []struct {
		name        string
		frontmatter map[string]any
		wantPaths   []string
	}{
		{
			name: "no deprecated fields used",
			frontmatter: map[string]any{
				"engine": "copilot",
				"tools":  map[string]any{"bash": true},
			},
			wantPaths: nil,
		},
		{
			name: "tools.grep present",
			frontmatter: map[string]any{
				"tools": map[string]any{"grep": true},
			},
			wantPaths: []string{"tools.grep"},
		},
		{
			name: "nested tools.github.repos present",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"github": map[string]any{"repos": []any{"owner/repo"}},
				},
			},
			wantPaths: []string{"tools.github.repos"},
		},
		{
			name: "top-level deprecated field present",
			frontmatter: map[string]any{
				"old": "value",
			},
			wantPaths: []string{"old"},
		},
		{
			name: "multiple deprecated fields",
			frontmatter: map[string]any{
				"old":   "value",
				"tools": map[string]any{"grep": true},
			},
			wantPaths: []string{"old", "tools.grep"},
		},
		{
			name: "tools key present but not grep",
			frontmatter: map[string]any{
				"tools": map[string]any{"bash": true},
			},
			wantPaths: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found := FindDeprecatedFieldsInFrontmatterDeep(tt.frontmatter, fields)

			foundPaths := make(map[string]bool)
			for _, f := range found {
				foundPaths[f.Path] = true
			}

			if len(found) != len(tt.wantPaths) {
				t.Errorf("FindDeprecatedFieldsInFrontmatterDeep() found %d, want %d (%v vs %v)",
					len(found), len(tt.wantPaths), foundPaths, tt.wantPaths)
			}
			for _, p := range tt.wantPaths {
				if !foundPaths[p] {
					t.Errorf("expected path %q to be found, got %v", p, foundPaths)
				}
			}
		})
	}
}

func TestFieldExistsAtPath(t *testing.T) {
	m := map[string]any{
		"tools": map[string]any{
			"grep":   true,
			"github": map[string]any{"repos": []any{"owner/repo"}},
		},
	}

	tests := []struct {
		segments []string
		want     bool
	}{
		{[]string{"tools"}, true},
		{[]string{"tools", "grep"}, true},
		{[]string{"tools", "github", "repos"}, true},
		{[]string{"tools", "bash"}, false},
		{[]string{"engine"}, false},
		{[]string{}, false},
		// tools is not a scalar — deeper navigation not possible from non-map
		{[]string{"tools", "grep", "nested"}, false},
	}

	for _, tt := range tests {
		got := fieldExistsAtPath(m, tt.segments)
		if got != tt.want {
			t.Errorf("fieldExistsAtPath(%v) = %v, want %v", tt.segments, got, tt.want)
		}
	}
}
