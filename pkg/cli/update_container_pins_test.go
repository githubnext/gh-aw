//go:build !integration

package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCollectImagesFromLockFiles verifies that container image tags are correctly
// extracted from download_docker_images.sh invocations in lock files.
func TestCollectImagesFromLockFiles(t *testing.T) {
	tests := []struct {
		name            string
		lockFileContent string
		expectedImages  []string
	}{
		{
			name: "single image in lock file",
			lockFileContent: `name: test
jobs:
  setup:
    steps:
      - name: Download container images
        run: bash "${RUNNER_TEMP}/gh-aw/actions/download_docker_images.sh" node:lts-alpine
`,
			expectedImages: []string{"node:lts-alpine"},
		},
		{
			name: "multiple images in lock file",
			lockFileContent: `name: test
jobs:
  setup:
    steps:
      - name: Download container images
        run: bash "${RUNNER_TEMP}/gh-aw/actions/download_docker_images.sh" ghcr.io/github/gh-aw-mcpg:v0.2.17 ghcr.io/github/github-mcp-server:v0.32.0 node:lts-alpine
`,
			expectedImages: []string{
				"ghcr.io/github/gh-aw-mcpg:v0.2.17",
				"ghcr.io/github/github-mcp-server:v0.32.0",
				"node:lts-alpine",
			},
		},
		{
			name: "no docker images in lock file",
			lockFileContent: `name: test
jobs:
  build:
    steps:
      - uses: actions/checkout@v5
`,
			expectedImages: []string{},
		},
		{
			name: "images deduplicated across multiple lock files",
			// This test sets up two lock files with overlapping images.
			// The collect function should deduplicate.
			lockFileContent: `name: first
jobs:
  setup:
    steps:
      - run: bash "${RUNNER_TEMP}/gh-aw/actions/download_docker_images.sh" node:lts-alpine
`,
			// Second lock file added via helper below.
			expectedImages: []string{"node:lts-alpine"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
			require.NoError(t, os.MkdirAll(workflowsDir, 0755))

			// Write the primary lock file.
			lockPath := filepath.Join(workflowsDir, "test.lock.yml")
			require.NoError(t, os.WriteFile(lockPath, []byte(tt.lockFileContent), 0644))

			// For the deduplication test, write a second lock file with the same image.
			if tt.name == "images deduplicated across multiple lock files" {
				second := `name: second
jobs:
  setup:
    steps:
      - run: bash "${RUNNER_TEMP}/gh-aw/actions/download_docker_images.sh" node:lts-alpine
`
				require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "second.lock.yml"), []byte(second), 0644))
			}

			images, err := collectImagesFromLockFiles(workflowsDir)
			require.NoError(t, err, "collectImagesFromLockFiles should not error")
			assert.Equal(t, tt.expectedImages, images, "collected images")
		})
	}
}

// TestCollectImagesFromLockFiles_MissingDir verifies that a non-existent workflow
// directory returns nil without error.
func TestCollectImagesFromLockFiles_MissingDir(t *testing.T) {
	images, err := collectImagesFromLockFiles("/nonexistent/path/to/workflows")
	require.NoError(t, err, "missing dir should not return error")
	assert.Nil(t, images, "missing dir should return nil images")
}

