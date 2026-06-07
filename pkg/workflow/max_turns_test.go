//go:build integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"

	"github.com/github/gh-aw/pkg/testutil"
)

func TestMaxTurnsCompilation(t *testing.T) {
	tests := []struct {
		name             string
		content          string
		expectedMaxRuns  string
		shouldInclude    bool
	}{
		{
			name: "workflow with max-turns",
			content: `---
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  pull-requests: read
engine:
  id: claude
  max-turns: 3
tools:
  github:
    allowed: [issue_read]
---

# Test Max Turns

This workflow tests the max-turns feature.`,
			expectedMaxRuns: "3",
			shouldInclude:   true,
		},
		{
			name: "workflow without max-turns",
			content: `---
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: claude
tools:
  github:
    allowed: [issue_read]
---

# Test Without Max Turns

This workflow should not include max-turns.`,
			expectedMaxRuns: "",
			shouldInclude:   false,
		},
		{
			name: "workflow with max-turns and timeout",
			content: `---
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  pull-requests: read
engine:
  id: claude
  max-turns: 10
timeout-minutes: 15
strict: false
tools:
  github:
    allowed: [issue_read]
---

# Test Max Turns and Timeout

This workflow tests max-turns with timeout.`,
			expectedMaxRuns: "10",
			shouldInclude:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for the test
			tmpDir := testutil.TempDir(t, "max-turns-test")

			// Create the test workflow file
			testFile := filepath.Join(tmpDir, "test-workflow.md")
			if err := os.WriteFile(testFile, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			// Compile the workflow
			compiler := NewCompiler()
			if err := compiler.CompileWorkflow(testFile); err != nil {
				t.Fatalf("Failed to compile workflow: %v", err)
			}

			// Read the generated lock file
			lockFile := stringutil.MarkdownToLockFile(testFile)
			lockContent, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatalf("Failed to read lock file: %v", err)
			}

			lockContentStr := string(lockContent)

			if tt.shouldInclude {
				// Verify max-turns is emitted through AWF apiProxy.maxRuns
				expectedMaxRuns := `"maxRuns":` + tt.expectedMaxRuns
				if !strings.Contains(lockContentStr, expectedMaxRuns) {
					t.Errorf("Expected max-turns to be emitted via apiProxy.maxRuns. Expected: %s\nActual content:\n%s", expectedMaxRuns, lockContentStr)
				}

				// Verify GH_AW_MAX_TURNS environment variable is set
				expectedEnvVar := "GH_AW_MAX_TURNS: " + tt.expectedMaxRuns
				if !strings.Contains(lockContentStr, expectedEnvVar) {
					t.Errorf("Expected GH_AW_MAX_TURNS environment variable to be set. Expected: %s\nActual content:\n%s", expectedEnvVar, lockContentStr)
				}

				// Verify it's in the correct context (under the Claude CLI execution)
				if !strings.Contains(lockContentStr, "claude --print") {
					t.Error("Expected to find claude command in generated workflow")
				}

				if strings.Contains(lockContentStr, "--max-turns") {
					t.Error("Did not expect --max-turns in Claude CLI command")
				}
			} else {
				// Verify --max-turns is NOT included when not specified
				if strings.Contains(lockContentStr, "--max-turns") {
					t.Error("Expected --max-turns NOT to be included when not specified in frontmatter")
				}

				// Verify GH_AW_MAX_TURNS falls back to the vars expression when not specified
				if !strings.Contains(lockContentStr, "GH_AW_MAX_TURNS: ${{ vars.GH_AW_DEFAULT_MAX_TURNS") {
					t.Errorf("Expected GH_AW_MAX_TURNS to include vars fallback expression when max-turns not specified in frontmatter.\nLock content:\n%s", lockContentStr)
				}
			}
		})
	}
}

