//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgenticWorkflowsWriteToolsSchema(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value any
		valid bool
	}{
		{"default", nil, true},
		{"enabled", true, true},
		{"disabled", false, true},
		{"object defaults", map[string]any{}, true},
		{"opt in", map[string]any{"allow-write-tools": true}, true},
		{"opt out", map[string]any{"allow-write-tools": false}, true},
		{"non boolean", map[string]any{"allow-write-tools": "true"}, false},
		{"caller flag", map[string]any{"allow_write_tools": true}, false},
		{"unknown", map[string]any{"trusted": true}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := parser.ValidateMainWorkflowFrontmatterWithSchemaAndLocation(map[string]any{
				"on": "workflow_dispatch", "tools": map[string]any{"agentic-workflows": tc.value},
			}, "test.md")
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestAgenticWorkflowsWriteToolsRendering(t *testing.T) {
	for _, mode := range []ActionMode{ActionModeDev, ActionModeRelease, ActionModeScript, ActionModeAction} {
		for _, format := range []string{"json", "toml"} {
			for _, allow := range []bool{false, true} {
				t.Run(string(mode)+"/"+format+"/"+map[bool]string{false: "default", true: "opt-in"}[allow], func(t *testing.T) {
					data := workflowDataWithAgenticWorkflows()
					data.ActionMode = mode
					data.Tools["agentic-workflows"] = map[string]any{"allow-write-tools": allow}
					renderer := buildMCPRendererFactory(data, format, false, false)(true)
					var output strings.Builder
					renderer.RenderAgenticWorkflowsMCP(&output)
					if allow {
						assert.Contains(t, output.String(), `"mcp-server", "--validate-actor", "--allow-write-tools"`)
					} else {
						assert.NotContains(t, output.String(), "--allow-write-tools")
					}
					if !mode.IsDev() || allow {
						assert.Contains(t, output.String(), "--validate-actor")
					}
				})
			}
		}
	}
}

func TestAgenticWorkflowsWriteToolsCompile(t *testing.T) {
	for _, config := range []string{"", "true", "{}", "{allow-write-tools: false}", "{allow-write-tools: true}"} {
		t.Run(config, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "test.md")
			require.NoError(t, os.WriteFile(path, []byte("---\non: workflow_dispatch\npermissions:\n  actions: read\ntools:\n  agentic-workflows: "+config+"\n---\nTest local compile.\n"), 0o600))
			compiler := NewCompiler()
			compiler.SetActionMode(ActionModeRelease)
			require.NoError(t, compiler.CompileWorkflow(path))
			lock, err := os.ReadFile(strings.TrimSuffix(path, ".md") + ".lock.yml")
			require.NoError(t, err)
			assert.Equal(t, strings.Contains(config, ": true"), strings.Contains(string(lock), "--allow-write-tools"))
			assert.Contains(t, string(lock), "--validate-actor")
			tools, err := ParseToolsConfig(map[string]any{"agentic-workflows": map[string]any{"allow-write-tools": true}})
			require.NoError(t, err)
			assert.True(t, tools.AgenticWorkflows.Enabled)
			assert.True(t, tools.AgenticWorkflows.AllowWriteTools)
			assert.Equal(t, map[string]any{"allow-write-tools": true}, tools.ToMap()["agentic-workflows"])
		})
	}
}

func TestDailyCLIToolsTesterPolicy(t *testing.T) {
	path := filepath.Join("..", "..", ".github", "workflows", "daily-cli-tools-tester.md")
	source, err := os.ReadFile(path)
	require.NoError(t, err)
	compiler := NewCompiler(WithWorkflowIdentifier("daily-cli-tools-tester"))
	data, err := compiler.ParseWorkflowFile(path)
	require.NoError(t, err)
	assert.True(t, parseAgenticWorkflowsTool(data.Tools["agentic-workflows"]).AllowWriteTools)
	prompt := string(source)
	assert.Contains(t, prompt, "dry_run: true")
	assert.Contains(t, prompt, "strict: true")
	assert.Contains(t, prompt, "never list, search, or read that raw directory")
	assert.Contains(t, prompt, "if session policy permits reading it")
	assert.Contains(t, prompt, "A later `noop` must not override that incomplete status or claim success")
	assert.NotContains(t, prompt, "find /tmp/gh-aw/aw-mcp/logs/")
	assert.NotContains(t, prompt, "ls -R /tmp/gh-aw/aw-mcp/logs/")
	assert.NotContains(t, prompt, "Trigger it")
	assert.NotContains(t, prompt, "run one bulk compile")
}

