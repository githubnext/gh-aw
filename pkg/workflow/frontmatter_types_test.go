package workflow

import (
	"strings"
	"testing"
)

func TestUnmarshalFromMap(t *testing.T) {
	t.Run("extracts simple string field", func(t *testing.T) {
		data := map[string]any{
			"name": "test-workflow",
		}

		var result string

		err := unmarshalFromMap(data, "name", &result)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if result != "test-workflow" {
			t.Errorf("got %q, want %q", result, "test-workflow")
		}
	})

	t.Run("extracts nested map", func(t *testing.T) {
		data := map[string]any{
			"tools": map[string]any{
				"bash": map[string]any{
					"enabled": true,
					"timeout": 300,
				},
			},
		}

		var result map[string]any
		err := unmarshalFromMap(data, "tools", &result)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if result == nil {
			t.Fatal("result is nil")
		}

		bash, ok := result["bash"].(map[string]any)
		if !ok {
			t.Fatal("bash tool not found or wrong type")
		}

		if bash["enabled"] != true {
			t.Errorf("bash.enabled = %v, want true", bash["enabled"])
		}
	})

	t.Run("returns error for missing key", func(t *testing.T) {
		data := map[string]any{
			"name": "test",
		}

		var result string

		err := unmarshalFromMap(data, "missing", &result)
		if err == nil {
			t.Error("expected error for missing key, got nil")
		}

		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("error should mention 'not found', got: %v", err)
		}
	})

	t.Run("handles numeric types", func(t *testing.T) {
		data := map[string]any{
			"timeout": 42,
			"count":   int64(100),
			"ratio":   3.14,
		}

		var timeout int
		if err := unmarshalFromMap(data, "timeout", &timeout); err != nil {
			t.Errorf("timeout unmarshal error: %v", err)
		}
		if timeout != 42 {
			t.Errorf("timeout = %d, want 42", timeout)
		}

		var count int64
		if err := unmarshalFromMap(data, "count", &count); err != nil {
			t.Errorf("count unmarshal error: %v", err)
		}
		if count != 100 {
			t.Errorf("count = %d, want 100", count)
		}

		var ratio float64
		if err := unmarshalFromMap(data, "ratio", &ratio); err != nil {
			t.Errorf("ratio unmarshal error: %v", err)
		}
		if ratio != 3.14 {
			t.Errorf("ratio = %f, want 3.14", ratio)
		}
	})

	t.Run("handles arrays", func(t *testing.T) {
		data := map[string]any{
			"items": []any{"a", "b", "c"},
		}

		var items []string
		err := unmarshalFromMap(data, "items", &items)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if len(items) != 3 {
			t.Errorf("got %d items, want 3", len(items))
		}

		if items[0] != "a" || items[1] != "b" || items[2] != "c" {
			t.Errorf("got %v, want [a b c]", items)
		}
	})
}

