// Built-in job augmentation applies user configuration to compiler-generated jobs.
package workflow

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/typeutil"
)

func (c *Compiler) applyBuiltinJobPreSteps(data *WorkflowData) error {
	if data == nil || data.Jobs == nil {
		return nil
	}

	for jobName, jobConfig := range data.Jobs {
		configMap, ok := jobConfig.(map[string]any)
		if !ok {
			return fmt.Errorf("jobs.%s must be an object, got %T. Example: jobs:\n  job-name:\n    setup-steps: []", jobName, jobConfig)
		}

		_, hasSetupSteps := configMap["setup-steps"]
		_, hasPreSteps := configMap["pre-steps"]
		_, hasSteps := configMap["steps"]
		if err := validateRestrictedBuiltinSetupSteps(jobName, hasSetupSteps); err != nil {
			return err
		}
		if !hasSetupSteps && !hasPreSteps && !hasSteps {
			continue
		}

		targetJobName := jobName
		if jobName == "pre-activation" {
			targetJobName = string(constants.PreActivationJobName)
		}

		if err := validateRestrictedBuiltinSteps(jobName, targetJobName, hasSteps); err != nil {
			return err
		}

		job, exists := c.jobManager.GetJob(targetJobName)
		if !exists {
			continue
		}

		var setupSteps []string
		var preSteps []string
		var regularSteps []string
		if hasSetupSteps {
			steps, err := c.extractPinnedJobSteps("setup-steps", jobName, configMap, data)
			if err != nil {
				return fmt.Errorf("setup-steps for built-in job '%s' could not be processed: %w. Check that setup-steps is an array of valid step objects", jobName, err)
			}
			setupSteps = append(setupSteps, steps...)
		}
		if hasPreSteps {
			steps, err := c.extractPinnedJobSteps("pre-steps", jobName, configMap, data)
			if err != nil {
				return fmt.Errorf("pre-steps for built-in job '%s' could not be processed: %w. Check that pre-steps is an array of valid step objects", jobName, err)
			}
			preSteps = append(preSteps, steps...)
		}
		if hasSteps && targetJobName == string(constants.ActivationJobName) {
			steps, err := c.extractPinnedJobSteps("steps", jobName, configMap, data)
			if err != nil {
				return fmt.Errorf("steps for built-in job '%s' could not be processed: %w. Check that steps is an array of valid step objects", jobName, err)
			}
			regularSteps = append(regularSteps, steps...)
		}
		if len(setupSteps) == 0 && len(preSteps) == 0 && len(regularSteps) == 0 {
			continue
		}

		job.Steps = insertActivationStepsBeforeArtifactStaging(targetJobName, job.Steps, regularSteps)
		job.Steps = insertPreStepsAtEarliestBoundary(job.Steps, preSteps)
		job.Steps = insertSetupStepsAtStart(job.Steps, setupSteps)
		compilerJobsLog.Printf("Inserted %d setup-step(s), %d pre-step(s), and %d step(s) into built-in job '%s'", len(setupSteps), len(preSteps), len(regularSteps), targetJobName)
	}

	return nil
}

func insertActivationStepsBeforeArtifactStaging(jobName string, steps []string, activationSteps []string) []string {
	if len(activationSteps) == 0 {
		return steps
	}
	if jobName != string(constants.ActivationJobName) {
		return steps
	}

	insertIdx := len(steps)
	for i, step := range steps {
		if strings.Contains(step, "name: "+constants.ActivationStageAmbientFoldersStepName) ||
			strings.Contains(step, "name: "+constants.ActivationUploadArtifactStepName) {
			insertIdx = i
			break
		}
	}

	result := make([]string, 0, typeutil.SafeAllocationCapacity(len(steps), len(activationSteps)))
	result = append(result, steps[:insertIdx]...)
	result = append(result, activationSteps...)
	result = append(result, steps[insertIdx:]...)
	return result
}

