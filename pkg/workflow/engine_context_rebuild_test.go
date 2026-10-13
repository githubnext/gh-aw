//go:build !integration

package workflow

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractEngineMaxContextRebuild(t *testing.T) {
	for _, value := range []any{1, 50, uint64(50), int64(50), 29.5} {
		t.Run(fmt.Sprint(value), func(t *testing.T) {
			_, config, _ := NewCompiler().ExtractEngineConfig(map[string]any{
				"engine": map[string]any{"id": "codex", "max-context-rebuild": value},
			})
			require.NotNil(t, config)
			require.NoError(t, validateMaxContextRebuildConfig(config, NewCodexEngine()))
			env := map[string]string{}
			applyEngineMaxContextRebuildEnv(env, &WorkflowData{EngineConfig: config}, NewCodexEngine())
			assert.Equal(t, fmt.Sprint(value), env["GH_AW_CODEX_MAX_REBUILD_FACTOR"])
		})
	}
}

func TestExtractInlineEngineMaxContextRebuild(t *testing.T) {
	_, config, _ := NewCompiler().ExtractEngineConfig(map[string]any{
		"engine": map[string]any{
			"runtime":             map[string]any{"id": "codex"},
			"max-context-rebuild": uint64(50),
		},
	})
	require.NotNil(t, config)
	assert.Equal(t, "codex", config.ID)
	assert.InDelta(t, 50, config.MaxContextRebuild, 0)
}

func TestValidateMaxContextRebuildConfig(t *testing.T) {
	for _, engine := range []CodingAgentEngine{NewCopilotEngine(), NewClaudeEngine(), NewGeminiEngine(), NewPiEngine(), NewAgyEngine()} {
		t.Run(engine.GetID(), func(t *testing.T) {
			require.NoError(t, validateMaxContextRebuildConfig(nil, engine))
			require.NoError(t, validateMaxContextRebuildConfig(&EngineConfig{}, engine))
			require.ErrorContains(t, validateMaxContextRebuildConfig(&EngineConfig{MaxContextRebuild: 50}, engine), "does not support")
			assert.False(t, engine.GetCapabilities().MaxContextRebuild)
			assert.Empty(t, engine.GetMaxContextRebuildEnvVarName())
		})
	}
	for _, factor := range []float64{0.5, -1, math.NaN(), math.Inf(1)} {
		require.ErrorContains(t, validateMaxContextRebuildConfig(&EngineConfig{MaxContextRebuild: factor}, NewCodexEngine()), "finite factor")
	}
}

func TestCodexMaxContextRebuildExecutionEnv(t *testing.T) {
	engine := NewCodexEngine()
	assert.True(t, engine.GetCapabilities().MaxContextRebuild)
	assert.Equal(t, "GH_AW_CODEX_MAX_REBUILD_FACTOR", engine.GetMaxContextRebuildEnvVarName())
	for _, test := range []struct {
		name   string
		config *EngineConfig
		want   string
	}{
		{name: "omitted"},
		{name: "configured", config: &EngineConfig{MaxContextRebuild: 29.5}, want: "29.5"},
		{name: "legacy env", config: &EngineConfig{Env: map[string]string{"GH_AW_CODEX_MAX_REBUILD_FACTOR": "50"}}, want: "50"},
		{name: "explicit env override", config: &EngineConfig{MaxContextRebuild: 25, Env: map[string]string{"GH_AW_CODEX_MAX_REBUILD_FACTOR": "50"}}, want: "50"},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, firewall := range []bool{false, true} {
				env := engine.buildCodexExecutionEnv(&WorkflowData{EngineConfig: test.config}, firewall, false, "GH_AW_MODEL_AGENT_CODEX")
				assert.Equal(t, test.want, env["GH_AW_CODEX_MAX_REBUILD_FACTOR"])
				if test.want == "" {
					assert.NotContains(t, env, "GH_AW_CODEX_MAX_REBUILD_FACTOR")
				}
			}
		})
	}
}

