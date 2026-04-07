package workflow

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
)

// generateMainJobSteps generates the complete sequence of steps for the main agent execution job
// This is the heart of the workflow, orchestrating all steps from checkout through AI execution to artifact upload
func (c *Compiler) generateMainJobSteps(yaml *strings.Builder, data *WorkflowData) error {
	compilerYamlLog.Printf("Generating main job steps for workflow: %s", data.Name)

	// Mask OTLP telemetry headers early so authentication tokens cannot leak in runner
	// debug logs. The workflow-level OTEL_EXPORTER_OTLP_HEADERS env var is available
	// from the very first step, so masking can happen before any other work.
	if isOTLPHeadersPresent(data) {
		yaml.WriteString(generateOTLPHeadersMaskStep())
	}

	// Determine if we need to add a checkout step
	needsCheckout := c.shouldAddCheckoutStep(data)
	compilerYamlLog.Printf("Checkout step needed: %t", needsCheckout)

	// Build a CheckoutManager with any user-configured checkouts
	checkoutMgr := NewCheckoutManager(data.CheckoutConfigs)

	// Propagate the platform (host) repo resolved by the activation job so that
	// checkout steps in this job and in safe_outputs can use the correct repository
	// for .github/.agents sparse checkouts when called cross-repo.
	// The activation job exposes this as needs.activation.outputs.target_repo.
	if hasWorkflowCallTrigger(data.On) && !data.InlinedImports {
		checkoutMgr.SetCrossRepoTargetRepo("${{ needs.activation.outputs.target_repo }}")
	}

	// Mint checkout app tokens directly in the agent job before checkout steps are executed.
	// Tokens cannot be passed via job outputs from the activation job because
	// actions/create-github-app-token calls ::add-mask:: on the token, and the GitHub Actions
	// runner silently drops masked values when used as job outputs (runner v2.308+).
	// By minting here, the token is available as steps.checkout-app-token-{index}.outputs.token
	// within the same job, just like the github-mcp-app-token pattern.
	if checkoutMgr.HasAppAuth() {
		compilerYamlLog.Print("Generating checkout app token minting steps in agent job")
		var checkoutPermissions *Permissions
		if data.Permissions != "" {
			parser := NewPermissionsParser(data.Permissions)
			checkoutPermissions = parser.ToPermissions()
		} else {
			checkoutPermissions = NewPermissions()
		}
		for _, step := range checkoutMgr.GenerateCheckoutAppTokenSteps(c, checkoutPermissions) {
			yaml.WriteString(step)
		}
	}

	// Add checkout step first if needed
	if needsCheckout {
		// Emit the default workspace checkout, applying any user-supplied overrides
		defaultLines := checkoutMgr.GenerateDefaultCheckoutStep(
			c.trialMode,
			c.trialLogicalRepoSlug,
			GetActionPin,
		)
		for _, line := range defaultLines {
			yaml.WriteString(line)
		}

		// Add CLI build steps in dev mode (after automatic checkout, before other steps)
		// This builds the gh-aw CLI and Docker image for use by the agentic-workflows MCP server
		// Only generate build steps if agentic-workflows tool is enabled
		if c.actionMode.IsDev() {
			if _, hasAgenticWorkflows := data.Tools["agentic-workflows"]; hasAgenticWorkflows {
				compilerYamlLog.Printf("Generating CLI build steps for dev mode (agentic-workflows tool enabled)")
				c.generateDevModeCLIBuildSteps(yaml)
			} else {
				compilerYamlLog.Printf("Skipping CLI build steps in dev mode (agentic-workflows tool not enabled)")
			}
		}
	}

	// Emit additional (non-default) user-configured checkouts
	additionalLines := checkoutMgr.GenerateAdditionalCheckoutSteps(GetActionPin)
	for _, line := range additionalLines {
		yaml.WriteString(line)
	}

	// Add checkout steps for repository imports
	// Each repository import needs to be checked out into a temporary folder
	// so the merge script can copy files from it
	if len(data.RepositoryImports) > 0 {
		compilerYamlLog.Printf("Adding checkout steps for %d repository imports", len(data.RepositoryImports))
		c.generateRepositoryImportCheckouts(yaml, data.RepositoryImports)
	}

	// Add checkout step for legacy agent import (if present)
	// This handles the older import format where a specific agent file is imported
	if data.AgentFile != "" && data.AgentImportSpec != "" {
		compilerYamlLog.Printf("Adding checkout step for legacy agent import: %s", data.AgentImportSpec)
		c.generateLegacyAgentImportCheckout(yaml, data.AgentImportSpec)
	}

	// Add merge remote .github folder step for repository imports or agent imports
	needsGithubMerge := (len(data.RepositoryImports) > 0) || (data.AgentFile != "" && data.AgentImportSpec != "")
	if needsGithubMerge {
		compilerYamlLog.Printf("Adding merge remote .github folder step")
		yaml.WriteString("      - name: Merge remote .github folder\n")
		fmt.Fprintf(yaml, "        uses: %s\n", GetActionPin("actions/github-script"))
		yaml.WriteString("        env:\n")

		// Set repository imports if present
		if len(data.RepositoryImports) > 0 {
			// Convert to JSON array for the script
			repoImportsJSON, err := json.Marshal(data.RepositoryImports)
			if err != nil {
				return fmt.Errorf("failed to marshal repository imports for merge step: %w", err)
			}
			writeYAMLEnv(yaml, "          ", "GH_AW_REPOSITORY_IMPORTS", string(repoImportsJSON))
		}

		// Set agent import spec if present (legacy path)
		if data.AgentFile != "" && data.AgentImportSpec != "" {
			writeYAMLEnv(yaml, "          ", "GH_AW_AGENT_FILE", data.AgentFile)
			writeYAMLEnv(yaml, "          ", "GH_AW_AGENT_IMPORT_SPEC", data.AgentImportSpec)
		}

		yaml.WriteString("        with:\n")
		yaml.WriteString("          script: |\n")
		yaml.WriteString("            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');\n")
		yaml.WriteString("            setupGlobals(core, github, context, exec, io);\n")
		yaml.WriteString("            const { main } = require('${{ runner.temp }}/gh-aw/actions/merge_remote_agent_github_folder.cjs');\n")
		yaml.WriteString("            await main();\n")
	}

	// Add automatic runtime setup steps if needed
	// This detects runtimes from custom steps and MCP configs
	runtimeRequirements := DetectRuntimeRequirements(data)

	// Deduplicate runtime setup steps from custom steps
	// This removes any runtime setup action steps (like actions/setup-go) from custom steps
	// since we're adding them. It also preserves user-customized setup actions and
	// filters those runtimes from requirements so we don't generate duplicates.
	if len(runtimeRequirements) > 0 && data.CustomSteps != "" {
		deduplicatedCustomSteps, filteredRequirements, err := DeduplicateRuntimeSetupStepsFromCustomSteps(data.CustomSteps, runtimeRequirements)
		if err != nil {
			compilerYamlLog.Printf("Warning: failed to deduplicate runtime setup steps: %v", err)
		} else {
			data.CustomSteps = deduplicatedCustomSteps
			runtimeRequirements = filteredRequirements
		}
	}

	// Generate runtime setup steps (after filtering out user-customized ones)
	runtimeSetupSteps := GenerateRuntimeSetupSteps(runtimeRequirements)
	compilerYamlLog.Printf("Detected runtime requirements: %d runtimes, %d setup steps", len(runtimeRequirements), len(runtimeSetupSteps))

	// Decision logic for where to place runtime steps:
	// 1. If we added checkout above (needsCheckout == true), add runtime steps now (after checkout, before custom steps)
	// 2. If custom steps contain checkout, add runtime steps AFTER the first checkout in custom steps
	// 3. Otherwise, add runtime steps now (before custom steps)

	customStepsContainCheckout := data.CustomSteps != "" && ContainsCheckout(data.CustomSteps)
	compilerYamlLog.Printf("Custom steps contain checkout: %t (len(customSteps)=%d)", customStepsContainCheckout, len(data.CustomSteps))

	if needsCheckout || !customStepsContainCheckout {
		// Case 1 or 3: Add runtime steps before custom steps
		// This ensures checkout -> runtime -> custom steps order
		compilerYamlLog.Printf("Adding %d runtime steps before custom steps (needsCheckout=%t, !customStepsContainCheckout=%t)", len(runtimeSetupSteps), needsCheckout, !customStepsContainCheckout)
		for _, step := range runtimeSetupSteps {
			for _, line := range step {
				yaml.WriteString(line + "\n")
			}
		}

	}

	// Create /tmp/gh-aw/ base directory for all temporary files
	// This must be created before custom steps so they can use the temp directory
	yaml.WriteString("      - name: Create gh-aw temp directory\n")
	yaml.WriteString("        run: bash ${RUNNER_TEMP}/gh-aw/actions/create_gh_aw_tmp_dir.sh\n")

	// Configure gh CLI for GitHub Enterprise hosts (*.ghe.com / GHES).
	// This step runs configure_gh_for_ghe.sh which:
	//   1. Detects the GitHub host from GITHUB_SERVER_URL
	//   2. For github.com: exits immediately (no-op)
	//   3. For GHE/GHES: authenticates gh CLI with the enterprise host and sets
	//      GH_HOST=<host> in GITHUB_ENV so every subsequent step in this job
	//      picks up the correct host without manual per-step configuration.
	// Must run after the setup action (so the script is available at ${RUNNER_TEMP}/gh-aw/actions/)
	// and before any custom steps that invoke gh CLI commands.
	yaml.WriteString("      - name: Configure gh CLI for GitHub Enterprise\n")
	yaml.WriteString("        run: bash ${RUNNER_TEMP}/gh-aw/actions/configure_gh_for_ghe.sh\n")
	yaml.WriteString("        env:\n")
	yaml.WriteString("          GH_TOKEN: ${{ github.token }}\n")

	// Start DIFC proxy for pre-agent gh CLI calls (only when guard policies are configured
	// and pre-agent steps with GH_TOKEN are present). The proxy routes gh CLI calls through
	// integrity filtering before the agent runs. Must start before custom steps.
	c.generateStartDIFCProxyStep(yaml, data)

	// Set GH_REPO after the proxy starts so gh CLI can resolve the target repository.
	// start_difc_proxy.sh writes GH_HOST=localhost:18443 to GITHUB_ENV, which causes gh
	// CLI to fail resolving the repository from the git remote. Setting GH_REPO tells gh
	// which repo to target while keeping GH_HOST pointed at the proxy for integrity
	// filtering. Works on both github.com and GHEC.
	c.generateSetGHRepoAfterDIFCProxyStep(yaml, data)

	// Add custom steps if present
	if data.CustomSteps != "" {
		if customStepsContainCheckout && len(runtimeSetupSteps) > 0 {
			// Custom steps contain checkout and we have runtime steps to insert
			// Insert runtime steps after the first checkout step
			compilerYamlLog.Printf("Calling addCustomStepsWithRuntimeInsertion: %d runtime steps to insert after checkout", len(runtimeSetupSteps))
			c.addCustomStepsWithRuntimeInsertion(yaml, data.CustomSteps, runtimeSetupSteps, data.ParsedTools)
		} else {
			// No checkout in custom steps or no runtime steps, just add custom steps as-is
			compilerYamlLog.Printf("Calling addCustomStepsAsIs (customStepsContainCheckout=%t, runtimeStepsCount=%d)", customStepsContainCheckout, len(runtimeSetupSteps))
			c.addCustomStepsAsIs(yaml, data.CustomSteps)
		}
	}

	// Add cache steps if cache configuration is present
	compilerYamlLog.Printf("Generating cache steps for workflow")
	generateCacheSteps(yaml, data, c.verbose)

	// Add cache-memory steps if cache-memory configuration is present
	compilerYamlLog.Printf("Generating cache-memory steps for workflow")
	generateCacheMemorySteps(yaml, data)

	// Add repo-memory clone steps if repo-memory configuration is present
	compilerYamlLog.Printf("Generating repo-memory steps for workflow")
	generateRepoMemorySteps(yaml, data)

	// Configure git credentials for agentic workflows.
	// Git credential configuration requires a .git directory in the workspace, which is only
	// present when the repository was checked out. Skip these steps when checkout is disabled
	// and no custom steps perform a checkout, since git remote set-url origin would fail
	// with "fatal: not a git repository" otherwise.
	needsGitConfig := needsCheckout || customStepsContainCheckout
	compilerYamlLog.Printf("Git credential configuration needed: %t (needsCheckout=%t, customStepsContainCheckout=%t)", needsGitConfig, needsCheckout, customStepsContainCheckout)
	if needsGitConfig {
		gitConfigSteps := c.generateGitConfigurationSteps()
		for _, line := range gitConfigSteps {
			yaml.WriteString(line)
		}
	}

	// Add step to checkout PR branch if the event is pull_request
	c.generatePRReadyForReviewCheckout(yaml, data)

	// Add Node.js setup if the engine requires it and it's not already set up in custom steps
	engine, err := c.getAgenticEngine(data.AI)

	if err != nil {
		return err
	}

	// Ensure MCP gateway defaults are set before generating aw_info.json
	// This is needed so that awmg_version is populated correctly
	if HasMCPServers(data) {
		ensureDefaultMCPGatewayConfig(data)
	}

	// Add engine-specific installation steps (includes Node.js setup and secret validation for npm-based engines)
	installSteps := engine.GetInstallationSteps(data)
	compilerYamlLog.Printf("Adding %d engine installation steps for %s", len(installSteps), engine.GetID())
	for _, step := range installSteps {
		for _, line := range step {
			yaml.WriteString(line + "\n")
		}
	}

	// GH_AW_SAFE_OUTPUTS is now set at job level, no setup step needed

	// Mint the GitHub MCP App token directly in the agent job.
	// The token cannot be passed via job outputs from the activation job because
	// actions/create-github-app-token calls ::add-mask:: on the token, and the
	// GitHub Actions runner silently drops masked values in job outputs (runner v2.308+).
	// By minting the token here, the app-id / private-key secrets are accessed only
	// within this job and the minted token is available as steps.github-mcp-app-token.outputs.token.
	for _, step := range c.generateGitHubMCPAppTokenMintingSteps(data) {
		yaml.WriteString(step)
	}

	// Add GitHub MCP lockdown detection step if needed
	c.generateGitHubMCPLockdownDetectionStep(yaml, data)

	// Add step to parse blocked-users and approval-labels guard variables into JSON arrays
	c.generateParseGuardVarsStep(yaml, data)

	// Stop DIFC proxy before starting the MCP gateway. The proxy must be stopped first
	// to avoid double-filtering: the gateway uses the same guard policy for the agent phase.
	c.generateStopDIFCProxyStep(yaml, data)

	// Add MCP setup
	if err := c.generateMCPSetup(yaml, data.Tools, engine, data); err != nil {
		return fmt.Errorf("failed to generate MCP setup: %w", err)
	}

	// Stop-time safety checks are now handled by a dedicated job (stop_time_check)
	// No longer generated in the main job steps

	// Download activation artifact from activation job (contains aw_info.json and prompt.txt).
	// In workflow_call context, apply the per-invocation prefix to avoid name clashes.
	compilerYamlLog.Print("Adding activation artifact download step")
	activationArtifactName := artifactPrefixExprForDownstreamJob(data) + constants.ActivationArtifactName
	yaml.WriteString("      - name: Download activation artifact\n")
	fmt.Fprintf(yaml, "        uses: %s\n", GetActionPin("actions/download-artifact"))
	yaml.WriteString("        with:\n")
	fmt.Fprintf(yaml, "          name: %s\n", activationArtifactName)
	yaml.WriteString("          path: /tmp/gh-aw\n")

	// Collect artifact paths for unified upload at the end
	var artifactPaths []string
	artifactPaths = append(artifactPaths, "/tmp/gh-aw/aw-prompts/prompt.txt")

	logFileFull := "/tmp/gh-aw/agent-stdio.log"

	// Clean git credentials before executing the agentic engine
	// This ensures that any credentials left on disk by custom steps are removed
	// to prevent the agent from accessing or exfiltrating them
	gitCleanerSteps := c.generateGitCredentialsCleanerStep()
	for _, line := range gitCleanerSteps {
		yaml.WriteString(line)
	}

	// Emit engine config steps (from RenderConfig) before the AI execution step.
	// These steps write runtime config files to disk (e.g. provider/model config files).
	// Most engines return no steps here; only engines that require config files use this.
	if len(data.EngineConfigSteps) > 0 {
		compilerYamlLog.Printf("Adding %d engine config steps for %s", len(data.EngineConfigSteps), engine.GetID())
		for _, step := range data.EngineConfigSteps {
			stepYAML, err := ConvertStepToYAML(step)
			if err != nil {
				return fmt.Errorf("failed to render engine config step: %w", err)
			}
			yaml.WriteString(stepYAML)
		}
	}

	// Add AI execution step using the agentic engine
	compilerYamlLog.Printf("Generating engine execution steps for %s", engine.GetID())
	c.generateEngineExecutionSteps(yaml, data, engine, logFileFull)

	// Add inference access error detection step for Copilot engine
	// This step detects when the Copilot CLI fails due to the token lacking inference access
	// It must run in the main job (not threat detection job) to avoid step ID conflicts
	if _, ok := engine.(*CopilotEngine); ok {
		detectionStep := generateInferenceAccessErrorDetectionStep()
		for _, line := range detectionStep {
			yaml.WriteString(line + "\n")
		}
	}

	// Mark that we've completed agent execution - step order validation starts from here
	compilerYamlLog.Print("Marking agent execution as complete for step order tracking")
	c.stepOrderTracker.MarkAgentExecutionComplete()

	// Regenerate git credentials after agent execution
	// This allows safe-outputs operations (like create_pull_request) to work properly
	// We regenerate the credentials rather than restoring from backup.
	// Only emit these steps when a checkout was performed (requires a .git directory).
	if needsGitConfig {
		gitConfigStepsAfterAgent := c.generateGitConfigurationSteps()
		for _, line := range gitConfigStepsAfterAgent {
			yaml.WriteString(line)
		}
	}

	// Collect firewall logs BEFORE secret redaction so secrets in logs can be redacted
	for _, step := range engine.GetFirewallLogsCollectionStep(data) {
		for _, line := range step {
			yaml.WriteString(line + "\n")
		}
	}

	// Run engine pre-bundle steps to relocate files before secret redaction.
	// This ensures all artifact paths share a common ancestor under /tmp/gh-aw/.
	for _, step := range engine.GetPreBundleSteps(data) {
		for _, line := range step {
			yaml.WriteString(line + "\n")
		}
	}

	// Stop MCP gateway after agent execution and before secret redaction
	// This ensures the gateway process is properly cleaned up
	// The MCP gateway is always enabled, even when agent sandbox is disabled
	c.generateStopMCPGateway(yaml, data)

	// Add secret redaction step BEFORE any artifact uploads
	// This ensures all artifacts are scanned for secrets before being uploaded
	c.generateSecretRedactionStep(yaml, yaml.String(), data)

	// Append the agent step summary to the real $GITHUB_STEP_SUMMARY after secrets are redacted.
	// The agent writes its GITHUB_STEP_SUMMARY content to AgentStepSummaryPath (a file inside
	// /tmp/gh-aw/ that is reachable in both AWF sandbox and non-sandbox modes).
	// secret redaction already scanned this file, so it is safe to append.
	c.generateAgentStepSummaryAppend(yaml)

	// Add output collection step only if safe-outputs feature is used (GH_AW_SAFE_OUTPUTS functionality)
	if data.SafeOutputs != nil {
		c.generateOutputCollectionStep(yaml, data)
	}

	// Merge engine-declared output files into the unified artifact instead of creating a
	// separate agent_outputs artifact. The cleanup step is still generated so workspace files
	// are removed after collection.
	if enginePaths := getEngineArtifactPaths(engine); len(enginePaths) > 0 {
		artifactPaths = append(artifactPaths, enginePaths...)
		c.generateEngineOutputCleanup(yaml, engine)
	}

	// Extract and upload squid access logs (if any proxy tools were used)
	c.generateExtractAccessLogs(yaml, data.Tools)
	c.generateUploadAccessLogs(yaml, data.Tools)

	// Collect MCP logs path if any MCP tools were used
	artifactPaths = append(artifactPaths, "/tmp/gh-aw/mcp-logs/")

	// Collect DIFC proxy logs (proxy-tls certs + container stderr) when proxy was injected
	artifactPaths = append(artifactPaths, difcProxyLogPaths(data)...)

	// Collect MCPScripts logs path if mcp-scripts is enabled
	if IsMCPScriptsEnabled(data.MCPScripts, data) {
		artifactPaths = append(artifactPaths, "/tmp/gh-aw/mcp-scripts/logs/")
	}

	// parse agent logs for GITHUB_STEP_SUMMARY
	c.generateLogParsing(yaml, engine)

	// parse mcp-scripts logs for GITHUB_STEP_SUMMARY (if mcp-scripts is enabled)
	if IsMCPScriptsEnabled(data.MCPScripts, data) {
		c.generateMCPScriptsLogParsing(yaml)
	}

	// parse MCP gateway logs for GITHUB_STEP_SUMMARY
	// The MCP gateway is always enabled, even when agent sandbox is disabled
	c.generateMCPGatewayLogParsing(yaml)

	// Add firewall log parsing and dedicated audit upload for all firewall-enabled engines.
	// This replaces the previous per-engine blocks (Copilot, Codex, Claude) and extends
	// support to all engines (including Gemini) so every agentic workflow uploads audit logs.
	if isFirewallEnabled(data) {
		firewallLogParsing := generateFirewallLogParsingStep(data.Name)
		for _, line := range firewallLogParsing {
			yaml.WriteString(line + "\n")
		}
	}

	// Parse token-usage.jsonl and append to step summary (requires AWF v0.25.8+)
	if isFirewallEnabled(data) {
		c.generateTokenUsageSummary(yaml)
		// Include the aggregated agent_usage.json in the agent artifact so third-party
		// tools can consume structured token data without parsing the step summary.
		artifactPaths = append(artifactPaths, "/tmp/gh-aw/"+constants.TokenUsageFilename)
	}

	// Synthesize a compact observability section from runtime artifacts when OTLP is enabled.
	c.generateObservabilitySummary(yaml, data)

	// Collect agent stdio logs path for unified upload
	artifactPaths = append(artifactPaths, logFileFull)

	// Collect agent-generated files path for unified upload
	// This directory is used by workflows that instruct the agent to write files
	// (e.g., smoke-claude status summaries)
	artifactPaths = append(artifactPaths, "/tmp/gh-aw/agent/")

	// Collect GitHub API rate-limit log for observability.
	// Written by github_rate_limit_logger.cjs during REST API calls.
	artifactPaths = append(artifactPaths, "/tmp/gh-aw/"+constants.GithubRateLimitsFilename)

	// Collect safe outputs and agent output paths for the unified artifact.
	// These were previously uploaded as separate safe-output and agent-output artifacts.
	if data.SafeOutputs != nil {
		// Raw safe-output NDJSON (copied to /tmp/gh-aw/ by generateOutputCollectionStep)
		artifactPaths = append(artifactPaths, "/tmp/gh-aw/"+constants.SafeOutputsFilename)
		// Processed agent output JSON produced by collect_ndjson_output.cjs
		artifactPaths = append(artifactPaths, "/tmp/gh-aw/"+constants.AgentOutputFilename)

		// Write a minimal agent_output.json placeholder when the engine fails before
		// producing any safe outputs, so downstream safe_outputs and conclusion jobs
		// receive a valid (empty) JSON file instead of an ENOENT error.
		// The placeholder is only written if the engine did not already write the file.
		c.generateAgentOutputPlaceholderStep(yaml)
	}

	// Add post-execution cleanup step for Copilot engine
	if copilotEngine, ok := engine.(*CopilotEngine); ok {
		cleanupStep := copilotEngine.GetCleanupStep(data)
		for _, line := range cleanupStep {
			yaml.WriteString(line + "\n")
		}
	}

	// Add repo-memory artifact upload to save state for push job
	generateRepoMemoryArtifactUpload(yaml, data)

	// Add cache-memory git commit steps (after agent execution, before validation)
	// This commits agent-written changes to the current integrity branch.
	generateCacheMemoryGitCommitSteps(yaml, data)

	// Add cache-memory validation (after agent execution)
	// This validates file types before cache is saved or uploaded
	generateCacheMemoryValidation(yaml, data)

	// Add cache-memory artifact upload (after agent execution)
	// This ensures artifacts are uploaded after the agent has finished modifying the cache
	generateCacheMemoryArtifactUpload(yaml, data)

	// Add safe-outputs assets artifact upload (after agent execution)
	// This creates a separate artifact for assets that will be downloaded by upload_assets job
	generateSafeOutputsAssetsArtifactUpload(yaml, data)

	// Add safe-outputs upload-artifact staging upload (after agent execution)
	// This creates a separate artifact for files the model staged for artifact upload,
	// to be downloaded and processed by the upload_artifact job
	generateSafeOutputsArtifactStagingUpload(yaml, data)

	// Collect git patch path if safe-outputs with PR operations is configured
	// NOTE: Git patch generation has been moved to the safe-outputs MCP server
	// The patch is now generated when create_pull_request or push_to_pull_request_branch
	// tools are called, providing immediate error feedback if no changes are present.
	// Include patches in the artifact when:
	// 1. Safe outputs needs them for checkout (non-staged create_pull_request/push_to_pull_request_branch)
	// 2. Threat detection is enabled (detection job needs patches for security analysis, even when the
	//    safe-output handler is staged and doesn't need checkout itself)
	threatDetectionNeedsPatches := IsDetectionJobEnabled(data.SafeOutputs)
	if usesPatchesAndCheckouts(data.SafeOutputs) || threatDetectionNeedsPatches {
		artifactPaths = append(artifactPaths, "/tmp/gh-aw/aw-*.patch")
		// Bundle files are generated when patch-format: bundle is configured.
		// Both formats use the same download path in the safe_outputs job, so
		// include the bundle glob unconditionally alongside the patch glob.
		// The artifact upload step already sets if-no-files-found: ignore, so
		// this is safe even when no bundle files exist.
		artifactPaths = append(artifactPaths, "/tmp/gh-aw/aw-*.bundle")
	}

	// Add post-steps (if any) after AI execution
	c.generatePostSteps(yaml, data)

	// Generate single unified artifact upload with all collected paths.
	// In workflow_call context, apply the per-invocation prefix to avoid name clashes.
	agentArtifactPrefix := artifactPrefixExprForDownstreamJob(data)
	c.generateUnifiedArtifactUpload(yaml, artifactPaths, agentArtifactPrefix)

	// Upload firewall audit logs as a dedicated artifact so users can inspect network
	// activity, policy decisions, and blocked domains after the run (AWF v0.25.0+).
	if isFirewallEnabled(data) {
		c.generateFirewallAuditLogsUploadStep(yaml, agentArtifactPrefix)
	}

	// Add GitHub MCP app token invalidation step if configured (runs always, even on failure)
	c.generateGitHubMCPAppTokenInvalidationStep(yaml, data)

	// Add checkout app token invalidation steps if configured (runs always, even on failure)
	if checkoutMgr.HasAppAuth() {
		compilerYamlLog.Print("Generating checkout app token invalidation steps")
		invalidationSteps := checkoutMgr.GenerateCheckoutAppTokenInvalidationSteps(c)
		for _, step := range invalidationSteps {
			yaml.WriteString(step)
		}
	}

	// In dev mode the setup action is referenced via a local path (./actions/setup), so its files
	// live in the workspace. When a checkout: entry targets an external repository without a path
	// (e.g. "checkout: [{repository: owner/other-repo}]"), actions/checkout replaces the workspace
	// root with the external repository content, removing the actions/setup directory.
	// Without restoring it, the runner's post-step for Setup Scripts would fail with
	// "Can't find 'action.yml', 'action.yaml' or 'Dockerfile' under .../actions/setup".
	// We add a restore checkout step (if: always()) as the final step so the post-step
	// can always find action.yml and complete its /tmp/gh-aw cleanup.
	if c.actionMode.IsDev() && checkoutMgr.HasExternalRootCheckout() {
		yaml.WriteString(c.generateRestoreActionsSetupStep())
		compilerYamlLog.Print("Added restore actions folder step to agent job (dev mode with external root checkout)")
	}

	// Validate step ordering - this is a compiler check to ensure security
	if err := c.stepOrderTracker.ValidateStepOrdering(); err != nil {
		// This is a compiler bug if validation fails
		return fmt.Errorf("step ordering validation failed: %w", err)
	}
	return nil
}

