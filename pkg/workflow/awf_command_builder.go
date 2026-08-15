// This file contains AWF command and argument assembly helpers.

package workflow

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/console"
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
	isCloudHypervisor := isCloudHypervisorRuntime(config.WorkflowData)

	// Auto-detect ARC/DinD split daemon topology at runtime: probe DOCKER_HOST for a
	// tcp:// scheme and pass it through to AWF via --docker-host.
	// All behaviors avoid requiring workflow-authored sandbox.agent.args for standard ARC DinD setups.
	// When AWF also supports chroot config (v0.27.1+), the chroot patch logic is embedded
	// inside the same if-block so the script only contains one DOCKER_HOST condition check.
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
	if isCloudHypervisor {
		toolCacheMountProbe = ""
		toolCacheMountRef = ""
	}

	// Build the expandable args string for args that need shell variable expansion.
	// These MUST be appended as raw (unescaped) strings because single-quoting would
	// prevent the runner's shell from expanding ${GITHUB_WORKSPACE} and ${RUNNER_TEMP}.
	ghAwDir := constants.GhAwRootDirShell
	expandableArgs := `--container-workdir "${GITHUB_WORKSPACE}"`
	if !isCloudHypervisor {
		expandableArgs += fmt.Sprintf(
			` --mount "%s:%s:ro" --mount "%s:/host%s:ro"`,
			ghAwDir, ghAwDir, ghAwDir, ghAwDir,
		)
	}
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
	if !isCloudHypervisor && config.WorkflowData != nil && config.WorkflowData.SafeOutputs != nil && config.WorkflowData.SafeOutputs.UploadArtifact != nil {
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
	if isCloudHypervisorRuntime(config.WorkflowData) {
		expandableArgs += ` --cloud-hypervisor-binary "${GH_AW_CLOUD_HYPERVISOR_BINARY}"` +
			` --cloud-hypervisor-kernel "${GH_AW_CLOUD_HYPERVISOR_KERNEL}"` +
			` --cloud-hypervisor-rootfs "${GH_AW_CLOUD_HYPERVISOR_ROOTFS}"` +
			` --cloud-hypervisor-supervisor "${GH_AW_CLOUD_HYPERVISOR_SUPERVISOR}"` +
			` --cloud-hypervisor-binary-sha256 "${GH_AW_CLOUD_HYPERVISOR_BINARY_SHA256}"` +
			` --cloud-hypervisor-kernel-sha256 "${GH_AW_CLOUD_HYPERVISOR_KERNEL_SHA256}"` +
			` --cloud-hypervisor-rootfs-sha256 "${GH_AW_CLOUD_HYPERVISOR_ROOTFS_SHA256}"` +
			` --cloud-hypervisor-supervisor-sha256 "${GH_AW_CLOUD_HYPERVISOR_SUPERVISOR_SHA256}"` +
			` --cloud-hypervisor-virtiofsd-sha256 "${GH_AW_CLOUD_HYPERVISOR_VIRTIOFSD_SHA256}"`
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

	var awfArgs []string

	// Add TTY flag if needed (Claude requires this), except for docker-sbx where
	// sbx exec --tty can terminate long-running Claude sessions prematurely.
	if config.UsesTTY && !isDockerSbxRuntime(config.WorkflowData) && !isCloudHypervisorRuntime(config.WorkflowData) {
		awfArgs = append(awfArgs, "--tty")
	}

	// docker-sbx: tell AWF to launch the agent inside a Docker sbx microVM instead
	// of as a standard Docker Compose service. Guard on the effective AWF version so
	// older binaries do not receive an unknown flag.
	if isDockerSbxRuntime(config.WorkflowData) && awfSupportsContainerRuntime(firewallConfig) {
		awfArgs = append(awfArgs, "--container-runtime", "sbx")
		awfHelpersLog.Print("Added --container-runtime sbx for docker-sbx microVM runtime")
	} else if isDockerSbxRuntime(config.WorkflowData) {
		awfHelpersLog.Printf("Skipping --container-runtime sbx: AWF version %q is older than required minimum %s", getAWFImageTag(firewallConfig), constants.AWFContainerRuntimeMinVersion)
	}
	if isCloudHypervisorRuntime(config.WorkflowData) && awfSupportsCloudHypervisor(firewallConfig) {
		awfArgs = append(
			awfArgs,
			"--container-runtime", "cloud-hypervisor",
			"--cloud-hypervisor-preview",
			"--cloud-hypervisor-vcpus", strconv.Itoa(constants.DefaultCloudHypervisorVCPUs),
			"--cloud-hypervisor-memory-mib", strconv.Itoa(constants.DefaultCloudHypervisorMemoryMiB),
		)
		awfHelpersLog.Print("Added cloud-hypervisor runtime arguments")
	} else if isCloudHypervisorRuntime(config.WorkflowData) {
		awfHelpersLog.Printf("Skipping cloud-hypervisor runtime flags: AWF version %q is older than required minimum %s", getAWFImageTag(firewallConfig), constants.AWFCloudHypervisorMinVersion)
	}

	// Pass all environment variables to the container, but exclude every variable whose
	// step-env value comes from a GitHub Actions secret. AWF's API proxy (--enable-api-proxy)
	// handles authentication for these tokens transparently, so the container does not need
	// the raw values. Excluding them via --exclude-env prevents a prompt-injected agent from
	// exfiltrating tokens through bash tools such as `env` or `printenv`.
	// The caller computes ExcludeEnvVarNames from ComputeAWFExcludeEnvVarNames() so that every
	// secret-bearing variable is covered — not just a hardcoded subset.
	// --exclude-env requires AWF v0.25.3+; skip the flags for workflows that pin an older version.
	awfArgs = append(awfArgs, "--env-all")
	if awfSupportsExcludeEnv(firewallConfig) {
		// Sort for deterministic output in compiled lock files.
		sortedExclude := make([]string, len(config.ExcludeEnvVarNames))
		copy(sortedExclude, config.ExcludeEnvVarNames)
		sort.Strings(sortedExclude)
		for _, excludedVar := range sortedExclude {
			awfArgs = append(awfArgs, "--exclude-env", excludedVar)
		}
	} else {
		awfHelpersLog.Printf("Skipping --exclude-env: AWF version %q is older than minimum %s", getAWFImageTag(firewallConfig), constants.AWFExcludeEnvMinVersion)
	}

	// Note: --container-workdir "${GITHUB_WORKSPACE}" and --mount "${RUNNER_TEMP}/gh-aw:..."
	// are intentionally NOT added here. They contain shell variable references that require
	// double-quote expansion. These args are appended raw in BuildAWFCommand to ensure
	// ${GITHUB_WORKSPACE} and ${RUNNER_TEMP} are expanded by the runner's shell.

	// Mount the /tmp/gh-aw runtime tree read-write so the agentic engine can write
	// logs, cache-memory, and other runtime artifacts there (e.g. Codex/Copilot logs,
	// threat-detection files). This is distinct from the ${RUNNER_TEMP}/gh-aw setup
	// tree above, which stays read-only. Matches constants.DefaultTmpGhAwMount already
	// used for containerized MCP servers (see mcp_renderer_builtin.go) so the same
	// read-write access is guaranteed for every sandbox runtime (chroot, gVisor,
	// docker-sbx), not just topologies where /tmp/gh-aw happens to be writable by
	// default via the host filesystem.
	if !isCloudHypervisorRuntime(config.WorkflowData) {
		awfArgs = append(awfArgs, "--mount", constants.DefaultTmpGhAwMount)
	}

	// Add custom mounts from agent config if specified
	if !isCloudHypervisorRuntime(config.WorkflowData) && agentConfig != nil && len(agentConfig.Mounts) > 0 {
		// Sort mounts for consistent output
		sortedMounts := make([]string, len(agentConfig.Mounts))
		copy(sortedMounts, agentConfig.Mounts)
		sort.Strings(sortedMounts)

		for _, mount := range sortedMounts {
			awfArgs = append(awfArgs, "--mount", mount)
		}
		awfHelpersLog.Printf("Added %d custom mounts from agent config", len(sortedMounts))
	}

	// Set log level
	awfLogLevel := string(constants.AWFDefaultLogLevel)
	if firewallConfig != nil && firewallConfig.LogLevel != "" {
		awfLogLevel = firewallConfig.LogLevel
	}
	awfArgs = append(awfArgs, "--log-level", awfLogLevel)
	if isFeatureEnabled(constants.AwfDiagnosticLogsFeatureFlag, config.WorkflowData) {
		awfArgs = append(awfArgs, "--diagnostic-logs")
		awfHelpersLog.Print("Added --diagnostic-logs because awf-diagnostic-logs feature flag is enabled")
	}

	// Legacy security mode: emit --legacy-security and --enable-host-access.
	isLegacy := agentConfig != nil && agentConfig.LegacySecurity
	if isLegacy {
		if awfSupportsLegacySecurity(firewallConfig) {
			awfArgs = append(awfArgs, "--legacy-security")
			awfHelpersLog.Print("Added --legacy-security (legacy-security: enable in frontmatter)")
		} else {
			// AWF versions older than v0.27.32 don't support --legacy-security;
			// they run in legacy mode by default so the flag is unnecessary.
			awfHelpersLog.Printf("Skipping --legacy-security: AWF version %q is older than minimum %s (legacy mode is the default for older versions)", getAWFImageTag(firewallConfig), constants.AWFLegacySecurityMinVersion)
		}

		awfArgs = append(awfArgs, "--enable-host-access")
		awfHelpersLog.Print("Added --enable-host-access for legacy security mode")

		// --allow-host-ports requires --enable-host-access, so this is only ever
		// emitted in legacy-security mode. AWF's strict security mode (the default)
		// does not provide a route to host services even when --allow-host-ports is
		// combined with --enable-host-access, so emitting it there would be both
		// invalid (strict mode strips --enable-host-access on incompatible runtimes)
		// and misleading (it would not make services reachable).
		hostPorts := collectAllowedHostPorts(config.WorkflowData, agentConfig)
		if len(hostPorts) > 0 {
			if awfSupportsAllowHostPorts(firewallConfig) {
				hostPortsValue := joinPorts(hostPorts)
				awfArgs = append(awfArgs, "--allow-host-ports", hostPortsValue)
				awfHelpersLog.Printf("Added --allow-host-ports %s", hostPortsValue)
			} else {
				warning := fmt.Sprintf("sandbox host ports require AWF %s or newer; skipping --allow-host-ports for AWF version %q", constants.AWFAllowHostPortsMinVersion, getAWFImageTag(firewallConfig))
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(warning))
				awfHelpersLog.Printf("Warning: %s", warning)
			}
		}
	} else {
		awfHelpersLog.Print("Strict security: skipping host-access flag (default)")
		if agentConfig != nil && len(agentConfig.AllowHostPorts) > 0 {
			warning := "sandbox.agent.allow-host-ports has no effect in strict security mode (the default); set sandbox.agent.legacy-security: enable to reach host ports"
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(warning))
			awfHelpersLog.Printf("Warning: %s", warning)
		}
	}

	// Skip pulling images since they are pre-downloaded
	awfArgs = append(awfArgs, "--skip-pull")
	awfHelpersLog.Print("Using --skip-pull since images are pre-downloaded")

	// Enable CLI proxy sidecar when GitHub mode is gh-proxy.
	// Start the difc-proxy on the host and tell AWF where to connect
	// (firewall v0.25.17+).
	if isGitHubCLIModeEnabled(config.WorkflowData) {
		if awfSupportsCliProxy(firewallConfig) {
			difcProxyHost := "host.docker.internal:18443"
			if isAWFNetworkIsolationEnabled(config.WorkflowData) {
				difcProxyHost = "awmg-cli-proxy:18443"
			}
			awfArgs = append(awfArgs, "--difc-proxy-host", difcProxyHost)
			awfArgs = append(awfArgs, "--difc-proxy-ca-cert", constants.TmpDIFCProxyTLSCACert)
			awfHelpersLog.Print("Added --difc-proxy-host and --difc-proxy-ca-cert for CLI proxy sidecar")
		} else {
			awfHelpersLog.Printf("Skipping CLI proxy flags: AWF version %q is older than minimum %s", getAWFImageTag(firewallConfig), constants.AWFCliProxyMinVersion)
		}
	}

	// Pass base path if URL contains a path component
	// This is required for endpoints with path prefixes (e.g., Databricks /serving-endpoints,
	// Azure OpenAI /openai/deployments/<name>, corporate LLM routers with path-based routing)
	// Base paths remain as CLI flags — they are not yet represented in the config file schema.
	openaiBasePath := extractAPIBasePath(config.WorkflowData, "OPENAI_BASE_URL")
	if openaiBasePath != "" {
		awfArgs = append(awfArgs, "--openai-api-base-path", openaiBasePath)
		awfHelpersLog.Printf("Added --openai-api-base-path=%s", openaiBasePath)
	}

	anthropicBasePath := extractAPIBasePath(config.WorkflowData, "ANTHROPIC_BASE_URL")
	if anthropicBasePath != "" {
		awfArgs = append(awfArgs, "--anthropic-api-base-path", anthropicBasePath)
		awfHelpersLog.Printf("Added --anthropic-api-base-path=%s", anthropicBasePath)
	}

	geminiBasePath := extractAPIBasePath(config.WorkflowData, "GEMINI_API_BASE_URL")
	if geminiBasePath != "" {
		awfArgs = append(awfArgs, "--gemini-api-base-path", geminiBasePath)
		awfHelpersLog.Printf("Added --gemini-api-base-path=%s", geminiBasePath)
	}

	// Add SSL Bump support for HTTPS content inspection (v0.9.0+)
	sslBumpArgs := getSSLBumpArgs(firewallConfig)
	awfArgs = append(awfArgs, sslBumpArgs...)

	// Add custom args if specified in firewall config
	if firewallConfig != nil && len(firewallConfig.Args) > 0 {
		awfArgs = append(awfArgs, firewallConfig.Args...)
	}

	// Add custom args from agent config if specified
	if agentConfig != nil && len(agentConfig.Args) > 0 {
		awfArgs = append(awfArgs, agentConfig.Args...)
		awfHelpersLog.Printf("Added %d custom args from agent config", len(agentConfig.Args))
	}

	// Pass memory limit to AWF container if specified in agent config
	if agentConfig != nil && agentConfig.Memory != "" {
		awfArgs = append(awfArgs, "--memory-limit", agentConfig.Memory)
		awfHelpersLog.Printf("Set AWF memory limit to %s", agentConfig.Memory)
	}

	awfHelpersLog.Printf("Built %d AWF arguments", len(awfArgs))
	return awfArgs
}