// Helper functions for test setup

// testCompiler creates a test compiler with validation skipped
func testCompiler() *Compiler {
	c := NewCompiler()
	c.SetSkipValidation(true)
	return c
}

// workflowDataWithAgenticWorkflows creates test workflow data with agentic-workflows tool
func workflowDataWithAgenticWorkflows(options ...func(*WorkflowData)) *WorkflowData {
	wd := &WorkflowData{
		Tools: map[string]any{
			"agentic-workflows": nil,
		},
	}
	for _, opt := range options {
		opt(wd)
	}
	return wd
}

// withImportedFiles is an option for workflowDataWithAgenticWorkflows
func withImportedFiles(files ...string) func(*WorkflowData) {
	return func(wd *WorkflowData) {
		wd.ImportedFiles = files
	}
}

// Test functions

func TestAgenticWorkflowsSyntaxVariations(t *testing.T) {
	tests := []struct {
		name        string
		toolValue   any
		shouldWork  bool
		description string
	}{
		{
			name:        "agentic-workflows with nil (no value)",
			toolValue:   nil,
			shouldWork:  true,
			description: "Should enable agentic-workflows when field is present without value",
		},
		{
			name:        "agentic-workflows with true",
			toolValue:   true,
			shouldWork:  true,
			description: "Should enable agentic-workflows with boolean true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a minimal workflow with the agentic-workflows tool
			frontmatter := map[string]any{
				"on":    "workflow_dispatch",
				"tools": map[string]any{"agentic-workflows": tt.toolValue},
			}

			// Create compiler using helper
			c := testCompiler()

			// Extract tools from frontmatter
			tools := extractToolsMapFromFrontmatter(frontmatter)

			// Merge tools
			mergedTools, err := c.mergeToolsAndMCPServers(tools, make(map[string]any), "")

			if tt.shouldWork {
				require.NoError(t, err, "agentic-workflows tool should merge without errors for: %s", tt.description)
				assert.Contains(t, mergedTools, "agentic-workflows",
					"merged tools should contain agentic-workflows after successful merge")
			} else {
				require.Error(t, err, "agentic-workflows tool should fail for: %s", tt.description)
			}
		})
	}
}

func TestAgenticWorkflowsMCPConfigGeneration(t *testing.T) {
	engines := []struct {
		name   string
		engine CodingAgentEngine
	}{
		{"Claude", NewClaudeEngine()},
		{"Copilot", NewCopilotEngine()},
		{"Codex", NewCodexEngine()},
		{"Pi", NewPiEngine()},
	}

	for _, e := range engines {
		t.Run(e.name, func(t *testing.T) {
			// Create workflow data using helper
			workflowData := workflowDataWithAgenticWorkflows()

			// Generate MCP config
			var yaml strings.Builder
			mcpTools := []string{"agentic-workflows"}

			err := e.engine.RenderMCPConfig(&yaml, workflowData.Tools, mcpTools, workflowData)
			require.NoError(t, err)
			result := yaml.String()

			// Verify the MCP config contains agentic-workflows
			assert.Contains(t, result, constants.AgenticWorkflowsMCPServerID.String(),
				"%s engine should generate MCP config with agenticworkflows server name", e.name)
			assert.Contains(t, result, "gh",
				"%s engine MCP config should use gh CLI command for agentic-workflows", e.name)
			assert.Contains(t, result, "mcp-server",
				"%s engine MCP config should include mcp-server argument for gh-aw extension", e.name)
		})
	}
}

func TestAgenticWorkflowsHasMCPServers(t *testing.T) {
	// Create workflow data using helper
	workflowData := workflowDataWithAgenticWorkflows()

	assert.True(t, HasMCPServers(workflowData),
		"HasMCPServers should return true when agentic-workflows tool is configured")
}

// TestPiMCPConfig_SafeOutputs verifies that the Pi engine generates a valid MCP config
// that includes the safeoutputs server. This directly documents the bug fixed in
// cda969a: Pi engine was not rendering MCP config so safeoutputs was never mounted.
func TestPiMCPConfig_SafeOutputs(t *testing.T) {
	workflowData := &WorkflowData{
		Tools: map[string]any{
			"safe-outputs": map[string]any{
				"create-issue": map[string]any{},
			},
		},
	}

	var sb strings.Builder
	err := NewPiEngine().RenderMCPConfig(&sb, workflowData.Tools, []string{"safe-outputs"}, workflowData)
	require.NoError(t, err)

	result := sb.String()
	assert.Contains(t, result, "safeoutputs",
		"Pi MCP config must include safeoutputs server so the CLI can be mounted")
	assert.Contains(t, result, "start_safe_outputs_mcp.sh",
		"Pi MCP config must reference the safe-outputs startup script")
}