// addCustomStepsAsIs adds custom steps without modification
func (c *Compiler) addCustomStepsAsIs(yaml *strings.Builder, customSteps string) {
	// Remove "steps:" line and adjust indentation
	lines := strings.Split(customSteps, "\n")
	if len(lines) > 1 {
		for _, line := range lines[1:] {
			// Skip empty lines
			if strings.TrimSpace(line) == "" {
				yaml.WriteString("\n")
				continue
			}

			// Simply add 6 spaces for job context indentation
			yaml.WriteString("      " + line + "\n")
		}
	}
}

// addCustomStepsWithRuntimeInsertion adds custom steps and inserts runtime steps after the first checkout
func (c *Compiler) addCustomStepsWithRuntimeInsertion(yaml *strings.Builder, customSteps string, runtimeSetupSteps []GitHubActionStep, tools *ToolsConfig) {
	// Remove "steps:" line and adjust indentation
	lines := strings.Split(customSteps, "\n")
	if len(lines) <= 1 {
		return
	}

	insertedRuntime := false
	i := 1 // Start from index 1 to skip "steps:" line

	for i < len(lines) {
		line := lines[i]

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			yaml.WriteString("\n")
			i++
			continue
		}

		// Add the line with proper indentation
		yaml.WriteString("      " + line + "\n")

		// Check if this line starts a step with "- name:" or "- uses:"
		trimmed := strings.TrimSpace(line)
		isStepStart := strings.HasPrefix(trimmed, "- name:") || strings.HasPrefix(trimmed, "- uses:")

		if isStepStart && !insertedRuntime {
			// This is the start of a step, check if it's a checkout step
			isCheckoutStep := false

			// Look ahead to find "uses:" line with "checkout"
			for j := i + 1; j < len(lines); j++ {
				nextLine := lines[j]
				nextTrimmed := strings.TrimSpace(nextLine)

				// Stop if we hit the next step
				if strings.HasPrefix(nextTrimmed, "- name:") || strings.HasPrefix(nextTrimmed, "- uses:") {
					break
				}

				// Check if this is a uses line with checkout
				if strings.Contains(nextTrimmed, "uses:") && strings.Contains(nextTrimmed, "checkout") {
					isCheckoutStep = true
					break
				}
			}

			if isCheckoutStep {
				// This is a checkout step, copy all its lines until the next step
				i++
				for i < len(lines) {
					nextLine := lines[i]
					nextTrimmed := strings.TrimSpace(nextLine)

					// Stop if we hit the next step
					if strings.HasPrefix(nextTrimmed, "- name:") || strings.HasPrefix(nextTrimmed, "- uses:") {
						break
					}

					// Add the line
					if nextTrimmed == "" {
						yaml.WriteString("\n")
					} else {
						yaml.WriteString("      " + nextLine + "\n")
					}
					i++
				}

				// Now insert runtime steps after the checkout step
				compilerYamlLog.Printf("Inserting %d runtime setup steps after checkout in custom steps", len(runtimeSetupSteps))
				for _, step := range runtimeSetupSteps {
					for _, stepLine := range step {
						yaml.WriteString(stepLine + "\n")
					}
				}

				insertedRuntime = true
				continue // Continue with the next iteration (i is already advanced)
			}
		}

		i++
	}
}

