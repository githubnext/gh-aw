//go:build !integration

package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstallCopilotCLIScriptUsesToolcacheBeforeDownload(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err, "Failed to get working directory")

	projectRoot := filepath.Join(wd, "..", "..")
	installScript := filepath.Join(projectRoot, "actions", "setup", "sh", "install_copilot_cli.sh")

	tempDir := t.TempDir()
	toolcacheBin := filepath.Join(tempDir, "toolcache", "copilot-cli", "1.2.3", "x64", "bin")
	require.NoError(t, os.MkdirAll(toolcacheBin, 0o755))

	cachedCopilot := filepath.Join(toolcacheBin, "copilot")
	require.NoError(t, os.WriteFile(cachedCopilot, []byte("#!/usr/bin/env bash\necho 'copilot 1.2.3'\n"), 0o755))

	fakeBinDir := filepath.Join(tempDir, "fake-bin")
	require.NoError(t, os.MkdirAll(fakeBinDir, 0o755))

	curlLog := filepath.Join(tempDir, "curl.log")
	sudoScript := filepath.Join(fakeBinDir, "sudo")
	curlScript := filepath.Join(fakeBinDir, "curl")

	require.NoError(t, os.WriteFile(sudoScript, []byte(`#!/usr/bin/env bash
if [ "${1:-}" = "chown" ]; then
  exit 0
fi
exec "$@"
`), 0o755))
	require.NoError(t, os.WriteFile(curlScript, []byte(`#!/usr/bin/env bash
echo curl-invoked >> "`+curlLog+`"
exit 97
`), 0o755))

	githubPath := filepath.Join(tempDir, "github-path")
	cmd := exec.Command("bash", installScript, "1.2.3")
	cmd.Env = append(os.Environ(),
		"RUNNER_TOOL_CACHE="+filepath.Join(tempDir, "toolcache"),
		"GITHUB_PATH="+githubPath,
		"PATH="+fakeBinDir+":"+os.Getenv("PATH"),
	)

	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "install_copilot_cli.sh should succeed with a toolcache hit: %s", output)

	assert.Contains(t, string(output), "Using cached GitHub Copilot CLI", "script should use the toolcache before downloading")
	assert.NoFileExists(t, curlLog, "curl should not run when a cached Copilot CLI is available")

	githubPathContent, err := os.ReadFile(githubPath)
	require.NoError(t, err, "Expected the script to append the cached bin dir to GITHUB_PATH")
	assert.Contains(t, string(githubPathContent), toolcacheBin, "cached Copilot bin directory should be exported for later steps")
}
