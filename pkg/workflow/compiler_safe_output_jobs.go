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

	threatDetectionEnabled := IsDetectionJobEnabled(data.SafeOutputs)
	if err := c.addDetectionSafeOutputJob(data, threatDetectionEnabled); err != nil {
		return err
	}

	safeOutputJobNames, err := c.buildPrimarySafeOutputJobNames(data, jobName, markdownPath, threatDetectionEnabled)
	if err != nil {
		return err
	}

	unlockJob, err := c.addUnlockSafeOutputJob(data, threatDetectionEnabled)
	if err != nil {
		return err
	}

	return c.addConclusionSafeOutputJob(data, jobName, safeOutputJobNames, unlockJob)
}

func (c *Compiler) addDetectionSafeOutputJob(data *WorkflowData, threatDetectionEnabled bool) error {
	if !threatDetectionEnabled {
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

func (c *Compiler) buildPrimarySafeOutputJobNames(data *WorkflowData, jobName, markdownPath string, threatDetectionEnabled bool) ([]string, error) {
	var safeOutputJobNames []string
	consolidatedJob, consolidatedStepNames, err := c.buildConsolidatedSafeOutputsJob(data, jobName, markdownPath)
	if err != nil {
		return nil, fmt.Errorf("failed to build consolidated safe outputs job: %w", err)
	}
	if consolidatedJob != nil {
		if err := c.jobManager.AddJob(consolidatedJob); err != nil {
			return nil, fmt.Errorf("failed to add consolidated safe outputs job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, consolidatedJob.Name)
		compilerSafeOutputJobsLog.Printf("Added consolidated safe outputs job with %d steps: %v", len(consolidatedStepNames), consolidatedStepNames)
	}

	safeJobNames, err := c.buildSafeJobs(data, threatDetectionEnabled)
	if err != nil {
		return nil, fmt.Errorf("failed to build safe-jobs: %w", err)
	}
	safeOutputJobNames = append(safeOutputJobNames, safeJobNames...)
	compilerSafeOutputJobsLog.Printf("Added %d custom safe-job names to conclusion dependencies", len(safeJobNames))

	safeOutputJobNames, err = c.appendUploadAssetsJobName(data, jobName, threatDetectionEnabled, safeOutputJobNames)
	if err != nil {
		return nil, err
	}
	safeOutputJobNames, err = c.appendCodeScanningUploadJobName(data, safeOutputJobNames)
	if err != nil {
		return nil, err
	}
	callWorkflowJobNames, err := c.buildCallWorkflowJobs(data, markdownPath)
	if err != nil {
		return nil, fmt.Errorf("failed to build call-workflow fan-out jobs: %w", err)
	}
	safeOutputJobNames = append(safeOutputJobNames, callWorkflowJobNames...)
	compilerSafeOutputJobsLog.Printf("Added %d call-workflow fan-out jobs", len(callWorkflowJobNames))
	return safeOutputJobNames, nil
}

func (c *Compiler) appendUploadAssetsJobName(data *WorkflowData, jobName string, threatDetectionEnabled bool, safeOutputJobNames []string) ([]string, error) {
	if data.SafeOutputs != nil && data.SafeOutputs.UploadAssets != nil {
		compilerSafeOutputJobsLog.Print("Building separate upload_assets job")
		uploadAssetsJob, err := c.buildUploadAssetsJob(data, jobName, threatDetectionEnabled)
		if err != nil {
			return nil, fmt.Errorf("failed to build upload_assets job: %w", err)
		}
		if err := c.jobManager.AddJob(uploadAssetsJob); err != nil {
			return nil, fmt.Errorf("failed to add upload_assets job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, uploadAssetsJob.Name)
		compilerSafeOutputJobsLog.Printf("Added separate upload_assets job")
	}
	return safeOutputJobNames, nil
}

func (c *Compiler) appendCodeScanningUploadJobName(data *WorkflowData, safeOutputJobNames []string) ([]string, error) {
	if data.SafeOutputs != nil && data.SafeOutputs.CreateCodeScanningAlerts != nil &&
		!isHandlerStaged(templatableBoolIsTrue(data.SafeOutputs.Staged), data.SafeOutputs.CreateCodeScanningAlerts.Staged) {
		compilerSafeOutputJobsLog.Print("Building separate upload_code_scanning_sarif job")
		codeScanningJob, err := c.buildCodeScanningUploadJob(data)
		if err != nil {
			return nil, fmt.Errorf("failed to build upload_code_scanning_sarif job: %w", err)
		}
		if err := c.jobManager.AddJob(codeScanningJob); err != nil {
			return nil, fmt.Errorf("failed to add upload_code_scanning_sarif job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, codeScanningJob.Name)
		compilerSafeOutputJobsLog.Printf("Added separate upload_code_scanning_sarif job")
	}
	return safeOutputJobNames, nil
}

func (c *Compiler) addUnlockSafeOutputJob(data *WorkflowData, threatDetectionEnabled bool) (*Job, error) {
	unlockJob, err := c.buildUnlockJob(data, threatDetectionEnabled)
	if err != nil {
		return nil, fmt.Errorf("failed to build unlock job: %w", err)
	}
	if unlockJob != nil {
		if err := c.jobManager.AddJob(unlockJob); err != nil {
			return nil, fmt.Errorf("failed to add unlock job: %w", err)
		}
		compilerSafeOutputJobsLog.Print("Added dedicated unlock job")
	}
	return unlockJob, nil
}

func (c *Compiler) addConclusionSafeOutputJob(data *WorkflowData, jobName string, safeOutputJobNames []string, unlockJob *Job) error {
	conclusionJob, err := c.buildConclusionJob(data, jobName, safeOutputJobNames)
	if err != nil {
		return fmt.Errorf("failed to build conclusion job: %w", err)
	}
	if conclusionJob != nil {
		c.addConclusionJobDependencies(conclusionJob, unlockJob)
		if err := c.jobManager.AddJob(conclusionJob); err != nil {
			return fmt.Errorf("failed to add conclusion job: %w", err)
		}
	}
	return nil
}

func (c *Compiler) addConclusionJobDependencies(conclusionJob *Job, unlockJob *Job) {
	if unlockJob != nil {
		conclusionJob.Needs = append(conclusionJob.Needs, "unlock")
		compilerSafeOutputJobsLog.Printf("Added unlock job dependency to conclusion job")
	}
	if _, exists := c.jobManager.GetJob("push_repo_memory"); exists {
		conclusionJob.Needs = append(conclusionJob.Needs, "push_repo_memory")
		compilerSafeOutputJobsLog.Printf("Added push_repo_memory dependency to conclusion job")
	}
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

	var jobNames []string

	for _, workflowName := range config.Workflows {
		jobName, err := c.addCallWorkflowJob(data, config, workflowName, markdownPath)
		if err != nil {
			return nil, err
		}
		jobNames = append(jobNames, jobName)
	}

	return jobNames, nil
}

func (c *Compiler) addCallWorkflowJob(data *WorkflowData, config *CallWorkflowConfig, workflowName, markdownPath string) (string, error) {
	jobName := "call-" + sanitizeJobName(workflowName)
	workflowPath := callWorkflowPath(config, workflowName)
	callJob := &Job{
		Name:  jobName,
		Needs: []string{"safe_outputs"},
		If:    fmt.Sprintf("needs.safe_outputs.outputs.call_workflow_name == '%s'", workflowName),
		Uses:  workflowPath,
		With:  buildCallWorkflowWith(workflowName, jobName, markdownPath),
	}
	configureCallWorkflowSecrets(callJob, workflowName, jobName, markdownPath)
	configureCallWorkflowPermissions(callJob, data, workflowName, jobName, markdownPath)
	if err := c.jobManager.AddJob(callJob); err != nil {
		return "", fmt.Errorf("failed to add call-workflow job '%s': %w", jobName, err)
	}
	compilerSafeOutputJobsLog.Printf("Added call-workflow job: %s (uses: %s)", jobName, workflowPath)
	return jobName, nil
}

func callWorkflowPath(config *CallWorkflowConfig, workflowName string) string {
	if workflowPath, ok := config.WorkflowFiles[workflowName]; ok && workflowPath != "" {
		return workflowPath
	}
	return fmt.Sprintf("./.github/workflows/%s.lock.yml", workflowName)
}

func buildCallWorkflowWith(workflowName, jobName, markdownPath string) map[string]any {
	with := map[string]any{}
	if markdownPath == "" {
		return with
	}
	workflowInputs := extractCallWorkflowInputsForJob(workflowName, markdownPath)
	if workflowInputs == nil {
		return with
	}
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

func extractCallWorkflowInputsForJob(workflowName, markdownPath string) map[string]any {
	fileResult, findErr := findWorkflowFile(workflowName, markdownPath)
	if findErr != nil {
		compilerSafeOutputJobsLog.Printf("Warning: could not find worker workflow file for '%s': %v. "+
			"Typed inputs will not be forwarded in the with: block.", workflowName, findErr)
		return nil
	}
	workflowInputs, inputErr := extractCallWorkflowInputsFromFileResult(fileResult, workflowName)
	if inputErr != nil {
		compilerSafeOutputJobsLog.Printf("Warning: could not extract workflow_call inputs for '%s': %v. "+
			"Typed inputs will not be forwarded in the with: block.", workflowName, inputErr)
		return nil
	}
	return workflowInputs
}

func extractCallWorkflowInputsFromFileResult(fileResult *findWorkflowFileResult, workflowName string) (map[string]any, error) {
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

func configureCallWorkflowSecrets(callJob *Job, workflowName, jobName, markdownPath string) {
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
	for _, s := range workerSecrets {
		callJob.Secrets[s] = fmt.Sprintf("${{ secrets.%s }}", s)
	}
	compilerSafeOutputJobsLog.Printf("Mapped %d explicit secrets for call-workflow job '%s'", len(workerSecrets), jobName)
}

func configureCallWorkflowPermissions(callJob *Job, data *WorkflowData, workflowName, jobName, markdownPath string) {
	callerPerms := data.CachedPermissions
	if callerPerms == nil {
		callerPerms = NewPermissionsParser(data.Permissions).ToPermissions()
	}
	effectivePerms, importedPerms := callWorkflowEffectivePermissions(callerPerms, workflowName, jobName, markdownPath)
	if effectivePerms == nil {
		return
	}
	rendered := effectivePerms.RenderToYAML()
	if rendered == "" {
		return
	}
	callJob.PermissionsComment = buildCallWorkflowPermissionsComment(workflowName, importedPerms)
	callJob.Permissions = rendered
	compilerSafeOutputJobsLog.Printf("Set permissions on call-workflow job '%s': %s", jobName, rendered)
}

func callWorkflowEffectivePermissions(callerPerms *Permissions, workflowName, jobName, markdownPath string) (*Permissions, *callWorkflowPermissionImport) {
	effectivePerms := callerPerms
	var importedPerms *callWorkflowPermissionImport
	if markdownPath == "" {
		return effectivePerms, importedPerms
	}
	importedPerms, permErr := extractCallWorkflowPermissionImport(workflowName, markdownPath)
	if permErr != nil {
		compilerSafeOutputJobsLog.Printf("Could not extract worker permissions for call-workflow job '%s' (falling back to caller-only permissions): %v", jobName, permErr)
		return effectivePerms, importedPerms
	}
	if importedPerms != nil && importedPerms.permissions != nil {
		merged := NewPermissions()
		merged.Merge(callerPerms)
		merged.Merge(importedPerms.permissions)
		effectivePerms = merged
		compilerSafeOutputJobsLog.Printf("Merged caller and worker permissions for call-workflow job '%s'", jobName)
	}
	return effectivePerms, importedPerms
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