func normalizeBuiltinJobAlias(jobName string) string {
	switch jobName {
	case string(constants.PreActivationHyphenJobName):
		return string(constants.PreActivationJobName)
	case string(constants.SafeOutputsHyphenJobName):
		return string(constants.SafeOutputsJobName)
	default:
		return jobName
	}
}

func extractBuiltinJobNeedsAugmentation(jobName string, configMap map[string]any) ([]string, error) {
	needsValue, exists := configMap["needs"]
	if !exists || needsValue == nil {
		return nil, nil
	}

	switch typedNeeds := needsValue.(type) {
	case string:
		return []string{typedNeeds}, nil
	case []any:
		needs := make([]string, 0, len(typedNeeds))
		for i, rawNeed := range typedNeeds {
			need, ok := rawNeed.(string)
			if !ok {
				return nil, fmt.Errorf("jobs.%s.needs[%d] must be a string, got %T. Example: needs: ['build', 'test']", jobName, i, rawNeed)
			}
			needs = append(needs, need)
		}
		return needs, nil
	default:
		return nil, fmt.Errorf("jobs.%s.needs expects a string or array of strings, got %T. Example: needs: [build, test]", jobName, needsValue)
	}
}

func extractBuiltinJobIfAugmentation(jobName string, configMap map[string]any) (string, error) {
	ifValue, exists := configMap["if"]
	if !exists || ifValue == nil {
		return "", nil
	}

	ifCondition, ok := ifValue.(string)
	if !ok {
		return "", fmt.Errorf("jobs.%s.if expects a string, got %T. Example: if: github.event_name == 'push'", jobName, ifValue)
	}

	// Strip "if: " prefix to match the Job.If contract (bare expression, no prefix).
	// This mirrors how custom jobs normalize their if fields via extractExpressionFromIfString.
	if strings.HasPrefix(ifCondition, "if: ") {
		ifCondition = strings.TrimSpace(ifCondition[4:])
	}

	return ifCondition, nil
}

