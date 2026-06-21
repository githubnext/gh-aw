package workflow

import (
	"fmt"
	"os"
	"strings"

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

	// Build upload_code_scanning_sarif job as a separate job if create-code-scanning-alert is configured.
	// This job runs after safe_outputs and only when the safe_outputs job exported a SARIF file.
	// It is separate to avoid the checkout step (needed to restore HEAD to github.sha) from
	// interfering with other safe-output operations in the consolidated safe_outputs job.
	if data.SafeOutputs != nil && data.SafeOutputs.CreateCodeScanningAlerts != nil &&
		!isHandlerStaged(false || data.SafeOutputs.Staged, data.SafeOutputs.CreateCodeScanningAlerts.Staged) {
		compilerSafeOutputJobsLog.Print("Building separate upload_code_scanning_sarif job")
		codeScanningJob, err := c.buildCodeScanningUploadJob(data)
		if err != nil {
			return fmt.Errorf("failed to build upload_code_scanning_sarif job: %w", err)
		}
		if err := c.jobManager.AddJob(codeScanningJob); err != nil {
			return fmt.Errorf("failed to add upload_code_scanning_sarif job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, codeScanningJob.Name)
		compilerSafeOutputJobsLog.Printf("Added separate upload_code_scanning_sarif job")
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
//   - forwards declared workflow_call inputs in `with:` so worker steps can reference inputs.<name> directly:
//   - non-payload inputs: `fromJSON(needs.safe_outputs.outputs.call_workflow_payload).<name>`
//   - `payload` is forwarded as the raw transport only when the worker declares it
//     (GitHub Actions rejects undeclared inputs)
//   - inherits all caller secrets via `secrets: inherit`
//   - includes a job-level `permissions:` block derived from the CALLER's own
//     declared permissions (not the worker's). The caller controls its own
//     permission surface; the compiler validates that the declared permissions
//     cover what the worker requires and warns if they do not.
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

		// Build the with: block. Forward one entry per declared workflow_call input
		// on the worker, derived from the payload, so that worker steps can reference
		// inputs.<name> directly without parsing JSON. The canonical `payload`
		// envelope is only forwarded when the worker explicitly declares a `payload`
		// input; GitHub Actions rejects a `uses:` step that passes an input the
		// called workflow does not declare, so it must not be added unconditionally.
		jobNeeds := []string{"safe_outputs"}
		with := map[string]any{}

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
							// The worker explicitly declares the canonical payload
							// envelope input; forward the raw transport rather than a
							// fromJSON expression.
							with["payload"] = "${{ needs.safe_outputs.outputs.call_workflow_payload }}"
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
			Name:  jobName,
			Needs: jobNeeds,
			If:    fmt.Sprintf("needs.safe_outputs.outputs.call_workflow_name == '%s'", workflowName),
			Uses:  workflowPath,
			With:  with,
		}

		// Infer the minimal set of secrets required by the worker workflow so we can
		// pass them explicitly instead of using secrets: inherit. This requires the
		// worker to have been compiled with on.workflow_call.secrets declarations.
		// If the worker has not yet been compiled (no .lock.yml/.yml), or declares no
		// secrets, fall back to secrets: inherit for backward compatibility.
		if markdownPath != "" {
			workerSecrets, secretsErr := extractCallWorkflowSecrets(workflowName, markdownPath)
			if secretsErr != nil {
				compilerSafeOutputJobsLog.Printf("Warning: could not extract secrets for call-workflow job '%s': %v. "+
					"Falling back to secrets: inherit.", jobName, secretsErr)
				callJob.SecretsInherit = true
			} else if len(workerSecrets) == 0 {
				// No secrets were extracted from the worker. This can mean either the
				// worker declares no workflow_call secrets or its compiled file was not
				// found yet. Fall back to secrets: inherit for backward compatibility.
				compilerSafeOutputJobsLog.Printf("No workflow_call secrets could be extracted for worker '%s' "+
					"(worker may declare none or its compiled file may not exist yet); using secrets: inherit", workflowName)
				callJob.SecretsInherit = true
			} else {
				// Map each declared secret explicitly.
				callJob.Secrets = make(map[string]string, len(workerSecrets))
				for _, s := range workerSecrets {
					callJob.Secrets[s] = fmt.Sprintf("${{ secrets.%s }}", s)
				}
				compilerSafeOutputJobsLog.Printf("Mapped %d explicit secrets for call-workflow job '%s'", len(workerSecrets), jobName)
			}
		} else {
			callJob.SecretsInherit = true
		}

		// Derive the call-<worker> job's permission envelope from the CALLER's own
		// declared permissions, not from the worker. GitHub validates reusable
		// workflow calls against the caller job's declared permissions, so the caller
		// must declare a scope that is sufficient for the worker. We never inflate the
		// caller's permissions to match the worker (doing so would, for example,
		// materialise speculative scopes like vulnerability-alerts that GitHub rejects).
		// Instead the caller controls its own surface and we validate it below.
		callerPerms := data.CachedPermissions
		if callerPerms == nil {
			callerPerms = NewPermissionsParser(data.Permissions).ToPermissions()
		}
		if callerPerms != nil {
			rendered := callerPerms.RenderToYAML()
			if rendered != "" {
				callJob.Permissions = rendered
				compilerSafeOutputJobsLog.Printf("Set permissions on call-workflow job '%s' from caller's declared permissions: %s", jobName, rendered)
			}
		}

		// Validate (without modifying) that the caller's declared permissions cover what
		// the worker requires. Emit a warning when they do not, so the user can widen the
		// caller's `permissions:` block. This never alters the compiled permissions.
		if markdownPath != "" {
			workerPerms, permErr := extractCallWorkflowPermissions(workflowName, markdownPath)
			if permErr != nil {
				// Non-fatal: log and continue. The worker file may not exist yet (it may be
				// compiled in the same batch), in which case validation is simply skipped.
				compilerSafeOutputJobsLog.Printf("Could not extract worker permissions for call-workflow job '%s' (validation skipped): %v", jobName, permErr)
			} else if workerPerms != nil {
				if missing := findUncoveredWorkerPermissions(callerPerms, workerPerms); len(missing) > 0 {
					fmt.Fprintln(os.Stderr, formatCompilerMessage(markdownPath, "warning",
						fmt.Sprintf("call-workflow target '%s' may require permissions not granted by this workflow: %s.\n"+
							"GitHub Actions constrains a called workflow's GITHUB_TOKEN to the caller job's permissions, "+
							"so the worker's jobs may fail. Add the missing scope(s) to this workflow's `permissions:` block.",
							workflowName, strings.Join(missing, ", "))))
					c.IncrementWarningCount()
					compilerSafeOutputJobsLog.Printf("Caller permissions insufficient for worker '%s': missing %s", workflowName, strings.Join(missing, ", "))
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
