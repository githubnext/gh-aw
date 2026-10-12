package workflow

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
)

const nvxGuestRuntimeDir = "/workspace/.gh-aw-nvx/gh-aw"

func rewriteNVXEngineCommand(command string) string {
	// NVX never exports runner.temp. Its tool cache is optional, and the
	// workload layer already supplies toolchains for workspace-only runs.
	optionalToolCacheSetup := strings.ReplaceAll(GetNpmBinPathSetup(), `: "${RUNNER_TOOL_CACHE:?RUNNER_TOOL_CACHE must be set}"; `, "")
	command = strings.ReplaceAll(command, GetNpmBinPathSetup(),
		`if [ -n "${RUNNER_TOOL_CACHE:-}" ]; then `+optionalToolCacheSetup+`; fi`)
	command = strings.ReplaceAll(command, constants.GhAwRootDirShell, nvxGuestRuntimeDir)
	return fmt.Sprintf(`"%s/bin/node" "%s/actions/nvx_assets.cjs" guest %s`,
		nvxGuestRuntimeDir, nvxGuestRuntimeDir, shellEscapeArg(command))
}

func buildNVXAssetSetup() string {
	return `if [ ! -x /usr/bin/patchelf ]; then
  sudo apt-get update || exit $?
  sudo apt-get install -y --no-install-recommends patchelf || exit $?
fi
node "${RUNNER_TEMP}/gh-aw/actions/nvx_assets.cjs" stage || exit $?
export GH_AW_NVX_HOST_ROOT="${RUNNER_TEMP}/gh-aw"
export GH_AW_NVX_HOST_WORKSPACE="${GITHUB_WORKSPACE}"
export GH_AW_NVX_HOST_HOME="${HOME}"`
}

func generateNVXAssetCollectionStep(data *WorkflowData) GitHubActionStep {
	if !isNVXRuntime(data) || data.UseSamples {
		return nil
	}
	return GitHubActionStep{
		"      - name: Collect NVX guest outputs and remove runtime assets",
		"        if: always()",
		`        run: node "${RUNNER_TEMP}/gh-aw/actions/nvx_assets.cjs" collect`,
	}
}
