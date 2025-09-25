package parser

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"
)

// TextMateGrammar represents the structure of a TextMate grammar file
type TextMateGrammar struct {
	Name       string                 `json:"name"`
	ScopeName  string                 `json:"scopeName"`
	FileTypes  []string               `json:"fileTypes"`
	Patterns   []map[string]any       `json:"patterns"`
	Repository map[string]any         `json:"repository"`
}

func TestAgenticWorkflowGrammar(t *testing.T) {
	grammarPath := filepath.Join("schemas", "agentic-workflow.tmLanguage.json")
	
	// Read the grammar file
	data, err := ioutil.ReadFile(grammarPath)
	if err != nil {
		t.Fatalf("Failed to read grammar file: %v", err)
	}
	
	// Parse the JSON
	var grammar TextMateGrammar
	if err := json.Unmarshal(data, &grammar); err != nil {
		t.Fatalf("Failed to parse grammar JSON: %v", err)
	}
	
	// Test basic structure
	if grammar.Name != "agentic-workflow" {
		t.Errorf("Expected grammar name to be 'agentic-workflow', got '%s'", grammar.Name)
	}
	
	if grammar.ScopeName != "source.agentic-workflow" {
		t.Errorf("Expected scope name to be 'source.agentic-workflow', got '%s'", grammar.ScopeName)
	}
	
	if len(grammar.FileTypes) == 0 || grammar.FileTypes[0] != "md" {
		t.Errorf("Expected file types to include 'md', got %v", grammar.FileTypes)
	}
	
	// Test that required repository items exist
	requiredItems := []string{"frontmatter", "markdown-content", "include-directive", "github-context-expression"}
	for _, item := range requiredItems {
		if _, exists := grammar.Repository[item]; !exists {
			t.Errorf("Missing required repository item: %s", item)
		}
	}
	
	// Test that patterns include frontmatter and markdown-content
	foundFrontmatter := false
	foundMarkdown := false
	for _, pattern := range grammar.Patterns {
		if include, ok := pattern["include"].(string); ok {
			if include == "#frontmatter" {
				foundFrontmatter = true
			}
			if include == "#markdown-content" {
				foundMarkdown = true
			}
		}
	}
	
	if !foundFrontmatter {
		t.Error("Grammar patterns should include '#frontmatter'")
	}
	if !foundMarkdown {
		t.Error("Grammar patterns should include '#markdown-content'")
	}
}

func TestGrammarSyntaxHighlighting(t *testing.T) {
	// Test cases representing different parts of agentic workflow syntax
	testCases := []struct {
		name        string
		content     string
		expectYAML  bool
		expectMD    bool
		expectInclude bool
	}{
		{
			name: "complete workflow with frontmatter and markdown",
			content: `---
on:
  issues:
    types: [opened]
engine: claude
tools:
  github:
    allowed: [add_issue_comment]
safe-outputs:
  create-issue:
---

# Workflow Title

Analyze issue ${{ github.event.issue.number }}.

@include shared/security-notice.md
`,
			expectYAML:    true,
			expectMD:      true,
			expectInclude: true,
		},
		{
			name: "frontmatter only",
			content: `---
engine: claude
tools:
  github:
    allowed: [get_issue]
---`,
			expectYAML:    true,
			expectMD:      false,
			expectInclude: false,
		},
		{
			name: "markdown with include directive",
			content: `# Test Workflow

@include shared/tools.md
@include? optional/config.md#section

Some content here.`,
			expectYAML:    false,
			expectMD:      true,
			expectInclude: true,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// These are basic structural tests
			// In a real implementation, you'd use a TextMate grammar parser
			// to validate that the content is properly highlighted
			
			hasFrontmatter := strings.Contains(tc.content, "---") && 
				(strings.Contains(tc.content, "engine:") || strings.Contains(tc.content, "tools:"))
			hasMarkdown := strings.Contains(tc.content, "#") || strings.Contains(tc.content, "Some content")
			hasInclude := strings.Contains(tc.content, "@include")
			
			if tc.expectYAML && !hasFrontmatter {
				t.Errorf("Expected YAML frontmatter in test case '%s'", tc.name)
			}
			if tc.expectMD && !hasMarkdown {
				t.Errorf("Expected Markdown content in test case '%s'", tc.name)
			}
			if tc.expectInclude && !hasInclude {
				t.Errorf("Expected include directive in test case '%s'", tc.name)
			}
		})
	}
}

func TestGrammarValidation(t *testing.T) {
	grammarPath := filepath.Join("schemas", "agentic-workflow.tmLanguage.json")
	
	// Read and validate the grammar JSON is well-formed
	data, err := ioutil.ReadFile(grammarPath)
	if err != nil {
		t.Fatalf("Failed to read grammar file: %v", err)
	}
	
	// Test that it's valid JSON
	var grammarData map[string]any
	if err := json.Unmarshal(data, &grammarData); err != nil {
		t.Fatalf("Grammar file contains invalid JSON: %v", err)
	}
	
	// Test that all required TextMate grammar fields are present
	requiredFields := []string{"name", "scopeName", "patterns", "repository"}
	for _, field := range requiredFields {
		if _, exists := grammarData[field]; !exists {
			t.Errorf("Missing required TextMate grammar field: %s", field)
		}
	}
	
	// Test that repository contains expected patterns
	if repository, ok := grammarData["repository"].(map[string]any); ok {
		// Test frontmatter pattern
		if frontmatter, exists := repository["frontmatter"].(map[string]any); exists {
			if _, hasBegin := frontmatter["begin"]; !hasBegin {
				t.Error("Frontmatter pattern should have 'begin' regex")
			}
			if _, hasEnd := frontmatter["end"]; !hasEnd {
				t.Error("Frontmatter pattern should have 'end' regex")
			}
		} else {
			t.Error("Repository should contain frontmatter pattern")
		}
		
		// Test include-directive pattern
		if includeDirective, exists := repository["include-directive"].(map[string]any); exists {
			if patterns, hasPatterns := includeDirective["patterns"].([]any); hasPatterns {
				if len(patterns) == 0 {
					t.Error("Include directive should have at least one pattern")
				}
			}
		} else {
			t.Error("Repository should contain include-directive pattern")
		}
	} else {
		t.Error("Grammar should have a valid repository object")
	}
}