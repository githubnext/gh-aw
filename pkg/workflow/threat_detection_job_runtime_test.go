//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
)

func TestBuildDetectionJobStepsCodexAvoidsDuplicateContainerPullStep(t *testing.T) {
	compiler := NewCompiler()

	data := &WorkflowData{
		AI: "codex",
		SafeOutputs: &SafeOutputsConfig{
			ThreatDetection: &ThreatDetectionConfig{},
		},
	}

	steps := compiler.buildDetectionJobSteps(data)
	stepsString := strings.Join(steps, "")

	if count := strings.Count(stepsString, "name: Download container images"); count != 1 {
		t.Errorf("Expected exactly one 'Download container images' step for Codex detection, got %d.\n%s", count, stepsString)
	}
}

func TestBuildDetectionJobStepsCodexExternalDetectorIncludesContainerDownload(t *testing.T) {
	// Regression test: when engine=codex and gh-aw-detection feature is enabled (external
	// detector path), the detection job must include a "Download container images" step.
	// Previously the step was omitted under the incorrect assumption that MCP setup generation
	// would emit it — MCP setup is only called for the inline codex detection path.
	compiler := NewCompiler()

	t.Run("codex with gh-aw-detection includes Download container images", func(t *testing.T) {
		data := &WorkflowData{
			AI: "codex",
			SafeOutputs: &SafeOutputsConfig{
				ThreatDetection: &ThreatDetectionConfig{},
			},
			Features: map[string]any{
				string(constants.GHAWDetectionFeatureFlag): true,
			},
			SandboxConfig: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					Type: SandboxTypeAWF,
				},
			},
		}

		steps := compiler.buildDetectionJobSteps(data)
		joined := strings.Join(steps, "")

		if !strings.Contains(joined, "Download container images") {
			t.Errorf("expected 'Download container images' step in codex external detector detection job steps\ngot:\n%s", joined)
		}
		if !strings.Contains(joined, "download_docker_images.sh") {
			t.Errorf("expected 'download_docker_images.sh' in detection job steps\ngot:\n%s", joined)
		}
	})

	t.Run("codex with gh-aw-detection disabled emits exactly one container download (inline path via MCP setup)", func(t *testing.T) {
		data := &WorkflowData{
			AI: "codex",
			SafeOutputs: &SafeOutputsConfig{
				ThreatDetection: &ThreatDetectionConfig{},
			},
			Features: map[string]any{
				string(constants.GHAWDetectionFeatureFlag): false,
			},
			SandboxConfig: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					Type: SandboxTypeAWF,
				},
			},
		}

		steps := compiler.buildDetectionJobSteps(data)
		joined := strings.Join(steps, "")

		// For the inline codex path, MCP setup generation (inside buildDetectionEngineExecutionStep)
		// emits the "Download container images" step exactly once. buildPullAWFContainersStep must
		// NOT also emit it, or the step would appear twice and trip duplicate-step validation.
		downloadCount := strings.Count(joined, "Download container images")
		if downloadCount != 1 {
			t.Errorf("expected exactly one 'Download container images' step for inline codex path, got %d\n%s", downloadCount, joined)
		}
	})
}

func TestBuildPullAWFContainersStepPropagatesFeatures(t *testing.T) {
	compiler := NewCompiler()

	t.Run("cli-proxy image included when feature flag is enabled", func(t *testing.T) {
		data := &WorkflowData{
			AI: "copilot",
			SafeOutputs: &SafeOutputsConfig{
				ThreatDetection: &ThreatDetectionConfig{},
			},
			Features: map[string]any{
				string(constants.CliProxyFeatureFlag): true,
			},
			SandboxConfig: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					Type: SandboxTypeAWF,
				},
			},
		}

		steps := compiler.buildPullAWFContainersStep(data)
		stepsString := strings.Join(steps, "")

		if !strings.Contains(stepsString, "cli-proxy") {
			t.Error("Expected cli-proxy image in pull step when cli-proxy feature flag is enabled")
		}
	})

	t.Run("cli-proxy image excluded when feature flag is not set", func(t *testing.T) {
		data := &WorkflowData{
			AI: "copilot",
			SafeOutputs: &SafeOutputsConfig{
				ThreatDetection: &ThreatDetectionConfig{},
			},
			Features: map[string]any{},
			SandboxConfig: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					Type: SandboxTypeAWF,
				},
			},
		}

		steps := compiler.buildPullAWFContainersStep(data)
		stepsString := strings.Join(steps, "")

		if strings.Contains(stepsString, "cli-proxy") {
			t.Error("Expected no cli-proxy image in pull step when cli-proxy feature flag is not set")
		}
	})
}

func TestBuildPullAWFContainersStepPropagatesRunnerTopology(t *testing.T) {
	compiler := NewCompiler()
	buildToolsImagePrefix := constants.DefaultFirewallRegistry + "/build-tools:"

	t.Run("arc-dind includes build-tools image", func(t *testing.T) {
		data := &WorkflowData{
			AI: "copilot",
			RunnerConfig: &RunnerConfig{
				Topology: RunnerTopologyArcDind,
			},
			SafeOutputs: &SafeOutputsConfig{
				ThreatDetection: &ThreatDetectionConfig{},
			},
		}

		steps := compiler.buildPullAWFContainersStep(data)
		stepsString := strings.Join(steps, "")

		if !strings.Contains(stepsString, buildToolsImagePrefix) {
			t.Errorf("expected build-tools image prefix %q in detection pull step for arc-dind;\ngot:\n%s", buildToolsImagePrefix, stepsString)
		}
	})

	t.Run("non-arc-dind excludes build-tools image", func(t *testing.T) {
		data := &WorkflowData{
			AI: "copilot",
			SafeOutputs: &SafeOutputsConfig{
				ThreatDetection: &ThreatDetectionConfig{},
			},
		}

		steps := compiler.buildPullAWFContainersStep(data)
		stepsString := strings.Join(steps, "")

		if strings.Contains(stepsString, buildToolsImagePrefix) {
			t.Errorf("did not expect build-tools image prefix %q in detection pull step without arc-dind;\ngot:\n%s", buildToolsImagePrefix, stepsString)
		}
	})

	t.Run("permissions do not change pulled images", func(t *testing.T) {
		baseData := &WorkflowData{
			AI: "copilot",
			SafeOutputs: &SafeOutputsConfig{
				ThreatDetection: &ThreatDetectionConfig{},
			},
		}
		withPermissions := &WorkflowData{
			AI:                baseData.AI,
			SafeOutputs:       baseData.SafeOutputs,
			Permissions:       "contents: read",
			CachedPermissions: NewPermissionsContentsRead(),
		}

		baseSteps := strings.Join(compiler.buildPullAWFContainersStep(baseData), "")
		permissionSteps := strings.Join(compiler.buildPullAWFContainersStep(withPermissions), "")

		if permissionSteps != baseSteps {
			t.Errorf("expected detection pull step to ignore permissions when collecting images;\nwithout permissions:\n%s\nwith permissions:\n%s", baseSteps, permissionSteps)
		}
	})
}
