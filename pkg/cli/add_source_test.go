//go:build !integration

package cli

import (
	"strings"
	"testing"
)

// TestAddSourceToWorkflow tests the addSourceToWorkflow function
func TestAddSourceToWorkflow(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		source      string
		expectError bool
		checkSource bool
	}{
		{
			name: "add_source_to_workflow_with_frontmatter",
			content: `---
on: push
permissions:
  contents: read
engine: claude
---

# Test Workflow

This is a test workflow.`,
			source:      "githubnext/agentics/workflows/ci-doctor.md@v1.0.0",
			expectError: false,
			checkSource: true,
		},
		{
			name: "add_source_to_workflow_without_frontmatter",
			content: `# Test Workflow

This is a test workflow without frontmatter.`,
			source:      "githubnext/agentics/workflows/test.md@main",
			expectError: false,
			checkSource: true,
		},
		{
			name: "add_source_to_existing_workflow_with_fields",
			content: `---
description: "Test workflow description"
on: push
permissions:
  contents: read
engine: claude
tools:
  github:
    allowed: [list_commits]
---

# Test Workflow

This is a test workflow.`,
			source:      "githubnext/agentics/workflows/complex.md@v1.0.0",
			expectError: false,
			checkSource: true,
		},
		{
			name: "verify_on_keyword_not_quoted",
			content: `---
on:
  push:
    branches: [main]
  pull_request:
    types: [opened]
permissions:
  contents: read
engine: claude
---

# Test Workflow

This workflow has complex 'on' triggers.`,
			source:      "githubnext/agentics/workflows/test.md@v1.0.0",
			expectError: false,
			checkSource: true,
		},
		{
			name: "preserve_formatting_with_comments_and_blank_lines",
			content: `---
on:
    workflow_dispatch:

    schedule:
        # Run daily at 2am UTC, all days except Saturday and Sunday
        - cron: "0 2 * * 1-5"

    stop-after: +48h # workflow will no longer trigger after 48 hours

timeout_minutes: 30

permissions: read-all

network: defaults

engine: claude

tools:
    # Web tools for testing
    web-search: null
    
    # Memory cache
    cache-memory: true
---

# Well Formatted Workflow

This workflow has proper formatting with comments and blank lines.`,
			source:      "githubnext/agentics/workflows/formatted.md@v1.0.0",
			expectError: false,
			checkSource: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := addSourceToWorkflow(tt.content, tt.source)

			if tt.expectError && err == nil {
				t.Errorf("addSourceToWorkflow() expected error, got nil")
				return
			}

			if !tt.expectError && err != nil {
				t.Errorf("addSourceToWorkflow() error = %v", err)
				return
			}

			if !tt.expectError && tt.checkSource {
				// Verify that the source field is present in the result
				if !strings.Contains(result, "source:") {
					t.Errorf("addSourceToWorkflow() result does not contain 'source:' field")
				}
				if !strings.Contains(result, tt.source) {
					t.Errorf("addSourceToWorkflow() result does not contain source value '%s'", tt.source)
				}

				// Verify that frontmatter delimiters are present
				if !strings.Contains(result, "---") {
					t.Errorf("addSourceToWorkflow() result does not contain frontmatter delimiters")
				}

				// Verify that markdown content is preserved
				if strings.Contains(tt.content, "# Test Workflow") && !strings.Contains(result, "# Test Workflow") {
					t.Errorf("addSourceToWorkflow() result does not preserve markdown content")
				}
				if strings.Contains(tt.content, "# Well Formatted Workflow") && !strings.Contains(result, "# Well Formatted Workflow") {
					t.Errorf("addSourceToWorkflow() result does not preserve markdown content")
				}

				// Verify that "on" keyword is not quoted
				if strings.Contains(result, `"on":`) {
					t.Errorf("addSourceToWorkflow() result contains quoted 'on' keyword, should be unquoted. Result:\n%s", result)
				}

				// For the formatting preservation test, verify that comments and blank lines are preserved
				if tt.name == "preserve_formatting_with_comments_and_blank_lines" {
					if !strings.Contains(result, "# Run daily at 2am UTC, all days except Saturday and Sunday") {
						t.Errorf("addSourceToWorkflow() result does not preserve comments")
					}
					if !strings.Contains(result, "stop-after: +48h # workflow will no longer trigger") {
						t.Errorf("addSourceToWorkflow() result does not preserve inline comments")
					}
					if !strings.Contains(result, "    # Web tools for testing") {
						t.Errorf("addSourceToWorkflow() result does not preserve indented comments")
					}
					// Check that there are still blank lines by checking for consecutive newlines
					if !strings.Contains(result, "\n\n") {
						t.Errorf("addSourceToWorkflow() result does not preserve blank lines")
					}
				}
			}
		})
	}
}