func TestMaxTurnsValidation(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectError bool
	}{
		{
			name: "valid integer max-turns",
			content: `---
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  pull-requests: read
engine:
  id: claude
  max-turns: 5
---

# Valid Max Turns`,
			expectError: false,
		},
		{
			name: "invalid string max-turns",
			content: `---
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: claude
max-turns: "5"
---

# Invalid String Max Turns`,
			expectError: true,
		},
		{
			name: "zero max-turns",
			content: `---
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  pull-requests: read
engine:
  id: claude
  max-turns: 0
---

# Zero Max Turns`,
			expectError: false, // Zero should be valid (might mean unlimited)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for the test
			tmpDir := testutil.TempDir(t, "max-turns-validation-test")

			// Create the test workflow file
			testFile := filepath.Join(tmpDir, "test-workflow.md")
			if err := os.WriteFile(testFile, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			// Compile the workflow
			compiler := NewCompiler()
			err := compiler.CompileWorkflow(testFile)

			if tt.expectError && err == nil {
				t.Error("Expected compilation to fail but it succeeded")
			} else if !tt.expectError && err != nil {
				t.Errorf("Expected compilation to succeed but it failed: %v", err)
			}
		})
	}
}

func TestTopLevelMaxTurnsCompilationForCodex(t *testing.T) {
	tmpDir := testutil.TempDir(t, "top-level-max-turns-codex")

	testContent := `---
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: codex
max-turns: "${{ inputs.max-turns }}"
---

# Test Top-Level Max Turns
`

	testFile := filepath.Join(tmpDir, "test-workflow.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	lockFile := stringutil.MarkdownToLockFile(testFile)
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockContentStr := string(lockContent)
	if !strings.Contains(lockContentStr, `GH_AW_MAX_TURNS: "${{ inputs.max-turns }}"`) &&
		!strings.Contains(lockContentStr, "GH_AW_MAX_TURNS: ${{ inputs.max-turns }}") {
		t.Errorf("Expected top-level max-turns to compile into GH_AW_MAX_TURNS.\nLock file content:\n%s", lockContentStr)
	}
}

func TestMaxTurnsFromSharedImport(t *testing.T) {
	// This test verifies that max-turns is correctly propagated when
	// the engine config is sourced from a shared import rather than defined inline.
	// The bug was that max-turns was silently dropped because it was serialized as
	// JSON (int -> float64) but only int/uint64/string types were handled.

	// Create a temporary directory for the test
	tmpDir := testutil.TempDir(t, "max-turns-import-test")

	// Create the shared import file with top-level max-turns
	sharedContent := `---
max-turns: 100
engine:
  id: claude
permissions:
  contents: read
  issues: read
  pull-requests: read
---
`
	sharedDir := filepath.Join(tmpDir, "shared")
	if err := os.MkdirAll(sharedDir, 0755); err != nil {
		t.Fatal(err)
	}
	sharedFile := filepath.Join(sharedDir, "common.md")
	if err := os.WriteFile(sharedFile, []byte(sharedContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create the main workflow that imports the shared config
	mainContent := `---
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  pull-requests: read
imports:
  - shared/common.md
tools:
  github:
    allowed: [issue_read]
---

# Test Max Turns From Shared Import

This workflow imports max-turns from a shared import.
`
	mainFile := filepath.Join(tmpDir, "test-workflow.md")
	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Compile the workflow
	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(mainFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockFile := stringutil.MarkdownToLockFile(mainFile)
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockContentStr := string(lockContent)

	// Verify max-turns is emitted via apiProxy.maxRuns in AWF config JSON.
	if !strings.Contains(lockContentStr, `"maxRuns":100`) {
		t.Errorf("Expected \"maxRuns\":100 in compiled output when max-turns is set in shared import.\nLock file content:\n%s", lockContentStr)
	}
	if strings.Contains(lockContentStr, "--max-turns") {
		t.Errorf("Did not expect --max-turns in compiled output when max-turns is set in shared import.\nLock file content:\n%s", lockContentStr)
	}

	// Verify GH_AW_MAX_TURNS env var is set
	if !strings.Contains(lockContentStr, "GH_AW_MAX_TURNS: 100") {
		t.Errorf("Expected GH_AW_MAX_TURNS: 100 in compiled output.\nLock file content:\n%s", lockContentStr)
	}
}
