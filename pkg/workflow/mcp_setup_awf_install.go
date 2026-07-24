package workflow

import "strings"

func generateAgenticWorkflowsInstallStep(c *Compiler, yaml *strings.Builder, hasAgenticWorkflows bool, workflowData *WorkflowData) {
	if !hasAgenticWorkflows {
		return
	}

	cliVersion := resolveAgenticWorkflowsCLIVersion(c, workflowData)
	effectiveToken := getEffectiveGitHubToken("")
	actionRepo := GitHubActionsOrgRepo + "/setup-cli"
	installStep, err := generateGhAwSetupStep(ghAwSetupStepConfig{
		actionMode:           c.actionMode,
		cliVersion:           cliVersion,
		actionRepo:           actionRepo,
		fallbackActionRefTag: cliVersion,
		workflowData:         workflowData,
		withFields: map[string]string{
			"github-token": effectiveToken,
		},
	})
	if err != nil {
		mcpSetupGeneratorLog.Printf("Failed to resolve pinned setup-cli action reference for %s@%s: %v", actionRepo, cliVersion, err)
	}
	for _, line := range installStep {
		yaml.WriteString(line + "\n")
	}
	yaml.WriteString("      - name: Copy gh-aw binary for MCP server\n")
	yaml.WriteString("        run: |\n")
	yaml.WriteString("          gh aw --version\n")
	yaml.WriteString("          # Copy the gh-aw binary to ${RUNNER_TEMP}/gh-aw for MCP server containerization\n")
	yaml.WriteString("          mkdir -p \"${RUNNER_TEMP}/gh-aw\"\n")
	yaml.WriteString("          GH_AW_BIN=\"\"\n")
	yaml.WriteString("          GH_AW_BIN=$(command -v gh-aw 2>/dev/null) || true\n")
	yaml.WriteString("          if [ -z \"$GH_AW_BIN\" ]; then\n")
	yaml.WriteString("            GH_AW_BIN=$(find \"${HOME}/.local/share/gh/extensions/gh-aw\" -name 'gh-aw' -type f 2>/dev/null | head -1) || true\n")
	yaml.WriteString("          fi\n")
	yaml.WriteString("          if [ -z \"$GH_AW_BIN\" ] && [ -n \"${GH_CONFIG_DIR:-}\" ]; then\n")
	yaml.WriteString("            GH_AW_BIN=$(find \"${GH_CONFIG_DIR}/extensions/gh-aw\" -name 'gh-aw' -type f 2>/dev/null | head -1) || true\n")
	yaml.WriteString("          fi\n")
	yaml.WriteString("          if [ -z \"$GH_AW_BIN\" ] && [ -f \"${GITHUB_WORKSPACE}/gh-aw\" ]; then\n")
	yaml.WriteString("            GH_AW_BIN=\"${GITHUB_WORKSPACE}/gh-aw\"\n")
	yaml.WriteString("          fi\n")
	yaml.WriteString("          if [ -n \"$GH_AW_BIN\" ] && [ -f \"$GH_AW_BIN\" ]; then\n")
	yaml.WriteString("            cp \"$GH_AW_BIN\" \"${RUNNER_TEMP}/gh-aw/gh-aw\"\n")
	yaml.WriteString("            chmod +x \"${RUNNER_TEMP}/gh-aw/gh-aw\"\n")
	yaml.WriteString("            echo \"Copied gh-aw binary to ${RUNNER_TEMP}/gh-aw/gh-aw\"\n")
	yaml.WriteString("          else\n")
	yaml.WriteString("            echo \"::error::Failed to find gh-aw binary for MCP server\"\n")
	yaml.WriteString("            exit 1\n")
	yaml.WriteString("          fi\n")
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
