//go:build !integration

package workflow

import (
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
)

// TestExtractAPITargetHost tests the extractAPITargetHost function that extracts
// hostnames from custom API base URLs in engine.env
func TestExtractAPITargetHost(t *testing.T) {
	tests := []struct {
		name         string
		workflowData *WorkflowData
		envVar       string
		expected     string
	}{
		{
			name: "extracts hostname from HTTPS URL with path",
			workflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					Env: map[string]string{
						"OPENAI_BASE_URL": "https://llm-router.internal.example.com/v1",
					},
				},
			},
			envVar:   "OPENAI_BASE_URL",
			expected: "llm-router.internal.example.com",
		},
		{
			name: "extracts hostname from HTTP URL with port and path",
			workflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					Env: map[string]string{
						"ANTHROPIC_BASE_URL": "http://localhost:8080/v1",
					},
				},
			},
			envVar:   "ANTHROPIC_BASE_URL",
			expected: "localhost:8080",
		},
		{
			name: "handles hostname without protocol or path",
			workflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					Env: map[string]string{
						"OPENAI_BASE_URL": "api.openai.com",
					},
				},
			},
			envVar:   "OPENAI_BASE_URL",
			expected: "api.openai.com",
		},
		{
			name: "handles hostname with port but no protocol",
			workflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					Env: map[string]string{
						"OPENAI_BASE_URL": "localhost:8000",
					},
				},
			},
			envVar:   "OPENAI_BASE_URL",
			expected: "localhost:8000",
		},
		{
			name: "returns empty string when env var not set",
			workflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					Env: map[string]string{
						"OTHER_VAR": "value",
					},
				},
			},
			envVar:   "OPENAI_BASE_URL",
			expected: "",
		},
		{
			name: "returns empty string when engine config is nil",
			workflowData: &WorkflowData{
				EngineConfig: nil,
			},
			envVar:   "OPENAI_BASE_URL",
			expected: "",
		},
		{
			name:         "returns empty string when workflow data is nil",
			workflowData: nil,
			envVar:       "OPENAI_BASE_URL",
			expected:     "",
		},
		{
			name: "returns empty string for empty URL",
			workflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					Env: map[string]string{
						"OPENAI_BASE_URL": "",
					},
				},
			},
			envVar:   "OPENAI_BASE_URL",
			expected: "",
		},
		{
			name: "extracts Azure OpenAI endpoint hostname",
			workflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					Env: map[string]string{
						"OPENAI_BASE_URL": "https://my-resource.openai.azure.com/openai/deployments/gpt-4",
					},
				},
			},
			envVar:   "OPENAI_BASE_URL",
			expected: "my-resource.openai.azure.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractAPITargetHost(tt.workflowData, tt.envVar)
			assert.Equal(t, tt.expected, result, "Extracted hostname should match expected value")
		})
	}
}

// TestExtractAPITargetAuthHeader tests the extractAPITargetAuthHeader function that reads
// the custom auth header name from sandbox.agent.targets.<provider>.authHeader in frontmatter.
func TestExtractAPITargetAuthHeader(t *testing.T) {
	makeWorkflowData := func(provider, authHeader string) *WorkflowData {
		return &WorkflowData{
			SandboxConfig: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					Targets: map[string]*AgentAPIProxyTargetConfig{
						provider: {AuthHeader: authHeader},
					},
				},
			},
		}
	}

	t.Run("returns authHeader for openai provider", func(t *testing.T) {
		result := extractAPITargetAuthHeader(makeWorkflowData("openai", "api-key"), "openai")
		assert.Equal(t, "api-key", result)
	})

	t.Run("returns authHeader for anthropic provider", func(t *testing.T) {
		result := extractAPITargetAuthHeader(makeWorkflowData("anthropic", "x-custom-header"), "anthropic")
		assert.Equal(t, "x-custom-header", result)
	})

	t.Run("returns empty string when sandbox config is absent", func(t *testing.T) {
		wd := &WorkflowData{EngineConfig: &EngineConfig{ID: "codex"}}
		assert.Empty(t, extractAPITargetAuthHeader(wd, "openai"))
	})

	t.Run("returns empty string when provider is absent", func(t *testing.T) {
		wd := &WorkflowData{
			SandboxConfig: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					Targets: map[string]*AgentAPIProxyTargetConfig{},
				},
			},
		}
		assert.Empty(t, extractAPITargetAuthHeader(wd, "openai"))
	})

	t.Run("returns empty string for nil WorkflowData", func(t *testing.T) {
		assert.Empty(t, extractAPITargetAuthHeader(nil, "openai"))
	})

	t.Run("returns empty string when targets is nil", func(t *testing.T) {
		wd := &WorkflowData{
			SandboxConfig: &SandboxConfig{
				Agent: &AgentSandboxConfig{},
			},
		}
		assert.Empty(t, extractAPITargetAuthHeader(wd, "openai"))
	})
}

