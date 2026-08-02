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

type awfArcDindRuntimeConfig struct {
	dockerHostProbe string
	prefixProbe     string
	dockerHostRef   string
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
	firewallConfig := getFirewallConfig(config.WorkflowData)
	arcDind := buildAWFArcDindRuntimeConfig(config, firewallConfig)
	expandableArgs, dockerHostProbe := buildAWFExpandableArgs(isArcDind, arcDind.dockerHostProbe)
	arcDind.dockerHostProbe = dockerHostProbe
	configFileSetup, expandableArgs := buildAWFConfigFileSetup(config, expandableArgs)
	expandableArgs = addAWFUploadArtifactMount(expandableArgs, config.WorkflowData)
	expandableArgs = addAWFServicePortArgs(expandableArgs, config.WorkflowData)
	command := buildCompleteAWFCommand(
		config,
		GetAWFCommandPrefix(config.WorkflowData),
		BuildAWFArgs(config),
		expandableArgs,
		configFileSetup,
		arcDind,
		buildModelsJSONPathExportScript(isArcDind),
	)
	awfHelpersLog.Print("Successfully built AWF command")
	return command
}

func buildAWFArcDindRuntimeConfig(config AWFCommandConfig, firewallConfig *FirewallConfig) awfArcDindRuntimeConfig {
	runtimeConfig := awfArcDindRuntimeConfig{
		dockerHostProbe: fmt.Sprintf(`%s=""
if [[ "${DOCKER_HOST:-}" =~ %s ]]; then
  %s="${DOCKER_HOST}"
fi`, awfDockerHostVarName, awfArcDindDockerHostRegex, awfDockerHostVarName),
		dockerHostRef: fmt.Sprintf("${%s:+--docker-host \"$%s\"}", awfDockerHostVarName, awfDockerHostVarName),
	}
	if !awfSupportsDockerHostPathPrefix(firewallConfig) {
		return runtimeConfig
	}
	chrootPatchBody := buildAWFArcDindChrootPatchBody(config.WorkflowData, firewallConfig)
	if chrootPatchBody == "" {
		return runtimeConfig
	}
	runtimeConfig.prefixProbe = fmt.Sprintf(`if [[ "${DOCKER_HOST:-}" =~ %s ]]; then%s
fi`, awfArcDindDockerHostRegex, chrootPatchBody)
	return runtimeConfig
}

func buildAWFArcDindChrootPatchBody(workflowData *WorkflowData, firewallConfig *FirewallConfig) string {
	if !awfSupportsChrootConfig(firewallConfig) {
		return ""
	}
	if workflowData != nil && workflowData.IsDetectionRun {
		return "\n" + buildArcDindChrootConfigPatchBodyBash()
	}
	return "\n" + buildArcDindChrootConfigPatchBody()
}

func buildAWFToolCacheMountSupport() (string, string) {
	probe := fmt.Sprintf(`%s=""
GH_AW_TOOL_CACHE="${RUNNER_TOOL_CACHE:?RUNNER_TOOL_CACHE must be set}"
if [ -d "$GH_AW_TOOL_CACHE" ]; then
  if [[ "$GH_AW_TOOL_CACHE" != /opt/* ]]; then
    %s="$GH_AW_TOOL_CACHE:$GH_AW_TOOL_CACHE:ro"
  fi
fi`, awfToolCacheMountVarName, awfToolCacheMountVarName)
	ref := fmt.Sprintf("${%s:+--mount \"$%s\"}", awfToolCacheMountVarName, awfToolCacheMountVarName)
	return probe, ref
}

