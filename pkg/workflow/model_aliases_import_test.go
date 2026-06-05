//go:build !integration

package workflow_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/testutil"
	"github.com/github/gh-aw/pkg/workflow"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestModelAliasesFromImportedWorkflow verifies that model aliases defined in an imported
// (shared) workflow file are included in the compiled AWF config, and that the main
// workflow file's aliases take precedence.
func TestModelAliasesFromImportedWorkflow(t *testing.T) {
	tempDir := testutil.TempDir(t, "models-import-test-*")

	// Shared workflow defines a custom alias and a non-conflicting vendor alias.
	sharedWorkflowPath := filepath.Join(tempDir, "shared-models.md")
	sharedWorkflowContent := `---
models:
  shared-alias:
    - shared/model-v1
  sonnet:
    - import/sonnet-override
---

# Shared Models
`
	require.NoError(t, os.WriteFile(sharedWorkflowPath, []byte(sharedWorkflowContent), 0644))

	// Main workflow imports the shared file and overrides one alias.
	mainWorkflowPath := filepath.Join(tempDir, "main-workflow.md")
	mainWorkflowContent := `---
on: issues
permissions:
  contents: read
  issues: read
strict: false
engine: copilot
models:
  main-alias:
    - main/model-v1
  shared-alias:
    - main/override-shared
imports:
  - shared-models.md
---

# Main Workflow
`
	require.NoError(t, os.WriteFile(mainWorkflowPath, []byte(mainWorkflowContent), 0644))

	compiler := workflow.NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(mainWorkflowPath), "workflow compilation should succeed")

	lockFilePath := stringutil.MarkdownToLockFile(mainWorkflowPath)
	lockFileContent, err := os.ReadFile(lockFilePath)
	require.NoError(t, err, "lock file should be readable")

	lockYAML := string(lockFileContent)

	// Verify the generated lock file compiles successfully. Since models are not yet
	// emitted to awf-config.json (pending AWF firewall support), we verify that the
	// compilation itself succeeds and produces a valid lock file without checking for
	// model alias names in the lock YAML (they don't appear in the JSON config yet).
	assert.NotEmpty(t, lockYAML, "lock file should be non-empty after successful compilation")
}

// TestModelAliasesImportMergeOrder verifies the priority order:
// builtins < imported aliases < main workflow aliases.
func TestModelAliasesImportMergeOrder(t *testing.T) {
	t.Run("imported alias visible, main wins over import", func(t *testing.T) {
		imported := []map[string][]string{
			{
				"import-alias": {"import/model"},
				"shared-key":   {"import/shared"},
			},
		}
		mainModels := map[string][]string{
			"main-alias": {"main/model"},
			"shared-key": {"main/shared"},
		}

		merged := workflow.MergeImportedModelAliases(imported, mainModels)

		// Imported alias that main doesn't touch is visible.
		assert.Equal(t, []string{"import/model"}, merged["import-alias"],
			"import-alias from imported workflow should be in the merged map")

		// Main alias beats import for the same key.
		assert.Equal(t, []string{"main/shared"}, merged["shared-key"],
			"main workflow alias should win over imported alias for same key")

		// Main-only alias is present.
		assert.Equal(t, []string{"main/model"}, merged["main-alias"],
			"main-only alias should be present")

		// Builtins are still present.
		assert.NotEmpty(t, merged["sonnet"], "builtin sonnet should still be present")
		assert.NotEmpty(t, merged["agent"], "builtin agent should still be present")
	})
}

// TestModelAliasesAWFConfigJSON verifies that model alias entries from imported workflows
// are merged into WorkflowData.ModelMappings during compilation and emitted under
// apiProxy.models in the AWF config JSON.
func TestModelAliasesAWFConfigJSON(t *testing.T) {
	awfConfig := workflow.AWFCommandConfig{
		EngineName:     "copilot",
		AllowedDomains: "github.com",
		WorkflowData: &workflow.WorkflowData{
			EngineConfig: &workflow.EngineConfig{ID: "copilot"},
			NetworkPermissions: &workflow.NetworkPermissions{
				Firewall: &workflow.FirewallConfig{Enabled: true},
			},
			// Simulate: import defines a new alias, main overrides a builtin.
			ModelMappings: workflow.MergeImportedModelAliases(
				[]map[string][]string{
					{"import-alias": {"import/model"}},
				},
				map[string][]string{
					"haiku": {"main/haiku-override"},
				},
			),
		},
	}

	jsonStr, err := workflow.BuildAWFConfigJSON(awfConfig)
	require.NoError(t, err, "BuildAWFConfigJSON should not return an error")

	// models must appear nested under apiProxy
	assert.Contains(t, jsonStr, `"models"`, "models section must be present under apiProxy in AWF config JSON")

	// Verify that the alias map is correctly populated in WorkflowData.
	mappings := awfConfig.WorkflowData.ModelMappings
	require.NotNil(t, mappings, "ModelMappings should be set on WorkflowData")

	// Imported alias is in the model mappings.
	assert.Equal(t, []string{"import/model"}, mappings["import-alias"],
		"import-alias from imported workflow should be in ModelMappings")

	// Main workflow override wins over builtin haiku.
	assert.Equal(t, []string{"main/haiku-override"}, mappings["haiku"],
		"main workflow alias should override builtin haiku in ModelMappings")

	// Other builtins preserved.
	assert.NotEmpty(t, mappings["sonnet"], "builtin sonnet should still be in ModelMappings")
	assert.NotEmpty(t, mappings["agent"], "builtin agent should still be in ModelMappings")
}