// applyBuiltinJobAugmentations merges supported jobs.<built-in> fields into
// compiler-generated jobs. needs entries are added additively; if conditions are combined
// with compiler-generated conditions via logical AND.
func (c *Compiler) applyBuiltinJobAugmentations(data *WorkflowData) error {
	if data == nil || data.Jobs == nil {
		return nil
	}

	allJobs := c.jobManager.GetAllJobs()
	for configuredJobName, rawConfig := range data.Jobs {
		targetJobName := normalizeBuiltinJobAlias(configuredJobName)
		if !isBuiltinJobName(targetJobName) {
			continue
		}

		configMap, ok := rawConfig.(map[string]any)
		if !ok {
			return fmt.Errorf("jobs.%s expects an object, got %T. Example: jobs:\n  %s:\n    runs-on: ubuntu-latest", configuredJobName, rawConfig, configuredJobName)
		}

		augmentedNeeds, err := extractBuiltinJobNeedsAugmentation(configuredJobName, configMap)
		if err != nil {
			return err
		}
		augmentedIf, err := extractBuiltinJobIfAugmentation(configuredJobName, configMap)
		if err != nil {
			return err
		}
		_, hasPermissions := configMap["permissions"]
		_, hasTimeout := configMap["timeout-minutes"]
		if hasTimeout && targetJobName != string(constants.AgentJobName) && targetJobName != string(constants.DetectionJobName) {
			return fmt.Errorf("jobs.%s.timeout-minutes is supported only for the generated agent and detection jobs", configuredJobName)
		}
		if len(augmentedNeeds) == 0 && augmentedIf == "" && !hasPermissions && !hasTimeout {
			continue
		}

		targetJob, exists := c.jobManager.GetJob(targetJobName)
		if !exists {
			// Report the actual field(s) the author configured so they can identify the problem.
			augmentedField := configuredJobName + ".needs"
			if len(augmentedNeeds) == 0 {
				if augmentedIf != "" {
					augmentedField = configuredJobName + ".if"
				} else if hasTimeout {
					augmentedField = configuredJobName + ".timeout-minutes"
				} else {
					augmentedField = configuredJobName + ".permissions"
				}
			} else if augmentedIf != "" || hasPermissions || hasTimeout {
				augmentedField = configuredJobName
			}
			return fmt.Errorf("jobs.%s requires an existing built-in job %q, but this workflow does not generate it. Add the corresponding trigger/feature, or rename the job", augmentedField, targetJobName)
		}

		if hasPermissions {
			if err := applyBuiltinJobPermissionsAugmentation(configuredJobName, targetJobName, configMap, targetJob); err != nil {
				return err
			}
		}
		if hasTimeout {
			if err := extractCustomJobTimeoutMinutes(targetJob, configuredJobName, configMap); err != nil {
				return err
			}
		}

		normalizedNeeds := make([]string, 0, len(augmentedNeeds))
		for _, rawNeed := range augmentedNeeds {
			need := normalizeBuiltinJobAlias(rawNeed)
			if need == targetJobName {
				return fmt.Errorf("jobs.%s.needs lists %q, but a job should not depend on itself. Remove the self-reference from needs", configuredJobName, rawNeed)
			}
			if _, known := allJobs[need]; !known {
				return fmt.Errorf("jobs.%s.needs: unknown job %q. Expected a job defined in this workflow or a generated built-in job. Example:\njobs:\n  %s:\n    needs: [activation]", configuredJobName, rawNeed, configuredJobName)
			}
			normalizedNeeds = append(normalizedNeeds, need)
		}

		compilerOwnedNeeds := selectCompilerOwnedNeeds(targetJob.Needs, data.Jobs)

		seen := make(map[string]struct{}, len(targetJob.Needs)+len(normalizedNeeds))
		mergedNeeds := make([]string, 0, len(targetJob.Needs)+len(normalizedNeeds))
		for _, need := range targetJob.Needs {
			if _, alreadySeen := seen[need]; alreadySeen {
				continue
			}
			seen[need] = struct{}{}
			mergedNeeds = append(mergedNeeds, need)
		}
		for _, need := range normalizedNeeds {
			if _, alreadySeen := seen[need]; alreadySeen {
				continue
			}
			seen[need] = struct{}{}
			mergedNeeds = append(mergedNeeds, need)
		}
		targetJob.Needs = mergedNeeds
		if augmentedIf != "" {
			// Guard against status-function bypasses: when the user condition contains a
			// status function (always, failure, cancelled, success), GitHub Actions removes
			// the implicit success() check for all needs. Compiler-owned prerequisites such
			// as activation perform security and permission checks and must always succeed,
			// so we add explicit result == 'success' guards for those jobs only.
			guardedIf := c.guardIfAgainstStatusFuncBypass(augmentedIf, compilerOwnedNeeds)
			targetJob.If = c.combineJobIfConditions(targetJob.If, guardedIf)
			compilerJobsLog.Printf("Applied jobs.%s.if augmentation to %q", configuredJobName, targetJobName)
		}
		if len(normalizedNeeds) > 0 {
			compilerJobsLog.Printf("Applied jobs.%s.needs augmentation to %q: %v", configuredJobName, targetJobName, normalizedNeeds)
		}
	}
	return nil
}

