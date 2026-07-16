// Package workflow - external-detector-specific install, run, and conclude logic.
package workflow

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
)

func (c *Compiler) buildPrepareDetectionEngineConfigForExternalDetectorStep(data *WorkflowData) []string {
	if c.getExternalThreatDetectionEngineID(data) != "codex" {
		return nil
	}

	const emptyMCPServersJSON = `{"mcpServers":{}}`
	shellCodexConfigPath := constants.ShellMcpConfigDir + "/config.toml"
	codexHomeConfigPath := constants.TmpMcpConfigDir + "/config.toml"
	codexAPIBase := NewCodexEngine().getOpenAIProxyProviderBaseURL()
	codexWSSBase := codexProxyWebsocketBaseURL(codexAPIBase)
	codexConfig := buildExternalDetectorCodexConfig(codexAPIBase, codexWSSBase)
	codexConfigDelimiter := GenerateHeredocDelimiterFromContent("CODEX_DETECTION_CONFIG", codexConfig)

	return []string{
		"      - name: Prepare Codex config for threat-detect\n",
		fmt.Sprintf("        if: %s\n", detectionStepCondition),
		"        run: |\n",
		fmt.Sprintf("          mkdir -p %q %q %q\n", constants.ShellMcpConfigDir, constants.TmpMcpConfigDir, constants.TmpMcpConfigLogsDir),
		fmt.Sprintf("          printf '%%s\\n' %q > %q\n", emptyMCPServersJSON, constants.ShellMcpServersJsonPath),
		"          # Point Codex at the AWF OpenAI proxy and disable websocket startup.\n",
		fmt.Sprintf("          cat > %q << %s\n", shellCodexConfigPath, codexConfigDelimiter),
		codexConfig,
		fmt.Sprintf("          %s\n", codexConfigDelimiter),
		fmt.Sprintf("          cp %q %q\n", shellCodexConfigPath, codexHomeConfigPath),
		fmt.Sprintf("          chmod 600 %q %q\n", shellCodexConfigPath, codexHomeConfigPath),
	}
}

func buildExternalDetectorCodexConfig(apiBase, wssBase string) string {
	return strings.Join([]string{
		"          [history]",
		"          persistence = \"none\"",
		"",
		"          model_provider = \"" + codexOpenAIProxyProviderID + "\"",
		"",
		"          [model_providers." + codexOpenAIProxyProviderID + "]",
		"          name = \"" + codexOpenAIProxyProviderName + "\"",
		"          base_url = \"" + apiBase + "\"",
		"          api_base = \"" + apiBase + "\"",
		"          wss_base = \"" + wssBase + "\"",
		"          env_key = \"OPENAI_API_KEY\"",
		"          supports_websockets = false",
		"",
	}, "\n")
}

func codexProxyWebsocketBaseURL(apiBase string) string {
	switch {
	case strings.HasPrefix(apiBase, "https://"):
		return "wss://" + strings.TrimPrefix(apiBase, "https://")
	case strings.HasPrefix(apiBase, "http://"):
		return "ws://" + strings.TrimPrefix(apiBase, "http://")
	default:
		return apiBase
	}
}

// buildThreatDetectionWorkflowData creates the shared minimal WorkflowData used by
// detection-job helper steps so topology- and feature-dependent behavior stays in sync.
// It always initializes SandboxConfig.Agent because downstream detection helpers
// extend the agent sandbox configuration (for example, external-detector mounts).
// Callers can pass an empty engineID to inherit the detection job's default engine
// resolution from the source WorkflowData.
func buildThreatDetectionWorkflowData(data *WorkflowData, engineID string) *WorkflowData {
	if engineID == "" {
		engineID = data.AI
	}
	if engineID == "" {
		engineID = "claude"
	}

	return &WorkflowData{
		AI:                engineID,
		ActionCache:       data.ActionCache,
		Features:          data.Features,
		Permissions:       data.Permissions,
		CachedPermissions: data.CachedPermissions,
		IsDetectionRun:    true,
		RunnerConfig:      data.RunnerConfig,
		SandboxConfig: &SandboxConfig{
			Agent: &AgentSandboxConfig{
				Type: SandboxTypeAWF,
			},
		},
	}
}

