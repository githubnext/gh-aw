//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/types"
)

// TestEngineVersionTypeHandling tests that engine.version correctly handles
// numeric types (int, float) and string types as specified in the schema.
// This is a regression test to prevent type handling inconsistencies.
func TestEngineVersionTypeHandling(t *testing.T) {
	tests := []struct {
		name            string
		frontmatter     map[string]any
		expectedVersion string
	}{
		{
			name: "string version",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id":      "copilot",
					"version": "beta",
				},
			},
			expectedVersion: "beta",
		},
		{
			name: "integer version",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id":      "copilot",
					"version": 20,
				},
			},
			expectedVersion: "20",
		},
		{
			name: "float version",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id":      "copilot",
					"version": 3.11,
				},
			},
			expectedVersion: "3.11",
		},
		{
			name: "int64 version",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id":      "copilot",
					"version": int64(142),
				},
			},
			expectedVersion: "142",
		},
		{
			name: "uint64 version",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id":      "copilot",
					"version": uint64(999),
				},
			},
			expectedVersion: "999",
		},
		{
			name: "version with semantic versioning",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id":      "copilot",
					"version": "v1.2.3",
				},
			},
			expectedVersion: "v1.2.3",
		},
		{
			name: "version with build metadata",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id":      "copilot",
					"version": "1.0.0-beta.1+build.123",
				},
			},
			expectedVersion: "1.0.0-beta.1+build.123",
		},
	}

	compiler := NewCompiler()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, config := compiler.ExtractEngineConfig(tt.frontmatter)

			if config == nil {
				t.Fatal("Expected config to be non-nil")
			}

			if config.Version != tt.expectedVersion {
				t.Errorf("Expected version %q, got %q", tt.expectedVersion, config.Version)
			}
		})
	}
}

// TestEngineVersionNotProvided tests that when no version is provided,
// the Version field remains empty (default behavior).
func TestEngineVersionNotProvided(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter map[string]any
	}{
		{
			name: "engine without version field",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id": "copilot",
				},
			},
		},
		{
			name: "engine as string (backward compatibility)",
			frontmatter: map[string]any{
				"engine": "copilot",
			},
		},
	}

	compiler := NewCompiler()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, config := compiler.ExtractEngineConfig(tt.frontmatter)

			if config == nil {
				t.Fatal("Expected config to be non-nil")
			}

			if config.Version != "" {
				t.Errorf("Expected empty version, got %q", config.Version)
			}
		})
	}
}

// TestEngineVersionWithOtherFields tests that version works correctly
// alongside other engine configuration fields.
func TestEngineVersionWithOtherFields(t *testing.T) {
	frontmatter := map[string]any{
		"engine": map[string]any{
			"id":      "copilot",
			"version": "0.0.369",
			"model":   "gpt-4",
			"env": map[string]any{
				"DEBUG": "true",
			},
		},
	}

	compiler := NewCompiler()
	_, config := compiler.ExtractEngineConfig(frontmatter)

	if config == nil {
		t.Fatal("Expected config to be non-nil")
	}

	if config.ID != "copilot" {
		t.Errorf("Expected ID 'copilot', got %q", config.ID)
	}

	if config.Version != "0.0.369" {
		t.Errorf("Expected version '0.0.369', got %q", config.Version)
	}

	if config.Model != "gpt-4" {
		t.Errorf("Expected model 'gpt-4', got %q", config.Model)
	}

	if len(config.Env) != 1 {
		t.Errorf("Expected 1 env var, got %d", len(config.Env))
	}

	if config.Env["DEBUG"] != "true" {
		t.Errorf("Expected DEBUG='true', got %q", config.Env["DEBUG"])
	}
}

