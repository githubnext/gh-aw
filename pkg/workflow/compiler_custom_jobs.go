// Package workflow implements custom-job orchestration for workflow compilation.
//
// The compiler job builders are split into focused modules for maintainability:
//
//   - compiler_jobs.go: Core job orchestration and cross-job dependency wiring
//   - compiler_custom_jobs.go: Custom job construction and dependency wiring
//   - compiler_custom_job_properties.go: Custom job property mapping
//   - compiler_custom_job_execution.go: Reusable workflows and step execution
//   - compiler_builtin_job_augmentation.go: Built-in job augmentation
//   - compiler_job_step_helpers.go: Shared job step validation and insertion
//
// This separation keeps the orchestration flow compact while preserving the
// existing custom job behavior.
package workflow

import (
	"fmt"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/setutil"
)

// buildCustomJobs creates custom jobs defined in the frontmatter jobs section
func (c *Compiler) buildCustomJobs(data *WorkflowData, activationJobCreated bool) error {
	compilerJobsLog.Printf("Building %d custom jobs", len(data.Jobs))

	promptReferencedJobs, onNeedsJobs := c.getCustomJobDependencySets(data)

	for jobName, jobConfig := range data.Jobs {
		if c.shouldSkipCustomJob(jobName) {
			continue
		}
		configMap, ok := jobConfig.(map[string]any)
		if !ok {
			continue
		}

		job, err := c.buildCustomJob(
			jobName,
			configMap,
			data,
			activationJobCreated,
			promptReferencedJobs,
			onNeedsJobs,
		)
		if err != nil {
			return err
		}

		if err := c.jobManager.AddJob(job); err != nil {
			return fmt.Errorf("custom job '%s' could not be added: %w. Check the job configuration for conflicting names or unsupported fields", jobName, err)
		}
		compilerJobsLog.Printf("Successfully added custom job '%s' with %d needs dependencies", jobName, len(job.Needs))
	}

	compilerJobsLog.Print("Completed building all custom jobs")
	return nil
}

func (c *Compiler) getCustomJobDependencySets(data *WorkflowData) (map[string]struct{}, map[string]struct{}) {
	// Pre-compute jobs referenced in the markdown body with no explicit needs.
	// These run before activation (not after), so we must not auto-add activation to them.
	promptReferencedJobsSlice := c.getCustomJobsReferencedInPromptWithNoActivationDep(data)
	promptReferencedJobs := make(map[string]struct{}, len(promptReferencedJobsSlice))
	for _, j := range promptReferencedJobsSlice {
		promptReferencedJobs[j] = struct{}{}
	}

	// Also include jobs with no explicit needs that are referenced in engine.env.
	// These run before activation (activation depends on them for secret validation etc.),
	// so we must not auto-add activation to them either — doing so would create a cycle
	// (activation → job → activation).
	for _, j := range c.getEngineEnvReferencedCustomJobsWithNoExplicitNeeds(data) {
		promptReferencedJobs[j] = struct{}{}
	}

	onNeedsJobs := make(map[string]struct{}, len(data.OnNeeds))
	for _, j := range data.OnNeeds {
		onNeedsJobs[j] = struct{}{}
	}

	return promptReferencedJobs, onNeedsJobs
}

func (c *Compiler) shouldSkipCustomJob(jobName string) bool {
	// Skip jobs.pre-activation (or pre_activation) as it's handled specially in buildPreActivationJob
	if jobName == string(constants.PreActivationJobName) || jobName == "pre-activation" {
		compilerJobsLog.Printf("Skipping jobs.%s (handled in buildPreActivationJob)", jobName)
		return true
	}

	// Built-in jobs are already created before buildCustomJobs; treat jobs.<builtin>
	// entries as customization-only and do not create duplicate jobs.
	if _, exists := c.jobManager.GetJob(jobName); exists {
		compilerJobsLog.Printf("Skipping jobs.%s (built-in job already exists)", jobName)
		return true
	}

	return false
}

func (c *Compiler) buildCustomJob(
	jobName string,
	configMap map[string]any,
	data *WorkflowData,
	activationJobCreated bool,
	promptReferencedJobs map[string]struct {
	}, onNeedsJobs map[string]struct {
	}) (*Job, error) {
	job := &Job{Name: jobName}

	hasExplicitNeeds := extractCustomJobNeeds(job, configMap)
	c.applyAutomaticActivationDependency(job, jobName, hasExplicitNeeds, activationJobCreated, promptReferencedJobs, onNeedsJobs)

	if err := c.extractCustomJobProperties(job, jobName, configMap); err != nil {
		return nil, err
	}

	if err := c.configureCustomJobExecution(job, jobName, configMap, data); err != nil {
		return nil, err
	}

	return job, nil
}

func extractCustomJobNeeds(job *Job, configMap map[string]any) bool {
	needs, hasNeeds := configMap["needs"]
	if !hasNeeds {
		return false
	}

	if needsList, ok := needs.([]any); ok {
		for _, need := range needsList {
			if needStr, ok := need.(string); ok {
				job.Needs = append(job.Needs, needStr)
			}
		}
	} else if needStr, ok := needs.(string); ok {
		// Single dependency as string
		job.Needs = append(job.Needs, needStr)
	}

	return true
}

func (c *Compiler) applyAutomaticActivationDependency(
	job *Job,
	jobName string,
	hasExplicitNeeds bool,
	activationJobCreated bool,
	promptReferencedJobs map[string]struct {
	}, onNeedsJobs map[string]struct {
	}) {
	// If no explicit needs and activation job exists, automatically add activation as dependency
	// This ensures custom jobs wait for workflow validation before executing.
	// Exception: jobs whose outputs are referenced in the markdown body run before activation
	// (so the activation job can include their outputs in the prompt).
	isReferencedInMarkdown := setutil.Contains(promptReferencedJobs, jobName)
	isOnNeedsDependency := setutil.Contains(onNeedsJobs, jobName)

	if !hasExplicitNeeds && activationJobCreated && !isReferencedInMarkdown && !isOnNeedsDependency {
		job.Needs = append(job.Needs, string(constants.ActivationJobName))
		compilerJobsLog.Printf("Added automatic dependency: custom job '%s' now depends on '%s'", jobName, string(constants.ActivationJobName))
	} else if !hasExplicitNeeds && isReferencedInMarkdown {
		compilerJobsLog.Printf("Custom job '%s' referenced in markdown body runs before activation (no auto-added dependency)", jobName)
	} else if !hasExplicitNeeds && isOnNeedsDependency {
		compilerJobsLog.Printf("Custom job '%s' listed in on.needs runs before activation (no auto-added dependency)", jobName)
	}
}
