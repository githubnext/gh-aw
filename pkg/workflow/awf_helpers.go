// This file provides helper functions for AWF (Agentic Workflow Firewall) integration.
//
// AWF is the network firewall/sandbox used by gh-aw to control network egress for
// AI agent execution. This file consolidates common AWF logic that was previously
// duplicated across multiple engine implementations (Copilot, Claude, Codex).
//
// # Key Functions
//
// AWF Command Building:
//   - BuildAWFCommand() - Builds complete AWF command with all arguments
//   - BuildAWFArgs() - Constructs common AWF arguments from configuration
//   - GetAWFCommandPrefix() - Determines AWF command (custom vs standard)
//   - WrapCommandInShell() - Wraps engine command in shell for AWF execution
//
// AWF Configuration:
//   - GetAWFDomains() - Combines allowed/blocked domains from various sources
//   - GetSSLBumpArgs() - Returns SSL bump configuration arguments
//   - GetAWFImageTag() - Returns pinned AWF image tag
//
// These functions extract shared AWF patterns from engine implementations,
// providing a consistent and maintainable approach to AWF integration.

package workflow

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/setutil"
	"github.com/github/gh-aw/pkg/workflow/compilerenv"
)

var awfHelpersLog = logger.New("workflow:awf_helpers")

const (
	awfDockerHostVarName       = "GH_AW_DOCKER_HOST"
	awfToolCacheMountVarName   = "GH_AW_TOOL_CACHE_MOUNT"
	awfMaxAICreditsVarName     = "GH_AW_MAX_AI_CREDITS"
	awfConfigRuntimePathExpr   = "${RUNNER_TEMP}/gh-aw/awf-config.json"
	awfModelsJSONPathExpr      = "/tmp/gh-aw/models.json"
	awfArcDindRootPathExpr     = "${RUNNER_TEMP}/gh-aw"
	awfArcDindHomePathExpr     = "${RUNNER_TEMP}/gh-aw/home"
	awfArcDindProxyLogsDirExpr = "${RUNNER_TEMP}/gh-aw/sandbox/firewall/logs"
	awfArcDindAuditDirExpr     = "${RUNNER_TEMP}/gh-aw/sandbox/firewall/audit"
	// Bash regex used in [[ ... =~ ... ]] to detect TCP Docker hosts (ARC/DinD).
	// Any tcp:// DOCKER_HOST indicates the Docker daemon runs on a separate filesystem,
	// requiring --docker-host so AWF connects to the correct daemon.
	// This covers localhost, pod IPs, K8s service names (e.g., tcp://dind:2375), and
	// any other TCP Docker daemon configuration.
	awfArcDindDockerHostRegex = `^tcp://`

	// awfArcDindChrootBinariesSourcePath is the runner-side directory that AWF overlays
	// at /usr/local/bin inside chroot mode for ARC/DinD split-filesystem runners.
	// This is the gh-aw staging directory that holds pre-downloaded binaries (e.g., copilot).
	awfArcDindChrootBinariesSourcePath = awfArcDindRootPathExpr

	// awfArcDindChrootIdentityHome is the home directory path exported inside chroot mode
	// for ARC/DinD runners. A dedicated directory under ${RUNNER_TEMP}/gh-aw is used so that the
	// runner user has a consistent home that exists on the daemon-visible filesystem.
	awfArcDindChrootIdentityHome = awfArcDindHomePathExpr

	// awfShellcheckDirective suppresses shellcheck warnings only on the generated AWF
	// invocation line:
	//   - SC1003 is expected because generated GitHub expression literals can include
	//     single quotes (for example ports['<port>']) and must survive unchanged.
	//   - SC2016 is expected because ${RUNNER_TEMP} and similar runtime variables appear
	//     inside the single-quoted bash -c '...' argument intentionally — they are expanded
	//     by the outer runner shell before AWF receives them, not by the inner bash -c.
	//   - SC2086 is expected because compiler-owned AWF argument fragments are emitted
	//     as intentional expandable shell snippets (for example ${GH_AW_TOOL_CACHE_MOUNT:+...}
	//     and ${GH_AW_DOCKER_HOST:+...}).
	//
	// User-controlled values remain quoted via shellEscapeArg/shellJoinArgs.
	awfShellcheckDirective = "# shellcheck disable=SC1003,SC2016,SC2086"
)

// AWFCommandConfig contains configuration for building AWF commands.
// This struct centralizes all the parameters needed to construct an AWF-wrapped command.
type AWFCommandConfig struct {
	// EngineName is the engine ID (e.g., "copilot", "claude", "codex")
	EngineName string

	// EngineCommand is the command to execute inside AWF
	EngineCommand string

	// LogFile is the path to the log file
	LogFile string

	// WorkflowData contains all workflow configuration
	WorkflowData *WorkflowData

	// UsesTTY indicates if the engine requires a TTY (e.g., Claude)
	UsesTTY bool

	// AllowedDomains is the comma-separated list of allowed domains
	AllowedDomains string

	// PathSetup is optional shell commands to run before the engine command
	// (e.g., npm PATH setup)
	PathSetup string

	// ExcludeEnvVarNames is the list of environment variable names to exclude from
	// the agent container's visible environment via --exclude-env. These are the env
	// var keys whose step-env values contain secret references (${{ secrets.* }}).
	// Computed from the engine's GetRequiredSecretNames() so that every secret-bearing
	// variable is excluded — the agent can never read raw token values via `env`/`printenv`.
	// Requires AWF v0.25.3+ for --exclude-env support.
	ExcludeEnvVarNames []string

	// ResolveMaxAICreditsFromEnv switches maxAiCredits runtime resolution from an inline
	// GitHub Actions expression in run: to the GH_AW_MAX_AI_CREDITS step env variable.
	// When true and max-ai-credits is unset, BuildAWFCommand emits:
	//   GH_AW_MAX_AI_CREDITS="${GH_AW_MAX_AI_CREDITS:-<default>}"
	// instead of embedding ${{ vars.* }} directly in run:.
	ResolveMaxAICreditsFromEnv bool
}

