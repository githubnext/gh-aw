//go:build integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOnNeedsCompilesAndWiresActivationDependencies(t *testing.T) {
	tmpDir := testutil.TempDir(t, "on-needs-integration")
	compiler := NewCompiler()

	workflowContent := `---
on:
  workflow_dispatch:
  needs: [secrets_fetcher]
  github-app:
    client-id: ${{ needs.secrets_fetcher.outputs.app_id }}
    private-key: ${{ needs.secrets_fetcher.outputs.private_key }}
engine: copilot
jobs:
  secrets_fetcher:
    runs-on: ubuntu-latest
    outputs:
      app_id: ${{ steps.fetch.outputs.app_id }}
      private_key: ${{ steps.fetch.outputs.private_key }}
    steps:
      - id: fetch
        run: |
          echo "app_id=123" >> "$GITHUB_OUTPUT"
          echo "private_key=key" >> "$GITHUB_OUTPUT"
---
Run with on.needs
`

	workflowFile := filepath.Join(tmpDir, "test-on-needs.md")
	require.NoError(t, os.WriteFile(workflowFile, []byte(workflowContent), 0644), "should write test workflow")

	require.NoError(t, compiler.CompileWorkflow(workflowFile), "workflow should compile with on.needs and on.github-app needs expression")

	lockFile := filepath.Join(tmpDir, "test-on-needs.lock.yml")
	lockBytes, err := os.ReadFile(lockFile)
	require.NoError(t, err, "should read compiled lock file")

	var lock map[string]any
	require.NoError(t, yaml.Unmarshal(lockBytes, &lock), "compiled lock file should be valid YAML")

	// Verify on.needs is NOT emitted into the compiled on: section
	onSection, ok := lock["on"].(map[string]any)
	require.True(t, ok, "compiled workflow should contain on section")
	assert.NotContains(t, onSection, "needs", "on.needs must not appear as a key in the compiled on: section")

	// Verify on.needs is not in the raw compiled text (even as a comment)
	lockContent := string(lockBytes)
	assert.False(t, containsNeedsInOnSection(lockContent), "on.needs must not appear in the compiled on: section text")

	jobs, ok := lock["jobs"].(map[string]any)
	require.True(t, ok, "compiled workflow should contain jobs map")

	preActivation, ok := jobs["pre_activation"].(map[string]any)
	require.True(t, ok, "compiled workflow should contain pre_activation job")
	assert.Contains(t, preActivation["needs"], "secrets_fetcher", "pre_activation should depend on on.needs job")

	activation, ok := jobs["activation"].(map[string]any)
	require.True(t, ok, "compiled workflow should contain activation job")
	assert.Contains(t, activation["needs"], "secrets_fetcher", "activation should depend on on.needs job")
}

// containsNeedsInOnSection checks whether the "needs:" key appears in the "on:" section
// of a compiled workflow YAML string, including a commented-out "# needs:" line.
func containsNeedsInOnSection(yamlContent string) bool {
	inOnSection := false
	onIndent := 0
	childIndent := -1
	for _, line := range strings.Split(yamlContent, "\n") {
		trimmed := strings.TrimLeft(line, " \t")
		// Track when we enter/leave the on: section
		if trimmed == "on:" || strings.HasPrefix(trimmed, `"on":`) {
			inOnSection = true
			onIndent = len(line) - len(trimmed)
			childIndent = -1
			continue
		}
		// Leave on: section when we hit a top-level key
		if inOnSection && len(line) > 0 && line[0] != ' ' && line[0] != '\t' && !strings.HasPrefix(trimmed, "#") {
			inOnSection = false
		}
		if !inOnSection {
			continue
		}
		indent := len(line) - len(trimmed)
		if indent <= onIndent {
			continue
		}
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			normalized := strings.TrimSpace(strings.TrimPrefix(trimmed, "#"))
			if childIndent != -1 && indent == childIndent && strings.HasPrefix(normalized, "needs:") {
				return true
			}
			continue
		}
		if childIndent == -1 {
			childIndent = indent
		}
		if indent != childIndent {
			continue
		}
		normalized := strings.TrimSpace(strings.TrimPrefix(trimmed, "#"))
		// Check for needs: inside on: section as a direct child key, even if it
		// has been commented out.
		if strings.HasPrefix(normalized, "needs:") {
			return true
		}
	}
	return false
}
