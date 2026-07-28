package workflow

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
)

func buildModelsJSONPathExportScript(isArcDind bool) string {
	modelsJSONPathExpr := awfModelsJSONPathExpr
	if isArcDind {
		modelsJSONPathExpr = awfArcDindRootPathExpr + "/models.json"
	}
	return fmt.Sprintf(`export GH_AW_MODELS_JSON_PATH="%s"`, modelsJSONPathExpr)
}

func rewriteArcDindPath(path string) string {
	return strings.ReplaceAll(path, constants.TmpGhAwDir, awfArcDindRootPathExpr)
}

func rewriteArcDindEngineCommand(command string) string {
	rewritten := rewriteArcDindPath(command)
	return fmt.Sprintf("export HOME=%s\n%s", awfArcDindHomePathExpr, rewritten)
}

// buildArcDindChrootConfigPatchBody returns the Node.js command that patches the AWF
// config file with chroot.binariesSourcePath and chroot.identity.*. It is designed to be
// embedded inside a bash if-block that already guards on DOCKER_HOST=tcp://...
//
// Using the repository JavaScript helper avoids a runtime Python dependency and keeps the
// patch logic aligned with the rest of the actions/setup/js helpers.
// The config path under ${RUNNER_TEMP}/gh-aw is updated in place.
func buildArcDindChrootConfigPatchBody() string {
	return fmt.Sprintf(
		`  GH_AW_CHROOT_BINARIES_SOURCE_PATH="%s" GH_AW_CHROOT_IDENTITY_HOME="%s" node "${RUNNER_TEMP}/gh-aw/actions/patch_awf_chroot_config.cjs"`,
		awfArcDindChrootBinariesSourcePath,
		awfArcDindChrootIdentityHome,
	)
}

// buildArcDindChrootConfigPatchBodyBash returns bash commands (using jq) that patch the AWF
// config file with chroot.binariesSourcePath and chroot.identity.*. This is the bash
// equivalent of buildArcDindChrootConfigPatchBody, used for detection runs where Python
// must not be injected.
// The config path under ${RUNNER_TEMP}/gh-aw is updated in place.
func buildArcDindChrootConfigPatchBodyBash() string {
	return fmt.Sprintf(
		`  _GH_AW_CHROOT_JSON=$(jq -c --arg src "%s" --arg user "$(id -un)" --argjson uid "$(id -u)" --argjson gid "$(id -g)" --arg home "%s" '.chroot={"binariesSourcePath":$src,"identity":{"user":$user,"uid":$uid,"gid":$gid,"home":$home}}' "${RUNNER_TEMP}/gh-aw/awf-config.json") || { echo "chroot config patch failed" >&2; exit 1; }
  printf '%%s\n' "$_GH_AW_CHROOT_JSON" > "${RUNNER_TEMP}/gh-aw/awf-config.json"
  printf '%%s\n' "$_GH_AW_CHROOT_JSON" > "%s/awf-config.json"`,
		awfArcDindChrootBinariesSourcePath,
		awfArcDindChrootIdentityHome,
		awfArcDindChrootBinariesSourcePath,
	)
}
