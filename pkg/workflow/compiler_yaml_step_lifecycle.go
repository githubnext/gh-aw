package workflow

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow/compilerenv"
)

var compilerYamlStepLifecycleLog = logger.New("workflow:compiler_yaml:steps")

func (c *Compiler) generatePreSteps(yaml *strings.Builder, data *WorkflowData) {
	writeStepsSection(yaml, data.PreSteps)
}

func (c *Compiler) generatePostSteps(yaml *strings.Builder, data *WorkflowData) {
	writeStepsSection(yaml, data.PostSteps)
}

func (c *Compiler) generatePreAgentSteps(yaml *strings.Builder, data *WorkflowData) {
	writeStepsSection(yaml, data.PreAgentSteps)
}

// writeStepsSection writes a steps section (pre-steps, pre-agent-steps, or post-steps) to the YAML builder,
// stripping the header line and normalising indentation to match the agent job step format:
// top-level items get 6-space indent (      - name:) and nested properties get 8-space indent (        run:).
func writeStepsSection(yaml *strings.Builder, stepsYAML string) {
	if stepsYAML == "" {
		return
	}
	lines := strings.Split(stepsYAML, "\n")
	for _, line := range lines[1:] { // skip the "pre-steps:" / "pre-agent-steps:" / "post-steps:" header line
		trimmed := strings.TrimRight(line, " ")
		if strings.TrimSpace(trimmed) == "" {
			yaml.WriteString("\n")
			continue
		}
		if strings.HasPrefix(line, "  ") {
			yaml.WriteString("        " + line[2:] + "\n")
		} else {
			yaml.WriteString("      " + line + "\n")
		}
	}
}

func (c *Compiler) generateCreateAwInfo(yaml *strings.Builder, data *WorkflowData, engine CodingAgentEngine) {
	info := c.buildAwInfoContext(data, engine)

	compilerYamlStepLifecycleLog.Printf("Generating aw_info step: engine=%s, modelConfigured=%t, version=%s, firewallEnabled=%t, staged=%s", info.engineID, info.modelConfigured, info.version, info.firewallEnabled, info.stagedValue)

	yaml.WriteString("      - name: Generate agentic run info\n")
	yaml.WriteString("        id: generate_aw_info\n")
	yaml.WriteString("        env:\n")
	c.writeAwInfoEnv(yaml, data, engine, info)
	fmt.Fprintf(yaml, "        uses: %s\n", getCachedActionPin("actions/github-script", data))
	yaml.WriteString("        with:\n")
	yaml.WriteString("          script: |\n")
	yaml.WriteString("            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');\n")
	yaml.WriteString("            setupGlobals(core, github, context, exec, io, getOctokit);\n")
	yaml.WriteString("            const { main } = require('${{ runner.temp }}/gh-aw/actions/generate_aw_info.cjs');\n")
	yaml.WriteString("            await main(core, context);\n")
}

type awInfoContext struct {
	engineID          string
	modelConfigured   bool
	modelEnvVar       string
	agentVersion      string
	version           string
	stagedValue       string
	domainsJSON       string
	firewallEnabled   bool
	firewallVersion   string
	mcpGatewayVersion string
	firewallType      string
}

func (c *Compiler) buildAwInfoContext(data *WorkflowData, engine CodingAgentEngine) awInfoContext {
	engineID := engine.GetID()
	if data.EngineConfig != nil && data.EngineConfig.ID != "" {
		engineID = data.EngineConfig.ID
	} else if data.AI != "" {
		engineID = data.AI
	}
	agentVersion := getInstallationVersion(data, engine)
	info := awInfoContext{
		engineID:        engineID,
		modelConfigured: data.EngineConfig != nil && data.EngineConfig.Model != "",
		modelEnvVar:     modelEnvVarForEngine(engineID),
		agentVersion:    agentVersion,
		version:         agentVersion,
		stagedValue:     "false",
		domainsJSON:     "[]",
	}
	if data.EngineConfig != nil && data.EngineConfig.Version != "" {
		info.version = data.EngineConfig.Version
	}
	if data.SafeOutputs != nil && data.SafeOutputs.Staged != nil {
		info.stagedValue = data.SafeOutputs.Staged.String()
	}
	c.populateAwInfoNetworkFields(data, &info)
	return info
}

