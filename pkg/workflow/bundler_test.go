package workflow

import (
	"strings"
	"testing"
)

func TestBundleJavaScriptFromSources(t *testing.T) {
	// Create helper content
	helperContent := `// Helper module for validation
function validatePositiveInteger(value, fieldName, lineNum) {
  if (value === undefined || value === null) {
    return {
      isValid: false,
      error: "Line " + lineNum + ": " + fieldName + " is required",
    };
  }
  
  if (typeof value !== "number" && typeof value !== "string") {
    return {
      isValid: false,
      error: "Line " + lineNum + ": " + fieldName + " must be a number or string",
    };
  }
  
  const parsed = typeof value === "string" ? parseInt(value, 10) : value;
  if (isNaN(parsed) || parsed <= 0 || !Number.isInteger(parsed)) {
    return {
      isValid: false,
      error: "Line " + lineNum + ": " + fieldName + " must be a positive integer (got: " + value + ")",
    };
  }
  
  return { isValid: true, normalizedValue: parsed };
}

module.exports = { validatePositiveInteger };
`

	// Create main content that requires the helper
	mainContent := `// Main script
const { validatePositiveInteger } = require('./helper.cjs');

async function main() {
  const result = validatePositiveInteger(5, 'testField', 1);
  console.log(result);
}

main();
`

	// Create sources map
	sources := map[string]string{
		"helper.cjs": helperContent,
	}

	// Bundle the main content
	bundled, err := BundleJavaScriptFromSources(mainContent, sources, "")
	if err != nil {
		t.Fatalf("BundleJavaScriptFromSources failed: %v", err)
	}

	// Verify the bundled output
	t.Logf("Bundled output:\n%s", bundled)

	// Check that the require statement is replaced with inlined content
	if strings.Contains(bundled, "require('./helper.cjs')") {
		t.Error("Bundled output still contains require statement")
	}

	// Check that the helper function is included
	if !strings.Contains(bundled, "function validatePositiveInteger") {
		t.Error("Bundled output does not contain inlined function")
	}

	// Check that module.exports is removed
	if strings.Contains(bundled, "module.exports") {
		t.Error("Bundled output still contains module.exports")
	}

	// Check that inlining comments are present
	if !strings.Contains(bundled, "Inlined from ./helper.cjs") {
		t.Error("Bundled output does not contain inlining comment")
	}
}

func TestBundleJavaScriptFromSourcesWithoutRequires(t *testing.T) {
	// Create a simple content without any requires
	simpleContent := `// Simple script
function hello() {
  console.log("Hello, world!");
}

hello();
`

	// Empty sources map
	sources := map[string]string{}

	// Bundle the simple content
	bundled, err := BundleJavaScriptFromSources(simpleContent, sources, "")
	if err != nil {
		t.Fatalf("BundleJavaScriptFromSources failed: %v", err)
	}

	// Verify the bundled output is the same as the input
	if !strings.Contains(bundled, "function hello()") {
		t.Error("Bundled output does not contain original content")
	}

	// Should not have any inlining comments
	if strings.Contains(bundled, "Inlined from") {
		t.Error("Bundled output contains unexpected inlining comments")
	}
}

func TestRemoveExports(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "module.exports with object",
			input: `function test() {
  return 42;
}

module.exports = { test };
`,
			expected: `function test() {
  return 42;
}

`,
		},
		{
			name: "exports.property assignment",
			input: `function helper() {
  return "help";
}

exports.helper = helper;
`,
			expected: `function helper() {
  return "help";
}

`,
		},
		{
			name: "no exports",
			input: `function standalone() {
  return true;
}
`,
			expected: `function standalone() {
  return true;
}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeExports(tt.input)
			if result != tt.expected {
				t.Errorf("removeExports() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestBundleJavaScriptFromSourcesWithMultipleRequires(t *testing.T) {
	// Create helper1 content
	helper1Content := `function helperOne() {
  return "one";
}

module.exports = { helperOne };
`

	// Create helper2 content
	helper2Content := `function helperTwo() {
  return "two";
}

module.exports = { helperTwo };
`

	// Create main content that requires both helpers
	mainContent := `const { helperOne } = require('./helper1.cjs');
const { helperTwo } = require('./helper2.cjs');

async function main() {
  console.log(helperOne(), helperTwo());
}

main();
`

	// Create sources map
	sources := map[string]string{
		"helper1.cjs": helper1Content,
		"helper2.cjs": helper2Content,
	}

	// Bundle the main content
	bundled, err := BundleJavaScriptFromSources(mainContent, sources, "")
	if err != nil {
		t.Fatalf("BundleJavaScriptFromSources failed: %v", err)
	}

	// Verify both helpers are inlined
	if !strings.Contains(bundled, "function helperOne") {
		t.Error("Bundled output does not contain helperOne")
	}

	if !strings.Contains(bundled, "function helperTwo") {
		t.Error("Bundled output does not contain helperTwo")
	}

	// Verify both require statements are gone
	if strings.Contains(bundled, "require('./helper1.cjs')") {
		t.Error("Bundled output still contains require for helper1")
	}

	if strings.Contains(bundled, "require('./helper2.cjs')") {
		t.Error("Bundled output still contains require for helper2")
	}
}

func TestBundleJavaScriptFromSourcesWithNestedPath(t *testing.T) {
	// Create helper in lib directory
	helperContent := `function sanitize(text) {
  return text.trim();
}

module.exports = { sanitize };
`

	// Create main content that requires the helper from lib
	mainContent := `const { sanitize } = require('./lib/sanitize.cjs');

async function main() {
  console.log(sanitize("  hello  "));
}

main();
`

	// Create sources map with nested path
	sources := map[string]string{
		"lib/sanitize.cjs": helperContent,
	}

	// Bundle the main content
	bundled, err := BundleJavaScriptFromSources(mainContent, sources, "")
	if err != nil {
		t.Fatalf("BundleJavaScriptFromSources failed: %v", err)
	}

	// Verify the helper function is included
	if !strings.Contains(bundled, "function sanitize") {
		t.Error("Bundled output does not contain sanitize function")
	}

	// Check that the require statement is replaced
	if strings.Contains(bundled, "require('./lib/sanitize.cjs')") {
		t.Error("Bundled output still contains require statement")
	}
}
