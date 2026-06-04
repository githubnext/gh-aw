//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
)

func TestExtractEngineConfig(t *testing.T) {
	compiler := NewCompiler()

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
			name: "top-level max-effective-tokens without engine",
			frontmatter: map[string]any{
				"max-effective-tokens": 10000000,
			},
			expectedEngineSetting: "",
			expectedConfig:        &EngineConfig{MaxEffectiveTokens: 10000000},
		},
		{
			name: "top-level max-runs without engine",
			frontmatter: map[string]any{
				"max-runs": 25,
			},
			expectedEngineSetting: "",
			expectedConfig:        &EngineConfig{MaxRuns: 25},
		},
		{
			name: "top-level max-turns without engine",
			frontmatter: map[string]any{
				"max-turns": 25,
			},
			expectedEngineSetting: "",
			expectedConfig:        &EngineConfig{MaxTurns: "25"},
		},
		{
			name: "top-level max-turns expression without engine",
			frontmatter: map[string]any{
				"max-turns": "${{ inputs.max-turns }}",
			},
			expectedEngineSetting: "",
			expectedConfig:        &EngineConfig{MaxTurns: "${{ inputs.max-turns }}"},
		},
		{
			name: "top-level max-turns zero is ignored",
			frontmatter: map[string]any{
				"max-turns": 0,
			},
			expectedEngineSetting: "",
			expectedConfig:        nil,
		},
		{
			name: "top-level max-turns negative is ignored",
			frontmatter: map[string]any{
				"max-turns": -1,
			},
			expectedEngineSetting: "",
			expectedConfig:        nil,
		},
		{
			name: "top-level negative max-effective-tokens disables budget and steering",
			frontmatter: map[string]any{
				"max-effective-tokens": -1,
			},
			expectedEngineSetting: "",
			expectedConfig:        &EngineConfig{MaxEffectiveTokens: -1},
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
			name: "object format - with expression version",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id":      "copilot",
					"version": "${{ inputs.engine-version }}",
				},
			},
			expectedEngineSetting: "copilot",
			expectedConfig:        &EngineConfig{ID: "copilot", Version: "${{ inputs.engine-version }}"},
		},
		{
			name: "object format - with integer version",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id":      "copilot",
					"version": 20,
				},
			},
			expectedEngineSetting: "copilot",
			expectedConfig:        &EngineConfig{ID: "copilot", Version: "20"},
		},
		{
			name: "object format - with float version",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id":      "claude",
					"version": 3.11,
				},
			},
			expectedEngineSetting: "claude",
			expectedConfig:        &EngineConfig{ID: "claude", Version: "3.11"},
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
			name: "object format - with top-level max-effective-tokens",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id": "claude",
				},
				"max-effective-tokens": 10000000,
			},
			expectedEngineSetting: "claude",
			expectedConfig:        &EngineConfig{ID: "claude", MaxEffectiveTokens: 10000000},
		},
		{
			name: "object format - with top-level max-runs",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id": "claude",
				},
				"max-runs": 12,
			},
			expectedEngineSetting: "claude",
			expectedConfig:        &EngineConfig{ID: "claude", MaxRuns: 12},
		},
		{
			name: "object format - with top-level max-turns",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id": "codex",
				},
				"max-turns": 12,
			},
			expectedEngineSetting: "codex",
			expectedConfig:        &EngineConfig{ID: "codex", MaxTurns: "12"},
		},
		{
			name: "object format - top-level max-turns overrides engine max-turns",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id":        "codex",
					"max-turns": 3,
				},
				"max-turns": "${{ inputs.max-turns }}",
			},
			expectedEngineSetting: "codex",
			expectedConfig:        &EngineConfig{ID: "codex", MaxTurns: "${{ inputs.max-turns }}"},
		},
		{
			name: "object format - with top-level max-runs as string",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id": "claude",
				},
				"max-runs": "12",
			},
			expectedEngineSetting: "claude",
			expectedConfig:        &EngineConfig{ID: "claude", MaxRuns: 12},
		},
		{
			name: "object format - with top-level max-effective-tokens as string",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id": "claude",
				},
				"max-effective-tokens": "10000000",
			},
			expectedEngineSetting: "claude",
			expectedConfig:        &EngineConfig{ID: "claude", MaxEffectiveTokens: 10000000},
		},
		{
			name: "object format - with top-level max-effective-tokens as suffix string",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id": "claude",
				},
				"max-effective-tokens": "100M",
			},
			expectedEngineSetting: "claude",
			expectedConfig:        &EngineConfig{ID: "claude", MaxEffectiveTokens: 100000000},
		},
		{
			name: "object format - with top-level negative max-effective-tokens",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id": "claude",
				},
				"max-effective-tokens": -1,
			},
			expectedEngineSetting: "claude",
			expectedConfig:        &EngineConfig{ID: "claude", MaxEffectiveTokens: -1},
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
			// float64 is what json.Unmarshal produces for numbers when deserializing engine
			// config JSON from shared imports (JSON roundtrip: YAML int -> JSON -> Go float64)
			name: "object format - with max-turns as float64 (JSON roundtrip from shared import)",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id":        "claude",
					"max-turns": float64(100),
				},
			},
			expectedEngineSetting: "claude",
			expectedConfig:        &EngineConfig{ID: "claude", MaxTurns: "100"},
		},
		{
			name: "object format - with env vars",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id": "claude",
					"env": map[string]any{
						"CUSTOM_VAR":  "value1",
						"ANOTHER_VAR": "${{ secrets.SECRET_VAR }}",
					},
				},
			},
			expectedEngineSetting: "claude",
			expectedConfig:        &EngineConfig{ID: "claude", Env: map[string]string{"CUSTOM_VAR": "value1", "ANOTHER_VAR": "${{ secrets.SECRET_VAR }}"}},
		},
		{
			name: "object format - complete with env vars",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id":        "claude",
					"version":   "beta",
					"model":     "claude-3-5-sonnet-20241022",
					"max-turns": 5,
					"env": map[string]any{
						"AWS_REGION":   "us-west-2",
						"API_ENDPOINT": "https://api.example.com",
					},
				},
			},
			expectedEngineSetting: "claude",
			expectedConfig:        &EngineConfig{ID: "claude", Version: "beta", Model: "claude-3-5-sonnet-20241022", MaxTurns: "5", Env: map[string]string{"AWS_REGION": "us-west-2", "API_ENDPOINT": "https://api.example.com"}},
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
		{
			name: "object format - with user-agent (hyphen)",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id":         "codex",
					"user-agent": "my-custom-agent-hyphen",
				},
			},
			expectedEngineSetting: "codex",
			expectedConfig:        &EngineConfig{ID: "codex", UserAgent: "my-custom-agent-hyphen"},
		},
		{
			name: "object format - with copilot harness script",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id":      "copilot",
					"harness": "custom_copilot_harness.cjs",
				},
			},
			expectedEngineSetting: "copilot",
			expectedConfig:        &EngineConfig{ID: "copilot", HarnessScript: "custom_copilot_harness.cjs"},
		},
		{
			name: "object format - with copilot sdk driver script",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id":                 "copilot",
					"copilot-sdk-driver": "custom_copilot_sdk_driver.cjs",
				},
			},
			expectedEngineSetting: "copilot",
			expectedConfig:        &EngineConfig{ID: "copilot", CopilotSDKDriver: "custom_copilot_sdk_driver.cjs"},
		},
		{
			name: "object format - complete with user-agent",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id":         "codex",
					"version":    "beta",
					"model":      "gpt-4o",
					"max-turns":  3,
					"user-agent": "complete-custom-agent",
					"env": map[string]any{
						"CUSTOM_VAR": "value1",
					},
				},
			},
			expectedEngineSetting: "codex",
			expectedConfig:        &EngineConfig{ID: "codex", Version: "beta", Model: "gpt-4o", MaxTurns: "3", UserAgent: "complete-custom-agent", Env: map[string]string{"CUSTOM_VAR": "value1"}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			engineSetting, config := compiler.ExtractEngineConfig(test.frontmatter)

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

				if config.MaxEffectiveTokens != test.expectedConfig.MaxEffectiveTokens {
					t.Errorf("Expected config.MaxEffectiveTokens '%d', got '%d'", test.expectedConfig.MaxEffectiveTokens, config.MaxEffectiveTokens)
				}

				if config.MaxRuns != test.expectedConfig.MaxRuns {
					t.Errorf("Expected config.MaxRuns '%d', got '%d'", test.expectedConfig.MaxRuns, config.MaxRuns)
				}

				if config.UserAgent != test.expectedConfig.UserAgent {
					t.Errorf("Expected config.UserAgent '%s', got '%s'", test.expectedConfig.UserAgent, config.UserAgent)
				}

				if config.HarnessScript != test.expectedConfig.HarnessScript {
					t.Errorf("Expected config.HarnessScript '%s', got '%s'", test.expectedConfig.HarnessScript, config.HarnessScript)
				}

				if config.CopilotSDKDriver != test.expectedConfig.CopilotSDKDriver {
					t.Errorf("Expected config.CopilotSDKDriver '%s', got '%s'", test.expectedConfig.CopilotSDKDriver, config.CopilotSDKDriver)
				}

				if len(config.Env) != len(test.expectedConfig.Env) {
					t.Errorf("Expected config.Env length %d, got %d", len(test.expectedConfig.Env), len(config.Env))
				} else {
					for key, expectedValue := range test.expectedConfig.Env {
						if actualValue, exists := config.Env[key]; !exists {
							t.Errorf("Expected config.Env to contain key '%s'", key)
						} else if actualValue != expectedValue {
							t.Errorf("Expected config.Env['%s'] = '%s', got '%s'", key, expectedValue, actualValue)
						}
					}
				}

			}
		})
	}
}

