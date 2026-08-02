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
	state := &safeOutputsJobBuildState{threatDetectionEnabled: IsDetectionJobEnabled(data.SafeOutputs)}
	if err := c.addDetectionJobIfNeeded(data, state); err != nil {
		return err
	}
	if err := c.addPrimarySafeOutputJobs(data, jobName, markdownPath, state); err != nil {
		return err
	}
	if err := c.addOptionalSafeOutputJobs(data, jobName, markdownPath, state); err != nil {
		return err
	}
	unlockJob, err := c.addUnlockJobIfNeeded(data, state.threatDetectionEnabled)
	if err != nil {
		return err
	}
	return c.addConclusionSafeOutputJob(data, jobName, state, unlockJob)
}

type safeOutputsJobBuildState struct {
	threatDetectionEnabled bool
	safeOutputJobNames     []string
}

func (c *Compiler) addDetectionJobIfNeeded(data *WorkflowData, state *safeOutputsJobBuildState) error {
	if !state.threatDetectionEnabled {
		return nil
	}
	detectionJob, err := c.buildDetectionJob(data)
	if err != nil {
		return fmt.Errorf("failed to build detection job: %w", err)
	}
	if detectionJob == nil {
		return nil
	}
	if err := c.jobManager.AddJob(detectionJob); err != nil {
		return fmt.Errorf("failed to add detection job: %w", err)
	}
	compilerSafeOutputJobsLog.Print("Added separate detection job")
	return nil
}

func (c *Compiler) addPrimarySafeOutputJobs(data *WorkflowData, jobName, markdownPath string, state *safeOutputsJobBuildState) error {
	consolidatedJob, consolidatedStepNames, err := c.buildConsolidatedSafeOutputsJob(data, jobName, markdownPath)
	if err != nil {
		return fmt.Errorf("failed to build consolidated safe outputs job: %w", err)
	}
	if consolidatedJob != nil {
		if err := c.jobManager.AddJob(consolidatedJob); err != nil {
			return fmt.Errorf("failed to add consolidated safe outputs job: %w", err)
		}
		state.safeOutputJobNames = append(state.safeOutputJobNames, consolidatedJob.Name)
		compilerSafeOutputJobsLog.Printf("Added consolidated safe outputs job with %d steps: %v", len(consolidatedStepNames), consolidatedStepNames)
	}
	safeJobNames, err := c.buildSafeJobs(data, state.threatDetectionEnabled)
	if err != nil {
		return fmt.Errorf("failed to build safe-jobs: %w", err)
	}
	state.safeOutputJobNames = append(state.safeOutputJobNames, safeJobNames...)
	compilerSafeOutputJobsLog.Printf("Added %d custom safe-job names to conclusion dependencies", len(safeJobNames))
	return nil
}

func (c *Compiler) addOptionalSafeOutputJobs(data *WorkflowData, jobName, markdownPath string, state *safeOutputsJobBuildState) error {
	if err := c.addUploadAssetsJobIfNeeded(data, jobName, state); err != nil {
		return err
	}
	if err := c.addCodeScanningJobIfNeeded(data, state); err != nil {
		return err
	}
	callWorkflowJobNames, err := c.buildCallWorkflowJobs(data, markdownPath)
	if err != nil {
		return fmt.Errorf("failed to build call-workflow fan-out jobs: %w", err)
	}
	state.safeOutputJobNames = append(state.safeOutputJobNames, callWorkflowJobNames...)
	compilerSafeOutputJobsLog.Printf("Added %d call-workflow fan-out jobs", len(callWorkflowJobNames))
	return nil
}

func (c *Compiler) addUploadAssetsJobIfNeeded(data *WorkflowData, jobName string, state *safeOutputsJobBuildState) error {
	if data.SafeOutputs == nil || data.SafeOutputs.UploadAssets == nil {
		return nil
	}
	compilerSafeOutputJobsLog.Print("Building separate upload_assets job")
	uploadAssetsJob, err := c.buildUploadAssetsJob(data, jobName, state.threatDetectionEnabled)
	if err != nil {
		return fmt.Errorf("failed to build upload_assets job: %w", err)
	}
	if err := c.jobManager.AddJob(uploadAssetsJob); err != nil {
		return fmt.Errorf("failed to add upload_assets job: %w", err)
	}
	state.safeOutputJobNames = append(state.safeOutputJobNames, uploadAssetsJob.Name)
	compilerSafeOutputJobsLog.Print("Added separate upload_assets job")
	return nil
}

func (c *Compiler) addCodeScanningJobIfNeeded(data *WorkflowData, state *safeOutputsJobBuildState) error {
	if data.SafeOutputs == nil || data.SafeOutputs.CreateCodeScanningAlerts == nil ||
		isHandlerStaged(templatableBoolIsTrue(data.SafeOutputs.Staged), data.SafeOutputs.CreateCodeScanningAlerts.Staged) {
		return nil
	}
	compilerSafeOutputJobsLog.Print("Building separate upload_code_scanning_sarif job")
	codeScanningJob, err := c.buildCodeScanningUploadJob(data)
	if err != nil {
		return fmt.Errorf("failed to build upload_code_scanning_sarif job: %w", err)
	}
	if err := c.jobManager.AddJob(codeScanningJob); err != nil {
		return fmt.Errorf("failed to add upload_code_scanning_sarif job: %w", err)
	}
	state.safeOutputJobNames = append(state.safeOutputJobNames, codeScanningJob.Name)
	compilerSafeOutputJobsLog.Print("Added separate upload_code_scanning_sarif job")
	return nil
}

