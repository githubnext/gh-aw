package workflow

import (
	"testing"
)

func TestNewTools(t *testing.T) {
	t.Run("creates empty tools from nil map", func(t *testing.T) {
		tools := NewTools(nil)
		if tools == nil {
			t.Fatal("expected non-nil tools")
		}
		if tools.Custom == nil {
			t.Error("expected non-nil Custom map")
		}
		if len(tools.GetToolNames()) != 0 {
			t.Errorf("expected 0 tools, got %d", len(tools.GetToolNames()))
		}
	})

	t.Run("creates empty tools from empty map", func(t *testing.T) {
		tools := NewTools(map[string]any{})
		if tools == nil {
			t.Fatal("expected non-nil tools")
		}
		if len(tools.GetToolNames()) != 0 {
			t.Errorf("expected 0 tools, got %d", len(tools.GetToolNames()))
		}
	})

	t.Run("parses known tools", func(t *testing.T) {
		toolsMap := map[string]any{
			"github":    map[string]any{"allowed": []any{"get_issue"}},
			"bash":      []any{"echo", "ls"},
			"edit":      nil,
			"web-fetch": nil,
		}

		tools := NewTools(toolsMap)
		if tools == nil {
			t.Fatal("expected non-nil tools")
		}

		if !tools.HasTool("github") {
			t.Error("expected GitHub tool to be set")
		}
		if !tools.HasTool("bash") {
			t.Error("expected Bash tool to be set")
		}
		if !tools.HasTool("edit") {
			t.Error("expected Edit tool to be set")
		}
		if !tools.HasTool("web-fetch") {
			t.Error("expected WebFetch tool to be set")
		}

		names := tools.GetToolNames()
		if len(names) != 4 {
			t.Errorf("expected 4 tools, got %d: %v", len(names), names)
		}
	})

	t.Run("parses custom MCP tools", func(t *testing.T) {
		toolsMap := map[string]any{
			"github":      nil,
			"my-custom":   map[string]any{"command": "node", "args": []any{"server.js"}},
			"another-mcp": map[string]any{"type": "http", "url": "http://localhost:8080"},
		}

		tools := NewTools(toolsMap)
		if tools == nil {
			t.Fatal("expected non-nil tools")
		}

		if len(tools.Custom) != 2 {
			t.Errorf("expected 2 custom tools, got %d", len(tools.Custom))
		}

		if tools.Custom["my-custom"] == nil {
			t.Error("expected my-custom tool in Custom map")
		}
		if tools.Custom["another-mcp"] == nil {
			t.Error("expected another-mcp tool in Custom map")
		}

		names := tools.GetToolNames()
		if len(names) != 3 {
			t.Errorf("expected 3 tools, got %d: %v", len(names), names)
		}
	})
}

func TestHasTool(t *testing.T) {
	toolsMap := map[string]any{
		"github":    nil,
		"bash":      []any{"echo"},
		"my-custom": map[string]any{"command": "node"},
	}

	tools := NewTools(toolsMap)

	tests := []struct {
		name     string
		toolName string
		expected bool
	}{
		{"github exists", "github", true},
		{"bash exists", "bash", true},
		{"custom exists", "my-custom", true},
		{"edit doesn't exist", "edit", false},
		{"web-fetch doesn't exist", "web-fetch", false},
		{"unknown doesn't exist", "unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tools.HasTool(tt.toolName)
			if result != tt.expected {
				t.Errorf("HasTool(%q) = %v, want %v", tt.toolName, result, tt.expected)
			}
		})
	}

	t.Run("nil tools returns false", func(t *testing.T) {
		var tools *Tools
		if tools.HasTool("github") {
			t.Error("expected false for nil tools")
		}
	})
}