func TestExtractEngineConfig_EngineAuthMapsToAWFEnv(t *testing.T) {
	compiler := NewCompiler()
	_, config := compiler.ExtractEngineConfig(map[string]any{
		"engine": map[string]any{
			"id": "copilot",
			"auth": map[string]any{
				"type":            "github-oidc",
				"audience":        "https://cognitiveservices.azure.com",
				"azure-tenant-id": "tenant-id",
				"azure-client-id": "client-id",
				"azure-scope":     "https://cognitiveservices.azure.com/.default",
				"azure-cloud":     "public",
			},
		},
	})

	assert.NotNil(t, config)
	if assert.NotNil(t, config.Auth) {
		assert.Equal(t, "github-oidc", config.Auth.Type)
		assert.Equal(t, "https://cognitiveservices.azure.com", config.Auth.Audience)
		assert.Equal(t, "tenant-id", config.Auth.AzureTenantID)
		assert.Equal(t, "client-id", config.Auth.AzureClientID)
		assert.Equal(t, "https://cognitiveservices.azure.com/.default", config.Auth.AzureScope)
		assert.Equal(t, "public", config.Auth.AzureCloud)
	}

	assert.Equal(t, "github-oidc", config.Env["AWF_AUTH_TYPE"])
	assert.Equal(t, "https://cognitiveservices.azure.com", config.Env["AWF_AUTH_OIDC_AUDIENCE"])
	assert.Equal(t, "tenant-id", config.Env["AWF_AUTH_AZURE_TENANT_ID"])
	assert.Equal(t, "client-id", config.Env["AWF_AUTH_AZURE_CLIENT_ID"])
	assert.Equal(t, "https://cognitiveservices.azure.com/.default", config.Env["AWF_AUTH_AZURE_SCOPE"])
	assert.Equal(t, "public", config.Env["AWF_AUTH_AZURE_CLOUD"])
}

