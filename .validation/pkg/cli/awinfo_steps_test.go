//go:build !integration

package cli

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAwInfoStepsFieldParsing(t *testing.T) {
	tests := []struct {
		name          string
		jsonContent   string
		expectedSteps AwInfoSteps
	}{
		{
			name: "firewall enabled with squid",
			jsonContent: `{
				"engine_id": "copilot",
				"engine_name": "Copilot",
				"model": "gpt-4",
				"version": "1.0",
				"workflow_name": "test-workflow",
				"staged": false,
				"steps": {
					"firewall": "squid"
				},
				"created_at": "2025-01-27T15:00:00Z"
			}`,
			expectedSteps: AwInfoSteps{
				Firewall: "squid",
			},
		},
		{
			name: "firewall disabled (empty string)",
			jsonContent: `{
				"engine_id": "copilot",
				"engine_name": "Copilot",
				"model": "gpt-4",
				"version": "1.0",
				"workflow_name": "test-workflow",
				"staged": false,
				"steps": {
					"firewall": ""
				},
				"created_at": "2025-01-27T15:00:00Z"
			}`,
			expectedSteps: AwInfoSteps{
				Firewall: "",
			},
		},
		{
			name: "no steps field (backward compatibility)",
			jsonContent: `{
				"engine_id": "claude",
				"engine_name": "Claude",
				"model": "claude-3-sonnet",
				"version": "20240620",
				"workflow_name": "test-workflow",
				"staged": false,
				"created_at": "2025-01-27T15:00:00Z"
			}`,
			expectedSteps: AwInfoSteps{
				Firewall: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var info AwInfo
			err := json.Unmarshal([]byte(tt.jsonContent), &info)
			require.NoError(t, err, "Failed to unmarshal JSON")

			assert.Equal(t, tt.expectedSteps.Firewall, info.Steps.Firewall, tt.name)
		})
	}
}

func TestAwInfoStepsMarshaling(t *testing.T) {
	original := AwInfo{
		EngineID:     "copilot",
		EngineName:   "Copilot",
		Model:        "gpt-4",
		Version:      "1.0",
		WorkflowName: "test-workflow",
		Staged:       false,
		Steps: AwInfoSteps{
			Firewall: "squid",
		},
		CreatedAt: "2025-01-27T15:00:00Z",
	}

	jsonData, err := json.Marshal(original)
	require.NoError(t, err, "Failed to marshal AwInfo")

	var roundTripped AwInfo
	err = json.Unmarshal(jsonData, &roundTripped)
	require.NoError(t, err, "Failed to unmarshal marshaled JSON")

	assert.Equal(t, original.Steps.Firewall, roundTripped.Steps.Firewall, "Steps.Firewall should survive JSON round-trip")
}

func TestGetFirewallVersion(t *testing.T) {
	tests := []struct {
		name     string
		info     AwInfo
		expected string
	}{
		{
			name:     "prefer new awf_version field",
			info:     AwInfo{AwfVersion: "1.2.3", FirewallVersion: "0.9.0"},
			expected: "1.2.3",
		},
		{
			name:     "fallback to firewall_version when awf_version is empty",
			info:     AwInfo{AwfVersion: "", FirewallVersion: "0.9.0"},
			expected: "0.9.0",
		},
		{
			name:     "empty when both fields are empty",
			info:     AwInfo{AwfVersion: "", FirewallVersion: ""},
			expected: "",
		},
		{
			name:     "only new awf_version field set",
			info:     AwInfo{AwfVersion: "2.0.0"},
			expected: "2.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.info.GetFirewallVersion(), tt.name)
		})
	}
}
