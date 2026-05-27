//go:build !integration

package workflow

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/types"
	"github.com/github/gh-aw/pkg/workflow/compilerenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuildAWFConfigJSON verifies that BuildAWFConfigJSON produces a valid JSON config
// that contains the expected network, apiProxy, and container fields.
func TestBuildAWFConfigJSON(t *testing.T) {
	t.Run("basic config with allowed domains and API proxy enabled", func(t *testing.T) {
		config := AWFCommandConfig{
			EngineName:     "copilot",
			AllowedDomains: "github.com,api.github.com",
			WorkflowData: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot"},
				NetworkPermissions: &NetworkPermissions{
					Firewall: &FirewallConfig{Enabled: true},
				},
			},
		}

		jsonStr, err := BuildAWFConfigJSON(config)
		require.NoError(t, err, "BuildAWFConfigJSON should not return an error")

		// Must be valid JSON
		var parsed map[string]any
		require.NoError(t, json.Unmarshal([]byte(jsonStr), &parsed), "result must be valid JSON")

		// Schema reference
		assert.Contains(t, jsonStr, "$schema", "should include $schema reference")

		// Network section with allowDomains
		assert.Contains(t, jsonStr, `"allowDomains"`, "should include allowDomains")
		assert.Contains(t, jsonStr, "github.com", "should include github.com in allowDomains")
		assert.Contains(t, jsonStr, "api.github.com", "should include api.github.com in allowDomains")

		// apiProxy section with enabled: true
		assert.Contains(t, jsonStr, `"apiProxy"`, "should include apiProxy section")
		assert.Contains(t, jsonStr, `"enabled":true`, "apiProxy should be enabled")
		assert.Contains(t, jsonStr, fmt.Sprintf(`"maxRuns":%d`, constants.DefaultMaxRuns), "apiProxy should emit default maxRuns")
		assert.Contains(t, jsonStr, fmt.Sprintf(`"maxEffectiveTokens":%d`, constants.DefaultMaxEffectiveTokens), "apiProxy should emit default maxEffectiveTokens")

		// container.imageTag
		assert.Contains(t, jsonStr, `"imageTag"`, "should include imageTag")
	})

	t.Run("blocked domains are included in the network section", func(t *testing.T) {
		config := AWFCommandConfig{
			EngineName:     "copilot",
			AllowedDomains: "github.com",
			WorkflowData: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot"},
				NetworkPermissions: &NetworkPermissions{
					Blocked:  []string{"ads.example.com"},
					Firewall: &FirewallConfig{Enabled: true},
				},
			},
		}

		jsonStr, err := BuildAWFConfigJSON(config)
		require.NoError(t, err, "BuildAWFConfigJSON should not return an error")

		assert.Contains(t, jsonStr, `"blockDomains"`, "should include blockDomains")
		assert.Contains(t, jsonStr, "ads.example.com", "should include the blocked domain")
	})

	t.Run("openai API target is included in apiProxy targets", func(t *testing.T) {
		config := AWFCommandConfig{
			EngineName:     "codex",
			AllowedDomains: "github.com",
			WorkflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					ID: "codex",
					Env: map[string]string{
						"OPENAI_BASE_URL": "https://my-proxy.internal.example.com/v1",
					},
				},
				NetworkPermissions: &NetworkPermissions{
					Firewall: &FirewallConfig{Enabled: true},
				},
			},
		}

		jsonStr, err := BuildAWFConfigJSON(config)
		require.NoError(t, err, "BuildAWFConfigJSON should not return an error")

		assert.Contains(t, jsonStr, `"targets"`, "should include targets in apiProxy")
		assert.Contains(t, jsonStr, `"openai"`, "should include openai target")
		assert.Contains(t, jsonStr, "my-proxy.internal.example.com", "should include the openai host")
	})

	t.Run("configured max-effective-tokens is emitted in apiProxy config", func(t *testing.T) {
		config := AWFCommandConfig{
			EngineName:     "copilot",
			AllowedDomains: "github.com",
			WorkflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					ID:                 "copilot",
					MaxEffectiveTokens: 424242,
				},
				NetworkPermissions: &NetworkPermissions{
					Firewall: &FirewallConfig{Enabled: true},
				},
			},
		}

		jsonStr, err := BuildAWFConfigJSON(config)
		require.NoError(t, err)
		assert.Contains(t, jsonStr, `"maxEffectiveTokens":424242`, "apiProxy should emit configured maxEffectiveTokens")
	})

	t.Run("enterprise default max-effective-tokens env var is used when frontmatter is unset", func(t *testing.T) {
		t.Setenv(compilerenv.DefaultMaxEffectiveTokens, "123456")
		config := AWFCommandConfig{
			EngineName:     "copilot",
			AllowedDomains: "github.com",
			WorkflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					ID: "copilot",
				},
				NetworkPermissions: &NetworkPermissions{
					Firewall: &FirewallConfig{Enabled: true},
				},
			},
		}

		jsonStr, err := BuildAWFConfigJSON(config)
		require.NoError(t, err)
		assert.Contains(t, jsonStr, `"maxEffectiveTokens":123456`, "apiProxy should emit env var default maxEffectiveTokens")
	})

	t.Run("token steering is enabled by default in apiProxy config", func(t *testing.T) {
		config := AWFCommandConfig{
			EngineName:     "copilot",
			AllowedDomains: "github.com",
			WorkflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					ID: "copilot",
				},
				NetworkPermissions: &NetworkPermissions{
					Firewall: &FirewallConfig{Enabled: true},
				},
			},
		}

		jsonStr, err := BuildAWFConfigJSON(config)
		require.NoError(t, err)
		assert.Contains(t, jsonStr, `"enableTokenSteering":true`, "apiProxy should emit enableTokenSteering by default")
	})

	t.Run("token steering is disabled when max-effective-tokens is negative", func(t *testing.T) {
		config := AWFCommandConfig{
			EngineName:     "copilot",
			AllowedDomains: "github.com",
			WorkflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					ID:                 "copilot",
					MaxEffectiveTokens: -1,
				},
				NetworkPermissions: &NetworkPermissions{
					Firewall: &FirewallConfig{Enabled: true},
				},
			},
		}

		jsonStr, err := BuildAWFConfigJSON(config)
		require.NoError(t, err)
		assert.NotContains(t, jsonStr, `"enableTokenSteering"`, "apiProxy should omit enableTokenSteering when max-effective-tokens is negative")
		assert.NotContains(t, jsonStr, `"maxEffectiveTokens"`, "apiProxy should omit maxEffectiveTokens when negative (disabled)")
	})

	t.Run("token steering is skipped for unsupported AWF versions", func(t *testing.T) {
		config := AWFCommandConfig{
			EngineName:     "copilot",
			AllowedDomains: "github.com",
			WorkflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					ID: "copilot",
				},
				NetworkPermissions: &NetworkPermissions{
					Firewall: &FirewallConfig{
						Enabled: true,
						Version: "v0.25.43",
					},
				},
			},
		}

		jsonStr, err := BuildAWFConfigJSON(config)
		require.NoError(t, err)
		assert.NotContains(t, jsonStr, `"enableTokenSteering"`, "apiProxy should omit enableTokenSteering for unsupported AWF versions")
	})

	t.Run("configured max-runs is emitted in apiProxy config", func(t *testing.T) {
		config := AWFCommandConfig{
			EngineName:     "copilot",
			AllowedDomains: "github.com",
			WorkflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					ID:      "copilot",
					MaxRuns: 37,
				},
				NetworkPermissions: &NetworkPermissions{
					Firewall: &FirewallConfig{Enabled: true},
				},
			},
		}

		jsonStr, err := BuildAWFConfigJSON(config)
		require.NoError(t, err)
		assert.Contains(t, jsonStr, `"maxRuns":37`, "apiProxy should emit configured maxRuns")
	})

	t.Run("default max-runs is emitted when not configured", func(t *testing.T) {
		config := AWFCommandConfig{
			EngineName:     "copilot",
			AllowedDomains: "github.com",
			WorkflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					ID: "copilot",
				},
				NetworkPermissions: &NetworkPermissions{
					Firewall: &FirewallConfig{Enabled: true},
				},
			},
		}

		jsonStr, err := BuildAWFConfigJSON(config)
		require.NoError(t, err)
		assert.Contains(t, jsonStr, fmt.Sprintf(`"maxRuns":%d`, constants.DefaultMaxRuns), "apiProxy should emit default maxRuns when unset")
	})

	t.Run("engine token-weights multipliers are emitted in apiProxy modelMultipliers", func(t *testing.T) {
		config := AWFCommandConfig{
			EngineName:     "copilot",
			AllowedDomains: "github.com",
			WorkflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					ID: "copilot",
					TokenWeights: &types.TokenWeights{
						Multipliers: map[string]float64{
							"gpt-5":      1.2,
							"gpt-5-mini": 0.8,
						},
					},
				},
				NetworkPermissions: &NetworkPermissions{
					Firewall: &FirewallConfig{Enabled: true},
				},
			},
		}

		jsonStr, err := BuildAWFConfigJSON(config)
		require.NoError(t, err)
		assert.Contains(t, jsonStr, `"modelMultipliers"`, "apiProxy should emit modelMultipliers")
		assert.Contains(t, jsonStr, `"gpt-5":1.2`, "apiProxy should include configured model multiplier")
		assert.Contains(t, jsonStr, `"gpt-5-mini":0.8`, "apiProxy should include configured model multiplier")
	})

	t.Run("apiProxy modelMultipliers omitted when engine token-weights multipliers are empty", func(t *testing.T) {
		config := AWFCommandConfig{
			EngineName:     "copilot",
			AllowedDomains: "github.com",
			WorkflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					ID: "copilot",
					TokenWeights: &types.TokenWeights{
						Multipliers: map[string]float64{},
					},
				},
				NetworkPermissions: &NetworkPermissions{
					Firewall: &FirewallConfig{Enabled: true},
				},
			},
		}

		jsonStr, err := BuildAWFConfigJSON(config)
		require.NoError(t, err)
		assert.NotContains(t, jsonStr, `"modelMultipliers"`, "apiProxy should omit modelMultipliers when empty")
	})

	t.Run("anthropic API target is included in apiProxy targets", func(t *testing.T) {
		config := AWFCommandConfig{
			EngineName:     "claude",
			AllowedDomains: "github.com",
			WorkflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					ID: "claude",
					Env: map[string]string{
						"ANTHROPIC_BASE_URL": "https://corp-gateway.example.com/anthropic",
					},
				},
				NetworkPermissions: &NetworkPermissions{
					Firewall: &FirewallConfig{Enabled: true},
				},
			},
		}

		jsonStr, err := BuildAWFConfigJSON(config)
		require.NoError(t, err)

		assert.Contains(t, jsonStr, `"anthropic"`, "should include anthropic target")
		assert.Contains(t, jsonStr, "corp-gateway.example.com", "should include the anthropic host")
	})

	t.Run("antigravity engine routes API target through gemini provider", func(t *testing.T) {
		config := AWFCommandConfig{
			EngineName:     "antigravity",
			AllowedDomains: "github.com",
			WorkflowData: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "antigravity"},
				NetworkPermissions: &NetworkPermissions{
					Firewall: &FirewallConfig{Enabled: true},
				},
			},
		}

		jsonStr, err := BuildAWFConfigJSON(config)
		require.NoError(t, err)

		assert.Contains(t, jsonStr, `"gemini"`, "should include gemini target for antigravity engine")
		assert.Contains(t, jsonStr, "generativelanguage.googleapis.com", "should include default Gemini API hostname")
		assert.NotContains(t, jsonStr, `"antigravity"`, "should not include unsupported antigravity target key")
	})

	t.Run("antigravity custom base URL maps to gemini provider target", func(t *testing.T) {
		config := AWFCommandConfig{
			EngineName:     "antigravity",
			AllowedDomains: "github.com",
			WorkflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					ID: "antigravity",
					Env: map[string]string{
						"ANTIGRAVITY_API_BASE_URL": "https://antigravity-proxy.internal.example.com/v1",
					},
				},
				NetworkPermissions: &NetworkPermissions{
					Firewall: &FirewallConfig{Enabled: true},
				},
			},
		}

		jsonStr, err := BuildAWFConfigJSON(config)
		require.NoError(t, err)

		assert.Contains(t, jsonStr, `"gemini"`, "should include gemini target for antigravity engine")
		assert.Contains(t, jsonStr, "antigravity-proxy.internal.example.com", "should include host from ANTIGRAVITY_API_BASE_URL")
		assert.NotContains(t, jsonStr, `"antigravity"`, "should not include unsupported antigravity target key")
	})

	t.Run("no API targets section when no custom endpoints are configured", func(t *testing.T) {
		config := AWFCommandConfig{
			EngineName:     "copilot",
			AllowedDomains: "github.com",
			WorkflowData: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot"},
				NetworkPermissions: &NetworkPermissions{
					Firewall: &FirewallConfig{Enabled: true},
				},
			},
		}

		jsonStr, err := BuildAWFConfigJSON(config)
		require.NoError(t, err)

		// No custom targets for the default copilot engine
		assert.NotContains(t, jsonStr, `"targets"`, "should not include targets when no custom endpoints")
	})

	t.Run("image tag with digest metadata is included in container section", func(t *testing.T) {
		config := AWFCommandConfig{
			EngineName:     "copilot",
			AllowedDomains: "github.com",
			WorkflowData: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot"},
				NetworkPermissions: &NetworkPermissions{
					Firewall: &FirewallConfig{Enabled: true},
				},
			},
		}

		jsonStr, err := BuildAWFConfigJSON(config)
		require.NoError(t, err)

		assert.Contains(t, jsonStr, `"container"`, "should include container section")
		assert.Contains(t, jsonStr, `"imageTag"`, "should include imageTag in container section")
	})

	t.Run("empty allowed domains produces no network section", func(t *testing.T) {
		config := AWFCommandConfig{
			EngineName:     "copilot",
			AllowedDomains: "",
			WorkflowData: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot"},
				NetworkPermissions: &NetworkPermissions{
					Firewall: &FirewallConfig{Enabled: true},
				},
			},
		}

		jsonStr, err := BuildAWFConfigJSON(config)
		require.NoError(t, err)

		assert.NotContains(t, jsonStr, `"network"`, "should not include network section when no domains")
	})

	t.Run("output is compact valid JSON (no pretty-print indentation)", func(t *testing.T) {
		config := AWFCommandConfig{
			EngineName:     "copilot",
			AllowedDomains: "github.com",
			WorkflowData: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot"},
				NetworkPermissions: &NetworkPermissions{
					Firewall: &FirewallConfig{Enabled: true},
				},
			},
		}

		jsonStr, err := BuildAWFConfigJSON(config)
		require.NoError(t, err)

		// Compact JSON: no leading whitespace inside the object, no newlines.
		// This is important for safe embedding in a printf command.
		assert.NotContains(t, jsonStr, "\n", "JSON output should not contain newlines (must be compact)")
		assert.NotContains(t, jsonStr, "    ", "JSON output should not contain indentation")
	})

	t.Run("github actions expressions preserve && operators in allowDomains", func(t *testing.T) {
		config := AWFCommandConfig{
			EngineName:     "copilot",
			AllowedDomains: "${{ env.MCP_ENV == 'staging' && env.MCP_URL_STAGING || env.MCP_URL_PROD }}",
			WorkflowData: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot"},
				NetworkPermissions: &NetworkPermissions{
					Firewall: &FirewallConfig{Enabled: true},
				},
			},
		}

		jsonStr, err := BuildAWFConfigJSON(config)
		require.NoError(t, err, "BuildAWFConfigJSON should not return an error")

		assert.Contains(t, jsonStr, "&&", "JSON output should preserve && in GitHub Actions expressions")
		assert.NotContains(t, jsonStr, "\\u0026", "JSON output should not HTML-escape '&' characters")
	})
}