func TestBehaviorDefinedEngineMaxContextRebuild(t *testing.T) {
	def, err := parseEngineDefinitionFromJSON(`{
		"id": "context-rebuild-test",
		"display-name": "Context Rebuild Test",
		"behaviors": {
			"capabilities": {"max-context-rebuild": true},
			"execution": {"command-name": "rebuild-cli", "max-context-rebuild-env-var": "REBUILD_FACTOR_LIMIT"}
		}
	}`)
	require.NoError(t, err)
	require.NotNil(t, def)
	engine, err := NewBehaviorDefinedEngine(def)
	require.NoError(t, err)
	assert.True(t, engine.GetCapabilities().MaxContextRebuild)
	assert.Equal(t, "REBUILD_FACTOR_LIMIT", engine.GetMaxContextRebuildEnvVarName())

	data := &WorkflowData{Name: "test", EngineConfig: &EngineConfig{MaxContextRebuild: 50}}
	require.NoError(t, validateMaxContextRebuildConfig(data.EngineConfig, engine))
	step := strings.Join(engine.GetExecutionSteps(data, "/tmp/test.log")[0], "\n")
	assert.Contains(t, step, "REBUILD_FACTOR_LIMIT: 50")
	assert.NotContains(t, step, "GH_AW_CODEX_MAX_REBUILD_FACTOR")

	data.EngineConfig.Env = map[string]string{"REBUILD_FACTOR_LIMIT": "29.5"}
	step = strings.Join(engine.GetExecutionSteps(data, "/tmp/test.log")[0], "\n")
	assert.Contains(t, step, "REBUILD_FACTOR_LIMIT: 29.5")

	data.EngineConfig.MaxContextRebuild = 0
	data.EngineConfig.Env = nil
	step = strings.Join(engine.GetExecutionSteps(data, "/tmp/test.log")[0], "\n")
	assert.NotContains(t, step, "REBUILD_FACTOR_LIMIT")
}

func TestBehaviorDefinedEngineMaxContextRebuildDefinitionValidation(t *testing.T) {
	for _, test := range []struct {
		name      string
		supported bool
		execution *EngineExecutionDefinition
		wantError string
	}{
		{name: "unsupported"},
		{name: "missing execution", supported: true, wantError: "without a valid environment variable"},
		{name: "missing mapping", supported: true, execution: &EngineExecutionDefinition{}, wantError: "without a valid environment variable"},
		{name: "invalid mapping", supported: true, execution: &EngineExecutionDefinition{MaxContextRebuildEnvVar: "INVALID-NAME"}, wantError: "without a valid environment variable"},
		{name: "mapping without capability", execution: &EngineExecutionDefinition{MaxContextRebuildEnvVar: "REBUILD_LIMIT"}, wantError: "without enabling the capability"},
	} {
		t.Run(test.name, func(t *testing.T) {
			engine, err := NewBehaviorDefinedEngine(&EngineDefinition{
				ID: "test",
				Behaviors: &EngineBehaviorDefinition{
					Capabilities: EngineCapabilitiesDefinition{MaxContextRebuild: test.supported},
					Execution:    test.execution,
				},
			})
			if test.wantError != "" {
				require.ErrorContains(t, err, test.wantError)
				return
			}
			require.NoError(t, err)
			require.ErrorContains(t, validateMaxContextRebuildConfig(&EngineConfig{MaxContextRebuild: 50}, engine), "does not support")
		})
	}
}