func TestGetToolNames(t *testing.T) {
	t.Run("empty tools returns empty list", func(t *testing.T) {
		tools := NewTools(nil)
		names := tools.GetToolNames()
		if len(names) != 0 {
			t.Errorf("expected 0 names, got %d", len(names))
		}
	})

	t.Run("returns all tool names", func(t *testing.T) {
		toolsMap := map[string]any{
			"github":    nil,
			"bash":      []any{"echo"},
			"edit":      nil,
			"my-custom": map[string]any{},
		}

		tools := NewTools(toolsMap)
		names := tools.GetToolNames()

		if len(names) != 4 {
			t.Errorf("expected 4 names, got %d", len(names))
		}

		// Check that all expected names are present
		expectedNames := map[string]bool{
			"github":    false,
			"bash":      false,
			"edit":      false,
			"my-custom": false,
		}

		for _, name := range names {
			if _, ok := expectedNames[name]; ok {
				expectedNames[name] = true
			}
		}

		for name, found := range expectedNames {
			if !found {
				t.Errorf("expected to find tool %q in names list", name)
			}
		}
	})

	t.Run("nil tools returns empty list", func(t *testing.T) {
		var tools *Tools
		names := tools.GetToolNames()
		if len(names) != 0 {
			t.Errorf("expected 0 names, got %d", len(names))
		}
	})
}

func TestGitHubConfigParsing(t *testing.T) {
	t.Run("returns nil when github not set", func(t *testing.T) {
		tools := NewTools(map[string]any{})
		if tools.GitHub != nil {
			t.Error("expected nil GitHub config when github not set")
		}
	})

	t.Run("parses github config map", func(t *testing.T) {
		toolsMap := map[string]any{
			"github": map[string]any{
				"allowed":      []any{"get_issue", "create_issue"},
				"mode":         "remote",
				"version":      "v1.0.0",
				"args":         []any{"--verbose"},
				"read-only":    true,
				"github-token": "${{ secrets.MY_TOKEN }}",
				"toolset":      []any{"repos", "issues"},
			},
		}

		tools := NewTools(toolsMap)
		config := tools.GitHub

		if config == nil {
			t.Fatal("expected non-nil config")
		}

		if len(config.Allowed) != 2 {
			t.Errorf("expected 2 allowed tools, got %d", len(config.Allowed))
		}
		if config.Allowed[0] != "get_issue" {
			t.Errorf("expected first allowed tool to be 'get_issue', got %q", config.Allowed[0])
		}

		if config.Mode != "remote" {
			t.Errorf("expected mode 'remote', got %q", config.Mode)
		}

		if config.Version != "v1.0.0" {
			t.Errorf("expected version 'v1.0.0', got %q", config.Version)
		}

		if len(config.Args) != 1 {
			t.Errorf("expected 1 arg, got %d", len(config.Args))
		}

		if !config.ReadOnly {
			t.Error("expected ReadOnly to be true")
		}

		if config.GitHubToken != "${{ secrets.MY_TOKEN }}" {
			t.Errorf("expected github-token to be '${{ secrets.MY_TOKEN }}', got %q", config.GitHubToken)
		}

		if len(config.Toolset) != 2 {
			t.Errorf("expected 2 toolsets, got %d", len(config.Toolset))
		}
	})
}

func TestPlaywrightConfigParsing(t *testing.T) {
	t.Run("returns nil when playwright not set", func(t *testing.T) {
		tools := NewTools(map[string]any{})
		if tools.Playwright != nil {
			t.Error("expected nil Playwright config when playwright not set")
		}
	})

	t.Run("parses playwright config map", func(t *testing.T) {
		toolsMap := map[string]any{
			"playwright": map[string]any{
				"version":         "v1.41.0",
				"allowed_domains": []any{"github.com", "example.com"},
				"args":            []any{"--headless"},
			},
		}

		tools := NewTools(toolsMap)
		config := tools.Playwright

		if config == nil {
			t.Fatal("expected non-nil config")
		}

		if config.Version != "v1.41.0" {
			t.Errorf("expected version 'v1.41.0', got %q", config.Version)
		}

		if len(config.AllowedDomains) != 2 {
			t.Errorf("expected 2 allowed domains, got %d", len(config.AllowedDomains))
		}

		if len(config.Args) != 1 {
			t.Errorf("expected 1 arg, got %d", len(config.Args))
		}
	})
}