// TestBuildAWFConfigSchemaURL verifies that buildAWFConfigSchemaURL returns a release-pinned
// URL that tracks the AWF version in use.
func TestBuildAWFConfigSchemaURL(t *testing.T) {
	tests := []struct {
		name           string
		firewallConfig *FirewallConfig
		wantContains   string
		wantURL        string // exact URL match, takes precedence over wantContains when set
	}{
		{
			name:           "nil config uses DefaultFirewallVersion",
			firewallConfig: nil,
			wantContains:   string(constants.DefaultFirewallVersion),
		},
		{
			name:           "empty version uses DefaultFirewallVersion",
			firewallConfig: &FirewallConfig{Enabled: true},
			wantContains:   string(constants.DefaultFirewallVersion),
		},
		{
			name:           "pinned version with v prefix",
			firewallConfig: &FirewallConfig{Enabled: true, Version: "v0.24.0"},
			wantContains:   "v0.24.0",
		},
		{
			name:           "pinned version without v prefix gets v added",
			firewallConfig: &FirewallConfig{Enabled: true, Version: "0.24.0"},
			wantContains:   "v0.24.0",
		},
		{
			name:           "latest version uses /releases/latest/download/ URL",
			firewallConfig: &FirewallConfig{Enabled: true, Version: "latest"},
			wantURL:        "https://github.com/github/gh-aw-firewall/releases/latest/download/awf-config.schema.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := buildAWFConfigSchemaURL(tt.firewallConfig)

			if tt.wantURL != "" {
				assert.Equal(t, tt.wantURL, url, "schema URL should match the expected URL exactly")
				return
			}

			assert.Contains(t, url, tt.wantContains, "schema URL should contain the expected version")
			assert.Contains(t, url, "https://github.com/github/gh-aw-firewall/releases/download/", "schema URL should use the release download path")
			assert.True(t, strings.HasSuffix(url, "awf-config.schema.json"), "schema URL should end with awf-config.schema.json")
		})
	}
}