// generateRepositoryImportCheckouts generates checkout steps for repository imports
// Each repository is checked out into a temporary folder at .github/aw/imports/<owner>-<repo>-<sanitized-ref>
// relative to GITHUB_WORKSPACE. This allows the merge script to copy files from pre-checked-out folders instead of doing git operations
func (c *Compiler) generateRepositoryImportCheckouts(yaml *strings.Builder, repositoryImports []string) {
	for _, repoImport := range repositoryImports {
		compilerYamlLog.Printf("Generating checkout step for repository import: %s", repoImport)

		// Parse the import spec to extract owner, repo, and ref
		// Format: owner/repo@ref or owner/repo
		owner, repo, ref := parseRepositoryImportSpec(repoImport)
		if owner == "" || repo == "" {
			compilerYamlLog.Printf("Warning: failed to parse repository import: %s", repoImport)
			continue
		}

		// Generate a sanitized directory name for the checkout
		// Use a consistent format: owner-repo-ref
		// NOTE: Path must be relative to GITHUB_WORKSPACE for actions/checkout@v6
		sanitizedRef := sanitizeRefForPath(ref)
		checkoutPath := fmt.Sprintf(".github/aw/imports/%s-%s-%s", owner, repo, sanitizedRef)

		// Generate the checkout step
		fmt.Fprintf(yaml, "      - name: Checkout repository import %s/%s@%s\n", owner, repo, ref)
		fmt.Fprintf(yaml, "        uses: %s\n", GetActionPin("actions/checkout"))
		yaml.WriteString("        with:\n")
		fmt.Fprintf(yaml, "          repository: %s/%s\n", owner, repo)
		fmt.Fprintf(yaml, "          ref: %s\n", ref)
		fmt.Fprintf(yaml, "          path: %s\n", checkoutPath)
		yaml.WriteString("          sparse-checkout: |\n")
		yaml.WriteString("            .github/\n")
		yaml.WriteString("          persist-credentials: false\n")

		compilerYamlLog.Printf("Added checkout step: %s/%s@%s -> %s", owner, repo, ref, checkoutPath)
	}
}

