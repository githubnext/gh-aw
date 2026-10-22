//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDailyBYOKOllamaStartup(t *testing.T) {
	var source struct {
		Steps []struct {
			Name string            `yaml:"name"`
			Env  map[string]string `yaml:"env"`
			Run  string            `yaml:"run"`
		} `yaml:"steps"`
	}
	readConformanceFrontmatter(t, "../../.github/workflows/daily-byok-ollama-test.md", &source)
	for _, step := range source.Steps {
		if step.Name != "Start Ollama service" {
			continue
		}
		// llama.cpp b9840 reads this inherited flag before loading model weights.
		assert.Equal(t, "0", step.Env["LLAMA_ARG_REPACK"])
		assert.Contains(t, step.Run, `nohup ollama serve > "$OLLAMA_LOG" 2>&1 < /dev/null &`)
		return
	}
	t.Fatal("missing Start Ollama service step")
}

func TestDailyBYOKOllamaDetectionUsesHostedProvider(t *testing.T) {
	var source struct {
		Engine      EngineConfig      `yaml:"engine"`
		Permissions map[string]string `yaml:"permissions"`
		SafeOutputs map[string]any    `yaml:"safe-outputs"`
	}
	readConformanceFrontmatter(t, "../../.github/workflows/daily-byok-ollama-test.md", &source)
	detection := NewCompiler().parseThreatDetectionConfig(source.SafeOutputs)
	require.NotNil(t, detection)
	require.NotNil(t, detection.EngineConfig)
	require.Equal(t, "write", source.Permissions["copilot-requests"])
	data := &WorkflowData{
		AI: "copilot", EngineConfig: &source.Engine,
		Permissions:        "contents: read\ncopilot-requests: write",
		SafeOutputs:        &SafeOutputsConfig{ThreatDetection: detection},
		NetworkPermissions: &NetworkPermissions{Firewall: &FirewallConfig{Enabled: true}},
	}
	assert.Equal(t, "http://host.docker.internal:11434/v1", source.Engine.Env["COPILOT_PROVIDER_BASE_URL"])
	assert.Equal(t, "qwen2.5:0.5b", source.Engine.Env["COPILOT_MODEL"])
	compiler := NewCompiler()
	require.Equal(t, "claude", compiler.getThreatDetectionEngineID(data))
	detector := buildExternalDetectorWorkflowData(data, "claude")
	assert.Empty(t, detector.EngineConfig.Env)
	assert.Equal(t, LLMProviderGitHub, NewClaudeEngine().ResolveLLMProvider(detector))
	for name, build := range map[string]func(*WorkflowData) []string{
		"external": compiler.buildExternalDetectorExecutionStep,
		"inline":   compiler.buildDetectionEngineExecutionStep,
	} {
		t.Run(name, func(t *testing.T) {
			steps := strings.Join(build(data), "\n")
			assert.NotContains(t, steps, "host.docker.internal:11434")
			assert.NotContains(t, steps, "env.OLLAMA_API_KEY")
			assert.NotContains(t, steps, "qwen2.5:0.5b")
			assert.Contains(t, steps, "COPILOT_GITHUB_TOKEN: ${{ github.token }}")
			assert.Contains(t, steps, "claude-haiku-4.5")
			assert.NotContains(t, steps, "secrets.ANTHROPIC_API_KEY")
		})
	}
}