// TestBuildAWFConfigJSON_SchemaURLIsVersionPinned verifies that the $schema field in the
// generated config uses a release-pinned URL that matches the AWF version in use.
func TestBuildAWFConfigJSON_SchemaURLIsVersionPinned(t *testing.T) {
	t.Run("default version when no version pinned", func(t *testing.T) {
		config := AWFCommandConfig{
			EngineName:     "copilot",
			AllowedDomains: "github.com",
			WorkflowData: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot"},
				NetworkPermissions: &NetworkPermissions{
					Firewall: &FirewallConfig{Enabled: true},
				},
			},
		}

		jsonStr, err := BuildAWFConfigJSON(config)
		require.NoError(t, err)

		expectedVersion := string(constants.DefaultFirewallVersion)
		assert.Contains(t, jsonStr, expectedVersion, "schema URL should contain the default firewall version")
		assert.Contains(t, jsonStr, "releases/download/", "schema URL should use release download path")
		assert.Contains(t, jsonStr, "awf-config.schema.json", "schema URL should reference awf-config.schema.json")
	})

	t.Run("pinned version appears in schema URL", func(t *testing.T) {
		config := AWFCommandConfig{
			EngineName:     "copilot",
			AllowedDomains: "github.com",
			WorkflowData: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot"},
				NetworkPermissions: &NetworkPermissions{
					Firewall: &FirewallConfig{Enabled: true, Version: "v0.24.0"},
				},
			},
		}

		jsonStr, err := BuildAWFConfigJSON(config)
		require.NoError(t, err)

		assert.Contains(t, jsonStr, "v0.24.0", "schema URL should contain the pinned version")
		assert.NotContains(t, jsonStr, string(constants.DefaultFirewallVersion), "schema URL should not contain default version when version is pinned")
	})
}

