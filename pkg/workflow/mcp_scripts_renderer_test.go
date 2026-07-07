//go:build !integration

package workflow

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCollectMCPScriptsSecrets(t *testing.T) {
	tests := []struct {
		name        string
		config      *MCPScriptsConfig
		expectedLen int
	}{
		{
			name:        "nil config",
			config:      nil,
			expectedLen: 0,
		},
		{
			name: "tool with secrets",
			config: &MCPScriptsConfig{
				Tools: map[string]*MCPScriptToolConfig{
					"test": {
						Name: "test",
						Env: map[string]string{
							"API_KEY": "${{ secrets.API_KEY }}",
						},
					},
				},
			},
			expectedLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collectMCPScriptsSecrets(tt.config)

			if len(result) != tt.expectedLen {
				t.Errorf("Expected %d secrets, got %d", tt.expectedLen, len(result))
			}
		})
	}
}

func TestCollectMCPScriptsSecretsStability(t *testing.T) {
	config := &MCPScriptsConfig{
		Tools: map[string]*MCPScriptToolConfig{
			"zebra-tool": {
				Name: "zebra-tool",
				Env: map[string]string{
					"ZEBRA_SECRET": "${{ secrets.ZEBRA }}",
					"ALPHA_SECRET": "${{ secrets.ALPHA }}",
				},
			},
			"alpha-tool": {
				Name: "alpha-tool",
				Env: map[string]string{
					"BETA_SECRET": "${{ secrets.BETA }}",
				},
			},
		},
	}

	// Test collectMCPScriptsSecrets stability
	iterations := 10
	secretResults := make([]map[string]string, iterations)
	for i := range iterations {
		secretResults[i] = collectMCPScriptsSecrets(config)
	}

	// All iterations should produce same key set
	for i := 1; i < iterations; i++ {
		if len(secretResults[i]) != len(secretResults[0]) {
			t.Errorf("collectMCPScriptsSecrets produced different number of secrets on iteration %d", i+1)
		}
		for key, val := range secretResults[0] {
			if secretResults[i][key] != val {
				t.Errorf("collectMCPScriptsSecrets produced different value for key %s on iteration %d", key, i+1)
			}
		}
	}
}

func TestMCPScriptsScopeConformance(t *testing.T) {
	config := &MCPScriptsConfig{
		Tools: map[string]*MCPScriptToolConfig{
			"alpha-tool": {
				Name: "alpha-tool",
				Env: map[string]string{
					"ALPHA_ONLY":    "${{ secrets.ALPHA_ONLY }}",
					"SHARED_SECRET": "${{ secrets.SHARED_SECRET }}",
				},
			},
			"beta-tool": {
				Name: "beta-tool",
				Env: map[string]string{
					"BETA_ONLY":     "${{ secrets.BETA_ONLY }}",
					"SHARED_SECRET": "${{ secrets.SHARED_SECRET }}",
				},
			},
		},
	}

	toolsJSON := GenerateMCPScriptsToolsConfig(config)
	if strings.Contains(toolsJSON, "secrets.ALPHA_ONLY") ||
		strings.Contains(toolsJSON, "secrets.BETA_ONLY") ||
		strings.Contains(toolsJSON, "secrets.SHARED_SECRET") {
		t.Fatalf("tools.json must not contain secret expressions: %s", toolsJSON)
	}

	var rendered MCPScriptsConfigJSON
	if err := json.Unmarshal([]byte(toolsJSON), &rendered); err != nil {
		t.Fatalf("failed to parse tools.json: %v", err)
	}

	var alphaEnv, betaEnv map[string]string
	for _, tool := range rendered.Tools {
		switch tool.Name {
		case "alpha-tool":
			alphaEnv = tool.Env
		case "beta-tool":
			betaEnv = tool.Env
		}
	}

	if alphaEnv == nil || betaEnv == nil {
		t.Fatalf("expected both alpha-tool and beta-tool in tools.json: %+v", rendered.Tools)
	}

	// SN-SCOPE-01/03: per-tool secret bindings are explicit and isolated in tools.json.
	if alphaEnv["ALPHA_ONLY"] != "ALPHA_ONLY" || alphaEnv["SHARED_SECRET"] != "SHARED_SECRET" {
		t.Fatalf("alpha-tool env bindings are incorrect: %+v", alphaEnv)
	}
	if betaEnv["BETA_ONLY"] != "BETA_ONLY" || betaEnv["SHARED_SECRET"] != "SHARED_SECRET" {
		t.Fatalf("beta-tool env bindings are incorrect: %+v", betaEnv)
	}

	// SN-SCOPE-02: tool-specific bindings do not include other tool-only secrets.
	if _, exists := alphaEnv["BETA_ONLY"]; exists {
		t.Fatalf("alpha-tool env must not include beta-tool secret binding: %+v", alphaEnv)
	}
	if _, exists := betaEnv["ALPHA_ONLY"]; exists {
		t.Fatalf("beta-tool env must not include alpha-tool secret binding: %+v", betaEnv)
	}
}