// collectAllowedHostPorts merges the default host-access ports (80, 443, and the
// MCP gateway port) with any explicit sandbox.agent.allow-host-ports values.
//
// This is only called in legacy-security mode: --allow-host-ports requires
// --enable-host-access, which is legacy-only. GitHub Actions services: ports
// are intentionally NOT derived here — AWF's --allow-host-service-ports flag
// (see ExtractServicePortExpressions) is the correct mechanism for reaching
// services, since it resolves the actual (possibly dynamically assigned) host
// port at runtime via ${{ job.services['<id>'].ports['<port>'] }} expressions
// rather than relying on a static port number.
func collectAllowedHostPorts(workflowData *WorkflowData, agentConfig *AgentSandboxConfig) []int {
	ports := map[int]struct{}{
		80:  {},
		443: {},
	}
	ports[getMCPGatewayPort(workflowData)] = struct{}{}
	if agentConfig != nil {
		for _, port := range agentConfig.AllowHostPorts {
			if port < minPort || port > maxPort {
				continue
			}
			// Defense-in-depth: dangerous ports must never reach --allow-host-ports,
			// even if validateAllowHostPorts was bypassed or its call order changes.
			if _, dangerous := awfDangerousHostPorts[port]; dangerous {
				continue
			}
			ports[port] = struct{}{}
		}
	}
	result := make([]int, 0, len(ports))
	for port := range ports {
		result = append(result, port)
	}
	sort.Ints(result)
	return result
}

