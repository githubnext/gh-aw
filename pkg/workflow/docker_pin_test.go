//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestApplyContainerPins verifies that applyContainerPins substitutes
// cached digest references while leaving unpinned images unchanged.
func TestApplyContainerPins(t *testing.T) {
	imageTag := strings.TrimPrefix(string(constants.DefaultFirewallVersion), "v")
	defaultFirewallAgentImage := constants.DefaultFirewallRegistry + "/agent:" + imageTag
	defaultFirewallAgentPin, ok := getEmbeddedContainerPin(defaultFirewallAgentImage)
	require.True(t, ok, "embedded pin must exist for %s", defaultFirewallAgentImage)

	tests := []struct {
		name            string
		images          []string
		pins            map[string]ContainerPin
		expectedRefs    []string
		expectedDigests []string // expected Digest field in corresponding pin entry
	}{
		{
			name:            "no pins - images returned unchanged",
			images:          []string{"example.com/custom:1.0.0", "alpine:3.20"},
			pins:            nil,
			expectedRefs:    []string{"example.com/custom:1.0.0", "alpine:3.20"},
			expectedDigests: []string{"", ""},
		},
		{
			name:            "embedded pin used when cache is absent",
			images:          []string{"node:lts-alpine"},
			pins:            nil,
			expectedRefs:    []string{"node:lts-alpine@sha256:2bdb65ed1dab192432bc31c95f94155ca5ad7fc1392fb7eb7526ab682fa5bf14"},
			expectedDigests: []string{"sha256:2bdb65ed1dab192432bc31c95f94155ca5ad7fc1392fb7eb7526ab682fa5bf14"},
		},
		{
			name:            "embedded firewall pin used when cache is absent",
			images:          []string{defaultFirewallAgentImage},
			pins:            nil,
			expectedRefs:    []string{defaultFirewallAgentPin.PinnedImage},
			expectedDigests: []string{defaultFirewallAgentPin.Digest},
		},
		{
			name:            "embedded gh-aw-node pin used when cache is absent",
			images:          []string{constants.DefaultGhAwNodeImage},
			pins:            nil,
			expectedRefs:    []string{"ghcr.io/github/gh-aw-node@sha256:529d02eb970b1161aa25c593a9c3df57fdfad5a8add328cb3b6eccef66f3183b"},
			expectedDigests: []string{"sha256:529d02eb970b1161aa25c593a9c3df57fdfad5a8add328cb3b6eccef66f3183b"},
		},
		{
			name:   "pinned image replaced with digest reference",
			images: []string{"node:lts-alpine"},
			pins: map[string]ContainerPin{
				"node:lts-alpine": {
					Image:       "node:lts-alpine",
					Digest:      "sha256:abc123",
					PinnedImage: "node:lts-alpine@sha256:abc123",
				},
			},
			expectedRefs:    []string{"node:lts-alpine@sha256:abc123"},
			expectedDigests: []string{"sha256:abc123"},
		},
		{
			name:   "only matching image is pinned",
			images: []string{"node:lts-alpine", "busybox:latest"},
			pins: map[string]ContainerPin{
				"node:lts-alpine": {
					Image:       "node:lts-alpine",
					Digest:      "sha256:abc123",
					PinnedImage: "node:lts-alpine@sha256:abc123",
				},
			},
			expectedRefs:    []string{"node:lts-alpine@sha256:abc123", "busybox:latest"},
			expectedDigests: []string{"sha256:abc123", ""},
		},
		{
			name:            "empty images list",
			images:          nil,
			pins:            nil,
			expectedRefs:    []string{},
			expectedDigests: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var workflowData *WorkflowData
			if tt.pins != nil {
				cache := NewActionCache(t.TempDir())
				for k, v := range tt.pins {
					cache.SetContainerPin(k, v.Digest, v.PinnedImage)
				}
				workflowData = &WorkflowData{ActionCache: cache}
			}

			refs, pinEntries := applyContainerPins(tt.images, workflowData)
			require.Len(t, refs, len(tt.expectedRefs), "refs length")
			require.Len(t, pinEntries, len(tt.expectedDigests), "pin entries length")
			for i, img := range refs {
				assert.Equal(t, tt.expectedRefs[i], img, "ref at index %d", i)
				assert.Equal(t, tt.expectedDigests[i], pinEntries[i].Digest, "digest at index %d", i)
			}
		})
	}
}

