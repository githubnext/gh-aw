//go:build integration

package workflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGradersWorkflowIntegration(t *testing.T) {
	file := filepath.Join(t.TempDir(), "workflow.md")
	require.NoError(t, os.WriteFile(file, []byte(`---
on: workflow_dispatch
engine: copilot
strict: false
graders:
  retries:
    threshold: 2
experiments:
  prompt:
    variants: [control, candidate]
    metric: graders.loops.value
---
Test.
`), 0o600))
	compiler := NewCompiler(WithVersion("dev"))
	require.NoError(t, compiler.CompileWorkflow(file))
	content, err := os.ReadFile(stringutil.MarkdownToLockFile(file))
	require.NoError(t, err)
	conclusion := extractJobSection(string(content), "conclusion")
	assert.Contains(t, conclusion, "generate_usage_artifacts.cjs")
	assert.NotContains(t, extractJobSection(string(content), "agent"), "GRADER")
	var manifest graderManifest
	require.NoError(t, json.Unmarshal(graderEnvironmentPayload(conclusion, "GH_AW_GRADER_MANIFEST"), &manifest))
	assert.Equal(t, 2, manifest.Version)
	assert.NotContains(t, string(content), "GH_AW_GRADER_EXEC")
}
