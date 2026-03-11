//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuiltinEngineMarkdownFiles verifies that every built-in engine has a corresponding
// .md file registered in the parser's builtin virtual FS and that the file's frontmatter
// parses as a valid shared agentic workflow (no on: field, engine.id matches engine id).
func TestBuiltinEngineMarkdownFiles(t *testing.T) {
	// Initialise catalog so builtins are registered.
	catalog := NewEngineCatalog(NewEngineRegistry())
	require.NotNil(t, catalog, "engine catalog should be created")

	builtinEngineIDs := []string{"claude", "codex", "copilot", "gemini"}

	for _, id := range builtinEngineIDs {
		t.Run(id, func(t *testing.T) {
			path := builtinEnginePath(id)

			// The file must be registered as a builtin virtual file.
			assert.True(t, parser.BuiltinVirtualFileExists(path),
				"builtin virtual file should exist for engine %s at path %s", id, path)

			// Read via the parser's ReadFile — exercises the virtual FS.
			content, err := parser.ReadFile(path)
			require.NoError(t, err, "should read builtin engine file for %s", id)
			require.NotEmpty(t, content, "builtin engine file for %s should not be empty", id)

			// Parse the file as a shared agentic workflow.
			result, parseErr := parser.ExtractFrontmatterFromContent(string(content))
			require.NoError(t, parseErr, "engine %s .md frontmatter should parse without error", id)
			require.NotEmpty(t, result.Frontmatter, "engine %s .md should have frontmatter", id)

			// Must have an engine: key whose id matches the engine id.
			engineField, hasEngine := result.Frontmatter["engine"]
			require.True(t, hasEngine, "engine %s .md frontmatter must contain an engine: key", id)
			engineObj, isMap := engineField.(map[string]any)
			require.True(t, isMap, "engine %s .md engine: field must be an object, got %T", id, engineField)
			assert.Equal(t, id, engineObj["id"],
				"engine %s .md engine.id should match the engine id", id)

			// Must NOT have on: field (shared workflow, not a main workflow).
			_, hasOnField := result.Frontmatter["on"]
			assert.False(t, hasOnField,
				"engine %s .md should be a shared workflow (no on: field)", id)
		})
	}
}

// TestBuiltinEngineStringFormInjection verifies that when a workflow uses the short/string
// form "engine: <id>", the compiler transparently injects the builtin .md as an import and
// produces a valid lock file with the correct engine ID.
func TestBuiltinEngineStringFormInjection(t *testing.T) {
	tests := []struct {
		engineID   string
		engineStep string // distinctive step name in the lock file
	}{
		{"copilot", `GH_AW_INFO_ENGINE_ID: "copilot"`},
		{"codex", `GH_AW_INFO_ENGINE_ID: "codex"`},
		{"claude", `GH_AW_INFO_ENGINE_ID: "claude"`},
		{"gemini", `GH_AW_INFO_ENGINE_ID: "gemini"`},
	}

	for _, tt := range tests {
		t.Run(tt.engineID, func(t *testing.T) {
			tmpDir := testutil.TempDir(t, "test-engine-injection-*")
			workflowsDir := filepath.Join(tmpDir, constants.GetWorkflowDir())
			require.NoError(t, os.MkdirAll(workflowsDir, 0755))

			md := "---\n" +
				"name: Test Engine Injection\n" +
				"on:\n" +
				"  issues:\n" +
				"    types: [opened]\n" +
				"permissions:\n" +
				"  contents: read\n" +
				"  issues: read\n" +
				"engine: " + tt.engineID + "\n" +
				"---\n\n" +
				"# Task\n\nDo something.\n"

			mainFile := filepath.Join(workflowsDir, "test-engine-injection.md")
			require.NoError(t, os.WriteFile(mainFile, []byte(md), 0644))

			compiler := NewCompiler()
			err := compiler.CompileWorkflow(mainFile)
			require.NoError(t, err, "compilation should succeed for engine %s (string form)", tt.engineID)

			lockFile := filepath.Join(workflowsDir, "test-engine-injection.lock.yml")
			lockContent, err := os.ReadFile(lockFile)
			require.NoError(t, err, "lock file should be created for engine %s", tt.engineID)

			assert.Contains(t, string(lockContent), tt.engineStep,
				"lock file for engine %s should contain %q", tt.engineID, tt.engineStep)
		})
	}
}

// TestBuiltinEngineStringFormInjection_CLIOverrideNotInjected verifies that when a CLI
// --engine override is active, the builtin .md injection is skipped and the override engine
// is used instead.
func TestBuiltinEngineStringFormInjection_CLIOverrideNotInjected(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-engine-override-*")
	workflowsDir := filepath.Join(tmpDir, constants.GetWorkflowDir())
	require.NoError(t, os.MkdirAll(workflowsDir, 0755))

	md := "---\n" +
		"name: Test Engine Override\n" +
		"on:\n" +
		"  issues:\n" +
		"    types: [opened]\n" +
		"permissions:\n" +
		"  contents: read\n" +
		"  issues: read\n" +
		"engine: copilot\n" +
		"---\n\n" +
		"# Task\n\nDo something.\n"

	mainFile := filepath.Join(workflowsDir, "test-engine-override.md")
	require.NoError(t, os.WriteFile(mainFile, []byte(md), 0644))

	// Use --engine codex to override the markdown's "engine: copilot".
	compiler := NewCompiler(WithEngineOverride("codex"))
	err := compiler.CompileWorkflow(mainFile)
	require.NoError(t, err, "compilation should succeed with --engine override")

	lockFile := filepath.Join(workflowsDir, "test-engine-override.lock.yml")
	lockContent, err := os.ReadFile(lockFile)
	require.NoError(t, err, "lock file should be created")

	assert.Contains(t, string(lockContent), `GH_AW_INFO_ENGINE_ID: "codex"`,
		"lock file should use the overridden engine, not the markdown's copilot")
	assert.NotContains(t, string(lockContent), `GH_AW_INFO_ENGINE_ID: "copilot"`,
		"lock file should not contain the overridden copilot engine ID")
}