func modelEnvVarForEngine(engineID string) string {
	switch engineID {
	case "copilot":
		return constants.EnvVarModelAgentCopilot
	case "claude":
		return constants.EnvVarModelAgentClaude
	case "codex":
		return constants.EnvVarModelAgentCodex
	case "opencode":
		return constants.EnvVarModelAgentOpenCode
	default:
		return constants.EnvVarModelAgentCustom
	}
}

func (c *Compiler) populateAwInfoNetworkFields(data *WorkflowData, info *awInfoContext) {
	var allowedDomains []string
	if data.NetworkPermissions != nil {
		allowedDomains = data.NetworkPermissions.Allowed
	}
	if firewallConfig := getFirewallConfig(data); firewallConfig != nil {
		info.firewallEnabled = firewallConfig.Enabled
		info.firewallVersion = firewallConfig.Version
		if info.firewallEnabled && info.firewallVersion == "" {
			info.firewallVersion = string(constants.DefaultFirewallVersion)
		}
	}
	if len(allowedDomains) > 0 {
		b, _ := json.Marshal(allowedDomains) //nolint:jsonmarshalignoredeerror // marshaling a string slice cannot fail
		info.domainsJSON = string(b)
	}
	if data.SandboxConfig != nil && data.SandboxConfig.MCP != nil && data.SandboxConfig.MCP.Version != "" {
		info.mcpGatewayVersion = data.SandboxConfig.MCP.Version
	}
	if isFirewallEnabled(data) {
		info.firewallType = "squid"
	}
}

func (c *Compiler) writeAwInfoEnv(yaml *strings.Builder, data *WorkflowData, engine CodingAgentEngine, info awInfoContext) {
	c.writeAwInfoCoreEnv(yaml, data, engine, info)
	c.writeAwInfoSourceEnv(yaml, data)
	c.writeAwInfoLockdownEnv(yaml, data)
	writeAwInfoJSONEnv(yaml, data)
}

func (c *Compiler) writeAwInfoCoreEnv(yaml *strings.Builder, data *WorkflowData, engine CodingAgentEngine, info awInfoContext) {
	fmt.Fprintf(yaml, "          GH_AW_INFO_ENGINE_ID: \"%s\"\n", info.engineID)
	fmt.Fprintf(yaml, "          GH_AW_INFO_ENGINE_NAME: \"%s\"\n", engine.GetDisplayName())
	c.writeAwInfoModelEnv(yaml, data, info)
	fmt.Fprintf(yaml, "          GH_AW_INFO_VERSION: \"%s\"\n", info.version)
	fmt.Fprintf(yaml, "          GH_AW_INFO_AGENT_VERSION: \"%s\"\n", info.agentVersion)
	if IsReleasedVersion(c.version) {
		fmt.Fprintf(yaml, "          GH_AW_INFO_CLI_VERSION: \"%s\"\n", c.version)
	}
	fmt.Fprintf(yaml, "          GH_AW_INFO_WORKFLOW_NAME: \"%s\"\n", data.Name)
	fmt.Fprintf(yaml, "          GH_AW_INFO_EXPERIMENTAL: \"%t\"\n", engine.IsExperimental())
	fmt.Fprintf(yaml, "          GH_AW_INFO_SUPPORTS_TOOLS_ALLOWLIST: \"%t\"\n", engine.GetCapabilities().ToolsAllowlist)
	fmt.Fprintf(yaml, "          GH_AW_INFO_STAGED: \"%s\"\n", info.stagedValue)
	fmt.Fprintf(yaml, "          GH_AW_INFO_ALLOWED_DOMAINS: '%s'\n", info.domainsJSON)
	fmt.Fprintf(yaml, "          GH_AW_INFO_FIREWALL_ENABLED: \"%t\"\n", info.firewallEnabled)
	fmt.Fprintf(yaml, "          GH_AW_INFO_AWF_VERSION: \"%s\"\n", info.firewallVersion)
	fmt.Fprintf(yaml, "          GH_AW_INFO_AWMG_VERSION: \"%s\"\n", info.mcpGatewayVersion)
	fmt.Fprintf(yaml, "          GH_AW_INFO_FIREWALL_TYPE: \"%s\"\n", info.firewallType)
}

