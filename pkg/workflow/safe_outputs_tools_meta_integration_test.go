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
func TestToolsMetaJSONContainsDescriptionSuffixes(t *testing.T) {
	tmpDir := testutil.TempDir(t, "tools-meta-desc-test")

	testContent := `---
on: push
name: Test Tools Meta Description
engine: copilot
safe-outputs:
  create-issue:
    max: 5
    title-prefix: "[bot] "
    labels:
      - automation
      - testing
  add-comment:
    max: 10
---

Test workflow for tools meta description suffixes.
`

	testFile := filepath.Join(tmpDir, "test-tools-meta-desc.md")
	require.NoError(t, os.WriteFile(testFile, []byte(testContent), 0644), "should write test file")

	compiler := NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(testFile), "compilation should succeed")

	lockFile := filepath.Join(tmpDir, "test-tools-meta-desc.lock.yml")
	yamlBytes, err := os.ReadFile(lockFile)
	require.NoError(t, err, "should read lock file")
	yamlStr := string(yamlBytes)

	// Must write tools_meta.json (new approach), NOT tools.json
	assert.Contains(t, yamlStr, "cat > ${RUNNER_TEMP}/gh-aw/safeoutputs/tools_meta.json", "should write tools_meta.json")
	assert.NotContains(t, yamlStr, "cat > ${RUNNER_TEMP}/gh-aw/safeoutputs/tools.json", "should NOT inline tools.json")

	// Must invoke the JavaScript generator at runtime
	assert.Contains(t, yamlStr, "node ${RUNNER_TEMP}/gh-aw/actions/generate_safe_outputs_tools.cjs", "should invoke JS generator")

	// Extract the tools_meta.json content from the lock file
	meta := extractToolsMetaFromLockFile(t, yamlStr)

	// Verify description suffixes are present and correct
	createIssueSuffix, ok := meta.DescriptionSuffixes["create_issue"]
	require.True(t, ok, "create_issue should have a description suffix")
	assert.Contains(t, createIssueSuffix, "CONSTRAINTS:", "suffix should contain CONSTRAINTS")
	assert.Contains(t, createIssueSuffix, "Maximum 5 issue(s)", "suffix should include max constraint")
	assert.Contains(t, createIssueSuffix, `Title will be prefixed with "[bot] "`, "suffix should include title prefix")
	assert.Contains(t, createIssueSuffix, `Labels ["automation" "testing"]`, "suffix should include labels")

	addCommentSuffix, ok := meta.DescriptionSuffixes["add_comment"]
	require.True(t, ok, "add_comment should have a description suffix")
	assert.Contains(t, addCommentSuffix, "Maximum 10 comment(s)", "suffix should include max constraint")
}

// TestToolsMetaJSONDescriptionsMatchFilteredTools verifies that the description suffixes
// in tools_meta.json match what generateFilteredToolsJSON would have embedded directly.
// This is the primary regression test for the strategy change.
func TestToolsMetaJSONDescriptionsMatchFilteredTools(t *testing.T) {
	tests := []struct {
		name        string
		safeOutputs *SafeOutputsConfig
		toolName    string
	}{
		{
			name: "create_issue with max and labels",
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("3")},
					TitlePrefix:          "[automated] ",
					Labels:               []string{"bot", "enhancement"},
					AllowedLabels:        []string{"bug", "feature"},
				},
			},
			toolName: "create_issue",
		},
		{
			name: "add_comment with max and target",
			safeOutputs: &SafeOutputsConfig{
				AddComments: &AddCommentsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("5")},
					Target:               "trigger",
				},
			},
			toolName: "add_comment",
		},
		{
			name: "create_pull_request with draft and reviewers",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("2")},
					TitlePrefix:          "[pr] ",
					Labels:               []string{"ci"},
					Draft:                strPtr("true"),
					Reviewers:            []string{"reviewer1"},
				},
			},
			toolName: "create_pull_request",
		},
		{
			name: "upload_asset with max and size limit",
			safeOutputs: &SafeOutputsConfig{
				UploadAssets: &UploadAssetsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")},
					MaxSizeKB:            1024,
					AllowedExts:          []string{".zip", ".tar.gz"},
				},
			},
			toolName: "upload_asset",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := &WorkflowData{SafeOutputs: tt.safeOutputs}

			// Old approach: full tool JSON with description already embedded
			filteredJSON, err := generateFilteredToolsJSON(data, ".github/workflows/test.md")
			require.NoError(t, err, "generateFilteredToolsJSON should not error")

			var filteredTools []map[string]any
			require.NoError(t, json.Unmarshal([]byte(filteredJSON), &filteredTools), "filtered JSON should parse")

			var oldTool map[string]any
			for _, tool := range filteredTools {
				if tool["name"] == tt.toolName {
					oldTool = tool
					break
				}
			}
			require.NotNil(t, oldTool, "old approach should include tool %s", tt.toolName)
			oldDescription, ok := oldTool["description"].(string)
			require.True(t, ok, "old tool should have string description")

			// New approach: tools_meta.json with description suffix
			metaJSON, err := generateToolsMetaJSON(data, ".github/workflows/test.md")
			require.NoError(t, err, "generateToolsMetaJSON should not error")

			var meta ToolsMeta
			require.NoError(t, json.Unmarshal([]byte(metaJSON), &meta), "meta JSON should parse")

			// The suffix should equal the constraint portion of the old description.
			// Old description = baseDescription + suffix
			// Example: "Create a new GitHub issue..." + " CONSTRAINTS: Maximum 3 issue(s) can be created."
			suffix, hasSuffix := meta.DescriptionSuffixes[tt.toolName]
			if strings.Contains(oldDescription, "CONSTRAINTS:") {
				require.True(t, hasSuffix, "tools_meta should have suffix for %s when old description has constraints", tt.toolName)
				assert.True(t, strings.HasSuffix(oldDescription, suffix),
					"old description should end with the suffix from tools_meta\n  old:    %q\n  suffix: %q", oldDescription, suffix)
			} else {
				assert.False(t, hasSuffix, "tools_meta should NOT have suffix for %s when there are no constraints", tt.toolName)
			}
		})
	}
}