// TestApplyContainerPins_DefaultFirewallVersion is a regression test for gh-aw#43307:
// all four gh-aw-firewall images at constants.DefaultFirewallVersion (including cli-proxy,
// which was new in v0.82) must have entries in the embedded pin table so that consumer
// compiles without a local cache still emit digest-pinned references.
// Using constants means the test automatically tracks version bumps.
func TestApplyContainerPins_DefaultFirewallVersion(t *testing.T) {
	imageTag := strings.TrimPrefix(string(constants.DefaultFirewallVersion), "v")
	sidecars := []string{"agent", "api-proxy", "cli-proxy", "squid"}

	for _, sidecar := range sidecars {
		image := constants.DefaultFirewallRegistry + "/" + sidecar + ":" + imageTag
		t.Run(sidecar, func(t *testing.T) {
			pin, ok := getEmbeddedContainerPin(image)
			require.True(t, ok, "embedded pin must exist for %s", image)
			require.NotEmpty(t, pin.Digest, "Digest must be non-empty for %s", image)
			require.NotEmpty(t, pin.PinnedImage, "PinnedImage must be non-empty for %s", image)

			refs, pinEntries := applyContainerPins([]string{image}, nil)
			require.Len(t, refs, 1)
			assert.Equal(t, pin.PinnedImage, refs[0], "resolved ref for %s", image)
			assert.Equal(t, pin.Digest, pinEntries[0].Digest, "digest in manifest entry for %s", image)
		})
	}
}

// TestCollectDockerImages_StoresInWorkflowData verifies that collectDockerImages
// populates workflowData.DockerImages and DockerImagePins with the collected image refs.
func TestCollectDockerImages_StoresInWorkflowData(t *testing.T) {
	workflowData := &WorkflowData{
		SandboxConfig: &SandboxConfig{
			MCP: &MCPGatewayRuntimeConfig{
				Container: constants.DefaultMCPGatewayContainer,
			},
		},
	}

	tools := map[string]any{}

	images := collectDockerImages(tools, workflowData, ActionModeRelease)

	// DockerImages on workflowData should now be populated (MCP gateway from sandbox config).
	require.NotEmpty(t, workflowData.DockerImages, "DockerImages should be populated after collectDockerImages")
	assert.Equal(t, images, workflowData.DockerImages, "DockerImages should match the returned slice")

	// DockerImagePins should also be populated with matching Image fields.
	require.NotEmpty(t, workflowData.DockerImagePins, "DockerImagePins should be populated")
	assert.Len(t, workflowData.DockerImagePins, len(workflowData.DockerImages), "pin count should match image count")
}

// TestCollectDockerImages_SafeOutputsAddsGhAwNodeImage verifies that enabling
// safe-outputs adds the published gh-aw-node container to the default Docker pull
// list and manifest data, while not falling back to node:lts-alpine.
func TestCollectDockerImages_SafeOutputsAddsGhAwNodeImage(t *testing.T) {
	workflowData := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			CreateIssues: &CreateIssuesConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{},
			},
		},
	}

	images := collectDockerImages(map[string]any{}, workflowData, ActionModeRelease)

	pinnedGhAwNodeImage := resolveContainerImage(constants.DefaultGhAwNodeImage, nil)
	assert.Contains(t, images, pinnedGhAwNodeImage,
		"safe-outputs should add the gh-aw-node container image to the Docker pull list")
	require.NotEmpty(t, workflowData.DockerImagePins, "DockerImagePins should be populated")
	assert.Contains(t, workflowData.DockerImagePins, GHAWManifestContainer{
		Image:       constants.DefaultGhAwNodeImage,
		Digest:      "sha256:529d02eb970b1161aa25c593a9c3df57fdfad5a8add328cb3b6eccef66f3183b",
		PinnedImage: pinnedGhAwNodeImage,
	}, "safe-outputs should add gh-aw-node to manifest container pins")

	for _, img := range images {
		assert.NotContains(t, img, constants.DefaultNodeAlpineLTSImage,
			"safe-outputs should not add node:lts-alpine (or any digest-pinned form) to the Docker pull list")
	}
}

