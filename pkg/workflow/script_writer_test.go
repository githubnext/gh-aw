package workflow

import (
	"strings"
	"testing"
)

func TestWriteScript(t *testing.T) {
	tests := []struct {
		name        string
		title       string
		jsSource    string
		env         map[string]string
		githubToken string
		expected    string
	}{
		{
			name:        "Basic script without environment variables",
			title:       "Test Script",
			jsSource:    `console.log("Hello, World!");`,
			env:         nil,
			githubToken: "",
			expected: `      - name: Test Script
        uses: actions/github-script@v8
        with:
          script: |
            console.log("Hello, World!");
`,
		},
		{
			name:        "Script with environment variables",
			title:       "Script with Env",
			jsSource:    `console.log("Test");`,
			githubToken: "",
			env: map[string]string{
				"TEST_VAR":     "test_value",
				"ANOTHER_VAR": "${{ github.token }}",
			},
			expected: `      - name: Script with Env
        uses: actions/github-script@v8
        env:
          ANOTHER_VAR: ${{ github.token }}
          TEST_VAR: test_value
        with:
          script: |
            console.log("Test");
`,
		},
		{
			name:        "Script with comments removed",
			title:       "Clean Script",
			githubToken: "",
			jsSource: `// This is a single line comment
const value = "test"; // inline comment
/* This is a 
   multi-line comment */
console.log(value);`,
			env: map[string]string{
				"DEBUG": "true",
			},
			expected: `      - name: Clean Script
        uses: actions/github-script@v8
        env:
          DEBUG: true
        with:
          script: |
            const value = "test";
            console.log(value);
`,
		},
		{
			name:        "Empty environment map",
			title:       "No Env Script",
			jsSource:    `return "success";`,
			env:         map[string]string{},
			githubToken: "",
			expected: `      - name: No Env Script
        uses: actions/github-script@v8
        with:
          script: |
            return "success";
`,
		},
		{
			name:     "Empty environment map",
			title:    "No Env Script",
			jsSource: `return "success";`,
			env:      map[string]string{},
			expected: `      - name: No Env Script
        uses: actions/github-script@v8
        with:
          script: |
            return "success";
`,
		},
		{
			name:     "Script with GitHub token",
			title:    "Token Script",
			jsSource: `console.log("test");`,
			env:      map[string]string{"TEST": "value"},
			githubToken: "${{ secrets.GITHUB_TOKEN }}",
			expected: `      - name: Token Script
        uses: actions/github-script@v8
        env:
          TEST: value
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            console.log("test");
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var yaml strings.Builder
			writeScript(&yaml, tt.title, tt.jsSource, tt.env, tt.githubToken)
			result := yaml.String()

			if result != tt.expected {
				t.Errorf("writeScript() = \n%q\n\nwant:\n%q", result, tt.expected)
			}
		})
	}
}

func TestRemoveJavaScriptComments(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "No comments",
			input:    `const x = 42;\nconsole.log(x);`,
			expected: `const x = 42;\nconsole.log(x);`,
		},
		{
			name:     "Single line comment only",
			input:    "// This is a comment\nconst x = 42;",
			expected: "const x = 42;",
		},
		{
			name:     "Inline single line comment",
			input:    "const x = 42; // This is inline\nconsole.log(x);",
			expected: "const x = 42;\nconsole.log(x);",
		},
		{
			name:     "Multi-line comment",
			input:    "/* This is a\n   multi-line comment */\nconst x = 42;",
			expected: "const x = 42;",
		},
		{
			name:     "Multi-line comment on single line",
			input:    "const x = /* comment */ 42;",
			expected: "const x =  42;",
		},
		{
			name: "Mixed comments",
			input: `// Header comment
const x = 42; // inline comment
/* Multi-line
   comment here */
console.log(x); // final comment`,
			expected: `const x = 42;
console.log(x);`,
		},
		{
			name:     "Empty lines and whitespace",
			input:    "// Comment\n\nconst x = 42;\n\n// Another comment\nconsole.log(x);",
			expected: "const x = 42;\nconsole.log(x);",
		},
		{
			name:     "Nested-like comments (not actually nested)",
			input:    "/* outer /* inner */ comment */\nconst x = 42;",
			expected: "comment */\nconst x = 42;",
		},
		{
			name:     "Comment with special characters",
			input:    "// Comment with @#$%^&*()_+ special chars\nconst x = 42;",
			expected: "const x = 42;",
		},
		{
			name:     "Multiple single-line comments on same line",
			input:    "const x = 42; // comment1 // comment2",
			expected: "const x = 42;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeJavaScriptComments(tt.input)
			if result != tt.expected {
				t.Errorf("removeJavaScriptComments() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestRemoveJavaScriptCommentsRegex(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Single line comment",
			input:    "const x = 42; // comment\nconsole.log(x);",
			expected: "const x = 42;\nconsole.log(x);",
		},
		{
			name:     "Multi-line comment",
			input:    "const x = /* comment */ 42;",
			expected: "const x =  42;",
		},
		{
			name: "Mixed comments",
			input: `// Header comment
const x = 42; // inline comment
/* Multi-line comment */
console.log(x);`,
			expected: `const x = 42;
console.log(x);`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeJavaScriptCommentsRegex(tt.input)
			if result != tt.expected {
				t.Errorf("removeJavaScriptCommentsRegex() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestWriteScriptEnvironmentVariableOrdering(t *testing.T) {
	// Test that environment variables are written in a consistent order
	env := map[string]string{
		"Z_VAR": "z_value",
		"A_VAR": "a_value",
		"M_VAR": "m_value",
	}

	var yaml strings.Builder
	writeScript(&yaml, "Test", "console.log('test');", env, "")
	result := yaml.String()

	// Check that all environment variables are present
	if !strings.Contains(result, "A_VAR: a_value") {
		t.Error("Expected A_VAR in output")
	}
	if !strings.Contains(result, "M_VAR: m_value") {
		t.Error("Expected M_VAR in output")
	}
	if !strings.Contains(result, "Z_VAR: z_value") {
		t.Error("Expected Z_VAR in output")
	}

	// Check that env section is properly formatted
	if !strings.Contains(result, "        env:\n") {
		t.Error("Expected env section with proper indentation")
	}
}

func TestWriteScriptIntegrationWithExistingFunctions(t *testing.T) {
	// Test that writeScript integrates properly with existing WriteJavaScriptToYAML function
	jsCode := `function test() {
    return "hello";
}
test();`

	var yaml strings.Builder
	writeScript(&yaml, "Integration Test", jsCode, map[string]string{"TEST": "value"}, "")
	result := yaml.String()

	// Check that JavaScript is properly indented (12 spaces for script content)
	lines := strings.Split(result, "\n")
	jsStarted := false
	for _, line := range lines {
		if strings.Contains(line, "script: |") {
			jsStarted = true
			continue
		}
		if jsStarted && strings.TrimSpace(line) != "" {
			if !strings.HasPrefix(line, "            ") {
				t.Errorf("JavaScript line not properly indented: %q", line)
			}
		}
	}
}