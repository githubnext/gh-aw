package workflow

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var onNeedsValidationLog = logger.New("workflow:on_needs_validation")

var onNeedsOutputExpressionPattern = regexp.MustCompile(`^\$\{\{\s*needs\.([A-Za-z_][A-Za-z0-9_-]*)\.outputs\.[A-Za-z_][A-Za-z0-9_-]*\s*\}\}$`)

func (c *Compiler) validateOnNeeds(data *WorkflowData) error {
	if data == nil {
		return nil
	}

	if err := validateOnNeedsTargets(data); err != nil {
		return err
	}

	if err := c.validateOnGitHubAppNeedsExpressions(data); err != nil {
		return err
	}

	return nil
}

func validateOnNeedsTargets(data *WorkflowData) error {
	if len(data.OnNeeds) == 0 {
		return nil
	}

	customJobs := make(map[string]bool, len(data.Jobs))
	for jobName := range data.Jobs {
		if isReservedOnNeedsTarget(jobName) {
			continue
		}
		customJobs[jobName] = true
	}

	for _, need := range data.OnNeeds {
		if isReservedOnNeedsTarget(need) {
			return fmt.Errorf(
				"on.needs: built-in job %q is not allowed. Expected one of the workflow's custom jobs. Example: on.needs: [secrets_fetcher]",
				need,
			)
		}
		if !customJobs[need] {
			return fmt.Errorf(
				"on.needs: unknown job %q. Expected one of the workflow's custom jobs. Example: on.needs: [secrets_fetcher]",
				need,
			)
		}

		if jobConfig, ok := data.Jobs[need].(map[string]any); ok {
			if jobDependsOnActivation(jobConfig) || jobDependsOnPreActivation(jobConfig) {
				return fmt.Errorf(
					"on.needs: job %q cannot depend on activation/pre_activation because pre_activation and activation depend on on.needs jobs",
					need,
				)
			}
		}
	}

	onNeedsValidationLog.Printf("Validated %d on.needs dependency target(s)", len(data.OnNeeds))
	return nil
}

func (c *Compiler) validateOnGitHubAppNeedsExpressions(data *WorkflowData) error {
	if data == nil || data.ActivationGitHubApp == nil {
		return nil
	}

	allowed := make(map[string]bool, len(data.OnNeeds))
	for _, j := range data.OnNeeds {
		allowed[j] = true
	}
	for _, j := range c.getCustomJobsDependingOnPreActivation(data.Jobs) {
		allowed[j] = true
	}
	for _, j := range c.getCustomJobsReferencedInPromptWithNoActivationDep(data) {
		allowed[j] = true
	}

	fields := map[string]string{
		"app-id":      data.ActivationGitHubApp.AppID,
		"private-key": data.ActivationGitHubApp.PrivateKey,
	}

	for fieldName, value := range fields {
		jobName, ok := extractNeedsJobFromOutputExpression(value)
		if !ok {
			continue
		}

		if isReservedOnNeedsTarget(jobName) {
			return fmt.Errorf("on.github-app.%s: built-in job %q is not allowed in needs expressions", fieldName, jobName)
		}
		if _, exists := data.Jobs[jobName]; !exists {
			return fmt.Errorf("on.github-app.%s: unknown job %q in needs expression", fieldName, jobName)
		}
		if !allowed[jobName] {
			return fmt.Errorf(
				"on.github-app.%s references needs.%s.outputs.* but job %q is not available before activation. Add it to on.needs (example: on.needs: [%s])",
				fieldName,
				jobName,
				jobName,
				jobName,
			)
		}
	}

	return nil
}

func extractNeedsJobFromOutputExpression(value string) (string, bool) {
	match := onNeedsOutputExpressionPattern.FindStringSubmatch(strings.TrimSpace(value))
	if len(match) != 2 {
		return "", false
	}
	return match[1], true
}

func isReservedOnNeedsTarget(jobName string) bool {
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