// applyBuiltinJobPermissionsAugmentation merges user-declared jobs.<built-in>.permissions
// into a compiler-generated built-in job (e.g. safe_outputs, conclusion). The merge is
// additive: the compiler-computed permissions are preserved and the user's declared scopes
// are added on top, with write overriding read. This ensures scopes such as id-token: write
// that authors declare under jobs.*.permissions are retained in the compiled lock file rather
// than being dropped by the minimal least-privilege permission computation.
func applyBuiltinJobPermissionsAugmentation(configuredJobName, targetJobName string, configMap map[string]any, targetJob *Job) error {
	permissionsValue, exists := configMap["permissions"]
	if !exists || permissionsValue == nil {
		return nil
	}

	userPermissions := NewPermissionsParserFromValue(permissionsValue).ToPermissions()
	if userPermissions == nil {
		return nil
	}

	// Start from the compiler-computed permissions already rendered on the job, then merge
	// the user-declared permissions additively so no compiler-required scope is lost.
	merged := NewPermissionsParser(targetJob.Permissions).ToPermissions()
	merged.Merge(userPermissions)
	targetJob.Permissions = merged.RenderToYAML()
	compilerJobsLog.Printf("Applied jobs.%s.permissions augmentation to %q", configuredJobName, targetJobName)
	return nil
}

// selectCompilerOwnedNeeds returns the prerequisites of a built-in job that the compiler owns,
// i.e. the needs that are not custom jobs declared under top-level `jobs:`. Custom jobs are
// auto-wired as prerequisites of built-in jobs but remain author-owned, so the author picks
// their result semantics (for example an `if: always()` agent that analyses a failing probe
// job). Compiler-owned prerequisites such as activation must stay guarded.
func selectCompilerOwnedNeeds(needs []string, customJobs map[string]any) []string {
	owned := make([]string, 0, len(needs))
	for _, need := range needs {
		if _, isCustomJob := customJobs[need]; isCustomJob && !isBuiltinJobName(need) {
			continue
		}
		owned = append(owned, need)
	}
	return owned
}

// guardIfAgainstStatusFuncBypass returns userCondition augmented with explicit
// needs.<need>.result == 'success' guards for each compiler-owned prerequisite, but only
// when userCondition contains a GitHub Actions status function (always, failure, cancelled,
// success).
//
// GitHub Actions removes the implicit success() check for ALL needs entries the moment any
// status function appears in a job's if expression. Compiler-owned prerequisites such as
// activation perform security and permission checks; they must always succeed before the
// target job runs. This function makes those guards explicit so user-supplied status functions
// cannot inadvertently (or intentionally) bypass them. User-supplied needs are intentionally
// excluded: authors choose their own result semantics for setup jobs they own.
func (c *Compiler) guardIfAgainstStatusFuncBypass(userCondition string, compilerNeeds []string) string {
	if len(compilerNeeds) == 0 {
		return userCondition
	}

	// Use string-based detection: the expression parser tokenises status function calls such as
	// failure() as single ExpressionNode literals, so AST-based containsStatusFunc cannot be used
	// here. A substring check is sufficient since GitHub Actions has a fixed, well-known set of
	// status functions and user-defined functions are not supported in the expression language.
	bare := stripExpressionWrapper(userCondition)
	if !ifExpressionContainsStatusFunc(bare) {
		return userCondition
	}

	// Build explicit success guards for each compiler-owned prerequisite.
	compilerJobsLog.Printf("Status function detected in user if condition; adding explicit success guards for compiler needs: %v", compilerNeeds)
	combined := ConditionNode(&ExpressionNode{Expression: bare})
	for _, need := range compilerNeeds {
		guard := &ExpressionNode{Expression: fmt.Sprintf("needs.%s.result == 'success'", need)}
		combined = BuildAnd(combined, guard)
	}
	return RenderCondition(combined)
}

// ifExpressionContainsStatusFunc reports whether the GitHub Actions expression string
// contains a call to any of the four status check functions (always, success, failure,
// cancelled). When present, GitHub Actions removes the implicit success() gate that would
// otherwise be applied to all needs entries.
func ifExpressionContainsStatusFunc(expr string) bool {
	return strings.Contains(expr, "always(") ||
		strings.Contains(expr, "success(") ||
		strings.Contains(expr, "failure(") ||
		strings.Contains(expr, "cancelled(")
}