func TestParseFrontmatterConfig(t *testing.T) {
	t.Run("parses minimal workflow config", func(t *testing.T) {
		frontmatter := map[string]any{
			"name":   "test-workflow",
			"engine": "claude",
		}

		config, err := ParseFrontmatterConfig(frontmatter)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if config.Name != "test-workflow" {
			t.Errorf("Name = %q, want %q", config.Name, "test-workflow")
		}

		if config.Engine != "claude" {
			t.Errorf("Engine = %q, want %q", config.Engine, "claude")
		}
	})

	t.Run("parses complete workflow config", func(t *testing.T) {
		frontmatter := map[string]any{
			"name":        "full-workflow",
			"description": "A complete workflow",
			"engine":      "copilot",
			"source":      "owner/repo/path@main",
			"tracker-id":  "test-tracker-123",
			"tools": map[string]any{
				"bash": map[string]any{
					"enabled": true,
				},
			},
			"mcp-servers": map[string]any{
				"github": map[string]any{
					"mode": "remote",
				},
			},
			"safe-outputs": map[string]any{
				"create-issue": map[string]any{
					"enabled": true,
				},
			},
		}

		config, err := ParseFrontmatterConfig(frontmatter)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Check core fields
		if config.Name != "full-workflow" {
			t.Errorf("Name = %q, want %q", config.Name, "full-workflow")
		}

		if config.Description != "A complete workflow" {
			t.Errorf("Description = %q, want %q", config.Description, "A complete workflow")
		}

		if config.Engine != "copilot" {
			t.Errorf("Engine = %q, want %q", config.Engine, "copilot")
		}

		if config.Source != "owner/repo/path@main" {
			t.Errorf("Source = %q, want %q", config.Source, "owner/repo/path@main")
		}

		if config.TrackerID != "test-tracker-123" {
			t.Errorf("TrackerID = %q, want %q", config.TrackerID, "test-tracker-123")
		}

		// Check nested configuration sections
		if config.Tools == nil {
			t.Error("Tools should not be nil")
		}

		if config.MCPServers == nil {
			t.Error("MCPServers should not be nil")
		}

		if config.SafeOutputs == nil {
			t.Error("SafeOutputs should not be nil")
		}
	})

	t.Run("handles timeout-minutes as int", func(t *testing.T) {
		frontmatter := map[string]any{
			"timeout-minutes": 60,
		}

		config, err := ParseFrontmatterConfig(frontmatter)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if config.TimeoutMinutes != 60 {
			t.Errorf("TimeoutMinutes = %d, want 60", config.TimeoutMinutes)
		}
	})

	t.Run("handles empty frontmatter", func(t *testing.T) {
		frontmatter := map[string]any{}

		config, err := ParseFrontmatterConfig(frontmatter)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if config.Name != "" {
			t.Errorf("Name should be empty, got %q", config.Name)
		}

		if config.Tools != nil {
			t.Errorf("Tools should be nil for empty frontmatter, got %v", config.Tools)
		}
	})

	t.Run("handles network configuration", func(t *testing.T) {
		frontmatter := map[string]any{
			"network": map[string]any{
				"allowed": []any{"github.com", "api.github.com"},
				"firewall": map[string]any{
					"enabled": true,
				},
			},
		}

		config, err := ParseFrontmatterConfig(frontmatter)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if config.Network == nil {
			t.Fatal("Network should not be nil")
		}

		// Check that allowed domains are present
		if len(config.Network.Allowed) != 2 {
			t.Errorf("expected 2 allowed domains, got %d", len(config.Network.Allowed))
		}
	})

	t.Run("handles sandbox configuration", func(t *testing.T) {
		frontmatter := map[string]any{
			"sandbox": map[string]any{
				"agent": map[string]any{
					"type": "awf",
				},
			},
		}

		config, err := ParseFrontmatterConfig(frontmatter)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if config.Sandbox == nil {
			t.Fatal("Sandbox should not be nil")
		}
	})

	t.Run("handles jobs configuration", func(t *testing.T) {
		frontmatter := map[string]any{
			"jobs": map[string]any{
				"test-job": map[string]any{
					"runs-on": "ubuntu-latest",
					"steps": []any{
						map[string]any{
							"run": "echo hello",
						},
					},
				},
			},
		}

		config, err := ParseFrontmatterConfig(frontmatter)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if config.Jobs == nil {
			t.Fatal("Jobs should not be nil")
		}

		if _, ok := config.Jobs["test-job"]; !ok {
			t.Error("test-job should exist in Jobs")
		}
	})

	t.Run("preserves complex nested structures", func(t *testing.T) {
		frontmatter := map[string]any{
			"safe-jobs": map[string]any{
				"custom-job": map[string]any{
					"conditions": []any{
						map[string]any{
							"field": "status",
							"value": "success",
						},
					},
				},
			},
		}

		config, err := ParseFrontmatterConfig(frontmatter)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if config.SafeJobs == nil {
			t.Fatal("SafeJobs should not be nil")
		}

		customJob, ok := config.SafeJobs["custom-job"]
		if !ok {
			t.Fatal("custom-job should exist")
		}

		if customJob == nil {
			t.Fatal("custom-job should not be nil")
		}
	})
}

func TestFrontmatterConfigFieldExtraction(t *testing.T) {
	t.Run("extracts tools using struct", func(t *testing.T) {
		frontmatter := map[string]any{
			"tools": map[string]any{
				"bash": map[string]any{
					"enabled": true,
				},
				"playwright": map[string]any{
					"version": "v1.41.0",
				},
			},
		}

		config, err := ParseFrontmatterConfig(frontmatter)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify tools can be accessed via ToMap()
		if config.Tools == nil {
			t.Fatal("Tools should not be nil")
		}

		toolsMap := config.Tools.ToMap()
		if len(toolsMap) < 2 {
			t.Errorf("expected at least 2 tools, got %d", len(toolsMap))
		}

		if _, ok := toolsMap["bash"]; !ok {
			t.Error("bash tool should exist")
		}

		if _, ok := toolsMap["playwright"]; !ok {
			t.Error("playwright tool should exist")
		}
	})

	t.Run("extracts mcp-servers using struct", func(t *testing.T) {
		frontmatter := map[string]any{
			"mcp-servers": map[string]any{
				"github": map[string]any{
					"mode":     "remote",
					"toolsets": []any{"repos", "issues"},
				},
			},
		}

		config, err := ParseFrontmatterConfig(frontmatter)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(config.MCPServers) != 1 {
			t.Errorf("expected 1 mcp server, got %d", len(config.MCPServers))
		}

		if _, ok := config.MCPServers["github"]; !ok {
			t.Error("github mcp server should exist")
		}
	})

	t.Run("extracts runtimes using struct", func(t *testing.T) {
		frontmatter := map[string]any{
			"runtimes": map[string]any{
				"node": map[string]any{
					"version": "20",
				},
			},
		}

		config, err := ParseFrontmatterConfig(frontmatter)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(config.Runtimes) != 1 {
			t.Errorf("expected 1 runtime, got %d", len(config.Runtimes))
		}

		if _, ok := config.Runtimes["node"]; !ok {
			t.Error("node runtime should exist")
		}
	})
}