func TestExtractEngineConfig_EngineEnvTakesPrecedenceOverEngineAuth(t *testing.T) {
	compiler := NewCompiler()
	_, config := compiler.ExtractEngineConfig(map[string]any{
		"engine": map[string]any{
			"id": "copilot",
			"env": map[string]any{
				"AWF_AUTH_TYPE":          "static",
				"AWF_AUTH_OIDC_AUDIENCE": "from-engine-env",
			},
			"auth": map[string]any{
				"type":     "github-oidc",
				"audience": "from-engine-auth",
			},
		},
	})

	assert.NotNil(t, config)
	assert.Equal(t, "static", config.Env["AWF_AUTH_TYPE"])
	assert.Equal(t, "from-engine-env", config.Env["AWF_AUTH_OIDC_AUDIENCE"])
}

func TestExtractEngineConfig_AnthropicWIFMapsToAWFEnv(t *testing.T) {
	compiler := NewCompiler()
	_, config := compiler.ExtractEngineConfig(map[string]any{
		"engine": map[string]any{
			"id": "claude",
			"auth": map[string]any{
				"type":               "github-oidc",
				"provider":           "anthropic",
				"federation-rule-id": "fr_01ABC",
				"organization-id":    "org_01XYZ",
				"service-account-id": "sa_01DEF",
				"workspace-id":       "ws_01GHI",
			},
		},
	})

	assert.NotNil(t, config)
	if assert.NotNil(t, config.Auth) {
		assert.Equal(t, "github-oidc", config.Auth.Type)
		assert.Equal(t, "anthropic", config.Auth.Provider)
		assert.Equal(t, "fr_01ABC", config.Auth.AnthropicFederationRuleID)
		assert.Equal(t, "org_01XYZ", config.Auth.AnthropicOrganizationID)
		assert.Equal(t, "sa_01DEF", config.Auth.AnthropicServiceAccountID)
		assert.Equal(t, "ws_01GHI", config.Auth.AnthropicWorkspaceID)
	}

	assert.Equal(t, "github-oidc", config.Env["AWF_AUTH_TYPE"])
	assert.Equal(t, "anthropic", config.Env["AWF_AUTH_PROVIDER"])
	assert.Equal(t, "fr_01ABC", config.Env["AWF_AUTH_ANTHROPIC_FEDERATION_RULE_ID"])
	assert.Equal(t, "org_01XYZ", config.Env["AWF_AUTH_ANTHROPIC_ORGANIZATION_ID"])
	assert.Equal(t, "sa_01DEF", config.Env["AWF_AUTH_ANTHROPIC_SERVICE_ACCOUNT_ID"])
	assert.Equal(t, "ws_01GHI", config.Env["AWF_AUTH_ANTHROPIC_WORKSPACE_ID"])
}

