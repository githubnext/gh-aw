package workflow

import (
	"strings"
	"testing"
)

func TestNewIndentWriter(t *testing.T) {
	t.Run("creates writer with custom indent", func(t *testing.T) {
		writer := NewIndentWriter("  ")
		if writer.indent != "  " {
			t.Errorf("Expected indent '  ', got '%s'", writer.indent)
		}
	})

	t.Run("creates writer with tab indent", func(t *testing.T) {
		writer := NewIndentWriter("\t")
		if writer.indent != "\t" {
			t.Errorf("Expected indent '\\t', got '%s'", writer.indent)
		}
	})
}

func TestNewIndentWriterWithSpaces(t *testing.T) {
	tests := []struct {
		name     string
		spaces   int
		expected string
	}{
		{"zero spaces", 0, ""},
		{"two spaces", 2, "  "},
		{"four spaces", 4, "    "},
		{"twelve spaces", 12, "            "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := NewIndentWriterWithSpaces(tt.spaces)
			if writer.indent != tt.expected {
				t.Errorf("Expected indent '%s', got '%s'", tt.expected, writer.indent)
			}
		})
	}
}

func TestIndentWriter_WriteLine(t *testing.T) {
	tests := []struct {
		name     string
		indent   string
		lines    []string
		expected []string
	}{
		{
			name:   "single line with 2 spaces",
			indent: "  ",
			lines:  []string{"console.log('hello');"},
			expected: []string{
				"  console.log('hello');\n",
			},
		},
		{
			name:   "multiple lines with 4 spaces",
			indent: "    ",
			lines:  []string{"const x = 1;", "console.log(x);"},
			expected: []string{
				"    const x = 1;\n",
				"    console.log(x);\n",
			},
		},
		{
			name:   "preserves existing indentation",
			indent: "  ",
			lines:  []string{"if (true) {", "  console.log('indented');", "}"},
			expected: []string{
				"  if (true) {\n",
				"    console.log('indented');\n",
				"  }\n",
			},
		},
		{
			name:     "skips empty lines",
			indent:   "  ",
			lines:    []string{"console.log('first');", "", "console.log('second');"},
			expected: []string{
				"  console.log('first');\n",
				"  console.log('second');\n",
			},
		},
		{
			name:     "skips whitespace-only lines",
			indent:   "  ",
			lines:    []string{"console.log('first');", "   ", "\t", "console.log('second');"},
			expected: []string{
				"  console.log('first');\n",
				"  console.log('second');\n",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := NewIndentWriter(tt.indent)
			for _, line := range tt.lines {
				writer.WriteLine(line)
			}

			result := writer.Lines()
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d lines, got %d", len(tt.expected), len(result))
				return
			}

			for i, line := range result {
				if line != tt.expected[i] {
					t.Errorf("Line %d: expected %q, got %q", i, tt.expected[i], line)
				}
			}
		})
	}
}

func TestIndentWriter_WriteString(t *testing.T) {
	tests := []struct {
		name     string
		indent   string
		content  string
		expected []string
	}{
		{
			name:    "multiline string with newlines",
			indent:  "    ",
			content: "const x = 1;\nconsole.log(x);\nreturn x;",
			expected: []string{
				"    const x = 1;\n",
				"    console.log(x);\n",
				"    return x;\n",
			},
		},
		{
			name:    "string with empty lines",
			indent:  "  ",
			content: "first line\n\nsecond line\n\nthird line",
			expected: []string{
				"  first line\n",
				"  second line\n",
				"  third line\n",
			},
		},
		{
			name:     "empty string",
			indent:   "  ",
			content:  "",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := NewIndentWriter(tt.indent)
			writer.WriteString(tt.content)

			result := writer.Lines()
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d lines, got %d", len(tt.expected), len(result))
				return
			}

			for i, line := range result {
				if line != tt.expected[i] {
					t.Errorf("Line %d: expected %q, got %q", i, tt.expected[i], line)
				}
			}
		})
	}
}

func TestIndentWriter_WriteStringf(t *testing.T) {
	writer := NewIndentWriter("  ")
	writer.WriteStringf("const %s = %d;", "x", 42)
	writer.WriteStringf("console.log('%s: %d');", "x", 42)

	expected := []string{
		"  const x = 42;\n",
		"  console.log('x: 42');\n",
	}

	result := writer.Lines()
	if len(result) != len(expected) {
		t.Errorf("Expected %d lines, got %d", len(expected), len(result))
		return
	}

	for i, line := range result {
		if line != expected[i] {
			t.Errorf("Line %d: expected %q, got %q", i, expected[i], line)
		}
	}
}

func TestIndentWriter_WriteLinef(t *testing.T) {
	writer := NewIndentWriter("    ")
	writer.WriteLinef("const %s = %d;", "counter", 10)
	writer.WriteLinef("if (%s > %d) {", "counter", 5)
	writer.WriteLinef("  console.log('Greater than %d');", 5)
	writer.WriteLine("}")

	expected := []string{
		"    const counter = 10;\n",
		"    if (counter > 5) {\n",
		"      console.log('Greater than 5');\n",
		"    }\n",
	}

	result := writer.Lines()
	if len(result) != len(expected) {
		t.Errorf("Expected %d lines, got %d", len(expected), len(result))
		return
	}

	for i, line := range result {
		if line != expected[i] {
			t.Errorf("Line %d: expected %q, got %q", i, expected[i], line)
		}
	}
}