func TestCompileWorkflowMaxContextRebuild(t *testing.T) {
	for _, test := range []struct {
		name      string
		engine    string
		shared    string
		override  string
		wantEnv   string
		wantError string
	}{
		{name: "codex", engine: "  id: codex\n  max-context-rebuild: 50\n", wantEnv: "GH_AW_CODEX_MAX_REBUILD_FACTOR: 50"},
		{name: "unsupported copilot", engine: "  id: copilot\n  max-context-rebuild: 50\n", wantError: "does not support the max-context-rebuild feature"},
		{name: "unsupported engine override", engine: "  id: codex\n  max-context-rebuild: 50\n", override: "copilot", wantError: "does not support the max-context-rebuild feature"},
		{name: "imported codex configuration", shared: "  id: codex\n  max-context-rebuild: 29.5\n", wantEnv: "GH_AW_CODEX_MAX_REBUILD_FACTOR: 29.5"},
		{name: "unsupported imported configuration", shared: "  id: claude\n  max-context-rebuild: 50\n", wantError: "does not support the max-context-rebuild feature"},
		{
			name:   "declarative engine",
			engine: "  id: rebuild-test\n  max-context-rebuild: 50\n",
			shared: `  id: rebuild-test
  display-name: Rebuild Test
  behaviors:
    capabilities:
      max-context-rebuild: true
    execution:
      command-name: rebuild-cli
      max-context-rebuild-env-var: REBUILD_FACTOR_LIMIT
`,
			wantEnv: "REBUILD_FACTOR_LIMIT: 50",
		},
		{
			name:      "unsupported declarative engine",
			engine:    "  id: rebuild-test\n  max-context-rebuild: 50\n",
			shared:    "  id: rebuild-test\n  display-name: Rebuild Test\n  behaviors:\n    execution:\n      command-name: rebuild-cli\n",
			wantError: "does not support the max-context-rebuild feature",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := testutil.TempDir(t, "max-context-rebuild")
			content := "---\non: workflow_dispatch\n"
			if test.engine != "" {
				content += "engine:\n" + test.engine
			}
			if test.shared != "" {
				require.NoError(t, os.WriteFile(filepath.Join(dir, "shared.md"), []byte("---\nengine:\n"+test.shared+"---\n"), 0o644))
				content += "imports:\n  - shared.md\n"
			}
			content += "---\n\nRun the workflow.\n"
			file := filepath.Join(dir, "workflow.md")
			require.NoError(t, os.WriteFile(file, []byte(content), 0o644))
			err := NewCompiler(WithVersion("dev"), WithEngineOverride(test.override)).CompileWorkflow(file)
			if test.wantError != "" {
				require.ErrorContains(t, err, test.wantError)
				return
			}
			require.NoError(t, err)
			lock, err := os.ReadFile(filepath.Join(dir, "workflow.lock.yml"))
			require.NoError(t, err)
			assert.Contains(t, string(lock), test.wantEnv)
		})
	}
}

