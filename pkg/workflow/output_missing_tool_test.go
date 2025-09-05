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
						if len(job.Depends) != 1 || job.Depends[0] != "main-job" {
							t.Errorf("Expected job to depend on 'main-job', got %v", job.Depends)
						}
					}
				}
			}
		})
	}
}

func TestMissingToolPromptGeneration(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	// Create workflow data with missing-tool enabled
	data := &WorkflowData{
		MarkdownContent: "Test workflow content",
		SafeOutputs: &SafeOutputsConfig{
			MissingTool: &MissingToolConfig{Max: 10},
		},
	}

	var yaml strings.Builder
	compiler.generatePrompt(&yaml, data, &ClaudeEngine{})

	output := yaml.String()

	// Check that missing-tool is mentioned in the header
	if !strings.Contains(output, "Reporting Missing Tools or Functionality") {
		t.Error("Expected 'Reporting Missing Tools or Functionality' in prompt header")
	}

	// Check that missing-tool instructions are present
	if !strings.Contains(output, "**Reporting Missing Tools or Functionality**") {
		t.Error("Expected missing-tool instructions section")
	}

	// Check for JSON format example
	if !strings.Contains(output, `"type": "missing-tool"`) {
		t.Error("Expected missing-tool JSON example")
	}

	// Check for required fields documentation
	if !strings.Contains(output, `"tool":`) {
		t.Error("Expected tool field documentation")
	}
	if !strings.Contains(output, `"reason":`) {
		t.Error("Expected reason field documentation")
	}
	if !strings.Contains(output, `"alternatives":`) {
		t.Error("Expected alternatives field documentation")
	}

	// Check that the example is included in JSONL examples
	if !strings.Contains(output, `{"type": "missing-tool", "tool": "docker"`) {
		t.Error("Expected missing-tool example in JSONL section")
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := compiler.parseMissingToolConfig(tt.configData)

			if tt.expectMax == -1 {
				if config != nil {
					t.Error("Expected nil config when missing-tool key is absent")
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

func TestMissingToolScriptEmbedding(t *testing.T) {
	// Test that the missing tool script is properly embedded
	if strings.TrimSpace(missingToolScript) == "" {
		t.Error("missingToolScript should not be empty")
	}

	// Verify it contains expected JavaScript content
	expectedContent := []string{
		"async function main()",
		"GITHUB_AW_AGENT_OUTPUT",
		"GITHUB_AW_MISSING_TOOL_MAX",
		"missing-tool",
		"JSON.parse",
		"core.setOutput",
		"tools_reported",
		"total_count",
	}

	for _, content := range expectedContent {
		if !strings.Contains(missingToolScript, content) {
			t.Errorf("Missing expected content in script: %s", content)
		}
	}

	// Verify it handles JSON format
	if !strings.Contains(missingToolScript, "JSON.parse") {
		t.Error("Script should handle JSON format")
	}
}