func TestCompileWorkflowWithExtendedEngine(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "extended-engine-test")

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
  issues: read
  pull-requests: read
engine: claude
strict: false
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
  issues: read
  pull-requests: read
strict: false
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
  issues: read
  pull-requests: read
strict: false
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

			compiler := NewCompiler()
			workflowData, err := compiler.ParseWorkflowFile(testFile)
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
		engine         CodingAgentEngine
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
			workflowData := &WorkflowData{
				Name:         "test-workflow",
				EngineConfig: tt.engineConfig,
			}
			steps := tt.engine.GetExecutionSteps(workflowData, "test-log")

			if len(steps) == 0 {
				t.Fatalf("Expected at least one step, got none")
			}

			// Convert first step to YAML string for testing
			stepContent := strings.Join([]string(steps[0]), "\n")

			switch tt.engine.GetID() {
			case "claude":
				if tt.expectedModel != "" {
					// Claude passes model via native ANTHROPIC_MODEL env var
					expectedEnvLine := "ANTHROPIC_MODEL: " + tt.expectedModel
					if !strings.Contains(stepContent, expectedEnvLine) {
						t.Errorf("Expected step to contain env var for model %s, got step content:\n%s", tt.expectedModel, stepContent)
					}
					// Should NOT embed --model in the shell command
					if strings.Contains(stepContent, "--model "+tt.expectedModel) {
						t.Errorf("Model should not be embedded as --model flag, got step content:\n%s", stepContent)
					}
				}

			case "codex":
				if tt.expectedModel != "" {
					// Codex passes model via GH_AW_MODEL_*_CODEX with shell expansion
					// The workflow has no SafeOutputs, so it uses the detection env var
					expectedEnvLine := constants.EnvVarModelDetectionCodex + ": " + tt.expectedModel
					if !strings.Contains(stepContent, expectedEnvLine) {
						t.Errorf("Expected step to contain env var for model %s, got step content:\n%s", tt.expectedModel, stepContent)
					}
				}
			}
		})
	}
}