func TestExtractMapFromFrontmatter(t *testing.T) {
	tests := []struct {
		name         string
		frontmatter  map[string]any
		key          string
		expectedLen  int
		expectedKeys []string
	}{
		{
			name: "extracts existing map",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"github": nil,
					"bash":   []any{"echo"},
				},
			},
			key:          "tools",
			expectedLen:  2,
			expectedKeys: []string{"github", "bash"},
		},
		{
			name: "returns empty map when key doesn't exist",
			frontmatter: map[string]any{
				"other": "value",
			},
			key:          "tools",
			expectedLen:  0,
			expectedKeys: []string{},
		},
		{
			name: "returns empty map when value is not a map",
			frontmatter: map[string]any{
				"tools": "not-a-map",
			},
			key:          "tools",
			expectedLen:  0,
			expectedKeys: []string{},
		},
		{
			name: "returns empty map when value is nil",
			frontmatter: map[string]any{
				"tools": nil,
			},
			key:          "tools",
			expectedLen:  0,
			expectedKeys: []string{},
		},
		{
			name: "returns empty map when value is array",
			frontmatter: map[string]any{
				"tools": []string{"github", "bash"},
			},
			key:          "tools",
			expectedLen:  0,
			expectedKeys: []string{},
		},
		{
			name:         "handles nil frontmatter",
			frontmatter:  nil,
			key:          "tools",
			expectedLen:  0,
			expectedKeys: []string{},
		},
		{
			name:         "handles empty frontmatter",
			frontmatter:  map[string]any{},
			key:          "tools",
			expectedLen:  0,
			expectedKeys: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractMapFromFrontmatter(tt.frontmatter, tt.key)

			if result == nil {
				t.Fatal("expected non-nil result")
			}

			if len(result) != tt.expectedLen {
				t.Errorf("expected map with %d entries, got %d", tt.expectedLen, len(result))
			}

			for _, key := range tt.expectedKeys {
				if _, ok := result[key]; !ok {
					t.Errorf("expected key %q to exist in result", key)
				}
			}
		})
	}
}

func TestExtractToolsFromFrontmatter(t *testing.T) {
	frontmatter := map[string]any{
		"tools": map[string]any{
			"github": nil,
			"bash":   []any{"echo"},
		},
		"mcp-servers": map[string]any{
			"my-server": map[string]any{"command": "node"},
		},
	}

	result := extractToolsFromFrontmatter(frontmatter)

	if len(result) != 2 {
		t.Errorf("expected 2 tools, got %d", len(result))
	}

	if _, ok := result["github"]; !ok {
		t.Error("expected 'github' key in result")
	}

	if _, ok := result["bash"]; !ok {
		t.Error("expected 'bash' key in result")
	}

	// Should not include mcp-servers
	if _, ok := result["my-server"]; ok {
		t.Error("unexpected 'my-server' key in result")
	}
}

func TestExtractMCPServersFromFrontmatter(t *testing.T) {
	frontmatter := map[string]any{
		"tools": map[string]any{
			"github": nil,
		},
		"mcp-servers": map[string]any{
			"my-server":      map[string]any{"command": "node"},
			"another-server": map[string]any{"command": "python"},
		},
	}

	result := extractMCPServersFromFrontmatter(frontmatter)

	if len(result) != 2 {
		t.Errorf("expected 2 MCP servers, got %d", len(result))
	}

	if _, ok := result["my-server"]; !ok {
		t.Error("expected 'my-server' key in result")
	}

	if _, ok := result["another-server"]; !ok {
		t.Error("expected 'another-server' key in result")
	}

	// Should not include tools
	if _, ok := result["github"]; ok {
		t.Error("unexpected 'github' key in result")
	}
}

func TestExtractRuntimesFromFrontmatter(t *testing.T) {
	frontmatter := map[string]any{
		"tools": map[string]any{
			"github": nil,
		},
		"runtimes": map[string]any{
			"node":   map[string]any{"version": "18"},
			"python": map[string]any{"version": "3.11"},
		},
	}

	result := extractRuntimesFromFrontmatter(frontmatter)

	if len(result) != 2 {
		t.Errorf("expected 2 runtimes, got %d", len(result))
	}

	if _, ok := result["node"]; !ok {
		t.Error("expected 'node' key in result")
	}

	if _, ok := result["python"]; !ok {
		t.Error("expected 'python' key in result")
	}

	// Should not include tools
	if _, ok := result["github"]; ok {
		t.Error("unexpected 'github' key in result")
	}
}