// TestBuildAWFConfigJSON_DomainDeduplication verifies that duplicate domain entries
// in the comma-separated allowed domains list are removed.
func TestBuildAWFConfigJSON_DomainDeduplication(t *testing.T) {
	config := AWFCommandConfig{
		EngineName:     "copilot",
		AllowedDomains: "github.com,api.github.com,github.com", // github.com duplicated
		WorkflowData: &WorkflowData{
			EngineConfig: &EngineConfig{ID: "copilot"},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{Enabled: true},
			},
		},
	}

	jsonStr, err := BuildAWFConfigJSON(config)
	require.NoError(t, err)

	var parsed struct {
		Network struct {
			AllowDomains []string `json:"allowDomains"`
		} `json:"network"`
	}
	require.NoError(t, json.Unmarshal([]byte(jsonStr), &parsed))

	// github.com should appear exactly once
	count := 0
	for _, d := range parsed.Network.AllowDomains {
		if d == "github.com" {
			count++
		}
	}
	assert.Equal(t, 1, count, "github.com should appear exactly once after deduplication")
}

// TestSplitDomainList verifies the splitDomainList helper handles edge cases.
func TestSplitDomainList(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple comma-separated list",
			input:    "github.com,api.github.com",
			expected: []string{"github.com", "api.github.com"},
		},
		{
			name:     "list with spaces after commas",
			input:    "github.com, api.github.com, raw.githubusercontent.com",
			expected: []string{"github.com", "api.github.com", "raw.githubusercontent.com"},
		},
		{
			name:     "single domain",
			input:    "github.com",
			expected: []string{"github.com"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "wildcards are preserved",
			input:    "*.github.com,*.githubusercontent.com",
			expected: []string{"*.github.com", "*.githubusercontent.com"},
		},
		{
			name:     "duplicates are removed",
			input:    "github.com,api.github.com,github.com",
			expected: []string{"github.com", "api.github.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitDomainList(tt.input)
			assert.Equal(t, tt.expected, result, "splitDomainList(%q)", tt.input)
		})
	}
}

