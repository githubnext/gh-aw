package workflow

import (
	"fmt"

	"github.com/github/gh-aw/pkg/logger"
)

var compilerSafeOutputJobsLog = logger.New("workflow:compiler_safe_output_jobs")

// buildSafeOutputsJobs builds all safe output jobs based on the configuration in data.SafeOutputs.
// It creates a separate detection job (if threat detection is enabled), a consolidated safe_outputs
// job containing all safe output operations as steps, plus custom safe-jobs and the conclusion job.
// When call-workflow is configured, it also generates conditional `uses:` fan-out jobs
// (one per allowed worker workflow) that run after safe_outputs.
func (c *Compiler) buildSafeOutputsJobs(data *WorkflowData, jobName, markdownPath string) error {
	if data.SafeOutputs == nil {
		compilerSafeOutputJobsLog.Print("No safe outputs configured, skipping safe outputs jobs")
		return nil
	}
	compilerSafeOutputJobsLog.Print("Building safe outputs jobs")

	// Detection is always enabled for safe-outputs workflows unless threat-detection is explicitly
	// disabled (threat-detection: false) or the engine is disabled with no custom steps
	// (threat-detection: { engine: false } with no steps). ThreatDetection is nil only when
	// explicitly disabled. When engine is false with no custom steps, the detection job has
	// nothing to run so it is skipped entirely.
	threatDetectionEnabled := IsDetectionJobEnabled(data.SafeOutputs)

	// Build the separate detection job. Detection runs by default for all safe-outputs workflows
	// and is only skipped when ThreatDetection is nil (i.e. threat-detection: false was set).
	// The detection job runs after the agent job, downloads the agent artifact,
	// and outputs detection_success and detection_conclusion for downstream jobs.
	if threatDetectionEnabled {
		detectionJob, err := c.buildDetectionJob(data)
		if err != nil {
			return fmt.Errorf("failed to build detection job: %w", err)
		}
		if detectionJob != nil {
			if err := c.jobManager.AddJob(detectionJob); err != nil {
				return fmt.Errorf("failed to add detection job: %w", err)
			}
			compilerSafeOutputJobsLog.Print("Added separate detection job")
		}
	}

	// Track safe output job names to establish dependencies for conclusion job
	var safeOutputJobNames []string

	// Build consolidated safe outputs job containing all safe output operations as steps
	consolidatedJob, consolidatedStepNames, err := c.buildConsolidatedSafeOutputsJob(data, jobName, markdownPath)
	if err != nil {
		return fmt.Errorf("failed to build consolidated safe outputs job: %w", err)
	}
	if consolidatedJob != nil {
		if err := c.jobManager.AddJob(consolidatedJob); err != nil {
			return fmt.Errorf("failed to add consolidated safe outputs job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, consolidatedJob.Name)
		compilerSafeOutputJobsLog.Printf("Added consolidated safe outputs job with %d steps: %v", len(consolidatedStepNames), consolidatedStepNames)
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

	// Build upload_assets job as a separate job if configured
	// This needs to be separate from the consolidated safe_outputs job because it requires:
	// 1. Git configuration for pushing to orphaned branches
	// 2. Checkout with proper credentials
	// 3. Different permissions (contents: write)
	if data.SafeOutputs != nil && data.SafeOutputs.UploadAssets != nil {
		compilerSafeOutputJobsLog.Print("Building separate upload_assets job")
		uploadAssetsJob, err := c.buildUploadAssetsJob(data, jobName, threatDetectionEnabled)
		if err != nil {
			return fmt.Errorf("failed to build upload_assets job: %w", err)
		}
		if err := c.jobManager.AddJob(uploadAssetsJob); err != nil {
			return fmt.Errorf("failed to add upload_assets job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, uploadAssetsJob.Name)
		compilerSafeOutputJobsLog.Printf("Added separate upload_assets job")
	}

	// Build conditional call-workflow fan-out jobs if configured.
	// Each allowed worker gets its own `uses:` job with an `if:` condition that
	// checks whether safe_outputs selected it. Only one runs per execution.
	callWorkflowJobNames, err := c.buildCallWorkflowJobs(data, markdownPath)
	if err != nil {
		return fmt.Errorf("failed to build call-workflow fan-out jobs: %w", err)
	}
	safeOutputJobNames = append(safeOutputJobNames, callWorkflowJobNames...)
	compilerSafeOutputJobsLog.Printf("Added %d call-workflow fan-out jobs", len(callWorkflowJobNames))

	// Build dedicated unlock job if lock-for-agent is enabled
	// This job is separate from conclusion to ensure it always runs, even if other jobs fail
	// It depends on agent and detection (if enabled) to run after workflow execution completes
	unlockJob, err := c.buildUnlockJob(data, threatDetectionEnabled)
	if err != nil {
		return fmt.Errorf("failed to build unlock job: %w", err)
	}
	if unlockJob != nil {
		if err := c.jobManager.AddJob(unlockJob); err != nil {
			return fmt.Errorf("failed to add unlock job: %w", err)
		}
		compilerSafeOutputJobsLog.Print("Added dedicated unlock job")
	}

	// Build conclusion job if add-comment is configured OR if command trigger is configured with reactions
	// This job runs last, after all safe output jobs (and push_repo_memory if configured), to update the activation comment on failure
	// The buildConclusionJob function itself will decide whether to create the job based on the configuration
	conclusionJob, err := c.buildConclusionJob(data, jobName, safeOutputJobNames)
	if err != nil {
		return fmt.Errorf("failed to build conclusion job: %w", err)
	}
	if conclusionJob != nil {
		// If unlock job exists, conclusion should depend on it to run after unlock completes
		if unlockJob != nil {
			conclusionJob.Needs = append(conclusionJob.Needs, "unlock")
			compilerSafeOutputJobsLog.Printf("Added unlock job dependency to conclusion job")
		}
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

// buildCallWorkflowJobs generates one conditional `uses:` job per workflow in the
// call-workflow allowlist. Each job:
//   - depends on safe_outputs
//   - has an `if:` that checks needs.safe_outputs.outputs.call_workflow_name
//   - uses: the relative path to the worker's .lock.yml (or .yml)
//   - passes payload as the canonical `with:` input
//   - also passes one `with:` entry per declared workflow_call input (except
//     payload) as `fromJSON(needs.safe_outputs.outputs.call_workflow_payload).<name>`
//     so that worker steps can reference inputs.<name> directly
//   - inherits all caller secrets via `secrets: inherit`
//   - includes a job-level `permissions:` block that is the union of all the
//     worker's job-level permissions, so GitHub allows the nested jobs to run
//
// Returns the names of all generated jobs so they can be added to the conclusion
// job's `needs` list.
func (c *Compiler) buildCallWorkflowJobs(data *WorkflowData, markdownPath string) ([]string, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.CallWorkflow == nil {
		return nil, nil
	}

	config := data.SafeOutputs.CallWorkflow
	if len(config.Workflows) == 0 {
		return nil, nil
	}

	compilerSafeOutputJobsLog.Printf("Building %d call-workflow fan-out jobs", len(config.Workflows))

	var jobNames []string

	for _, workflowName := range config.Workflows {
		// Build the job name: "call-{sanitized-workflow-name}"
		// sanitizeJobName normalizes underscores to hyphens (NormalizeSafeOutputIdentifier + dash conversion)
		sanitizedName := sanitizeJobName(workflowName)
		jobName := "call-" + sanitizedName

		// Determine the relative path to the worker workflow file
		workflowPath, ok := config.WorkflowFiles[workflowName]
		if !ok || workflowPath == "" {
			// Fallback: construct path from name
			workflowPath = fmt.Sprintf("./.github/workflows/%s.lock.yml", workflowName)
		}

		// Build the with: block. Always include the canonical payload transport,
		// then add per-input entries derived from the payload for every declared
		// workflow_call input on the worker (except 'payload' itself) so that
		// worker steps can reference inputs.<name> directly without parsing JSON.
		with := map[string]any{
			"payload": "${{ needs.safe_outputs.outputs.call_workflow_payload }}",
		}

		if markdownPath != "" {
			fileResult, findErr := findWorkflowFile(workflowName, markdownPath)
			if findErr != nil {
				compilerSafeOutputJobsLog.Printf("Warning: could not find worker workflow file for '%s': %v. "+
					"Typed inputs will not be forwarded in the with: block.", workflowName, findErr)
			} else {
				var workflowInputs map[string]any
				var inputErr error
				switch {
				case fileResult.lockExists:
					workflowInputs, inputErr = extractWorkflowCallInputs(fileResult.lockPath)
				case fileResult.ymlExists:
					workflowInputs, inputErr = extractWorkflowCallInputs(fileResult.ymlPath)
				case fileResult.mdExists:
					workflowInputs, inputErr = extractMDWorkflowCallInputs(fileResult.mdPath)
				default:
					compilerSafeOutputJobsLog.Printf("Warning: no worker file found for '%s'; "+
						"typed inputs will not be forwarded in the with: block.", workflowName)
				}
				if inputErr != nil {
					compilerSafeOutputJobsLog.Printf("Warning: could not extract workflow_call inputs for '%s': %v. "+
						"Typed inputs will not be forwarded in the with: block.", workflowName, inputErr)
				} else if workflowInputs != nil {
					typedInputCount := 0
					for inputName := range workflowInputs {
						if inputName == "payload" {
							continue
						}
						with[inputName] = fmt.Sprintf("${{ fromJSON(needs.safe_outputs.outputs.call_workflow_payload).%s }}", inputName)
						typedInputCount++
					}
					compilerSafeOutputJobsLog.Printf("Forwarding %d typed inputs for call-workflow job '%s'", typedInputCount, jobName)
				}
			}
		}

		callJob := &Job{
			Name:           jobName,
			Needs:          []string{"safe_outputs"},
			If:             fmt.Sprintf("needs.safe_outputs.outputs.call_workflow_name == '%s'", workflowName),
			Uses:           workflowPath,
			SecretsInherit: true,
			With:           with,
		}

		// Compute the permission superset required by the worker's jobs and
		// attach it to the caller job. GitHub validates reusable workflow calls
		// against the caller job's declared permission envelope; without a
		// permissions block the nested jobs are constrained to `none`.
		if markdownPath != "" {
			perms, permErr := extractCallWorkflowPermissions(workflowName, markdownPath)
			if permErr != nil {
				// Non-fatal: log and continue without permissions rather than aborting compilation.
				// The call-* job will be created without a permissions block; this may cause
				// GitHub to reject nested worker jobs that require non-none permissions.
				compilerSafeOutputJobsLog.Printf("Warning: could not extract permissions for call-workflow job '%s': %v. "+
					"Ensure the target workflow file exists and contains valid YAML. "+
					"The job will be created without a permissions block.", jobName, permErr)
			} else if perms != nil {
				rendered := perms.RenderToYAML()
				if rendered != "" {
					callJob.Permissions = rendered
					compilerSafeOutputJobsLog.Printf("Set permissions on call-workflow job '%s': %s", jobName, rendered)
				}
			}
		}

		if err := c.jobManager.AddJob(callJob); err != nil {
			return nil, fmt.Errorf("failed to add call-workflow job '%s': %w", jobName, err)
		}

		jobNames = append(jobNames, jobName)
		compilerSafeOutputJobsLog.Printf("Added call-workflow job: %s (uses: %s)", jobName, workflowPath)
	}

	return jobNames, nil
}
