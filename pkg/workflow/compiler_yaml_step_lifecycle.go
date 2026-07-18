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

type awInfoParams struct {
	engineID               string
	engineName             string
	modelConfigured        bool
	model                  string
	modelEnvVar            string
	version                string
	agentVersion           string
	workflowName           string
	stagedValue            string
	domainsJSON            string
	firewallEnabled        bool
	firewallVersion        string
	mcpGatewayVersion      string
	firewallType           string
	experimental           bool
	supportsToolsAllowlist bool
	frontmatterSource      string
	frontmatterEmoji       string
	compiledStrict         bool
	targetRepoExpr         string
	includeLockdown        bool
	customGitHubToken      string
	modelCostsJSON         string
	featuresJSON           string
	skillsJSON             string
}

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
	for _, line := range lines[1:] {
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
	params := resolveAwInfoParams(c, data, engine)
	compilerYamlStepLifecycleLog.Printf("Generating aw_info step: engine=%s, modelConfigured=%t, version=%s, firewallEnabled=%t, staged=%s", params.engineID, params.modelConfigured, params.version, params.firewallEnabled, params.stagedValue)
	writeAwInfoStep(yaml, c, data, engine, params)
}

func resolveAwInfoParams(c *Compiler, data *WorkflowData, engine CodingAgentEngine) awInfoParams {
	engineID := resolveAwInfoEngineID(data, engine)
	agentVersion := getInstallationVersion(data, engine)
	params := awInfoParams{
		engineID:               engineID,
		engineName:             engine.GetDisplayName(),
		modelConfigured:        data.EngineConfig != nil && data.EngineConfig.Model != "",
		version:                resolveAwInfoVersion(data, agentVersion),
		agentVersion:           agentVersion,
		workflowName:           data.Name,
		stagedValue:            resolveAwInfoStagedValue(data),
		domainsJSON:            resolveAwInfoDomainsJSON(data),
		firewallEnabled:        resolveAwInfoFirewallEnabled(data),
		firewallVersion:        resolveAwInfoFirewallVersion(data),
		mcpGatewayVersion:      resolveAwInfoMCPGatewayVersion(data),
		firewallType:           resolveAwInfoFirewallType(data),
		experimental:           engine.IsExperimental(),
		supportsToolsAllowlist: engine.GetCapabilities().ToolsAllowlist,
		frontmatterSource:      data.Source,
		frontmatterEmoji:       data.FrontmatterEmoji,
		compiledStrict:         c.effectiveStrictMode(data.RawFrontmatter),
		targetRepoExpr:         resolveAwInfoTargetRepoExpr(data),
		modelCostsJSON:         marshalAwInfoJSON(data.ModelCosts),
		featuresJSON:           marshalAwInfoJSON(data.Features),
		skillsJSON:             marshalAwInfoSkillsJSON(data.Skills),
	}
	params.model, params.modelEnvVar = resolveAwInfoModelValues(data, engineID)
	params.includeLockdown, params.customGitHubToken = resolveAwInfoLockdownValues(data)
	return params
}

func resolveAwInfoEngineID(data *WorkflowData, engine CodingAgentEngine) string {
	if data.EngineConfig != nil && data.EngineConfig.ID != "" {
		return data.EngineConfig.ID
	}
	if data.AI != "" {
		return data.AI
	}
	return engine.GetID()
}

func resolveAwInfoVersion(data *WorkflowData, agentVersion string) string {
	if data.EngineConfig != nil && data.EngineConfig.Version != "" {
		return data.EngineConfig.Version
	}
	return agentVersion
}

func resolveAwInfoStagedValue(data *WorkflowData) string {
	if data.SafeOutputs != nil && data.SafeOutputs.Staged != nil {
		return data.SafeOutputs.Staged.String()
	}
	return "false"
}

func resolveAwInfoDomainsJSON(data *WorkflowData) string {
	if data.NetworkPermissions == nil || len(data.NetworkPermissions.Allowed) == 0 {
		return "[]"
	}
	b, _ := json.Marshal(data.NetworkPermissions.Allowed) //nolint:jsonmarshalignoredeerror
	return string(b)
}

func resolveAwInfoFirewallEnabled(data *WorkflowData) bool {
	if firewallConfig := getFirewallConfig(data); firewallConfig != nil {
		return firewallConfig.Enabled
	}
	return false
}

func resolveAwInfoFirewallVersion(data *WorkflowData) string {
	firewallConfig := getFirewallConfig(data)
	if firewallConfig == nil || !firewallConfig.Enabled {
		return ""
	}
	if firewallConfig.Version != "" {
		return firewallConfig.Version
	}
	return string(constants.DefaultFirewallVersion)
}