// TestBuildAWFCommand_UsesConfigFile verifies that BuildAWFCommand always produces a run step
// that writes a JSON config file and references it via --config.
func TestBuildAWFCommand_UsesConfigFile(t *testing.T) {
	config := AWFCommandConfig{
		EngineName:     "copilot",
		EngineCommand:  "copilot --prompt-file /tmp/prompt.txt",
		LogFile:        "/tmp/gh-aw/agent-stdio.log",
		AllowedDomains: "github.com,api.github.com",
		WorkflowData: &WorkflowData{
			EngineConfig: &EngineConfig{ID: "copilot"},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{Enabled: true},
			},
		},
	}

	command := BuildAWFCommand(config)

	// Should write the config file using printf
	assert.Contains(t, command, "printf", "expected printf command to write the config file")
	assert.Contains(t, command, "awf-config.json", "expected awf-config.json reference")

	// Should copy the config file to /tmp/gh-aw/awf-config.json for artifact upload
	assert.Contains(t, command, constants.AWFConfigFilePath, "expected awf-config.json to be copied to /tmp/gh-aw/")

	// Should reference the config file via --config
	assert.Contains(t, command, "--config", "expected --config flag in AWF invocation")

	// Should NOT have --allow-domains as a CLI flag (moved to config file)
	assert.NotContains(t, command, "--allow-domains", "expected --allow-domains to be absent from CLI args")

	// Should NOT have --enable-api-proxy as a CLI flag (moved to config file)
	assert.NotContains(t, command, "--enable-api-proxy", "expected --enable-api-proxy to be absent from CLI args")

	// Should NOT have --image-tag as a CLI flag (moved to config file)
	assert.NotContains(t, command, "--image-tag", "expected --image-tag to be absent from CLI args")

	// The JSON content in the printf command should have the expected structure
	assert.Contains(t, command, `"allowDomains"`, "config JSON should include allowDomains")
	assert.Contains(t, command, `"enabled":true`, "config JSON should have apiProxy enabled")
}