// buildPullAWFContainersStep creates a step that pre-pulls AWF (agent workflow firewall)
// container images in the detection job. The detection engine runs inside AWF, which uses
// the firewall stack containers needed for the selected topology (squid, agent, api-proxy,
// plus build-tools on arc-dind). Pre-pulling avoids on-demand pulls at runtime. Only AWF
// images are pulled here; MCP server images are not needed for detection.
func (c *Compiler) buildPullAWFContainersStep(data *WorkflowData) []string {
	// Build a minimal WorkflowData that represents the detection engine context so
	// collectDockerImages returns only the AWF firewall images (no MCP tool images).
	detectionData := buildThreatDetectionWorkflowData(data, "")
	detectionData.Tools = map[string]any{}

	images := collectDockerImages(detectionData.Tools, detectionData, c.actionMode)
	if len(images) == 0 {
		return nil
	}

	var b strings.Builder
	generateDownloadDockerImagesStep(&b, images)
	if b.Len() == 0 {
		return nil
	}

	// Split the generated YAML into individual lines so each is a separate entry
	lines := strings.Split(b.String(), "\n")
	var steps []string
	for _, line := range lines {
		if line != "" {
			steps = append(steps, line+"\n")
		}
	}
	return steps
}

// getThreatDetectionEngineID returns the effective engine ID for the detection job.
// It mirrors threat-detection engine resolution: threat-detection.engine overrides main engine.
func (c *Compiler) getThreatDetectionEngineID(data *WorkflowData) string {
	var engineID string

	if data.SafeOutputs != nil && data.SafeOutputs.ThreatDetection != nil &&
		data.SafeOutputs.ThreatDetection.EngineConfig != nil &&
		data.SafeOutputs.ThreatDetection.EngineConfig.ID != "" {
		engineID = data.SafeOutputs.ThreatDetection.EngineConfig.ID
	} else {
		engineID = data.AI
		if engineID == "" && data.EngineConfig != nil && data.EngineConfig.ID != "" {
			engineID = data.EngineConfig.ID
		}
	}

	if engineID == "" {
		engineID = "claude"
	}

	// Threat detection currently does not support the Pi engine backend.
	// Normalize to Copilot so workflows with engine: pi still get a working detector.
	if engineID == "pi" {
		return "copilot"
	}

	return engineID
}

// getExternalThreatDetectionEngineID returns the engine used by the external
// threat-detect path. Threat-detection engine resolution is centralized in
// getThreatDetectionEngineID, including Pi -> Copilot normalization.
func (c *Compiler) getExternalThreatDetectionEngineID(data *WorkflowData) string {
	return c.getThreatDetectionEngineID(data)
}

// buildInstallAWFForExternalDetectorStep creates the AWF installation step required
// by the external detector execution path, which invokes `awf` directly.
func (c *Compiler) buildInstallAWFForExternalDetectorStep(data *WorkflowData) []string {
	version := string(constants.DefaultFirewallVersion)
	if firewallConfig := getFirewallConfig(data); firewallConfig != nil && firewallConfig.Version != "" {
		version = firewallConfig.Version
	}

	step := generateAWFInstallationStep(version, nil)
	if len(step) == 0 {
		return nil
	}

	lines := make([]string, 0, len(step))
	for _, line := range step {
		lines = append(lines, line+"\n")
	}
	return lines
}

// buildInstallDetectionEngineForExternalDetectorStep installs the selected detection
// engine in the external detector path so threat-detect can invoke the engine binary.
func (c *Compiler) buildInstallDetectionEngineForExternalDetectorStep(data *WorkflowData) []string {
	engineID := c.getExternalThreatDetectionEngineID(data)
	engine, err := c.getAgenticEngine(engineID)
	if err != nil {
		threatLog.Printf("Failed to resolve detection engine %q for external detector installation: %v (compilation will continue without engine install steps; threat-detect will only succeed if the engine binary is already available at runtime)", engineID, err)
		return nil
	}

	// Build a synthetic detection WorkflowData solely to generate the engine's
	// installation steps for this separate detection job context.
	threatDetectionData := buildThreatDetectionWorkflowData(data, engineID)
	threatDetectionData.Tools = map[string]any{
		"bash": []any{"*"},
	}
	threatDetectionData.EngineConfig = &EngineConfig{ID: engineID}

	if canReuseThreatDetectionEngineConfigForExternalDetector(data, engineID) {
		threatDetectionData.EngineConfig = cloneThreatDetectionEngineConfig(engineID, data.SafeOutputs.ThreatDetection.EngineConfig)
	}
	threatDetectionData.EngineConfig.Env = mergeThreatDetectionEngineEnv(data, threatDetectionData.EngineConfig.Env)
	if threatDetectionData.EngineConfig.APITarget == "" && data.EngineConfig != nil {
		threatDetectionData.EngineConfig.APITarget = data.EngineConfig.APITarget
	}

	installSteps := engine.GetInstallationSteps(threatDetectionData)
	var lines []string
	for _, step := range installSteps {
		if isAWFBinaryInstallStep(step) {
			continue
		}
		for _, line := range step {
			lines = append(lines, line+"\n")
		}
	}

	return lines
}