func shouldUseWorkflowCallNetworkAllowedInput(data *WorkflowData) bool {
	return data != nil &&
		data.NetworkPermissions != nil &&
		data.NetworkPermissions.AllowedInput &&
		hasWorkflowCallTrigger(data.On)
}

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

// applyDefaultMaxAICreditsEnvToMap adds the runtime max-ai-credits GitHub Actions expression
// to env when no compile-time max-ai-credits is configured.
//
// This keeps the organization/repository variable override behavior while allowing AWF run:
// scripts to read GH_AW_MAX_AI_CREDITS from step env instead of embedding ${{ vars.* }}
// directly in run blocks.
func applyDefaultMaxAICreditsEnvToMap(env map[string]string, workflowData *WorkflowData) {
	if env == nil {
		return
	}
	if workflowData != nil && workflowData.EngineConfig != nil && workflowData.EngineConfig.MaxAICredits != 0 {
		return
	}
	if workflowData != nil && workflowData.IsEvalsRun {
		env[awfMaxAICreditsVarName] = compilerenv.BuildDefaultEvalsMaxAICreditsExpression(strconv.FormatInt(constants.DefaultDetectionMaxAICredits, 10))
		return
	}
	if workflowData != nil && workflowData.IsDetectionRun {
		env[awfMaxAICreditsVarName] = compilerenv.BuildDefaultDetectionMaxAICreditsExpression(strconv.FormatInt(constants.DefaultDetectionMaxAICredits, 10))
		return
	}
	env[awfMaxAICreditsVarName] = compilerenv.BuildDefaultMaxAICreditsExpression(strconv.FormatInt(constants.DefaultMaxAICredits, 10))
}

// injectMaxAICreditsExpression inserts "maxAiCredits":expr into the apiProxy
// JSON object of awfConfigJSON directly after the "maxRuns" field value.
//
// expr is a shell variable reference such as "${GH_AW_MAX_AI_CREDITS}". The
// caller emits a local export line before the printf command that assigns the
// GitHub Actions runtime expression to that variable, so the ${{ }} expression
// lives on one clean, dedicated line rather than being embedded inside the JSON.
//
// shellEscapeArgWithVarPreserved is then used to double-quote the JSON arg while
// preserving the ${varName} reference for bash expansion and escaping bare $ signs
// (e.g. "$schema" → "\$schema").
func injectMaxAICreditsExpression(awfConfigJSON string, expr string) string {
	const maxRunsKey = `"maxRuns":`
	idx := strings.Index(awfConfigJSON, maxRunsKey)
	if idx == -1 {
		awfHelpersLog.Print("Warning: could not find maxRuns in AWF config JSON; maxAiCredits expression not injected")
		return awfConfigJSON
	}
	// Scan past the integer value of maxRuns.
	valueEnd := idx + len(maxRunsKey)
	for valueEnd < len(awfConfigJSON) && awfConfigJSON[valueEnd] >= '0' && awfConfigJSON[valueEnd] <= '9' {
		valueEnd++
	}
	return awfConfigJSON[:valueEnd] + `,"maxAiCredits":` + expr + awfConfigJSON[valueEnd:]
}

func buildWorkflowCallNetworkAllowedUpdateScript() (string, error) {
	ecosystemDomains := getLoadedEcosystemDomains()
	ecosystemMap := make(map[string][]string, safeAllocationCapacity(len(ecosystemDomains), len(compoundEcosystems)))
	for ecosystem := range ecosystemDomains {
		ecosystemMap[ecosystem] = getEcosystemDomains(ecosystem)
	}
	for ecosystem := range compoundEcosystems {
		ecosystemMap[ecosystem] = getEcosystemDomains(ecosystem)
	}

	ecosystemJSON, err := json.Marshal(ecosystemMap)
	if err != nil {
		return "", fmt.Errorf("marshal network allowed ecosystem map: %w", err)
	}

	// Pass the ecosystem map JSON via an env var and invoke the JavaScript
	// implementation deployed by actions/setup to ${RUNNER_TEMP}/gh-aw/actions/.
	// Using node avoids any Python dependency and eliminates quote-injection risk:
	// shellEscapeArg safely single-quotes and escapes the JSON payload.
	return fmt.Sprintf(`GH_AW_ECOSYSTEM_MAP_JSON=%s node "${RUNNER_TEMP}/gh-aw/actions/update_network_allowed.cjs"`,
		shellEscapeArg(string(ecosystemJSON))), nil
}

func buildArcDindProbe(config AWFCommandConfig, firewallConfig *FirewallConfig) (string, string, string) {
	prefixProbe := ""
	dockerHostProbe := fmt.Sprintf(`%s=""
if [[ "${DOCKER_HOST:-}" =~ %s ]]; then
  %s="${DOCKER_HOST}"
fi`,
		awfDockerHostVarName,
		awfArcDindDockerHostRegex,
		awfDockerHostVarName,
	)
	dockerHostRef := fmt.Sprintf("${%s:+--docker-host \"$%s\"}", awfDockerHostVarName, awfDockerHostVarName)
	if !awfSupportsDockerHostPathPrefix(firewallConfig) {
		return prefixProbe, dockerHostProbe, dockerHostRef
	}

	chrootPatchBody := ""
	if awfSupportsChrootConfig(firewallConfig) {
		if config.WorkflowData != nil && config.WorkflowData.IsDetectionRun {
			chrootPatchBody = "\n" + buildArcDindChrootConfigPatchBodyBash()
		} else {
			chrootPatchBody = "\n" + buildArcDindChrootConfigPatchBody()
		}
	}
	if chrootPatchBody != "" {
		prefixProbe = fmt.Sprintf(`if [[ "${DOCKER_HOST:-}" =~ %s ]]; then%s
fi`,
			awfArcDindDockerHostRegex,
			chrootPatchBody,
		)
	}
	if isArcDindTopology(config.WorkflowData) {
		dockerHostProbe += fmt.Sprintf("\nmkdir -p \"%s\" \"%s\"",
			awfArcDindHomePathExpr,
			awfArcDindRootPathExpr+"/sandbox/agent",
		)
		dockerHostProbe += fmt.Sprintf("\nif [ -d /tmp/gh-aw/aw-prompts ]; then cp -a /tmp/gh-aw/aw-prompts \"%s/aw-prompts\"; fi",
			awfArcDindRootPathExpr,
		)
	}

	return prefixProbe, dockerHostProbe, dockerHostRef
}