func buildAWFExpandableArgs(isArcDind bool, dockerHostProbe string) (string, string) {
	ghAwDir := constants.GhAwRootDirShell
	expandableArgs := fmt.Sprintf(
		`--container-workdir "${GITHUB_WORKSPACE}" --mount "%s:%s:ro" --mount "%s:/host%s:ro"`,
		ghAwDir, ghAwDir, ghAwDir, ghAwDir,
	)
	if !isArcDind {
		return expandableArgs, dockerHostProbe
	}
	expandableArgs += fmt.Sprintf(
		` --mount "%s:%s:rw" --mount "%s:%s:rw"`,
		awfArcDindHomePathExpr, awfArcDindHomePathExpr,
		awfArcDindRootPathExpr+"/sandbox/agent", awfArcDindRootPathExpr+"/sandbox/agent",
	)
	expandableArgs += ` --mount "${GITHUB_WORKSPACE}:${GITHUB_WORKSPACE}:rw"`
	dockerHostProbe += fmt.Sprintf("\nmkdir -p \"%s\" \"%s\"",
		awfArcDindHomePathExpr,
		awfArcDindRootPathExpr+"/sandbox/agent",
	)
	dockerHostProbe += fmt.Sprintf("\nif [ -d /tmp/gh-aw/aw-prompts ]; then cp -a /tmp/gh-aw/aw-prompts \"%s/aw-prompts\"; fi",
		awfArcDindRootPathExpr,
	)
	return expandableArgs, dockerHostProbe
}

func buildAWFConfigFileSetup(config AWFCommandConfig, expandableArgs string) (string, string) {
	awfConfigJSON, err := BuildAWFConfigJSON(config)
	if err != nil {
		awfHelpersLog.Printf("Warning: failed to build AWF config JSON: %v", err)
		return "", expandableArgs
	}
	maxAICreditsExportLine, awfConfigJSON := buildAWFConfigRuntimeBudgetSetup(config, awfConfigJSON)
	configFileSetup := buildAWFConfigPrintfScript(awfConfigJSON, maxAICreditsExportLine)
	configFileSetup = appendAWFWorkflowCallNetworkUpdater(configFileSetup, config.WorkflowData)
	configFileSetup += fmt.Sprintf("\ncp %q %s", awfConfigRuntimePathExpr, constants.AWFConfigFilePath)
	awfHelpersLog.Print("Using AWF config file (--config flag)")
	return configFileSetup, fmt.Sprintf("--config %q ", awfConfigRuntimePathExpr) + expandableArgs
}

func buildAWFConfigRuntimeBudgetSetup(config AWFCommandConfig, awfConfigJSON string) (string, string) {
	if config.WorkflowData != nil && config.WorkflowData.EngineConfig != nil && config.WorkflowData.EngineConfig.MaxAICredits != 0 {
		return "", awfConfigJSON
	}
	defaultMaxAICredits := resolveAWFDefaultMaxAICredits(config.WorkflowData)
	awfConfigJSON = injectMaxAICreditsExpression(awfConfigJSON, fmt.Sprintf("${%s}", awfMaxAICreditsVarName))
	awfHelpersLog.Printf("Injected maxAiCredits local var reference into AWF config JSON")
	if config.ResolveMaxAICreditsFromEnv {
		return fmt.Sprintf(`%s="${%s:-%s}"`, awfMaxAICreditsVarName, awfMaxAICreditsVarName, defaultMaxAICredits), awfConfigJSON
	}
	return fmt.Sprintf(`%s="%s"`, awfMaxAICreditsVarName, buildAWFDefaultMaxAICreditsExpression(config.WorkflowData, defaultMaxAICredits)), awfConfigJSON
}

func resolveAWFDefaultMaxAICredits(workflowData *WorkflowData) string {
	switch {
	case workflowData != nil && workflowData.IsEvalsRun:
		return strconv.FormatInt(constants.DefaultDetectionMaxAICredits, 10)
	case workflowData != nil && workflowData.IsDetectionRun:
		return strconv.FormatInt(constants.DefaultDetectionMaxAICredits, 10)
	default:
		return strconv.FormatInt(constants.DefaultMaxAICredits, 10)
	}
}

func buildAWFDefaultMaxAICreditsExpression(workflowData *WorkflowData, defaultMaxAICredits string) string {
	switch {
	case workflowData != nil && workflowData.IsEvalsRun:
		return compilerenv.BuildDefaultEvalsMaxAICreditsExpression(defaultMaxAICredits)
	case workflowData != nil && workflowData.IsDetectionRun:
		return compilerenv.BuildDefaultDetectionMaxAICreditsExpression(defaultMaxAICredits)
	default:
		return compilerenv.BuildDefaultMaxAICreditsExpression(defaultMaxAICredits)
	}
}