// TestToolsMetaJSONRepoParamsMatchFilteredTools verifies that repo parameter definitions
// in tools_meta.json match the repo parameters embedded in the old full tools.json.
func TestToolsMetaJSONRepoParamsMatchFilteredTools(t *testing.T) {
	tests := []struct {
		name            string
		safeOutputs     *SafeOutputsConfig
		toolName        string
		expectRepoParam bool
		expectWildcard  bool
	}{
		{
			name: "create_issue with allowed-repos",
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")},
					TargetRepoSlug:       "org/default-repo",
					AllowedRepos:         []string{"org/other-repo"},
				},
			},
			toolName:        "create_issue",
			expectRepoParam: true,
		},
		{
			name: "update_issue with wildcard target-repo",
			safeOutputs: &SafeOutputsConfig{
				UpdateIssues: &UpdateIssuesConfig{
					UpdateEntityConfig: UpdateEntityConfig{
						SafeOutputTargetConfig: SafeOutputTargetConfig{
							TargetRepoSlug: "*",
						},
					},
				},
			},
			toolName:        "update_issue",
			expectRepoParam: true,
			expectWildcard:  true,
		},
		{
			name: "create_issue without allowed-repos",
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")},
					TargetRepoSlug:       "org/target",
				},
			},
			toolName:        "create_issue",
			expectRepoParam: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := &WorkflowData{SafeOutputs: tt.safeOutputs}

			// Old approach: get repo param from embedded tools.json
			filteredJSON, err := generateFilteredToolsJSON(data, ".github/workflows/test.md")
			require.NoError(t, err, "generateFilteredToolsJSON should not error")

			var filteredTools []map[string]any
			require.NoError(t, json.Unmarshal([]byte(filteredJSON), &filteredTools), "filtered JSON should parse")

			var oldTool map[string]any
			for _, tool := range filteredTools {
				if tool["name"] == tt.toolName {
					oldTool = tool
					break
				}
			}
			require.NotNil(t, oldTool, "old approach should include tool %s", tt.toolName)

			oldInputSchema, _ := oldTool["inputSchema"].(map[string]any)
			oldProperties, _ := oldInputSchema["properties"].(map[string]any)
			oldRepoParam, oldHasRepo := oldProperties["repo"]

			// New approach
			metaJSON, err := generateToolsMetaJSON(data, ".github/workflows/test.md")
			require.NoError(t, err, "generateToolsMetaJSON should not error")

			var meta ToolsMeta
			require.NoError(t, json.Unmarshal([]byte(metaJSON), &meta), "meta JSON should parse")

			newRepoParam, newHasRepo := meta.RepoParams[tt.toolName]

			// Both old and new approaches should agree on whether repo param is present
			assert.Equal(t, oldHasRepo, newHasRepo,
				"old and new approaches should agree on repo param presence for %s", tt.toolName)

			if tt.expectRepoParam {
				require.True(t, newHasRepo, "tools_meta should include repo_param for %s", tt.toolName)

				// Description should be consistent
				oldRepoMap, _ := oldRepoParam.(map[string]any)
				oldRepoDesc, _ := oldRepoMap["description"].(string)
				newRepoDesc, _ := newRepoParam["description"].(string)
				assert.Equal(t, oldRepoDesc, newRepoDesc, "repo param description should match between old and new")

				if tt.expectWildcard {
					assert.Contains(t, newRepoDesc, "Any repository can be targeted",
						"wildcard repo param description should mention any repo")
				}
			} else {
				assert.False(t, newHasRepo, "tools_meta should NOT include repo_param for %s", tt.toolName)
			}
		})
	}
}

