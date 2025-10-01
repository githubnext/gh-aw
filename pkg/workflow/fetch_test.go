package workflow

import (
	"strings"
	"testing"
)

func TestAddMCPFetchServerIfNeeded(t *testing.T) {
	tests := []struct {
		name           string
		tools          map[string]any
		engineSupports bool
		expectFetch    bool
		expectWebFetch bool
	}{
		{
			name: "web-fetch requested, engine supports it",
			tools: map[string]any{
				"web-fetch": nil,
			},
			engineSupports: true,
			expectFetch:    false,
			expectWebFetch: true,
		},
		{
			name: "web-fetch requested, engine does not support it",
			tools: map[string]any{
				"web-fetch": nil,
			},
			engineSupports: false,
			expectFetch:    true,
			expectWebFetch: false,
		},
		{
			name: "web-fetch not requested",
			tools: map[string]any{
				"bash": nil,
			},
			engineSupports: false,
			expectFetch:    false,
			expectWebFetch: false,
		},
		{
			name:           "empty tools",
			tools:          map[string]any{},
			engineSupports: false,
			expectFetch:    false,
			expectWebFetch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use the registry to get an actual engine
			registry := GetGlobalEngineRegistry()
			var engine CodingAgentEngine
			if tt.engineSupports {
				engine, _ = registry.GetEngine("claude") // Claude supports web-fetch
			} else {
				engine, _ = registry.GetEngine("codex") // Codex doesn't support web-fetch
			}

			updatedTools, addedServers := AddMCPFetchServerIfNeeded(tt.tools, engine)

			// Check if mcp/fetch was added
			_, hasFetch := updatedTools["mcp/fetch"]
			if hasFetch != tt.expectFetch {
				t.Errorf("Expected mcp/fetch present=%v, got %v", tt.expectFetch, hasFetch)
			}

			// Check if web-fetch was kept or removed
			_, hasWebFetch := updatedTools["web-fetch"]
			if hasWebFetch != tt.expectWebFetch {
				t.Errorf("Expected web-fetch present=%v, got %v", tt.expectWebFetch, hasWebFetch)
			}

			// Check the returned list of added servers
			if tt.expectFetch {
				if len(addedServers) != 1 || addedServers[0] != "mcp/fetch" {
					t.Errorf("Expected addedServers to contain 'mcp/fetch', got %v", addedServers)
				}
			} else {
				if len(addedServers) != 0 {
					t.Errorf("Expected no added servers, got %v", addedServers)
				}
			}
		})
	}
}

func TestRenderMCPFetchServerConfig(t *testing.T) {
	tests := []struct {
		name         string
		format       string
		indent       string
		isLast       bool
		expectSubstr []string
	}{
		{
			name:   "JSON format, not last",
			format: "json",
			indent: "    ",
			isLast: false,
			expectSubstr: []string{
				`"mcp/fetch": {`,
				`"command": "docker"`,
				`"ghcr.io/modelcontextprotocol/servers/fetch:latest"`,
				`},`,
			},
		},
		{
			name:   "JSON format, last",
			format: "json",
			indent: "    ",
			isLast: true,
			expectSubstr: []string{
				`"mcp/fetch": {`,
				`"command": "docker"`,
				`"ghcr.io/modelcontextprotocol/servers/fetch:latest"`,
				`}`, // No comma
			},
		},
		{
			name:   "TOML format",
			format: "toml",
			indent: "  ",
			isLast: false,
			expectSubstr: []string{
				`[mcp_servers."mcp/fetch"]`,
				`command = "docker"`,
				`"ghcr.io/modelcontextprotocol/servers/fetch:latest"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var yaml strings.Builder
			renderMCPFetchServerConfig(&yaml, tt.format, tt.indent, tt.isLast)
			output := yaml.String()

			for _, substr := range tt.expectSubstr {
				if !strings.Contains(output, substr) {
					t.Errorf("Expected output to contain %q, but it didn't.\nFull output:\n%s", substr, output)
				}
			}
		})
	}
}

func TestEngineSupportsWebFetch(t *testing.T) {
	registry := GetGlobalEngineRegistry()

	tests := []struct {
		engineID       string
		expectsSupport bool
	}{
		{"claude", true},
		{"codex", false},
		{"copilot", false},
		{"custom", false},
	}

	for _, tt := range tests {
		t.Run(tt.engineID, func(t *testing.T) {
			engine, err := registry.GetEngine(tt.engineID)
			if err != nil {
				t.Fatalf("Failed to get engine %s: %v", tt.engineID, err)
			}

			actualSupport := engine.SupportsWebFetch()
			if actualSupport != tt.expectsSupport {
				t.Errorf("Expected engine %s to have SupportsWebFetch()=%v, got %v",
					tt.engineID, tt.expectsSupport, actualSupport)
			}
		})
	}
}