func (c *Compiler) writeAwInfoModelEnv(yaml *strings.Builder, data *WorkflowData, info awInfoContext) {
	if info.modelConfigured {
		fmt.Fprintf(yaml, "          GH_AW_INFO_MODEL: \"%s\"\n", data.EngineConfig.Model)
		return
	}
	defaultModel := getDefaultAgentModel(info.engineID)
	defaultModelOverrideVar := getDefaultModelOverrideVar(info.engineID)
	if defaultModel != "" && defaultModelOverrideVar != "" {
		fmt.Fprintf(yaml, "          GH_AW_INFO_MODEL: %s\n", compilerenv.BuildModelOverrideExpression(info.modelEnvVar, defaultModelOverrideVar, defaultModel))
	} else if defaultModel != "" {
		fmt.Fprintf(yaml, "          GH_AW_INFO_MODEL: ${{ vars.%s || '%s' }}\n", info.modelEnvVar, defaultModel)
	} else if defaultModelOverrideVar != "" {
		fmt.Fprintf(yaml, "          GH_AW_INFO_MODEL: %s\n", compilerenv.BuildModelOverrideExpressionEmptyFallback(info.modelEnvVar, defaultModelOverrideVar))
	} else {
		fmt.Fprintf(yaml, "          GH_AW_INFO_MODEL: ${{ vars.%s || '' }}\n", info.modelEnvVar)
	}
}

func (c *Compiler) writeAwInfoSourceEnv(yaml *strings.Builder, data *WorkflowData) {
	if data.Source != "" {
		fmt.Fprintf(yaml, "          GH_AW_INFO_FRONTMATTER_SOURCE: %q\n", data.Source)
		yaml.WriteString("          GH_AW_INFO_BODY_MODIFIED: \"false\"\n")
	}
	if data.FrontmatterEmoji != "" {
		fmt.Fprintf(yaml, "          GH_AW_INFO_FRONTMATTER_EMOJI: %q\n", data.FrontmatterEmoji)
	}
	fmt.Fprintf(yaml, "          GH_AW_COMPILED_STRICT: \"%t\"\n", c.effectiveStrictMode(data.RawFrontmatter))
	if hasWorkflowCallTrigger(data.On) && !data.InlinedImports {
		yaml.WriteString("          GH_AW_INFO_TARGET_REPO: ${{ steps.resolve-host-repo.outputs.target_repo }}\n")
	}
}

func (c *Compiler) writeAwInfoLockdownEnv(yaml *strings.Builder, data *WorkflowData) {
	githubTool, hasGitHub := data.Tools["github"]
	if hasGitHub && githubTool != false {
		toolConfig, _ := githubTool.(map[string]any)
		if hasGitHubLockdownExplicitlySet(toolConfig) && getGitHubLockdown(toolConfig) {
			yaml.WriteString("          GITHUB_MCP_LOCKDOWN_EXPLICIT: \"true\"\n")
			yaml.WriteString("          GH_AW_GITHUB_TOKEN: ${{ secrets.GH_AW_GITHUB_TOKEN }}\n")
			yaml.WriteString("          GH_AW_GITHUB_MCP_SERVER_TOKEN: ${{ secrets.GH_AW_GITHUB_MCP_SERVER_TOKEN }}\n")
			if customToken := getGitHubToken(toolConfig); customToken != "" {
				fmt.Fprintf(yaml, "          CUSTOM_GITHUB_TOKEN: %s\n", customToken)
			}
		}
	}
}

func writeAwInfoJSONEnv(yaml *strings.Builder, data *WorkflowData) {
	if len(data.ModelCosts) > 0 {
		writeEscapedJSONEnv(yaml, "GH_AW_INFO_MODEL_COSTS", data.ModelCosts)
	}
	if len(data.Features) > 0 {
		writeEscapedJSONEnv(yaml, "GH_AW_INFO_FEATURES", data.Features)
	}
	if len(data.Skills) > 0 {
		if skillsJSON, err := json.Marshal(data.Skills); err == nil {
			escapedSkillsJSON := strings.ReplaceAll(string(skillsJSON), "'", "''")
			fmt.Fprintf(yaml, "          GH_AW_INFO_SKILLS: '%s'\n", escapedSkillsJSON)
		} else {
			compilerYamlStepLifecycleLog.Printf("Failed to marshal skills for GH_AW_INFO_SKILLS, engine will not receive skill list: %v", err)
		}
	}
}

func writeEscapedJSONEnv(yaml *strings.Builder, name string, value any) {
	if jsonBytes, err := json.Marshal(value); err == nil {
		escapedJSON := strings.ReplaceAll(string(jsonBytes), "'", "''")
		fmt.Fprintf(yaml, "          %s: '%s'\n", name, escapedJSON)
	}
}

