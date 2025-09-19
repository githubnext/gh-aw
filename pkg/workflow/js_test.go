package workflow

import (
	"strings"
	"testing"
)

func TestFormatJavaScriptForYAML(t *testing.T) {
	tests := []struct {
		name     string
		script   string
		expected []string
	}{
		{
			name:     "empty string",
			script:   "",
			expected: []string{},
		},
		{
			name:   "single line without empty lines",
			script: "console.log('hello');",
			expected: []string{
				"            console.log('hello');\n",
			},
		},
		{
			name:   "multiple lines without empty lines",
			script: "const x = 1;\nconsole.log(x);",
			expected: []string{
				"            const x = 1;\n",
				"            console.log(x);\n",
			},
		},
		{
			name:   "script with empty lines should skip them",
			script: "const x = 1;\n\nconsole.log(x);\n\nreturn x;",
			expected: []string{
				"            const x = 1;\n",
				"            console.log(x);\n",
				"            return x;\n",
			},
		},
		{
			name:   "script with only whitespace lines should skip them",
			script: "const x = 1;\n   \n\t\nconsole.log(x);",
			expected: []string{
				"            const x = 1;\n",
				"            console.log(x);\n",
			},
		},
		{
			name:   "script with leading and trailing empty lines",
			script: "\n\nconst x = 1;\nconsole.log(x);\n\n",
			expected: []string{
				"            const x = 1;\n",
				"            console.log(x);\n",
			},
		},
		{
			name:   "script with indented code",
			script: "if (true) {\n  console.log('indented');\n}",
			expected: []string{
				"            if (true) {\n",
				"              console.log('indented');\n",
				"            }\n",
			},
		},
		{
			name:   "complex script with mixed content",
			script: "// Comment\nconst github = require('@actions/github');\n\nconst token = process.env.GITHUB_TOKEN;\n\n// Another comment\nif (token) {\n  console.log('Token found');\n}\n",
			expected: []string{
				"            // Comment\n",
				"            const github = require('@actions/github');\n",
				"            const token = process.env.GITHUB_TOKEN;\n",
				"            // Another comment\n",
				"            if (token) {\n",
				"              console.log('Token found');\n",
				"            }\n",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatJavaScriptForYAML(tt.script)

			if len(result) != len(tt.expected) {
				t.Errorf("FormatJavaScriptForYAML() returned %d lines, expected %d", len(result), len(tt.expected))
				t.Errorf("Got: %v", result)
				t.Errorf("Expected: %v", tt.expected)
				return
			}

			for i, line := range result {
				if line != tt.expected[i] {
					t.Errorf("FormatJavaScriptForYAML() line %d = %q, expected %q", i, line, tt.expected[i])
				}
			}
		})
	}
}