func getMCPGatewayPort(workflowData *WorkflowData) int {
	if workflowData != nil && workflowData.SandboxConfig != nil &&
		workflowData.SandboxConfig.MCP != nil && workflowData.SandboxConfig.MCP.Port > 0 {
		return workflowData.SandboxConfig.MCP.Port
	}
	return int(DefaultMCPGatewayPort)
}

func joinPorts(ports []int) string {
	parts := make([]string, len(ports))
	for i, port := range ports {
		parts[i] = strconv.Itoa(port)
	}
	return strings.Join(parts, ",")
}

// GetAWFCommandPrefix determines the AWF command to use (custom or standard).
// This extracts the common pattern for determining AWF command from agent config.
//
// Parameters:
//   - workflowData: The workflow data containing agent configuration
//
// Returns:
//   - string: The AWF command to use (e.g., "sudo --preserve-env awf",
//     "sudo -E awf", "awf", or custom command)
func GetAWFCommandPrefix(workflowData *WorkflowData) string {
	agentConfig := getAgentConfig(workflowData)
	if agentConfig != nil && agentConfig.Command != "" {
		awfHelpersLog.Printf("Using custom AWF command: %s", agentConfig.Command)
		return agentConfig.Command
	}

	// Cloud Hypervisor needs host privileges to access KVM and configure the VM.
	// This is still AWF strict security: the guest remains network-isolated and
	// no legacy-security or host-access flags are implied by the sudo prefix.
	if isCloudHypervisorRuntime(workflowData) {
		awfHelpersLog.Print("Using privileged AWF command for cloud-hypervisor strict security")
		return string(constants.AWFCloudHypervisorCommand)
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