func TestEngineConfigurationWithCustomEnvVars(t *testing.T) {
	tests := []struct {
		name         string
		engine       CodingAgentEngine
		engineConfig *EngineConfig
		hasOutput    bool
	}{
		{
			name:   "Claude with custom env vars",
			engine: NewClaudeEngine(),
			engineConfig: &EngineConfig{
				ID:  "claude",
				Env: map[string]string{"AWS_REGION": "us-west-2", "CUSTOM_VAR": "${{ secrets.MY_SECRET }}"},
			},
			hasOutput: false,
		},
		{
			name:   "Claude with custom env vars and output",
			engine: NewClaudeEngine(),
			engineConfig: &EngineConfig{
				ID:  "claude",
				Env: map[string]string{"API_ENDPOINT": "https://api.example.com", "DEBUG_MODE": "true"},
			},
			hasOutput: true,
		},
		{
			name:   "Codex with custom env vars",
			engine: NewCodexEngine(),
			engineConfig: &EngineConfig{
				ID:  "codex",
				Env: map[string]string{"CUSTOM_API_KEY": "test123", "PROXY_URL": "http://proxy.example.com"},
			},
			hasOutput: false,
		},
		{
			name:   "Codex with custom env vars and output",
			engine: NewCodexEngine(),
			engineConfig: &EngineConfig{
				ID:  "codex",
				Env: map[string]string{"ENVIRONMENT": "production", "LOG_LEVEL": "debug"},
			},
			hasOutput: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflowData := &WorkflowData{
				Name:         "test-workflow",
				EngineConfig: tt.engineConfig,
			}
			if tt.hasOutput {
				workflowData.SafeOutputs = &SafeOutputsConfig{}
			}
			steps := tt.engine.GetExecutionSteps(workflowData, "test-log")

			if len(steps) == 0 {
				t.Fatalf("Expected at least one step, got none")
			}

			// Convert first step to YAML string for testing
			stepContent := strings.Join([]string(steps[0]), "\n")

			switch tt.engine.GetID() {
			case "claude":
				// For Claude, custom env vars should be in claude_env input
				if tt.engineConfig != nil && len(tt.engineConfig.Env) > 0 {
					foundEnvVar := false
					for key, value := range tt.engineConfig.Env {
						if strings.Contains(stepContent, key+":") && strings.Contains(stepContent, value) {
							foundEnvVar = true
							break
						}
					}
					if !foundEnvVar {
						t.Errorf("Expected step to contain custom environment variables, got step content:\n%s", stepContent)
					}
				}

			case "codex":
				// For Codex, custom env vars should be in the step's env section
				if tt.engineConfig != nil && len(tt.engineConfig.Env) > 0 {
					foundEnvVar := false
					for key, expectedValue := range tt.engineConfig.Env {
						envLine := key + ": " + expectedValue
						if strings.Contains(stepContent, envLine) {
							foundEnvVar = true
							break
						}
					}
					if !foundEnvVar {
						t.Errorf("Expected step to contain custom environment variables, got step content:\n%s", stepContent)
					}
				}
			}
		})
	}
}

func TestNilEngineConfig(t *testing.T) {
	engines := []CodingAgentEngine{
		NewClaudeEngine(),
		NewCodexEngine(),
	}

	for _, engine := range engines {
		t.Run(engine.GetID(), func(t *testing.T) {
			// Should not panic when engineConfig is nil
			workflowData := &WorkflowData{
				Name: "test-workflow",
			}
			steps := engine.GetExecutionSteps(workflowData, "test-log")

			// Engines should return at least one step
			if len(steps) == 0 {
				t.Errorf("Expected at least one step for engine %s, got none", engine.GetID())
			}

			// Check that the first step has some content
			if len(steps) > 0 && len(steps[0]) == 0 {
				t.Errorf("Expected non-empty step content for engine %s", engine.GetID())
			}
		})
	}
}

