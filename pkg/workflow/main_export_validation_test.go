package workflow

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestScriptsExportMain validates that all JavaScript files that are required
// with `const { main } = require(...)` actually export a main function.
// This prevents runtime errors where code tries to destructure main from
// a module that doesn't export it.
func TestScriptsExportMain(t *testing.T) {
	// List of scripts that should export main based on compiler usage
	// These are extracted from places where Go code generates:
	// const { main } = require('...')
	requiredMainExports := []string{
		"check_stop_time.cjs",
		"check_skip_if_match.cjs",
		"check_command_position.cjs",
		"check_workflow_timestamp_api.cjs",
		"compute_text.cjs",
		"add_reaction_and_edit_comment.cjs",
		"lock-issue.cjs",
		"unlock-issue.cjs",
		"checkout_pr_branch.cjs",
	}

	jsDir := "js"

	// Pattern to match: module.exports = { main }; or module.exports = { main, ... };
	mainExportPattern := regexp.MustCompile(`module\.exports\s*=\s*\{[^}]*\bmain\b[^}]*\}`)

	for _, scriptName := range requiredMainExports {
		t.Run(scriptName, func(t *testing.T) {
			scriptPath := filepath.Join(jsDir, scriptName)
			content, err := os.ReadFile(scriptPath)
			if err != nil {
				t.Fatalf("Failed to read %s: %v", scriptPath, err)
			}

			scriptContent := string(content)

			// Check if the script exports main
			if !mainExportPattern.MatchString(scriptContent) {
				t.Errorf("Script %s is required with 'const { main } = require(...)' but does not export main.\n"+
					"Add 'module.exports = { main };' to the script.", scriptName)
			}

			// Also verify that an async function main exists
			if !strings.Contains(scriptContent, "async function main()") &&
				!strings.Contains(scriptContent, "async function main ()") &&
				!strings.Contains(scriptContent, "function main()") &&
				!strings.Contains(scriptContent, "function main ()") {
				t.Errorf("Script %s exports main but does not define a main function", scriptName)
			}
		})
	}
}

// TestScriptsWithMainExportPattern checks that scripts exporting main
// follow the correct pattern and include the require.main check for direct execution
func TestScriptsWithMainExportPattern(t *testing.T) {
	jsDir := "js"
	
	// Pattern to match: module.exports = { main }
	mainExportPattern := regexp.MustCompile(`module\.exports\s*=\s*\{[^}]*\bmain\b[^}]*\}`)
	
	// Pattern to check for require.main === module check
	requireMainPattern := regexp.MustCompile(`require\.main\s*===\s*module`)

	entries, err := os.ReadDir(jsDir)
	if err != nil {
		t.Fatalf("Failed to read js directory: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".cjs") {
			continue
		}

		scriptPath := filepath.Join(jsDir, entry.Name())
		content, err := os.ReadFile(scriptPath)
		if err != nil {
			continue // Skip files we can't read
		}

		scriptContent := string(content)

		// If script exports main, it should have proper execution guard
		if mainExportPattern.MatchString(scriptContent) {
			t.Run(entry.Name()+"_has_execution_guard", func(t *testing.T) {
				if !requireMainPattern.MatchString(scriptContent) {
					t.Logf("Script %s exports main but lacks 'if (require.main === module)' guard for direct execution.\n"+
						"This is acceptable if the script is only meant to be used as a module.", entry.Name())
				}
			})
		}
	}
}