func TestWriteJavaScriptToYAML(t *testing.T) {
	tests := []struct {
		name     string
		script   string
		expected string
	}{
		{
			name:     "empty string",
			script:   "",
			expected: "",
		},
		{
			name:     "single line without empty lines",
			script:   "console.log('hello');",
			expected: "            console.log('hello');\n",
		},
		{
			name:     "multiple lines without empty lines",
			script:   "const x = 1;\nconsole.log(x);",
			expected: "            const x = 1;\n            console.log(x);\n",
		},
		{
			name:     "script with empty lines should skip them",
			script:   "const x = 1;\n\nconsole.log(x);\n\nreturn x;",
			expected: "            const x = 1;\n            console.log(x);\n            return x;\n",
		},
		{
			name:     "script with only whitespace lines should skip them",
			script:   "const x = 1;\n   \n\t\nconsole.log(x);",
			expected: "            const x = 1;\n            console.log(x);\n",
		},
		{
			name:     "script with leading and trailing empty lines",
			script:   "\n\nconst x = 1;\nconsole.log(x);\n\n",
			expected: "            const x = 1;\n            console.log(x);\n",
		},
		{
			name:     "script with indented code",
			script:   "if (true) {\n  console.log('indented');\n}",
			expected: "            if (true) {\n              console.log('indented');\n            }\n",
		},
		{
			name:     "complex script with mixed content",
			script:   "// Comment\nconst github = require('@actions/github');\n\nconst token = process.env.GITHUB_TOKEN;\n\n// Another comment\nif (token) {\n  console.log('Token found');\n}\n",
			expected: "            // Comment\n            const github = require('@actions/github');\n            const token = process.env.GITHUB_TOKEN;\n            // Another comment\n            if (token) {\n              console.log('Token found');\n            }\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var yaml strings.Builder
			WriteJavaScriptToYAML(&yaml, tt.script)
			result := yaml.String()

			if result != tt.expected {
				t.Errorf("WriteJavaScriptToYAML() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestEmbeddedScriptsNotEmpty(t *testing.T) {
	tests := []struct {
		name   string
		script string
	}{
		{"createPullRequestScript", createPullRequestScript},
		{"createIssueScript", createIssueScript},
		{"createCommentScript", createCommentScript},
		{"collectJSONLOutputScript", collectJSONLOutputScript},
		{"addLabelsScript", addLabelsScript},
		{"updateIssueScript", updateIssueScript},
		{"setupAgentOutputScript", setupAgentOutputScript},
		{"addReactionScript", addReactionScript},
		{"addReactionAndEditCommentScript", addReactionAndEditCommentScript},
		{"missingToolScript", missingToolScript},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if strings.TrimSpace(tt.script) == "" {
				t.Errorf("Embedded script %s is empty", tt.name)
			}
		})
	}
}

func TestFormatJavaScriptForYAMLProducesValidIndentation(t *testing.T) {
	script := "const x = 1;\nif (x > 0) {\n  console.log('positive');\n}"
	result := FormatJavaScriptForYAML(script)

	// Check that all lines start with proper indentation (12 spaces)
	for i, line := range result {
		if !strings.HasPrefix(line, "            ") {
			t.Errorf("Line %d does not start with proper indentation: %q", i, line)
		}
		if !strings.HasSuffix(line, "\n") {
			t.Errorf("Line %d does not end with newline: %q", i, line)
		}
	}
}

func TestWriteJavaScriptToYAMLProducesValidIndentation(t *testing.T) {
	script := "const x = 1;\nif (x > 0) {\n  console.log('positive');\n}"
	var yaml strings.Builder
	WriteJavaScriptToYAML(&yaml, script)
	result := yaml.String()

	lines := strings.Split(result, "\n")
	// Remove last empty line from split
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	// Check that all lines start with proper indentation (12 spaces)
	for i, line := range lines {
		if !strings.HasPrefix(line, "            ") {
			t.Errorf("Line %d does not start with proper indentation: %q", i, line)
		}
	}
}

func TestJavaScriptFormattingConsistency(t *testing.T) {
	// Test that both functions produce equivalent output
	testScript := "const x = 1;\n\nconsole.log(x);\n\nreturn x;"

	// Test FormatJavaScriptForYAML
	formattedLines := FormatJavaScriptForYAML(testScript)
	formattedResult := strings.Join(formattedLines, "")

	// Test WriteJavaScriptToYAML
	var yaml strings.Builder
	WriteJavaScriptToYAML(&yaml, testScript)
	writeResult := yaml.String()

	if formattedResult != writeResult {
		t.Errorf("FormatJavaScriptForYAML and WriteJavaScriptToYAML produce different results")
		t.Errorf("FormatJavaScriptForYAML: %q", formattedResult)
		t.Errorf("WriteJavaScriptToYAML: %q", writeResult)
	}
}

func BenchmarkFormatJavaScriptForYAML(b *testing.B) {
	script := `const github = require('@actions/github');
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
		FormatJavaScriptForYAML(script)
	}
}

func TestFormatJavaScriptForYAMLWithoutComments(t *testing.T) {
	tests := []struct {
		name     string
		script   string
		expected []string
	}{
		{
			name:     "empty string",
			script:   "",
			expected: []string{},
		},
		{
			name:   "single line without comments",
			script: "console.log('hello');",
			expected: []string{
				"            console.log('hello');\n",
			},
		},
		{
			name:   "single line with comment",
			script: "console.log('hello'); // This is a comment",
			expected: []string{
				"            console.log('hello');\n",
			},
		},
		{
			name:     "line with only comment",
			script:   "// This is only a comment",
			expected: []string{},
		},
		{
			name:   "multiple lines with mixed comments",
			script: "const x = 1; // variable declaration\nconsole.log(x);\n// another comment\nreturn x;",
			expected: []string{
				"            const x = 1;\n",
				"            console.log(x);\n",
				"            return x;\n",
			},
		},
		{
			name:   "comment with URL should be preserved in string",
			script: `const url = "https://example.com"; // URL comment`,
			expected: []string{
				"            const url = \"https://example.com\";\n",
			},
		},
		{
			name:   "complex script with mixed content",
			script: "// Header comment\nconst github = require('@actions/github'); // import\n\nconst token = process.env.GITHUB_TOKEN;\n\n// Check token\nif (token) {\n  console.log('Token found'); // log message\n}\n",
			expected: []string{
				"            const github = require('@actions/github');\n",
				"            const token = process.env.GITHUB_TOKEN;\n",
				"            if (token) {\n",
				"              console.log('Token found');\n",
				"            }\n",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatJavaScriptForYAMLWithoutComments(tt.script)

			if len(result) != len(tt.expected) {
				t.Errorf("FormatJavaScriptForYAMLWithoutComments() returned %d lines, expected %d", len(result), len(tt.expected))
				t.Errorf("Got: %v", result)
				t.Errorf("Expected: %v", tt.expected)
				return
			}

			for i, line := range result {
				if line != tt.expected[i] {
					t.Errorf("FormatJavaScriptForYAMLWithoutComments() line %d = %q, expected %q", i, line, tt.expected[i])
				}
			}
		})
	}
}