func TestBuildAWFCommand_ModelMultipliersLoadedFromFile(t *testing.T) {
	config := AWFCommandConfig{
		EngineName:    "copilot",
		EngineCommand: "copilot --prompt-file /tmp/prompt.txt",
		LogFile:       "/tmp/gh-aw/agent-stdio.log",
		WorkflowData: &WorkflowData{
			EngineConfig: &EngineConfig{
				ID: "copilot",
				TokenWeights: &types.TokenWeights{
					Multipliers: map[string]float64{
						"my-custom-model": 2.5,
					},
				},
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{Enabled: true},
			},
		},
	}

	command := BuildAWFCommand(config)

	assert.Contains(t, command, awfModelMultipliersFilePath, "expected model multipliers artifact path in runtime updater script")
	assert.Contains(t, command, `node "${RUNNER_TEMP}/gh-aw/actions/merge_awf_model_multipliers.cjs"`, "expected runtime updater script to invoke JS model multiplier merger")
	assert.NotContains(t, command, "my-custom-model", "expected custom model multipliers to be omitted from inline AWF config JSON")
}

func TestBuildAWFCommand_PreservesGitHubExpressionOperatorsInConfigJSON(t *testing.T) {
	config := AWFCommandConfig{
		EngineName:     "copilot",
		EngineCommand:  "copilot --prompt-file /tmp/prompt.txt",
		LogFile:        "/tmp/gh-aw/agent-stdio.log",
		AllowedDomains: "${{ env.MCP_ENV == 'staging' && env.MCP_URL_STAGING || env.MCP_URL_PROD }}",
		WorkflowData: &WorkflowData{
			EngineConfig: &EngineConfig{ID: "copilot"},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{Enabled: true},
			},
		},
	}

	command := BuildAWFCommand(config)

	assert.Contains(t, command, "env.MCP_ENV == 'staging'", "expected full GitHub Actions expression to be preserved")
	assert.Contains(t, command, "&&", "expected AWF config JSON in command to preserve &&")
	assert.NotContains(t, command, "\\u0026", "expected AWF config JSON in command to not HTML-escape '&'")
}