// TestToolsMetaJSONDynamicToolsFromCustomJobs verifies that custom safe-job tool
// definitions are placed in dynamic_tools in tools_meta.json.
func TestToolsMetaJSONDynamicToolsFromCustomJobs(t *testing.T) {
	data := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			Jobs: map[string]*SafeJobConfig{
				"deploy_app": {
					Description: "Deploy the application",
					Inputs: map[string]*InputDefinition{
						"env": {
							Type:        "choice",
							Options:     []string{"staging", "production"},
							Description: "Target environment",
							Required:    true,
						},
					},
				},
			},
		},
	}

	// Old approach produces a single custom-job tool in filtered JSON
	filteredJSON, err := generateFilteredToolsJSON(data, ".github/workflows/test.md")
	require.NoError(t, err, "generateFilteredToolsJSON should not error")
	var filteredTools []map[string]any
	require.NoError(t, json.Unmarshal([]byte(filteredJSON), &filteredTools), "filtered JSON should parse")
	require.Len(t, filteredTools, 1, "old approach should have exactly 1 tool")
	assert.Equal(t, "deploy_app", filteredTools[0]["name"], "tool should be named deploy_app")

	// New approach places it in dynamic_tools
	metaJSON, err := generateToolsMetaJSON(data, ".github/workflows/test.md")
	require.NoError(t, err, "generateToolsMetaJSON should not error")
	var meta ToolsMeta
	require.NoError(t, json.Unmarshal([]byte(metaJSON), &meta), "meta JSON should parse")

	require.Len(t, meta.DynamicTools, 1, "tools_meta should have exactly 1 dynamic tool")
	dynTool := meta.DynamicTools[0]
	assert.Equal(t, "deploy_app", dynTool["name"], "dynamic tool name should match")
	assert.Equal(t, "Deploy the application", dynTool["description"], "dynamic tool description should match")

	inputSchema, ok := dynTool["inputSchema"].(map[string]any)
	require.True(t, ok, "dynamic tool should have inputSchema")
	properties, ok := inputSchema["properties"].(map[string]any)
	require.True(t, ok, "inputSchema should have properties")
	envProp, ok := properties["env"].(map[string]any)
	require.True(t, ok, "env property should exist")
	assert.Equal(t, "string", envProp["type"], "choice type should be mapped to string in JSON Schema")
	assert.Equal(t, []any{"staging", "production"}, envProp["enum"], "enum values should match options")
}

// TestToolsMetaJSONCompiledWorkflowEmbedsMeta verifies end-to-end that a compiled lock file
// contains tools_meta.json (not an inlined tools.json) with the right description constraints.
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
func TestToolsMetaJSONPushRepoMemory(t *testing.T) {
	data := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{},
		RepoMemoryConfig: &RepoMemoryConfig{
			Memories: []RepoMemoryEntry{
				{ID: "default", MaxFileSize: 102400, MaxPatchSize: 10240, MaxFileCount: 100},
			},
		},
	}

	// Old approach includes push_repo_memory in filtered tools
	filteredJSON, err := generateFilteredToolsJSON(data, ".github/workflows/test.md")
	require.NoError(t, err, "generateFilteredToolsJSON should not error")
	var filteredTools []map[string]any
	require.NoError(t, json.Unmarshal([]byte(filteredJSON), &filteredTools), "filtered JSON should parse")
	var foundOld bool
	for _, tool := range filteredTools {
		if tool["name"] == "push_repo_memory" {
			foundOld = true
			break
		}
	}
	assert.True(t, foundOld, "old approach should include push_repo_memory when RepoMemoryConfig is set")

	// New approach: push_repo_memory is a predefined static tool, so it appears in
	// computeEnabledToolNames and its description suffix (if any) goes in tools_meta.
	enabledTools := computeEnabledToolNames(data)
	assert.True(t, enabledTools["push_repo_memory"], "computeEnabledToolNames should include push_repo_memory")
}

