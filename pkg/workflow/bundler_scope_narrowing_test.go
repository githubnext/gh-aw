package workflow

import (
	"strings"
	"testing"
)

// TestBundleJavaScriptScopeNarrowing tests what happens when the FIRST require
// is inside a function, which would cause all other requires to be written there too
func TestBundleJavaScriptScopeNarrowing(t *testing.T) {
	// Main content where the FIRST require is inside a function
	mainContent := `async function main() {
  const fs = require("fs");
  
  if (fs.existsSync("/tmp/test.txt")) {
    const content = fs.readFileSync("/tmp/test.txt", "utf8");
    console.log(content);
  }
}

// This function runs at module load time, BEFORE main() is called
const path = require("path");
console.log("Module loaded, path:", path.basename("/tmp/file.txt"));

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

	// Critical check: path is used OUTSIDE the function, so it must be defined OUTSIDE too
	pathRequireIndex := strings.Index(bundled, `const path = require("path")`)
	pathBasenameIndex := strings.Index(bundled, "path.basename")
	mainFuncIndex := strings.Index(bundled, "async function main()")
	
	if pathRequireIndex == -1 {
		t.Fatal("path require not found")
	}
	if pathBasenameIndex == -1 {
		t.Fatal("path.basename not found")
	}
	if mainFuncIndex == -1 {
		t.Fatal("main function not found")
	}

	// Check if path require is inside main function (it shouldn't be, or path.basename will fail)
	// Find the closing brace of main function
	mainFuncEnd := strings.Index(bundled[mainFuncIndex:], "\n}\n")
	if mainFuncEnd == -1 {
		mainFuncEnd = strings.Index(bundled[mainFuncIndex:], "\n}")
	}
	if mainFuncEnd != -1 {
		mainFuncEnd += mainFuncIndex // Make it absolute position
		
		if pathRequireIndex > mainFuncIndex && pathRequireIndex < mainFuncEnd {
			t.Errorf("FOUND THE BUG: path require is inside main() but path.basename is outside")
			t.Logf("main() starts at %d, ends at %d", mainFuncIndex, mainFuncEnd)
			t.Logf("path require at %d", pathRequireIndex)
			t.Logf("path.basename at %d", pathBasenameIndex)
			
			// Show the problematic bundled code
			t.Logf("\nProblematic bundled code:\n%s", bundled)
		} else {
			// Path is correctly at top level
			t.Logf("✓ path require is correctly at top level (outside main function)")
			t.Logf("main() range: %d - %d", mainFuncIndex, mainFuncEnd)
			t.Logf("path require at %d", pathRequireIndex)
		}
	}

	// Also check that path is defined before it's used
	if pathRequireIndex > pathBasenameIndex {
		t.Errorf("path.basename appears before path require")
	}
}
