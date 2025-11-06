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
	mainContent := `const { sanitize } = require('./sanitize.cjs');

async function main() {
  console.log(sanitize("  hello  "));
}

main();
`

	// Create sources map with nested path
	sources := map[string]string{
		"sanitize.cjs": helperContent,
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
	if strings.Contains(bundled, "require('./sanitize.cjs')") {
		t.Error("Bundled output still contains require statement")
	}
}

// TestValidateNoLocalRequires tests the validateNoLocalRequires function
func TestValidateNoLocalRequires(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectError bool
	}{
		{
			name: "no local requires",
			content: `function test() {
  console.log("hello");
}`,
			expectError: false,
		},
		{
			name: "npm package require is ok",
			content: `const fs = require("fs");
const path = require("path");
function test() {
  console.log("hello");
}`,
			expectError: false,
		},
		{
			name: "local require with ./ should error",
			content: `const { helper } = require("./helper.cjs");
function test() {
  console.log("hello");
}`,
			expectError: true,
		},
		{
			name: "local require with ../ should error",
			content: `const utils = require("../utils.cjs");
function test() {
  console.log("hello");
}`,
			expectError: true,
		},
		{
			name: "multiple local requires should error",
			content: `const { helper } = require("./helper.cjs");
const utils = require("../utils.cjs");
function test() {
  console.log("hello");
}`,
			expectError: true,
		},
		{
			name: "require in string should error",
			content: `const code = 'const x = require("./helper.cjs");';
function test() {
  console.log(code);
}`,
			expectError: true,
		},
		{
			name: "require in double-quoted string should error",
			content: `const code = "const x = require('./helper.cjs');";
function test() {
  console.log(code);
}`,
			expectError: true,
		},
		{
			name:        "require in backtick string should error",
			content:     "const code = `const x = require('./helper.cjs');`;\nfunction test() {\n  console.log(code);\n}",
			expectError: true,
		},
		{
			name: "inlined content markers with npm requires is ok",
			content: `// === Inlined from ./helper.cjs ===
const fs = require("fs");
function helper() {
  return "test";
}
// === End of ./helper.cjs ===`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateNoLocalRequires(tt.content)
			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// TestBundleJavaScriptValidationSuccess tests that validation passes for properly bundled code
func TestBundleJavaScriptValidationSuccess(t *testing.T) {
	// Create helper content
	helperContent := `function helperFunc() {
  return "helper";
}

module.exports = { helperFunc };
`

	// Create main content with local require
	mainContent := `const { helperFunc } = require('./helper.cjs');

function main() {
  console.log(helperFunc());
}

main();
`

	sources := map[string]string{
		"helper.cjs": helperContent,
	}

	// This should succeed - the require will be inlined and validation should pass
	bundled, err := BundleJavaScriptFromSources(mainContent, sources, "")
	if err != nil {
		t.Fatalf("Expected bundling to succeed, but got error: %v", err)
	}

	// Verify the bundled output doesn't contain the local require
	if strings.Contains(bundled, `require('./helper.cjs')`) {
		t.Error("Bundled output still contains local require - validation should have caught this")
	}
}

// TestBundleJavaScriptValidationFailure tests that validation fails when a local require cannot be inlined
func TestBundleJavaScriptValidationFailure(t *testing.T) {
	// Create main content with a local require to a non-existent file
	mainContent := `const { helper } = require('./missing.cjs');

function main() {
  console.log(helper());
}

main();
`

	sources := map[string]string{
		// No "missing.cjs" in sources
	}

	// This should fail because missing.cjs is not in sources
	_, err := BundleJavaScriptFromSources(mainContent, sources, "")
	if err == nil {
		t.Fatal("Expected bundling to fail due to missing file, but got no error")
	}

	// Check that the error mentions the missing file
	if !strings.Contains(err.Error(), "missing.cjs") {
		t.Errorf("Error should mention missing file, got: %v", err)
	}
}

// TestBundleJavaScriptWithNpmPackages tests that npm package requires are preserved
func TestBundleJavaScriptWithNpmPackages(t *testing.T) {
	// Create helper content
	helperContent := `const path = require("path");

function helperFunc(filepath) {
  return path.basename(filepath);
}

module.exports = { helperFunc };
`

	// Create main content with both local and npm requires
	mainContent := `const fs = require("fs");
const { helperFunc } = require('./helper.cjs');

function main() {
  const files = fs.readdirSync(".");
  files.forEach(f => console.log(helperFunc(f)));
}

main();
`

	sources := map[string]string{
		"helper.cjs": helperContent,
	}

	// This should succeed - npm requires should be kept, local require should be inlined
	bundled, err := BundleJavaScriptFromSources(mainContent, sources, "")
	if err != nil {
		t.Fatalf("Expected bundling to succeed, but got error: %v", err)
	}

	// Verify npm requires are still present
	if !strings.Contains(bundled, `require("fs")`) {
		t.Error("Bundled output should still contain npm require for 'fs'")
	}
	if !strings.Contains(bundled, `require("path")`) {
		t.Error("Bundled output should still contain npm require for 'path'")
	}

	// Verify local require is gone
	if strings.Contains(bundled, `require('./helper.cjs')`) {
		t.Error("Bundled output should not contain local require")
	}
}

// TestBundleJavaScriptFallbackValidation tests that validation catches local requires
// even when bundling fails and falls back to using source as-is
func TestBundleJavaScriptFallbackValidation(t *testing.T) {
	// Create main content with a local require to a non-existent file
	mainContent := `const fs = require("fs");
const { generateStagedPreview } = require("./staged_preview.cjs");

async function main() {
  console.log("test");
}

main();
`

	// Empty sources map - staged_preview.cjs is not available
	sources := map[string]string{}

	// This should fail because staged_preview.cjs is not in sources
	// The bundler should fail, and then the fallback code should be validated
	_, err := BundleJavaScriptFromSources(mainContent, sources, "")
	if err == nil {
		t.Fatal("Expected bundling to fail and validator to catch local require in fallback, but got no error")
	}

	// Check that the error mentions the missing file (bundling error) or local require (validation error)
	errMsg := err.Error()
	if !strings.Contains(errMsg, "staged_preview.cjs") {
		t.Errorf("Error should mention staged_preview.cjs, got: %v", err)
	}
}
