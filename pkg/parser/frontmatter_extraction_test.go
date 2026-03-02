//go:build !integration

package parser

import (
	"testing"
)

func TestExtractFrontmatterFromContent(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		wantYAML     map[string]any
		wantMarkdown string
		wantErr      bool
	}{
		{
			name: "valid frontmatter and markdown",
			content: `---
title: Test Workflow
on: push
---

# Test Workflow

This is a test workflow.`,
			wantYAML: map[string]any{
				"title": "Test Workflow",
				"on":    "push",
			},
			wantMarkdown: "# Test Workflow\n\nThis is a test workflow.",
		},
		{
			name: "no frontmatter",
			content: `# Test Workflow

This is a test workflow without frontmatter.`,
			wantYAML:     map[string]any{},
			wantMarkdown: "# Test Workflow\n\nThis is a test workflow without frontmatter.",
		},
		{
			name: "empty frontmatter",
			content: `---
---

# Test Workflow

This is a test workflow with empty frontmatter.`,
			wantYAML:     map[string]any{},
			wantMarkdown: "# Test Workflow\n\nThis is a test workflow with empty frontmatter.",
		},
		{
			name:    "unclosed frontmatter",
			content: "---\ntitle: Test\nno closing delimiter",
			wantErr: true,
		},
		{
			name:    "no-break whitespace in values",
			content: "---\ntitle:\u00A0Test\u00A0Workflow\nengine:\u00A0copilot\n---\n\n# Content",
			wantYAML: map[string]any{
				"title":  "Test Workflow",
				"engine": "copilot",
			},
			wantMarkdown: "# Content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExtractFrontmatterFromContent(tt.content)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ExtractFrontmatterFromContent() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ExtractFrontmatterFromContent() error = %v", err)
				return
			}

			// Check frontmatter
			if len(tt.wantYAML) != len(result.Frontmatter) {
				t.Errorf("ExtractFrontmatterFromContent() frontmatter length = %v, want %v", len(result.Frontmatter), len(tt.wantYAML))
			}

			for key, expectedValue := range tt.wantYAML {
				if actualValue, exists := result.Frontmatter[key]; !exists {
					t.Errorf("ExtractFrontmatterFromContent() missing key %v", key)
				} else if actualValue != expectedValue {
					t.Errorf("ExtractFrontmatterFromContent() frontmatter[%v] = %v, want %v", key, actualValue, expectedValue)
				}
			}

			// Check markdown
			if result.Markdown != tt.wantMarkdown {
				t.Errorf("ExtractFrontmatterFromContent() markdown = %v, want %v", result.Markdown, tt.wantMarkdown)
			}
		})
	}
}

func TestExtractMarkdownSection(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		sectionName string
		expected    string
		wantErr     bool
	}{
		{
			name: "basic H1 section",
			content: `# Introduction

This is the introduction.

# Setup

This is the setup section.

# Configuration

This is the configuration.`,
			sectionName: "Setup",
			expected: `# Setup

This is the setup section.`,
		},
		{
			name: "H2 section",
			content: `# Main Title

## Subsection 1

Content for subsection 1.

## Subsection 2

Content for subsection 2.`,
			sectionName: "Subsection 1",
			expected: `## Subsection 1

Content for subsection 1.`,
		},
		{
			name: "nested sections",
			content: `# Main

## Sub1

Content 1

### Sub1.1

Nested content

## Sub2

Content 2`,
			sectionName: "Sub1",
			expected: `## Sub1

Content 1

### Sub1.1

Nested content`,
		},
		{
			name:        "section not found",
			content:     "# Title\n\nContent",
			sectionName: "NonExistent",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExtractMarkdownSection(tt.content, tt.sectionName)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ExtractMarkdownSection() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ExtractMarkdownSection() error = %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("ExtractMarkdownSection() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGenerateDefaultWorkflowName(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		expected string
	}{
		{
			name:     "simple filename",
			filePath: "test-workflow.md",
			expected: "Test Workflow",
		},
		{
			name:     "multiple hyphens",
			filePath: "my-test-workflow-file.md",
			expected: "My Test Workflow File",
		},
		{
			name:     "full path",
			filePath: "/path/to/my-workflow.md",
			expected: "My Workflow",
		},
		{
			name:     "no extension",
			filePath: "workflow",
			expected: "Workflow",
		},
		{
			name:     "single word",
			filePath: "test.md",
			expected: "Test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateDefaultWorkflowName(tt.filePath)
			if result != tt.expected {
				t.Errorf("generateDefaultWorkflowName() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestExtractMarkdownContent(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
		wantErr  bool
	}{
		{
			name: "with frontmatter",
			content: `---
title: Test
---

# Markdown

This is markdown.`,
			expected: "# Markdown\n\nThis is markdown.",
		},
		{
			name:     "no frontmatter",
			content:  "# Just Markdown\n\nContent here.",
			expected: "# Just Markdown\n\nContent here.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExtractMarkdownContent(tt.content)

			if tt.wantErr && err == nil {
				t.Errorf("ExtractMarkdownContent() expected error, got nil")
				return
			}

			if !tt.wantErr && err != nil {
				t.Errorf("ExtractMarkdownContent() error = %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("ExtractMarkdownContent() = %q, want %q", result, tt.expected)
			}
		})
	}
}
