//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApplyCopilotAuthMethodChoice(t *testing.T) {
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
			cfg := &AddInteractiveConfig{}
			cfg.applyCopilotAuthMethodChoice(tc.authMethod)
			assert.Equal(t, tc.wantCopilotReqs, cfg.UseCopilotRequests)
		})
	}
}

func TestApplyCopilotAuthMethodChoice_ReEntryClearsOldValue(t *testing.T) {
	cfg := &AddInteractiveConfig{}

	// First selection: copilot-requests
	cfg.applyCopilotAuthMethodChoice("copilot-requests")
	assert.True(t, cfg.UseCopilotRequests)

	// User changes selection to PAT — old value must not persist
	cfg.applyCopilotAuthMethodChoice("pat")
	assert.False(t, cfg.UseCopilotRequests)
}