func (c *Compiler) addUnlockJobIfNeeded(data *WorkflowData, threatDetectionEnabled bool) (*Job, error) {
	unlockJob, err := c.buildUnlockJob(data, threatDetectionEnabled)
	if err != nil {
		return nil, fmt.Errorf("failed to build unlock job: %w", err)
	}
	if unlockJob == nil {
		return nil, nil
	}
	if err := c.jobManager.AddJob(unlockJob); err != nil {
		return nil, fmt.Errorf("failed to add unlock job: %w", err)
	}
	compilerSafeOutputJobsLog.Print("Added dedicated unlock job")
	return unlockJob, nil
}

func (c *Compiler) addConclusionSafeOutputJob(data *WorkflowData, jobName string, state *safeOutputsJobBuildState, unlockJob *Job) error {
	conclusionJob, err := c.buildConclusionJob(data, jobName, state.safeOutputJobNames)
	if err != nil {
		return fmt.Errorf("failed to build conclusion job: %w", err)
	}
	if conclusionJob == nil {
		return nil
	}
	if unlockJob != nil {
		conclusionJob.Needs = append(conclusionJob.Needs, "unlock")
		compilerSafeOutputJobsLog.Printf("Added unlock job dependency to conclusion job")
	}
	if _, exists := c.jobManager.GetJob("push_repo_memory"); exists {
		conclusionJob.Needs = append(conclusionJob.Needs, "push_repo_memory")
		compilerSafeOutputJobsLog.Printf("Added push_repo_memory dependency to conclusion job")
	}
	if err := c.jobManager.AddJob(conclusionJob); err != nil {
		return fmt.Errorf("failed to add conclusion job: %w", err)
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
//   - includes a job-level `permissions:` block equal to the union of the
//     caller's declared permissions and the called worker's required permissions
//   - adds a help comment explaining why imported worker permissions appear on
//     the call job and where to review them in the worker workflow source
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
	jobNames := make([]string, 0, len(config.Workflows))
	for _, workflowName := range config.Workflows {
		callJob, jobName, err := c.buildSingleCallWorkflowJob(data, markdownPath, workflowName, config)
		if err != nil {
			return nil, err
		}
		if err := c.jobManager.AddJob(callJob); err != nil {
			return nil, fmt.Errorf("failed to add call-workflow job '%s': %w", jobName, err)
		}
		jobNames = append(jobNames, jobName)
		compilerSafeOutputJobsLog.Printf("Added call-workflow job: %s (uses: %s)", jobName, callJob.Uses)
	}
	return jobNames, nil
}

func (c *Compiler) buildSingleCallWorkflowJob(data *WorkflowData, markdownPath, workflowName string, config *CallWorkflowConfig) (*Job, string, error) {
	jobName := "call-" + sanitizeJobName(workflowName)
	callJob := &Job{
		Name:  jobName,
		Needs: []string{"safe_outputs"},
		If:    fmt.Sprintf("needs.safe_outputs.outputs.call_workflow_name == '%s'", workflowName),
		Uses:  resolveCallWorkflowPath(config, workflowName),
		With:  buildCallWorkflowInputs(workflowName, markdownPath, jobName),
	}
	c.configureCallWorkflowSecrets(callJob, workflowName, markdownPath, jobName)
	c.applyCallWorkflowPermissions(callJob, data, workflowName, markdownPath, jobName)
	return callJob, jobName, nil
}

func resolveCallWorkflowPath(config *CallWorkflowConfig, workflowName string) string {
	if workflowPath := config.WorkflowFiles[workflowName]; workflowPath != "" {
		return workflowPath
	}
	return fmt.Sprintf("./.github/workflows/%s.lock.yml", workflowName)
}

func buildCallWorkflowInputs(workflowName, markdownPath, jobName string) map[string]any {
	if markdownPath == "" {
		return map[string]any{}
	}
	fileResult, findErr := findWorkflowFile(workflowName, markdownPath)
	if findErr != nil {
		compilerSafeOutputJobsLog.Printf("Warning: could not find worker workflow file for '%s': %v. Typed inputs will not be forwarded in the with: block.", workflowName, findErr)
		return map[string]any{}
	}
	workflowInputs, err := loadCallWorkflowInputs(fileResult, workflowName)
	if err != nil || workflowInputs == nil {
		if err != nil {
			compilerSafeOutputJobsLog.Printf("Warning: could not extract workflow_call inputs for '%s': %v. Typed inputs will not be forwarded in the with: block.", workflowName, err)
		}
		return map[string]any{}
	}
	with := map[string]any{}
	typedInputCount := 0
	for inputName := range workflowInputs {
		if inputName == "payload" {
			with["payload"] = "${{ needs.safe_outputs.outputs.call_workflow_payload }}"
			continue
		}
		with[inputName] = buildCallWorkflowInputExpression(inputName)
		typedInputCount++
	}
	compilerSafeOutputJobsLog.Printf("Forwarding %d typed inputs for call-workflow job '%s'", typedInputCount, jobName)
	return with
}

func loadCallWorkflowInputs(fileResult *findWorkflowFileResult, workflowName string) (map[string]any, error) {
	switch {
	case fileResult.lockExists:
		return extractWorkflowCallInputs(fileResult.lockPath)
	case fileResult.ymlExists:
		return extractWorkflowCallInputs(fileResult.ymlPath)
	case fileResult.mdExists:
		return extractMDWorkflowCallInputs(fileResult.mdPath)
	default:
		compilerSafeOutputJobsLog.Printf("Warning: no worker file found for '%s'; typed inputs will not be forwarded in the with: block.", workflowName)
		return nil, nil
	}
}

func (c *Compiler) configureCallWorkflowSecrets(callJob *Job, workflowName, markdownPath, jobName string) {
	if markdownPath == "" {
		callJob.SecretsInherit = true
		return
	}
	workerSecrets, secretsErr := extractCallWorkflowSecrets(workflowName, markdownPath)
	if secretsErr != nil {
		compilerSafeOutputJobsLog.Printf("Warning: could not extract secrets for call-workflow job '%s': %v. Falling back to secrets: inherit.", jobName, secretsErr)
		callJob.SecretsInherit = true
		return
	}
	if len(workerSecrets) == 0 {
		compilerSafeOutputJobsLog.Printf("No workflow_call secrets could be extracted for worker '%s' (worker may declare none or its compiled file may not exist yet); using secrets: inherit", workflowName)
		callJob.SecretsInherit = true
		return
	}
	callJob.Secrets = make(map[string]string, len(workerSecrets))
	for _, secret := range workerSecrets {
		callJob.Secrets[secret] = fmt.Sprintf("${{ secrets.%s }}", secret)
	}
	compilerSafeOutputJobsLog.Printf("Mapped %d explicit secrets for call-workflow job '%s'", len(workerSecrets), jobName)
}

func (c *Compiler) applyCallWorkflowPermissions(callJob *Job, data *WorkflowData, workflowName, markdownPath, jobName string) {
	effectivePerms, importedPerms := computeCallWorkflowPermissions(data, workflowName, markdownPath, jobName)
	if effectivePerms == nil {
		return
	}
	if rendered := effectivePerms.RenderToYAML(); rendered != "" {
		callJob.PermissionsComment = buildCallWorkflowPermissionsComment(workflowName, importedPerms)
		callJob.Permissions = rendered
		compilerSafeOutputJobsLog.Printf("Set permissions on call-workflow job '%s': %s", jobName, rendered)
	}
}

func computeCallWorkflowPermissions(data *WorkflowData, workflowName, markdownPath, jobName string) (*Permissions, *callWorkflowPermissionImport) {
	callerPerms := data.CachedPermissions
	if callerPerms == nil {
		callerPerms = NewPermissionsParser(data.Permissions).ToPermissions()
	}
	if markdownPath == "" {
		return callerPerms, nil
	}
	importedPerms, permErr := extractCallWorkflowPermissionImport(workflowName, markdownPath)
	if permErr != nil {
		compilerSafeOutputJobsLog.Printf("Could not extract worker permissions for call-workflow job '%s' (falling back to caller-only permissions): %v", jobName, permErr)
		return callerPerms, nil
	}
	if importedPerms == nil || importedPerms.permissions == nil {
		return callerPerms, importedPerms
	}
	merged := NewPermissions()
	merged.Merge(callerPerms)
	merged.Merge(importedPerms.permissions)
	compilerSafeOutputJobsLog.Printf("Merged caller and worker permissions for call-workflow job '%s'", jobName)
	return merged, importedPerms
}

func buildCallWorkflowInputExpression(inputName string) string {
	payloadExpr := "fromJSON(needs.safe_outputs.outputs.call_workflow_payload)"
	if isBareActionsIdentifier(inputName) {
		return fmt.Sprintf("${{ %s.%s }}", payloadExpr, inputName)
	}

	escapedInputName := escapeActionsSingleQuotedString(inputName)
	return fmt.Sprintf("${{ %s['%s'] }}", payloadExpr, escapedInputName)
}

// isBareActionsIdentifier reports whether a name can be safely referenced via
// dot access in GitHub Actions expressions (letters/underscore followed by
// letters, digits, or underscore).
func isBareActionsIdentifier(name string) bool {
	if name == "" {
		return false
	}

	for i, r := range name {
		if i == 0 {
			if r != '_' && (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') {
				return false
			}
			continue
		}

		if r != '_' && (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return false
		}
	}

	return true
}
