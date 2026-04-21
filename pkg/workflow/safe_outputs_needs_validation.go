package workflow

import (
	"fmt"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var safeOutputsNeedsValidationLog = logger.New("workflow:safe_outputs_needs_validation")

func validateSafeOutputsNeeds(data *WorkflowData) error {
	if data == nil || data.SafeOutputs == nil {
		return nil
	}

	if err := validateSafeOutputsNeedsField(data, "needs", data.SafeOutputs.Needs); err != nil {
		return err
	}

	return nil
}

func validateSafeOutputsNeedsField(data *WorkflowData, fieldName string, needs []string) error {
	if len(needs) == 0 {
		return nil
	}

	customJobs := make(map[string]bool, len(data.Jobs))
	for jobName := range data.Jobs {
		if isReservedSafeOutputsNeedsTarget(jobName) {
			continue
		}
		customJobs[jobName] = true
	}

	for _, need := range needs {
		if isReservedSafeOutputsNeedsTarget(need) {
			return fmt.Errorf(
				"safe-outputs.%s: built-in job %q is not allowed. Expected one of the workflow's custom jobs. Example: safe-outputs.%s: [secrets_fetcher]",
				fieldName,
				need,
				fieldName,
			)
		}
		if !customJobs[need] {
			return fmt.Errorf(
				"safe-outputs.%s: unknown job %q. Expected one of the workflow's custom jobs. Example: safe-outputs.%s: [secrets_fetcher]",
				fieldName,
				need,
				fieldName,
			)
		}
	}

	safeOutputsNeedsValidationLog.Printf("Validated %d safe-outputs.%s dependency target(s)", len(needs), fieldName)
	return nil
}

func isReservedSafeOutputsNeedsTarget(jobName string) bool {
	switch jobName {
	case string(constants.AgentJobName),
		string(constants.ActivationJobName),
		string(constants.PreActivationJobName),
		"pre-activation",
		string(constants.ConclusionJobName),
		string(constants.SafeOutputsJobName),
		"safe-outputs",
		string(constants.DetectionJobName),
		string(constants.UnlockJobName),
		"push_repo_memory",
		"update_cache_memory":
		return true
	default:
		return false
	}
}