func buildAWFConfigPrintfScript(awfConfigJSON string, maxAICreditsExportLine string) string {
	printfArg := buildAWFConfigPrintfArg(awfConfigJSON, maxAICreditsExportLine != "")
	printfLine := "printf '%%s\\n' %s > %q"
	if strings.HasPrefix(printfArg, "'") {
		printfLine = "# shellcheck disable=SC2016\nprintf '%%s\\n' %s > %q"
	}
	configFileSetup := fmt.Sprintf(printfLine, printfArg, awfConfigRuntimePathExpr)
	if maxAICreditsExportLine == "" {
		return configFileSetup
	}
	return maxAICreditsExportLine + "\n" + configFileSetup
}

func buildAWFConfigPrintfArg(awfConfigJSON string, hasRuntimeBudget bool) string {
	preservedVars := make([]string, 0, 2)
	if hasRuntimeBudget {
		preservedVars = append(preservedVars, awfMaxAICreditsVarName)
	}
	if strings.Contains(awfConfigJSON, awfArcDindRootPathExpr) {
		preservedVars = append(preservedVars, "RUNNER_TEMP")
	}
	if len(preservedVars) == 0 {
		return shellEscapeArg(awfConfigJSON)
	}
	return shellEscapeArgWithVarsPreserved(awfConfigJSON, preservedVars...)
}

func appendAWFWorkflowCallNetworkUpdater(configFileSetup string, workflowData *WorkflowData) string {
	if !shouldUseWorkflowCallNetworkAllowedInput(workflowData) {
		return configFileSetup
	}
	updateScript, err := buildWorkflowCallNetworkAllowedUpdateScript()
	if err != nil {
		awfHelpersLog.Printf("Warning: failed to build workflow_call network_allowed updater: %v", err)
		return configFileSetup
	}
	return configFileSetup + "\n" + updateScript
}

func addAWFUploadArtifactMount(expandableArgs string, workflowData *WorkflowData) string {
	if workflowData == nil || workflowData.SafeOutputs == nil || workflowData.SafeOutputs.UploadArtifact == nil {
		return expandableArgs
	}
	stagingDir := SafeOutputsUploadArtifactsDir
	awfHelpersLog.Print("Added read-write mount for upload_artifact staging directory")
	return expandableArgs + fmt.Sprintf(` --mount "%s:%s:rw"`, stagingDir, stagingDir)
}

func addAWFServicePortArgs(expandableArgs string, workflowData *WorkflowData) string {
	if workflowData == nil || workflowData.ServicePortExpressions == "" {
		return expandableArgs
	}
	agentCfg := getAgentConfig(workflowData)
	if agentCfg != nil && agentCfg.LegacySecurity {
		awfHelpersLog.Printf("Added --allow-host-service-ports with %s", workflowData.ServicePortExpressions)
		return expandableArgs + fmt.Sprintf(` --allow-host-service-ports "%s"`, workflowData.ServicePortExpressions)
	}
	awfHelpersLog.Print("Skipping --allow-host-service-ports: requires legacy-security mode")
	return expandableArgs
}

func buildAWFEngineCommand(engineCommand string, isArcDind bool) string {
	if isArcDind {
		return rewriteArcDindEngineCommand(engineCommand)
	}
	return engineCommand
}

