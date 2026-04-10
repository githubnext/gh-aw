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
		name     string
		images   []string
		pins     map[string]ContainerPin
		expected []string
	}{
		{
			name:     "no pins - images returned unchanged",
			images:   []string{"node:lts-alpine", "alpine:latest"},
			pins:     nil,
			expected: []string{"node:lts-alpine", "alpine:latest"},
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
			expected: []string{"node:lts-alpine@sha256:abc123"},
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
			expected: []string{"node:lts-alpine@sha256:abc123", "alpine:latest"},
		},
		{
			name:     "empty images list",
			images:   nil,
			pins:     nil,
			expected: []string{},
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

			result := applyContainerPins(tt.images, workflowData)
			require.Len(t, result, len(tt.expected), "result length")
			for i, img := range result {
				assert.Equal(t, tt.expected[i], img, "image at index %d", i)
			}
		})
	}
}

// TestCollectDockerImages_StoresInWorkflowData verifies that collectDockerImages
// populates workflowData.DockerImages with the collected (and pinned) image refs.
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
}

// TestMergeDockerImages verifies deduplication when merging two slices.
func TestMergeDockerImages(t *testing.T) {
	existing := []string{"image-a", "image-b"}
	newImages := []string{"image-b", "image-c"}

	result := mergeDockerImages(existing, newImages)

	assert.Equal(t, []string{"image-a", "image-b", "image-c"}, result, "deduplicated merge")
}