// TestMergeDockerImages verifies deduplication when merging two slices.
func TestMergeDockerImages(t *testing.T) {
	existing := []string{"image-a", "image-b"}
	newImages := []string{"image-b", "image-c"}

	result := mergeDockerImages(existing, newImages)

	assert.Equal(t, []string{"image-a", "image-b", "image-c"}, result, "deduplicated merge")
}

// TestMergeDockerImagePins verifies deduplication when merging two GHAWManifestContainer slices.
func TestMergeDockerImagePins(t *testing.T) {
	existing := []GHAWManifestContainer{
		{Image: "image-a", Digest: "sha256:aaa"},
		{Image: "image-b"},
	}
	newPins := []GHAWManifestContainer{
		{Image: "image-b", Digest: "sha256:bbb"}, // duplicate — should not replace existing
		{Image: "image-c", Digest: "sha256:ccc"},
	}

	result := mergeDockerImagePins(existing, newPins)

	require.Len(t, result, 3, "deduplicated merge length")
	assert.Equal(t, "image-a", result[0].Image)
	assert.Equal(t, "image-b", result[1].Image)
	assert.Equal(t, "image-c", result[2].Image)
	assert.Equal(t, "sha256:ccc", result[2].Digest)
}

// TestApplyContainerPins_ContainerPinMappings verifies that applyContainerPins
// applies container_pins redirects before digest lookup so that both the
// pre-download list and the manifest entries reference the mapped image.
func TestApplyContainerPins_ContainerPinMappings(t *testing.T) {
	const digest = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	t.Run("digest-pinned mapped image returned as mapped", func(t *testing.T) {
		workflowData := &WorkflowData{
			ContainerPinMappings: map[string]string{
				"ghcr.io/owner/image:v1": "registry.acme.com/image:v1@sha256:" + digest,
			},
		}
		refs, pinEntries := applyContainerPins([]string{"ghcr.io/owner/image:v1"}, workflowData)
		require.Len(t, refs, 1)
		assert.Equal(t, "registry.acme.com/image:v1@sha256:"+digest, refs[0],
			"source image should be redirected to the mapped registry")
		assert.Equal(t, "registry.acme.com/image:v1@sha256:"+digest, pinEntries[0].Image,
			"manifest entry Image should reflect the mapped image")
		assert.Empty(t, pinEntries[0].Digest, "mapped reference already includes its digest")
	})

	t.Run("unmapped image passes through unchanged", func(t *testing.T) {
		workflowData := &WorkflowData{
			ContainerPinMappings: map[string]string{
				"other.registry.io/image:v1": "registry.acme.com/image:v1@sha256:" + digest,
			},
		}
		refs, _ := applyContainerPins([]string{"ghcr.io/owner/image:v1"}, workflowData)
		require.Len(t, refs, 1)
		assert.Equal(t, "ghcr.io/owner/image:v1", refs[0],
			"image not in mappings should be returned unchanged")
	})

	t.Run("nil ContainerPinMappings leaves images unchanged", func(t *testing.T) {
		workflowData := &WorkflowData{}
		refs, _ := applyContainerPins([]string{"ghcr.io/owner/image:v1"}, workflowData)
		require.Len(t, refs, 1)
		assert.Equal(t, "ghcr.io/owner/image:v1", refs[0])
	})
}