func buildCompleteAWFCommand(
	config AWFCommandConfig,
	awfCommand string,
	awfArgs []string,
	expandableArgs string,
	configFileSetup string,
	arcDind awfArcDindRuntimeConfig,
	modelsJSONPathExport string,
) string {
	toolCacheMountProbe, toolCacheMountRef := buildAWFToolCacheMountSupport()
	shellWrappedCommand := WrapCommandInShell(buildAWFEngineCommand(config.EngineCommand, isArcDindTopology(config.WorkflowData)))
	preCreateLog := fmt.Sprintf("(umask 177 && touch %s)", shellEscapeArg(config.LogFile))
	writeAgentCLIStartMs := "printf '%s' \"$(date +%s%3N)\" > " + shellEscapeArg(AgentCLIStartMsPath)
	joinedArgs := shellJoinArgs(awfArgs)
	logFileArg := shellEscapeArg(config.LogFile)
	switch {
	case config.PathSetup != "" && configFileSetup != "":
		return formatAWFCommandWithPathSetupAndConfig(writeAgentCLIStartMs, config, preCreateLog, configFileSetup, modelsJSONPathExport, arcDind, toolCacheMountProbe, awfCommand, expandableArgs, toolCacheMountRef, joinedArgs, shellWrappedCommand, logFileArg)
	case config.PathSetup != "":
		return formatAWFCommandWithPathSetup(writeAgentCLIStartMs, config, preCreateLog, modelsJSONPathExport, arcDind, toolCacheMountProbe, awfCommand, expandableArgs, toolCacheMountRef, joinedArgs, shellWrappedCommand, logFileArg)
	case configFileSetup != "":
		return formatAWFCommandWithConfig(writeAgentCLIStartMs, preCreateLog, configFileSetup, modelsJSONPathExport, arcDind, toolCacheMountProbe, awfCommand, expandableArgs, toolCacheMountRef, joinedArgs, shellWrappedCommand, logFileArg)
	default:
		return formatAWFCommand(writeAgentCLIStartMs, preCreateLog, modelsJSONPathExport, arcDind, toolCacheMountProbe, awfCommand, expandableArgs, toolCacheMountRef, joinedArgs, shellWrappedCommand, logFileArg)
	}
}

func formatAWFCommandWithPathSetupAndConfig(writeAgentCLIStartMs string, config AWFCommandConfig, preCreateLog string, configFileSetup string, modelsJSONPathExport string, arcDind awfArcDindRuntimeConfig, toolCacheMountProbe string, awfCommand string, expandableArgs string, toolCacheMountRef string, joinedArgs string, shellWrappedCommand string, logFileArg string) string {
	return fmt.Sprintf(`set -o pipefail
%s
%s
%s
%s
%s
%s
%s
%s
%s
%s %s %s %s %s \
  -- %s 2>&1 | tee -a %s`,
		writeAgentCLIStartMs, config.PathSetup, preCreateLog, configFileSetup, modelsJSONPathExport,
		arcDind.dockerHostProbe, arcDind.prefixProbe, toolCacheMountProbe, awfShellcheckDirective,
		awfCommand, expandableArgs, toolCacheMountRef, arcDind.dockerHostRef, joinedArgs, shellWrappedCommand, logFileArg)
}

func formatAWFCommandWithPathSetup(writeAgentCLIStartMs string, config AWFCommandConfig, preCreateLog string, modelsJSONPathExport string, arcDind awfArcDindRuntimeConfig, toolCacheMountProbe string, awfCommand string, expandableArgs string, toolCacheMountRef string, joinedArgs string, shellWrappedCommand string, logFileArg string) string {
	return fmt.Sprintf(`set -o pipefail
%s
%s
%s
%s
%s
%s
%s
%s
%s %s %s %s %s \
  -- %s 2>&1 | tee -a %s`,
		writeAgentCLIStartMs, config.PathSetup, preCreateLog, modelsJSONPathExport, arcDind.dockerHostProbe,
		arcDind.prefixProbe, toolCacheMountProbe, awfShellcheckDirective, awfCommand, expandableArgs,
		toolCacheMountRef, arcDind.dockerHostRef, joinedArgs, shellWrappedCommand, logFileArg)
}

func formatAWFCommandWithConfig(writeAgentCLIStartMs string, preCreateLog string, configFileSetup string, modelsJSONPathExport string, arcDind awfArcDindRuntimeConfig, toolCacheMountProbe string, awfCommand string, expandableArgs string, toolCacheMountRef string, joinedArgs string, shellWrappedCommand string, logFileArg string) string {
	return fmt.Sprintf(`set -o pipefail
%s
%s
%s
%s
%s
%s
%s
%s
%s %s %s %s %s \
  -- %s 2>&1 | tee -a %s`,
		writeAgentCLIStartMs, preCreateLog, configFileSetup, modelsJSONPathExport, arcDind.dockerHostProbe,
		arcDind.prefixProbe, toolCacheMountProbe, awfShellcheckDirective, awfCommand, expandableArgs,
		toolCacheMountRef, arcDind.dockerHostRef, joinedArgs, shellWrappedCommand, logFileArg)
}

