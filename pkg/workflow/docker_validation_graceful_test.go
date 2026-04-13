//go:build !integration

package workflow

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidateDockerImage_SkipsWhenDockerUnavailable verifies that
// validateDockerImage degrades gracefully (returns nil) when Docker
// is not installed or the daemon is not running, instead of returning
// an error that surfaces as a spurious warning.
func TestValidateDockerImage_SkipsWhenDockerUnavailable(t *testing.T) {
	// If docker is not installed or daemon not running, validation should
	// silently pass — no error, no warning.
	if _, err := exec.LookPath("docker"); err != nil {
		err := validateDockerImage("ghcr.io/some/image:latest", false)
		assert.NoError(t, err, "should silently skip when Docker is not installed")
		return
	}
	if !isDockerDaemonRunning() {
		err := validateDockerImage("ghcr.io/some/image:latest", false)
		assert.NoError(t, err, "should silently skip when Docker daemon is not running")
		return
	}

	t.Skip("Docker is available — graceful degradation path not exercised")
}

// TestValidateDockerImage_StillRejectsHyphenWithoutDocker verifies that
// the argument injection check still works even when Docker is unavailable.
func TestValidateDockerImage_StillRejectsHyphenWithoutDocker(t *testing.T) {
	// The hyphen-prefix guard runs before the Docker availability check,
	// so it should always reject invalid names regardless of Docker state.
	err := validateDockerImage("-malicious", false)
	require.Error(t, err, "should reject image names starting with hyphen regardless of Docker availability")
	assert.Contains(t, err.Error(), "names must not start with '-'",
		"error should explain why the name is invalid")
}

// TestValidateContainerImages_NoWarningWithoutDocker verifies that
// validateContainerImages does not produce errors when Docker is unavailable
// and the workflow references container-based tools.
func TestValidateContainerImages_NoWarningWithoutDocker(t *testing.T) {
	if _, lookErr := exec.LookPath("docker"); lookErr == nil && isDockerDaemonRunning() {
		t.Skip("Docker is available — graceful degradation path not exercised")
	}

	workflowData := &WorkflowData{
		Tools: map[string]any{
			"serena": map[string]any{
				"container": "ghcr.io/github/serena-mcp-server",
				"version":   "latest",
			},
		},
	}

	compiler := NewCompiler()
	err := compiler.validateContainerImages(workflowData)
	assert.NoError(t, err, "container image validation should silently pass when Docker is unavailable")
}
