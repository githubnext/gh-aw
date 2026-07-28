//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildAWFImageTagWithDigests(t *testing.T) {
	t.Run("includes digest metadata for known firewall images", func(t *testing.T) {
		imageTag := strings.TrimPrefix(string(constants.DefaultFirewallVersion), "v")
		tag := buildAWFImageTagWithDigests(imageTag, nil)

		assert.Contains(t, tag, imageTag, "should keep original AWF tag")
		assert.Contains(t, tag, "squid=sha256:", "should include squid digest metadata")
		assert.Contains(t, tag, "agent=sha256:", "should include agent digest metadata")
		assert.Contains(t, tag, "api-proxy=sha256:", "should include api-proxy digest metadata")
		assert.Contains(t, tag, "cli-proxy=sha256:", "should include cli-proxy digest metadata")
	})

	t.Run("leaves tag unchanged when digests are unavailable", func(t *testing.T) {
		tag := buildAWFImageTagWithDigests("0.0.1", nil)
		assert.Equal(t, "0.0.1", tag, "should not append digest metadata when no pins are available")
	})

	t.Run("includes build-tools digest for arc-dind topology", func(t *testing.T) {
		imageTag := strings.TrimPrefix(string(constants.DefaultFirewallVersion), "v")
		buildToolsImage := constants.DefaultFirewallRegistry + "/build-tools:" + imageTag
		cache := &ActionCache{ContainerPins: make(map[string]ContainerPin)}
		cache.SetContainerPin(
			buildToolsImage,
			"sha256:1111111111111111111111111111111111111111111111111111111111111111",
			buildToolsImage+"@sha256:1111111111111111111111111111111111111111111111111111111111111111",
		)
		workflowData := &WorkflowData{
			RunnerConfig: &RunnerConfig{Topology: RunnerTopologyArcDind},
			ActionCache:  cache,
		}
		tag := buildAWFImageTagWithDigests(imageTag, workflowData)

		assert.Contains(t, tag, "build-tools=sha256:", "should include build-tools digest metadata for arc-dind topology")
	})

	t.Run("excludes build-tools digest without arc-dind topology", func(t *testing.T) {
		imageTag := strings.TrimPrefix(string(constants.DefaultFirewallVersion), "v")
		tag := buildAWFImageTagWithDigests(imageTag, nil)

		assert.NotContains(t, tag, "build-tools=", "should not include build-tools digest metadata without arc-dind topology")
	})
}

func TestBuildAWFArgs_ImageTagIncludesDigests(t *testing.T) {
	// Use the default firewall version so this test tracks pin/version updates.
	config := AWFCommandConfig{
		EngineName:     "copilot",
		AllowedDomains: "github.com",
		WorkflowData: &WorkflowData{
			EngineConfig: &EngineConfig{ID: "copilot"},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{Enabled: true, Version: string(constants.DefaultFirewallVersion)},
			},
		},
	}

	// When the AWF version supports --config (default), --image-tag moves to the JSON config file.
	// Verify the config file JSON contains the image tag with digest metadata.
	awfConfigJSON, err := BuildAWFConfigJSON(config)
	require.NoError(t, err, "BuildAWFConfigJSON should not error")
	assert.Contains(t, awfConfigJSON, "imageTag", "expected imageTag in AWF config JSON")
	assert.Contains(t, awfConfigJSON, "squid=sha256:", "expected squid digest metadata in AWF config JSON")
	assert.Contains(t, awfConfigJSON, "agent=sha256:", "expected agent digest metadata in AWF config JSON")
	assert.Contains(t, awfConfigJSON, "api-proxy=sha256:", "expected api-proxy digest metadata in AWF config JSON")

	// --image-tag should NOT appear in the CLI args (it's in the config file).
	args := BuildAWFArgs(config)
	argsStr := strings.Join(args, " ")
	assert.NotContains(t, argsStr, "--image-tag", "expected --image-tag to be absent from CLI args when config file is used")
}
