package workflow

import (
	"github.com/github/gh-aw/pkg/logger"
)

var specializedOutputsLog = logger.New("workflow:compiler_safe_outputs_specialized")

// buildCreateAgentTaskStepConfig builds the configuration for creating an agent session
func (c *Compiler) buildCreateAgentSessionStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.CreateAgentSessions
	specializedOutputsLog.Print("Building create-agent-session step config")

	var customEnvVars []string
	customEnvVars = append(customEnvVars, c.buildStepLevelSafeOutputEnvVars(data, cfg.TargetRepoSlug)...)
	customEnvVars = append(customEnvVars, buildAllowedReposEnvVar("GH_AW_ALLOWED_REPOS", cfg.AllowedRepos)...)

	condition := BuildSafeOutputType("create_agent_session")

	return SafeOutputStepConfig{
		StepName:                "Create Agent Session",
		StepID:                  "create_agent_session",
		Script:                  "const { main } = require('${{ runner.temp }}/gh-aw/actions/create_agent_session.cjs'); await main();",
		CustomEnvVars:           customEnvVars,
		Condition:               condition,
		Token:                   cfg.GitHubToken,
		UseCopilotRequestsToken: true,
	}
}
