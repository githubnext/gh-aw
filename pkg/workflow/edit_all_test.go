package workflow

import (
	"testing"
)

func TestEditAllTool(t *testing.T) {
	t.Run("parses edit-all tool", func(t *testing.T) {
		toolsMap := map[string]any{
			"edit-all": nil,
		}

		tools := NewTools(toolsMap)
		if tools == nil {
			t.Fatal("expected non-nil tools")
		}

		if !tools.HasTool("edit-all") {
			t.Error("expected edit-all tool to be set")
		}

		if tools.EditAll == nil {
			t.Error("expected EditAll field to be set")
		}

		names := tools.GetToolNames()
		if len(names) != 1 {
			t.Errorf("expected 1 tool, got %d: %v", len(names), names)
		}

		if names[0] != "edit-all" {
			t.Errorf("expected tool name 'edit-all', got '%s'", names[0])
		}
	})

	t.Run("parses both edit and edit-all tools", func(t *testing.T) {
		toolsMap := map[string]any{
			"edit":     nil,
			"edit-all": nil,
		}

		tools := NewTools(toolsMap)
		if tools == nil {
			t.Fatal("expected non-nil tools")
		}

		if !tools.HasTool("edit") {
			t.Error("expected edit tool to be set")
		}

		if !tools.HasTool("edit-all") {
			t.Error("expected edit-all tool to be set")
		}

		if tools.Edit == nil {
			t.Error("expected Edit field to be set")
		}

		if tools.EditAll == nil {
			t.Error("expected EditAll field to be set")
		}

		names := tools.GetToolNames()
		if len(names) != 2 {
			t.Errorf("expected 2 tools, got %d: %v", len(names), names)
		}
	})

	t.Run("edit-all not confused with custom tools", func(t *testing.T) {
		toolsMap := map[string]any{
			"edit-all":   nil,
			"my-custom":  map[string]any{"command": "node"},
			"another":    map[string]any{"type": "http"},
		}

		tools := NewTools(toolsMap)
		if tools == nil {
			t.Fatal("expected non-nil tools")
		}

		if !tools.HasTool("edit-all") {
			t.Error("expected edit-all tool to be set")
		}

		if len(tools.Custom) != 2 {
			t.Errorf("expected 2 custom tools, got %d", len(tools.Custom))
		}

		if _, exists := tools.Custom["edit-all"]; exists {
			t.Error("edit-all should not be in Custom map")
		}
	})

	t.Run("HasTool returns false for edit-all when not set", func(t *testing.T) {
		toolsMap := map[string]any{
			"github": nil,
		}

		tools := NewTools(toolsMap)
		if tools.HasTool("edit-all") {
			t.Error("expected edit-all tool to not be set")
		}
	})
}

func TestEditDeprecationWithEditAll(t *testing.T) {
	t.Run("edit tool is deprecated but still works", func(t *testing.T) {
		toolsMap := map[string]any{
			"edit": nil,
		}

		tools := NewTools(toolsMap)
		if !tools.HasTool("edit") {
			t.Error("expected edit tool to still work for backward compatibility")
		}

		if tools.Edit == nil {
			t.Error("expected Edit field to be set")
		}
	})

	t.Run("edit-all is the preferred tool", func(t *testing.T) {
		toolsMap := map[string]any{
			"edit-all": nil,
		}

		tools := NewTools(toolsMap)
		if !tools.HasTool("edit-all") {
			t.Error("expected edit-all tool to be set")
		}

		if tools.EditAll == nil {
			t.Error("expected EditAll field to be set")
		}
	})
}