// parseRepositoryImportSpec parses a repository import specification
// Format: owner/repo@ref or owner/repo (defaults to "main" if no ref)
// Returns: owner, repo, ref
func parseRepositoryImportSpec(importSpec string) (owner, repo, ref string) {
	// Remove section reference if present (file.md#Section)
	cleanSpec := importSpec
	if before, _, ok := strings.Cut(importSpec, "#"); ok {
		cleanSpec = before
	}

	// Split on @ to get path and ref
	parts := strings.Split(cleanSpec, "@")
	pathPart := parts[0]
	ref = "main" // default ref
	if len(parts) > 1 {
		ref = parts[1]
	}

	// Parse path: owner/repo
	slashParts := strings.Split(pathPart, "/")
	if len(slashParts) != 2 {
		return "", "", ""
	}

	owner = slashParts[0]
	repo = slashParts[1]

	return owner, repo, ref
}

// generateLegacyAgentImportCheckout generates a checkout step for legacy agent imports
// Legacy format: owner/repo/path/to/file.md@ref
// This checks out the entire repository (not just .github folder) since the file could be anywhere
func (c *Compiler) generateLegacyAgentImportCheckout(yaml *strings.Builder, agentImportSpec string) {
	compilerYamlLog.Printf("Generating checkout step for legacy agent import: %s", agentImportSpec)

	// Parse the import spec to extract owner, repo, and ref
	owner, repo, ref := parseRepositoryImportSpec(agentImportSpec)
	if owner == "" || repo == "" {
		compilerYamlLog.Printf("Warning: failed to parse legacy agent import spec: %s", agentImportSpec)
		return
	}

	// Generate a sanitized directory name for the checkout
	sanitizedRef := sanitizeRefForPath(ref)
	checkoutPath := fmt.Sprintf("/tmp/gh-aw/repo-imports/%s-%s-%s", owner, repo, sanitizedRef)

	// Generate the checkout step
	fmt.Fprintf(yaml, "      - name: Checkout agent import %s/%s@%s\n", owner, repo, ref)
	fmt.Fprintf(yaml, "        uses: %s\n", GetActionPin("actions/checkout"))
	yaml.WriteString("        with:\n")
	fmt.Fprintf(yaml, "          repository: %s/%s\n", owner, repo)
	fmt.Fprintf(yaml, "          ref: %s\n", ref)
	fmt.Fprintf(yaml, "          path: %s\n", checkoutPath)
	yaml.WriteString("          sparse-checkout: |\n")
	yaml.WriteString("            .github/\n")
	yaml.WriteString("          persist-credentials: false\n")

	compilerYamlLog.Printf("Added legacy agent checkout step: %s/%s@%s -> %s", owner, repo, ref, checkoutPath)
}