func buildToolCacheMountProbe() (string, string) {
	mountProbe := fmt.Sprintf(`%s=""
GH_AW_TOOL_CACHE="${RUNNER_TOOL_CACHE:?RUNNER_TOOL_CACHE must be set}"
if [ -d "$GH_AW_TOOL_CACHE" ]; then
  if [[ "$GH_AW_TOOL_CACHE" != /opt/* ]]; then
    %s="$GH_AW_TOOL_CACHE:$GH_AW_TOOL_CACHE:ro"
  fi
fi`,
		awfToolCacheMountVarName,
		awfToolCacheMountVarName,
	)
	mountRef := fmt.Sprintf("${%s:+--mount \"$%s\"}", awfToolCacheMountVarName, awfToolCacheMountVarName)
	return mountProbe, mountRef
}

func buildExpandableAWFArgs(isArcDind bool, workflowData *WorkflowData) string {
	ghAwDir := constants.GhAwRootDirShell
	expandableArgs := fmt.Sprintf(
		`--container-workdir "${GITHUB_WORKSPACE}" --mount "%s:%s:ro" --mount "%s:/host%s:ro"`,
		ghAwDir, ghAwDir, ghAwDir, ghAwDir,
	)
	if isArcDind {
		expandableArgs += fmt.Sprintf(
			` --mount "%s:%s:rw" --mount "%s:%s:rw"`,
			awfArcDindHomePathExpr, awfArcDindHomePathExpr,
			awfArcDindRootPathExpr+"/sandbox/agent", awfArcDindRootPathExpr+"/sandbox/agent",
		)
		expandableArgs += ` --mount "${GITHUB_WORKSPACE}:${GITHUB_WORKSPACE}:rw"`
	}
	if workflowData != nil && workflowData.SafeOutputs != nil && workflowData.SafeOutputs.UploadArtifact != nil {
		expandableArgs += fmt.Sprintf(` --mount "%s:%s:rw"`, SafeOutputsUploadArtifactsDir, SafeOutputsUploadArtifactsDir)
		awfHelpersLog.Print("Added read-write mount for upload_artifact staging directory")
	}
	if workflowData != nil && workflowData.ServicePortExpressions != "" {
		agentCfg := getAgentConfig(workflowData)
		if agentCfg != nil && agentCfg.LegacySecurity {
			expandableArgs += fmt.Sprintf(` --allow-host-service-ports "%s"`, workflowData.ServicePortExpressions)
			awfHelpersLog.Printf("Added --allow-host-service-ports with %s", workflowData.ServicePortExpressions)
		} else {
			awfHelpersLog.Print("Skipping --allow-host-service-ports: requires legacy-security mode")
		}
	}
	return expandableArgs
}

func defaultMaxAICredits(config AWFCommandConfig) string {
	defaultValue := strconv.FormatInt(constants.DefaultMaxAICredits, 10)
	if config.WorkflowData == nil {
		return defaultValue
	}
	if config.WorkflowData.IsEvalsRun || config.WorkflowData.IsDetectionRun {
		return strconv.FormatInt(constants.DefaultDetectionMaxAICredits, 10)
	}
	return defaultValue
}

func buildMaxAICreditsExportLine(config AWFCommandConfig, awfConfigJSON string) (string, string) {
	if config.WorkflowData != nil && config.WorkflowData.EngineConfig != nil && config.WorkflowData.EngineConfig.MaxAICredits != 0 {
		return awfConfigJSON, ""
	}

	defaultValue := defaultMaxAICredits(config)
	awfConfigJSON = injectMaxAICreditsExpression(awfConfigJSON, fmt.Sprintf("${%s}", awfMaxAICreditsVarName))
	if config.ResolveMaxAICreditsFromEnv {
		return awfConfigJSON, fmt.Sprintf(`%s="${%s:-%s}"`, awfMaxAICreditsVarName, awfMaxAICreditsVarName, defaultValue)
	}

	expr := compilerenv.BuildDefaultMaxAICreditsExpression(defaultValue)
	if config.WorkflowData != nil {
		switch {
		case config.WorkflowData.IsEvalsRun:
			expr = compilerenv.BuildDefaultEvalsMaxAICreditsExpression(defaultValue)
		case config.WorkflowData.IsDetectionRun:
			expr = compilerenv.BuildDefaultDetectionMaxAICreditsExpression(defaultValue)
		}
	}
	return awfConfigJSON, fmt.Sprintf(`%s="%s"`, awfMaxAICreditsVarName, expr)
}

func buildConfigPrintfArg(awfConfigJSON, maxAICreditsExportLine string) string {
	preservedVars := make([]string, 0, 2)
	if maxAICreditsExportLine != "" {
		preservedVars = append(preservedVars, awfMaxAICreditsVarName)
	}
	if strings.Contains(awfConfigJSON, awfArcDindRootPathExpr) {
		preservedVars = append(preservedVars, "RUNNER_TEMP")
	}
	if len(preservedVars) > 0 {
		return shellEscapeArgWithVarsPreserved(awfConfigJSON, preservedVars...)
	}
	return shellEscapeArg(awfConfigJSON)
}

