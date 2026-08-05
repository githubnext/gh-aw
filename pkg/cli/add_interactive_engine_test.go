//go:build !integration

package cli

import (
	"testing"

	"charm.land/huh/v2"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
)

func TestApplyCopilotAuthMethodChoice(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		authMethod      string
		wantCopilotReqs bool
	}{
		{
			name:            "copilot-requests sets UseCopilotRequests true",
			authMethod:      "copilot-requests",
			wantCopilotReqs: true,
		},
		{
			name:            "pat sets UseCopilotRequests false",
			authMethod:      "pat",
			wantCopilotReqs: false,
		},
		{
			name:            "empty value (form cancelled) sets UseCopilotRequests false",
			authMethod:      "",
			wantCopilotReqs: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cfg := &AddInteractiveConfig{}
			cfg.applyCopilotAuthMethodChoice(tc.authMethod)
			assert.Equal(t, tc.wantCopilotReqs, cfg.UseCopilotRequests)
		})
	}
}

func TestApplyCopilotAuthMethodChoice_ReEntryClearsOldValue(t *testing.T) {
	t.Parallel()
	cfg := &AddInteractiveConfig{}

	// First selection: copilot-requests
	cfg.applyCopilotAuthMethodChoice("copilot-requests")
	assert.True(t, cfg.UseCopilotRequests)

	// User changes selection to PAT — old value must not persist
	cfg.applyCopilotAuthMethodChoice("pat")
	assert.False(t, cfg.UseCopilotRequests)
}

func TestPrioritizeEngineOption(t *testing.T) {
	t.Parallel()
	options := []huh.Option[string]{
		huh.NewOption("B", "b"),
		huh.NewOption("A", "a"),
	}

	prioritizeEngineOption(options, "a")
	assert.Equal(t, "a", options[0].Value)
	assert.Equal(t, "b", options[1].Value)

	prioritizeEngineOption(options, "missing")
	assert.Equal(t, "a", options[0].Value)
	assert.Equal(t, "b", options[1].Value)
}

func TestDetermineDefaultEngine(t *testing.T) {
	t.Parallel()
	makeSecrets := func(keys ...string) map[string]struct{} {
		m := make(map[string]struct{}, len(keys))
		for _, k := range keys {
			m[k] = struct{}{}
		}
		return m
	}

	tests := []struct {
		name                    string
		engineOverride          string
		existingSecrets         map[string]struct{}
		workflowSpecifiedEngine string
		want                    string
	}{
		{
			name:                    "engine override takes priority over everything",
			engineOverride:          "codex",
			existingSecrets:         makeSecrets(constants.AnthropicAPIKey),
			workflowSpecifiedEngine: "claude",
			want:                    "codex",
		},
		{
			name:                    "non-default secret overrides workflow preference",
			existingSecrets:         makeSecrets(constants.AnthropicAPIKey),
			workflowSpecifiedEngine: "codex",
			want:                    string(constants.ClaudeEngine),
		},
		{
			name:                    "default-engine (Copilot) secret falls through to workflow preference",
			existingSecrets:         makeSecrets(constants.CopilotGitHubToken),
			workflowSpecifiedEngine: string(constants.ClaudeEngine),
			want:                    string(constants.ClaudeEngine),
		},
		{
			name:                    "default-engine secret with no workflow preference stays default",
			existingSecrets:         makeSecrets(constants.CopilotGitHubToken),
			workflowSpecifiedEngine: "",
			want:                    string(constants.DefaultEngine),
		},
		{
			name:                    "no secret defers to workflow preference",
			existingSecrets:         nil,
			workflowSpecifiedEngine: string(constants.ClaudeEngine),
			want:                    string(constants.ClaudeEngine),
		},
		{
			name:                    "no secret no workflow returns default engine",
			existingSecrets:         nil,
			workflowSpecifiedEngine: "",
			want:                    string(constants.DefaultEngine),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cfg := &AddInteractiveConfig{
				EngineOverride:  tc.engineOverride,
				existingSecrets: tc.existingSecrets,
			}
			got := cfg.determineDefaultEngine(tc.workflowSpecifiedEngine)
			assert.Equal(t, tc.want, got)
		})
	}
}
