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

func TestInstallCopilotCLIScriptResolvesCompatVersionBeforeToolcacheLookup(t *testing.T) {
	const compatVersion = "1.0.51"

	wd, err := os.Getwd()
	require.NoError(t, err, "Failed to get working directory")

	projectRoot := filepath.Join(wd, "..", "..")
	installScript := filepath.Join(projectRoot, "actions", "setup", "sh", "install_copilot_cli.sh")

	tempDir := t.TempDir()
	toolcacheBin := filepath.Join(tempDir, "toolcache", "copilot-cli", compatVersion, "x64", "bin")
	require.NoError(t, os.MkdirAll(toolcacheBin, 0o755))

	cachedCopilot := filepath.Join(toolcacheBin, "copilot")
	require.NoError(t, os.WriteFile(cachedCopilot, []byte("#!/usr/bin/env bash\necho 'copilot "+compatVersion+"'\n"), 0o755))

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
set -euo pipefail
output_file=""
url=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    -o)
      output_file="$2"
      shift 2
      ;;
    *)
      url="$1"
      shift
      ;;
  esac
done
echo "$url" >> "`+curlLog+`"
if [[ "$url" == *"/compat.json" ]]; then
  cat > "$output_file" <<'JSON'
{
  "agent-compat-v1": {
    "copilot": [
      {
        "min-gh-aw": "0.72.0",
        "max-gh-aw": "*",
        "min-agent": "1.0.21",
        "max-agent": "`+compatVersion+`"
      }
    ]
  }
}
JSON
  exit 0
fi
echo "unexpected URL: $url" >&2
exit 97
`), 0o755))

	githubPath := filepath.Join(tempDir, "github-path")
	cmd := exec.Command("bash", installScript)
	cmd.Env = append(os.Environ(),
		"RUNNER_TOOL_CACHE="+filepath.Join(tempDir, "toolcache"),
		"GITHUB_PATH="+githubPath,
		"GH_AW_COMPILED_VERSION=v0.72.5",
		"PATH="+fakeBinDir+":"+os.Getenv("PATH"),
	)

	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "install_copilot_cli.sh should resolve compat version and use toolcache: %s", output)

	assert.Contains(t, string(output), "Resolved Copilot CLI version from compatibility matrix: "+compatVersion)
	assert.Contains(t, string(output), "Using compat-resolved Copilot CLI version: "+compatVersion)
	assert.Contains(t, string(output), "Using cached GitHub Copilot CLI")

	curlLogContent, err := os.ReadFile(curlLog)
	require.NoError(t, err, "Expected curl to fetch compatibility matrix")
	assert.Contains(t, string(curlLogContent), "/compat.json", "compat matrix should be downloaded")
	assert.NotContains(t, string(curlLogContent), "SHA256SUMS.txt", "release downloads should not run when toolcache is hit")
}
