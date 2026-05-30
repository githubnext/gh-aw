//go:build !integration

package workflow

import (
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
