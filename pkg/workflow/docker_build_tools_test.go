//go:build !integration

package workflow

import (
	"slices"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
)

func TestCollectDockerImages_BuildToolsForArcDind(t *testing.T) {
	// Use a version without "v" prefix — getAWFImageTag strips it
	awfImageTag := "0.27.13"

	t.Run("includes build-tools image when topology is arc-dind and firewall enabled", func(t *testing.T) {
		workflowData := &WorkflowData{
			AI: "claude",
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
					Version: awfImageTag,
				},
			},
			RunnerConfig: &RunnerConfig{Topology: RunnerTopologyArcDind},
		}

		images := collectDockerImages(nil, workflowData, ActionModeRelease)

		buildToolsImage := constants.DefaultFirewallRegistry + "/build-tools:" + awfImageTag
		if !slices.Contains(images, buildToolsImage) {
			t.Errorf("Expected build-tools image %q in collected images for arc-dind, got: %v", buildToolsImage, images)
		}
	})

	t.Run("excludes build-tools image when topology is not arc-dind", func(t *testing.T) {
		workflowData := &WorkflowData{
			AI: "claude",
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
					Version: awfImageTag,
				},
			},
		}

		images := collectDockerImages(nil, workflowData, ActionModeRelease)

		buildToolsImage := constants.DefaultFirewallRegistry + "/build-tools:" + awfImageTag
		if slices.Contains(images, buildToolsImage) {
			t.Errorf("Did not expect build-tools image %q in collected images without arc-dind topology, got: %v", buildToolsImage, images)
		}
	})

	t.Run("excludes build-tools image when firewall is disabled even with arc-dind topology", func(t *testing.T) {
		workflowData := &WorkflowData{
			AI:           "claude",
			RunnerConfig: &RunnerConfig{Topology: RunnerTopologyArcDind},
		}

		images := collectDockerImages(nil, workflowData, ActionModeRelease)

		for _, img := range images {
			if strings.Contains(img, "/build-tools:") {
				t.Errorf("Did not expect any build-tools image when firewall is disabled, got: %v", images)
			}
		}
	})

	t.Run("build-tools image uses the correct AWF image tag", func(t *testing.T) {
		customTag := "0.27.5"
		workflowData := &WorkflowData{
			AI: "copilot",
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
					Version: customTag,
				},
			},
			RunnerConfig: &RunnerConfig{Topology: RunnerTopologyArcDind},
		}

		images := collectDockerImages(nil, workflowData, ActionModeRelease)

		expectedImage := constants.DefaultFirewallRegistry + "/build-tools:" + customTag
		if !slices.Contains(images, expectedImage) {
			t.Errorf("Expected build-tools image %q with custom tag, got: %v", expectedImage, images)
		}
	})
}
