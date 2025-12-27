package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestJSweepWorkflowConfiguration validates that the jsweep workflow is properly configured
// to process a single JavaScript file with TypeScript validation and prettier formatting.
func TestJSweepWorkflowConfiguration(t *testing.T) {
	// Read the jsweep.md file
	jsweepPath := filepath.Join("..", "..", ".github", "workflows", "jsweep.md")
	content, err := os.ReadFile(jsweepPath)
	if err != nil {
		t.Fatalf("Failed to read jsweep.md: %v", err)
	}

	mdContent := string(content)

	// Test 1: Verify the workflow processes one file, not three
	t.Run("ProcessesSingleFile", func(t *testing.T) {
		if !strings.Contains(mdContent, "one .cjs file per day") {
			t.Error("jsweep workflow should process one .cjs file per day")
		}
		if strings.Contains(mdContent, "three .cjs files per day") {
			t.Error("jsweep workflow should not process three files")
		}
		if !strings.Contains(mdContent, "Pick the **one file**") {
			t.Error("jsweep workflow should pick one file")
		}
		if strings.Contains(mdContent, "Pick the **three files**") {
			t.Error("jsweep workflow should not pick three files")
		}
	})

	// Test 2: Verify TypeScript validation is configured
	t.Run("TypeScriptValidation", func(t *testing.T) {
		if !strings.Contains(mdContent, "npm run typecheck") {
			t.Error("jsweep workflow should include TypeScript validation with 'npm run typecheck'")
		}
		if !strings.Contains(mdContent, "verify no type errors") {
			t.Error("jsweep workflow should verify no type errors")
		}
		if !strings.Contains(mdContent, "type safety") {
			t.Error("jsweep workflow should mention type safety")
		}
	})

	// Test 3: Verify prettier formatting is configured
	t.Run("PrettierFormatting", func(t *testing.T) {
		if !strings.Contains(mdContent, "npm run format:cjs") {
			t.Error("jsweep workflow should include prettier formatting with 'npm run format:cjs'")
		}
		if !strings.Contains(mdContent, "ensure consistent formatting") {
			t.Error("jsweep workflow should ensure consistent formatting")
		}
		if !strings.Contains(mdContent, "prettier") {
			t.Error("jsweep workflow should mention prettier")
		}
	})

	// Test 4: Verify the PR title format is correct for single file
	t.Run("PRTitleFormat", func(t *testing.T) {
		if !strings.Contains(mdContent, "Title: `[jsweep] Clean <filename>`") {
			t.Error("jsweep workflow should have PR title format for single file: [jsweep] Clean <filename>")
		}
		if strings.Contains(mdContent, "Clean <file1>, <file2>, <file3>") {
			t.Error("jsweep workflow should not have PR title format for three files")
		}
	})

	// Test 5: Verify the workflow runs tests
	t.Run("RunsTests", func(t *testing.T) {
		if !strings.Contains(mdContent, "npm run test:js") {
			t.Error("jsweep workflow should run JavaScript tests with 'npm run test:js'")
		}
		if !strings.Contains(mdContent, "verify all tests pass") {
			t.Error("jsweep workflow should verify all tests pass")
		}
	})

	// Test 6: Verify testing requirements
	t.Run("TestingRequirements", func(t *testing.T) {
		if !strings.Contains(mdContent, "Testing is NOT optional") {
			t.Error("jsweep workflow should specify that testing is not optional")
		}
		if !strings.Contains(mdContent, "the file must have comprehensive test coverage") {
			t.Error("jsweep workflow should require comprehensive test coverage for the file")
		}
		if strings.Contains(mdContent, "every file must have comprehensive test coverage") {
			t.Error("jsweep workflow should refer to 'the file' (singular) not 'every file'")
		}
	})

	// Test 7: Verify the workflow description
	t.Run("WorkflowDescription", func(t *testing.T) {
		if !strings.Contains(mdContent, "description: Daily JavaScript unbloater that cleans one .cjs file per day") {
			t.Error("jsweep workflow description should specify 'one .cjs file per day'")
		}
		if strings.Contains(mdContent, "description: Daily JavaScript unbloater that cleans three .cjs files per day") {
			t.Error("jsweep workflow description should not specify 'three .cjs files per day'")
		}
	})

	// Test 8: Verify the workflow has a valid lock file
	t.Run("HasValidLockFile", func(t *testing.T) {
		lockPath := filepath.Join("..", "..", ".github", "workflows", "jsweep.lock.yml")
		_, err := os.Stat(lockPath)
		if err != nil {
			t.Errorf("jsweep.lock.yml should exist and be accessible: %v", err)
		}
	})
}

// TestJSweepWorkflowLockFile validates that the compiled jsweep.lock.yml file
// contains the expected configuration for single file processing.
func TestJSweepWorkflowLockFile(t *testing.T) {
	// Read the jsweep.lock.yml file
	lockPath := filepath.Join("..", "..", ".github", "workflows", "jsweep.lock.yml")
	content, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read jsweep.lock.yml: %v", err)
	}

	lockContent := string(content)

	// Test 1: Verify the compiled workflow processes one file
	t.Run("CompiledProcessesSingleFile", func(t *testing.T) {
		if !strings.Contains(lockContent, "one .cjs file per day") {
			t.Error("Compiled jsweep workflow should process one .cjs file per day")
		}
		if strings.Contains(lockContent, "three .cjs files per day") {
			t.Error("Compiled jsweep workflow should not process three files")
		}
	})

	// Test 2: Verify TypeScript validation is in the compiled workflow
	t.Run("CompiledTypeScriptValidation", func(t *testing.T) {
		if !strings.Contains(lockContent, "npm run typecheck") {
			t.Error("Compiled jsweep workflow should include TypeScript validation")
		}
	})

	// Test 3: Verify prettier formatting is in the compiled workflow
	t.Run("CompiledPrettierFormatting", func(t *testing.T) {
		if !strings.Contains(lockContent, "npm run format:cjs") {
			t.Error("Compiled jsweep workflow should include prettier formatting")
		}
	})
}
