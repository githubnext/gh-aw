// Package workflow - inline (non-external) engine execution step for threat detection.
package workflow

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/workflow/compilerenv"
)

// buildDetectionEngineExecutionStep creates the engine execution step for inline threat detection.
// It uses the same agentic engine already installed in the agent job, but runs it through
// sandbox.agent (AWF) with no allowed domains (network fully blocked) and no MCP configured.
func (c *Compiler) buildDetectionEngineExecutionStep(data *WorkflowData) []string {
	// Check if threat detection has engine explicitly disabled
	if data.SafeOutputs != nil && data.SafeOutputs.ThreatDetection != nil {
		if data.SafeOutputs.ThreatDetection.EngineDisabled {
			// Engine explicitly disabled with engine: false
			return []string{
				"      # AI engine disabled for threat detection (engine: false)\n",
			}
		}
	}

	// Determine which engine to use: threat detection engine from frontmatter,
	// otherwise main engine.
	engineSetting := c.getThreatDetectionEngineID(data)

	engineConfig := data.EngineConfig
	hasThreatDetectionEngineConfig := data.SafeOutputs != nil &&
		data.SafeOutputs.ThreatDetection != nil &&
		data.SafeOutputs.ThreatDetection.EngineConfig != nil
	if hasThreatDetectionEngineConfig {
		engineConfig = data.SafeOutputs.ThreatDetection.EngineConfig
	}
	// Preserve the original engine identity before Pi is normalized to Copilot for
	// detection. Precedence matches runtime engine resolution: explicit
	// threat-detection.engine.id overrides the main engine config, which overrides
	// the legacy top-level AI field.
	originalEngineID := data.AI
	if data.EngineConfig != nil && data.EngineConfig.ID != "" {
		originalEngineID = data.EngineConfig.ID
	}
	if hasThreatDetectionEngineConfig && data.SafeOutputs.ThreatDetection.EngineConfig.ID != "" {
		originalEngineID = data.SafeOutputs.ThreatDetection.EngineConfig.ID
	}

	// Get the engine instance
	engine, err := c.getAgenticEngine(engineSetting)
	if err != nil {
		return []string{"      # Engine not found, skipping execution\n"}
	}

	// Build a detection engine config inheriting ID, Model, Version, Env, Config, Args, APITarget.
	// MaxTurns, Concurrency, UserAgent, Firewall, Agent, and MaxAICredits are intentionally
	// omitted — MaxAICredits is set independently below from safe-outputs.threat-detection
	// so the detection budget is always resolved from its own default expression rather than
	// silently reusing the main agent budget.
	detectionEngineConfig := engineConfig
	if detectionEngineConfig == nil {
		detectionEngineConfig = &EngineConfig{ID: engineSetting}
	} else {
		detectionEngineConfig = &EngineConfig{
			ID:            detectionEngineConfig.ID,
			Model:         detectionEngineConfig.Model,
			Version:       detectionEngineConfig.Version,
			Env:           detectionEngineConfig.Env,
			Config:        detectionEngineConfig.Config,
			Args:          detectionEngineConfig.Args,
			APITarget:     detectionEngineConfig.APITarget,
			HarnessScript: detectionEngineConfig.HarnessScript,
			Driver:        detectionEngineConfig.Driver,
		}
	}
	if detectionEngineConfig.ID == "" {
		detectionEngineConfig.ID = engineSetting
	}
	if data.SafeOutputs != nil && data.SafeOutputs.ThreatDetection != nil && data.SafeOutputs.ThreatDetection.MaxAICredits != 0 {
		detectionEngineConfig.MaxAICredits = data.SafeOutputs.ThreatDetection.MaxAICredits
	}

	// Apply enterprise and engine default detection models when no model was explicitly configured.
	// GetDefaultDetectionModel() returns a cost-effective model optimised for detection
	// (e.g. "gpt-5.1-codex-mini" for Copilot). Other engines return "" (no default).
	// This was accidentally removed in commit a93e36ea4 while fixing engine.agent propagation.
	if detectionEngineConfig.Model == "" {
		if defaultModel := compilerenv.ResolveDefaultDetectionModel(""); defaultModel != "" {
			detectionEngineConfig.Model = defaultModel
		} else if defaultModel := engine.GetDefaultDetectionModel(); defaultModel != "" {
			detectionEngineConfig.Model = defaultModel
		}
	}

	// Inherit APITarget from the main engine config for GHE/custom endpoints if not already set.
	// This ensures the threat detection AWF invocation receives the same --copilot-api-target
	// and GHE-specific domains in --allow-domains as the main agent AWF invocation.
	if detectionEngineConfig.APITarget == "" && data.EngineConfig != nil && data.EngineConfig.APITarget != "" {
		detectionEngineConfig.APITarget = data.EngineConfig.APITarget
	}
	if engineSetting == "copilot" && originalEngineID == "pi" {
		// Pi requires provider/model syntax (for example "copilot/gpt-5.4"), but the
		// Copilot CLI expects only the model ID. extractPiModelID preserves bare model
		// names unchanged, so empty or already-normalized values keep their current
		// fallback behavior while provider-scoped Pi models become Copilot-compatible.
		detectionEngineConfig.Model = extractPiModelID(detectionEngineConfig.Model)
	}

	// Create minimal WorkflowData for threat detection.
	// SandboxConfig with AWF enabled ensures the engine runs inside the firewall.
	// NetworkPermissions.Allowed preserves only literal user-specified domains when Copilot
	// BYOK is enabled so secret-backed provider URLs can still be paired with an explicit
	// provider hostname in network.allowed without re-opening whole ecosystem allow-lists.
	// No MCP servers are configured for detection.
	// bash: ["*"] allows all shell commands — AWF's network firewall is the primary
	// constraint, so restricting individual bash commands inside the sandbox adds friction
	// without meaningful security benefit.
	// RunnerConfig is propagated from the main workflow data so that arc-dind topology
	// handling (daemon-visible Copilot staging step + daemon-visible spawn path) applies
	// to the detection job the same way it applies to the agent job.
	threatDetectionData := &WorkflowData{
		Tools: map[string]any{
			"bash": []any{"*"},
		},
		SafeOutputs:       nil,
		EngineConfig:      detectionEngineConfig,
		AI:                engineSetting,
		Features:          data.Features,
		Permissions:       data.Permissions,
		CachedPermissions: data.CachedPermissions,
		IsDetectionRun:    true,              // Mark as detection run for phase tagging
		RunnerConfig:      data.RunnerConfig, // propagate runner.topology (e.g. arc-dind) to the detection job
		NetworkPermissions: &NetworkPermissions{
			Allowed: getThreatDetectionAdditionalAllowedDomains(data),
		},
		SandboxConfig: &SandboxConfig{
			Agent: &AgentSandboxConfig{
				Type: SandboxTypeAWF,
			},
		},
	}

	var steps []string

	// Install the engine in the detection job. The detection job runs on a separate fresh
	// runner where the agent's installed tools are not available, so we must install them here.
	installSteps := engine.GetInstallationSteps(threatDetectionData)

	// Ensure node is on PATH when the engine's execution wraps the CLI with a harness
	// script (see engineRequiresNodeHarness). The detection job does not go through
	// DetectRuntimeRequirements, so the setup must be emitted here explicitly. Guard
	// against engines whose install steps already bundle Setup Node.js (Claude/Codex
	// via BuildStandardNpmEngineInstallSteps) — a duplicate would trip
	// JobManager.ValidateDuplicateSteps and hard-fail the compile.
	if engineRequiresNodeHarness(engine) && !installStepsContainNodeSetup(installSteps) {
		for _, line := range GenerateNodeJsSetupStep() {
			steps = append(steps, line+"\n")
		}
	}

	for _, step := range installSteps {
		for _, line := range step {
			steps = append(steps, line+"\n")
		}
	}

	// Codex detection runs with no MCP tools, but still needs MCP gateway/config bootstrap
	// so config.toml includes the OpenAI proxy provider used by AWF API proxy mode.
	if engine.GetID() == "codex" {
		var mcpSetup strings.Builder
		if err := c.generateMCPSetup(&mcpSetup, threatDetectionData.Tools, engine, threatDetectionData); err == nil {
			for line := range strings.SplitSeq(mcpSetup.String(), "\n") {
				if line != "" {
					steps = append(steps, line+"\n")
				}
			}
		} else {
			threatLog.Printf("Failed to generate MCP setup for Codex detection; OpenAI proxy configuration may be incomplete: %v", err)
		}
	}

	logFile := constants.ThreatDetectionLogPath
	executionSteps := engine.GetExecutionSteps(threatDetectionData, logFile)
	for _, step := range executionSteps {
		for i, line := range step {
			// Prefix step IDs with "detection_" to avoid conflicts with agent job steps
			// (e.g., "agentic_execution" is already used by the main engine execution step)
			prefixed := strings.Replace(line, "id: agentic_execution", "id: detection_agentic_execution", 1)
			steps = append(steps, prefixed+"\n")
			// Inject the if condition and continue-on-error after the first line (- name:).
			// continue-on-error: true ensures that infrastructure failures (e.g. unhealthy
			// AWF container, Claude API errors) do not mark the detection job as failed.
			// The "Parse and conclude" step always runs (if: always()) and handles the
			// missing/incomplete detection log as parse_error in warn mode (exit 0).
			if i == 0 {
				steps = append(steps, fmt.Sprintf("        if: %s\n", detectionStepCondition))
				steps = append(steps, "        continue-on-error: true\n")
			}
		}
	}

	return steps
}
