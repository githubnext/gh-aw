//go:build integration

package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
)

func TestMaxTurnsValidationWithSupportedEngines(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name: "max-turns with codex engine should succeed",
			content: `---
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  pull-requests: read
engine:
  id: codex
  max-turns: 5
---

# Test Workflow

This should succeed because AWF max-turns is supported across engines.`,
		},
		{
			name: "max-turns with claude engine should succeed",
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

# Test Workflow

This should succeed because AWF max-turns is supported across engines.`,
		},
		{
			name: "top-level max-turns expression with copilot engine should succeed",
			content: `---
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
max-turns: "${{ inputs.max-turns }}"
---

# Test Workflow

This should succeed because top-level max-turns supports expressions.`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for the test
			tmpDir := testutil.TempDir(t, "max-turns-validation-test")

			// Create a test workflow file
			testFile := filepath.Join(tmpDir, "test.md")
			if err := os.WriteFile(testFile, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			// Create a compiler instance
			compiler := NewCompiler()
			compiler.SetSkipValidation(false)

			// Try to compile the workflow
			err := compiler.CompileWorkflow(testFile)

			if err != nil {
				t.Errorf("Expected compilation to succeed but got error: %v", err)
			}
		})
	}
}

func TestEngineSupportsMaxTurns(t *testing.T) {
	tests := []struct {
		name            string
		engineID        string
		expectedSupport bool
	}{
		{
			name:            "claude engine supports max-turns",
			engineID:        "claude",
			expectedSupport: true,
		},
		{
			name:            "codex engine supports max-turns",
			engineID:        "codex",
			expectedSupport: true,
		},
		{
			name:            "copilot engine supports max-turns",
			engineID:        "copilot",
			expectedSupport: true,
		},
		{
			name:            "gemini engine supports max-turns",
			engineID:        "gemini",
			expectedSupport: true,
		},
		{
			name:            "opencode engine supports max-turns",
			engineID:        "opencode",
			expectedSupport: true,
		},
		{
			name:            "crush engine supports max-turns",
			engineID:        "crush",
			expectedSupport: true,
		},
		{
			name:            "pi engine supports max-turns",
			engineID:        "pi",
			expectedSupport: true,
		},
		{
			name:            "antigravity engine supports max-turns",
			engineID:        "antigravity",
			expectedSupport: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := GetGlobalEngineRegistry()
			engine, err := registry.GetEngine(tt.engineID)
			if err != nil {
				t.Fatalf("Failed to get engine '%s': %v", tt.engineID, err)
			}

			actualSupport := engine.GetCapabilities().MaxTurns
			if actualSupport != tt.expectedSupport {
				t.Errorf("Expected engine '%s' to have MaxTurns capability = %v, but got %v",
					tt.engineID, tt.expectedSupport, actualSupport)
			}
		})
	}
}
