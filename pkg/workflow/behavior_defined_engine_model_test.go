//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBehaviorDefinedEngineModelEnvProviderSeparator(t *testing.T) {
	tests := []struct {
		name      string
		prefix    string
		separator string
		model     string
		expected  string
	}{
		{
			name:     "no prefix keeps the model unchanged",
			model:    "copilot/claude-sonnet-4-5",
			expected: "copilot/claude-sonnet-4-5",
		},
		{
			name:     "prefix defaults to a slash separator",
			prefix:   "openai",
			model:    "copilot/claude-sonnet-4-5",
			expected: "openai/claude-sonnet-4-5",
		},
		{
			name:      "explicit separator is used instead of a slash",
			prefix:    "openai-chat",
			separator: ":",
			model:     "copilot/claude-sonnet-4-5",
			expected:  "openai-chat:claude-sonnet-4-5",
		},
		{
			name:      "model without a provider is left unchanged",
			prefix:    "openai-chat",
			separator: ":",
			model:     "gpt-5",
			expected:  "gpt-5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec := &EngineExecutionDefinition{
				CommandName:               "test-cli",
				StepName:                  "Execute Test CLI",
				ModelEnvVarName:           "TEST_MODEL",
				ModelEnvProviderPrefix:    tt.prefix,
				ModelEnvProviderSeparator: tt.separator,
			}
			engine, err := NewBehaviorDefinedEngine(&EngineDefinition{
				ID:          "testmodel",
				DisplayName: "TestModel",
				Behaviors:   &EngineBehaviorDefinition{Execution: exec},
			})
			require.NoError(t, err)

			env := map[string]string{}
			engine.applyBehaviorDefinedModelEnv(exec, &WorkflowData{Model: tt.model}, env)
			assert.Equal(t, tt.expected, env["TEST_MODEL"])
		})
	}
}
