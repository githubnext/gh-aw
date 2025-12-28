package workflow

import (
	"strings"
	"testing"
)

func TestMissingToolSafeOutput(t *testing.T) {
	tests := []struct {
		name         string
		frontmatter  map[string]any
		expectConfig bool
		expectJob    bool
		expectMax    int
	}{
		{
			name:         "No safe-outputs config should NOT enable missing-tool by default",
			frontmatter:  map[string]any{"name": "Test"},
			expectConfig: false,
			expectJob:    false,
			expectMax:    0,
		},
		{
			name: "Safe-outputs with other config should enable missing-tool by default",
			frontmatter: map[string]any{
				"name": "Test",
				"safe-outputs": map[string]any{
					"create-issue": nil,
				},
			},
			expectConfig: true,
			expectJob:    true,
			expectMax:    0,
		},
		{
			name: "Explicit missing-tool: false should disable it",
			frontmatter: map[string]any{
				"name": "Test",
				"safe-outputs": map[string]any{
					"create-issue": nil,
					"missing-tool": false,
				},
			},
			expectConfig: false,
			expectJob:    false,
			expectMax:    0,
		},
		{
			name: "Explicit missing-tool config with max",
			frontmatter: map[string]any{
				"name": "Test",
				"safe-outputs": map[string]any{
					"missing-tool": map[string]any{
						"max": 5,
					},
				},
			},
			expectConfig: true,
			expectJob:    true,
			expectMax:    5,
		},
		{
			name: "Missing-tool with other safe outputs",
			frontmatter: map[string]any{
				"name": "Test",
				"safe-outputs": map[string]any{
					"create-issue": nil,
					"missing-tool": nil,
				},
			},
			expectConfig: true,
			expectJob:    true,
			expectMax:    0,
		},
		{
			name: "Empty missing-tool config",
			frontmatter: map[string]any{
				"name": "Test",
				"safe-outputs": map[string]any{
					"missing-tool": nil,
				},
			},
			expectConfig: true,
			expectJob:    true,
			expectMax:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler(false, "", "test")

			// Extract safe outputs config
			safeOutputs := compiler.extractSafeOutputsConfig(tt.frontmatter)

			// Verify config expectations
			if tt.expectConfig {
				if safeOutputs == nil {
					t.Fatal("Expected SafeOutputsConfig to be created, but it was nil")
				}
				if safeOutputs.MissingTool == nil {
					t.Fatal("Expected MissingTool config to be enabled, but it was nil")
				}
				if safeOutputs.MissingTool.Max != tt.expectMax {
					t.Errorf("Expected max to be %d, got %d", tt.expectMax, safeOutputs.MissingTool.Max)
				}
			} else {
				if safeOutputs != nil && safeOutputs.MissingTool != nil {
					t.Error("Expected MissingTool config to be nil, but it was not")
				}
			}

			// Test job creation
			if tt.expectJob {
				if safeOutputs == nil || safeOutputs.MissingTool == nil {
					t.Error("Expected SafeOutputs and MissingTool config to exist for job creation test")
				} else {
					job, err := compiler.buildCreateOutputMissingToolJob(&WorkflowData{
						SafeOutputs: safeOutputs,
					}, "main-job")
					if err != nil {
						t.Errorf("Failed to build missing tool job: %v", err)
					}
					if job == nil {
						t.Error("Expected job to be created, but it was nil")
					}
					if job != nil {
						if job.Name != "missing_tool" {
							t.Errorf("Expected job name to be 'missing_tool', got '%s'", job.Name)
						}
						if len(job.Needs) != 1 || job.Needs[0] != "main-job" {
							t.Errorf("Expected job to depend on 'main-job', got %v", job.Needs)
						}
					}
				}
			}
		})
	}
}