func isAWFBinaryInstallStep(step GitHubActionStep) bool {
	for _, line := range step {
		if strings.Contains(line, "install_awf_binary.sh") {
			return true
		}
	}
	return false
}

func appendThreatDetectionRWMount(mounts []string) []string {
	threatDetectionMount := constants.ThreatDetectionDir + ":" + constants.ThreatDetectionDir + ":rw"
	if slices.Contains(mounts, threatDetectionMount) {
		return mounts
	}
	return append(mounts, threatDetectionMount)
}

// buildExternalDetectorExecutionStep creates the AWF execution step for the external
// threat-detect binary. It runs threat-detect inside the AWF firewall sandbox with a
// read-write mount so detection_result.json can be written from inside the container
// back to the host filesystem. This replaces the inline engine execution step when
// features: gh-aw-detection: true is set.
func (c *Compiler) buildExternalDetectorExecutionStep(data *WorkflowData) []string {
	if data.SafeOutputs != nil && data.SafeOutputs.ThreatDetection != nil &&
		data.SafeOutputs.ThreatDetection.EngineDisabled {
		return []string{
			"      # AI engine disabled for threat detection (engine: false)\n",
		}
	}

	engineID := c.getExternalThreatDetectionEngineID(data)
	engine, err := c.getAgenticEngine(engineID)
	if err != nil {
		return []string{fmt.Sprintf("      # Failed to resolve detection engine %q: %v\n", engineID, err)}
	}

	// Build detection WorkflowData for the external detector.
	// The rw mount for ThreatDetectionDir allows the threat-detect binary to write
	// detection_result.json from inside the AWF container to the host filesystem.
	threatDetectionData := buildThreatDetectionWorkflowData(data, engineID)
	threatDetectionData.Tools = map[string]any{
		"bash": []any{"*"},
	}
	threatDetectionData.EngineConfig = &EngineConfig{ID: engineID}
	threatDetectionData.NetworkPermissions = &NetworkPermissions{
		Allowed: getThreatDetectionAdditionalAllowedDomains(data),
	}
	// Add a read-write mount so the threat-detect binary can write
	// detection_result.json inside the container and it becomes visible
	// on the host through the bind mount.
	threatDetectionData.SandboxConfig.Agent.Mounts = appendThreatDetectionRWMount(threatDetectionData.SandboxConfig.Agent.Mounts)

	// Inherit engine config overrides from threat-detection config when set.
	if canReuseThreatDetectionEngineConfigForExternalDetector(data, engineID) {
		threatDetectionData.EngineConfig = cloneThreatDetectionEngineConfig(engineID, data.SafeOutputs.ThreatDetection.EngineConfig)
	}
	threatDetectionData.EngineConfig.Env = mergeThreatDetectionEngineEnv(data, threatDetectionData.EngineConfig.Env)
	// Inherit APITarget from main engine config for GHE/custom endpoints.
	if threatDetectionData.EngineConfig.APITarget == "" && data.EngineConfig != nil {
		threatDetectionData.EngineConfig.APITarget = data.EngineConfig.APITarget
	}

	// Compute which env vars to exclude from the AWF container. The API proxy
	// handles authentication, so the raw credentials must not reach the container.
	excludeEnvVarNames := ComputeAWFExcludeEnvVarNames(threatDetectionData, engineCoreSecretVarNames(engineID))

	// Compute allowed domains for the detection engine. The AWF firewall for the
	// detection job must permit the engine's required API endpoints. Without this,
	// engines such as Codex (which connects to api.openai.com and chatgpt.com) fail
	// with "domain not in allowlist" and the detection job exits with code 1/2.
	allowedDomains := GetAllowedDomainsForEngine(constants.EngineName(engineID), threatDetectionData.NetworkPermissions, data.Tools, data.Runtimes)
	// Extend the allowlist with any custom API target domains when engine.api-target
	// is set (e.g. GHE or a custom OpenAI-compatible endpoint).
	if threatDetectionData.EngineConfig != nil && threatDetectionData.EngineConfig.APITarget != "" {
		allowedDomains = mergeAPITargetDomains(allowedDomains, threatDetectionData.EngineConfig.APITarget)
	}

	// Build the threat-detect command. The binary reads the prepared detection
	// artifacts directory from /tmp/gh-aw/threat-detection/ (set up by previous
	// steps) and writes the structured verdict to detection_result.json there.
	// Prepend npm PATH setup so that npm-installed engine CLIs (e.g. claude, codex)
	// can be found inside the AWF container's chroot environment. threat-detect
	// invokes the engine binary as a subprocess and relies on PATH to locate it.
	npmPathSetup := GetNpmBinPathSetup()
	threatDetectCmd := fmt.Sprintf(
		"%s && threat-detect --engine %s --output %s %s",
		npmPathSetup,
		engineID,
		shellEscapeArg(constants.ThreatDetectionResultPath),
		shellEscapeArg(constants.ThreatDetectionDir),
	)

	// Build the complete AWF command. BuildAWFCommand handles config file setup,
	// ARC/DinD probes, tool cache mount, and the log tee pattern.
	awfConfig := AWFCommandConfig{
		EngineName:         engineID,
		EngineCommand:      threatDetectCmd,
		LogFile:            constants.ThreatDetectionLogPath,
		WorkflowData:       threatDetectionData,
		ExcludeEnvVarNames: excludeEnvVarNames,
		AllowedDomains:     allowedDomains,
	}
	command := BuildAWFCommand(awfConfig)

	steps := []string{
		"      - name: Execute threat detection with AWF\n",
		"        id: detection_agentic_execution\n",
		fmt.Sprintf("        if: %s\n", detectionStepCondition),
		"        continue-on-error: true\n",
		"        run: |\n",
	}
	for _, line := range strings.SplitAfter(command, "\n") {
		if line == "" {
			continue
		}
		prefixed := "          " + line
		if !strings.HasSuffix(prefixed, "\n") {
			prefixed += "\n"
		}
		steps = append(steps, prefixed)
	}

	// Reuse the engine's own execution env block so the external detector path
	// gets the same token/model/runtime environment configuration as the agent job.
	executionSteps := engine.GetExecutionSteps(threatDetectionData, constants.ThreatDetectionLogPath)
	if len(executionSteps) > 0 {
		envLines := extractStepEnvLines(executionSteps[0])
		if len(envLines) == 0 {
			threatLog.Printf("Detection engine %q execution step did not expose env lines; external detector will run with minimal env", engineID)
		}
		for _, line := range envLines {
			steps = append(steps, line+"\n")
		}
	} else {
		threatLog.Printf("Detection engine %q did not generate execution steps; external detector will run with minimal env", engineID)
	}

	return steps
}