func TestEngineBareFieldExtraction(t *testing.T) {
	compiler := NewCompiler()

	tests := []struct {
		name         string
		frontmatter  map[string]any
		expectedBare bool
	}{
		{
			name: "bare true",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id":   "copilot",
					"bare": true,
				},
			},
			expectedBare: true,
		},
		{
			name: "bare false",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id":   "copilot",
					"bare": false,
				},
			},
			expectedBare: false,
		},
		{
			name: "bare not set (default false)",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id": "copilot",
				},
			},
			expectedBare: false,
		},
		{
			name: "bare true for claude",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id":   "claude",
					"bare": true,
				},
			},
			expectedBare: true,
		},
		{
			name: "bare true for codex",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id":   "codex",
					"bare": true,
				},
			},
			expectedBare: true,
		},
		{
			name: "bare true for gemini",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id":   "gemini",
					"bare": true,
				},
			},
			expectedBare: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, config := compiler.ExtractEngineConfig(tt.frontmatter)
			if config == nil {
				t.Fatal("Expected config to be non-nil")
			}
			if config.Bare != tt.expectedBare {
				t.Errorf("Expected Bare=%v, got Bare=%v", tt.expectedBare, config.Bare)
			}
		})
	}
}

func TestEngineBareModeCopilotArgs(t *testing.T) {
	workflowData := &WorkflowData{
		Name: "test-workflow",
		EngineConfig: &EngineConfig{
			ID:   "copilot",
			Bare: true,
		},
	}

	engine := NewCopilotEngine()
	steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")

	var foundFlag bool
	for _, step := range steps {
		for _, line := range step {
			if strings.Contains(line, "--no-custom-instructions") {
				foundFlag = true
				break
			}
		}
	}
	if !foundFlag {
		t.Error("Expected --no-custom-instructions in copilot execution steps when bare=true")
	}
}

func TestEngineBareModeCopilotArgsNotPresent(t *testing.T) {
	workflowData := &WorkflowData{
		Name: "test-workflow",
		EngineConfig: &EngineConfig{
			ID:   "copilot",
			Bare: false,
		},
	}

	engine := NewCopilotEngine()
	steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")

	for _, step := range steps {
		for _, line := range step {
			if strings.Contains(line, "--no-custom-instructions") {
				t.Error("Expected --no-custom-instructions to be absent when bare=false")
				return
			}
		}
	}
}

func TestEngineBareModeClaude(t *testing.T) {
	workflowData := &WorkflowData{
		Name: "test-workflow",
		EngineConfig: &EngineConfig{
			ID:   "claude",
			Bare: true,
		},
	}

	engine := NewClaudeEngine()
	steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")

	var foundFlag bool
	for _, step := range steps {
		for _, line := range step {
			if strings.Contains(line, "--bare") {
				foundFlag = true
				break
			}
		}
	}
	if !foundFlag {
		t.Error("Expected --bare in claude execution steps when bare=true")
	}
}

func TestEngineBareModeClaude_NotPresent(t *testing.T) {
	workflowData := &WorkflowData{
		Name: "test-workflow",
		EngineConfig: &EngineConfig{
			ID:   "claude",
			Bare: false,
		},
	}

	engine := NewClaudeEngine()
	steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")

	for _, step := range steps {
		for _, line := range step {
			if strings.Contains(line, "--bare") {
				t.Error("Expected --bare to be absent in claude execution steps when bare=false")
				return
			}
		}
	}
}

func TestSupportsBareMode(t *testing.T) {
	tests := []struct {
		name     string
		engine   CodingAgentEngine
		expected bool
	}{
		{
			name:     "copilot supports bare mode",
			engine:   NewCopilotEngine(),
			expected: true,
		},
		{
			name:     "claude supports bare mode",
			engine:   NewClaudeEngine(),
			expected: true,
		},
		{
			name:     "codex does not support bare mode",
			engine:   NewCodexEngine(),
			expected: false,
		},
		{
			name:     "gemini does not support bare mode",
			engine:   NewGeminiEngine(),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.engine.GetCapabilities().BareMode,
				"BareMode capability should be %v for %s", tt.expected, tt.engine.GetID())
		})
	}
}