func buildConfigFileSetup(config AWFCommandConfig, awfConfigJSON, expandableArgs string) (string, string, error) {
	awfConfigJSON, maxAICreditsExportLine := buildMaxAICreditsExportLine(config, awfConfigJSON)
	if maxAICreditsExportLine != "" {
		awfHelpersLog.Printf("Injected maxAiCredits local var reference into AWF config JSON")
	}

	printfArg := buildConfigPrintfArg(awfConfigJSON, maxAICreditsExportLine)
	printfLine := "printf '%%s\\n' %s > %q"
	if strings.HasPrefix(printfArg, "'") {
		printfLine = "# shellcheck disable=SC2016\n" + printfLine
	}
	configFileSetup := fmt.Sprintf(printfLine, printfArg, awfConfigRuntimePathExpr)
	if maxAICreditsExportLine != "" {
		configFileSetup = maxAICreditsExportLine + "\n" + configFileSetup
	}
	if shouldUseWorkflowCallNetworkAllowedInput(config.WorkflowData) {
		updateScript, err := buildWorkflowCallNetworkAllowedUpdateScript()
		if err != nil {
			awfHelpersLog.Printf("Warning: failed to build workflow_call network_allowed updater: %v", err)
		} else {
			configFileSetup += "\n" + updateScript
		}
	}
	configFileSetup += fmt.Sprintf("\ncp %q %s", awfConfigRuntimePathExpr, constants.AWFConfigFilePath)
	updatedArgs := fmt.Sprintf("--config %q ", awfConfigRuntimePathExpr) + expandableArgs
	awfHelpersLog.Print("Using AWF config file (--config flag)")
	return configFileSetup, updatedArgs, nil
}

type awfCommandScriptParts struct {
	pathSetup         string
	configFileSetup   string
	modelsJSONPath    string
	dockerHostProbe   string
	prefixProbe       string
	toolCacheMount    string
	awfCommand        string
	expandableArgs    string
	toolCacheMountRef string
	dockerHostRef     string
	awfArgs           []string
}

func buildAWFCommandScript(config AWFCommandConfig, parts awfCommandScriptParts) string {
	preCreateLog := fmt.Sprintf("(umask 177 && touch %s)", shellEscapeArg(config.LogFile))
	writeAgentCLIStartMs := "printf '%s' \"$(date +%s%3N)\" > " + shellEscapeArg(AgentCLIStartMsPath)
	engineCommand := config.EngineCommand
	if isArcDindTopology(config.WorkflowData) {
		engineCommand = rewriteArcDindEngineCommand(engineCommand)
	}

	lines := []string{"set -o pipefail", writeAgentCLIStartMs}
	if parts.pathSetup != "" {
		lines = append(lines, parts.pathSetup)
	}
	lines = append(lines, preCreateLog)
	if parts.configFileSetup != "" {
		lines = append(lines, parts.configFileSetup)
	}
	lines = append(lines, parts.modelsJSONPath, parts.dockerHostProbe, parts.prefixProbe, parts.toolCacheMount, awfShellcheckDirective)
	lines = append(lines, fmt.Sprintf(`%s %s %s %s %s \
  -- %s 2>&1 | tee -a %s`,
		parts.awfCommand,
		parts.expandableArgs,
		parts.toolCacheMountRef,
		parts.dockerHostRef,
		shellJoinArgs(parts.awfArgs),
		WrapCommandInShell(engineCommand),
		shellEscapeArg(config.LogFile),
	))
	return strings.Join(lines, "\n")
}

// BuildAWFCommand builds a complete AWF command with all arguments.
// This consolidates the AWF command building logic that was duplicated across
// Copilot, Claude, and Codex engines.
//
// Parameters:
//   - config: AWF command configuration
//
// Returns:
//   - string: Complete AWF command with arguments and wrapped engine command
func BuildAWFCommand(config AWFCommandConfig) string {
	awfHelpersLog.Printf("Building AWF command for engine: %s", config.EngineName)
	isArcDind := isArcDindTopology(config.WorkflowData)
	awfCommand := GetAWFCommandPrefix(config.WorkflowData)
	awfArgs := BuildAWFArgs(config)
	firewallConfig := getFirewallConfig(config.WorkflowData)
	arcDindPrefixProbe, arcDindDockerHostProbe, arcDindDockerHostRef := buildArcDindProbe(config, firewallConfig)
	toolCacheMountProbe, toolCacheMountRef := buildToolCacheMountProbe()
	expandableArgs := buildExpandableAWFArgs(isArcDind, config.WorkflowData)
	configFileSetup := ""
	if awfConfigJSON, err := BuildAWFConfigJSON(config); err != nil {
		awfHelpersLog.Printf("Warning: failed to build AWF config JSON: %v", err)
	} else if setup, updatedArgs, err := buildConfigFileSetup(config, awfConfigJSON, expandableArgs); err != nil {
		awfHelpersLog.Printf("Warning: failed to build AWF config setup: %v", err)
	} else {
		configFileSetup = setup
		expandableArgs = updatedArgs
	}
	command := buildAWFCommandScript(config, awfCommandScriptParts{
		pathSetup:         config.PathSetup,
		configFileSetup:   configFileSetup,
		modelsJSONPath:    buildModelsJSONPathExportScript(isArcDind),
		dockerHostProbe:   arcDindDockerHostProbe,
		prefixProbe:       arcDindPrefixProbe,
		toolCacheMount:    toolCacheMountProbe,
		awfCommand:        awfCommand,
		expandableArgs:    expandableArgs,
		toolCacheMountRef: toolCacheMountRef,
		dockerHostRef:     arcDindDockerHostRef,
		awfArgs:           awfArgs,
	})
	awfHelpersLog.Print("Successfully built AWF command")
	return command
}