func (c *Compiler) generateOutputCollectionStep(yaml *strings.Builder, data *WorkflowData) error {
	writeSafeOutputCopyStep(yaml)
	writeOutputCollectionStepHeader(yaml, data)
	if err := c.writeOutputCollectionEnv(yaml, data); err != nil {
		return err
	}

	yaml.WriteString("        with:\n")
	yaml.WriteString("          script: |\n")
	yaml.WriteString("            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');\n")
	yaml.WriteString("            setupGlobals(core, github, context, exec, io, getOctokit);\n")
	yaml.WriteString("            const { main } = require('${{ runner.temp }}/gh-aw/actions/collect_ndjson_output.cjs');\n")
	yaml.WriteString("            await main();\n")

	return nil
}

func writeSafeOutputCopyStep(yaml *strings.Builder) {
	yaml.WriteString("      - name: Copy Safe Outputs\n")
	yaml.WriteString("        if: always()\n")
	yaml.WriteString("        env:\n")
	yaml.WriteString("          GH_AW_SAFE_OUTPUTS: ${{ steps.set-runtime-paths.outputs.GH_AW_SAFE_OUTPUTS }}\n")
	yaml.WriteString("        run: |\n")
	fmt.Fprintf(yaml, "          mkdir -p /tmp/gh-aw\n")
	fmt.Fprintf(yaml, "          cp \"$GH_AW_SAFE_OUTPUTS\" /tmp/gh-aw/%s 2>/dev/null || true\n", constants.SafeOutputsFilename)
}

func writeOutputCollectionStepHeader(yaml *strings.Builder, data *WorkflowData) {
	yaml.WriteString("      - name: Ingest agent output\n")
	yaml.WriteString("        id: collect_output\n")
	yaml.WriteString("        if: always()\n")
	fmt.Fprintf(yaml, "        uses: %s\n", getCachedActionPin("actions/github-script", data))
	yaml.WriteString("        env:\n")
	yaml.WriteString("          GH_AW_SAFE_OUTPUTS: ${{ steps.set-runtime-paths.outputs.GH_AW_SAFE_OUTPUTS }}\n")
}

func (c *Compiler) writeOutputCollectionEnv(yaml *strings.Builder, data *WorkflowData) error {
	domainsStr, err := c.outputCollectionAllowedDomains(data)
	if err != nil {
		return err
	}
	if domainsStr != "" {
		fmt.Fprintf(yaml, "          GH_AW_ALLOWED_DOMAINS: %q\n", domainsStr)
	}
	if data.SafeOutputs != nil && data.SafeOutputs.URLs != "" {
		fmt.Fprintf(yaml, "          GH_AW_SAFE_OUTPUTS_URLS: %q\n", data.SafeOutputs.URLs)
	}
	if data.SafeOutputs != nil && data.SafeOutputs.AllowGitHubReferences != nil {
		refsStr := strings.Join(data.SafeOutputs.AllowGitHubReferences, ",")
		fmt.Fprintf(yaml, "          GH_AW_ALLOWED_GITHUB_REFS: %q\n", refsStr)
	}
	yaml.WriteString("          GITHUB_SERVER_URL: ${{ github.server_url }}\n")
	yaml.WriteString("          GITHUB_API_URL: ${{ github.api_url }}\n")
	writeOutputCollectionCommandEnv(yaml, data)
	return nil
}

func (c *Compiler) outputCollectionAllowedDomains(data *WorkflowData) (string, error) {
	if data.SafeOutputs != nil && len(data.SafeOutputs.AllowedDomains) > 0 {
		return c.computeExpandedAllowedDomainsForSanitization(data)
	}
	return c.computeAllowedDomainsForSanitization(data)
}

func writeOutputCollectionCommandEnv(yaml *strings.Builder, data *WorkflowData) {
	if len(data.Command) > 0 {
		if commandsJSON, err := json.Marshal(data.Command); err == nil {
			fmt.Fprintf(yaml, "          GH_AW_COMMANDS: %q\n", string(commandsJSON))
		}
		if data.CommandPlaceholder != "" {
			fmt.Fprintf(yaml, "          GH_AW_COMMAND_PLACEHOLDER: %q\n", data.CommandPlaceholder)
		}
	}
	if len(data.LabelCommand) > 0 {
		if labelCommandsJSON, err := json.Marshal(data.LabelCommand); err == nil {
			fmt.Fprintf(yaml, "          GH_AW_LABEL_COMMANDS: %q\n", string(labelCommandsJSON))
		}
	}
}