func formatAWFCommand(writeAgentCLIStartMs string, preCreateLog string, modelsJSONPathExport string, arcDind awfArcDindRuntimeConfig, toolCacheMountProbe string, awfCommand string, expandableArgs string, toolCacheMountRef string, joinedArgs string, shellWrappedCommand string, logFileArg string) string {
	return fmt.Sprintf(`set -o pipefail
%s
%s
%s
%s
%s
%s
%s
%s %s %s %s %s \
  -- %s 2>&1 | tee -a %s`,
		writeAgentCLIStartMs, preCreateLog, modelsJSONPathExport, arcDind.dockerHostProbe,
		arcDind.prefixProbe, toolCacheMountProbe, awfShellcheckDirective, awfCommand, expandableArgs,
		toolCacheMountRef, arcDind.dockerHostRef, joinedArgs, shellWrappedCommand, logFileArg)
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
	awfArgs := appendAWFBaseArgs(nil, config, firewallConfig)
	awfArgs = appendAWFEnvArgs(awfArgs, config, firewallConfig)
	awfArgs = appendAWFMountArgs(awfArgs, agentConfig)
	awfArgs = appendAWFLoggingArgs(awfArgs, config, firewallConfig)
	awfArgs = appendAWFLegacySecurityArgs(awfArgs, config, firewallConfig, agentConfig)
	awfArgs = appendAWFSkipPullAndCLIProxyArgs(awfArgs, config, firewallConfig)
	awfArgs = appendAWFAPIBasePathArgs(awfArgs, config.WorkflowData)
	awfArgs = append(awfArgs, getSSLBumpArgs(firewallConfig)...)
	awfArgs = appendAWFCustomArgs(awfArgs, firewallConfig, agentConfig)
	awfHelpersLog.Printf("Built %d AWF arguments", len(awfArgs))
	return awfArgs
}

func appendAWFBaseArgs(awfArgs []string, config AWFCommandConfig, firewallConfig *FirewallConfig) []string {
	if config.UsesTTY && !isDockerSbxRuntime(config.WorkflowData) {
		awfArgs = append(awfArgs, "--tty")
	}
	if !isDockerSbxRuntime(config.WorkflowData) {
		return awfArgs
	}
	if awfSupportsContainerRuntime(firewallConfig) {
		awfHelpersLog.Print("Added --container-runtime sbx for docker-sbx microVM runtime")
		return append(awfArgs, "--container-runtime", "sbx")
	}
	awfHelpersLog.Printf("Skipping --container-runtime sbx: AWF version %q is older than required minimum %s", getAWFImageTag(firewallConfig), constants.AWFContainerRuntimeMinVersion)
	return awfArgs
}

func appendAWFEnvArgs(awfArgs []string, config AWFCommandConfig, firewallConfig *FirewallConfig) []string {
	awfArgs = append(awfArgs, "--env-all")
	if !awfSupportsExcludeEnv(firewallConfig) {
		awfHelpersLog.Printf("Skipping --exclude-env: AWF version %q is older than minimum %s", getAWFImageTag(firewallConfig), constants.AWFExcludeEnvMinVersion)
		return awfArgs
	}
	sortedExclude := append([]string(nil), config.ExcludeEnvVarNames...)
	sort.Strings(sortedExclude)
	for _, excludedVar := range sortedExclude {
		awfArgs = append(awfArgs, "--exclude-env", excludedVar)
	}
	return awfArgs
}

func appendAWFMountArgs(awfArgs []string, agentConfig *AgentSandboxConfig) []string {
	if agentConfig == nil || len(agentConfig.Mounts) == 0 {
		return awfArgs
	}
	sortedMounts := append([]string(nil), agentConfig.Mounts...)
	sort.Strings(sortedMounts)
	for _, mount := range sortedMounts {
		awfArgs = append(awfArgs, "--mount", mount)
	}
	awfHelpersLog.Printf("Added %d custom mounts from agent config", len(sortedMounts))
	return awfArgs
}