// TestBuildxDigestPattern verifies that the regex correctly extracts the top-level
// "Digest:" line from docker buildx imagetools inspect text output.
func TestBuildxDigestPattern(t *testing.T) {
	tests := []struct {
		name           string
		output         string
		expectedDigest string
		shouldMatch    bool
	}{
		{
			name: "standard OCI index output",
			output: `Name:      docker.io/mcp/brave-search:latest
MediaType: application/vnd.oci.image.index.v1+json
Digest:    sha256:f58a5c22c1196ec7bd1ca586ce216f2334fc298550ddcf652c0e8adb6d256d78

Manifests:
  Name:        docker.io/mcp/brave-search:latest@sha256:ae3b30d079370f67495d75085ffb73a11efcf9f9b23b919ffcb990ed2c076cfe
  MediaType:   application/vnd.oci.image.manifest.v1+json
  Platform:    linux/amd64
`,
			expectedDigest: "sha256:f58a5c22c1196ec7bd1ca586ce216f2334fc298550ddcf652c0e8adb6d256d78",
			shouldMatch:    true,
		},
		{
			name: "single-platform image",
			output: `Name:      ghcr.io/github/github-mcp-server:v0.32.0
MediaType: application/vnd.oci.image.manifest.v1+json
Digest:    sha256:abc123def456abc123def456abc123def456abc123def456abc123def456abc1
`,
			expectedDigest: "sha256:abc123def456abc123def456abc123def456abc123def456abc123def456abc1",
			shouldMatch:    true,
		},
		{
			name: "picks top-level Digest not manifest sub-digest",
			output: `Name:      node:lts-alpine
Digest:    sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa

Manifests:
  Name:        node:lts-alpine@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb
`,
			expectedDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			shouldMatch:    true,
		},
		{
			name:        "no digest in output",
			output:      "Name:      unknown\nMediaType: unknown\n",
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := buildxDigestPattern.FindSubmatch([]byte(tt.output))
			if tt.shouldMatch {
				require.NotNil(t, matches, "expected pattern to match")
				assert.Equal(t, tt.expectedDigest, string(matches[1]), "extracted digest")
			} else {
				assert.Nil(t, matches, "expected pattern not to match")
			}
		})
	}
}

// TestUpdateContainerPins_PinnedLockFilesPreserveContainerPins verifies that when
// lock files already contain digest-pinned image references (image:tag@sha256:...),
// the existing container pins in actions-lock.json are NOT pruned.  This is the
// regression test for the bug where gh aw update wiped out all container pins
// because collectImagesFromLockFiles returned digest-suffixed keys that did not
// match the base-tag keys used in the container pins map.
//
// The test calls UpdateContainerPins end-to-end: because all images in the lock
// file are already digest-pinned, Docker is never invoked.
func TestUpdateContainerPins_PinnedLockFilesPreserveContainerPins(t *testing.T) {
	tmpDir := t.TempDir()

	// actions-lock.json has an existing container pin using the base image tag as key.
	actionsLockDir := filepath.Join(tmpDir, ".github", "aw")
	require.NoError(t, os.MkdirAll(actionsLockDir, 0755))
	actionsLockContent := `{
  "entries": {},
  "containers": {
    "ghcr.io/github/gh-aw-firewall/agent:0.27.9": {
      "image": "ghcr.io/github/gh-aw-firewall/agent:0.27.9",
      "digest": "sha256:13f522853a688bfe24b04adbbe40b68101e8ef4b6fe0b636068527141bf1c269",
      "pinned_image": "ghcr.io/github/gh-aw-firewall/agent:0.27.9@sha256:13f522853a688bfe24b04adbbe40b68101e8ef4b6fe0b636068527141bf1c269"
    }
  }
}
`
	actionsLockPath := filepath.Join(actionsLockDir, "actions-lock.json")
	require.NoError(t, os.WriteFile(actionsLockPath, []byte(actionsLockContent), 0644))

	// The compiled lock file already embeds the digest-pinned reference (image:tag@sha256:...).
	// This is the real-world case after a prior successful gh aw update run.
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755))
	lockFileContent := `name: test
jobs:
  setup:
    steps:
      - name: Download container images
        run: bash "${RUNNER_TEMP}/gh-aw/actions/download_docker_images.sh" ghcr.io/github/gh-aw-firewall/agent:0.27.9@sha256:13f522853a688bfe24b04adbbe40b68101e8ef4b6fe0b636068527141bf1c269
`
	require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "my-workflow.lock.yml"), []byte(lockFileContent), 0644))

	// UpdateContainerPins uses "." as the repo root for the action cache, so we
	// chdir into tmpDir before calling it and restore the original directory after.
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir) //nolint:errcheck
	require.NoError(t, os.Chdir(tmpDir))

	// Call UpdateContainerPins end-to-end. Because the lock file image is already
	// digest-pinned (@sha256:...), Docker is never invoked. The function should
	// prune zero pins (the bug caused it to prune all of them) and return false
	// (no new pins were added).
	added, err := UpdateContainerPins(context.Background(), workflowsDir, false)
	require.NoError(t, err)
	assert.False(t, added, "no new pins should be added when all images are already pinned")

	// Reload the cache from disk and confirm the original pin is still present.
	cache := workflow.NewActionCache(tmpDir)
	require.NoError(t, cache.Load())

	pin, ok := cache.GetContainerPin("ghcr.io/github/gh-aw-firewall/agent:0.27.9")
	require.True(t, ok, "container pin should still be present after UpdateContainerPins")
	assert.Equal(t, "sha256:13f522853a688bfe24b04adbbe40b68101e8ef4b6fe0b636068527141bf1c269", pin.Digest)

	// Verify the on-disk lock file is unchanged.
	data, err := os.ReadFile(actionsLockPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "ghcr.io/github/gh-aw-firewall/agent:0.27.9", "container pin should still be in actions-lock.json")
}