// TestBuildAWFCommand_SchemaKeyEscapedWhenExpressionPresent verifies that the $schema JSON
// key is written as \$schema in the shell command when the AWF config JSON is double-quoted
// (i.e. when AllowedDomains contains a GitHub Actions expression). Without the escaping,
// bash expands $schema as a variable — which is always unset — producing an empty key that
// causes AWF to reject the config with "config. is not supported".
func TestBuildAWFCommand_SchemaKeyEscapedWhenExpressionPresent(t *testing.T) {
	config := AWFCommandConfig{
		EngineName:    "copilot",
		EngineCommand: "copilot --prompt-file /tmp/prompt.txt",
		LogFile:       "/tmp/gh-aw/agent-stdio.log",
		// AllowedDomains contains a GitHub Actions expression, forcing double-quote
		// shell wrapping for the config JSON. The $schema key must still be safe.
		AllowedDomains: "${{ env.DOMAINS }}",
		WorkflowData: &WorkflowData{
			EngineConfig: &EngineConfig{ID: "copilot"},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{Enabled: true},
			},
		},
	}

	command := BuildAWFCommand(config)

	// The config JSON is double-quoted because AllowedDomains contains ${{ }}.
	// The $schema key must appear as \$schema so bash does not expand it.
	assert.Contains(t, command, `\$schema`, "expected \\$schema (escaped) in double-quoted config JSON to prevent bash variable expansion")

	// The GitHub Actions expression must remain unescaped for runner evaluation.
	assert.Contains(t, command, "${{ env.DOMAINS }}", "expected GitHub Actions expression to be preserved unescaped")
}

// TestBuildAWFCommand_ConfigFileWithPathSetup verifies that the config file write command
// is correctly integrated with the path setup section.
func TestBuildAWFCommand_ConfigFileWithPathSetup(t *testing.T) {
	config := AWFCommandConfig{
		EngineName:     "copilot",
		EngineCommand:  "copilot --prompt-file /tmp/prompt.txt",
		LogFile:        "/tmp/gh-aw/agent-stdio.log",
		AllowedDomains: "github.com",
		PathSetup:      "export GH_AW_NODE_BIN=$(command -v node || true)",
		WorkflowData: &WorkflowData{
			EngineConfig: &EngineConfig{ID: "copilot"},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{Enabled: true},
			},
		},
	}

	command := BuildAWFCommand(config)

	// PathSetup, config write, and AWF invocation must all appear in order
	pathSetupIdx := strings.Index(command, "GH_AW_NODE_BIN")
	configWriteIdx := strings.Index(command, "awf-config.json")
	awfIdx := strings.Index(command, "sudo -E awf")

	assert.GreaterOrEqual(t, pathSetupIdx, 0, "path setup should appear in command")
	assert.GreaterOrEqual(t, configWriteIdx, 0, "config file write should appear in command")
	assert.GreaterOrEqual(t, awfIdx, 0, "AWF invocation should appear in command")

	// Order must be: path setup → config write → AWF invocation
	assert.Less(t, pathSetupIdx, configWriteIdx, "path setup must precede config file write")
	assert.Less(t, configWriteIdx, awfIdx, "config file write must precede AWF invocation")
}