func TestWriteJavaScriptToYAMLWithoutComments(t *testing.T) {
	tests := []struct {
		name     string
		script   string
		expected string
	}{
		{
			name:     "empty string",
			script:   "",
			expected: "",
		},
		{
			name:     "single line without comments",
			script:   "console.log('hello');",
			expected: "            console.log('hello');\n",
		},
		{
			name:     "single line with comment",
			script:   "console.log('hello'); // This is a comment",
			expected: "            console.log('hello');\n",
		},
		{
			name:     "line with only comment",
			script:   "// This is only a comment",
			expected: "",
		},
		{
			name:     "line with indented comment",
			script:   "  // Indented comment",
			expected: "",
		},
		{
			name:     "multiple lines with mixed comments",
			script:   "const x = 1; // variable declaration\nconsole.log(x);\n// another comment\nreturn x;",
			expected: "            const x = 1;\n            console.log(x);\n            return x;\n",
		},
		{
			name:     "comment with URL should be preserved in string",
			script:   `const url = "https://example.com"; // URL comment`,
			expected: "            const url = \"https://example.com\";\n",
		},
		{
			name:     "double slash in string should be preserved",
			script:   `console.log("http://example.com//path");`,
			expected: "            console.log(\"http://example.com//path\");\n",
		},
		{
			name:     "comment after string with double slash",
			script:   `const url = "https://example.com"; // This is a comment`,
			expected: "            const url = \"https://example.com\";\n",
		},
		{
			name:     "escaped quotes in string",
			script:   `console.log("She said \"Hello//World\""); // comment`,
			expected: "            console.log(\"She said \\\"Hello//World\\\"\");\n",
		},
		{
			name:     "single quotes with double slash",
			script:   `const path = 'path//to//file'; // comment`,
			expected: "            const path = 'path//to//file';\n",
		},
		{
			name:     "template literal with double slash",
			script:   "const url = `https://api.example.com//v1`; // comment",
			expected: "            const url = `https://api.example.com//v1`;\n",
		},
		{
			name:     "complex script with mixed content",
			script:   "// Header comment\nconst github = require('@actions/github'); // import\n\nconst token = process.env.GITHUB_TOKEN;\n\n// Check token\nif (token) {\n  console.log('Token found'); // log message\n}\n",
			expected: "            const github = require('@actions/github');\n            const token = process.env.GITHUB_TOKEN;\n            if (token) {\n              console.log('Token found');\n            }\n",
		},
		{
			name:     "comment at start of line with spaces",
			script:   "    // Comment with leading spaces\nconst x = 1;",
			expected: "            const x = 1;\n",
		},
		{
			name:     "code with trailing spaces and comment",
			script:   "const x = 1;   // comment with spaces",
			expected: "            const x = 1;\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var yaml strings.Builder
			WriteJavaScriptToYAMLWithoutComments(&yaml, tt.script)
			result := yaml.String()

			if result != tt.expected {
				t.Errorf("WriteJavaScriptToYAMLWithoutComments() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestRemoveSingleLineComment(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no comment",
			input:    "const x = 1;",
			expected: "const x = 1;",
		},
		{
			name:     "comment at end",
			input:    "const x = 1; // This is a comment",
			expected: "const x = 1;",
		},
		{
			name:     "only comment",
			input:    "// This is only a comment",
			expected: "",
		},
		{
			name:     "comment with leading spaces",
			input:    "    // Indented comment",
			expected: "",
		},
		{
			name:     "double slash in string",
			input:    `const url = "https://example.com";`,
			expected: `const url = "https://example.com";`,
		},
		{
			name:     "double slash in string with comment",
			input:    `const url = "https://example.com"; // comment`,
			expected: `const url = "https://example.com";`,
		},
		{
			name:     "escaped quotes",
			input:    `console.log("She said \"Hello//World\"");`,
			expected: `console.log("She said \"Hello//World\"");`,
		},
		{
			name:     "single quotes",
			input:    `const path = 'path//to//file';`,
			expected: `const path = 'path//to//file';`,
		},
		{
			name:     "template literal",
			input:    "const url = `https://api.example.com//v1`;",
			expected: "const url = `https://api.example.com//v1`;",
		},
		{
			name:     "comment with trailing spaces",
			input:    "const x = 1;   // comment",
			expected: "const x = 1;",
		},
		{
			name:     "empty line",
			input:    "",
			expected: "",
		},
		{
			name:     "only spaces",
			input:    "    ",
			expected: "    ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeSingleLineComment(tt.input)
			if result != tt.expected {
				t.Errorf("removeSingleLineComment(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func BenchmarkWriteJavaScriptToYAML(b *testing.B) {
	script := `const github = require('@actions/github');
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
		var yaml strings.Builder
		WriteJavaScriptToYAML(&yaml, script)
	}
}

func BenchmarkWriteJavaScriptToYAMLWithoutComments(b *testing.B) {
	script := `const github = require('@actions/github');
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
		var yaml strings.Builder
		WriteJavaScriptToYAMLWithoutComments(&yaml, script)
	}
}