// TestAddEngineToWorkflow tests the addEngineToWorkflow function
func TestAddEngineToWorkflow(t *testing.T) {
	tests := []struct {
		name            string
		content         string
		engine          string
		expectError     bool
		checkEngine     bool
		expectBlankLine bool // expect blank line after engine declaration
	}{
		{
			name: "add_new_engine_field_has_trailing_blank_line",
			content: `---
on: push
permissions:
  contents: read
---

# Test Workflow`,
			engine:          "claude",
			expectError:     false,
			checkEngine:     true,
			expectBlankLine: true,
		},
		{
			name: "update_existing_engine_field_no_extra_blank_line",
			content: `---
on: push
engine: codex
---

# Test Workflow`,
			engine:          "claude",
			expectError:     false,
			checkEngine:     true,
			expectBlankLine: false, // updating existing field doesn't add a blank line
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := addEngineToWorkflow(tt.content, tt.engine)

			if tt.expectError && err == nil {
				t.Errorf("addEngineToWorkflow() expected error, got nil")
				return
			}

			if !tt.expectError && err != nil {
				t.Errorf("addEngineToWorkflow() error = %v", err)
				return
			}

			if !tt.expectError && tt.checkEngine {
				// Verify that the engine field is present with the correct value
				if !strings.Contains(result, "engine: "+tt.engine) {
					t.Errorf("addEngineToWorkflow() result does not contain 'engine: %s'. Result:\n%s", tt.engine, result)
				}

				// Verify that frontmatter delimiters are present
				if !strings.Contains(result, "---") {
					t.Errorf("addEngineToWorkflow() result does not contain frontmatter delimiters")
				}

				// Check for blank line after engine declaration by verifying the frontmatter
				// contains the engine line followed immediately by a blank line.
				// Extract the frontmatter section (between --- markers)
				parts := strings.SplitN(result, "---", 3)
				if len(parts) < 3 {
					t.Errorf("addEngineToWorkflow() result has unexpected structure. Result:\n%s", result)
					return
				}
				frontmatter := parts[1]
				if tt.expectBlankLine {
					if !strings.Contains(frontmatter, "engine: "+tt.engine+"\n\n") {
						t.Errorf("addEngineToWorkflow() frontmatter does not have blank line after engine declaration. Frontmatter:\n%s", frontmatter)
					}
				} else {
					if strings.Contains(frontmatter, "engine: "+tt.engine+"\n\n") {
						t.Errorf("addEngineToWorkflow() frontmatter unexpectedly has blank line after engine declaration. Frontmatter:\n%s", frontmatter)
					}
				}
			}
		})
	}

	// Verify that engine declaration blank line separates it from a subsequently added source field.
	t.Run("engine_blank_line_separates_from_source", func(t *testing.T) {
		original := `---
on: push
permissions:
  contents: read
---

# Test Workflow`

		// Add engine first
		withEngine, err := addEngineToWorkflow(original, "claude")
		if err != nil {
			t.Fatalf("addEngineToWorkflow() error = %v", err)
		}

		// Then add source
		withSource, err := addSourceToWorkflow(withEngine, "owner/repo/workflows/test.md@abc123")
		if err != nil {
			t.Fatalf("addSourceToWorkflow() error = %v", err)
		}

		// Extract frontmatter
		parts := strings.SplitN(withSource, "---", 3)
		if len(parts) < 3 {
			t.Fatalf("unexpected file structure. Result:\n%s", withSource)
		}
		frontmatter := parts[1]

		// engine and source must NOT be adjacent (no blank line between would mean conflict risk)
		if strings.Contains(frontmatter, "engine: claude\nsource:") {
			t.Errorf("engine and source fields are adjacent with no blank line separator. Frontmatter:\n%s", frontmatter)
		}
		// engine line must be followed by a blank line
		if !strings.Contains(frontmatter, "engine: claude\n\n") {
			t.Errorf("engine field is not followed by a blank line. Frontmatter:\n%s", frontmatter)
		}
	})
}