func TestBuildAWFCommand_WorkflowCallNetworkAllowedUpdaterUsesRunnerTempEnv(t *testing.T) {
	config := AWFCommandConfig{
		EngineName:     "copilot",
		EngineCommand:  "copilot --prompt-file /tmp/prompt.txt",
		LogFile:        "/tmp/gh-aw/agent-stdio.log",
		AllowedDomains: "github.com",
		WorkflowData: &WorkflowData{
			On:           "workflow_call",
			EngineConfig: &EngineConfig{ID: "copilot"},
			NetworkPermissions: &NetworkPermissions{
				Allowed:      []string{"defaults"},
				AllowedInput: true,
				Firewall:     &FirewallConfig{Enabled: true},
			},
		},
	}

	command := BuildAWFCommand(config)

	assert.Contains(t, command, `os.environ.get("RUNNER_TEMP")`, "workflow_call network updater should resolve RUNNER_TEMP inside Python")
	assert.NotContains(t, command, `Path("${RUNNER_TEMP}/gh-aw/awf-config.json")`, "workflow_call network updater should not embed an unexpanded RUNNER_TEMP literal")
}

// TestBuildAWFCommand_WritesAgentCLIStartTimestamp verifies that BuildAWFCommand
// always emits a printf command that writes the epoch-ms timestamp to
// AgentCLIStartMsPath at the very beginning of the run block, before any
// PathSetup or config-file write, so sendJobConclusionSpan can use it as the
// true start time of the Execute Agent CLI step.
func TestBuildAWFCommand_WritesAgentCLIStartTimestamp(t *testing.T) {
	tests := []struct {
		name      string
		pathSetup string
	}{
		{"with PathSetup", "touch " + AgentStepSummaryPath},
		{"without PathSetup", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			config := AWFCommandConfig{
				EngineName:     "claude",
				EngineCommand:  "claude --print",
				LogFile:        "/tmp/gh-aw/agent-stdio.log",
				AllowedDomains: "api.anthropic.com",
				PathSetup:      tc.pathSetup,
				WorkflowData: &WorkflowData{
					EngineConfig: &EngineConfig{ID: "claude"},
					NetworkPermissions: &NetworkPermissions{
						Firewall: &FirewallConfig{Enabled: true},
					},
				},
			}

			command := BuildAWFCommand(config)

			// The timestamp write must appear in the command.
			assert.Contains(t, command, AgentCLIStartMsPath,
				"command must write agent CLI start timestamp to %s", AgentCLIStartMsPath)
			assert.Contains(t, command, "date +%s%3N",
				"command must use date +%%s%%3N to capture epoch milliseconds")

			// The timestamp write must appear before the AWF invocation so it captures
			// the step start time rather than the time after AWF container setup.
			tsIdx := strings.Index(command, AgentCLIStartMsPath)
			awfIdx := strings.Index(command, "sudo -E awf")
			assert.Less(t, tsIdx, awfIdx,
				"timestamp write must appear before AWF invocation")

			// The timestamp write must be the first substantive line after set -o pipefail.
			pipefailIdx := strings.Index(command, "set -o pipefail")
			assert.Less(t, pipefailIdx, tsIdx,
				"set -o pipefail must appear before timestamp write")
			// Nothing between set -o pipefail and the timestamp write should reference
			// PathSetup content (timestamp must come first).
			if tc.pathSetup != "" {
				pathSetupIdx := strings.Index(command, tc.pathSetup)
				assert.Greater(t, pathSetupIdx, tsIdx,
					"timestamp write must appear before PathSetup content")
			}
		})
	}
}
