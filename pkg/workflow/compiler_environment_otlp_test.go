//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestEnvironmentOTLPScoping(t *testing.T) {
	for _, engine := range []string{"codex", "pi"} {
		t.Run(engine, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "test.md")
			source := "---\non: workflow_dispatch\nstrict: false\nengine: " + engine + `
environment: aoai-model
checkout: false
tools:
  github: false
safe-outputs:
  noop:
  threat-detection: false
env:
  KEEP: unchanged
---
Call noop.
`
			require.NoError(t, os.WriteFile(path, []byte(source), 0600))
			compiler := NewCompiler()
			compiler.SetSkipValidation(true)
			require.NoError(t, compiler.CompileWorkflow(path))
			content, err := os.ReadFile(filepath.Join(dir, "test.lock.yml"))
			require.NoError(t, err)
			var compiled struct {
				Env  map[string]string         `yaml:"env"`
				Jobs map[string]map[string]any `yaml:"jobs"`
			}
			require.NoError(t, yaml.Unmarshal(content, &compiled))
			assert.Equal(t, map[string]string{"KEEP": "unchanged"}, compiled.Env)
			for _, name := range []string{"agent", "conclusion", "safe_outputs"} {
				job := compiled.Jobs[name]
				require.Equal(t, "aoai-model", job["environment"])
				env := job["env"].(map[string]any)
				assert.Equal(t, "${{ secrets.GH_AW_DEFAULT_OTLP_HEADERS }}", env["OTEL_EXPORTER_OTLP_HEADERS"])
				assert.Contains(t, env["GH_AW_OTLP_ENDPOINTS"], "${{ secrets.GH_AW_DEFAULT_OTLP_HEADERS }}")
			}
			require.Contains(t, compiled.Jobs, "activation")
			activation := compiled.Jobs["activation"]
			assert.NotContains(t, activation, "environment")
			assert.NotContains(t, activation["env"], "OTEL_EXPORTER_OTLP_HEADERS")
			assert.NotContains(t, activation["env"], "GH_AW_OTLP_ENDPOINTS")
			assert.NotContains(t, activation["env"], "OTEL_EXPORTER_OTLP_ENDPOINT")
			activationYAML, err := yaml.Marshal(activation)
			require.NoError(t, err)
			assert.NotContains(t, string(activationYAML), "secrets.GH_AW_DEFAULT_OTLP_HEADERS")
		})
	}
}

func TestScopeEnvironmentOTLPEnv(t *testing.T) {
	data := &WorkflowData{
		Environment: "environment: aoai-model",
		Env: `env:
  KEEP: unchanged
  OTEL_EXPORTER_OTLP_HEADERS: ${{ secrets.OTLP_HEADERS }}
  GH_AW_OTLP_ALL_HEADERS: ${{ secrets.OTLP_HEADERS }},${{ secrets.OTHER_HEADERS }}
  GH_AW_OTLP_ENDPOINTS: '[{"url":"https://example.com","headers":"${{ secrets.OTLP_HEADERS }}"}]'
  GH_AW_OTLP_ATTRIBUTES: '{"session":"example"}'`,
	}
	c := NewCompiler()
	for _, job := range []*Job{
		{Name: "activation", Environment: data.Environment},
		{Name: "agent", Environment: data.Environment, Env: map[string]string{"OTEL_SERVICE_NAME": "custom"}},
		{Name: "detection", Environment: data.Environment},
		{Name: "conclusion"},
		{Name: "safe_outputs", Environment: "environment: writes"},
		{Name: "custom", Environment: data.Environment},
	} {
		require.NoError(t, c.jobManager.AddJob(job))
	}
	scoped := c.scopeEnvironmentOTLPEnv(data)
	assert.Equal(t, map[string]any{"KEEP": "unchanged"}, parseEnvYAMLSection(scoped))
	for _, name := range []string{"agent", "detection", "safe_outputs"} {
		env := c.jobManager.GetAllJobs()[name].Env
		assert.Equal(t, "'${{ secrets.OTLP_HEADERS }}'", env["OTEL_EXPORTER_OTLP_HEADERS"])
		assert.Contains(t, env["GH_AW_OTLP_ALL_HEADERS"], "${{ secrets.OTHER_HEADERS }}")
		assert.Equal(t, `'{"session":"example"}'`, env["GH_AW_OTLP_ATTRIBUTES"])
	}
	assert.Equal(t, "custom", c.jobManager.GetAllJobs()["agent"].Env["OTEL_SERVICE_NAME"])
	for _, name := range []string{"activation", "conclusion", "custom"} {
		assert.Empty(t, c.jobManager.GetAllJobs()[name].Env)
	}
	assert.Equal(t, scoped, c.scopeEnvironmentOTLPEnv(data), "repeated body generation must be stable")
	assert.Contains(t, data.Env, "secrets.OTLP_HEADERS", "step generation retains telemetry configuration")

	data.Environment = ""
	assert.Equal(t, data.Env, c.scopeEnvironmentOTLPEnv(data), "workflows without an environment retain existing behavior")
	c.SetEnvironmentOverride("debug")
	assert.Equal(t, scoped, c.scopeEnvironmentOTLPEnv(data))
}
