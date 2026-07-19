package workflow

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
)

// collectArtifactPaths gathers all paths for the unified artifact upload.
// It starts from the initial paths already accumulated by generateAgentRunSteps and appends
// engine-declared output paths, log directories, observability files, safe-outputs files,
// patch/bundle paths, and firewall audit paths.
func (c *Compiler) collectArtifactPaths(data *WorkflowData, engine CodingAgentEngine, logFileFull string, initialPaths []string) []string {
	paths := initialPaths

	paths = append(paths, c.baseArtifactPaths(data, engine, logFileFull)...)
	paths = append(paths, safeOutputArtifactPaths(data)...)
	paths = append(paths, patchArtifactPaths(data)...)
	paths = append(paths, firewallArtifactPaths(data)...)

	// For ARC/DinD, rewrite all /tmp/gh-aw/ paths to ${{ runner.temp }}/gh-aw/ so
	// the artifact upload has a single root. A consolidation step (emitted before upload)
	// copies the files from /tmp/gh-aw/ to the runner.temp location. See gh-aw#34896 Bug B.
	if isArcDindTopology(data) {
		paths = rewriteTmpGhAwPathsForArcDind(paths)
	}

	compilerYamlLog.Printf("Collected %d artifact path(s) for unified agent upload", len(paths))
	return paths
}

func (c *Compiler) baseArtifactPaths(data *WorkflowData, engine CodingAgentEngine, logFileFull string) []string {
	paths := append([]string{}, getEngineArtifactPaths(engine)...)
	paths = append(paths, constants.TmpMcpLogsDir)
	paths = append(paths, difcProxyLogPaths(data)...)
	if IsMCPScriptsEnabled(data.MCPScripts) {
		paths = append(paths, constants.TmpMcpScriptsLogsDir)
	}
	if isFirewallEnabled(data) {
		paths = append(paths, constants.TmpGhAwDirSlash+constants.TokenUsageFilename)
	}
	paths = append(paths,
		logFileFull,
		constants.PreAgentAuditFilePath,
		constants.TmpGhAwAgentDir,
		constants.TmpGhAwDirSlash+constants.GithubRateLimitsFilename,
	)
	if isOTLPEnabled(data) {
		paths = append(paths, constants.TmpGhAwDirSlash+constants.OtelJsonlFilename)
		paths = append(paths, constants.TmpGhAwDirSlash+constants.OtlpExportErrorsFilename)
	}
	return paths
}

func safeOutputArtifactPaths(data *WorkflowData) []string {
	if data.SafeOutputs == nil {
		return nil
	}
	paths := []string{
		constants.TmpGhAwDirSlash + constants.SafeOutputsFilename,
		constants.TmpGhAwDirSlash + constants.AgentOutputFilename,
	}
	if data.SafeOutputs.CommentMemory != nil {
		paths = append(paths, constants.TmpCommentMemoryDir)
	}
	return paths
}

func patchArtifactPaths(data *WorkflowData) []string {
	if !usesPatchesAndCheckouts(data.SafeOutputs) && !IsDetectionJobEnabled(data.SafeOutputs) {
		return nil
	}
	return []string{constants.TmpAwPatchGlob, constants.TmpAwBundleGlob}
}

func firewallArtifactPaths(data *WorkflowData) []string {
	if !isFirewallEnabled(data) {
		return nil
	}
	if isArcDindTopology(data) {
		return []string{
			constants.AWFConfigFilePathExpr,
			constants.AWFProxyLogsDirExpr + "/",
			constants.AWFAuditDirExpr + "/",
			constants.AWFReflectFilePathExpr,
		}
	}
	return []string{
		constants.AWFConfigFilePath,
		constants.AWFProxyLogsDir + "/",
		constants.AWFAuditDir + "/",
		constants.AWFReflectFilePath,
	}
}