// TestToolsMetaJSONEmptyWhenNoSafeOutputs verifies that tools_meta.json is empty
// (no description_suffixes, no repo_params, no dynamic_tools) when safe-outputs is nil.
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

// TestToolsMetaJSONCustomDescriptionsAllToolTypes exercises all tool types that have
// constraint-bearing configurations to ensure no tool type regresses silently.
func TestToolsMetaJSONCustomDescriptionsAllToolTypes(t *testing.T) {
	tests := []struct {
		name         string
		safeOutputs  *SafeOutputsConfig
		toolName     string
		wantContains string
	}{
		{
			name: "create_discussion with max and category",
			safeOutputs: &SafeOutputsConfig{
				CreateDiscussions: &CreateDiscussionsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("2")},
					Category:             "announcements",
				},
			},
			toolName:     "create_discussion",
			wantContains: `Discussions will be created in category "announcements"`,
		},
		{
			name: "close_issue with max and required prefix",
			safeOutputs: &SafeOutputsConfig{
				CloseIssues: &CloseIssuesConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("5")},
					SafeOutputFilterConfig: SafeOutputFilterConfig{
						RequiredTitlePrefix: "[bot] ",
					},
				},
			},
			toolName:     "close_issue",
			wantContains: `Only issues with title prefix "[bot] "`,
		},
		{
			name: "add_labels with allowed labels",
			safeOutputs: &SafeOutputsConfig{
				AddLabels: &AddLabelsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("3")},
					Allowed:              []string{"bug", "enhancement"},
				},
			},
			toolName:     "add_labels",
			wantContains: `Only these labels are allowed: ["bug" "enhancement"]`,
		},
		{
			name: "update_project with project URL",
			safeOutputs: &SafeOutputsConfig{
				UpdateProjects: &UpdateProjectConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")},
					Project:              "https://github.com/orgs/org/projects/1",
				},
			},
			toolName:     "update_project",
			wantContains: "https://github.com/orgs/org/projects/1",
		},
		{
			name: "assign_to_agent with base branch",
			safeOutputs: &SafeOutputsConfig{
				AssignToAgent: &AssignToAgentConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")},
					BaseBranch:           "main",
					SafeOutputTargetConfig: SafeOutputTargetConfig{
						TargetRepoSlug: "org/target",
					},
				},
			},
			toolName:     "assign_to_agent",
			wantContains: `Pull requests will target the "main" branch`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := &WorkflowData{SafeOutputs: tt.safeOutputs}

			metaJSON, err := generateToolsMetaJSON(data, ".github/workflows/test.md")
			require.NoError(t, err, "generateToolsMetaJSON should not error")

			var meta ToolsMeta
			require.NoError(t, json.Unmarshal([]byte(metaJSON), &meta), "meta JSON should parse")

			suffix, ok := meta.DescriptionSuffixes[tt.toolName]
			require.True(t, ok, "tools_meta should have description_suffix for %s", tt.toolName)
			assert.Contains(t, suffix, tt.wantContains,
				"description suffix for %s should contain %q\n  got: %q", tt.toolName, tt.wantContains, suffix)

			// Regression guard: verify generateFilteredToolsJSON (old approach) would produce
			// the same constraint text in the tool's description.
			filteredJSON, err := generateFilteredToolsJSON(data, ".github/workflows/test.md")
			require.NoError(t, err, "generateFilteredToolsJSON should not error")

			// Parse the JSON to access the actual description string (avoids JSON escape issues)
			var filteredTools []map[string]any
			require.NoError(t, json.Unmarshal([]byte(filteredJSON), &filteredTools), "filtered JSON should parse")
			var oldTool map[string]any
			for _, tool := range filteredTools {
				if tool["name"] == tt.toolName {
					oldTool = tool
					break
				}
			}
			require.NotNil(t, oldTool, "old filtered tools should include %s", tt.toolName)
			oldDescription, ok := oldTool["description"].(string)
			require.True(t, ok, "old tool description should be a string")
			assert.Contains(t, oldDescription, tt.wantContains,
				"old filtered JSON should also contain the constraint text for %s (regression guard)", tt.toolName)
		})
	}
}

// extractToolsMetaFromLockFile parses the tools_meta.json heredoc embedded in a compiled lock file.
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
