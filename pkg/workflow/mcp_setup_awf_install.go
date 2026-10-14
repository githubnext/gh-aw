package workflow

import "strings"

func generateAgenticWorkflowsInstallStep(c *Compiler, yaml *strings.Builder, hasAgenticWorkflows bool, workflowData *WorkflowData) {
	if !hasAgenticWorkflows {
		return
	}

	cliVersion := resolveAgenticWorkflowsCLIVersion(c, workflowData)
	resolvedToken := resolveGitHubToken("")
	actionRepo := GitHubActionsOrgRepo + "/setup-cli"
	installStep, err := generateGhAwSetupStep(ghAwSetupStepConfig{
		actionMode:           c.actionMode,
		cliVersion:           cliVersion,
		actionRepo:           actionRepo,
		fallbackActionRefTag: cliVersion,
		workflowData:         workflowData,
		withFields: map[string]string{
			"github-token": resolvedToken,
		},
	})
	if err != nil {
		mcpSetupGeneratorLog.Printf("Failed to resolve pinned setup-cli action reference for %s@%s: %v", actionRepo, cliVersion, err)
	}
	for _, line := range installStep {
		yaml.WriteString(line + "\n")
	}
	yaml.WriteString("      - name: Copy gh-aw binary for MCP Server\n")
	yaml.WriteString("        run: |\n")
	for line := range strings.SplitSeq(agenticWorkflowsMCPHomeSetupScript(), "\n") {
		yaml.WriteString("          " + line + "\n")
	}
	yaml.WriteString("          bash \"${RUNNER_TEMP}/gh-aw/actions/copy_gh_aw_binary_for_mcp.sh\"\n")
}

func agenticWorkflowsMCPHomeSetupScript() string {
	return `set -euo pipefail
home_parent="${RUNNER_TEMP:?}/gh-aw"
if [[ "$home_parent" != /* || "$home_parent" == */../* || "$home_parent" == */./* || "$home_parent" == *$'\n'* || "$home_parent" == *$'\r'* || ! -d "$home_parent" ]]; then
  echo "::error::Agentic workflows MCP home requires an absolute runner-temp runtime directory" >&2
  exit 1
fi
check_path="$home_parent"
while [[ "$check_path" != / ]]; do
  if [[ -L "$check_path" ]]; then
    echo "::error::Agentic workflows MCP home must not traverse symlinks" >&2
    exit 1
  fi
  check_path="$(dirname -- "$check_path")"
done
` + agenticWorkflowsMCPHomeEnv + `="$(umask 077 && mktemp -d -- "$home_parent/aw-mcp-home.XXXXXXXXXX")"
printf '%s=%s\n' '` + agenticWorkflowsMCPHomeEnv + `' "$` + agenticWorkflowsMCPHomeEnv + `" >> "${GITHUB_ENV:?}"`
}

func resolveAgenticWorkflowsCLIVersion(c *Compiler, workflowData *WorkflowData) string {
	cliVersion := c.actionTag
	if cliVersion == "" {
		cliVersion = getActionTagFromFeatures(workflowData)
	}
	if cliVersion == "" {
		cliVersion = c.version
	}
	// "dev" and empty versions are not valid release pins; fall back to the
	// current compiler runtime version so setup-cli always receives a concrete
	// pinned release tag in non-dev modes.
	if cliVersion == "" || cliVersion == "dev" {
		cliVersion = getDefaultGhAWRuntimeVersion()
	}
	return cliVersion
}

func getActionTagFromFeatures(workflowData *WorkflowData) string {
	if workflowData == nil || workflowData.Features == nil {
		return ""
	}
	actionTagVal, exists := workflowData.Features["action-tag"]
	if !exists {
		return ""
	}
	actionTagStr, ok := actionTagVal.(string)
	if !ok || actionTagStr == "" {
		return ""
	}
	return actionTagStr
}
