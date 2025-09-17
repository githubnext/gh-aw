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

func TestFormatJavaScriptForYAMLWithOptions(t *testing.T) {
	tests := []struct {
		name            string
		script          string
		indentSpaces    int
		excludeComments bool
		expected        []string
	}{
		{
			name:            "custom 4-space indentation",
			script:          "const x = 1;\nconsole.log(x);",
			indentSpaces:    4,
			excludeComments: false,
			expected: []string{
				"    const x = 1;\n",
				"    console.log(x);\n",
			},
		},
		{
			name:            "exclude JavaScript comments",
			script:          "// Comment\nconst x = 1;\n// Another comment\nconsole.log(x);",
			indentSpaces:    2,
			excludeComments: true,
			expected: []string{
				"  const x = 1;\n",
				"  console.log(x);\n",
			},
		},
		{
			name:            "include JavaScript comments",
			script:          "// Comment\nconst x = 1;\n// Another comment\nconsole.log(x);",
			indentSpaces:    2,
			excludeComments: false,
			expected: []string{
				"  // Comment\n",
				"  const x = 1;\n",
				"  // Another comment\n",
				"  console.log(x);\n",
			},
		},
		{
			name:            "zero indentation",
			script:          "const x = 1;\nconsole.log(x);",
			indentSpaces:    0,
			excludeComments: false,
			expected: []string{
				"const x = 1;\n",
				"console.log(x);\n",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatJavaScriptForYAMLWithOptions(tt.script, tt.indentSpaces, tt.excludeComments)

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

func TestWriteJavaScriptToYAMLWithOptions(t *testing.T) {
	tests := []struct {
		name            string
		script          string
		indentSpaces    int
		excludeComments bool
		expected        string
	}{
		{
			name:            "8-space indentation with comments excluded",
			script:          "// Header comment\nconst x = 1;\nconsole.log(x);",
			indentSpaces:    8,
			excludeComments: true,
			expected:        "        const x = 1;\n        console.log(x);\n",
		},
		{
			name:            "6-space indentation with comments included",
			script:          "// Important comment\nconst y = 2;\nreturn y;",
			indentSpaces:    6,
			excludeComments: false,
			expected:        "      // Important comment\n      const y = 2;\n      return y;\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var yaml strings.Builder
			WriteJavaScriptToYAMLWithOptions(&yaml, tt.script, tt.indentSpaces, tt.excludeComments)
			result := yaml.String()

			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}
