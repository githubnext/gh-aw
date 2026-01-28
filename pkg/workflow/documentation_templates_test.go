package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDocumentationTemplatesExist verifies that all required documentation templates exist
func TestDocumentationTemplatesExist(t *testing.T) {
	templates := []string{
		"INDEX.template.md",
		"README.template.md",
		"QUICKREF.template.md",
		"EXAMPLE.template.md",
		"CONFIG.template.md",
	}

	for _, template := range templates {
		t.Run(template, func(t *testing.T) {
			path := filepath.Join("templates", template)
			_, err := os.Stat(path)
			assert.NoError(t, err, "Template %s should exist", template)
		})
	}
}

// TestTemplateStructure verifies that templates have proper structure and placeholders
func TestTemplateStructure(t *testing.T) {
	tests := []struct {
		name                 string
		file                 string
		requiredPlaceholders []string
		requiredSections     []string
	}{
		{
			name: "INDEX template",
			file: "INDEX.template.md",
			requiredPlaceholders: []string{
				"{{WORKFLOW_NAME}}",
				"{{REPOSITORY_URL}}",
				"{{TIMESTAMP}}",
			},
			requiredSections: []string{
				"Quick Navigation",
				"Documentation Structure",
				"Getting Started",
			},
		},
		{
			name: "README template",
			file: "README.template.md",
			requiredPlaceholders: []string{
				"{{WORKFLOW_NAME}}",
				"{{WORKFLOW_DESCRIPTION}}",
				"{{REPOSITORY_URL}}",
			},
			requiredSections: []string{
				"Overview",
				"Prerequisites",
				"Installation",
				"Quick Start",
				"Configuration",
				"Troubleshooting",
			},
		},
		{
			name: "QUICKREF template",
			file: "QUICKREF.template.md",
			requiredPlaceholders: []string{
				"{{WORKFLOW_NAME}}",
				"{{COMMAND_1}}",
			},
			requiredSections: []string{
				"Common Commands",
				"Command Reference",
				"Configuration Quick Reference",
				"Troubleshooting Cheat Sheet",
			},
		},
		{
			name: "EXAMPLE template",
			file: "EXAMPLE.template.md",
			requiredPlaceholders: []string{
				"{{WORKFLOW_NAME}}",
				"{{EXAMPLE_1_TITLE}}",
			},
			requiredSections: []string{
				"Overview",
				"Common Use Cases",
				"Sample Workflow Runs",
				"Interpreting Results",
			},
		},
		{
			name: "CONFIG template",
			file: "CONFIG.template.md",
			requiredPlaceholders: []string{
				"{{WORKFLOW_NAME}}",
				"{{CONFIGURATION_OVERVIEW}}",
			},
			requiredSections: []string{
				"Configuration Overview",
				"Configuration Options",
				"Environment Variables",
				"Permission Requirements",
				"Deployment Checklist",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join("templates", tt.file)
			content, err := os.ReadFile(path)
			require.NoError(t, err, "Should be able to read template file")

			contentStr := string(content)

			// Verify placeholders exist
			for _, placeholder := range tt.requiredPlaceholders {
				assert.Contains(t, contentStr, placeholder,
					"Template should contain placeholder %s", placeholder)
			}

			// Verify sections exist
			for _, section := range tt.requiredSections {
				assert.Contains(t, contentStr, section,
					"Template should contain section '%s'", section)
			}

			// Verify markdown structure
			assert.True(t, strings.HasPrefix(contentStr, "#"),
				"Template should start with a markdown heading")
		})
	}
}

// TestTemplatePlaceholderSyntax verifies that all placeholders use consistent syntax
func TestTemplatePlaceholderSyntax(t *testing.T) {
	templates := []string{
		"INDEX.template.md",
		"README.template.md",
		"QUICKREF.template.md",
		"EXAMPLE.template.md",
		"CONFIG.template.md",
	}

	for _, template := range templates {
		t.Run(template, func(t *testing.T) {
			path := filepath.Join("templates", template)
			content, err := os.ReadFile(path)
			require.NoError(t, err, "Should be able to read template file")

			contentStr := string(content)

			// Count placeholders - they should follow {{VARIABLE_NAME}} pattern
			// All placeholders should use uppercase with underscores
			placeholders := extractPlaceholders(contentStr)
			assert.NotEmpty(t, placeholders, "Template should contain placeholders")

			for _, placeholder := range placeholders {
				// Verify format: {{UPPER_CASE_WITH_UNDERSCORES}}
				assert.True(t, strings.HasPrefix(placeholder, "{{"),
					"Placeholder should start with {{")
				assert.True(t, strings.HasSuffix(placeholder, "}}"),
					"Placeholder should end with }}")

				// Extract variable name
				varName := strings.TrimPrefix(strings.TrimSuffix(placeholder, "}}"), "{{")
				assert.Equal(t, strings.ToUpper(varName), varName,
					"Placeholder variable names should be uppercase: %s", placeholder)

				// Should not contain spaces
				assert.NotContains(t, varName, " ",
					"Placeholder variable names should not contain spaces: %s", placeholder)
			}
		})
	}
}

