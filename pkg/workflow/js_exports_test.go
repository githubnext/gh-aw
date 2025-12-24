package workflow

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestJavaScriptExports verifies that all JavaScript scripts used in lock files
// export a main function, and that all scripts packaged in custom actions are valid.
//
// This test ensures:
// 1. Scripts called with `const { main } = require()` in lock files export main
// 2. Scripts are properly packaged in the setup action
// 3. MCP server files in setup-safe-inputs and setup-safe-outputs are present
func TestJavaScriptExports(t *testing.T) {
	t.Run("ScriptsInLockFilesExportMain", func(t *testing.T) {
		// Get list of scripts used in lock files
		usedScripts := getScriptsUsedInLockFiles(t)
		if len(usedScripts) == 0 {
			t.Fatal("Expected to find scripts used in lock files, found none")
		}

		t.Logf("Found %d unique scripts called with main in lock files", len(usedScripts))

		// Check each script exports main
		repoRoot := getRepoRoot(t)
		setupJSDir := filepath.Join(repoRoot, "actions", "setup", "js")
		missingMain := []string{}

		for _, script := range usedScripts {
			scriptPath := filepath.Join(setupJSDir, script)
			
			// Check if file exists
			if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
				t.Errorf("Script %s used in lock files but not found in %s", script, setupJSDir)
				continue
			}

			// Check if it exports main
			if !hasMainExport(t, scriptPath) {
				missingMain = append(missingMain, script)
			}
		}

		if len(missingMain) > 0 {
			t.Errorf("The following scripts are used in lock files but don't export main:\n  - %s",
				strings.Join(missingMain, "\n  - "))
		}
	})

	t.Run("SetupActionPackagesScripts", func(t *testing.T) {
		repoRoot := getRepoRoot(t)
		setupJSDir := filepath.Join(repoRoot, "actions", "setup", "js")
		
		// Check if directory exists
		if _, err := os.Stat(setupJSDir); os.IsNotExist(err) {
			t.Skip("setup action js/ directory not built yet (run 'make actions-build')")
		}

		// Count .cjs files
		entries, err := os.ReadDir(setupJSDir)
		if err != nil {
			t.Fatalf("Failed to read %s: %v", setupJSDir, err)
		}

		cjsCount := 0
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".cjs") {
				cjsCount++
			}
		}

		if cjsCount == 0 {
			t.Error("setup action should package .cjs files but found none")
		}

		t.Logf("setup action packages %d .cjs files", cjsCount)
	})

	t.Run("SafeInputsActionPackagesFiles", func(t *testing.T) {
		repoRoot := getRepoRoot(t)
		indexPath := filepath.Join(repoRoot, "actions", "setup-safe-inputs", "index.js")
		
		// Check if file exists
		if _, err := os.Stat(indexPath); os.IsNotExist(err) {
			t.Skip("setup-safe-inputs index.js not built yet (run 'make actions-build')")
		}

		content, err := os.ReadFile(indexPath)
		if err != nil {
			t.Fatalf("Failed to read %s: %v", indexPath, err)
		}

		// Check FILES object has content
		filesRegex := regexp.MustCompile(`const FILES = \{([^}]*)\}`)
		if !filesRegex.Match(content) {
			t.Error("setup-safe-inputs index.js should have FILES object")
		}

		t.Log("setup-safe-inputs index.js has embedded FILES object")
	})

	t.Run("SafeOutputsActionPackagesFiles", func(t *testing.T) {
		repoRoot := getRepoRoot(t)
		jsDir := filepath.Join(repoRoot, "actions", "setup-safe-outputs", "js")
		
		// Check if directory exists
		if _, err := os.Stat(jsDir); os.IsNotExist(err) {
			t.Skip("setup-safe-outputs js/ directory not built yet (run 'make actions-build')")
		}

		// Count .cjs files
		entries, err := os.ReadDir(jsDir)
		if err != nil {
			t.Fatalf("Failed to read %s: %v", jsDir, err)
		}

		cjsCount := 0
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".cjs") {
				cjsCount++
			}
		}

		if cjsCount == 0 {
			t.Error("setup-safe-outputs action should package .cjs files but found none")
		}

		t.Logf("setup-safe-outputs action packages %d .cjs files", cjsCount)
	})
}

// getScriptsUsedInLockFiles returns a list of scripts that are called with
// `const { main } = require()` in lock files
func getScriptsUsedInLockFiles(t *testing.T) []string {
	t.Helper()

	// Find repository root
	repoRoot := getRepoRoot(t)
	workflowsDir := filepath.Join(repoRoot, ".github", "workflows")
	entries, err := os.ReadDir(workflowsDir)
	if err != nil {
		t.Fatalf("Failed to read workflows directory: %v", err)
	}

	usedScripts := make(map[string]bool)
	requireRegex := regexp.MustCompile(`const { main } = require\('/tmp/gh-aw/actions/([^']+\.cjs)'\)`)

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".lock.yml") {
			continue
		}

		lockPath := filepath.Join(workflowsDir, entry.Name())
		content, err := os.ReadFile(lockPath)
		if err != nil {
			t.Logf("Warning: failed to read %s: %v", lockPath, err)
			continue
		}

		matches := requireRegex.FindAllStringSubmatch(string(content), -1)
		for _, match := range matches {
			if len(match) > 1 {
				usedScripts[match[1]] = true
			}
		}
	}

	// Convert map to sorted slice
	result := make([]string, 0, len(usedScripts))
	for script := range usedScripts {
		result = append(result, script)
	}
	
	// Simple sort
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i] > result[j] {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result
}

// hasMainExport checks if a JavaScript file exports a main function
func hasMainExport(t *testing.T, filepath string) bool {
	t.Helper()

	content, err := os.ReadFile(filepath)
	if err != nil {
		t.Fatalf("Failed to read %s: %v", filepath, err)
		return false
	}

	// Check for main function export patterns
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?m)^async function main`),
		regexp.MustCompile(`(?m)^function main`),
		regexp.MustCompile(`module\.exports.*main`),
		regexp.MustCompile(`exports\.main`),
	}

	for _, pattern := range patterns {
		if pattern.Match(content) {
			return true
		}
	}

	return false
}

// getRepoRoot is a helper that calls findRepoRoot and handles errors
func getRepoRoot(t *testing.T) string {
	t.Helper()
	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("Failed to find repository root: %v", err)
	}
	return root
}
