package workflow

import (
	"testing"
)

func TestGetToolFromWorkflowData(t *testing.T) {
	t.Run("returns tool from ParsedTools when available", func(t *testing.T) {
		data := &WorkflowData{
			Tools: map[string]any{
				"github": map[string]any{"allowed": []any{"get_issue"}},
			},
			ParsedTools: NewTools(map[string]any{
				"github": map[string]any{"allowed": []any{"get_issue"}},
			}),
		}

		result := getToolFromWorkflowData(data, "github")
		if result == nil {
			t.Error("expected non-nil result")
		}
	})

	t.Run("returns tool from Tools map when ParsedTools is nil", func(t *testing.T) {
		data := &WorkflowData{
			Tools: map[string]any{
				"github": map[string]any{"allowed": []any{"get_issue"}},
			},
			ParsedTools: nil,
		}

		result := getToolFromWorkflowData(data, "github")
		if result == nil {
			t.Error("expected non-nil result")
		}
	})
}

func TestHasToolInWorkflowData(t *testing.T) {
	t.Run("checks ParsedTools when available", func(t *testing.T) {
		data := &WorkflowData{
			Tools: map[string]any{
				"github": nil,
			},
			ParsedTools: NewTools(map[string]any{
				"github": nil,
			}),
		}

		if !hasToolInWorkflowData(data, "github") {
			t.Error("expected github tool to be present")
		}

		if hasToolInWorkflowData(data, "bash") {
			t.Error("expected bash tool to be absent")
		}
	})

	t.Run("checks Tools map when ParsedTools is nil", func(t *testing.T) {
		data := &WorkflowData{
			Tools: map[string]any{
				"github": nil,
			},
			ParsedTools: nil,
		}

		if !hasToolInWorkflowData(data, "github") {
			t.Error("expected github tool to be present")
		}

		if hasToolInWorkflowData(data, "bash") {
			t.Error("expected bash tool to be absent")
		}
	})
}

func TestGetToolNamesFromWorkflowData(t *testing.T) {
	t.Run("returns names from ParsedTools when available", func(t *testing.T) {
		data := &WorkflowData{
			Tools: map[string]any{
				"github": nil,
				"bash":   []any{"echo"},
			},
			ParsedTools: NewTools(map[string]any{
				"github": nil,
				"bash":   []any{"echo"},
			}),
		}

		names := getToolNamesFromWorkflowData(data)
		if len(names) != 2 {
			t.Errorf("expected 2 names, got %d", len(names))
		}
	})

	t.Run("returns names from Tools map when ParsedTools is nil", func(t *testing.T) {
		data := &WorkflowData{
			Tools: map[string]any{
				"github": nil,
				"bash":   []any{"echo"},
			},
			ParsedTools: nil,
		}

		names := getToolNamesFromWorkflowData(data)
		if len(names) != 2 {
			t.Errorf("expected 2 names, got %d", len(names))
		}
	})
}

func TestGetGitHubConfigFromWorkflowData(t *testing.T) {
	t.Run("returns typed config from ParsedTools when available", func(t *testing.T) {
		data := &WorkflowData{
			Tools: map[string]any{
				"github": map[string]any{
					"allowed": []any{"get_issue"},
					"mode":    "remote",
				},
			},
			ParsedTools: NewTools(map[string]any{
				"github": map[string]any{
					"allowed": []any{"get_issue"},
					"mode":    "remote",
				},
			}),
		}

		result := getGitHubConfigFromWorkflowData(data)
		if result == nil {
			t.Error("expected non-nil result")
		}

		// When ParsedTools is available, we get a *GitHubToolConfig
		if config, ok := result.(*GitHubToolConfig); ok {
			if config.Mode != "remote" {
				t.Errorf("expected mode 'remote', got %q", config.Mode)
			}
		}
	})

	t.Run("returns raw value from Tools map when ParsedTools is nil", func(t *testing.T) {
		data := &WorkflowData{
			Tools: map[string]any{
				"github": map[string]any{
					"allowed": []any{"get_issue"},
					"mode":    "remote",
				},
			},
			ParsedTools: nil,
		}

		result := getGitHubConfigFromWorkflowData(data)
		if result == nil {
			t.Error("expected non-nil result")
		}

		// When ParsedTools is nil, we get the raw map
		if _, ok := result.(map[string]any); !ok {
			t.Error("expected raw map[string]any")
		}
	})
}
