package workflow

import (
	"strings"
	"testing"
)

// TestBundleJavaScriptFsInsideFunctionWithMultilineDestructure tests the exact scenario
// from collect_ndjson_output.cjs where fs require is inside a function along with
// multiline destructured requires
func TestBundleJavaScriptFsInsideFunctionWithMultilineDestructure(t *testing.T) {
	// Create helper1 with multiline destructured require
	helper1Content := `const {
  validateItem,
  getMaxAllowedForType,
} = require("./validator.cjs");

function doValidation(item) {
  return validateItem(item);
}

module.exports = { doValidation };
`

	// Create validator content
	validatorContent := `function validateItem(item) {
  return { isValid: true };
}

function getMaxAllowedForType(type) {
  return 10;
}

module.exports = { validateItem, getMaxAllowedForType };
`

	// Create main content that mimics collect_ndjson_output.cjs structure
	mainContent := `async function main() {
  const fs = require("fs");
  const { doValidation } = require('./helper1.cjs');
  
  const configPath = "/tmp/config.json";
  try {
    if (fs.existsSync(configPath)) {
      const content = fs.readFileSync(configPath, "utf8");
      console.log(doValidation(content));
    }
  } catch (error) {
    console.error("Failed to read config: " + error.message);
  }
}

await main();
`

	sources := map[string]string{
		"helper1.cjs":   helper1Content,
		"validator.cjs": validatorContent,
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
		t.Logf("Full output:\n%s", bundled)
	}

	// Verify fs is used after it's defined
	fsIndex := strings.Index(bundled, `require("fs")`)
	existsIndex := strings.Index(bundled, "fs.existsSync")

	if fsIndex == -1 {
		t.Error("fs require not found in bundled output")
	}
	if existsIndex == -1 {
		t.Error("fs.existsSync not found in bundled output")
	}
	if fsIndex != -1 && existsIndex != -1 && fsIndex > existsIndex {
		t.Errorf("fs.existsSync (at %d) appears before require('fs') (at %d) - this would cause 'fs is not defined' error", existsIndex, fsIndex)

		// Show the problematic section
		start := max(0, existsIndex-100)
		end := min(len(bundled), existsIndex+100)
		t.Logf("Context around fs.existsSync:\n%s", bundled[start:end])
	}

	// Verify local requires are gone
	if strings.Contains(bundled, `require('./helper1.cjs')`) {
		t.Error("Bundled output still contains local require for helper1")
	}
	if strings.Contains(bundled, `require("./validator.cjs")`) {
		t.Error("Bundled output still contains local require for validator")
	}

	// Verify helper functions are included
	if !strings.Contains(bundled, "function doValidation") {
		t.Error("Bundled output does not contain inlined doValidation function")
	}
	if !strings.Contains(bundled, "function validateItem") {
		t.Error("Bundled output does not contain inlined validateItem function")
	}
}