func appendAWFLoggingArgs(awfArgs []string, config AWFCommandConfig, firewallConfig *FirewallConfig) []string {
	awfLogLevel := string(constants.AWFDefaultLogLevel)
	if firewallConfig != nil && firewallConfig.LogLevel != "" {
		awfLogLevel = firewallConfig.LogLevel
	}
	awfArgs = append(awfArgs, "--log-level", awfLogLevel)
	if isFeatureEnabled(constants.AwfDiagnosticLogsFeatureFlag, config.WorkflowData) {
		awfArgs = append(awfArgs, "--diagnostic-logs")
		awfHelpersLog.Print("Added --diagnostic-logs because awf-diagnostic-logs feature flag is enabled")
	}
	return awfArgs
}

func appendAWFLegacySecurityArgs(awfArgs []string, config AWFCommandConfig, firewallConfig *FirewallConfig, agentConfig *AgentSandboxConfig) []string {
	if agentConfig == nil || !agentConfig.LegacySecurity {
		awfHelpersLog.Print("Strict security: skipping host-access flags (default)")
		return awfArgs
	}
	awfArgs = appendAWFLegacySecurityModeArg(awfArgs, firewallConfig)
	awfArgs = append(awfArgs, "--enable-host-access")
	awfHelpersLog.Print("Added --enable-host-access for legacy security mode")
	return appendAWFAllowHostPortsArg(awfArgs, config.WorkflowData, firewallConfig)
}

func appendAWFLegacySecurityModeArg(awfArgs []string, firewallConfig *FirewallConfig) []string {
	if awfSupportsLegacySecurity(firewallConfig) {
		awfHelpersLog.Print("Added --legacy-security (legacy-security: enable in frontmatter)")
		return append(awfArgs, "--legacy-security")
	}
	awfHelpersLog.Printf("Skipping --legacy-security: AWF version %q is older than minimum %s (legacy mode is the default for older versions)", getAWFImageTag(firewallConfig), constants.AWFLegacySecurityMinVersion)
	return awfArgs
}

func appendAWFAllowHostPortsArg(awfArgs []string, workflowData *WorkflowData, firewallConfig *FirewallConfig) []string {
	if !awfSupportsAllowHostPorts(firewallConfig) {
		return awfArgs
	}
	mcpGatewayPort := int(DefaultMCPGatewayPort)
	if workflowData != nil && workflowData.SandboxConfig != nil && workflowData.SandboxConfig.MCP != nil && workflowData.SandboxConfig.MCP.Port > 0 {
		mcpGatewayPort = workflowData.SandboxConfig.MCP.Port
	}
	hostPorts := fmt.Sprintf("80,443,%d", mcpGatewayPort)
	awfHelpersLog.Printf("Added --allow-host-ports %s for legacy security mode", hostPorts)
	return append(awfArgs, "--allow-host-ports", hostPorts)
}

func appendAWFSkipPullAndCLIProxyArgs(awfArgs []string, config AWFCommandConfig, firewallConfig *FirewallConfig) []string {
	awfArgs = append(awfArgs, "--skip-pull")
	awfHelpersLog.Print("Using --skip-pull since images are pre-downloaded")
	if !isGitHubCLIModeEnabled(config.WorkflowData) {
		return awfArgs
	}
	if !awfSupportsCliProxy(firewallConfig) {
		awfHelpersLog.Printf("Skipping CLI proxy flags: AWF version %q is older than minimum %s", getAWFImageTag(firewallConfig), constants.AWFCliProxyMinVersion)
		return awfArgs
	}
	difcProxyHost := "host.docker.internal:18443"
	if isAWFNetworkIsolationEnabled(config.WorkflowData) {
		difcProxyHost = "awmg-cli-proxy:18443"
	}
	awfHelpersLog.Print("Added --difc-proxy-host and --difc-proxy-ca-cert for CLI proxy sidecar")
	return append(awfArgs, "--difc-proxy-host", difcProxyHost, "--difc-proxy-ca-cert", constants.TmpDIFCProxyTLSCACert)
}

