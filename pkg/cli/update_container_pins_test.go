//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"testing"

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
Digest:    sha256:ca96b8acb27d8cf601a8faef86a084602cffa41d8cb18caa1e29ba4d16989d22

Manifests:
  Name:        docker.io/mcp/brave-search:latest@sha256:ae3b30d079370f67495d75085ffb73a11efcf9f9b23b919ffcb990ed2c076cfe
  MediaType:   application/vnd.oci.image.manifest.v1+json
  Platform:    linux/amd64
`,
			expectedDigest: "sha256:ca96b8acb27d8cf601a8faef86a084602cffa41d8cb18caa1e29ba4d16989d22",
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
