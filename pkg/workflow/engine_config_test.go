package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractEngineConfig(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name                  string
		frontmatter           map[string]any
		expectedEngineSetting string
		expectedConfig        *EngineConfig
	}{
		{
			name:                  "no engine specified",
			frontmatter:           map[string]any{},
			expectedEngineSetting: "",
			expectedConfig:        nil,
		},
		{
			name:                  "string format - claude",
			frontmatter:           map[string]any{"engine": "claude"},
			expectedEngineSetting: "claude",
			expectedConfig:        &EngineConfig{ID: "claude"},
		},
		{
			name:                  "string format - codex",
			frontmatter:           map[string]any{"engine": "codex"},
			expectedEngineSetting: "codex",
			expectedConfig:        &EngineConfig{ID: "codex"},
		},
		{
			name: "object format - minimal (id only)",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id": "claude",
				},
			},
			expectedEngineSetting: "claude",
			expectedConfig:        &EngineConfig{ID: "claude"},
		},
		{
			name: "object format - with version",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id":      "claude",
					"version": "beta",
				},
			},
			expectedEngineSetting: "claude",
			expectedConfig:        &EngineConfig{ID: "claude", Version: "beta"},
		},
		{
			name: "object format - with model",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id":    "codex",
					"model": "gpt-4o",
				},
			},
			expectedEngineSetting: "codex",
			expectedConfig:        &EngineConfig{ID: "codex", Model: "gpt-4o"},
		},
		{
			name: "object format - complete",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id":      "claude",
					"version": "beta",
					"model":   "claude-3-5-sonnet-20241022",
				},
			},
			expectedEngineSetting: "claude",
			expectedConfig:        &EngineConfig{ID: "claude", Version: "beta", Model: "claude-3-5-sonnet-20241022"},
		},
		{
			name: "object format - with max-turns",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id":        "claude",
					"max-turns": 5,
				},
			},
			expectedEngineSetting: "claude",
			expectedConfig:        &EngineConfig{ID: "claude", MaxTurns: "5"},
		},
		{
			name: "object format - complete with max-turns",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id":        "claude",
					"version":   "beta",
					"model":     "claude-3-5-sonnet-20241022",
					"max-turns": 10,
				},
			},
			expectedEngineSetting: "claude",
			expectedConfig:        &EngineConfig{ID: "claude", Version: "beta", Model: "claude-3-5-sonnet-20241022", MaxTurns: "10"},
		},
		{
			name: "object format - missing id",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"version": "beta",
					"model":   "gpt-4o",
				},
			},
			expectedEngineSetting: "",
			expectedConfig:        &EngineConfig{Version: "beta", Model: "gpt-4o"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			engineSetting, config := compiler.extractEngineConfig(test.frontmatter)

			if engineSetting != test.expectedEngineSetting {
				t.Errorf("Expected engineSetting '%s', got '%s'", test.expectedEngineSetting, engineSetting)
			}

			if test.expectedConfig == nil {
				if config != nil {
					t.Errorf("Expected nil config, got %+v", config)
				}
			} else {
				if config == nil {
					t.Errorf("Expected config %+v, got nil", test.expectedConfig)
					return
				}

				if config.ID != test.expectedConfig.ID {
					t.Errorf("Expected config.ID '%s', got '%s'", test.expectedConfig.ID, config.ID)
				}

				if config.Version != test.expectedConfig.Version {
					t.Errorf("Expected config.Version '%s', got '%s'", test.expectedConfig.Version, config.Version)
				}

				if config.Model != test.expectedConfig.Model {
					t.Errorf("Expected config.Model '%s', got '%s'", test.expectedConfig.Model, config.Model)
				}

				if config.MaxTurns != test.expectedConfig.MaxTurns {
					t.Errorf("Expected config.MaxTurns '%s', got '%s'", test.expectedConfig.MaxTurns, config.MaxTurns)
				}
			}
		})
	}
}