func appendAWFAPIBasePathArgs(awfArgs []string, workflowData *WorkflowData) []string {
	awfArgs = appendAWFAPIBasePathArg(awfArgs, workflowData, "OPENAI_BASE_URL", "--openai-api-base-path")
	awfArgs = appendAWFAPIBasePathArg(awfArgs, workflowData, "ANTHROPIC_BASE_URL", "--anthropic-api-base-path")
	return appendAWFAPIBasePathArg(awfArgs, workflowData, "GEMINI_API_BASE_URL", "--gemini-api-base-path")
}

func appendAWFAPIBasePathArg(awfArgs []string, workflowData *WorkflowData, envVar string, flagName string) []string {
	basePath := extractAPIBasePath(workflowData, envVar)
	if basePath == "" {
		return awfArgs
	}
	awfHelpersLog.Printf("Added %s=%s", flagName, basePath)
	return append(awfArgs, flagName, basePath)
}

func appendAWFCustomArgs(awfArgs []string, firewallConfig *FirewallConfig, agentConfig *AgentSandboxConfig) []string {
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
	collector := newAWFExcludeEnvCollector()
	collector.addAll(coreSecretVarNames)
	collector.addRuntimeSecrets(workflowData)
	collector.addMCPScriptEnv(workflowData)
	collector.addWorkflowEnv(workflowData)
	if isGitHubCLIModeEnabled(workflowData) {
		collector.add("GH_TOKEN")
	}
	if workflowData != nil {
		collector.addAll(workflowData.ExcludedEnv)
	}
	awfHelpersLog.Printf("Computed %d AWF env vars to exclude", len(collector.names))
	return collector.names
}

type awfExcludeEnvCollector struct {
	seen  map[string]struct{}
	names []string
}

func newAWFExcludeEnvCollector() *awfExcludeEnvCollector {
	return &awfExcludeEnvCollector{seen: make(map[string]struct{})}
}

func (c *awfExcludeEnvCollector) add(name string) {
	if !setutil.Contains(c.seen, name) {
		c.seen[name] = struct{}{}
		c.names = append(c.names, name)
	}
}

func (c *awfExcludeEnvCollector) addAll(names []string) {
	for _, name := range names {
		c.add(name)
	}
}

func (c *awfExcludeEnvCollector) addSecretLikeEnv(env map[string]string) {
	for envName, envValue := range env {
		if strings.Contains(envValue, "${{ secrets.") || ContainsJobOutputExpr(envValue) {
			c.add(envName)
		}
	}
}

func (c *awfExcludeEnvCollector) addRuntimeSecrets(workflowData *WorkflowData) {
	if workflowData == nil {
		return
	}
	if HasMCPServers(workflowData) {
		c.add("MCP_GATEWAY_API_KEY")
	}
	if hasGitHubTool(workflowData.ParsedTools) {
		c.add("GITHUB_MCP_SERVER_TOKEN")
	}
	for varName := range collectHTTPMCPHeaderSecrets(workflowData.Tools) {
		c.add(varName)
	}
}

func (c *awfExcludeEnvCollector) addMCPScriptEnv(workflowData *WorkflowData) {
	if workflowData == nil || workflowData.MCPScripts == nil {
		return
	}
	for _, toolConfig := range workflowData.MCPScripts.Tools {
		c.addSecretLikeEnv(toolConfig.Env)
	}
}

func (c *awfExcludeEnvCollector) addWorkflowEnv(workflowData *WorkflowData) {
	if workflowData == nil {
		return
	}
	if workflowData.EngineConfig != nil {
		c.addSecretLikeEnv(workflowData.EngineConfig.Env)
	}
	if agentConfig := getAgentConfig(workflowData); agentConfig != nil {
		c.addSecretLikeEnv(agentConfig.Env)
	}
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

// awfSupportsBoundedQueries returns true when the effective AWF version supports
// the boundedQueries section in awf-config.json.
func awfSupportsBoundedQueries(firewallConfig *FirewallConfig) bool {
	return awfVersionAtLeast(firewallConfig, constants.AWFBoundedQueriesMinVersion)
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