func appendAWFContainerRuntimeArgs(awfArgs []string, config AWFCommandConfig, firewallConfig *FirewallConfig) []string {
	if config.UsesTTY && !isDockerSbxRuntime(config.WorkflowData) {
		awfArgs = append(awfArgs, "--tty")
	}
	if isDockerSbxRuntime(config.WorkflowData) && awfSupportsContainerRuntime(firewallConfig) {
		awfArgs = append(awfArgs, "--container-runtime", "sbx")
		awfHelpersLog.Print("Added --container-runtime sbx for docker-sbx microVM runtime")
	} else if isDockerSbxRuntime(config.WorkflowData) {
		awfHelpersLog.Printf("Skipping --container-runtime sbx: AWF version %q is older than required minimum %s", getAWFImageTag(firewallConfig), constants.AWFContainerRuntimeMinVersion)
	}
	return awfArgs
}

func appendAWFExcludeEnvArgs(awfArgs []string, excludeEnvVarNames []string, firewallConfig *FirewallConfig) []string {
	awfArgs = append(awfArgs, "--env-all")
	if !awfSupportsExcludeEnv(firewallConfig) {
		awfHelpersLog.Printf("Skipping --exclude-env: AWF version %q is older than minimum %s", getAWFImageTag(firewallConfig), constants.AWFExcludeEnvMinVersion)
		return awfArgs
	}
	sortedExclude := make([]string, len(excludeEnvVarNames))
	copy(sortedExclude, excludeEnvVarNames)
	sort.Strings(sortedExclude)
	for _, excludedVar := range sortedExclude {
		awfArgs = append(awfArgs, "--exclude-env", excludedVar)
	}
	return awfArgs
}

func appendAWFMountAndLogArgs(awfArgs []string, workflowData *WorkflowData, agentConfig *AgentSandboxConfig, firewallConfig *FirewallConfig) []string {
	if agentConfig != nil && len(agentConfig.Mounts) > 0 {
		sortedMounts := make([]string, len(agentConfig.Mounts))
		copy(sortedMounts, agentConfig.Mounts)
		sort.Strings(sortedMounts)
		for _, mount := range sortedMounts {
			awfArgs = append(awfArgs, "--mount", mount)
		}
		awfHelpersLog.Printf("Added %d custom mounts from agent config", len(sortedMounts))
	}
	awfLogLevel := string(constants.AWFDefaultLogLevel)
	if firewallConfig != nil && firewallConfig.LogLevel != "" {
		awfLogLevel = firewallConfig.LogLevel
	}
	awfArgs = append(awfArgs, "--log-level", awfLogLevel)
	if isFeatureEnabled(constants.AwfDiagnosticLogsFeatureFlag, workflowData) {
		awfArgs = append(awfArgs, "--diagnostic-logs")
		awfHelpersLog.Print("Added --diagnostic-logs because awf-diagnostic-logs feature flag is enabled")
	}
	return awfArgs
}

func appendAWFLegacySecurityArgs(awfArgs []string, config AWFCommandConfig, agentConfig *AgentSandboxConfig, firewallConfig *FirewallConfig) []string {
	if agentConfig == nil || !agentConfig.LegacySecurity {
		awfHelpersLog.Print("Strict security: skipping host-access flags (default)")
		return awfArgs
	}
	if awfSupportsLegacySecurity(firewallConfig) {
		awfArgs = append(awfArgs, "--legacy-security")
		awfHelpersLog.Print("Added --legacy-security (legacy-security: enable in frontmatter)")
	} else {
		awfHelpersLog.Printf("Skipping --legacy-security: AWF version %q is older than minimum %s (legacy mode is the default for older versions)", getAWFImageTag(firewallConfig), constants.AWFLegacySecurityMinVersion)
	}
	awfArgs = append(awfArgs, "--enable-host-access")
	awfHelpersLog.Print("Added --enable-host-access for legacy security mode")
	if awfSupportsAllowHostPorts(firewallConfig) {
		mcpGatewayPort := int(DefaultMCPGatewayPort)
		if config.WorkflowData != nil && config.WorkflowData.SandboxConfig != nil && config.WorkflowData.SandboxConfig.MCP != nil && config.WorkflowData.SandboxConfig.MCP.Port > 0 {
			mcpGatewayPort = config.WorkflowData.SandboxConfig.MCP.Port
		}
		hostPorts := fmt.Sprintf("80,443,%d", mcpGatewayPort)
		awfArgs = append(awfArgs, "--allow-host-ports", hostPorts)
		awfHelpersLog.Printf("Added --allow-host-ports %s for legacy security mode", hostPorts)
	}
	return awfArgs
}

func appendAWFCliProxyAndBasePathArgs(awfArgs []string, workflowData *WorkflowData, firewallConfig *FirewallConfig) []string {
	if isGitHubCLIModeEnabled(workflowData) {
		if awfSupportsCliProxy(firewallConfig) {
			difcProxyHost := "host.docker.internal:18443"
			if isAWFNetworkIsolationEnabled(workflowData) {
				difcProxyHost = "awmg-cli-proxy:18443"
			}
			awfArgs = append(awfArgs, "--difc-proxy-host", difcProxyHost, "--difc-proxy-ca-cert", constants.TmpDIFCProxyTLSCACert)
			awfHelpersLog.Print("Added --difc-proxy-host and --difc-proxy-ca-cert for CLI proxy sidecar")
		} else {
			awfHelpersLog.Printf("Skipping CLI proxy flags: AWF version %q is older than minimum %s", getAWFImageTag(firewallConfig), constants.AWFCliProxyMinVersion)
		}
	}
	for _, spec := range []struct{ envVar, flag string }{{"OPENAI_BASE_URL", "--openai-api-base-path"}, {"ANTHROPIC_BASE_URL", "--anthropic-api-base-path"}, {"GEMINI_API_BASE_URL", "--gemini-api-base-path"}} {
		if basePath := extractAPIBasePath(workflowData, spec.envVar); basePath != "" {
			awfArgs = append(awfArgs, spec.flag, basePath)
			awfHelpersLog.Printf("Added %s=%s", spec.flag, basePath)
		}
	}
	return awfArgs
}

