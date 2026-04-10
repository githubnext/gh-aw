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

// TestCollectImagesFromLockFiles_IgnoresNonLockFiles verifies that non-.lock.yml
// files are not scanned.
func TestCollectImagesFromLockFiles_IgnoresNonLockFiles(t *testing.T) {
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755))

	// This is a .yml file (not .lock.yml) — should be ignored.
	content := `run: bash "${RUNNER_TEMP}/gh-aw/actions/download_docker_images.sh" node:lts-alpine`
	require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "ci.yml"), []byte(content), 0644))

	images, err := collectImagesFromLockFiles(workflowsDir)
	require.NoError(t, err)
	assert.Empty(t, images, "non-lock-yml files should be ignored")
}
