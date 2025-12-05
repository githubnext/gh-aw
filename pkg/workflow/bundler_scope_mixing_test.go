package workflow

import (
	"strings"
	"testing"
)

// TestBundleJavaScriptWithMixedScopeRequires tests what happens when requires
// are at different scopes (top-level vs inside functions)
func TestBundleJavaScriptWithMixedScopeRequires(t *testing.T) {
	// Main content with top-level require AND function-scoped require
	mainContent := `const path = require("path");

function setup() {
  console.log("Setup with path:", path.basename("/tmp/file.txt"));
}

async function main() {
  const fs = require("fs");
  
  if (fs.existsSync("/tmp/test.txt")) {
    const content = fs.readFileSync("/tmp/test.txt", "utf8");
    console.log(content);
  }
}

setup();
await main();
`

	sources := map[string]string{}

	// Bundle the main content
	bundled, err := BundleJavaScriptFromSources(mainContent, sources, "")
	if err != nil {
		t.Fatalf("BundleJavaScriptFromSources failed: %v", err)
	}

	t.Logf("Bundled output:\n%s", bundled)

	// Verify both requires are present
	if !strings.Contains(bundled, `require("path")`) {
		t.Error("Bundled output does not contain path require")
	}
	if !strings.Contains(bundled, `require("fs")`) {
		t.Error("Bundled output does not contain fs require")
	}

	// Check that fs is defined before it's used
	fsRequireIndex := strings.Index(bundled, `const fs = require("fs")`)
	fsExistsIndex := strings.Index(bundled, "fs.existsSync")
	
	if fsRequireIndex == -1 {
		t.Error("fs require not found")
	}
	if fsExistsIndex == -1 {
		t.Error("fs.existsSync not found")
	}
	if fsRequireIndex > fsExistsIndex {
		t.Errorf("fs.existsSync appears before fs require - this causes 'fs is not defined' error")
		t.Logf("fs require at position %d, fs.existsSync at position %d", fsRequireIndex, fsExistsIndex)
	}

	// Check that path is defined before it's used
	pathRequireIndex := strings.Index(bundled, `const path = require("path")`)
	pathBasenameIndex := strings.Index(bundled, "path.basename")
	
	if pathRequireIndex == -1 {
		t.Error("path require not found")
	}
	if pathBasenameIndex == -1 {
		t.Error("path.basename not found")
	}
	if pathRequireIndex > pathBasenameIndex {
		t.Errorf("path.basename appears before path require - this causes 'path is not defined' error")
		t.Logf("path require at position %d, path.basename at position %d", pathRequireIndex, pathBasenameIndex)
	}
}
