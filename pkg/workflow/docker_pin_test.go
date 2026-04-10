//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestApplyContainerPins verifies that applyContainerPins substitutes
// cached digest references while leaving unpinned images unchanged.
func TestApplyContainerPins(t *testing.T) {
	tests := []struct {
		name            string
		images          []string
		pins            map[string]ContainerPin
		expectedRefs    []string
		expectedDigests []string // expected Digest field in corresponding pin entry
	}{
		{
			name:            "no pins - images returned unchanged",
			images:          []string{"node:lts-alpine", "alpine:latest"},
			pins:            nil,
			expectedRefs:    []string{"node:lts-alpine", "alpine:latest"},
			expectedDigests: []string{"", ""},
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
			images: []string{"node:lts-alpine", "alpine:latest"},
			pins: map[string]ContainerPin{
				"node:lts-alpine": {
					Image:       "node:lts-alpine",
					Digest:      "sha256:abc123",
					PinnedImage: "node:lts-alpine@sha256:abc123",
				},
			},
			expectedRefs:    []string{"node:lts-alpine@sha256:abc123", "alpine:latest"},
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

// TestCollectDockerImages_StoresInWorkflowData verifies that collectDockerImages
// populates workflowData.DockerImages and DockerImagePins with the collected image refs.
func TestCollectDockerImages_StoresInWorkflowData(t *testing.T) {
	workflowData := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			CreateIssues: &CreateIssuesConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{},
			},
		},
	}

	tools := map[string]any{}

	images := collectDockerImages(tools, workflowData, ActionModeRelease)

	// DockerImages on workflowData should now be populated (node:lts-alpine from safe-outputs).
	require.NotEmpty(t, workflowData.DockerImages, "DockerImages should be populated after collectDockerImages")
	assert.Equal(t, images, workflowData.DockerImages, "DockerImages should match the returned slice")

	// DockerImagePins should also be populated with matching Image fields.
	require.NotEmpty(t, workflowData.DockerImagePins, "DockerImagePins should be populated")
	assert.Len(t, workflowData.DockerImagePins, len(workflowData.DockerImages), "pin count should match image count")
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