func resolveAwInfoMCPGatewayVersion(data *WorkflowData) string {
	if data.SandboxConfig != nil && data.SandboxConfig.MCP != nil {
		return data.SandboxConfig.MCP.Version
	}
	return ""
}

func resolveAwInfoFirewallType(data *WorkflowData) string {
	if isFirewallEnabled(data) {
		return "squid"
	}
	return ""
}

func resolveAwInfoTargetRepoExpr(data *WorkflowData) string {
	if hasWorkflowCallTrigger(data.On) && !data.InlinedImports {
		return "${{ steps.resolve-host-repo.outputs.target_repo }}"
	}
	return ""
}

func resolveAwInfoModelValues(data *WorkflowData, engineID string) (string, string) {
	if data.EngineConfig != nil && data.EngineConfig.Model != "" {
		return data.EngineConfig.Model, ""
	}
	return "", resolveAwInfoModelEnvVar(engineID)
}

func resolveAwInfoModelEnvVar(engineID string) string {
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

func resolveAwInfoLockdownValues(data *WorkflowData) (bool, string) {
	githubTool, hasGitHub := data.Tools["github"]
	if !hasGitHub || githubTool == false {
		return false, ""
	}
	toolConfig, _ := githubTool.(map[string]any)
	if !hasGitHubLockdownExplicitlySet(toolConfig) || !getGitHubLockdown(toolConfig) {
		return false, ""
	}
	return true, getGitHubToken(toolConfig)
}

func marshalAwInfoJSON(value any) string {
	if isEmptyAwInfoValue(value) {
		return ""
	}
	b, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return strings.ReplaceAll(string(b), "'", "''")
}

func marshalAwInfoSkillsJSON(skills []string) string {
	if len(skills) == 0 {
		return ""
	}
	b, err := json.Marshal(skills)
	if err != nil {
		compilerYamlStepLifecycleLog.Printf("Failed to marshal skills for GH_AW_INFO_SKILLS, engine will not receive skill list: %v", err)
		return ""
	}
	return strings.ReplaceAll(string(b), "'", "''")
}

func isEmptyAwInfoValue(value any) bool {
	switch v := value.(type) {
	case nil:
		return true
	case []string:
		return len(v) == 0
	case map[string]float64:
		return len(v) == 0
	case map[string]any:
		return len(v) == 0
	default:
		return false
	}
}

func writeAwInfoStep(yaml *strings.Builder, c *Compiler, data *WorkflowData, engine CodingAgentEngine, p awInfoParams) {
	yaml.WriteString("      - name: Generate agentic run info\n")
	yaml.WriteString("        id: generate_aw_info\n")
	yaml.WriteString("        env:\n")
	writeAwInfoEnvVars(yaml, c, p)
	fmt.Fprintf(yaml, "        uses: %s\n", getCachedActionPin("actions/github-script", data))
	yaml.WriteString("        with:\n")
	yaml.WriteString("          script: |\n")
	yaml.WriteString("            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');\n")
	yaml.WriteString("            setupGlobals(core, github, context, exec, io, getOctokit);\n")
	yaml.WriteString("            const { main } = require('${{ runner.temp }}/gh-aw/actions/generate_aw_info.cjs');\n")
	yaml.WriteString("            await main(core, context);\n")
}

func writeAwInfoEnvVars(yaml *strings.Builder, c *Compiler, p awInfoParams) {
	fmt.Fprintf(yaml, "          GH_AW_INFO_ENGINE_ID: \"%s\"\n", p.engineID)
	fmt.Fprintf(yaml, "          GH_AW_INFO_ENGINE_NAME: \"%s\"\n", p.engineName)
	fmt.Fprintf(yaml, "          GH_AW_INFO_MODEL: %s\n", resolveAwInfoModelExpression(p.engineID, p.modelEnvVar, p.model))
	fmt.Fprintf(yaml, "          GH_AW_INFO_VERSION: \"%s\"\n", p.version)
	fmt.Fprintf(yaml, "          GH_AW_INFO_AGENT_VERSION: \"%s\"\n", p.agentVersion)
	writeAwInfoOptionalCLIEnvVar(yaml, c.version)
	fmt.Fprintf(yaml, "          GH_AW_INFO_WORKFLOW_NAME: \"%s\"\n", p.workflowName)
	fmt.Fprintf(yaml, "          GH_AW_INFO_EXPERIMENTAL: \"%t\"\n", p.experimental)
	fmt.Fprintf(yaml, "          GH_AW_INFO_SUPPORTS_TOOLS_ALLOWLIST: \"%t\"\n", p.supportsToolsAllowlist)
	fmt.Fprintf(yaml, "          GH_AW_INFO_STAGED: \"%s\"\n", p.stagedValue)
	fmt.Fprintf(yaml, "          GH_AW_INFO_ALLOWED_DOMAINS: '%s'\n", p.domainsJSON)
	fmt.Fprintf(yaml, "          GH_AW_INFO_FIREWALL_ENABLED: \"%t\"\n", p.firewallEnabled)
	fmt.Fprintf(yaml, "          GH_AW_INFO_AWF_VERSION: \"%s\"\n", p.firewallVersion)
	fmt.Fprintf(yaml, "          GH_AW_INFO_AWMG_VERSION: \"%s\"\n", p.mcpGatewayVersion)
	fmt.Fprintf(yaml, "          GH_AW_INFO_FIREWALL_TYPE: \"%s\"\n", p.firewallType)
	writeAwInfoOptionalEnvVars(yaml, p)
}

func resolveAwInfoModelExpression(engineID string, modelEnvVar string, model string) string {
	if model != "" {
		return fmt.Sprintf("\"%s\"", model)
	}
	defaultModel := getDefaultAgentModel(engineID)
	defaultModelOverrideVar := getDefaultModelOverrideVar(engineID)
	if defaultModel != "" && defaultModelOverrideVar != "" {
		return compilerenv.BuildModelOverrideExpression(modelEnvVar, defaultModelOverrideVar, defaultModel)
	}
	if defaultModel != "" {
		return fmt.Sprintf("${{ vars.%s || '%s' }}", modelEnvVar, defaultModel)
	}
	if defaultModelOverrideVar != "" {
		return compilerenv.BuildModelOverrideExpressionEmptyFallback(modelEnvVar, defaultModelOverrideVar)
	}
	return fmt.Sprintf("${{ vars.%s || '' }}", modelEnvVar)
}

func writeAwInfoOptionalCLIEnvVar(yaml *strings.Builder, version string) {
	if IsReleasedVersion(version) {
		fmt.Fprintf(yaml, "          GH_AW_INFO_CLI_VERSION: \"%s\"\n", version)
	}
}

func writeAwInfoOptionalEnvVars(yaml *strings.Builder, p awInfoParams) {
	writeAwInfoSourceEnvVars(yaml, p)
	if p.frontmatterEmoji != "" {
		fmt.Fprintf(yaml, "          GH_AW_INFO_FRONTMATTER_EMOJI: %q\n", p.frontmatterEmoji)
	}
	fmt.Fprintf(yaml, "          GH_AW_COMPILED_STRICT: \"%t\"\n", p.compiledStrict)
	if p.targetRepoExpr != "" {
		fmt.Fprintf(yaml, "          GH_AW_INFO_TARGET_REPO: %s\n", p.targetRepoExpr)
	}
	writeAwInfoLockdownEnvVars(yaml, p)
	writeAwInfoJSONEnvVar(yaml, "GH_AW_INFO_MODEL_COSTS", p.modelCostsJSON)
	writeAwInfoJSONEnvVar(yaml, "GH_AW_INFO_FEATURES", p.featuresJSON)
	writeAwInfoJSONEnvVar(yaml, "GH_AW_INFO_SKILLS", p.skillsJSON)
}

func writeAwInfoSourceEnvVars(yaml *strings.Builder, p awInfoParams) {
	if p.frontmatterSource == "" {
		return
	}
	fmt.Fprintf(yaml, "          GH_AW_INFO_FRONTMATTER_SOURCE: %q\n", p.frontmatterSource)
	yaml.WriteString("          GH_AW_INFO_BODY_MODIFIED: \"false\"\n")
}

func writeAwInfoLockdownEnvVars(yaml *strings.Builder, p awInfoParams) {
	if !p.includeLockdown {
		return
	}
	yaml.WriteString("          GITHUB_MCP_LOCKDOWN_EXPLICIT: \"true\"\n")
	yaml.WriteString("          GH_AW_GITHUB_TOKEN: ${{ secrets.GH_AW_GITHUB_TOKEN }}\n")
	yaml.WriteString("          GH_AW_GITHUB_MCP_SERVER_TOKEN: ${{ secrets.GH_AW_GITHUB_MCP_SERVER_TOKEN }}\n")
	if p.customGitHubToken != "" {
		fmt.Fprintf(yaml, "          CUSTOM_GITHUB_TOKEN: %s\n", p.customGitHubToken)
	}
}

func writeAwInfoJSONEnvVar(yaml *strings.Builder, name string, value string) {
	if value != "" {
		fmt.Fprintf(yaml, "          %s: '%s'\n", name, value)
	}
}

func (c *Compiler) generateOutputCollectionStep(yaml *strings.Builder, data *WorkflowData) error {
	writeOutputCollectionCopyStep(yaml)
	domainsStr, err := resolveAllowedDomainsStr(c, data)
	if err != nil {
		return err
	}
	yaml.WriteString("      - name: Ingest agent output\n")
	yaml.WriteString("        id: collect_output\n")
	yaml.WriteString("        if: always()\n")
	fmt.Fprintf(yaml, "        uses: %s\n", getCachedActionPin("actions/github-script", data))
	writeOutputCollectionEnvVars(yaml, data, domainsStr)
	yaml.WriteString("        with:\n")
	yaml.WriteString("          script: |\n")
	yaml.WriteString("            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');\n")
	yaml.WriteString("            setupGlobals(core, github, context, exec, io, getOctokit);\n")
	yaml.WriteString("            const { main } = require('${{ runner.temp }}/gh-aw/actions/collect_ndjson_output.cjs');\n")
	yaml.WriteString("            await main();\n")
	return nil
}

func writeOutputCollectionCopyStep(yaml *strings.Builder) {
	yaml.WriteString("      - name: Copy Safe Outputs\n")
	yaml.WriteString("        if: always()\n")
	yaml.WriteString("        env:\n")
	yaml.WriteString("          GH_AW_SAFE_OUTPUTS: ${{ steps.set-runtime-paths.outputs.GH_AW_SAFE_OUTPUTS }}\n")
	yaml.WriteString("        run: |\n")
	yaml.WriteString("          mkdir -p /tmp/gh-aw\n")
	fmt.Fprintf(yaml, "          cp \"$GH_AW_SAFE_OUTPUTS\" /tmp/gh-aw/%s 2>/dev/null || true\n", constants.SafeOutputsFilename)
}

func resolveAllowedDomainsStr(c *Compiler, data *WorkflowData) (string, error) {
	if data.SafeOutputs != nil && len(data.SafeOutputs.AllowedDomains) > 0 {
		return c.computeExpandedAllowedDomainsForSanitization(data)
	}
	return c.computeAllowedDomainsForSanitization(data)
}

func writeOutputCollectionEnvVars(yaml *strings.Builder, data *WorkflowData, domainsStr string) {
	yaml.WriteString("        env:\n")
	yaml.WriteString("          GH_AW_SAFE_OUTPUTS: ${{ steps.set-runtime-paths.outputs.GH_AW_SAFE_OUTPUTS }}\n")
	if domainsStr != "" {
		fmt.Fprintf(yaml, "          GH_AW_ALLOWED_DOMAINS: %q\n", domainsStr)
	}
	if data.SafeOutputs != nil && data.SafeOutputs.URLs != "" {
		fmt.Fprintf(yaml, "          GH_AW_SAFE_OUTPUTS_URLS: %q\n", data.SafeOutputs.URLs)
	}
	writeOutputCollectionOptionalEnvVars(yaml, data)
}

func writeOutputCollectionOptionalEnvVars(yaml *strings.Builder, data *WorkflowData) {
	if data.SafeOutputs != nil && data.SafeOutputs.AllowGitHubReferences != nil {
		fmt.Fprintf(yaml, "          GH_AW_ALLOWED_GITHUB_REFS: %q\n", strings.Join(data.SafeOutputs.AllowGitHubReferences, ","))
	}
	yaml.WriteString("          GITHUB_SERVER_URL: ${{ github.server_url }}\n")
	yaml.WriteString("          GITHUB_API_URL: ${{ github.api_url }}\n")
	if len(data.Command) > 0 {
		writeOutputCollectionCommandEnvVars(yaml, data)
	}
	if len(data.LabelCommand) > 0 {
		if labelCommandsJSON, err := json.Marshal(data.LabelCommand); err == nil {
			fmt.Fprintf(yaml, "          GH_AW_LABEL_COMMANDS: %q\n", string(labelCommandsJSON))
		}
	}
}

func writeOutputCollectionCommandEnvVars(yaml *strings.Builder, data *WorkflowData) {
	if commandsJSON, err := json.Marshal(data.Command); err == nil {
		fmt.Fprintf(yaml, "          GH_AW_COMMANDS: %q\n", string(commandsJSON))
	}
	if data.CommandPlaceholder != "" {
		fmt.Fprintf(yaml, "          GH_AW_COMMAND_PLACEHOLDER: %q\n", data.CommandPlaceholder)
	}
}