// TestUpdateContainerPins_PrunesStaleEntries verifies that UpdateContainerPins
// removes container pin entries from actions-lock.json that are no longer
// referenced in the compiled lock files (e.g. superseded AWF versions).
func TestUpdateContainerPins_PrunesStaleEntries(t *testing.T) {
	// Create a temp directory acting as the repo root.
	tmpDir := t.TempDir()

	// Write an actions-lock.json with a stale container pin and a live one.
	// The live pin (0.27.2) should be kept; the stale one (0.27.0) should be pruned.
	actionsLockDir := filepath.Join(tmpDir, ".github", "aw")
	require.NoError(t, os.MkdirAll(actionsLockDir, 0755))
	actionsLockContent := `{
  "entries": {},
  "containers": {
    "ghcr.io/github/gh-aw-firewall/agent:0.27.0": {
      "image": "ghcr.io/github/gh-aw-firewall/agent:0.27.0",
      "digest": "sha256:olddigest0000000000000000000000000000000000000000000000000000000",
      "pinned_image": "ghcr.io/github/gh-aw-firewall/agent:0.27.0@sha256:olddigest0000000000000000000000000000000000000000000000000000000"
    },
    "ghcr.io/github/gh-aw-firewall/agent:0.27.2": {
      "image": "ghcr.io/github/gh-aw-firewall/agent:0.27.2",
      "digest": "sha256:newdigest0000000000000000000000000000000000000000000000000000000",
      "pinned_image": "ghcr.io/github/gh-aw-firewall/agent:0.27.2@sha256:newdigest0000000000000000000000000000000000000000000000000000000"
    }
  }
}
`
	actionsLockPath := filepath.Join(actionsLockDir, "actions-lock.json")
	require.NoError(t, os.WriteFile(actionsLockPath, []byte(actionsLockContent), 0644))

	// Write a lock file referencing the NEW AWF version (0.27.2), not the old one.
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755))
	lockFileContent := `name: test
jobs:
  setup:
    steps:
      - name: Download container images
        run: bash "${RUNNER_TEMP}/gh-aw/actions/download_docker_images.sh" ghcr.io/github/gh-aw-firewall/agent:0.27.2
`
	require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "my-workflow.lock.yml"), []byte(lockFileContent), 0644))

	// collectImagesFromLockFiles should find the new version only.
	images, err := collectImagesFromLockFiles(workflowsDir)
	require.NoError(t, err)
	assert.Equal(t, []string{"ghcr.io/github/gh-aw-firewall/agent:0.27.2"}, images)

	// Load the cache and exercise PruneStaleContainerPins directly (Docker is not
	// available in unit tests so we can't call the full UpdateContainerPins function).
	cache := workflow.NewActionCache(tmpDir)
	require.NoError(t, cache.Load())

	imageSet := map[string]struct{}{"ghcr.io/github/gh-aw-firewall/agent:0.27.2": {}}
	pruned := cache.PruneStaleContainerPins(imageSet)
	assert.Equal(t, 1, pruned, "stale 0.27.0 entry should be pruned")

	_, ok := cache.GetContainerPin("ghcr.io/github/gh-aw-firewall/agent:0.27.0")
	assert.False(t, ok, "old-version pin should not be in cache after prune")

	pin, ok := cache.GetContainerPin("ghcr.io/github/gh-aw-firewall/agent:0.27.2")
	require.True(t, ok, "current-version pin should still be in cache")
	assert.Equal(t, "sha256:newdigest0000000000000000000000000000000000000000000000000000000", pin.Digest)

	// Save and verify the stale entry is gone from disk.
	require.NoError(t, cache.Save())

	data, err := os.ReadFile(actionsLockPath)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "0.27.0", "stale version should not appear in saved file")
	assert.Contains(t, string(data), "0.27.2", "current version should be in saved file")
}