// TestExtractAPIBasePath tests the extractAPIBasePath function that extracts
// path components from custom API base URLs in engine.env
func TestExtractAPIBasePath(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{"databricks serving endpoint", "https://host.com/serving-endpoints", "/serving-endpoints"},
		{"azure openai deployment", "https://host.com/openai/deployments/gpt-4", "/openai/deployments/gpt-4"},
		{"simple path", "https://host.com/v1", "/v1"},
		{"trailing slash stripped", "https://host.com/api/", "/api"},
		{"multiple trailing slashes stripped", "https://host.com/api///", "/api"},
		{"no path", "https://host.com", ""},
		{"bare hostname", "host.com", ""},
		{"root path only", "https://host.com/", ""},
		{"query string stripped", "https://host.com/api?param=value", "/api"},
		{"fragment stripped", "https://host.com/api#section", "/api"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflowData := &WorkflowData{
				EngineConfig: &EngineConfig{
					Env: map[string]string{
						"OPENAI_BASE_URL": tt.url,
					},
				},
			}
			result := extractAPIBasePath(workflowData, "OPENAI_BASE_URL")
			assert.Equal(t, tt.expected, result, "Extracted base path should match expected value")
		})
	}

	t.Run("returns empty string when workflow data is nil", func(t *testing.T) {
		result := extractAPIBasePath(nil, "OPENAI_BASE_URL")
		assert.Empty(t, result, "Should return empty string for nil workflow data")
	})

	t.Run("returns empty string when engine config is nil", func(t *testing.T) {
		workflowData := &WorkflowData{EngineConfig: nil}
		result := extractAPIBasePath(workflowData, "OPENAI_BASE_URL")
		assert.Empty(t, result, "Should return empty string when engine config is nil")
	})

	t.Run("returns empty string when env var not set", func(t *testing.T) {
		workflowData := &WorkflowData{
			EngineConfig: &EngineConfig{
				Env: map[string]string{"OTHER_VAR": "value"},
			},
		}
		result := extractAPIBasePath(workflowData, "OPENAI_BASE_URL")
		assert.Empty(t, result, "Should return empty string when env var not set")
	})
}

// TestGetCopilotAPITarget tests the GetCopilotAPITarget helper that resolves the effective
// Copilot API target from either engine.api-target or GITHUB_COPILOT_BASE_URL in engine.env.
func TestGetCopilotAPITarget(t *testing.T) {
	tests := []struct {
		name         string
		workflowData *WorkflowData
		expected     string
	}{
		{
			name: "engine.api-target takes precedence over GITHUB_COPILOT_BASE_URL",
			workflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					ID:        "copilot",
					APITarget: "api.acme.ghe.com",
					Env: map[string]string{
						"GITHUB_COPILOT_BASE_URL": "https://other.endpoint.com",
					},
				},
			},
			expected: "api.acme.ghe.com",
		},
		{
			name: "GITHUB_COPILOT_BASE_URL used as fallback when api-target not set",
			workflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					ID: "copilot",
					Env: map[string]string{
						"GITHUB_COPILOT_BASE_URL": "https://copilot-api.contoso-aw.ghe.com",
					},
				},
			},
			expected: "copilot-api.contoso-aw.ghe.com",
		},
		{
			name: "GITHUB_COPILOT_BASE_URL with path extracts hostname only",
			workflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					ID: "copilot",
					Env: map[string]string{
						"GITHUB_COPILOT_BASE_URL": "https://copilot-proxy.corp.example.com/v1",
					},
				},
			},
			expected: "copilot-proxy.corp.example.com",
		},
		{
			name: "empty when neither api-target nor GITHUB_COPILOT_BASE_URL is set",
			workflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					ID: "copilot",
				},
			},
			expected: "",
		},
		{
			name:         "empty when workflowData is nil",
			workflowData: nil,
			expected:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetCopilotAPITarget(tt.workflowData)
			assert.Equal(t, tt.expected, result, "GetCopilotAPITarget should return expected hostname")
		})
	}
}