func TestCompileWorkflowMaxContextRebuildImportedPreferencePreservesConfig(t *testing.T) {
	dir := testutil.TempDir(t, "max-context-rebuild-import-override")
	shared := `---
engine:
  id: codex
  max-context-rebuild: 29.5
  auth:
    type: github-oidc
    provider: gcp
    workload-identity-provider: projects/123/locations/global/workloadIdentityPools/test/providers/test
    service-account: test@example.iam.gserviceaccount.com
    project: test-project
  env:
    IMPORTED_ENGINE_SETTING: preserved
---
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "shared.md"), []byte(shared), 0o644))
	workflow := `---
on: workflow_dispatch
permissions:
  id-token: write
imports:
  - shared.md
engine:
  max-context-rebuild: 50
---

Run the workflow.
`
	file := filepath.Join(dir, "workflow.md")
	require.NoError(t, os.WriteFile(file, []byte(workflow), 0o644))
	require.NoError(t, NewCompiler(WithVersion("dev")).CompileWorkflow(file))

	lock, err := os.ReadFile(filepath.Join(dir, "workflow.lock.yml"))
	require.NoError(t, err)
	assert.Contains(t, string(lock), "GH_AW_CODEX_MAX_REBUILD_FACTOR: 50")
	assert.Contains(t, string(lock), "AWF_AUTH_TYPE: github-oidc")
	assert.Contains(t, string(lock), "AWF_AUTH_PROVIDER: gcp")
	assert.Contains(t, string(lock), "IMPORTED_ENGINE_SETTING: preserved")
}

func TestCompileWorkflowThreatDetectionMaxContextRebuild(t *testing.T) {
	for _, external := range []bool{false, true} {
		for _, engine := range []string{"codex", "copilot", "claude", "pi"} {
			t.Run(fmt.Sprintf("%s/external=%t", engine, external), func(t *testing.T) {
				dir := testutil.TempDir(t, "detection-max-context-rebuild")
				file := filepath.Join(dir, "workflow.md")
				content := fmt.Sprintf(`---
on: workflow_dispatch
engine: copilot
safe-outputs:
  create-issue:
  threat-detection:
    engine:
      id: %s
      max-context-rebuild: 50
features:
  gh-aw-detection: %t
---
Report a finding.
`, engine, external)
				require.NoError(t, os.WriteFile(file, []byte(content), 0o644))
				err := NewCompiler(WithVersion("dev")).CompileWorkflow(file)
				if engine != "codex" {
					require.ErrorContains(t, err, "safe-outputs.threat-detection")
					require.ErrorContains(t, err, "does not support the max-context-rebuild feature")
					return
				}
				require.NoError(t, err)
				lock, err := os.ReadFile(filepath.Join(dir, "workflow.lock.yml"))
				require.NoError(t, err)
				assert.Contains(t, extractJobSection(string(lock), "detection"), "GH_AW_CODEX_MAX_REBUILD_FACTOR: 50")
				assert.NotContains(t, extractJobSection(string(lock), "agent"), "GH_AW_CODEX_MAX_REBUILD_FACTOR")
			})
		}
	}
}

func TestThreatDetectionMaxContextRebuildInheritance(t *testing.T) {
	source := &EngineConfig{ID: "codex", MaxContextRebuild: 50}
	data := &WorkflowData{AI: "codex", EngineConfig: source}
	assert.InDelta(t, 50.0, resolveExternalDetectorEngineConfig(data, "codex").MaxContextRebuild, 0)
	assert.Zero(t, resolveExternalDetectorEngineConfig(data, "copilot").MaxContextRebuild)
	assert.InDelta(t, 50.0, cloneThreatDetectionEngineConfig("codex", source).MaxContextRebuild, 0)
	assert.Zero(t, cloneThreatDetectionEngineConfig("copilot", source).MaxContextRebuild)
	assert.InDelta(t, 50.0, source.MaxContextRebuild, 0)
}

func TestThreatDetectionMaxContextRebuildNormalizesLegacyEngineAndEnv(t *testing.T) {
	data := &WorkflowData{
		AI: "codex",
		EngineConfig: &EngineConfig{
			MaxContextRebuild: 50,
		},
		SafeOutputs: &SafeOutputsConfig{
			ThreatDetection: &ThreatDetectionConfig{
				EngineConfig: &EngineConfig{
					MaxContextRebuild: 50,
					Env: map[string]string{
						"GH_AW_CODEX_MAX_REBUILD_FACTOR": "50",
						"KEEP_ME":                        "preserved",
					},
				},
			},
		},
	}

	detectionConfig := resolveExternalDetectorEngineConfig(data, "copilot")
	assert.Zero(t, detectionConfig.MaxContextRebuild)
	assert.NotContains(t, detectionConfig.Env, "GH_AW_CODEX_MAX_REBUILD_FACTOR")
	assert.Equal(t, "preserved", detectionConfig.Env["KEEP_ME"])
	assert.InDelta(t, 50, data.SafeOutputs.ThreatDetection.EngineConfig.MaxContextRebuild, 0)
	assert.Equal(t, "50", data.SafeOutputs.ThreatDetection.EngineConfig.Env["GH_AW_CODEX_MAX_REBUILD_FACTOR"])
}