// extractStepEnvLines copies the YAML env: block from a rendered engine execution step.
// It intentionally stops when a comment line appears because comments in step templates
// are section separators, and consuming past them may bleed into non-env content.
func extractStepEnvLines(step GitHubActionStep) []string {
	envIndex := -1
	for i, line := range step {
		if strings.TrimSpace(line) == "env:" {
			envIndex = i
			break
		}
	}
	if envIndex == -1 {
		return nil
	}

	var envLines []string
	for _, line := range step[envIndex:] {
		if line == "" {
			continue
		}
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			break
		}
		if !strings.HasPrefix(line, stepEnvIndent) && trimmed != "env:" {
			break
		}
		envLines = append(envLines, line)
	}

	return envLines
}

// buildUploadDetectionArtifactStep creates a step that uploads both the structured
// verdict file (detection_result.json) and the detection log (detection.log) as the
// detection artifact. Used when features: gh-aw-detection: true is set; the inline
// path uses buildUploadDetectionLogStep which only uploads detection.log.
func (c *Compiler) buildUploadDetectionArtifactStep(data *WorkflowData) []string {
	detectionArtifactName := artifactPrefixExprForAgentDownstreamJob(data) + constants.DetectionArtifactName
	return []string{
		"      - name: Upload threat detection artifact\n",
		fmt.Sprintf("        if: %s\n", detectionStepCondition),
		fmt.Sprintf("        uses: %s\n", c.getActionPin("actions/upload-artifact")),
		"        with:\n",
		"          name: " + detectionArtifactName + "\n",
		"          path: |\n",
		"            " + constants.ThreatDetectionResultPath + "\n",
		"            " + constants.ThreatDetectionLogPath + "\n",
		"          if-no-files-found: ignore\n",
	}
}

