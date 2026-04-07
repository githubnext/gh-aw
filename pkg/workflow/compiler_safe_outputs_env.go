package workflow

import (
	"github.com/github/gh-aw/pkg/logger"
)

var compilerSafeOutputsEnvLog = logger.New("workflow:compiler_safe_outputs_env")

func (c *Compiler) addAllSafeOutputConfigEnvVars(steps *[]string, data *WorkflowData) {
	compilerSafeOutputsEnvLog.Print("Adding safe output config environment variables")
	if data.SafeOutputs == nil {
		compilerSafeOutputsEnvLog.Print("No safe outputs configured, skipping env var addition")
		return
	}

	// Add the global staged env var once if staged mode is enabled, not in trial mode,
	// and at least one handler is configured. Staged mode is independent of target-repo.
	if !c.trialMode && data.SafeOutputs.Staged && hasAnySafeOutputEnabled(data.SafeOutputs) {
		*steps = append(*steps, "          GH_AW_SAFE_OUTPUTS_STAGED: \"true\"\n")
		compilerSafeOutputsEnvLog.Print("Added staged flag")
	}

	// Check if copilot is in create-issue or create-pull-request assignees - enables inline copilot assignment
	if (data.SafeOutputs.CreateIssues != nil && hasCopilotAssignee(data.SafeOutputs.CreateIssues.Assignees)) ||
		(data.SafeOutputs.CreatePullRequests != nil && hasCopilotAssignee(data.SafeOutputs.CreatePullRequests.Assignees)) {
		*steps = append(*steps, "          GH_AW_ASSIGN_COPILOT: \"true\"\n")
		compilerSafeOutputsEnvLog.Print("Copilot assignment requested - enabled for create-issue or create-pull-request fallback issues")
	}

	// Note: All handler configuration is read from the config.json file at runtime.
}
