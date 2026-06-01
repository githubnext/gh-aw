//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetEngineSkillDir validates that GetEngineSkillDir returns the correct
// directory for each engine by delegating to the engine's AgentManifestPathPrefixes.
func TestGetEngineSkillDir(t *testing.T) {
	tests := []struct {
		name       string
		engineID   string
		expected   string
		registered bool // whether the engine must be in the global registry
	}{
		{name: "claude engine uses .claude/skills", engineID: "claude", expected: ".claude/skills", registered: true},
		{name: "codex engine uses .codex/skills", engineID: "codex", expected: ".codex/skills", registered: true},
		{name: "gemini engine uses .gemini/skills", engineID: "gemini", expected: ".gemini/skills", registered: true},
		{name: "crush engine uses .crush/skills", engineID: "crush", expected: ".crush/skills", registered: true},
		{name: "opencode engine uses .opencode/skills", engineID: "opencode", expected: ".opencode/skills", registered: true},
		{name: "antigravity engine uses .antigravity/skills", engineID: "antigravity", expected: ".antigravity/skills", registered: true},
		{name: "pi engine uses .pi/skills", engineID: "pi", expected: ".pi/skills", registered: true},
		{name: "copilot engine uses .github/skills", engineID: "copilot", expected: ".github/skills", registered: true},
		{name: "unknown engine falls back to .github/skills", engineID: "unknown", expected: ".github/skills", registered: false},
		{name: "empty engine ID falls back to .github/skills", engineID: "", expected: ".github/skills", registered: false},
	}

	registry := GetGlobalEngineRegistry()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.registered {
				require.True(t, registry.IsValidEngine(tt.engineID),
					"engine %q must be registered in the global registry", tt.engineID)
			}
			result := GetEngineSkillDir(tt.engineID)
			assert.Equal(t, tt.expected, result,
				"GetEngineSkillDir(%q) should return correct skill directory", tt.engineID)
		})
	}
}

// TestGetEngineSubAgentDir validates that GetEngineSubAgentDir returns the correct
// directory for each engine by delegating to the engine's AgentManifestPathPrefixes.
func TestGetEngineSubAgentDir(t *testing.T) {
	tests := []struct {
		name       string
		engineID   string
		expected   string
		registered bool // whether the engine must be in the global registry
	}{
		{name: "claude engine uses .claude/agents", engineID: "claude", expected: ".claude/agents", registered: true},
		{name: "codex engine uses .codex/agents", engineID: "codex", expected: ".codex/agents", registered: true},
		{name: "gemini engine uses .gemini/agents", engineID: "gemini", expected: ".gemini/agents", registered: true},
		{name: "crush engine uses .crush/agents", engineID: "crush", expected: ".crush/agents", registered: true},
		{name: "opencode engine uses .opencode/agents", engineID: "opencode", expected: ".opencode/agents", registered: true},
		{name: "antigravity engine uses .antigravity/agents", engineID: "antigravity", expected: ".antigravity/agents", registered: true},
		{name: "pi engine uses .pi/agents", engineID: "pi", expected: ".pi/agents", registered: true},
		{name: "copilot engine uses .github/agents", engineID: "copilot", expected: ".github/agents", registered: true},
		{name: "unknown engine falls back to .github/agents", engineID: "unknown", expected: ".github/agents", registered: false},
		{name: "empty engine ID falls back to .github/agents", engineID: "", expected: ".github/agents", registered: false},
	}

	registry := GetGlobalEngineRegistry()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.registered {
				require.True(t, registry.IsValidEngine(tt.engineID),
					"engine %q must be registered in the global registry", tt.engineID)
			}
			result := GetEngineSubAgentDir(tt.engineID)
			assert.Equal(t, tt.expected, result,
				"GetEngineSubAgentDir(%q) should return correct sub-agent directory", tt.engineID)
		})
	}
}
