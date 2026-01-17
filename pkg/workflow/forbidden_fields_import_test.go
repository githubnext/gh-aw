package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestForbiddenFieldsImportRejection tests that forbidden fields in shared workflows are rejected during compilation
func TestForbiddenFieldsImportRejection(t *testing.T) {
	// Use the SharedWorkflowForbiddenFields constant and create YAML examples for each
	forbiddenFieldYAML := map[string]string{
		"on":              `on: issues`,
		"command":         `command: /help`,
		"concurrency":     `concurrency: production`,
		"container":       `container: node:lts`,
		"env":             `env: {NODE_ENV: production}`,
		"environment":     `environment: staging`,
		"features":        `features: {test: true}`,
		"github-token":    `github-token: ${{ secrets.TOKEN }}`,
		"if":              `if: success()`,
		"name":            `name: Test Workflow`,
		"roles":           `roles: ["admin"]`,
		"run-name":        `run-name: Test Run`,
		"runs-on":         `runs-on: ubuntu-latest`,
		"sandbox":         `sandbox: {enabled: true}`,
		"source":          `source: owner/repo`,
		"strict":          `strict: true`,
		"timeout-minutes": `timeout-minutes: 30`,
		"timeout_minutes": `timeout_minutes: 30`,
		"tracker-id":      `tracker-id: "12345"`,
	}

	for _, field := range constants.SharedWorkflowForbiddenFields {
		yaml, ok := forbiddenFieldYAML[field]
		if !ok {
			t.Fatalf("Missing YAML example for forbidden field: %s. Please add to forbiddenFieldYAML map.", field)
		}

		t.Run("reject_import_"+field, func(t *testing.T) {
			tempDir := testutil.TempDir(t, "test-forbidden-"+field+"-*")
			workflowsDir := filepath.Join(tempDir, ".github", "workflows")
			require.NoError(t, os.MkdirAll(workflowsDir, 0755))

			// Create shared workflow with forbidden field
			sharedContent := `---
` + yaml + `
tools:
  bash: true
---

# Shared Workflow

This workflow has a forbidden field.
`
			sharedPath := filepath.Join(workflowsDir, "shared.md")
			require.NoError(t, os.WriteFile(sharedPath, []byte(sharedContent), 0644))

			// Create main workflow that imports the shared workflow
			mainContent := `---
on: issues
imports:
  - ./shared.md
---

# Main Workflow

This workflow imports a shared workflow with forbidden field.
`
			mainPath := filepath.Join(workflowsDir, "main.md")
			require.NoError(t, os.WriteFile(mainPath, []byte(mainContent), 0644))

			// Try to compile - should fail because shared workflow has forbidden field
			compiler := NewCompiler(false, tempDir, "test")
			err := compiler.CompileWorkflow(mainPath)

			// Should get error about forbidden field
			require.Error(t, err, "Expected error for forbidden field '%s'", field)
			assert.Contains(t, err.Error(), "cannot be used in shared workflows", 
				"Error should mention forbidden field, got: %v", err)
		})
	}
}