// generateDevModeCLIBuildSteps generates the steps needed to build the gh-aw CLI and Docker image in dev mode
// These steps are injected after checkout in dev mode to create a locally built Docker image that includes
// the gh-aw binary and all dependencies. The agentic-workflows MCP server uses this image instead of alpine:latest.
//
// The build process:
// 1. Setup Go using go.mod version
// 2. Build the gh-aw CLI binary for linux/amd64 (since it runs in a Linux container)
// 3. Setup Docker Buildx for advanced build features
// 4. Build Docker image and tag it as localhost/gh-aw:dev
//
// The built image is used by the agentic-workflows MCP server configuration (see mcp_config_builtin.go)
func (c *Compiler) generateDevModeCLIBuildSteps(yaml *strings.Builder) {
	compilerYamlLog.Print("Generating dev mode CLI build steps")

	// Step 1: Setup Go for building the CLI
	yaml.WriteString("      - name: Setup Go for CLI build\n")
	fmt.Fprintf(yaml, "        uses: %s\n", GetActionPin("actions/setup-go"))
	yaml.WriteString("        with:\n")
	yaml.WriteString("          go-version-file: go.mod\n")
	yaml.WriteString("          cache: true\n")

	// Step 2: Build CLI binary for linux/amd64
	// Use the standard build command from CI/Makefile (not release build)
	// CGO_ENABLED=0 for static linking (required for Alpine containers)
	yaml.WriteString("      - name: Build gh-aw CLI\n")
	yaml.WriteString("        run: |\n")
	yaml.WriteString("          echo \"Building gh-aw CLI for linux/amd64...\"\n")
	yaml.WriteString("          mkdir -p dist\n")
	yaml.WriteString("          VERSION=$(git describe --tags --always --dirty)\n")
	yaml.WriteString("          CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \\\n")
	yaml.WriteString("            -ldflags \"-s -w -X main.version=${VERSION}\" \\\n")
	yaml.WriteString("            -o dist/gh-aw-linux-amd64 \\\n")
	yaml.WriteString("            ./cmd/gh-aw\n")
	yaml.WriteString("          # Copy binary to root for direct execution in user-defined steps\n")
	yaml.WriteString("          cp dist/gh-aw-linux-amd64 ./gh-aw\n")
	yaml.WriteString("          chmod +x ./gh-aw\n")
	yaml.WriteString("          echo \"✓ Built gh-aw CLI successfully\"\n")

	// Step 3: Setup Docker Buildx
	yaml.WriteString("      - name: Setup Docker Buildx\n")
	fmt.Fprintf(yaml, "        uses: %s\n", GetActionPin("docker/setup-buildx-action"))

	// Step 4: Build Docker image
	// Use the Dockerfile at the repository root which expects BINARY build arg
	yaml.WriteString("      - name: Build gh-aw Docker image\n")
	fmt.Fprintf(yaml, "        uses: %s\n", GetActionPin("docker/build-push-action"))
	yaml.WriteString("        with:\n")
	yaml.WriteString("          context: .\n")
	yaml.WriteString("          platforms: linux/amd64\n")
	yaml.WriteString("          push: false\n")
	yaml.WriteString("          load: true\n")
	yaml.WriteString("          tags: localhost/gh-aw:dev\n")
	yaml.WriteString("          build-args: |\n")
	yaml.WriteString("            BINARY=dist/gh-aw-linux-amd64\n")
}