func TestFrontmatterConfigBackwardCompatibility(t *testing.T) {
	// This test ensures that the new structured types work with existing
	// frontmatter extraction patterns used throughout the codebase

	t.Run("compatible with extractMapFromFrontmatter pattern", func(t *testing.T) {
		frontmatter := map[string]any{
			"tools": map[string]any{
				"bash": map[string]any{
					"enabled": true,
				},
			},
		}

		// Old pattern: extractMapFromFrontmatter
		oldResult := extractMapFromFrontmatter(frontmatter, "tools")

		// New pattern: Parse then access field
		config, err := ParseFrontmatterConfig(frontmatter)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Convert tools back to map for comparison
		newResult := config.Tools.ToMap()

		// Both should have the same tool
		if _, oldOk := oldResult["bash"]; !oldOk {
			t.Error("old pattern missing bash tool")
		}
		if _, newOk := newResult["bash"]; !newOk {
			t.Error("new pattern missing bash tool")
		}
	})

	t.Run("supports round-trip conversion", func(t *testing.T) {
		originalFrontmatter := map[string]any{
			"name":            "test-workflow",
			"engine":          "copilot",
			"description":     "A test workflow",
			"tracker-id":      "test-tracker-12345678",
			"timeout-minutes": 30,
			"on": map[string]any{
				"issues": map[string]any{
					"types": []any{"opened", "labeled"},
				},
			},
			"env": map[string]string{
				"MY_VAR": "value",
			},
		}

		// Parse to struct
		config, err := ParseFrontmatterConfig(originalFrontmatter)
		if err != nil {
			t.Fatalf("parse error: %v", err)
		}

		// Convert back to map
		reconstructed := config.ToMap()

		// Verify key fields are preserved
		if reconstructed["name"] != "test-workflow" {
			t.Errorf("name mismatch: got %v", reconstructed["name"])
		}
		if reconstructed["engine"] != "copilot" {
			t.Errorf("engine mismatch: got %v", reconstructed["engine"])
		}
		if reconstructed["description"] != "A test workflow" {
			t.Errorf("description mismatch: got %v", reconstructed["description"])
		}
		if reconstructed["tracker-id"] != "test-tracker-12345678" {
			t.Errorf("tracker-id mismatch: got %v", reconstructed["tracker-id"])
		}
		if reconstructed["timeout-minutes"] != 30 {
			t.Errorf("timeout-minutes mismatch: got %v", reconstructed["timeout-minutes"])
		}

		// Verify nested structures
		if reconstructed["on"] == nil {
			t.Error("on should not be nil")
		}
		if reconstructed["env"] == nil {
			t.Error("env should not be nil")
		}
	})

	t.Run("preserves strongly-typed fields", func(t *testing.T) {
		frontmatter := map[string]any{
			"network": map[string]any{
				"allowed": []any{"github.com", "api.github.com"},
			},
			"sandbox": map[string]any{
				"agent": map[string]any{
					"type": "awf",
				},
			},
			"safe-outputs": map[string]any{
				"create-issue": map[string]any{
					"max": 5,
				},
			},
			"safe-inputs": map[string]any{
				"mode": "http",
			},
		}

		config, err := ParseFrontmatterConfig(frontmatter)
		if err != nil {
			t.Fatalf("parse error: %v", err)
		}

		// Verify strongly-typed fields are populated
		if config.Network == nil {
			t.Error("Network should be strongly typed")
		}
		if config.Sandbox == nil {
			t.Error("Sandbox should be strongly typed")
		}
		if config.SafeOutputs == nil {
			t.Error("SafeOutputs should be strongly typed")
		}
		if config.SafeInputs == nil {
			t.Error("SafeInputs should be strongly typed")
		}

		// Convert back and verify they're preserved
		reconstructed := config.ToMap()
		if reconstructed["network"] == nil {
			t.Error("network should be preserved in ToMap")
		}
		if reconstructed["sandbox"] == nil {
			t.Error("sandbox should be preserved in ToMap")
		}
		if reconstructed["safe-outputs"] == nil {
			t.Error("safe-outputs should be preserved in ToMap")
		}
		if reconstructed["safe-inputs"] == nil {
			t.Error("safe-inputs should be preserved in ToMap")
		}
	})
}