// buildExternalDetectorConcludeStep creates the conclude step for the external
// threat-detect binary. It runs `threat-detect conclude --result-file ...` which reads
// the structured detection_result.json and sets the detection_conclusion/detection_reason/
// detection_success step outputs and exports GH_AW_DETECTION_CONCLUSION/GH_AW_DETECTION_REASON,
// preserving the same gate contract as the inline parse_threat_detection_results.cjs path.
// The step ID (detection_conclusion) and env vars (RUN_DETECTION, DETECTION_AGENTIC_EXECUTION_OUTCOME,
// GH_AW_DETECTION_CONTINUE_ON_ERROR) are byte-identical to the inline conclude step.
func (c *Compiler) buildExternalDetectorConcludeStep(data *WorkflowData) []string {
	// Determine continue-on-error mode (same logic as buildDetectionConclusionStep).
	continueOnError := true
	var continueOnErrorExpr *string
	if data.SafeOutputs != nil && data.SafeOutputs.ThreatDetection != nil {
		continueOnError = data.SafeOutputs.ThreatDetection.IsContinueOnError()
		continueOnErrorExpr = data.SafeOutputs.ThreatDetection.ContinueOnErrorExpr
	}

	steps := []string{
		"      - name: Conclude threat detection\n",
		"        id: detection_conclusion\n",
		"        if: always()\n",
	}

	if continueOnErrorExpr != nil {
		steps = append(steps, fmt.Sprintf("        continue-on-error: %s\n", *continueOnErrorExpr))
	} else if continueOnError {
		steps = append(steps, "        continue-on-error: true\n")
	}

	var coeEnvLine string
	if continueOnErrorExpr != nil {
		coeEnvLine = fmt.Sprintf("          GH_AW_DETECTION_CONTINUE_ON_ERROR: %s\n", *continueOnErrorExpr)
	} else {
		coeEnvLine = fmt.Sprintf("          GH_AW_DETECTION_CONTINUE_ON_ERROR: %q\n", strconv.FormatBool(continueOnError))
	}

	steps = append(steps, []string{
		"        env:\n",
		"          RUN_DETECTION: ${{ steps.detection_guard.outputs.run_detection }}\n",
		"          DETECTION_AGENTIC_EXECUTION_OUTCOME: ${{ steps.detection_agentic_execution.outcome }}\n",
		coeEnvLine,
		"        run: |\n",
		fmt.Sprintf("          bash \"${RUNNER_TEMP}/gh-aw/actions/conclude_threat_detection.sh\" %s\n", shellEscapeArg(constants.ThreatDetectionResultPath)),
	}...)

	return steps
}

// buildWorkspaceCheckoutForDetectionStep creates a checkout step for the detection job.
// It runs only when the agent job produced a patch, so the detection engine can
// analyze code changes in the context of the surrounding codebase.
func (c *Compiler) buildWorkspaceCheckoutForDetectionStep(data *WorkflowData) []string {
	checkoutPin := getActionPin("actions/checkout")
	if checkoutPin == "" {
		threatLog.Print("No action pin found for actions/checkout, skipping workspace checkout step")
		return nil
	}

	steps := []string{
		"      - name: Checkout repository for patch context\n",
		fmt.Sprintf("        if: needs.%s.outputs.has_patch == 'true'\n", constants.AgentJobName),
		fmt.Sprintf("        uses: %s\n", checkoutPin),
		"        with:\n",
		"          persist-credentials: false\n",
	}

	threatLog.Print("Added conditional workspace checkout step for patch context")
	return steps
}
