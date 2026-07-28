package workflow

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/workflow/compilerenv"
)

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

	// Get AWF command prefix (custom or standard)
	awfCommand := GetAWFCommandPrefix(config.WorkflowData)

	// Build AWF arguments. The returned list contains only args that are safe to pass
	// through shellJoinArgs. Expandable-var args (--container-workdir "${GITHUB_WORKSPACE}"
	// and --mount "${RUNNER_TEMP}/...") are appended raw below so that shell variable
	// expansion is not suppressed by single-quoting.
	awfArgs := BuildAWFArgs(config)
	firewallConfig := getFirewallConfig(config.WorkflowData)

	// Auto-detect ARC/DinD split daemon topology at runtime: probe DOCKER_HOST for a
	// tcp:// scheme and pass it through to AWF via --docker-host.
	// All behaviors avoid requiring workflow-authored sandbox.agent.args for standard ARC DinD setups.
	// When AWF also supports chroot config (v0.27.1+), the Python patch body is embedded inside
	// the same if-block so the script only contains one DOCKER_HOST condition check.
	arcDindPrefixProbe := ""
	arcDindDockerHostProbe := fmt.Sprintf(`%s=""
if [[ "${DOCKER_HOST:-}" =~ %s ]]; then
  %s="${DOCKER_HOST}"
fi`,
		awfDockerHostVarName,
		awfArcDindDockerHostRegex,
		awfDockerHostVarName,
	)
	arcDindDockerHostRef := fmt.Sprintf("${%s:+--docker-host \"$%s\"}", awfDockerHostVarName, awfDockerHostVarName)
	if awfSupportsDockerHostPathPrefix(firewallConfig) {
		chrootPatchBody := ""
		if awfSupportsChrootConfig(firewallConfig) {
			if config.WorkflowData != nil && config.WorkflowData.IsDetectionRun {
				chrootPatchBody = "\n" + buildArcDindChrootConfigPatchBodyBash()
			} else {
				chrootPatchBody = "\n" + buildArcDindChrootConfigPatchBody()
			}
		}
		// NOTE: --docker-host-path-prefix is intentionally NOT passed. With sysroot-stage
		// active, all bind-mount source paths are on the shared work volume and visible to
		// the Docker daemon without translation. The prefix caused AWF to translate
		// GITHUB_WORKSPACE to a non-existent path, resulting in an empty workspace (gh-aw#34896).
		// The probe block is preserved for the chroot config patch which still requires the
		// DOCKER_HOST guard.
		if chrootPatchBody != "" {
			arcDindPrefixProbe = fmt.Sprintf(`if [[ "${DOCKER_HOST:-}" =~ %s ]]; then%s
fi`,
				awfArcDindDockerHostRegex,
				chrootPatchBody)
		}
	}
	toolCacheMountProbe := fmt.Sprintf(`%s=""
GH_AW_TOOL_CACHE="${RUNNER_TOOL_CACHE:?RUNNER_TOOL_CACHE must be set}"
if [ -d "$GH_AW_TOOL_CACHE" ]; then
  if [[ "$GH_AW_TOOL_CACHE" != /opt/* ]]; then
    %s="$GH_AW_TOOL_CACHE:$GH_AW_TOOL_CACHE:ro"
  fi
fi`,
		awfToolCacheMountVarName,
		awfToolCacheMountVarName,
	)
	toolCacheMountRef := fmt.Sprintf("${%s:+--mount \"$%s\"}", awfToolCacheMountVarName, awfToolCacheMountVarName)

	// Build the expandable args string for args that need shell variable expansion.
	// These MUST be appended as raw (unescaped) strings because single-quoting would
	// prevent the runner's shell from expanding ${GITHUB_WORKSPACE} and ${RUNNER_TEMP}.
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
		// Explicitly mount the workspace so AWF can see it without path-prefix translation.
		// GITHUB_WORKSPACE is on the shared work volume, so the Docker daemon can access it.
		expandableArgs += ` --mount "${GITHUB_WORKSPACE}:${GITHUB_WORKSPACE}:rw"`
		// Pre-create the rw mount source directories. AWF validates that mount source
		// paths exist before starting containers, so these must be created on the host
		// before the AWF invocation. The parent ${RUNNER_TEMP}/gh-aw/ already exists
		// (created by actions/setup), but the subdirectories may not.
		arcDindDockerHostProbe += fmt.Sprintf("\nmkdir -p \"%s\" \"%s\"",
			awfArcDindHomePathExpr,
			awfArcDindRootPathExpr+"/sandbox/agent",
		)
		// Copy prompt files to daemon-visible path. On ARC/DinD, /tmp/gh-aw/ is NOT
		// accessible to the Docker daemon. The activation job writes prompts to
		// /tmp/gh-aw/aw-prompts/, so we copy them to ${RUNNER_TEMP}/gh-aw/aw-prompts/.
		arcDindDockerHostProbe += fmt.Sprintf("\nif [ -d /tmp/gh-aw/aw-prompts ]; then cp -a /tmp/gh-aw/aw-prompts \"%s/aw-prompts\"; fi",
			awfArcDindRootPathExpr,
		)
	}

	// Generate a JSON config file and reference it via --config "${RUNNER_TEMP}/gh-aw/awf-config.json".
	// This replaces several verbose CLI flags (--allow-domains, --enable-api-proxy, --image-tag,
	// API targets) with a structured JSON file that is easier to audit and extend.
	//
	// The config file is written at runtime (inside the run: step) immediately before the AWF
	// invocation, using printf to a fixed path inside the pre-existing ${RUNNER_TEMP}/gh-aw/
	// directory that is already set up by actions/setup.
	var configFileSetup string
	awfConfigJSON, err := BuildAWFConfigJSON(config)
	if err != nil {
		awfHelpersLog.Printf("Warning: failed to build AWF config JSON: %v", err)
	} else {
		// When max-ai-credits is not set by frontmatter/imports, export a local shell
		// variable (GH_AW_MAX_AI_CREDITS) holding a GitHub Actions runtime expression,
		// then inject a reference to that variable (${GH_AW_MAX_AI_CREDITS}) into the
		// "maxAiCredits" field of the apiProxy JSON object. GitHub Actions evaluates
		// the ${{ }} expression before the shell runs, so the variable is set to the
		// resolved integer by the time printf writes the config file.
		//
		// Standard agent runs use vars.GH_AW_DEFAULT_MAX_AI_CREDITS with built-in
		// fallback 1000. Threat-detection runs use
		// vars.GH_AW_DEFAULT_DETECTION_MAX_AI_CREDITS with built-in fallback 400.
		// Evals runs use vars.GH_AW_DEFAULT_EVALS_MAX_AI_CREDITS with built-in
		// fallback 400 to align with detection budgets.
		// EngineConfig.MaxAICredits is 0 when no compile-time value was set
		// (neither frontmatter nor detection-engine config provided one).
		// In that case, emit a runtime expression that lets the org variable
		// or the built-in default resolve the budget at action run time.
		// For detection runs, use the detection-specific variable/fallback;
		// for standard agent runs, use the main-agent variable/fallback.
		var maxAICreditsExportLine string
		if config.WorkflowData == nil || config.WorkflowData.EngineConfig == nil || config.WorkflowData.EngineConfig.MaxAICredits == 0 {
			defaultMaxAICredits := strconv.FormatInt(constants.DefaultMaxAICredits, 10)
			if config.WorkflowData != nil {
				switch {
				case config.WorkflowData.IsEvalsRun:
					defaultMaxAICredits = strconv.FormatInt(constants.DefaultDetectionMaxAICredits, 10)
				case config.WorkflowData.IsDetectionRun:
					defaultMaxAICredits = strconv.FormatInt(constants.DefaultDetectionMaxAICredits, 10)
				}
			}
			awfConfigJSON = injectMaxAICreditsExpression(awfConfigJSON, fmt.Sprintf("${%s}", awfMaxAICreditsVarName))
			if config.ResolveMaxAICreditsFromEnv {
				maxAICreditsExportLine = fmt.Sprintf(`%s="${%s:-%s}"`, awfMaxAICreditsVarName, awfMaxAICreditsVarName, defaultMaxAICredits)
			} else {
				expr := compilerenv.BuildDefaultMaxAICreditsExpression(defaultMaxAICredits)
				if config.WorkflowData != nil {
					switch {
					case config.WorkflowData.IsEvalsRun:
						expr = compilerenv.BuildDefaultEvalsMaxAICreditsExpression(defaultMaxAICredits)
					case config.WorkflowData.IsDetectionRun:
						expr = compilerenv.BuildDefaultDetectionMaxAICreditsExpression(defaultMaxAICredits)
					}
				}
				maxAICreditsExportLine = fmt.Sprintf(`%s="%s"`, awfMaxAICreditsVarName, expr)
			}
			awfHelpersLog.Printf("Injected maxAiCredits local var reference into AWF config JSON")
		}
		// Write the config JSON to ${RUNNER_TEMP}/gh-aw/awf-config.json before AWF runs.
		// When the generated JSON contains compiler-owned runtime variables such as
		// ${GH_AW_MAX_AI_CREDITS} or ${RUNNER_TEMP}, use shellEscapeArgWithVarsPreserved
		// which always uses double-quote wrapping: it escapes bare $ signs (e.g.
		// "$schema" → "\$schema") while preserving both ${{ }} GitHub Actions expressions
		// (e.g. in AllowedDomains) and approved shell variable references so bash expands
		// them to runtime-resolved values. When no such variables are injected,
		// shellEscapeArg handles escaping normally.
		// Also copy it to /tmp/gh-aw/awf-config.json for the unified agent artifact upload.
		var printfArg string
		preservedVars := make([]string, 0, 2)
		if maxAICreditsExportLine != "" {
			preservedVars = append(preservedVars, awfMaxAICreditsVarName)
		}
		if strings.Contains(awfConfigJSON, awfArcDindRootPathExpr) {
			preservedVars = append(preservedVars, "RUNNER_TEMP")
		}
		if len(preservedVars) > 0 {
			printfArg = shellEscapeArgWithVarsPreserved(awfConfigJSON, preservedVars...)
		} else {
			printfArg = shellEscapeArg(awfConfigJSON)
		}
		// SC2016 ("Expressions don't expand in single quotes") is only triggered when
		// printfArg is single-quoted (no runtime variables injected). Double-quoted args
		// already escape bare $ signs as \$schema, so shellcheck does not warn there.
		var printfLine string
		if strings.HasPrefix(printfArg, "'") {
			printfLine = "# shellcheck disable=SC2016\nprintf '%%s\\n' %s > %q"
		} else {
			printfLine = "printf '%%s\\n' %s > %q"
		}
		configFileSetup = fmt.Sprintf(printfLine, printfArg, awfConfigRuntimePathExpr)
		if maxAICreditsExportLine != "" {
			configFileSetup = maxAICreditsExportLine + "\n" + configFileSetup
		}
		if shouldUseWorkflowCallNetworkAllowedInput(config.WorkflowData) {
			updateScript, updateErr := buildWorkflowCallNetworkAllowedUpdateScript()
			if updateErr != nil {
				awfHelpersLog.Printf("Warning: failed to build workflow_call network_allowed updater: %v", updateErr)
			} else {
				configFileSetup += "\n" + updateScript
			}
		}
		configFileSetup += fmt.Sprintf("\ncp %q %s", awfConfigRuntimePathExpr, constants.AWFConfigFilePath)
		// Add --config as the first expandable arg so it appears before --container-workdir.
		expandableArgs = fmt.Sprintf("--config %q ", awfConfigRuntimePathExpr) + expandableArgs
		awfHelpersLog.Print("Using AWF config file (--config flag)")
	}
	modelsJSONPathExport := buildModelsJSONPathExportScript(isArcDind)

	// When upload_artifact is configured, add a read-write mount for the staging directory
	// so the model can copy files there from inside the container. The parent ${RUNNER_TEMP}/gh-aw
	// is mounted :ro above; this child mount overrides access for the staging subdirectory only.
	// The staging directory must already exist on the host (created in Generate Safe Outputs Config step).
	if config.WorkflowData != nil && config.WorkflowData.SafeOutputs != nil && config.WorkflowData.SafeOutputs.UploadArtifact != nil {
		stagingDir := SafeOutputsUploadArtifactsDir
		expandableArgs += fmt.Sprintf(` --mount "%s:%s:rw"`, stagingDir, stagingDir)
		awfHelpersLog.Print("Added read-write mount for upload_artifact staging directory")
	}

	// Add --allow-host-service-ports for services with port mappings.
	// This flag requires --legacy-security since it grants host network access.
	// This is appended as a raw (expandable) arg because the value contains
	// ${{ job.services.<id>.ports['<port>'] }} expressions that include single quotes.
	// These expressions are resolved by the GitHub Actions runner before shell execution,
	// so they must not be shell-escaped.
	agentCfg := getAgentConfig(config.WorkflowData)
	isLegacyMode := agentCfg != nil && agentCfg.LegacySecurity
	if config.WorkflowData != nil && config.WorkflowData.ServicePortExpressions != "" && isLegacyMode {
		expandableArgs += fmt.Sprintf(` --allow-host-service-ports "%s"`, config.WorkflowData.ServicePortExpressions)
		awfHelpersLog.Printf("Added --allow-host-service-ports with %s", config.WorkflowData.ServicePortExpressions)
	} else if config.WorkflowData != nil && config.WorkflowData.ServicePortExpressions != "" {
		awfHelpersLog.Print("Skipping --allow-host-service-ports: requires legacy-security mode")
	}

	engineCommand := config.EngineCommand
	if isArcDind {
		engineCommand = rewriteArcDindEngineCommand(engineCommand)
	}

	// Wrap engine command in shell (command already includes any internal setup like npm PATH)
	shellWrappedCommand := WrapCommandInShell(engineCommand)

	// Pre-create the agent stdio log file with restrictive permissions (0600) before
	// starting the AWF container.  tee would otherwise create it with the default
	// umask (0644), leaving secrets (e.g. MCP gateway tokens) world-readable on the
	// runner host until the secret-redaction step runs.
	preCreateLog := fmt.Sprintf("(umask 177 && touch %s)", shellEscapeArg(config.LogFile))

	// Capture the epoch-millisecond timestamp at the very start of the Execute Agent CLI
	// step on the host, before the AWF container launches.  sendJobConclusionSpan reads
	// this file to set the dedicated gh-aw.<job>.agent span start time, which excludes
	// pre-agent overhead such as workspace audit and CLI proxy startup.
	writeAgentCLIStartMs := "printf '%s' \"$(date +%s%3N)\" > " + shellEscapeArg(AgentCLIStartMsPath)

	// Build the complete command with proper formatting.
	// configFileSetup (if non-empty) writes the AWF config JSON immediately before the
	// AWF invocation so the file is present when AWF parses --config.
	//
	// shellcheck directive rationale:
	//   - SC1003 is expected because this generated block intentionally contains GitHub
	//     expression literals (for example ${{ job.services.<id>.ports['<port>'] }})
	//     that include single quotes and must survive into runtime unchanged.
	//   - SC2086 is expected because a subset of AWF arguments are intentionally emitted
	//     as expandable shell fragments (for example ${GH_AW_TOOL_CACHE_MOUNT:+...} and
	//     ${GH_AW_DOCKER_HOST:+...}). These fragments are produced by trusted
	//     compiler-owned probes above and are not user-provided free-form shell input.
	//
	// We keep normal quoting for all user-controlled values via shellEscapeArg/shellJoinArgs
	// and scope this suppression to the generated AWF invocation line only.
	var command string
	if config.PathSetup != "" && configFileSetup != "" {
		command = fmt.Sprintf(`set -o pipefail
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
			writeAgentCLIStartMs,
			config.PathSetup,
			preCreateLog,
			configFileSetup,
			modelsJSONPathExport,
			arcDindDockerHostProbe,
			arcDindPrefixProbe,
			toolCacheMountProbe,
			awfShellcheckDirective,
			awfCommand,
			expandableArgs,
			toolCacheMountRef,
			arcDindDockerHostRef,
			shellJoinArgs(awfArgs),
			shellWrappedCommand,
			shellEscapeArg(config.LogFile))
	} else if config.PathSetup != "" {
		// Include path setup before AWF command (runs on host before AWF)
		command = fmt.Sprintf(`set -o pipefail
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
			writeAgentCLIStartMs,
			config.PathSetup,
			preCreateLog,
			modelsJSONPathExport,
			arcDindDockerHostProbe,
			arcDindPrefixProbe,
			toolCacheMountProbe,
			awfShellcheckDirective,
			awfCommand,
			expandableArgs,
			toolCacheMountRef,
			arcDindDockerHostRef,
			shellJoinArgs(awfArgs),
			shellWrappedCommand,
			shellEscapeArg(config.LogFile))
	} else if configFileSetup != "" {
		command = fmt.Sprintf(`set -o pipefail
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
			writeAgentCLIStartMs,
			preCreateLog,
			configFileSetup,
			modelsJSONPathExport,
			arcDindDockerHostProbe,
			arcDindPrefixProbe,
			toolCacheMountProbe,
			awfShellcheckDirective,
			awfCommand,
			expandableArgs,
			toolCacheMountRef,
			arcDindDockerHostRef,
			shellJoinArgs(awfArgs),
			shellWrappedCommand,
			shellEscapeArg(config.LogFile))
	} else {
		command = fmt.Sprintf(`set -o pipefail
%s
%s
%s
%s
%s
%s
%s
%s %s %s %s %s \
  -- %s 2>&1 | tee -a %s`,
			writeAgentCLIStartMs,
			preCreateLog,
			modelsJSONPathExport,
			arcDindDockerHostProbe,
			arcDindPrefixProbe,
			toolCacheMountProbe,
			awfShellcheckDirective,
			awfCommand,
			expandableArgs,
			toolCacheMountRef,
			arcDindDockerHostRef,
			shellJoinArgs(awfArgs),
			shellWrappedCommand,
			shellEscapeArg(config.LogFile))
	}

	awfHelpersLog.Print("Successfully built AWF command")
	return command
}
