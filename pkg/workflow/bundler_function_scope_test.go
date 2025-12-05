package workflow

import (
"strings"
"testing"
)

func TestBundleJavaScriptWithRequireInsideFunction(t *testing.T) {
// Create helper content that also uses fs
helperContent := `const path = require("path");

function helperFunc() {
  return path.basename("/tmp/file.txt");
}

module.exports = { helperFunc };
`

// Create main content with require inside a function
mainContent := `async function main() {
  const fs = require("fs");
  const { helperFunc } = require('./helper.cjs');
  
  if (fs.existsSync("/tmp/test.txt")) {
    console.log(helperFunc());
  }
}

main();
`

sources := map[string]string{
"helper.cjs": helperContent,
}

// Bundle the main content
bundled, err := BundleJavaScriptFromSources(mainContent, sources, "")
if err != nil {
t.Fatalf("BundleJavaScriptFromSources failed: %v", err)
}

t.Logf("Bundled output:\n%s", bundled)

// Verify fs require is present
if !strings.Contains(bundled, `require("fs")`) {
t.Error("Bundled output does not contain fs require")
}

// Verify fs is used
if !strings.Contains(bundled, "fs.existsSync") {
t.Error("Bundled output does not contain fs.existsSync usage")
}

// Verify path require is present (from inlined helper)
if !strings.Contains(bundled, `require("path")`) {
t.Error("Bundled output does not contain path require")
}

// Verify local require is gone
if strings.Contains(bundled, `require('./helper.cjs')`) {
t.Error("Bundled output still contains local require")
}

// Verify helper function is included
if !strings.Contains(bundled, "function helperFunc") {
t.Error("Bundled output does not contain inlined helper function")
}
}
