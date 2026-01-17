package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestForbiddenFieldsImportRejection tests that forbidden fields in shared workflows are rejected during compilation
func TestForbiddenFieldsImportRejection(t *testing.T) {
	forbiddenFields := map[string]string{
		"on":              `on: issues`,
		"cache":           `cache: npm`,
		"command":         `command: /help`,
		"concurrency":     `concurrency: production`,
		"container":       `container: node:lts`,
		"env":             `env: {NODE_ENV: production}`,
		"environment":     `environment: staging`,
		"features":        `features: {test: true}`,
		"github-token":    `github-token: ${{ secrets.TOKEN }}`,
		"if":              `if: success()`,
		// Note: "imports" is skipped because it triggers import file resolution before field validation
		"labels":          `labels: ["bug"]`,
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

	for field, yaml := range forbiddenFields {
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

// TestImportsFieldForbiddenInSharedWorkflows tests that the "imports" field is forbidden in shared workflows
// This is tested separately because import resolution happens before field validation
func TestImportsFieldForbiddenInSharedWorkflows(t *testing.T) {
	tempDir := testutil.TempDir(t, "test-forbidden-imports-*")
	workflowsDir := filepath.Join(tempDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755))

	// Create a valid shared workflow that the forbidden shared workflow will try to import
	otherSharedContent := `---
tools:
  bash: true
---

# Other Shared Workflow
`
	otherSharedPath := filepath.Join(workflowsDir, "other.md")
	require.NoError(t, os.WriteFile(otherSharedPath, []byte(otherSharedContent), 0644))

	// Create shared workflow with "imports" field (forbidden)
	sharedContent := `---
imports:
  - ./other.md
tools:
  bash: true
---

# Shared Workflow

This workflow has a forbidden imports field.
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

This workflow imports a shared workflow with forbidden imports field.
`
	mainPath := filepath.Join(workflowsDir, "main.md")
	require.NoError(t, os.WriteFile(mainPath, []byte(mainContent), 0644))

	// Try to compile - should fail because shared workflow has forbidden "imports" field
	compiler := NewCompiler(false, tempDir, "test")
	err := compiler.CompileWorkflow(mainPath)

	// Should get error about forbidden field
	require.Error(t, err, "Expected error for forbidden field 'imports'")
	assert.Contains(t, err.Error(), "cannot be used in shared workflows",
		"Error should mention forbidden field, got: %v", err)
}