func appendAWFCustomArgs(awfArgs []string, agentConfig *AgentSandboxConfig, firewallConfig *FirewallConfig) []string {
	awfArgs = append(awfArgs, getSSLBumpArgs(firewallConfig)...)
	if firewallConfig != nil && len(firewallConfig.Args) > 0 {
		awfArgs = append(awfArgs, firewallConfig.Args...)
	}
	if agentConfig != nil && len(agentConfig.Args) > 0 {
		awfArgs = append(awfArgs, agentConfig.Args...)
		awfHelpersLog.Printf("Added %d custom args from agent config", len(agentConfig.Args))
	}
	if agentConfig != nil && agentConfig.Memory != "" {
		awfArgs = append(awfArgs, "--memory-limit", agentConfig.Memory)
		awfHelpersLog.Printf("Set AWF memory limit to %s", agentConfig.Memory)
	}
	return awfArgs
}

// BuildAWFArgs constructs common AWF arguments from configuration.
// This extracts the shared AWF argument building logic from engine implementations.
//
// The following flags are expressed in the generated JSON config file written by
// BuildAWFCommand and are therefore not emitted here:
//   - --allow-domains / --block-domains   → network.allowDomains / network.blockDomains
//   - --image-tag                         → container.imageTag
//   - --openai-api-target                 → apiProxy.targets.openai.host
//   - --anthropic-api-target              → apiProxy.targets.anthropic.host
//   - --copilot-api-target                → apiProxy.targets.copilot.host
//   - --gemini-api-target                 → apiProxy.targets.gemini.host
//
// Note: --enable-api-proxy is deprecated in AWF v0.27.32+ (API proxy is always on).
// The apiProxy.enabled field is still emitted in the config file for backward compat.
//
// Parameters:
//   - config: AWF command configuration
//
// Returns:
//   - []string: List of AWF arguments (safe args only; expandable-var args like
//     --container-workdir and --mount are handled by BuildAWFCommand)
func BuildAWFArgs(config AWFCommandConfig) []string {
	awfHelpersLog.Printf("Building AWF args for engine: %s", config.EngineName)
	firewallConfig := getFirewallConfig(config.WorkflowData)
	agentConfig := getAgentConfig(config.WorkflowData)
	awfArgs := appendAWFContainerRuntimeArgs(nil, config, firewallConfig)
	awfArgs = appendAWFExcludeEnvArgs(awfArgs, config.ExcludeEnvVarNames, firewallConfig)
	awfArgs = appendAWFMountAndLogArgs(awfArgs, config.WorkflowData, agentConfig, firewallConfig)
	awfArgs = appendAWFLegacySecurityArgs(awfArgs, config, agentConfig, firewallConfig)
	awfArgs = append(awfArgs, "--skip-pull")
	awfHelpersLog.Print("Using --skip-pull since images are pre-downloaded")
	awfArgs = appendAWFCliProxyAndBasePathArgs(awfArgs, config.WorkflowData, firewallConfig)
	awfArgs = appendAWFCustomArgs(awfArgs, agentConfig, firewallConfig)
	awfHelpersLog.Printf("Built %d AWF arguments", len(awfArgs))
	return awfArgs
}

// GetAWFCommandPrefix determines the AWF command to use (custom or standard).
// This extracts the common pattern for determining AWF command from agent config.
//
// Parameters:
//   - workflowData: The workflow data containing agent configuration
//
// Returns:
//   - string: The AWF command to use (e.g., "sudo -E awf", "awf", or custom command)
func GetAWFCommandPrefix(workflowData *WorkflowData) string {
	agentConfig := getAgentConfig(workflowData)
	if agentConfig != nil && agentConfig.Command != "" {
		awfHelpersLog.Printf("Using custom AWF command: %s", agentConfig.Command)
		return agentConfig.Command
	}

	// Legacy security mode: use sudo for backward compatibility
	if agentConfig != nil && agentConfig.LegacySecurity {
		awfHelpersLog.Print("Using legacy AWF command (legacy-security: enable)")
		return string(constants.AWFLegacySecurityCommand)
	}

	// Default strict security: AWF runs rootless (no sudo)
	awfHelpersLog.Print("Using standard AWF command (strict security, no sudo)")
	return string(constants.AWFDefaultCommand)
}

// buildAWFImageTagWithDigests returns an image tag value for AWF's --image-tag flag.
// When known firewall container digests are available, it appends AWF's digest
// metadata format:
//
//	<tag>,squid=sha256:...,agent=sha256:...,api-proxy=sha256:...,cli-proxy=sha256:...
//
// For arc-dind topology, build-tools is also included:
//
//	<tag>,squid=sha256:...,agent=sha256:...,api-proxy=sha256:...,cli-proxy=sha256:...,build-tools=sha256:...
//
// This keeps AWF sidecar configuration aligned with digest-pinned pre-download images.
func buildAWFImageTagWithDigests(imageTag string, workflowData *WorkflowData) string {
	if imageTag == "" {
		return imageTag
	}

	type digestSpec struct {
		name  string
		image string
	}
	specs := []digestSpec{
		{name: "squid", image: constants.DefaultFirewallRegistry + "/squid:" + imageTag},
		{name: "agent", image: constants.DefaultFirewallRegistry + "/agent:" + imageTag},
		{name: "agent-act", image: constants.DefaultFirewallRegistry + "/agent-act:" + imageTag},
		{name: "api-proxy", image: constants.DefaultFirewallRegistry + "/api-proxy:" + imageTag},
		{name: "cli-proxy", image: constants.DefaultFirewallRegistry + "/cli-proxy:" + imageTag},
	}
	if isArcDindTopology(workflowData) {
		specs = append(specs, digestSpec{name: "build-tools", image: constants.DefaultFirewallRegistry + "/build-tools:" + imageTag})
	}

	parts := []string{imageTag}
	for _, spec := range specs {
		digest := lookupContainerDigest(spec.image, workflowData)
		if digest == "" {
			continue
		}
		parts = append(parts, spec.name+"="+digest)
	}

	if len(parts) == 1 {
		return imageTag
	}
	return strings.Join(parts, ",")
}

