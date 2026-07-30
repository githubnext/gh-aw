//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildDockerVolumeMount(t *testing.T) {
	tmpDir := t.TempDir()

	mount, err := buildDockerVolumeMount(tmpDir, "/workdir")
	require.NoError(t, err)
	require.Equal(t, tmpDir+":/workdir", mount)

	_, err = buildDockerVolumeMount("relative/path", "/workdir")
	require.Error(t, err)
	require.ErrorContains(t, err, "invalid host path")

	_, err = buildDockerVolumeMount(tmpDir, "workdir")
	require.Error(t, err)
	require.ErrorContains(t, err, "container path must be absolute")

	_, err = buildDockerVolumeMount(tmpDir, "/work:dir")
	require.Error(t, err)
	require.ErrorContains(t, err, "reserved characters")

	_, err = buildDockerVolumeMount(tmpDir, "/work\x00dir")
	require.Error(t, err)
	require.ErrorContains(t, err, "invalid control characters")

	if runtime.GOOS != "windows" {
		_, err = buildDockerVolumeMount("/tmp/repo:fixture", "/workdir")
		require.Error(t, err)
		require.ErrorContains(t, err, "unsupported ':'")
	}
}

func TestBuildDockerReadonlyFileMount(t *testing.T) {
	tmpDir := t.TempDir()
	policyFile := filepath.Join(tmpDir, ".grant.yaml")
	require.NoError(t, os.WriteFile(policyFile, []byte("policy: true\n"), 0o644))

	mount, err := buildDockerReadonlyFileMount(policyFile, "/tmp/policy.yaml")
	require.NoError(t, err)
	require.Equal(t, policyFile+":/tmp/policy.yaml:ro", mount)

	_, err = buildDockerReadonlyFileMount(tmpDir, "/tmp/policy.yaml")
	require.Error(t, err)
	require.ErrorContains(t, err, "not a regular file")

	_, err = buildDockerReadonlyFileMount(policyFile, "/tmp/policy:yaml")
	require.Error(t, err)
	require.ErrorContains(t, err, "reserved characters")

	_, err = buildDockerReadonlyFileMount(policyFile, "/tmp/policy\x00yaml")
	require.Error(t, err)
	require.ErrorContains(t, err, "invalid control characters")
}
