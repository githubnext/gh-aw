package workflow

import (
	"fmt"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var compilerSafeOutputJobsLog = logger.New("workflow:compiler_safe_output_jobs")

// buildSafeOutputsJobs builds all safe output jobs based on the configuration in data.SafeOutputs.
// It creates a consolidated safe_outputs job containing all safe output operations as steps,
// plus the threat detection job (if enabled), custom safe-jobs, and conclusion job.
func (c *Compiler) buildSafeOutputsJobs(data *WorkflowData, jobName, markdownPath string) error {
	if data.SafeOutputs == nil {
		compilerSafeOutputJobsLog.Print("No safe outputs configured, skipping safe outputs jobs")
		return nil
	}
	compilerSafeOutputJobsLog.Print("Building safe outputs jobs (consolidated mode)")

	// Track whether threat detection job is enabled
	threatDetectionEnabled := false

	// Build threat detection job if enabled
	if data.SafeOutputs.ThreatDetection != nil {
		compilerSafeOutputJobsLog.Print("Building threat detection job")
		detectionJob, err := c.buildThreatDetectionJob(data, jobName)
		if err != nil {
			return fmt.Errorf("failed to build detection job: %w", err)
		}
		if err := c.jobManager.AddJob(detectionJob); err != nil {
			return fmt.Errorf("failed to add detection job: %w", err)
		}
		compilerSafeOutputJobsLog.Printf("Successfully added threat detection job: %s", constants.DetectionJobName)
		threatDetectionEnabled = true
	}

	// Track safe output job names to establish dependencies for conclusion job
	var safeOutputJobNames []string

	// Build unified safe outputs job with single processor step
	unifiedJob, unifiedStepNames, err := c.buildUnifiedSafeOutputsJob(data, jobName, markdownPath)
	if err != nil {
		return fmt.Errorf("failed to build unified safe outputs job: %w", err)
	}
	if unifiedJob != nil {
		if err := c.jobManager.AddJob(unifiedJob); err != nil {
			return fmt.Errorf("failed to add unified safe outputs job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, unifiedJob.Name)
		compilerSafeOutputJobsLog.Printf("Added unified safe outputs job with single processor step: %v", unifiedStepNames)
	}

	// Build safe-jobs if configured
	// Safe-jobs should depend on agent job (always) AND detection job (if threat detection is enabled)
	// These custom safe-jobs should also be included in the conclusion job's dependencies
	safeJobNames, err := c.buildSafeJobs(data, threatDetectionEnabled)
	if err != nil {
		return fmt.Errorf("failed to build safe-jobs: %w", err)
	}
	// Add custom safe-job names to the list of safe output jobs
	safeOutputJobNames = append(safeOutputJobNames, safeJobNames...)
	compilerSafeOutputJobsLog.Printf("Added %d custom safe-job names to conclusion dependencies", len(safeJobNames))

	// Build conclusion job if add-comment is configured OR if command trigger is configured with reactions
	// This job runs last, after all safe output jobs (and push_repo_memory if configured), to update the activation comment on failure
	// The buildConclusionJob function itself will decide whether to create the job based on the configuration
	conclusionJob, err := c.buildConclusionJob(data, jobName, safeOutputJobNames)
	if err != nil {
		return fmt.Errorf("failed to build conclusion job: %w", err)
	}
	if conclusionJob != nil {
		// If push_repo_memory job exists, conclusion should depend on it
		// Check if the job was already created (it's created in buildJobs)
		if _, exists := c.jobManager.GetJob("push_repo_memory"); exists {
			conclusionJob.Needs = append(conclusionJob.Needs, "push_repo_memory")
			compilerSafeOutputJobsLog.Printf("Added push_repo_memory dependency to conclusion job")
		}
		if err := c.jobManager.AddJob(conclusionJob); err != nil {
			return fmt.Errorf("failed to add conclusion job: %w", err)
		}
	}

	return nil
}
