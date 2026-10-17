//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectionRedactionOrderingAndGates(t *testing.T) {
	for _, external := range []bool{false, true} {
		for _, topology := range []RunnerTopology{"", RunnerTopologyArcDind} {
			t.Run(map[bool]string{false: "inline", true: "external"}[external]+"/"+string(topology), func(t *testing.T) {
				data := &WorkflowData{
					AI:              "copilot",
					Features:        map[string]any{string(constants.GHAWDetectionFeatureFlag): external},
					FrontmatterYAML: "env:\n  PRIVATE: ${{ secrets.ARBITRARY_SECRET }}",
					MarkdownContent: "${{ secrets.PROMPT_SECRET }}",
					SafeOutputs: &SafeOutputsConfig{ThreatDetection: &ThreatDetectionConfig{
						PostSteps: []any{map[string]any{"name": "Late detection output", "run": "echo output"}},
					}},
					SandboxConfig: &SandboxConfig{Agent: &AgentSandboxConfig{Type: SandboxTypeAWF}},
				}
				if topology != "" {
					data.RunnerConfig = &RunnerConfig{Topology: topology}
				}
				content := strings.Join(NewCompiler().buildDetectionJobSteps(data), "")
				var parsed struct {
					Steps []struct {
						Name            string            `yaml:"name"`
						ID              string            `yaml:"id"`
						If              string            `yaml:"if"`
						ContinueOnError bool              `yaml:"continue-on-error"`
						Env             map[string]string `yaml:"env"`
					} `yaml:"steps"`
				}
				require.NoError(t, yaml.Unmarshal([]byte("steps:\n"+content), &parsed))
				positions := map[string]int{}
				outcomes := map[string]string{
					"steps.redact_detection_logs.outcome":         "success",
					"steps.redact_detection_artifacts.outcome":    "success",
					"steps.detection_guard.outputs.run_detection": "true",
				}
				consumers := 0
				for i, step := range parsed.Steps {
					positions[step.Name] = i
					if step.ID == "redact_detection_logs" || step.ID == "redact_detection_artifacts" {
						assert.Equal(t, "always()", step.If)
						assert.False(t, step.ContinueOnError, "redaction failures must fail the job even in warn mode")
						assert.Contains(t, step.Env["GH_AW_SECRET_NAMES"], "ARBITRARY_SECRET")
						assert.Contains(t, step.Env["GH_AW_SECRET_NAMES"], "PROMPT_SECRET")
						assert.Equal(t, "${{ secrets.ARBITRARY_SECRET }}", step.Env["SECRET_ARBITRARY_SECRET"])
						continue
					}
					var condition string
					switch step.Name {
					case "Parse threat detection token usage for step summary":
						require.Equal(t, detectionLogsRedactedCondition, step.If)
						condition = step.If
					case "Echo detection step summary", "Render detection log":
						require.Equal(t, detectionStepCondition+" && steps.redact_detection_logs.outcome == 'success' && steps.redact_detection_artifacts.outcome == 'success'", step.If)
						condition = step.If
					case "Upload threat detection log", "Upload threat detection artifact", "Parse and conclude threat detection", "Conclude threat detection":
						require.Equal(t, detectionArtifactsRedactedCondition, step.If)
						condition = step.If
					default:
						continue
					}
					consumers++
					node, err := ParseExpression(condition)
					require.NoError(t, err)
					assert.True(t, evaluateRedactionUploadGate(t, node, outcomes))
					for property := range outcomes {
						if !strings.HasSuffix(property, ".outcome") || !strings.Contains(condition, property) {
							continue
						}
						for _, outcome := range []string{"failure", "skipped", "cancelled", ""} {
							outcomes[property] = outcome
							assert.False(t, evaluateRedactionUploadGate(t, node, outcomes), "%s must be gated on %s=%s", step.Name, property, outcome)
						}
						outcomes[property] = "success"
					}
				}
				assert.GreaterOrEqual(t, consumers, 4)
				order := []string{"Copy detection firewall logs", "Late detection output", "Redact secrets in detection logs", "Parse threat detection token usage for step summary", "Redact secrets in detection artifacts", "Render detection log"}
				if external && topology == "arc-dind" {
					order = append([]string{"Collect ARC/DinD threat detection results"}, order...)
				}
				for i, name := range order {
					require.Contains(t, positions, name)
					if i > 0 {
						assert.Less(t, positions[order[i-1]], positions[name])
					}
				}
				assert.Contains(t, content, "/redact_secrets.cjs")
				assert.Contains(t, content, `additionalDirectories: ["/tmp/gh-aw/threat-detection/sandbox/firewall"]`)
			})
		}
	}
}