// TestCollectDockerImages_DefaultAlpineContainerPinMapping verifies that when
// container_pins maps DefaultAlpineImage, collectDockerImages applies the redirect
// so the pre-download list and manifest use the configured private mirror.
func TestCollectDockerImages_DefaultAlpineContainerPinMapping(t *testing.T) {
	const digest = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	mapped := "registry.acme.com/alpine@sha256:" + digest
	workflowData := &WorkflowData{
		ContainerPinMappings: map[string]string{
			constants.DefaultAlpineImage: mapped,
		},
	}

	tools := map[string]any{
		"agentic-workflows": map[string]any{},
	}

	images := collectDockerImages(tools, workflowData, ActionModeRelease)

	assert.Contains(t, images, mapped,
		"DefaultAlpineImage should be replaced by the container_pins mapped image in the pre-download list")
	for _, img := range images {
		assert.NotEqual(t, constants.DefaultAlpineImage, img,
			"original DefaultAlpineImage should not appear in the pre-download list when mapped")
	}
}

// TestResolveGatewayContainerFromMappings verifies that resolveGatewayContainerFromMappings
// applies the container_pins redirect and strips the digest for MCP Gateway compatibility.
func TestResolveGatewayContainerFromMappings(t *testing.T) {
	const digest = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	t.Run("mapped image has digest stripped for gateway", func(t *testing.T) {
		mappings := map[string]string{
			"ghcr.io/owner/image:v1": "registry.acme.com/image:v1@sha256:" + digest,
		}
		got := resolveGatewayContainerFromMappings("ghcr.io/owner/image:v1", mappings)
		assert.Equal(t, "registry.acme.com/image:v1", got,
			"digest should be stripped from mapped image for MCP Gateway compatibility")
	})

	t.Run("unmapped image passes through unchanged", func(t *testing.T) {
		mappings := map[string]string{
			"other.io/image:v1": "registry.acme.com/image:v1@sha256:" + digest,
		}
		got := resolveGatewayContainerFromMappings("ghcr.io/owner/image:v1", mappings)
		assert.Equal(t, "ghcr.io/owner/image:v1", got)
	})

	t.Run("nil mappings returns image unchanged", func(t *testing.T) {
		got := resolveGatewayContainerFromMappings("ghcr.io/owner/image:v1", nil)
		assert.Equal(t, "ghcr.io/owner/image:v1", got)
	})
}

// resolveContainerImage redirects the source image through container_pins before
// digest lookup, ensuring the mapped registry is used in the compiled output.
func TestResolveContainerImage_AppliesContainerPinMapping(t *testing.T) {
	const digest = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	t.Run("mapped image returns digest-pinned redirected image", func(t *testing.T) {
		data := &WorkflowData{
			ContainerPinMappings: map[string]string{
				"ghcr.io/owner/img:v1": "registry.acme.com/img:v1@sha256:" + digest,
			},
		}
		got := resolveContainerImage("ghcr.io/owner/img:v1", data)
		assert.Equal(t, "registry.acme.com/img:v1@sha256:"+digest, got,
			"resolveContainerImage should return the mapped replacement image")
	})

	t.Run("mapped image does not inherit embedded pin from source", func(t *testing.T) {
		data := &WorkflowData{
			ContainerPinMappings: map[string]string{
				"node:lts-alpine": "registry.acme.com/node:lts-alpine@sha256:" + digest,
			},
		}
		got := resolveContainerImage("node:lts-alpine", data)
		assert.Equal(t, "registry.acme.com/node:lts-alpine@sha256:"+digest, got,
			"mapped image keeps its configured digest instead of the source image digest")
	})

	t.Run("unmapped image passes through to normal resolution", func(t *testing.T) {
		data := &WorkflowData{
			ContainerPinMappings: map[string]string{
				"other.registry.io/image:v1": "registry.acme.com/image:v1@sha256:" + digest,
			},
		}
		// node:lts-alpine is not in the mappings but has an embedded digest pin.
		got := resolveContainerImage("node:lts-alpine", data)
		assert.Contains(t, got, "sha256:",
			"unmapped image should still be resolved through normal digest-pin path")
	})
}
