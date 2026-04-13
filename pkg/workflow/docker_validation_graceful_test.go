//go:build !integration && !js && !wasm

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
	if _, lookErr := exec.LookPath("docker"); lookErr != nil {
		err := validateDockerImage("ghcr.io/some/image:latest", false, false)
		assert.NoError(t, err, "should silently skip when Docker is not installed")
		return
	}
	if !isDockerDaemonRunning() {
		err := validateDockerImage("ghcr.io/some/image:latest", false, false)
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
	err := validateDockerImage("-malicious", false, false)
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

// TestValidateDockerImage_RequireDockerFailsWhenUnavailable verifies that
// when requireDocker is true, validateDockerImage returns an error instead
// of silently skipping when Docker is not installed or the daemon is not running.
func TestValidateDockerImage_RequireDockerFailsWhenUnavailable(t *testing.T) {
	if _, lookErr := exec.LookPath("docker"); lookErr != nil {
		err := validateDockerImage("ghcr.io/some/image:latest", false, true)
		require.Error(t, err, "should fail when Docker is not installed and requireDocker is true")
		assert.Contains(t, err.Error(), "docker not installed",
			"error should mention Docker is not installed")
		assert.Contains(t, err.Error(), "--validate-images",
			"error should mention the --validate-images flag")
		return
	}
	if !isDockerDaemonRunning() {
		err := validateDockerImage("ghcr.io/some/image:latest", false, true)
		require.Error(t, err, "should fail when Docker daemon is not running and requireDocker is true")
		assert.Contains(t, err.Error(), "docker daemon not running",
			"error should mention Docker daemon is not running")
		assert.Contains(t, err.Error(), "--validate-images",
			"error should mention the --validate-images flag")
		return
	}

	t.Skip("Docker is available — requireDocker failure path not exercised")
}

// TestValidateContainerImages_RequireDockerFailsWhenUnavailable verifies that
// when requireDocker is set on the compiler, validateContainerImages returns
// an error when Docker is unavailable.
func TestValidateContainerImages_RequireDockerFailsWhenUnavailable(t *testing.T) {
	if _, lookErr := exec.LookPath("docker"); lookErr == nil && isDockerDaemonRunning() {
		t.Skip("Docker is available — requireDocker failure path not exercised")
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
	compiler.SetRequireDocker(true)
	err := compiler.validateContainerImages(workflowData)
	require.Error(t, err, "container image validation should fail when Docker is unavailable and requireDocker is true")
}
