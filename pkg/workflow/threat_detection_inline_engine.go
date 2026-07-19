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
	if isThreatDetectionEngineDisabled(data) {
		return []string{"      # AI engine disabled for threat detection (engine: false)\n"}
	}

	engineSetting := c.getThreatDetectionEngineID(data)
	engineConfig, originalEngineID := resolveThreatDetectionEngineConfig(data)
	engine, err := c.getAgenticEngine(engineSetting)
	if err != nil {
		return []string{"      # Engine not found, skipping execution\n"}
	}

	detectionEngineConfig := buildDetectionEngineConfig(data, engine, engineConfig, engineSetting, originalEngineID)
	threatDetectionData := buildThreatDetectionWorkflowData(data, engineSetting)
	populateThreatDetectionExecutionData(threatDetectionData, data, detectionEngineConfig)

	var steps []string
	installSteps := engine.GetInstallationSteps(threatDetectionData)
	appendThreatDetectionInstallSteps(&steps, engine, installSteps)
	c.appendThreatDetectionMCPSetup(&steps, threatDetectionData, engine)
	appendThreatDetectionExecutionSteps(&steps, engine.GetExecutionSteps(threatDetectionData, constants.ThreatDetectionLogPath))

	return steps
}

func isThreatDetectionEngineDisabled(data *WorkflowData) bool {
	return data.SafeOutputs != nil &&
		data.SafeOutputs.ThreatDetection != nil &&
		data.SafeOutputs.ThreatDetection.EngineDisabled
}

func resolveThreatDetectionEngineConfig(data *WorkflowData) (*EngineConfig, string) {
	engineConfig := data.EngineConfig
	hasThreatDetectionEngineConfig := data.SafeOutputs != nil &&
		data.SafeOutputs.ThreatDetection != nil &&
		data.SafeOutputs.ThreatDetection.EngineConfig != nil
	if hasThreatDetectionEngineConfig {
		engineConfig = data.SafeOutputs.ThreatDetection.EngineConfig
	}
	originalEngineID := data.AI
	if data.EngineConfig != nil && data.EngineConfig.ID != "" {
		originalEngineID = data.EngineConfig.ID
	}
	if hasThreatDetectionEngineConfig && data.SafeOutputs.ThreatDetection.EngineConfig.ID != "" {
		originalEngineID = data.SafeOutputs.ThreatDetection.EngineConfig.ID
	}
	return engineConfig, originalEngineID
}

func buildDetectionEngineConfig(data *WorkflowData, engine CodingAgentEngine, engineConfig *EngineConfig, engineSetting, originalEngineID string) *EngineConfig {
	detectionEngineConfig := cloneDetectionEngineConfig(engineConfig, engineSetting)
	if data.SafeOutputs != nil && data.SafeOutputs.ThreatDetection != nil && data.SafeOutputs.ThreatDetection.MaxAICredits != 0 {
		detectionEngineConfig.MaxAICredits = data.SafeOutputs.ThreatDetection.MaxAICredits
	}
	if detectionEngineConfig.Model == "" {
		if defaultModel := compilerenv.ResolveDefaultDetectionModel(""); defaultModel != "" {
			detectionEngineConfig.Model = defaultModel
		} else if defaultModel := engine.GetDefaultDetectionModel(); defaultModel != "" {
			detectionEngineConfig.Model = defaultModel
		}
	}
	if detectionEngineConfig.APITarget == "" && data.EngineConfig != nil && data.EngineConfig.APITarget != "" {
		detectionEngineConfig.APITarget = data.EngineConfig.APITarget
	}
	if engineSetting == "copilot" && originalEngineID == "pi" {
		detectionEngineConfig.Model = extractPiModelID(detectionEngineConfig.Model)
	}
	return detectionEngineConfig
}

func cloneDetectionEngineConfig(engineConfig *EngineConfig, engineSetting string) *EngineConfig {
	if engineConfig == nil {
		return &EngineConfig{ID: engineSetting}
	}
	clone := &EngineConfig{
		ID:            engineConfig.ID,
		Model:         engineConfig.Model,
		Version:       engineConfig.Version,
		Env:           engineConfig.Env,
		Config:        engineConfig.Config,
		Args:          engineConfig.Args,
		APITarget:     engineConfig.APITarget,
		HarnessScript: engineConfig.HarnessScript,
		Driver:        engineConfig.Driver,
	}
	if clone.ID == "" {
		clone.ID = engineSetting
	}
	return clone
}

func populateThreatDetectionExecutionData(threatDetectionData, data *WorkflowData, detectionEngineConfig *EngineConfig) {
	threatDetectionData.Tools = map[string]any{"bash": []any{"*"}}
	threatDetectionData.EngineConfig = detectionEngineConfig
	threatDetectionData.ModelMappings = data.ModelMappings
	threatDetectionData.NetworkPermissions = &NetworkPermissions{
		Allowed: getThreatDetectionAdditionalAllowedDomains(data),
	}
}

func appendThreatDetectionInstallSteps(steps *[]string, engine CodingAgentEngine, installSteps []GitHubActionStep) {
	if engineRequiresNodeHarness(engine) && !installStepsContainNodeSetup(installSteps) {
		for _, line := range GenerateNodeJsSetupStep() {
			*steps = append(*steps, line+"\n")
		}
	}
	for _, step := range installSteps {
		for _, line := range step {
			*steps = append(*steps, line+"\n")
		}
	}
}

func (c *Compiler) appendThreatDetectionMCPSetup(steps *[]string, threatDetectionData *WorkflowData, engine CodingAgentEngine) {
	if engine.GetID() != "codex" {
		return
	}
	var mcpSetup strings.Builder
	if err := c.generateMCPSetup(&mcpSetup, threatDetectionData.Tools, engine, threatDetectionData); err == nil {
		for line := range strings.SplitSeq(mcpSetup.String(), "\n") {
			if line != "" {
				*steps = append(*steps, line+"\n")
			}
		}
	} else {
		threatLog.Printf("Failed to generate MCP setup for Codex detection; OpenAI proxy configuration may be incomplete: %v", err)
	}
}

func appendThreatDetectionExecutionSteps(steps *[]string, executionSteps []GitHubActionStep) {
	for _, step := range executionSteps {
		for i, line := range step {
			prefixed := strings.Replace(line, "id: agentic_execution", "id: detection_agentic_execution", 1)
			*steps = append(*steps, prefixed+"\n")
			if i == 0 {
				*steps = append(*steps, fmt.Sprintf("        if: %s\n", detectionStepCondition))
				*steps = append(*steps, "        continue-on-error: true\n")
			}
		}
	}
}
