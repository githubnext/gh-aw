package workflow

import (
	"strings"
	"testing"
)

func TestRuntimeModeString(t *testing.T) {
	tests := []struct {
		mode     RuntimeMode
		expected string
	}{
		{RuntimeModeGitHubScript, "github-script"},
		{RuntimeModeNodeJS, "nodejs"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.mode.String(); got != tt.expected {
				t.Errorf("RuntimeMode.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestBundleJavaScriptWithMode_GitHubScript(t *testing.T) {
	// Create helper content with module.exports
	helperContent := `function validateInput(value) {
  return value !== null && value !== undefined;
}

module.exports = { validateInput };
`

	// Create main content that requires the helper
	mainContent := `const { validateInput } = require('./helper.cjs');

function main() {
  const result = validateInput(42);
  console.log(result);
}

main();
`

	sources := map[string]string{
		"helper.cjs": helperContent,
	}

	// Bundle with GitHub Script mode
	bundled, err := BundleJavaScriptWithMode(mainContent, sources, "", RuntimeModeGitHubScript)
	if err != nil {
		t.Fatalf("BundleJavaScriptWithMode failed: %v", err)
	}

	// Verify the bundled output
	t.Logf("Bundled output:\n%s", bundled)

	// Check that the require statement is replaced with inlined content
	if strings.Contains(bundled, "require('./helper.cjs')") {
		t.Error("Bundled output still contains require statement")
	}

	// Check that the helper function is included
	if !strings.Contains(bundled, "function validateInput") {
		t.Error("Bundled output does not contain inlined function")
	}

	// Check that module.exports is removed (GitHub Script mode)
	if strings.Contains(bundled, "module.exports") {
		t.Error("GitHub Script mode: bundled output still contains module.exports")
	}

	// Check that inlining comments are present
	if !strings.Contains(bundled, "Inlined from ./helper.cjs") {
		t.Error("Bundled output does not contain inlining comment")
	}
}

func TestBundleJavaScriptWithMode_NodeJS(t *testing.T) {
	// Create helper content with module.exports
	helperContent := `function processData(data) {
  return data.trim().toUpperCase();
}

module.exports = { processData };
`

	// Create main content that requires the helper
	mainContent := `const { processData } = require('./helper.cjs');

function main() {
  const result = processData("  hello world  ");
  console.log(result);
}

module.exports = { main };
`

	sources := map[string]string{
		"helper.cjs": helperContent,
	}

	// Bundle with Node.js mode
	bundled, err := BundleJavaScriptWithMode(mainContent, sources, "", RuntimeModeNodeJS)
	if err != nil {
		t.Fatalf("BundleJavaScriptWithMode failed: %v", err)
	}

	// Verify the bundled output
	t.Logf("Bundled output:\n%s", bundled)

	// Check that the helper function is included
	if !strings.Contains(bundled, "function processData") {
		t.Error("Bundled output does not contain inlined function")
	}

	// Check that module.exports is PRESERVED (Node.js mode)
	if !strings.Contains(bundled, "module.exports") {
		t.Error("Node.js mode: bundled output should preserve module.exports")
	}

	// Count module.exports occurrences (should be 2: one from helper, one from main)
	count := strings.Count(bundled, "module.exports")
	if count != 2 {
		t.Errorf("Node.js mode: expected 2 module.exports statements, got %d", count)
	}
}

func TestBundleJavaScriptWithMode_GitHubScriptValidation(t *testing.T) {
	// Test that GitHub Script mode validates no module references
	helperContent := `function test() {
  return true;
}
// This should be removed
module.exports = { test };
`

	mainContent := `const { test } = require('./helper.cjs');
console.log(test());
`

	sources := map[string]string{
		"helper.cjs": helperContent,
	}

	// This should succeed - module.exports should be removed
	bundled, err := BundleJavaScriptWithMode(mainContent, sources, "", RuntimeModeGitHubScript)
	if err != nil {
		t.Fatalf("Expected bundling to succeed, but got error: %v", err)
	}

	// Verify no module.exports in output
	if strings.Contains(bundled, "module.exports") {
		t.Error("GitHub Script mode validation should have caught module.exports")
	}
}

func TestValidateNoModuleReferences(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectError bool
	}{
		{
			name: "no module references",
			content: `function test() {
  console.log("hello");
}`,
			expectError: false,
		},
		{
			name: "module.exports reference should error",
			content: `function test() {
  return 42;
}

module.exports = { test };`,
			expectError: true,
		},
		{
			name: "exports.property reference should error",
			content: `function helper() {
  return "help";
}

exports.helper = helper;`,
			expectError: true,
		},
		{
			name: "module.exports in comment should be ok",
			content: `function test() {
  // This function would normally be exported via module.exports
  return true;
}`,
			expectError: false,
		},
		{
			name: "multiple module references should error",
			content: `function test() {
  return 42;
}

module.exports = { test };
exports.helper = test;`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateNoModuleReferences(tt.content)
			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestBundleJavaScriptFromSources_BackwardCompatibility(t *testing.T) {
	// Test that the old function still works and defaults to GitHub Script mode
	helperContent := `function helper() {
  return "test";
}

module.exports = { helper };
`

	mainContent := `const { helper } = require('./helper.cjs');

function main() {
  console.log(helper());
}

main();
`

	sources := map[string]string{
		"helper.cjs": helperContent,
	}

	// Use old function signature
	bundled, err := BundleJavaScriptFromSources(mainContent, sources, "")
	if err != nil {
		t.Fatalf("BundleJavaScriptFromSources failed: %v", err)
	}

	// Should behave like GitHub Script mode (remove module.exports)
	if strings.Contains(bundled, "module.exports") {
		t.Error("Old function should default to GitHub Script mode and remove module.exports")
	}

	// Should still inline the helper
	if !strings.Contains(bundled, "function helper") {
		t.Error("Bundled output does not contain inlined function")
	}
}

func TestBundleJavaScriptWithMode_MultipleFiles_NodeJS(t *testing.T) {
	// Test bundling multiple files in Node.js mode
	helper1Content := `function helper1() {
  return "one";
}

module.exports = { helper1 };
`

	helper2Content := `function helper2() {
  return "two";
}

module.exports = { helper2 };
`

	mainContent := `const { helper1 } = require('./helper1.cjs');
const { helper2 } = require('./helper2.cjs');

function main() {
  console.log(helper1(), helper2());
}

module.exports = { main };
`

	sources := map[string]string{
		"helper1.cjs": helper1Content,
		"helper2.cjs": helper2Content,
	}

	// Bundle with Node.js mode
	bundled, err := BundleJavaScriptWithMode(mainContent, sources, "", RuntimeModeNodeJS)
	if err != nil {
		t.Fatalf("BundleJavaScriptWithMode failed: %v", err)
	}

	// Check both helpers are inlined
	if !strings.Contains(bundled, "function helper1") {
		t.Error("Bundled output does not contain helper1")
	}
	if !strings.Contains(bundled, "function helper2") {
		t.Error("Bundled output does not contain helper2")
	}

	// Check all module.exports are preserved (3 total)
	count := strings.Count(bundled, "module.exports")
	if count != 3 {
		t.Errorf("Node.js mode: expected 3 module.exports statements, got %d", count)
	}
}
