package workflow

import (
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

	// Test the prompt content generation directly
	promptContent := compiler.buildPromptContent(data)

	// Check that XML comments are not present in the generated prompt content
	if strings.Contains(promptContent, "<!-- This comment should be removed from the prompt -->") {
		t.Error("Expected single-line XML comment to be removed from prompt content")
	}

	if strings.Contains(promptContent, "<!-- Another comment") {
		t.Error("Expected multi-line XML comment to be removed from prompt content")
	}

	// Check that regular content is still present
	if !strings.Contains(promptContent, "# Workflow Title") {
		t.Error("Expected regular markdown content to be preserved")
	}

	if !strings.Contains(promptContent, "This is some content.") {
		t.Error("Expected regular content to be preserved")
	}

	if !strings.Contains(promptContent, "Final content.") {
		t.Error("Expected final content to be preserved")
	}

	// Also test the YAML generation uses JavaScript action
	var yaml strings.Builder
	compiler.generatePrompt(&yaml, data)

	output := yaml.String()

	// Check that it uses JavaScript action
	if !strings.Contains(output, "uses: actions/github-script@v8") {
		t.Error("Expected JavaScript action to be used for prompt generation")
	}

	// Check that content is passed as environment variable
	if !strings.Contains(output, "GITHUB_AW_PROMPT_CONTENT:") {
		t.Error("Expected prompt content to be passed as environment variable")
	}
}