func TestIndentWriter_ExcludeJSComments(t *testing.T) {
	tests := []struct {
		name     string
		exclude  bool
		content  string
		expected []string
	}{
		{
			name:    "exclude comments enabled",
			exclude: true,
			content: "// This is a comment\nconst x = 1;\n// Another comment\nconsole.log(x);",
			expected: []string{
				"  const x = 1;\n",
				"  console.log(x);\n",
			},
		},
		{
			name:    "exclude comments disabled",
			exclude: false,
			content: "// This is a comment\nconst x = 1;\n// Another comment\nconsole.log(x);",
			expected: []string{
				"  // This is a comment\n",
				"  const x = 1;\n",
				"  // Another comment\n",
				"  console.log(x);\n",
			},
		},
		{
			name:    "indented comments excluded",
			exclude: true,
			content: "if (true) {\n  // Indented comment\n  console.log('hello');\n}",
			expected: []string{
				"  if (true) {\n",
				"    console.log('hello');\n",
				"  }\n",
			},
		},
		{
			name:    "inline comments not excluded",
			exclude: true,
			content: "const x = 1; // This is not excluded\nconsole.log(x); // Neither is this",
			expected: []string{
				"  const x = 1; // This is not excluded\n",
				"  console.log(x); // Neither is this\n",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := NewIndentWriter("  ").ExcludeJSComments(tt.exclude)
			writer.WriteString(tt.content)

			result := writer.Lines()
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d lines, got %d", len(tt.expected), len(result))
				t.Errorf("Expected: %v", tt.expected)
				t.Errorf("Got: %v", result)
				return
			}

			for i, line := range result {
				if line != tt.expected[i] {
					t.Errorf("Line %d: expected %q, got %q", i, tt.expected[i], line)
				}
			}
		})
	}
}

func TestIndentWriter_String(t *testing.T) {
	writer := NewIndentWriter("  ")
	writer.WriteLine("const x = 1;")
	writer.WriteLine("console.log(x);")

	expected := "  const x = 1;\n  console.log(x);\n"
	result := writer.String()

	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestIndentWriter_WriteToBuilder(t *testing.T) {
	writer := NewIndentWriter("    ")
	writer.WriteLine("function test() {")
	writer.WriteLine("  return 'hello';")
	writer.WriteLine("}")

	var builder strings.Builder
	writer.WriteToBuilder(&builder)

	expected := "    function test() {\n      return 'hello';\n    }\n"
	result := builder.String()

	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestIndentWriter_Clear(t *testing.T) {
	writer := NewIndentWriter("  ")
	writer.WriteLine("line 1")
	writer.WriteLine("line 2")

	if writer.LineCount() != 2 {
		t.Errorf("Expected 2 lines before clear, got %d", writer.LineCount())
	}

	writer.Clear()

	if writer.LineCount() != 0 {
		t.Errorf("Expected 0 lines after clear, got %d", writer.LineCount())
	}

	// Test that we can still write after clearing
	writer.WriteLine("new line")
	if writer.LineCount() != 1 {
		t.Errorf("Expected 1 line after writing to cleared writer, got %d", writer.LineCount())
	}
}

func TestIndentWriter_LineCount(t *testing.T) {
	writer := NewIndentWriter("  ")

	if writer.LineCount() != 0 {
		t.Errorf("Expected 0 lines for new writer, got %d", writer.LineCount())
	}

	writer.WriteLine("line 1")
	if writer.LineCount() != 1 {
		t.Errorf("Expected 1 line, got %d", writer.LineCount())
	}

	writer.WriteLine("line 2")
	writer.WriteLine("line 3")
	if writer.LineCount() != 3 {
		t.Errorf("Expected 3 lines, got %d", writer.LineCount())
	}

	// Empty lines should not be counted
	writer.WriteLine("")
	writer.WriteLine("   ")
	if writer.LineCount() != 3 {
		t.Errorf("Expected 3 lines after adding empty lines, got %d", writer.LineCount())
	}
}

func TestIndentWriter_ChainedCalls(t *testing.T) {
	// Test fluent interface
	result := NewIndentWriter("  ").
		ExcludeJSComments(true).
		WriteLine("const x = 1;").
		WriteLinef("console.log('%s');", "x").
		WriteString("if (x > 0) {\n  console.log('positive');\n}").
		String()

	expected := "  const x = 1;\n  console.log('x');\n  if (x > 0) {\n    console.log('positive');\n  }\n"

	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// Benchmark tests
func BenchmarkIndentWriter_WriteLine(b *testing.B) {
	writer := NewIndentWriter("    ")
	line := "console.log('benchmark test');"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		writer.Clear()
		writer.WriteLine(line)
	}
}

func BenchmarkIndentWriter_WriteString(b *testing.B) {
	writer := NewIndentWriter("    ")
	content := `const github = require('@actions/github');
const core = require('@actions/core');

const token = process.env.GITHUB_TOKEN;
const context = github.context;

if (!token) {
  core.setFailed('GITHUB_TOKEN is required');
  return;
}

const octokit = github.getOctokit(token);

// Create a pull request
const result = await octokit.rest.pulls.create({
  owner: context.repo.owner,
  repo: context.repo.repo,
  title: 'Automated PR',
  head: 'feature-branch',
  base: 'main',
  body: 'This is an automated pull request'
});

console.log('PR created:', result.data.html_url);`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		writer.Clear()
		writer.WriteString(content)
	}
}

func BenchmarkIndentWriter_ExcludeComments(b *testing.B) {
	writer := NewIndentWriter("    ").ExcludeJSComments(true)
	content := `// This is a comment
const x = 1;
// Another comment
console.log(x);
// Final comment
return x;`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		writer.Clear()
		writer.WriteString(content)
	}
}