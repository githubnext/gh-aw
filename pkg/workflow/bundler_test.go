package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBundleJavaScript(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create a helper file
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

	helperPath := filepath.Join(tmpDir, "helper.cjs")
	err := os.WriteFile(helperPath, []byte(helperContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create helper file: %v", err)
	}

	// Create a main file that requires the helper
	mainContent := `// Main script
const { validatePositiveInteger } = require('./helper.cjs');

async function main() {
  const result = validatePositiveInteger(5, 'testField', 1);
  console.log(result);
}

main();
`

	mainPath := filepath.Join(tmpDir, "main.cjs")
	err = os.WriteFile(mainPath, []byte(mainContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create main file: %v", err)
	}

	// Bundle the main file
	bundled, err := BundleJavaScript(mainPath)
	if err != nil {
		t.Fatalf("BundleJavaScript failed: %v", err)
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

func TestBundleJavaScriptWithoutRequires(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create a simple file without any requires
	simpleContent := `// Simple script
function hello() {
  console.log("Hello, world!");
}

hello();
`

	simplePath := filepath.Join(tmpDir, "simple.cjs")
	err := os.WriteFile(simplePath, []byte(simpleContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create simple file: %v", err)
	}

	// Bundle the simple file
	bundled, err := BundleJavaScript(simplePath)
	if err != nil {
		t.Fatalf("BundleJavaScript failed: %v", err)
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

func TestBundleJavaScriptWithMultipleRequires(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create helper1.cjs
	helper1Content := `function helperOne() {
  return "one";
}

module.exports = { helperOne };
`

	helper1Path := filepath.Join(tmpDir, "helper1.cjs")
	err := os.WriteFile(helper1Path, []byte(helper1Content), 0644)
	if err != nil {
		t.Fatalf("Failed to create helper1 file: %v", err)
	}

	// Create helper2.cjs
	helper2Content := `function helperTwo() {
  return "two";
}

module.exports = { helperTwo };
`

	helper2Path := filepath.Join(tmpDir, "helper2.cjs")
	err = os.WriteFile(helper2Path, []byte(helper2Content), 0644)
	if err != nil {
		t.Fatalf("Failed to create helper2 file: %v", err)
	}

	// Create main file that requires both helpers
	mainContent := `const { helperOne } = require('./helper1.cjs');
const { helperTwo } = require('./helper2.cjs');

async function main() {
  console.log(helperOne(), helperTwo());
}

main();
`

	mainPath := filepath.Join(tmpDir, "main.cjs")
	err = os.WriteFile(mainPath, []byte(mainContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create main file: %v", err)
	}

	// Bundle the main file
	bundled, err := BundleJavaScript(mainPath)
	if err != nil {
		t.Fatalf("BundleJavaScript failed: %v", err)
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
