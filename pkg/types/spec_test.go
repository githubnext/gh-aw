//go:build !integration

package types_test

import (
	"encoding/json"
	"testing"

	"github.com/github/gh-aw/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSpec_Types_BaseMCPServerConfig validates that BaseMCPServerConfig has all documented
// fields and can be used for the server modes described in the types package README.md.
func TestSpec_Types_BaseMCPServerConfig(t *testing.T) {
	tests := []struct {
		name   string
		cfg    types.BaseMCPServerConfig
		checks func(t *testing.T, cfg types.BaseMCPServerConfig)
	}{
		{
			name: "stdio MCP server from spec example",
			cfg: types.BaseMCPServerConfig{
				Type:    "stdio",
				Command: "npx",
				Args:    []string{"-y", "@modelcontextprotocol/server-filesystem"},
				Env: map[string]string{
					"ALLOWED_PATHS": "/workspace",
				},
			},
			checks: func(t *testing.T, cfg types.BaseMCPServerConfig) {
				assert.Equal(t, "stdio", cfg.Type, "Type field must hold the server type")
				assert.Equal(t, "npx", cfg.Command, "Command field must hold the executable")
				assert.Equal(t, []string{"-y", "@modelcontextprotocol/server-filesystem"}, cfg.Args, "Args field must hold the arguments")
				assert.Equal(t, "/workspace", cfg.Env["ALLOWED_PATHS"], "Env field must hold environment variables")
			},
		},
		{
			name: "HTTP MCP server with OIDC auth from spec example",
			cfg: types.BaseMCPServerConfig{
				Type: "http",
				URL:  "https://my-mcp-server.example.com",
				Auth: &types.MCPAuthConfig{
					Type:     "github-oidc",
					Audience: "https://my-mcp-server.example.com",
				},
			},
			checks: func(t *testing.T, cfg types.BaseMCPServerConfig) {
				assert.Equal(t, "http", cfg.Type, "Type field must be 'http' for HTTP mode")
				assert.Equal(t, "https://my-mcp-server.example.com", cfg.URL, "URL field must hold the HTTP endpoint")
				require.NotNil(t, cfg.Auth, "Auth field must be non-nil for authenticated HTTP servers")
				assert.Equal(t, "github-oidc", cfg.Auth.Type, "Auth.Type must match the documented auth type")
				assert.Equal(t, "https://my-mcp-server.example.com", cfg.Auth.Audience, "Auth.Audience must match the server URL")
			},
		},
		{
			name: "container MCP server with mounts",
			cfg: types.BaseMCPServerConfig{
				Type:           "container",
				Container:      "my-mcp-image:latest",
				Entrypoint:     "/usr/bin/server",
				EntrypointArgs: []string{"--port", "8080"},
				Mounts:         []string{"/host/path:/container/path:ro"},
			},
			checks: func(t *testing.T, cfg types.BaseMCPServerConfig) {
				assert.Equal(t, "my-mcp-image:latest", cfg.Container, "Container field must hold the image")
				assert.Equal(t, "/usr/bin/server", cfg.Entrypoint, "Entrypoint field must hold the override entrypoint")
				assert.Equal(t, []string{"--port", "8080"}, cfg.EntrypointArgs, "EntrypointArgs must hold the entrypoint arguments")
				assert.Equal(t, []string{"/host/path:/container/path:ro"}, cfg.Mounts, "Mounts must hold volume mounts in source:dest:mode format")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.checks(t, tt.cfg)
		})
	}
}

// TestSpec_Types_BaseMCPServerConfig_JSONRoundTrip validates that BaseMCPServerConfig fields
// use both json and yaml struct tags as documented in the Design Notes section of the README.
// Spec: "All struct fields use both json and yaml struct tags so they can be round-tripped
// through both serialization formats."
func TestSpec_Types_BaseMCPServerConfig_JSONRoundTrip(t *testing.T) {
	original := types.BaseMCPServerConfig{
		Type:    "stdio",
		Command: "npx",
		Args:    []string{"-y", "@modelcontextprotocol/server-filesystem"},
		Env:     map[string]string{"ALLOWED_PATHS": "/workspace"},
		Version: "1.2.3",
		Headers: map[string]string{"X-Custom": "header"},
	}

	data, err := json.Marshal(original)
	require.NoError(t, err, "BaseMCPServerConfig must serialize to JSON without error")

	var decoded types.BaseMCPServerConfig
	require.NoError(t, json.Unmarshal(data, &decoded), "BaseMCPServerConfig must deserialize from JSON without error")

	assert.Equal(t, original.Type, decoded.Type, "Type must round-trip through JSON")
	assert.Equal(t, original.Command, decoded.Command, "Command must round-trip through JSON")
	assert.Equal(t, original.Args, decoded.Args, "Args must round-trip through JSON")
	assert.Equal(t, original.Env, decoded.Env, "Env must round-trip through JSON")
	assert.Equal(t, original.Version, decoded.Version, "Version must round-trip through JSON")
	assert.Equal(t, original.Headers, decoded.Headers, "Headers must round-trip through JSON")
}

// TestSpec_Types_MCPAuthConfig validates the MCPAuthConfig type documented in the README.
// Spec: "Authentication configuration for HTTP MCP servers. When configured, the MCP gateway
// dynamically acquires tokens and injects them as Authorization headers on each outgoing request."
func TestSpec_Types_MCPAuthConfig(t *testing.T) {
	tests := []struct {
		name     string
		auth     types.MCPAuthConfig
		wantType string
		wantAud  string
	}{
		{
			name: "github-oidc auth from spec example",
			auth: types.MCPAuthConfig{
				Type:     "github-oidc",
				Audience: "https://my-service.example.com",
			},
			wantType: "github-oidc",
			wantAud:  "https://my-service.example.com",
		},
		{
			name: "auth without explicit audience",
			auth: types.MCPAuthConfig{
				Type: "github-oidc",
			},
			wantType: "github-oidc",
			wantAud:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantType, tt.auth.Type, "Type field must match — spec says only 'github-oidc' is currently supported")
			assert.Equal(t, tt.wantAud, tt.auth.Audience, "Audience field must match — defaults to server URL if omitted")
		})
	}

	// Spec: "Type is the authentication type; currently only 'github-oidc' is supported."
	data, err := json.Marshal(types.MCPAuthConfig{Type: "github-oidc", Audience: "https://aud.example.com"})
	require.NoError(t, err, "MCPAuthConfig must serialize to JSON")
	assert.Contains(t, string(data), `"type":"github-oidc"`, "JSON must use documented field name 'type'")
	assert.Contains(t, string(data), `"audience"`, "JSON must use documented field name 'audience'")
}

