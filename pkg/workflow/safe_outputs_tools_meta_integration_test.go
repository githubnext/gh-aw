//go:build integration

package workflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestToolsMetaJSONContainsDescriptionSuffixes verifies that the compiled workflow YAML
// embeds a tools_meta.json with the correct description suffixes instead of the large
// inlined tools.json used in the old approach.
func TestToolsMetaJSONCompiledWorkflowEmbedsMeta(t *testing.T) {
	tmpDir := testutil.TempDir(t, "tools-meta-compiled-test")

	testContent := `---
on: push
name: Test Compiled Meta
engine: copilot
safe-outputs:
  create-issue:
    max: 2
    title-prefix: "[auto] "
    labels:
      - generated
  create-pull-request:
    max: 1
    allowed-repos:
      - org/other-repo
  missing-tool: {}
  noop: {}
---

Test workflow to verify tools_meta in compiled output.
`

	testFile := filepath.Join(tmpDir, "test-compiled-meta.md")
	require.NoError(t, os.WriteFile(testFile, []byte(testContent), 0644), "should write test file")

	compiler := NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(testFile), "compilation should succeed")

	lockFile := filepath.Join(tmpDir, "test-compiled-meta.lock.yml")
	yamlBytes, err := os.ReadFile(lockFile)
	require.NoError(t, err, "should read lock file")
	yamlStr := string(yamlBytes)

	// Structural checks: new strategy is in use
	assert.Contains(t, yamlStr, "tools_meta.json", "lock file should reference tools_meta.json")
	assert.Contains(t, yamlStr, "generate_safe_outputs_tools.cjs", "lock file should invoke JS generator")
	assert.NotContains(t, yamlStr, `cat > ${RUNNER_TEMP}/gh-aw/safeoutputs/tools.json`, "lock file should NOT inline tools.json")

	// tools_meta.json must contain create_issue description suffix with constraint text
	assert.Contains(t, yamlStr, "CONSTRAINTS:", "tools_meta should contain constraint text")
	assert.Contains(t, yamlStr, "Maximum 2 issue", "tools_meta should contain max constraint for create_issue")
	assert.Contains(t, yamlStr, `[auto]`, "tools_meta should contain title prefix for create_issue")
	assert.Contains(t, yamlStr, `generated`, "tools_meta should contain label for create_issue")

	// tools_meta.json must contain repo_params for create_pull_request (has allowed-repos)
	assert.Contains(t, yamlStr, `"repo_params"`, "tools_meta should contain repo_params section")

	meta := extractToolsMetaFromLockFile(t, yamlStr)

	// Verify tools without constraints do NOT get description suffixes
	_, hasMissingTool := meta.DescriptionSuffixes["missing_tool"]
	assert.False(t, hasMissingTool, "missing_tool should not have a description suffix")
	_, hasNoop := meta.DescriptionSuffixes["noop"]
	assert.False(t, hasNoop, "noop should not have a description suffix")
}

// TestToolsMetaJSONPushRepoMemory verifies that push_repo_memory appears in
// description_suffixes when repo-memory is configured.
func TestToolsMetaJSONEmptyWhenNoSafeOutputs(t *testing.T) {
	data := &WorkflowData{SafeOutputs: nil}

	metaJSON, err := generateToolsMetaJSON(data, ".github/workflows/test.md")
	require.NoError(t, err, "generateToolsMetaJSON should not error for nil safe-outputs")

	var meta ToolsMeta
	require.NoError(t, json.Unmarshal([]byte(metaJSON), &meta), "meta JSON should parse")

	assert.Empty(t, meta.DescriptionSuffixes, "description_suffixes should be empty when no safe-outputs")
	assert.Empty(t, meta.RepoParams, "repo_params should be empty when no safe-outputs")
	assert.Empty(t, meta.DynamicTools, "dynamic_tools should be empty when no safe-outputs")
}

func extractToolsMetaFromLockFile(t *testing.T, yamlStr string) ToolsMeta {
	t.Helper()

	// Find the randomized delimiter using regex (format: GH_AW_SAFE_OUTPUTS_TOOLS_META_<16hex>_EOF)
	delimRE := regexp.MustCompile(`GH_AW_SAFE_OUTPUTS_TOOLS_META_[0-9a-f]{16}_EOF`)
	delimMatch := delimRE.FindString(yamlStr)
	require.NotEmpty(t, delimMatch, "lock file should contain tools_meta heredoc delimiter")
	delimiter := delimMatch

	// Opening line is: cat > ... << 'GH_AW_SAFE_OUTPUTS_TOOLS_META_<hex>_EOF'
	openMarker := "<< '" + delimiter + "'\n"
	start := strings.Index(yamlStr, openMarker)
	require.NotEqual(t, -1, start, "lock file should contain tools_meta heredoc opening delimiter")
	contentStart := start + len(openMarker)

	// Closing line is: "          GH_AW_SAFE_OUTPUTS_TOOLS_META_<hex>_EOF" (indented)
	closeMarker := "          " + delimiter + "\n"
	end := strings.Index(yamlStr[contentStart:], closeMarker)
	require.NotEqual(t, -1, end, "lock file should contain tools_meta heredoc closing delimiter")

	raw := yamlStr[contentStart : contentStart+end]

	// Strip YAML indentation (each line is indented with "          ")
	var sb strings.Builder
	for _, line := range strings.Split(raw, "\n") {
		sb.WriteString(strings.TrimPrefix(line, "          "))
		sb.WriteString("\n")
	}

	var meta ToolsMeta
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(sb.String())), &meta),
		"tools_meta.json from lock file should be valid JSON")
	return meta
}

// constraint-bearing configurations to ensure no tool type regresses silently.