// generateSummarySteps emits all GITHUB_STEP_SUMMARY log-parsing steps for the agent job.
// It covers agent log parsing, MCP scripts, MCP gateway, firewall logs, token usage,
// AWF reflect summary, and observability summary.
func (c *Compiler) generateSummarySteps(yaml *strings.Builder, data *WorkflowData, engine CodingAgentEngine) {
	// Parse agent logs for GITHUB_STEP_SUMMARY
	c.generateLogParsing(yaml, data, engine)

	// Parse mcp-scripts logs for GITHUB_STEP_SUMMARY (if mcp-scripts is enabled)
	if IsMCPScriptsEnabled(data.MCPScripts) {
		c.generateMCPScriptsLogParsing(yaml, data)
	}

	// Parse MCP gateway logs for GITHUB_STEP_SUMMARY.
	// The MCP gateway is always enabled, even when agent sandbox is disabled.
	c.generateMCPGatewayLogParsing(yaml, data)

	// Add firewall log parsing for all firewall-enabled engines.
	// This replaces the previous per-engine blocks (Copilot, Codex, Claude) and extends
	// support to all engines (including Gemini) so every agentic workflow uploads audit logs.
	if isFirewallEnabled(data) {
		firewallLogParsing := generateFirewallLogParsingStep(data.Name, data)
		for _, line := range firewallLogParsing {
			yaml.WriteString(line)
			yaml.WriteByte('\n')
		}
	}

	// Parse token-usage.jsonl and append to step summary (requires AWF v0.25.8+)
	if isFirewallEnabled(data) {
		c.generateTokenUsageSummary(yaml, data)
	}

	// Append AWF API proxy reflection data (available endpoints and models) to step summary.
	// This data is fetched from the /reflect endpoint by copilot_harness.cjs before the
	// agent exits and persisted to /tmp/gh-aw/awf-reflect.json.
	if isFirewallEnabled(data) {
		c.generateAWFReflectSummary(yaml, data)
	}

	// Synthesize a compact observability section from runtime artifacts when OTLP is enabled.
	c.generateObservabilitySummary(yaml, data)
}

// generatePostAgentCollectionAndUpload orchestrates the post-agent phase:
// engine output cleanup, access log collection, artifact path accumulation via collectArtifactPaths,
// step-summary generation via generateSummarySteps, safe-outputs/memory/staging artifact uploads,
// post-steps, the unified artifact upload, token invalidation, dev-mode actions restore,
// and step-order validation.
func (c *Compiler) generatePostAgentCollectionAndUpload(yaml *strings.Builder, data *WorkflowData, engine CodingAgentEngine, artifactPaths []string, logFileFull string, checkoutMgr *CheckoutManager) error {
	compilerYamlLog.Print("Generating post-agent collection and upload steps")

	c.generatePostAgentCleanupAndLogs(yaml, data, engine)
	artifactPaths = c.collectArtifactPaths(data, engine, logFileFull, artifactPaths)
	c.generateSummarySteps(yaml, data, engine)
	c.generatePostAgentUploadsAndPostSteps(yaml, data)
	c.generateUnifiedAgentArtifactUpload(yaml, data, artifactPaths)
	c.generateAgentRestoreActionsSetupStep(yaml, checkoutMgr)
	if err := c.stepOrderTracker.ValidateStepOrdering(); err != nil {
		return fmt.Errorf("step ordering validation failed: %w", err)
	}
	return nil
}

func (c *Compiler) generatePostAgentCleanupAndLogs(yaml *strings.Builder, data *WorkflowData, engine CodingAgentEngine) {
	if len(getEngineArtifactPaths(engine)) > 0 {
		c.generateEngineOutputCleanup(yaml, engine)
	}
	c.generateExtractAccessLogs(yaml, data.Tools)
	c.generateUploadAccessLogs(yaml, data.Tools)
	if data.SafeOutputs != nil {
		c.generateAgentOutputPlaceholderStep(yaml)
	}
	if copilotEngine, ok := engine.(*CopilotEngine); ok {
		for _, line := range copilotEngine.GetCleanupStep(data) {
			yaml.WriteString(line)
			yaml.WriteByte('\n')
		}
	}
}

func (c *Compiler) generatePostAgentUploadsAndPostSteps(yaml *strings.Builder, data *WorkflowData) {
	generateRepoMemoryArtifactUpload(yaml, data, c.getActionPin)
	generateCacheMemoryGitCommitSteps(yaml, data)
	generateCacheMemoryValidation(yaml, data)
	generateCacheMemoryArtifactUpload(yaml, data, c.getActionPin)
	generateSafeOutputsAssetsArtifactUpload(yaml, data, c.getActionPin)
	generateSafeOutputsArtifactStagingUpload(yaml, data, c.getActionPin)
	c.generatePostSteps(yaml, data)
	if isArcDindTopology(data) {
		c.generateArcDindArtifactConsolidationStep(yaml)
	}
}

func (c *Compiler) generateUnifiedAgentArtifactUpload(yaml *strings.Builder, data *WorkflowData, artifactPaths []string) {
	agentArtifactPrefix := artifactPrefixExprForDownstreamJob(data)
	compilerYamlLog.Printf("Emitting unified agent artifact upload with %d path(s)", len(artifactPaths))
	c.generateUnifiedArtifactUpload(yaml, artifactPaths, agentArtifactPrefix)
}

func (c *Compiler) generateAgentRestoreActionsSetupStep(yaml *strings.Builder, checkoutMgr *CheckoutManager) {
	if c.actionMode.IsDev() && checkoutMgr.HasExternalRootCheckout() {
		yaml.WriteString(c.generateRestoreActionsSetupStep())
		compilerYamlLog.Print("Added restore actions folder step to agent job (dev mode with external root checkout)")
	}
}