func TestAgenticWorkflowsInstallStepIncludesGHToken(t *testing.T) {
	// Create workflow data using helper
	workflowData := workflowDataWithAgenticWorkflows()

	// Create compiler using helper
	c := testCompiler()
	c.actionMode = ActionModeAction
	c.version = "v0.72.1"

	// Generate MCP setup
	var yaml strings.Builder
	engine := NewCopilotEngine()

	require.NoError(t, c.generateMCPSetup(&yaml, workflowData.Tools, engine, workflowData))
	result := yaml.String()

	// Verify the install step is present
	assert.Contains(t, result, "Install gh-aw extension",
		"MCP setup should include gh-aw installation step when agentic-workflows tool is enabled")

	// Verify setup-cli action is used with default token expression
	assert.Contains(t, result, "uses: github/gh-aw-actions/setup-cli@",
		"install step should use setup-cli action")
	assert.Contains(t, result, "version: 'v0.72.1'",
		"install step should install the compiler release version")
	assert.Contains(t, result, "github-token: ${{ secrets.GH_AW_GITHUB_MCP_SERVER_TOKEN || secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}",
		"install step should use default github-token fallback chain when no custom token is specified")
	assert.NotContains(t, result, "setup-cli@main",
		"install step should not use mutable main ref for setup-cli action")
	assert.NotContains(t, result, "version: latest",
		"install step should not use mutable latest CLI version")

	// Verify follow-up copy/verification commands are present
	assert.Contains(t, result, "Copy gh-aw binary for MCP Server",
		"MCP setup should include a step to copy gh-aw binary for MCP Server containerization")
	assert.Contains(t, result, "bash \"${RUNNER_TEMP}/gh-aw/actions/copy_gh_aw_binary_for_mcp.sh\"",
		"install step should invoke shared copy_gh_aw_binary_for_mcp.sh helper")
}

func TestAgenticWorkflowsInstallStepPresentWithoutImport(t *testing.T) {
	// Create workflow data using helper with empty imports
	workflowData := workflowDataWithAgenticWorkflows(withImportedFiles())

	// Create compiler using helper
	c := testCompiler()
	c.actionMode = ActionModeDev

	// Generate MCP setup
	var yaml strings.Builder
	engine := NewCopilotEngine()

	require.NoError(t, c.generateMCPSetup(&yaml, workflowData.Tools, engine, workflowData))
	result := yaml.String()

	// Verify dev install step is present for agentic-workflows tool
	assert.Contains(t, result, "Build and install gh-aw CLI from source",
		"dev mode should build and install gh-aw from source")
	assert.Contains(t, result, "gh extension install .",
		"dev mode should install gh-aw extension from local checkout")
	assert.NotContains(t, result, "setup-cli@",
		"dev mode should not use setup-cli action")
}

// TestAgenticWorkflowsErrorCases tests error handling for invalid configurations
func TestAgenticWorkflowsErrorCases(t *testing.T) {
	tests := []struct {
		name          string
		toolValue     any
		expectedError bool
		description   string
	}{
		{
			name:          "agentic-workflows with false",
			toolValue:     false,
			expectedError: false,
			description:   "Should allow explicitly disabling agentic-workflows with false",
		},
		{
			name:          "agentic-workflows with empty map",
			toolValue:     map[string]any{},
			expectedError: false,
			description:   "Should handle empty configuration map without error",
		},
		{
			name:          "agentic-workflows with string value",
			toolValue:     "enabled",
			expectedError: false,
			description:   "Should handle string value (non-standard but permitted)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a minimal workflow with the agentic-workflows tool
			frontmatter := map[string]any{
				"on":    "workflow_dispatch",
				"tools": map[string]any{"agentic-workflows": tt.toolValue},
			}

			// Create compiler using helper
			c := testCompiler()

			// Extract tools from frontmatter
			tools := extractToolsMapFromFrontmatter(frontmatter)

			// Merge tools
			mergedTools, err := c.mergeToolsAndMCPServers(tools, make(map[string]any), "")

			if tt.expectedError {
				require.Error(t, err, "should fail for: %s", tt.description)
			} else {
				require.NoError(t, err, "should succeed for: %s", tt.description)
				// When tool is false, it should not be in merged tools (or be explicitly false)
				if tt.toolValue == false {
					// The tool might be present but set to false, or absent entirely
					if val, exists := mergedTools["agentic-workflows"]; exists {
						assert.False(t, val.(bool), "agentic-workflows should be false when explicitly disabled")
					}
				} else {
					// For other values, the tool should be present
					assert.Contains(t, mergedTools, "agentic-workflows",
						"merged tools should contain agentic-workflows for non-false values")
				}
			}
		})
	}
}

