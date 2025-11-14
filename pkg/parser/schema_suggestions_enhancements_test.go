package parser

import (
	"strings"
	"testing"
)

func TestGenerateSuggestionsAsList(t *testing.T) {
	tests := []struct {
		name         string
		errorMessage string
		jsonPath     string
		wantContains []string
		wantEmpty    bool
	}{
		{
			name:         "unknown property with typo",
			errorMessage: "additional properties 'engnie' not allowed",
			jsonPath:     "",
			wantContains: []string{"Did you mean 'engine'", "Valid frontmatter fields"},
		},
		{
			name:         "missing property 'on'",
			errorMessage: "missing property 'on'",
			jsonPath:     "",
			wantContains: []string{"Add the required field 'on'", "Example triggers:", "on: issues"},
		},
		{
			name:         "invalid engine value",
			errorMessage: "at '/engine': value must be one of 'claude', 'codex', 'copilot', 'custom'",
			jsonPath:     "/engine",
			wantContains: []string{"supported engines", "engine: copilot"},
		},
		{
			name:         "non-matching error",
			errorMessage: "some random error",
			jsonPath:     "",
			wantEmpty:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use the main workflow schema for testing
			suggestions := generateSuggestionsAsList(mainWorkflowSchema, tt.errorMessage, tt.jsonPath)

			if tt.wantEmpty {
				if len(suggestions) > 0 {
					t.Errorf("Expected no suggestions, got: %v", suggestions)
				}
				return
			}

			if len(suggestions) == 0 {
				t.Errorf("Expected suggestions, got none")
				return
			}

			// Check that all expected strings are present in at least one suggestion
			for _, want := range tt.wantContains {
				found := false
				for _, suggestion := range suggestions {
					if strings.Contains(suggestion, want) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected to find '%s' in suggestions, but got: %v", want, suggestions)
				}
			}
		})
	}
}

func TestGenerateDocumentationLink(t *testing.T) {
	tests := []struct {
		name         string
		errorMessage string
		jsonPath     string
		want         string
	}{
		{
			name:         "unknown property",
			errorMessage: "additional properties not allowed",
			jsonPath:     "",
			want:         "https://githubnext.github.io/gh-aw/reference/frontmatter/",
		},
		{
			name:         "engine error",
			errorMessage: "invalid engine value",
			jsonPath:     "/engine",
			want:         "https://githubnext.github.io/gh-aw/reference/frontmatter/#engine",
		},
		{
			name:         "tools error",
			errorMessage: "invalid tools configuration",
			jsonPath:     "/tools",
			want:         "https://githubnext.github.io/gh-aw/reference/frontmatter/#tools",
		},
		{
			name:         "permissions error",
			errorMessage: "invalid permissions",
			jsonPath:     "/permissions",
			want:         "https://githubnext.github.io/gh-aw/reference/frontmatter/#permissions",
		},
		{
			name:         "on trigger error",
			errorMessage: "missing property 'on'",
			jsonPath:     "",
			want:         "https://githubnext.github.io/gh-aw/reference/frontmatter/#on",
		},
		{
			name:         "timeout error",
			errorMessage: "invalid timeout-minutes",
			jsonPath:     "/timeout-minutes",
			want:         "https://githubnext.github.io/gh-aw/reference/frontmatter/#timeout-minutes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateDocumentationLink(mainWorkflowSchema, tt.errorMessage, tt.jsonPath)
			if got != tt.want {
				t.Errorf("generateDocumentationLink() = %v, want %v", got, tt.want)
			}
		})
	}
}