// TestTemplateInternalLinks verifies that internal documentation links are consistent
func TestTemplateInternalLinks(t *testing.T) {
	templates := []string{
		"INDEX.template.md",
		"README.template.md",
		"QUICKREF.template.md",
		"EXAMPLE.template.md",
		"CONFIG.template.md",
	}

	expectedLinks := map[string][]string{
		"INDEX.template.md":    {"README.md", "QUICKREF.md", "EXAMPLE.md", "CONFIG.md"},
		"README.template.md":   {"INDEX.md", "QUICKREF.md", "EXAMPLE.md", "CONFIG.md"},
		"QUICKREF.template.md": {"README.md", "EXAMPLE.md", "CONFIG.md", "INDEX.md"},
		"EXAMPLE.template.md":  {"README.md", "QUICKREF.md", "CONFIG.md", "INDEX.md"},
		"CONFIG.template.md":   {"README.md", "QUICKREF.md", "EXAMPLE.md", "INDEX.md"},
	}

	for _, template := range templates {
		t.Run(template, func(t *testing.T) {
			path := filepath.Join("templates", template)
			content, err := os.ReadFile(path)
			require.NoError(t, err, "Should be able to read template file")

			contentStr := string(content)

			// Check that each template links to other documentation files
			links := expectedLinks[template]
			for _, link := range links {
				assert.Contains(t, contentStr, link,
					"Template %s should link to %s", template, link)
			}
		})
	}
}

// TestTemplateCodeBlocksAreClosed verifies that all code blocks are properly closed
func TestTemplateCodeBlocksAreClosed(t *testing.T) {
	templates := []string{
		"INDEX.template.md",
		"README.template.md",
		"QUICKREF.template.md",
		"EXAMPLE.template.md",
		"CONFIG.template.md",
	}

	for _, template := range templates {
		t.Run(template, func(t *testing.T) {
			path := filepath.Join("templates", template)
			content, err := os.ReadFile(path)
			require.NoError(t, err, "Should be able to read template file")

			contentStr := string(content)
			lines := strings.Split(contentStr, "\n")

			// Count triple backticks - should be even (opening and closing)
			backtickCount := 0
			for _, line := range lines {
				if strings.HasPrefix(strings.TrimSpace(line), "```") {
					backtickCount++
				}
			}

			assert.Equal(t, 0, backtickCount%2,
				"Code blocks should be properly closed (even number of ``` markers)")
		})
	}
}

// TestTemplateNoHardcodedValues verifies templates don't contain hardcoded values
func TestTemplateNoHardcodedValues(t *testing.T) {
	templates := []string{
		"INDEX.template.md",
		"README.template.md",
		"QUICKREF.template.md",
		"EXAMPLE.template.md",
		"CONFIG.template.md",
	}

	// These are common patterns that should be placeholders, not hardcoded
	hardcodedPatterns := []string{
		"github.com/your-org",
		"https://github.com/example",
		"your-workflow-name",
		"example-workflow",
		"TODO:",
	}

	for _, template := range templates {
		t.Run(template, func(t *testing.T) {
			path := filepath.Join("templates", template)
			content, err := os.ReadFile(path)
			require.NoError(t, err, "Should be able to read template file")

			contentStr := string(content)

			for _, pattern := range hardcodedPatterns {
				assert.NotContains(t, contentStr, pattern,
					"Template should not contain hardcoded value: %s", pattern)
			}
		})
	}
}

// extractPlaceholders extracts all {{PLACEHOLDER}} patterns from content
func extractPlaceholders(content string) []string {
	var placeholders []string
	start := 0

	for {
		openIdx := strings.Index(content[start:], "{{")
		if openIdx == -1 {
			break
		}
		openIdx += start

		closeIdx := strings.Index(content[openIdx:], "}}")
		if closeIdx == -1 {
			break
		}
		closeIdx += openIdx + 2

		placeholder := content[openIdx:closeIdx]
		placeholders = append(placeholders, placeholder)
		start = closeIdx
	}

	return placeholders
}