func TestGeneratePromptIncludesGitHubAWPrompt(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	data := &WorkflowData{
		MarkdownContent: "Test workflow content",
	}

	var yaml strings.Builder
	compiler.generatePrompt(&yaml, data)

	output := yaml.String()

	// Check that GH_AW_PROMPT environment variable is always included
	if !strings.Contains(output, "GH_AW_PROMPT: /tmp/gh-aw/aw-prompts/prompt.txt") {
		t.Error("Expected 'GH_AW_PROMPT: /tmp/gh-aw/aw-prompts/prompt.txt' in prompt generation step")
	}

	// Check that env section is always present now
	if !strings.Contains(output, "env:") {
		t.Error("Expected 'env:' section in prompt generation step")
	}
}

func TestMissingToolPromptGeneration(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	// Create workflow data with missing-tool enabled
	data := &WorkflowData{
		MarkdownContent: "Test workflow content",
		SafeOutputs: &SafeOutputsConfig{
			MissingTool: &MissingToolConfig{BaseSafeOutputConfig: BaseSafeOutputConfig{Max: 10}},
		},
	}

	var yaml strings.Builder
	compiler.generatePrompt(&yaml, data)

	output := yaml.String()

	// Check that GH_AW_SAFE_OUTPUTS environment variable is included when SafeOutputs is configured
	// This is how safe outputs tools are now discovered (via MCP server tool discovery)
	if !strings.Contains(output, "GH_AW_SAFE_OUTPUTS: ${{ env.GH_AW_SAFE_OUTPUTS }}") {
		t.Error("Expected 'GH_AW_SAFE_OUTPUTS' environment variable when SafeOutputs is configured")
	}
}

func TestMissingToolNotEnabledByDefault(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	// Test with completely empty frontmatter
	emptyFrontmatter := map[string]any{}
	safeOutputs := compiler.extractSafeOutputsConfig(emptyFrontmatter)

	if safeOutputs != nil && safeOutputs.MissingTool != nil {
		t.Error("Expected MissingTool to not be enabled by default with empty frontmatter")
	}

	// Test with frontmatter that has other content but no safe-outputs
	frontmatterWithoutSafeOutputs := map[string]any{
		"name": "Test Workflow",
		"on":   map[string]any{"workflow_dispatch": nil},
	}
	safeOutputs = compiler.extractSafeOutputsConfig(frontmatterWithoutSafeOutputs)

	if safeOutputs != nil && safeOutputs.MissingTool != nil {
		t.Error("Expected MissingTool to not be enabled by default without safe-outputs section")
	}
}

func TestMissingToolConfigParsing(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name        string
		configData  map[string]any
		expectMax   int
		expectError bool
	}{
		{
			name:       "Empty config",
			configData: map[string]any{"missing-tool": nil},
			expectMax:  0,
		},
		{
			name: "Config with max as int",
			configData: map[string]any{
				"missing-tool": map[string]any{"max": 5},
			},
			expectMax: 5,
		},
		{
			name: "Config with max as float64 (from YAML)",
			configData: map[string]any{
				"missing-tool": map[string]any{"max": float64(10)},
			},
			expectMax: 10,
		},
		{
			name: "Config with max as int64",
			configData: map[string]any{
				"missing-tool": map[string]any{"max": int64(15)},
			},
			expectMax: 15,
		},
		{
			name:       "No missing-tool key",
			configData: map[string]any{},
			expectMax:  -1, // Indicates nil config
		},
		{
			name: "Explicit false disables missing-tool",
			configData: map[string]any{
				"missing-tool": false,
			},
			expectMax: -1, // Indicates nil config (disabled)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := compiler.parseMissingToolConfig(tt.configData)

			if tt.expectMax == -1 {
				if config != nil {
					t.Error("Expected nil config when missing-tool key is absent or disabled")
				}
			} else {
				if config == nil {
					t.Fatal("Expected non-nil config")
				}
				if config.Max != tt.expectMax {
					t.Errorf("Expected max %d, got %d", tt.expectMax, config.Max)
				}
			}
		})
	}
}
