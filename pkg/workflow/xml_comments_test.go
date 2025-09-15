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