// TestSpec_Types_TokenWeights validates the TokenWeights type documented in the README.
// Spec: "Defines custom model cost information for effective token computation.
// Specified under engine.token-weights in workflow frontmatter."
func TestSpec_Types_TokenWeights(t *testing.T) {
	weights := types.TokenWeights{
		Multipliers: map[string]float64{
			"gpt-4o": 2.5,
		},
		TokenClassWeights: &types.TokenClassWeights{
			Input:  1.0,
			Output: 3.0,
		},
	}

	assert.InDelta(t, 2.5, weights.Multipliers["gpt-4o"], 1e-9, "Multipliers must map model names to cost multipliers")
	require.NotNil(t, weights.TokenClassWeights, "TokenClassWeights must be settable")
	assert.InDelta(t, 1.0, weights.TokenClassWeights.Input, 1e-9, "TokenClassWeights.Input must hold the input token weight")
	assert.InDelta(t, 3.0, weights.TokenClassWeights.Output, 1e-9, "TokenClassWeights.Output must hold the output token weight")
}

// TestSpec_Types_TokenClassWeights validates the TokenClassWeights type documented in the README.
// Spec: "Per-token-class weights for effective token computation. Each field corresponds to
// one token class; a zero value means 'use the default weight'."
func TestSpec_Types_TokenClassWeights(t *testing.T) {
	// Spec documents these token classes:
	//   Input       → standard input tokens
	//   CachedInput → cache-hit input tokens
	//   Output      → generated output tokens
	//   Reasoning   → internal reasoning tokens
	//   CacheWrite  → cache-write tokens
	w := types.TokenClassWeights{
		Input:       1.0,
		CachedInput: 0.1,
		Output:      3.0,
		Reasoning:   2.0,
		CacheWrite:  1.5,
	}

	assert.InDelta(t, 1.0, w.Input, 1e-9, "Input must hold the standard input token weight")
	assert.InDelta(t, 0.1, w.CachedInput, 1e-9, "CachedInput must hold the cache-hit input token weight")
	assert.InDelta(t, 3.0, w.Output, 1e-9, "Output must hold the generated output token weight")
	assert.InDelta(t, 2.0, w.Reasoning, 1e-9, "Reasoning must hold the internal reasoning token weight")
	assert.InDelta(t, 1.5, w.CacheWrite, 1e-9, "CacheWrite must hold the cache-write token weight")

	// Spec: "a zero value means 'use the default weight'"
	zero := types.TokenClassWeights{}
	assert.InDelta(t, 0.0, zero.Input, 1e-9, "zero value of Input must be 0 (use default)")
	assert.InDelta(t, 0.0, zero.CachedInput, 1e-9, "zero value of CachedInput must be 0 (use default)")

	// Verify JSON field names from struct tags (hyphens, matching frontmatter schema)
	data, err := json.Marshal(w)
	require.NoError(t, err, "TokenClassWeights must serialize to JSON")
	assert.Contains(t, string(data), `"cached-input"`, "JSON must use hyphenated field name 'cached-input'")
	assert.Contains(t, string(data), `"cache-write"`, "JSON must use hyphenated field name 'cache-write'")
}

// TestSpec_Types_ZeroValueSafety validates that all types have sensible zero values
// and no required-but-unset-field panics.
// Spec (Design Notes): "BaseMCPServerConfig is designed to be embedded."
func TestSpec_Types_ZeroValueSafety(t *testing.T) {
	// Zero value of BaseMCPServerConfig must be usable without panicking.
	var cfg types.BaseMCPServerConfig
	assert.Empty(t, cfg.Type, "zero value Type must be empty string")
	assert.Nil(t, cfg.Auth, "zero value Auth must be nil")
	assert.Nil(t, cfg.Args, "zero value Args must be nil")

	// Zero value of TokenWeights must be usable.
	var tw types.TokenWeights
	assert.Nil(t, tw.TokenClassWeights, "zero value TokenClassWeights pointer must be nil (no overrides)")
	assert.Nil(t, tw.Multipliers, "zero value Multipliers must be nil (no overrides)")
}