func TestCompileWorkflowWithExtendedEngine(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "extended-engine-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name           string
		content        string
		expectedAI     string
		expectedConfig *EngineConfig
	}{
		{
			name: "string engine format",
			content: `---
on: push
permissions:
  contents: read
  issues: write
engine: claude
---

# Test Workflow

This is a test workflow.`,
			expectedAI:     "claude",
			expectedConfig: &EngineConfig{ID: "claude"},
		},
		{
			name: "object engine format - complete",
			content: `---
on: push
permissions:
  contents: read
  issues: write
engine:
  id: claude
  version: beta
  model: claude-3-5-sonnet-20241022
---

# Test Workflow

This is a test workflow.`,
			expectedAI:     "claude",
			expectedConfig: &EngineConfig{ID: "claude", Version: "beta", Model: "claude-3-5-sonnet-20241022"},
		},
		{
			name: "object engine format - codex with model",
			content: `---
on: push
permissions:
  contents: read
  issues: write
engine:
  id: codex
  model: gpt-4o
---

# Test Workflow

This is a test workflow.`,
			expectedAI:     "codex",
			expectedConfig: &EngineConfig{ID: "codex", Model: "gpt-4o"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testFile := filepath.Join(tmpDir, "test-workflow.md")
			if err := os.WriteFile(testFile, []byte(test.content), 0644); err != nil {
				t.Fatal(err)
			}

			compiler := NewCompiler(false, "", "test")
			workflowData, err := compiler.parseWorkflowFile(testFile)
			if err != nil {
				t.Fatalf("Failed to parse workflow: %v", err)
			}

			// Check AI field (backwards compatibility)
			if workflowData.AI != test.expectedAI {
				t.Errorf("Expected AI '%s', got '%s'", test.expectedAI, workflowData.AI)
			}

			// Check EngineConfig
			if test.expectedConfig == nil {
				if workflowData.EngineConfig != nil {
					t.Errorf("Expected nil EngineConfig, got %+v", workflowData.EngineConfig)
				}
			} else {
				if workflowData.EngineConfig == nil {
					t.Errorf("Expected EngineConfig %+v, got nil", test.expectedConfig)
					return
				}

				if workflowData.EngineConfig.ID != test.expectedConfig.ID {
					t.Errorf("Expected EngineConfig.ID '%s', got '%s'", test.expectedConfig.ID, workflowData.EngineConfig.ID)
				}

				if workflowData.EngineConfig.Version != test.expectedConfig.Version {
					t.Errorf("Expected EngineConfig.Version '%s', got '%s'", test.expectedConfig.Version, workflowData.EngineConfig.Version)
				}

				if workflowData.EngineConfig.Model != test.expectedConfig.Model {
					t.Errorf("Expected EngineConfig.Model '%s', got '%s'", test.expectedConfig.Model, workflowData.EngineConfig.Model)
				}
			}
		})
	}
}

func TestEngineConfigurationWithModel(t *testing.T) {
	tests := []struct {
		name           string
		engine         AgenticEngine
		engineConfig   *EngineConfig
		expectedModel  string
		expectedAPIKey string
	}{
		{
			name:   "Claude with model",
			engine: NewClaudeEngine(),
			engineConfig: &EngineConfig{
				ID:    "claude",
				Model: "claude-3-5-sonnet-20241022",
			},
			expectedModel:  "claude-3-5-sonnet-20241022",
			expectedAPIKey: "",
		},
		{
			name:   "Codex with model",
			engine: NewCodexEngine(),
			engineConfig: &EngineConfig{
				ID:    "codex",
				Model: "gpt-4o",
			},
			expectedModel:  "gpt-4o",
			expectedAPIKey: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.engine.GetExecutionConfig("test-workflow", "test-log", tt.engineConfig, false)

			switch tt.engine.GetID() {
			case "claude":
				if tt.expectedModel != "" {
					if config.Inputs["model"] != tt.expectedModel {
						t.Errorf("Expected model input to be %s, got: %s", tt.expectedModel, config.Inputs["model"])
					}
				}

			case "codex":
				if tt.expectedModel != "" {
					expectedModelArg := "model=" + tt.expectedModel
					if !strings.Contains(config.Command, expectedModelArg) {
						t.Errorf("Expected command to contain %s, got: %s", expectedModelArg, config.Command)
					}
				}
			}
		})
	}
}

func TestNilEngineConfig(t *testing.T) {
	engines := []AgenticEngine{
		NewClaudeEngine(),
		NewCodexEngine(),
	}

	for _, engine := range engines {
		t.Run(engine.GetID(), func(t *testing.T) {
			// Should not panic when engineConfig is nil
			config := engine.GetExecutionConfig("test-workflow", "test-log", nil, false)

			if config.StepName == "" {
				t.Errorf("Expected non-empty step name for engine %s", engine.GetID())
			}
		})
	}
}