// TestAllowedFieldsImportSuccess tests that allowed fields in shared workflows are successfully imported
func TestAllowedFieldsImportSuccess(t *testing.T) {
	allowedFields := map[string]string{
		"tools":          `tools: {bash: true}`,
		"engine":         `engine: copilot`,
		"network":        `network: {allowed: [defaults]}`,
		"mcp-servers":    `mcp-servers: {}`,
		"permissions":    `permissions: read-all`,
		"runtimes":       `runtimes: {node: {version: "20"}}`,
		"safe-outputs":   `safe-outputs: {}`,
		"safe-inputs":    `safe-inputs: {}`,
		"services":       `services: {}`,
		"steps":          `steps: []`,
		"secret-masking": `secret-masking: true`,
		"jobs": `jobs:
  test-job:
    runs-on: ubuntu-latest
    steps:
      - run: echo test`,
		"description": `description: "Test shared workflow"`,
		"metadata":    `metadata: {}`,
		"inputs": `inputs:
  test_input:
    description: "Test input"
    type: string`,
		"bots":       `bots: ["copilot", "dependabot"]`,
		"post-steps": `post-steps: [{run: echo cleanup}]`,
		"labels":     `labels: ["automation", "testing"]`,
		"cache": `cache:
  key: "cache-key"
  path: "node_modules"`,
	}

	for field, yaml := range allowedFields {
		t.Run("allow_import_"+field, func(t *testing.T) {
			tempDir := testutil.TempDir(t, "test-allowed-"+field+"-*")
			workflowsDir := filepath.Join(tempDir, ".github", "workflows")
			require.NoError(t, os.MkdirAll(workflowsDir, 0755))

			// Create shared workflow with allowed field
			sharedContent := `---
` + yaml + `
---

# Shared Workflow

This workflow has an allowed field: ` + field + `
`
			sharedPath := filepath.Join(workflowsDir, "shared.md")
			require.NoError(t, os.WriteFile(sharedPath, []byte(sharedContent), 0644))

			// Create main workflow that imports the shared workflow
			mainContent := `---
on: issues
imports:
  - ./shared.md
---

# Main Workflow

This workflow imports a shared workflow with allowed field.
`
			mainPath := filepath.Join(workflowsDir, "main.md")
			require.NoError(t, os.WriteFile(mainPath, []byte(mainContent), 0644))

			// Compile - should succeed because shared workflow has allowed field
			compiler := NewCompiler(false, tempDir, "test")
			err := compiler.CompileWorkflow(mainPath)

			// Should NOT get error about forbidden field
			if err != nil && strings.Contains(err.Error(), "cannot be used in shared workflows") {
				t.Errorf("Field '%s' should be allowed in shared workflows, got error: %v", field, err)
			}
		})
	}
}

// TestImportsFieldAllowedInSharedWorkflows tests that the "imports" field is allowed in shared workflows
// and that nested imports work correctly
func TestImportsFieldAllowedInSharedWorkflows(t *testing.T) {
tempDir := testutil.TempDir(t, "test-allowed-imports-*")
workflowsDir := filepath.Join(tempDir, ".github", "workflows")
require.NoError(t, os.MkdirAll(workflowsDir, 0755))

// Create a base shared workflow (level 2)
baseSharedContent := `---
tools:
  bash: true
labels: ["base"]
---

# Base Shared Workflow

This is the base shared workflow.
`
baseSharedPath := filepath.Join(workflowsDir, "base.md")
require.NoError(t, os.WriteFile(baseSharedPath, []byte(baseSharedContent), 0644))

// Create intermediate shared workflow with "imports" field (level 1)
intermediateSharedContent := `---
imports:
  - ./base.md
tools:
  curl: true
labels: ["intermediate"]
---

# Intermediate Shared Workflow

This shared workflow imports another shared workflow (nested imports).
`
intermediateSharedPath := filepath.Join(workflowsDir, "intermediate.md")
require.NoError(t, os.WriteFile(intermediateSharedPath, []byte(intermediateSharedContent), 0644))

// Create main workflow that imports the intermediate shared workflow
mainContent := `---
on: issues
imports:
  - ./intermediate.md
---

# Main Workflow

This workflow imports a shared workflow that itself has imports (nested).
`
mainPath := filepath.Join(workflowsDir, "main.md")
require.NoError(t, os.WriteFile(mainPath, []byte(mainContent), 0644))

// Compile - should succeed because shared workflows can have imports (nested imports are supported)
compiler := NewCompiler(false, tempDir, "test")
err := compiler.CompileWorkflow(mainPath)

// Should NOT get error about forbidden field
if err != nil && strings.Contains(err.Error(), "cannot be used in shared workflows") {
t.Errorf("Field 'imports' should be allowed in shared workflows, got error: %v", err)
}
}
