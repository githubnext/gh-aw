//go:build !integration

package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/workflow/compilerenv"
	"github.com/stretchr/testify/require"
)

func TestDetectionModelOverridePrecedence(t *testing.T) {
	compiler := NewCompiler()
	for _, engineID := range []string{"claude", "codex", "copilot"} {
		t.Run(engineID, func(t *testing.T) {
			suffix := strings.ToUpper(engineID)
			modelEnv := map[string]string{
				"claude":  "ANTHROPIC_MODEL",
				"codex":   "GH_AW_MODEL_DETECTION_CODEX",
				"copilot": "COPILOT_MODEL",
			}[engineID]
			for _, tc := range []struct {
				name           string
				mainModel      string
				detectionModel string
				compilerModel  string
				want           string
			}{
				{
					name: "runtime overrides before built-in alias",
					want: fmt.Sprintf("${{ vars.GH_AW_MODEL_DETECTION_%s || vars.GH_AW_DEFAULT_MODEL_%s || 'detection' }}", suffix, suffix),
				},
				{
					name:          "compile-time default remains explicit",
					compilerModel: "compile-model",
					want:          "compile-model",
				},
				{
					name:          "inherited main model before defaults",
					mainModel:     "main-model",
					compilerModel: "compile-model",
					want:          "main-model",
				},
				{
					name:           "detection model before inherited model",
					mainModel:      "main-model",
					detectionModel: "detection-model",
					compilerModel:  "compile-model",
					want:           "detection-model",
				},
				{
					name:           "explicit expression is preserved",
					detectionModel: "${{ inputs.model }}",
					want:           "${{ inputs.model }}",
				},
			} {
				t.Run(tc.name, func(t *testing.T) {
					t.Setenv(compilerenv.DefaultDetectionModel, tc.compilerModel)
					data := &WorkflowData{
						AI:           engineID,
						Model:        tc.mainModel,
						EngineConfig: &EngineConfig{ID: engineID},
						SafeOutputs: &SafeOutputsConfig{
							ThreatDetection: &ThreatDetectionConfig{Model: tc.detectionModel},
						},
					}
					for name, build := range map[string]func(*WorkflowData) []string{
						"inline":   compiler.buildDetectionEngineExecutionStep,
						"external": compiler.buildExternalDetectorExecutionStep,
					} {
						t.Run(name, func(t *testing.T) {
							steps := strings.Join(build(data), "")
							require.Contains(t, steps, modelEnv+": "+tc.want+"\n")
							require.NotContains(t, steps, "--model ${{", "model expressions must stay in step env, not shell source")
						})
					}
				})
			}
		})
	}
}

func TestCompiledDetectionModelOverrides(t *testing.T) {
	t.Setenv(compilerenv.DefaultDetectionModel, "")
	for _, engineID := range []string{"claude", "codex", "copilot"} {
		for _, external := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/external=%t", engineID, external), func(t *testing.T) {
				workflowPath := filepath.Join(t.TempDir(), "detection-model.md")
				source := fmt.Sprintf("---\non: workflow_dispatch\nengine: %s\nsafe-outputs:\n  create-issue:\nfeatures:\n  gh-aw-detection: %t\n---\nReport a finding.\n", engineID, external)
				require.NoError(t, os.WriteFile(workflowPath, []byte(source), 0644))
				require.NoError(t, NewCompiler().CompileWorkflow(workflowPath))
				lock, err := os.ReadFile(stringutil.MarkdownToLockFile(workflowPath))
				require.NoError(t, err)
				detection := extractJobSection(string(lock), "detection")
				require.NotEmpty(t, detection)
				suffix := strings.ToUpper(engineID)
				require.Contains(t, detection, fmt.Sprintf("${{ vars.GH_AW_MODEL_DETECTION_%s || vars.GH_AW_DEFAULT_MODEL_%s || 'detection' }}", suffix, suffix))
				require.Contains(t, detection, "continue-on-error: true", "the intentional default failure policy must not change")
			})
		}
	}
}