// TestEngineCommandField tests that the command field is correctly extracted
func TestEngineCommandField(t *testing.T) {
	tests := []struct {
		name            string
		frontmatter     map[string]any
		expectedCommand string
	}{
		{
			name: "command field provided",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id":      "copilot",
					"command": "/usr/local/bin/custom-copilot",
				},
			},
			expectedCommand: "/usr/local/bin/custom-copilot",
		},
		{
			name: "command field not provided",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id": "copilot",
				},
			},
			expectedCommand: "",
		},
		{
			name: "command with relative path",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id":      "claude",
					"command": "./bin/claude-cli",
				},
			},
			expectedCommand: "./bin/claude-cli",
		},
		{
			name: "command with environment variable",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id":      "codex",
					"command": "$HOME/.local/bin/codex",
				},
			},
			expectedCommand: "$HOME/.local/bin/codex",
		},
	}

	compiler := NewCompiler()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, config := compiler.ExtractEngineConfig(tt.frontmatter)

			if config == nil {
				t.Fatal("Expected config to be non-nil")
			}

			if config.Command != tt.expectedCommand {
				t.Errorf("Expected command %q, got %q", tt.expectedCommand, config.Command)
			}
		})
	}
}

// TestAPITargetExtraction tests that the api-target configuration is correctly
// extracted from frontmatter for custom API endpoints (GHEC, GHES, or custom AI endpoints).
func TestAPITargetExtraction(t *testing.T) {
	tests := []struct {
		name              string
		frontmatter       map[string]any
		expectedAPITarget string
	}{
		{
			name: "GHEC api-target",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id":         "copilot",
					"api-target": "api.acme.ghe.com",
				},
			},
			expectedAPITarget: "api.acme.ghe.com",
		},
		{
			name: "GHES api-target",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id":         "copilot",
					"api-target": "api.enterprise.githubcopilot.com",
				},
			},
			expectedAPITarget: "api.enterprise.githubcopilot.com",
		},
		{
			name: "custom api-target",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id":         "codex",
					"api-target": "api.custom.endpoint.com",
				},
			},
			expectedAPITarget: "api.custom.endpoint.com",
		},
		{
			name: "no api-target",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id": "copilot",
				},
			},
			expectedAPITarget: "",
		},
		{
			name: "empty api-target",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id":         "copilot",
					"api-target": "",
				},
			},
			expectedAPITarget: "",
		},
	}

	compiler := NewCompiler()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, config := compiler.ExtractEngineConfig(tt.frontmatter)

			if config == nil {
				t.Fatal("Expected config to be non-nil")
			}

			if config.APITarget != tt.expectedAPITarget {
				t.Errorf("Expected api-target %q, got %q", tt.expectedAPITarget, config.APITarget)
			}
		})
	}
}

func TestParseEngineTokenWeights(t *testing.T) {
	tests := []struct {
		name             string
		raw              any
		wantNil          bool
		wantMultipliers  map[string]float64
		wantClassWeights *types.TokenClassWeights
	}{
		{
			name:    "nil input returns nil",
			raw:     nil,
			wantNil: true,
		},
		{
			name:    "non-map input returns nil",
			raw:     "not-a-map",
			wantNil: true,
		},
		{
			name:    "empty map returns nil",
			raw:     map[string]any{},
			wantNil: true,
		},
		{
			name: "multipliers only",
			raw: map[string]any{
				"multipliers": map[string]any{
					"my-model": float64(2.5),
					"gpt-5":    float64(3.0),
				},
			},
			wantMultipliers: map[string]float64{
				"my-model": 2.5,
				"gpt-5":    3.0,
			},
		},
		{
			name: "token-class-weights only",
			raw: map[string]any{
				"token-class-weights": map[string]any{
					"output": float64(6.0),
				},
			},
			wantClassWeights: &types.TokenClassWeights{
				Output: 6.0,
			},
		},
		{
			name: "both multipliers and token-class-weights",
			raw: map[string]any{
				"multipliers": map[string]any{
					"custom-model": float64(1.5),
				},
				"token-class-weights": map[string]any{
					"input":        float64(1.0),
					"cached-input": float64(0.05),
					"output":       float64(5.0),
					"reasoning":    float64(5.0),
					"cache-write":  float64(1.0),
				},
			},
			wantMultipliers: map[string]float64{"custom-model": 1.5},
			wantClassWeights: &types.TokenClassWeights{
				Input:       1.0,
				CachedInput: 0.05,
				Output:      5.0,
				Reasoning:   5.0,
				CacheWrite:  1.0,
			},
		},
		{
			name: "integer multiplier values are accepted",
			raw: map[string]any{
				"multipliers": map[string]any{
					"int-model": int(2),
				},
			},
			wantMultipliers: map[string]float64{"int-model": 2.0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseEngineTokenWeights(tt.raw)
			if tt.wantNil {
				if got != nil {
					t.Errorf("expected nil, got %+v", got)
				}
				return
			}
			if got == nil {
				t.Fatal("expected non-nil result")
			}
			if tt.wantMultipliers != nil {
				for model, want := range tt.wantMultipliers {
					if got.Multipliers[model] != want {
						t.Errorf("multiplier[%q] = %v, want %v", model, got.Multipliers[model], want)
					}
				}
			}
			if tt.wantClassWeights != nil {
				if got.TokenClassWeights == nil {
					t.Fatal("expected TokenClassWeights to be set")
				}
				want := tt.wantClassWeights
				tcw := got.TokenClassWeights
				if want.Input != 0 && tcw.Input != want.Input {
					t.Errorf("Input weight = %v, want %v", tcw.Input, want.Input)
				}
				if want.CachedInput != 0 && tcw.CachedInput != want.CachedInput {
					t.Errorf("CachedInput weight = %v, want %v", tcw.CachedInput, want.CachedInput)
				}
				if want.Output != 0 && tcw.Output != want.Output {
					t.Errorf("Output weight = %v, want %v", tcw.Output, want.Output)
				}
				if want.Reasoning != 0 && tcw.Reasoning != want.Reasoning {
					t.Errorf("Reasoning weight = %v, want %v", tcw.Reasoning, want.Reasoning)
				}
				if want.CacheWrite != 0 && tcw.CacheWrite != want.CacheWrite {
					t.Errorf("CacheWrite weight = %v, want %v", tcw.CacheWrite, want.CacheWrite)
				}
			}
		})
	}
}

