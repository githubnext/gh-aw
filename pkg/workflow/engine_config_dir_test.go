//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGetEngineSkillDir validates that GetEngineSkillDir returns the correct
// directory for each engine by delegating to the engine's AgentManifestPathPrefixes.
func TestGetEngineSkillDir(t *testing.T) {
	tests := []struct {
		name     string
		engineID string
		expected string
	}{
		{name: "claude engine uses .claude/skills", engineID: "claude", expected: ".claude/skills"},
		{name: "codex engine uses .codex/skills", engineID: "codex", expected: ".codex/skills"},
		{name: "gemini engine uses .gemini/skills", engineID: "gemini", expected: ".gemini/skills"},
		{name: "crush engine uses .crush/skills", engineID: "crush", expected: ".crush/skills"},
		{name: "opencode engine uses .opencode/skills", engineID: "opencode", expected: ".opencode/skills"},
		{name: "antigravity engine uses .antigravity/skills", engineID: "antigravity", expected: ".antigravity/skills"},
		{name: "pi engine uses .pi/skills", engineID: "pi", expected: ".pi/skills"},
		{name: "copilot engine uses .github/skills", engineID: "copilot", expected: ".github/skills"},
		{name: "unknown engine falls back to .github/skills", engineID: "unknown", expected: ".github/skills"},
		{name: "empty engine ID falls back to .github/skills", engineID: "", expected: ".github/skills"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
		name     string
		engineID string
		expected string
	}{
		{name: "claude engine uses .claude/agents", engineID: "claude", expected: ".claude/agents"},
		{name: "codex engine uses .codex/agents", engineID: "codex", expected: ".codex/agents"},
		{name: "gemini engine uses .gemini/agents", engineID: "gemini", expected: ".gemini/agents"},
		{name: "crush engine uses .crush/agents", engineID: "crush", expected: ".crush/agents"},
		{name: "opencode engine uses .opencode/agents", engineID: "opencode", expected: ".opencode/agents"},
		{name: "antigravity engine uses .antigravity/agents", engineID: "antigravity", expected: ".antigravity/agents"},
		{name: "pi engine uses .pi/agents", engineID: "pi", expected: ".pi/agents"},
		{name: "copilot engine uses .github/agents", engineID: "copilot", expected: ".github/agents"},
		{name: "unknown engine falls back to .github/agents", engineID: "unknown", expected: ".github/agents"},
		{name: "empty engine ID falls back to .github/agents", engineID: "", expected: ".github/agents"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetEngineSubAgentDir(tt.engineID)
			assert.Equal(t, tt.expected, result,
				"GetEngineSubAgentDir(%q) should return correct sub-agent directory", tt.engineID)
		})
	}
}