// lookupContainerDigest resolves a container image digest from cache first, then
// falls back to embedded container pins.
func lookupContainerDigest(image string, workflowData *WorkflowData) string {
	var cache *ActionCache
	if workflowData != nil {
		cache = workflowData.ActionCache
	}
	if pin, ok := lookupContainerPin(image, cache); ok && pin.Digest != "" {
		return pin.Digest
	}
	return ""
}

// WrapCommandInShell wraps an engine command in a shell invocation for AWF execution.
// This is needed because AWF requires commands to be wrapped in shell for proper execution.
//
// set +o histexpand disables bash history expansion so that agent-authored strings
// containing '!' characters (e.g. "!**") cannot be silently misinterpreted or dropped.
// History expansion is meaningless for non-interactive execution and has no other effect.
//
// Parameters:
//   - command: The engine command to wrap (may include PATH setup and other initialization)
//
// Returns:
//   - string: Shell-wrapped command suitable for AWF execution
func WrapCommandInShell(command string) string {
	awfHelpersLog.Print("Wrapping command in shell for AWF execution")

	// Escape single quotes in the command by replacing ' with '\''
	escapedCommand := strings.ReplaceAll(command, "'", "'\\''")

	// Wrap in shell invocation.
	// set +o histexpand is first to prevent bash from expanding !-patterns in any
	// double-quoted strings that appear in the engine command or its arguments.
	return fmt.Sprintf("/bin/bash -c 'set +o histexpand; %s'", escapedCommand)
}

// ComputeAWFExcludeEnvVarNames returns the list of environment variable names that must be
// excluded from the agent container's visible environment via AWF's --exclude-env flag.
//
// Env var names are included when their step-env values contain a ${{ secrets.* }} reference
// OR a ${{ needs.JOB.outputs.OUTPUT }} job-output expression (which commonly carries
// ephemeral tokens such as GitHub App installation tokens).  Non-secret static vars
// (e.g. GH_DEBUG: "1" in mcp-scripts) are never excluded.
//
// Parameters:
//   - workflowData: the workflow being compiled
//   - coreSecretVarNames: engine-specific fixed secret env var names (e.g. ["COPILOT_GITHUB_TOKEN"])
//
// The function augments coreSecretVarNames with:
//   - MCP_GATEWAY_API_KEY when MCP servers are present
//   - GITHUB_MCP_SERVER_TOKEN when the GitHub tool is present
//   - HTTP MCP header secret var names (values always contain ${{ secrets.* }})
//   - mcp-scripts env var names whose values contain ${{ secrets.* }} or a job-output expression
//   - engine.env var names whose values contain ${{ secrets.* }} or a job-output expression
//   - agent.env var names whose values contain ${{ secrets.* }} or a job-output expression
//   - names listed in the frontmatter excluded-env field (unconditionally)
func ComputeAWFExcludeEnvVarNames(workflowData *WorkflowData, coreSecretVarNames []string) []string {
	seen := make(map[string]struct {
	})
	var names []string

	addUnique := func(name string) {
		if !setutil.Contains(seen, name) {
			seen[name] = struct {
			}{}
			names = append(names, name)
		}
	}

	// Core secret vars for this engine (always contain secret references).
	for _, name := range coreSecretVarNames {
		addUnique(name)
	}

	// MCP gateway API key is always a secret when MCP servers are present.
	if HasMCPServers(workflowData) {
		addUnique("MCP_GATEWAY_API_KEY")
	}

	// GitHub MCP server token is always a secret when the GitHub tool is present.
	if hasGitHubTool(workflowData.ParsedTools) {
		addUnique("GITHUB_MCP_SERVER_TOKEN")
	}

	// HTTP MCP header secrets: values are always ${{ secrets.* }} references.
	for varName := range collectHTTPMCPHeaderSecrets(workflowData.Tools) {
		addUnique(varName)
	}

	// mcp-scripts env vars: only add those whose configured values contain a secret reference
	// or a job-output expression (e.g. ${{ needs.fetch_token.outputs.token }}).
	// (Non-secret vars like GH_DEBUG: "1" must NOT be excluded.)
	if workflowData.MCPScripts != nil {
		for _, toolConfig := range workflowData.MCPScripts.Tools {
			for envName, envValue := range toolConfig.Env {
				if strings.Contains(envValue, "${{ secrets.") || ContainsJobOutputExpr(envValue) {
					addUnique(envName)
				}
			}
		}
	}

	// engine.env vars that contain a secret reference or a job-output expression.
	if workflowData.EngineConfig != nil {
		for varName, varValue := range workflowData.EngineConfig.Env {
			if strings.Contains(varValue, "${{ secrets.") || ContainsJobOutputExpr(varValue) {
				addUnique(varName)
			}
		}
	}

	// agent.env vars that contain a secret reference or a job-output expression.
	agentConfig := getAgentConfig(workflowData)
	if agentConfig != nil {
		for varName, varValue := range agentConfig.Env {
			if strings.Contains(varValue, "${{ secrets.") || ContainsJobOutputExpr(varValue) {
				addUnique(varName)
			}
		}
	}

	// GH_TOKEN when GitHub mode is gh-proxy: the token is passed in the AWF step env for the
	// host difc-proxy but must be excluded from the agent container.
	if isGitHubCLIModeEnabled(workflowData) {
		addUnique("GH_TOKEN")
	}

	// Explicitly excluded env vars from the frontmatter excluded-env field.
	// These are always excluded regardless of their value content.
	for _, name := range workflowData.ExcludedEnv {
		addUnique(name)
	}

	awfHelpersLog.Printf("Computed %d AWF env vars to exclude", len(names))
	return names
}

