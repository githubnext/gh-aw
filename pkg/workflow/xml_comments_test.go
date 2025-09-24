package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemoveXMLComments(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "No XML comments",
			input:    "This is regular markdown content",
			expected: "This is regular markdown content",
		},
		{
			name:     "Single line XML comment",
			input:    "Before <!-- this is a comment --> after",
			expected: "Before  after",
		},
		{
			name:     "XML comment at start of line",
			input:    "<!-- comment at start --> content",
			expected: " content",
		},
		{
			name:     "XML comment at end of line",
			input:    "content <!-- comment at end -->",
			expected: "content ",
		},
		{
			name:     "Entire line is XML comment",
			input:    "<!-- entire line comment -->",
			expected: "",
		},
		{
			name:     "Multiple XML comments on same line",
			input:    "<!-- first --> middle <!-- second --> end",
			expected: " middle  end",
		},
		{
			name: "Multiline XML comment",
			input: `Before comment
<!-- this is a
multiline comment
that spans multiple lines -->
After comment`,
			expected: `Before comment

After comment`,
		},
		{
			name: "Multiple separate XML comments",
			input: `First line
<!-- comment 1 -->
Middle line
<!-- comment 2 -->
Last line`,
			expected: `First line

Middle line

Last line`,
		},
		{
			name:     "XML comment with special characters",
			input:    "Text <!-- comment with & < > special chars --> more text",
			expected: "Text  more text",
		},
		{
			name:     "Nested-like XML comment (not actually nested)",
			input:    "<!-- outer <!-- inner --> -->",
			expected: " -->",
		},
		{
			name: "XML comment in code block should be preserved",
			input: `Regular text
` + "```" + `
<!-- this comment is in code -->
` + "```" + `
<!-- this comment should be removed -->
More text`,
			expected: `Regular text
` + "```" + `
<!-- this comment is in code -->
` + "```" + `

More text`,
		},
		{
			name: "XML comment in code block with 4 backticks should be preserved",
			input: `Regular text
` + "````" + `python
<!-- this comment is in code -->
` + "````" + `
<!-- this comment should be removed -->
More text`,
			expected: `Regular text
` + "````" + `python
<!-- this comment is in code -->
` + "````" + `

More text`,
		},
		{
			name: "XML comment in code block with tildes should be preserved",
			input: `Regular text
~~~bash
<!-- this comment is in code -->
~~~
<!-- this comment should be removed -->
More text`,
			expected: `Regular text
~~~bash
<!-- this comment is in code -->
~~~

More text`,
		},
		{
			name: "XML comment in code block with 5 tildes should be preserved",
			input: `Regular text
~~~~~
<!-- this comment is in code -->
~~~~~
<!-- this comment should be removed -->
More text`,
			expected: `Regular text
~~~~~
<!-- this comment is in code -->
~~~~~

More text`,
		},
		{
			name:     "Empty XML comment",
			input:    "Before <!---->  after",
			expected: "Before   after",
		},
		{
			name:     "XML comment with only whitespace",
			input:    "Before <!--   --> after",
			expected: "Before  after",
		},
		{
			name: "Mixed code block markers should not interfere",
			input: `Regular text
` + "````python" + `
some code
` + "~~~" + `
this is still in the same python block, not a new tilde block
` + "````" + `
<!-- this comment should be removed because we're outside code blocks -->
More text`,
			expected: `Regular text
` + "````python" + `
some code
` + "~~~" + `
this is still in the same python block, not a new tilde block
` + "````" + `

More text`,
		},
		{
			name: "Different marker types should not close each other",
			input: `Text before
` + "~~~bash" + `
code in tilde block
` + "```" + `
this is still in the tilde block, backticks don't close it
` + "~~~" + `
<!-- this comment should be removed -->
Final text`,
			expected: `Text before
` + "~~~bash" + `
code in tilde block
` + "```" + `
this is still in the tilde block, backticks don't close it
` + "~~~" + `

Final text`,
		},
		{
			name: "Nested same-type markers with proper count matching",
			input: `Content
` + "```" + `
code block
` + "```" + `
<!-- this comment should be removed -->
End`,
			expected: `Content
` + "```" + `
code block
` + "```" + `

End`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeXMLComments(tt.input)
			if result != tt.expected {
				t.Errorf("removeXMLComments() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGeneratePromptRemovesXMLComments(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	data := &WorkflowData{
		MarkdownContent: `# Workflow Title

This is some content.
<!-- This comment should be removed from the prompt -->
More content here.

<!-- Another comment
that spans multiple lines
should also be removed -->

Final content.`,
	}

	var yaml strings.Builder
	compiler.generatePrompt(&yaml, data)

	output := yaml.String()

	// Check that XML comments are not present in the generated output
	if strings.Contains(output, "<!-- This comment should be removed from the prompt -->") {
		t.Error("Expected single-line XML comment to be removed from prompt generation")
	}

	if strings.Contains(output, "<!-- Another comment") {
		t.Error("Expected multi-line XML comment to be removed from prompt generation")
	}

	// Check that regular content is still present
	if !strings.Contains(output, "# Workflow Title") {
		t.Error("Expected regular markdown content to be preserved")
	}

	if !strings.Contains(output, "This is some content.") {
		t.Error("Expected regular content to be preserved")
	}

	if !strings.Contains(output, "Final content.") {
		t.Error("Expected final content to be preserved")
	}
}

func TestValidateMarkdownSizeForGitHubActions(t *testing.T) {
	// Test short content - should pass validation
	shortContent := "# Short content\n\nThis is a brief workflow description."
	err := validateMarkdownSizeForGitHubActions(shortContent)
	if err != nil {
		t.Error("Short content should pass validation")
	}

	// Test content that exceeds the limit - should fail validation
	longContent := strings.Repeat("This is a very long line of content that will be repeated many times to exceed the character limit.\n", 500)
	err = validateMarkdownSizeForGitHubActions(longContent)

	if err == nil {
		t.Error("Long content should fail validation")
	}

	if !strings.Contains(err.Error(), "exceeds GitHub Actions script size limit") {
		t.Error("Error message should mention GitHub Actions script size limit")
	}

	if !strings.Contains(err.Error(), "characters when rendered") {
		t.Error("Error message should mention rendered character count")
	}
}

func TestCompileWorkflowWithCharacterLimitEnforcement(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "character-limit-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	compiler := NewCompiler(false, "", "test")

	// Test that normal-sized content compiles successfully
	normalContent := `---
on:
  issues:
    types: [opened]
permissions:
  issues: write
tools:
  github:
    allowed: [add_issue_comment]
engine: claude
---

# Normal Workflow

This is a normal-sized workflow that should compile successfully.`

	normalFile := filepath.Join(tmpDir, "normal-workflow.md")
	if err := os.WriteFile(normalFile, []byte(normalContent), 0644); err != nil {
		t.Fatal(err)
	}

	err = compiler.CompileWorkflow(normalFile)
	if err != nil {
		t.Errorf("Normal workflow should compile successfully, got error: %v", err)
	}

	// Test that oversized content fails compilation
	longContent := "---\n" +
		"on:\n" +
		"  issues:\n" +
		"    types: [opened]\n" +
		"permissions:\n" +
		"  issues: write\n" +
		"tools:\n" +
		"  github:\n" +
		"    allowed: [add_issue_comment]\n" +
		"engine: claude\n" +
		"---\n\n" +
		"# Very Long Workflow\n\n" +
		strings.Repeat("This is a very long line that will be repeated many times to test the character limit enforcement in GitHub Actions prompt generation.\n", 400)

	longFile := filepath.Join(tmpDir, "long-workflow.md")
	if err := os.WriteFile(longFile, []byte(longContent), 0644); err != nil {
		t.Fatal(err)
	}

	err = compiler.CompileWorkflow(longFile)
	if err == nil {
		t.Error("Long workflow should fail compilation due to size limit")
	}

	if !strings.Contains(err.Error(), "exceeds GitHub Actions script size limit") {
		t.Errorf("Error should mention GitHub Actions script size limit, got: %v", err)
	}
}