// TestBareMode_UnsupportedEngineNoFlag verifies that engines not supporting bare mode
// do not inject any bare-mode flags in their execution steps.
func TestBareMode_UnsupportedEngineNoFlag(t *testing.T) {
	t.Run("codex does not inject --no-system-prompt", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID:   "codex",
				Bare: true,
			},
		}

		engine := NewCodexEngine()
		steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")

		for _, step := range steps {
			for _, line := range step {
				assert.NotContains(t, line, "--no-system-prompt",
					"Codex should not inject --no-system-prompt (bare mode unsupported)")
			}
		}
	})

	t.Run("gemini does not inject GEMINI_SYSTEM_MD=/dev/null", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID:   "gemini",
				Bare: true,
			},
		}

		engine := NewGeminiEngine()
		steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")

		for _, step := range steps {
			for _, line := range step {
				if strings.Contains(line, "GEMINI_SYSTEM_MD") && strings.Contains(line, "/dev/null") {
					t.Error("Gemini should not inject GEMINI_SYSTEM_MD=/dev/null (bare mode unsupported)")
					return
				}
			}
		}
	})
}

// TestEngineMCPSessionTimeoutExtraction tests extraction of engine.mcp.session-timeout.
func TestEngineMCPSessionTimeoutExtraction(t *testing.T) {
	compiler := NewCompiler()

	tests := []struct {
		name            string
		frontmatter     map[string]any
		expectedTimeout string
	}{
		{
			name: "extracts session-timeout from engine.mcp",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id": "copilot",
					"mcp": map[string]any{
						"session-timeout": "4h",
					},
				},
			},
			expectedTimeout: "4h",
		},
		{
			name: "no mcp section - empty session timeout",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id": "copilot",
				},
			},
			expectedTimeout: "",
		},
		{
			name: "mcp section without session-timeout - empty",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id":  "copilot",
					"mcp": map[string]any{},
				},
			},
			expectedTimeout: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, config := compiler.ExtractEngineConfig(tt.frontmatter)
			if config == nil {
				t.Fatal("Expected non-nil config")
			}
			if config.MCPSessionTimeout != tt.expectedTimeout {
				t.Errorf("MCPSessionTimeout = %q, want %q", config.MCPSessionTimeout, tt.expectedTimeout)
			}
		})
	}
}

// TestEngineMCPToolTimeoutExtraction tests extraction of engine.mcp.tool-timeout.
func TestEngineMCPToolTimeoutExtraction(t *testing.T) {
	compiler := NewCompiler()

	tests := []struct {
		name            string
		frontmatter     map[string]any
		expectedTimeout string
	}{
		{
			name: "extracts tool-timeout from engine.mcp",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id": "copilot",
					"mcp": map[string]any{
						"tool-timeout": "2m",
					},
				},
			},
			expectedTimeout: "2m",
		},
		{
			name: "no mcp section - empty tool timeout",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id": "copilot",
				},
			},
			expectedTimeout: "",
		},
		{
			name: "mcp section without tool-timeout - empty",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id":  "copilot",
					"mcp": map[string]any{},
				},
			},
			expectedTimeout: "",
		},
		{
			name: "mcp section with both session-timeout and tool-timeout",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id": "copilot",
					"mcp": map[string]any{
						"session-timeout": "4h",
						"tool-timeout":    "5m",
					},
				},
			},
			expectedTimeout: "5m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, config := compiler.ExtractEngineConfig(tt.frontmatter)
			if config == nil {
				t.Fatal("Expected non-nil config")
			}
			if config.MCPToolTimeout != tt.expectedTimeout {
				t.Errorf("MCPToolTimeout = %q, want %q", config.MCPToolTimeout, tt.expectedTimeout)
			}
		})
	}
}