// TestAgenticWorkflowsNilSafety tests nil and empty input handling
func TestAgenticWorkflowsNilSafety(t *testing.T) {
	tests := []struct {
		name          string
		workflowData  *WorkflowData
		shouldHaveMCP bool
		description   string
	}{
		{
			name:          "nil workflow data",
			workflowData:  nil,
			shouldHaveMCP: false,
			description:   "Should handle nil workflow data gracefully",
		},
		{
			name: "nil tools map",
			workflowData: &WorkflowData{
				Tools: nil,
			},
			shouldHaveMCP: false,
			description:   "Should handle nil tools map gracefully",
		},
		{
			name: "empty tools map",
			workflowData: &WorkflowData{
				Tools: make(map[string]any),
			},
			shouldHaveMCP: false,
			description:   "Should handle empty tools map gracefully",
		},
		{
			name: "agentic-workflows with nil value",
			workflowData: &WorkflowData{
				Tools: map[string]any{
					"agentic-workflows": nil,
				},
			},
			shouldHaveMCP: true,
			description:   "Should detect agentic-workflows tool even with nil value",
		},
		{
			name: "agentic-workflows explicitly disabled",
			workflowData: &WorkflowData{
				Tools: map[string]any{
					"agentic-workflows": false,
				},
			},
			shouldHaveMCP: false,
			description:   "Should not detect MCP servers when agentic-workflows is explicitly false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that HasMCPServers doesn't panic
			var result bool
			assert.NotPanics(t, func() {
				result = HasMCPServers(tt.workflowData)
			}, "HasMCPServers should handle nil/empty data gracefully without panicking")

			// Verify the expected result
			assert.Equal(t, tt.shouldHaveMCP, result,
				"HasMCPServers result for: %s", tt.description)
		})
	}
}

// TestAgenticWorkflowsExtractToolsEdgeCases tests edge cases in extractToolsMapFromFrontmatter
func TestAgenticWorkflowsExtractToolsEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter map[string]any
		expectTools bool
		description string
	}{
		{
			name:        "nil frontmatter",
			frontmatter: nil,
			expectTools: false,
			description: "Should handle nil frontmatter without panic",
		},
		{
			name:        "empty frontmatter",
			frontmatter: map[string]any{},
			expectTools: false,
			description: "Should handle empty frontmatter",
		},
		{
			name: "frontmatter without tools",
			frontmatter: map[string]any{
				"on": "workflow_dispatch",
			},
			expectTools: false,
			description: "Should handle frontmatter without tools field",
		},
		{
			name: "tools with invalid type (string)",
			frontmatter: map[string]any{
				"tools": "not-a-map",
			},
			expectTools: false,
			description: "Should handle tools field with invalid type",
		},
		{
			name: "tools with nil value",
			frontmatter: map[string]any{
				"tools": nil,
			},
			expectTools: false,
			description: "Should handle tools field with nil value",
		},
		{
			name: "valid tools with agentic-workflows",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"agentic-workflows": nil,
				},
			},
			expectTools: true,
			description: "Should extract valid tools configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that extractToolsMapFromFrontmatter doesn't panic
			var result map[string]any
			assert.NotPanics(t, func() {
				result = extractToolsMapFromFrontmatter(tt.frontmatter)
			}, "extractToolsMapFromFrontmatter should handle edge cases without panicking")

			// Verify the expected result
			if tt.expectTools {
				assert.NotNil(t, result, "should extract tools for: %s", tt.description)
				assert.NotEmpty(t, result, "should extract non-empty tools for: %s", tt.description)
				assert.Contains(t, result, "agentic-workflows",
					"extracted tools should contain agentic-workflows for: %s", tt.description)
			} else {
				// ExtractMapField returns empty map (not nil) when field is missing or invalid
				assert.NotNil(t, result, "extractToolsMapFromFrontmatter should always return non-nil map")
				assert.Empty(t, result, "should return empty tools map for: %s", tt.description)
			}
		})
	}
}
