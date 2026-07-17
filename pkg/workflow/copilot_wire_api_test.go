//go:build !integration

package workflow

import (
	"strings"
	"testing"
)

func TestCopilotWireAPIForModel(t *testing.T) {
	tests := []struct {
		name    string
		model   string
		wantAPI string
	}{
		{name: "responses model gpt-5-mini", model: "gpt-5-mini", wantAPI: wireAPIResponses},
		{name: "responses model mai-code-1-flash-picker", model: "mai-code-1-flash-picker", wantAPI: wireAPIResponses},
		{name: "responses model gpt-5.4", model: "gpt-5.4", wantAPI: wireAPIResponses},
		{name: "completions model gemini-2.5-pro", model: "gemini-2.5-pro", wantAPI: wireAPICompletions},
		{name: "completions model raptor-mini", model: "raptor-mini", wantAPI: wireAPICompletions},
		{name: "unknown model returns empty", model: "unknown-model-xyz", wantAPI: ""},
		{name: "empty model returns empty", model: "", wantAPI: ""},
		{name: "case-insensitive lookup", model: "GPT-5-MINI", wantAPI: wireAPIResponses},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := copilotWireAPIForModel(tt.model)
			if got != tt.wantAPI {
				t.Errorf("copilotWireAPIForModel(%q) = %q, want %q", tt.model, got, tt.wantAPI)
			}
		})
	}
}

func TestBuildCopilotWireAPIResolutionScript(t *testing.T) {
	script := buildCopilotWireAPIResolutionScript()

	if script == "" {
		t.Fatal("expected non-empty wire API resolution script")
	}

	// Script must guard against overwriting a user-supplied value.
	if !strings.Contains(script, "COPILOT_PROVIDER_WIRE_API:-") {
		t.Error("script must check COPILOT_PROVIDER_WIRE_API before overwriting")
	}

	// Script must use $COPILOT_MODEL as the input.
	if !strings.Contains(script, "COPILOT_MODEL") {
		t.Error("script must read COPILOT_MODEL env var")
	}

	// Script must handle at least one responses model.
	if !strings.Contains(script, wireAPIResponses) {
		t.Errorf("script must include %q arm in case statement", wireAPIResponses)
	}

	// A known responses model must appear in the case pattern.
	if !strings.Contains(script, "gpt-5-mini") {
		t.Error("script must include 'gpt-5-mini' in responses case arm")
	}

	// A known responses model must appear in the case pattern.
	if !strings.Contains(script, "mai-code-1-flash-picker") {
		t.Error("script must include 'mai-code-1-flash-picker' in responses case arm")
	}

	// Script must export the variable.
	if !strings.Contains(script, "export COPILOT_PROVIDER_WIRE_API") {
		t.Error("script must export COPILOT_PROVIDER_WIRE_API")
	}

	t.Logf("Generated wire API resolution script:\n%s", script)
}

// TestAddCopilotModelEnvSetsWireAPIForResponsesModel verifies that addCopilotModelEnv
// automatically sets COPILOT_PROVIDER_WIRE_API when a static model has wire_api=responses.
func TestAddCopilotModelEnvSetsWireAPIForResponsesModel(t *testing.T) {
	engine := NewCopilotEngine()
	env := map[string]string{}
	workflowData := &WorkflowData{
		EngineConfig: &EngineConfig{
			Model: "gpt-5-mini",
		},
	}

	engine.addCopilotModelEnv(env, workflowData, true, "")

	if env["COPILOT_MODEL"] != "gpt-5-mini" {
		t.Errorf("expected COPILOT_MODEL=gpt-5-mini, got %q", env["COPILOT_MODEL"])
	}
	if env["COPILOT_PROVIDER_WIRE_API"] != wireAPIResponses {
		t.Errorf("expected COPILOT_PROVIDER_WIRE_API=%s, got %q", wireAPIResponses, env["COPILOT_PROVIDER_WIRE_API"])
	}
}

// TestAddCopilotModelEnvDoesNotSetWireAPIForCompletionsModel verifies that
// addCopilotModelEnv does NOT set COPILOT_PROVIDER_WIRE_API for models whose
// wire_api is completions (the Copilot CLI default).
func TestAddCopilotModelEnvDoesNotSetWireAPIForCompletionsModel(t *testing.T) {
	engine := NewCopilotEngine()
	env := map[string]string{}
	workflowData := &WorkflowData{
		EngineConfig: &EngineConfig{
			Model: "gemini-2.5-pro",
		},
	}

	engine.addCopilotModelEnv(env, workflowData, true, "")

	if env["COPILOT_MODEL"] != "gemini-2.5-pro" {
		t.Errorf("expected COPILOT_MODEL=gemini-2.5-pro, got %q", env["COPILOT_MODEL"])
	}
	if _, set := env["COPILOT_PROVIDER_WIRE_API"]; set {
		t.Errorf("expected COPILOT_PROVIDER_WIRE_API to be unset for completions model, got %q", env["COPILOT_PROVIDER_WIRE_API"])
	}
}

// TestAddCopilotModelEnvDoesNotSetWireAPIForUnknownModel verifies that
// addCopilotModelEnv does NOT set COPILOT_PROVIDER_WIRE_API when the model
// is not in the catalog (e.g. a BYOK custom deployment name).
func TestAddCopilotModelEnvDoesNotSetWireAPIForUnknownModel(t *testing.T) {
	engine := NewCopilotEngine()
	env := map[string]string{}
	workflowData := &WorkflowData{
		EngineConfig: &EngineConfig{
			Model: "o4-mini-aw",
		},
	}

	engine.addCopilotModelEnv(env, workflowData, true, "")

	if env["COPILOT_MODEL"] != "o4-mini-aw" {
		t.Errorf("expected COPILOT_MODEL=o4-mini-aw, got %q", env["COPILOT_MODEL"])
	}
	if _, set := env["COPILOT_PROVIDER_WIRE_API"]; set {
		t.Errorf("expected COPILOT_PROVIDER_WIRE_API to be unset for unknown model, got %q", env["COPILOT_PROVIDER_WIRE_API"])
	}
}