// addCliProxyGHTokenToEnv adds GH_TOKEN to the AWF step environment when GitHub
// mode is gh-proxy. The token is NOT used by AWF or its cli-proxy
// sidecar directly — the host difc-proxy (started by start_cli_proxy.sh) already
// has it. However, --env-all passes all step env vars into the agent container,
// so we explicitly set GH_TOKEN here to ensure --exclude-env GH_TOKEN can
// reliably strip it regardless of how the token enters the environment.
// The token is excluded from the agent container via --exclude-env GH_TOKEN, so only
// inject it when the effective AWF version supports both cli-proxy flags and
// --exclude-env.
//
// #nosec G101 -- This is NOT a hardcoded credential. It is a GitHub Actions expression
// template that is resolved at runtime by the GitHub Actions runner.
func addCliProxyGHTokenToEnv(env map[string]string, workflowData *WorkflowData) {
	firewallConfig := getFirewallConfig(workflowData)
	if isGitHubCLIModeEnabled(workflowData) &&
		isFirewallEnabled(workflowData) &&
		awfSupportsCliProxy(firewallConfig) &&
		awfSupportsExcludeEnv(firewallConfig) {
		env["GH_TOKEN"] = "${{ secrets.GH_AW_GITHUB_TOKEN || github.token }}"
		awfHelpersLog.Print("Added GH_TOKEN to env for CLI proxy (excluded from agent container)")
	}
}

// awfSupportsExcludeEnv returns true when the effective AWF version supports --exclude-env
// (introduced in AWF v0.25.3).
func awfSupportsExcludeEnv(firewallConfig *FirewallConfig) bool {
	return awfVersionAtLeast(firewallConfig, constants.AWFExcludeEnvMinVersion)
}

// awfVersionAtLeast returns true when the effective AWF version is at or above minVersion.
//
// If firewallConfig has no version set, DefaultFirewallVersion is used. "latest" always
// returns true. Non-semver strings (e.g. branch names) return false (conservative).
func awfVersionAtLeast(firewallConfig *FirewallConfig, minVersion constants.Version) bool {
	var versionStr string
	if firewallConfig != nil && firewallConfig.Version != "" {
		versionStr = firewallConfig.Version
	}
	return versionAtLeast(versionStr, string(constants.DefaultFirewallVersion), string(minVersion))
}

// awfSupportsCliProxy returns true when the effective AWF version supports --difc-proxy-host
// and --difc-proxy-ca-cert (introduced in AWF v0.26.0).
func awfSupportsCliProxy(firewallConfig *FirewallConfig) bool {
	return awfVersionAtLeast(firewallConfig, constants.AWFCliProxyMinVersion)
}

// awfSupportsAllowHostPorts returns true when the effective AWF version supports
// --allow-host-ports.
func awfSupportsAllowHostPorts(firewallConfig *FirewallConfig) bool {
	return awfVersionAtLeast(firewallConfig, constants.AWFAllowHostPortsMinVersion)
}

// awfSupportsDockerHostPathPrefix returns true when the effective AWF version supports
// --docker-host-path-prefix.
func awfSupportsDockerHostPathPrefix(firewallConfig *FirewallConfig) bool {
	return awfVersionAtLeast(firewallConfig, constants.AWFDockerHostPathPrefixMinVersion)
}

// awfSupportsTokenSteering returns true when the effective AWF version supports
// apiProxy.enableTokenSteering.
func awfSupportsTokenSteering(firewallConfig *FirewallConfig) bool {
	return awfVersionAtLeast(firewallConfig, constants.AWFTokenSteeringMinVersion)
}

// awfSupportsChrootConfig returns true when the effective AWF version supports
// chroot.binariesSourcePath and chroot.identity.* in the config file (AWF v0.27.1+).
func awfSupportsChrootConfig(firewallConfig *FirewallConfig) bool {
	return awfVersionAtLeast(firewallConfig, constants.AWFChrootConfigMinVersion)
}

// awfSupportsContainerRuntime returns true when the effective AWF version supports the
// containerRuntime field in the container config (gh-aw-firewall#6093).
// The field must not be emitted for older versions that do not recognise it.
func awfSupportsContainerRuntime(firewallConfig *FirewallConfig) bool {
	return awfVersionAtLeast(firewallConfig, constants.AWFContainerRuntimeMinVersion)
}

// awfSupportsLegacySecurity returns true when the effective AWF version supports the
// --legacy-security flag (v0.27.32+). Older versions default to legacy mode and do not
// recognize this flag.
func awfSupportsLegacySecurity(firewallConfig *FirewallConfig) bool {
	return awfVersionAtLeast(firewallConfig, constants.AWFLegacySecurityMinVersion)
}

// awfSupportsAPIProxyProviders returns true when the effective AWF version supports
// apiProxy.providers in awf-config.json.
func awfSupportsAPIProxyProviders(firewallConfig *FirewallConfig) bool {
	return awfVersionAtLeast(firewallConfig, constants.AWFAPIProxyProvidersMinVersion)
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