func TestExtractEngineConfigTokenWeights(t *testing.T) {
	compiler := NewCompiler()

	frontmatter := map[string]any{
		"engine": map[string]any{
			"id": "claude",
			"token-weights": map[string]any{
				"multipliers": map[string]any{
					"my-custom-model": float64(2.5),
				},
				"token-class-weights": map[string]any{
					"output": float64(6.0),
				},
			},
		},
	}

	_, config := compiler.ExtractEngineConfig(frontmatter)
	if config == nil {
		t.Fatal("Expected non-nil config")
	}
	if config.TokenWeights == nil {
		t.Fatal("Expected TokenWeights to be set")
	}
	if config.TokenWeights.Multipliers["my-custom-model"] != 2.5 {
		t.Errorf("Expected multiplier 2.5, got %v", config.TokenWeights.Multipliers["my-custom-model"])
	}
	if config.TokenWeights.TokenClassWeights == nil {
		t.Fatal("Expected TokenClassWeights to be set")
	}
	if config.TokenWeights.TokenClassWeights.Output != 6.0 {
		t.Errorf("Expected output weight 6.0, got %v", config.TokenWeights.TokenClassWeights.Output)
	}
}

func TestTokenWeightsSingleQuoteEscapingInYAML(t *testing.T) {
	compiler := NewCompiler()
	registry := GetGlobalEngineRegistry()
	engine, err := registry.GetEngine("claude")
	if err != nil {
		t.Fatalf("Failed to get claude engine: %v", err)
	}

	// Model name containing a single quote — must not break YAML single-quoted scalar
	workflowData := &WorkflowData{
		Name: "Test Workflow",
		EngineConfig: &EngineConfig{
			ID: "claude",
			TokenWeights: &types.TokenWeights{
				Multipliers: map[string]float64{
					"bob's-model": 2.0, // Single quote in key
				},
			},
		},
	}

	var out strings.Builder
	compiler.generateCreateAwInfo(&out, workflowData, engine)
	output := out.String()

	// The generated YAML must not contain an un-escaped single quote inside a single-quoted value.
	// In YAML, a single quote inside a single-quoted scalar is represented as ”.
	if !strings.Contains(output, "bob''s-model") {
		t.Errorf("Expected single quote to be escaped as '' in YAML output, got:\n%s", output)
	}
	// There must be no dangling unescaped single quote inside the GH_AW_INFO_TOKEN_WEIGHTS value
	if strings.Contains(output, "bob's-model") {
		t.Errorf("Unescaped single quote found in YAML output:\n%s", output)
	}
}