func TestGetCopilotAllowlistTargets(t *testing.T) {
	tests := []struct {
		name         string
		workflowData *WorkflowData
		expected     []string
	}{
		{
			name: "includes BYOK provider host and api-target when both are configured",
			workflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					ID:        "copilot",
					APITarget: "api.acme.ghe.com",
					Env: map[string]string{
						constants.CopilotProviderBaseURL: "https://llm.corp.example.com/v1",
					},
				},
			},
			expected: []string{"llm.corp.example.com", "api.acme.ghe.com"},
		},
		{
			name: "includes only BYOK provider host when no copilot api target is configured",
			workflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					ID: "copilot",
					Env: map[string]string{
						constants.CopilotProviderBaseURL: "http://localhost:11434/v1",
					},
				},
			},
			expected: []string{"localhost:11434"},
		},
		{
			name: "deduplicates identical provider and api targets",
			workflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					ID:        "copilot",
					APITarget: "llm.corp.example.com",
					Env: map[string]string{
						constants.CopilotProviderBaseURL: "https://llm.corp.example.com/v1",
					},
				},
			},
			expected: []string{"llm.corp.example.com"},
		},
		{
			name: "skips provider host extraction when BYOK base URL is a GitHub expression",
			workflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					ID: "copilot",
					Env: map[string]string{
						constants.CopilotProviderBaseURL: "${{ secrets.PROVIDER_BASE_URL }}",
					},
				},
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, GetCopilotAllowlistTargets(tt.workflowData), "GetCopilotAllowlistTargets should return expected targets for %s", tt.name)
		})
	}
}

func TestGetGeminiAPITarget(t *testing.T) {
	tests := []struct {
		name         string
		workflowData *WorkflowData
		engineName   string
		expected     string
	}{
		{
			name: "returns default target for gemini engine with no custom URL",
			workflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					ID: "gemini",
				},
			},
			engineName: "gemini",
			expected:   "generativelanguage.googleapis.com",
		},
		{
			name: "custom GEMINI_API_BASE_URL takes precedence over default",
			workflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					ID: "gemini",
					Env: map[string]string{
						"GEMINI_API_BASE_URL": "https://gemini-proxy.internal.company.com/v1",
					},
				},
			},
			engineName: "gemini",
			expected:   "gemini-proxy.internal.company.com",
		},
		{
			name: "returns empty for non-gemini engine without custom URL",
			workflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					ID: "claude",
				},
			},
			engineName: "claude",
			expected:   "",
		},
		{
			name:         "returns empty when workflowData is nil",
			workflowData: nil,
			engineName:   "gemini",
			expected:     "generativelanguage.googleapis.com",
		},
		{
			name: "returns custom target for non-gemini engine with GEMINI_API_BASE_URL",
			workflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					ID: "custom",
					Env: map[string]string{
						"GEMINI_API_BASE_URL": "https://custom-proxy.example.com",
					},
				},
			},
			engineName: "custom",
			expected:   "custom-proxy.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetGeminiAPITarget(tt.workflowData, tt.engineName)
			assert.Equal(t, tt.expected, result, "GetGeminiAPITarget should return expected hostname")
		})
	}
}
