package workflow

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/setutil"
	"github.com/github/gh-aw/pkg/sliceutil"
	"github.com/github/gh-aw/pkg/stringutil"
)

var consolidatedSafeOutputsJobLog = logger.New("workflow:compiler_safe_outputs_job")

// stepNameLinePrefix matches the canonical YAML line emitted by this compiler for
// step starts in job.Steps (6-space indent + "- name: ").
const stepNameLinePrefix = "      - name: "

// getSafeOutputsHeadApp returns the first non-nil HeadGitHubApp config from
// create-pull-request or push-to-pull-request-branch handlers, used to generate
// the safe-outputs-head-app-token step.
func getSafeOutputsHeadApp(safeOutputs *SafeOutputsConfig) *GitHubAppConfig {
	if safeOutputs == nil {
		return nil
	}
	if safeOutputs.CreatePullRequests != nil && safeOutputs.CreatePullRequests.HeadGitHubApp != nil {
		return safeOutputs.CreatePullRequests.HeadGitHubApp
	}
	if safeOutputs.PushToPullRequestBranch != nil && safeOutputs.PushToPullRequestBranch.HeadGitHubApp != nil {
		return safeOutputs.PushToPullRequestBranch.HeadGitHubApp
	}
	return nil
}

// getSafeOutputsHeadRepoSlug returns the HeadRepoSlug associated with the configured
// head-github-app, used to scope the minted token to the correct fork repository.
func getSafeOutputsHeadRepoSlug(safeOutputs *SafeOutputsConfig) string {
	if safeOutputs == nil {
		return ""
	}
	if safeOutputs.CreatePullRequests != nil && safeOutputs.CreatePullRequests.HeadGitHubApp != nil {
		return safeOutputs.CreatePullRequests.HeadRepoSlug
	}
	if safeOutputs.PushToPullRequestBranch != nil && safeOutputs.PushToPullRequestBranch.HeadGitHubApp != nil {
		return safeOutputs.PushToPullRequestBranch.HeadRepoSlug
	}
	return ""
}

// headRepoNameFromSlug extracts the repository name (without owner) from an "owner/repo"
// slug for use as the fallback repositories value in app token minting.
// Returns an empty string when the slug is absent, cannot be parsed, or contains an expression.
// The expression guard uses "${{" as a conservative prefix; standard GitHub Actions expressions
// always begin with exactly this prefix so this check is sufficient in practice.
func headRepoNameFromSlug(slug string) string {
	if slug == "" {
		return ""
	}
	parts := strings.SplitN(slug, "/", 2)
	if len(parts) == 2 && !strings.Contains(parts[1], "${{") {
		return parts[1]
	}
	return ""
}

// messagesContainPreActivationRef reports whether any message template in cfg
// contains a reference to a needs.pre_activation.outputs.* expression.
// When true, the safe_outputs and conclusion jobs must declare pre_activation
// in their needs so that GitHub Actions can resolve the expression at runtime.
func messagesContainPreActivationRef(cfg *SafeOutputMessagesConfig) bool {
	if cfg == nil {
		return false
	}
	preActivationOutputsRef := "needs." + string(constants.PreActivationJobName) + ".outputs."
	for _, field := range []string{
		cfg.Footer,
		cfg.FooterInstall,
		cfg.FooterWorkflowRecompile,
		cfg.FooterWorkflowRecompileComment,
		cfg.StagedTitle,
		cfg.StagedDescription,
		cfg.ActivationComments,
		cfg.RunStarted,
		cfg.RunSuccess,
		cfg.RunFailure,
		cfg.DetectionFailure,
		cfg.PullRequestCreated,
		cfg.IssueCreated,
		cfg.CommitPushed,
		cfg.AgentFailureIssue,
		cfg.AgentFailureComment,
		cfg.BodyHeader,
	} {
		if strings.Contains(field, preActivationOutputsRef) {
			return true
		}
	}
	return false
}

// buildConsolidatedSafeOutputsJob builds a single job containing all safe output operations
// as separate steps within that job. This reduces the number of jobs in the workflow
// while maintaining observability through distinct step names, IDs, and outputs.
//
// File mode: Instead of inlining bundled JavaScript in YAML, this function:
// 1. Collects all JavaScript files needed by enabled safe outputs
// 2. Generates a "Setup JavaScript files" step to write them to /tmp/gh-aw/scripts/
// 3. Each safe output step requires from the local filesystem
func (c *Compiler) buildConsolidatedSafeOutputsJob(data *WorkflowData, mainJobName, markdownPath string) (*Job, []string, error) {
	if data.SafeOutputs == nil {
		consolidatedSafeOutputsJobLog.Print("No safe outputs configured, skipping consolidated job")
		return nil, nil, nil
	}

	consolidatedSafeOutputsJobLog.Print("Building consolidated safe outputs job with file mode")

	// Compute permissions and threat detection flag up front; both are used across phases.
	permissions := ComputePermissionsForSafeOutputs(data.SafeOutputs)
	// When observability.otlp.github-app is configured without app-id/private-key
	// credentials, id-token: write is needed so the safe_outputs job can mint the OTLP
	// OIDC token via core.getIDToken(audience) (mirrors threat_detection_job.go).
	if hasOTLPGitHubOIDCAuth(data.ParsedFrontmatter, data.RawFrontmatter) {
		permissions.Set(PermissionIdToken, PermissionWrite)
	}
	threatDetectionEnabled := IsDetectionJobEnabled(data.SafeOutputs)

	// Compute artifact prefix once; it is referenced in all three phases.
	agentArtifactPrefix := artifactPrefixExprForDownstreamJob(data)

	// Phase 1: Setup action, artifact downloads, and user-provided steps
	setupSteps, err := c.buildSafeOutputsSetupAndDownloadSteps(data, agentArtifactPrefix)
	if err != nil {
		return nil, nil, err
	}

	// Phase 2: Handler manager, SARIF, custom actions, and named outputs
	handlerSteps, outputs, safeOutputStepNames, err := c.buildSafeOutputsHandlerOutputsAndActionSteps(data, agentArtifactPrefix, markdownPath)
	if err != nil {
		return nil, nil, err
	}

	// Early return when no safe output handler steps were emitted
	if len(safeOutputStepNames) == 0 {
		consolidatedSafeOutputsJobLog.Print("No safe output steps were added")
		return nil, nil, nil
	}

	// Combine the setup steps with the handler steps
	steps := append(setupSteps, handlerSteps...)

	// Phase 3: App-token insertion, finalization, job condition/deps, and job construction
	return c.buildSafeOutputsJobFromParts(buildSafeOutputsJobFromPartsOptions{
		data:                   data,
		mainJobName:            mainJobName,
		markdownPath:           markdownPath,
		agentArtifactPrefix:    agentArtifactPrefix,
		steps:                  steps,
		outputs:                outputs,
		safeOutputStepNames:    safeOutputStepNames,
		permissions:            permissions,
		threatDetectionEnabled: threatDetectionEnabled,
	})
}

// buildSafeOutputsSetupAndDownloadSteps builds the initial steps for the consolidated safe
// outputs job: setup action (with optional actions-folder checkout), OTLP header masking,
// agent artifact downloads, patch artifact download (when PR operations are configured),
// shared PR checkout, GH Enterprise host configuration, and user-provided steps.
func (c *Compiler) buildSafeOutputsSetupAndDownloadSteps(data *WorkflowData, agentArtifactPrefix string) ([]string, error) {
	var steps []string

	steps = c.appendSafeOutputsSetupActionSteps(steps, data)
	steps = appendSafeOutputsTelemetryMaskSteps(steps, data)
	steps = append(steps, buildAgentOutputDownloadSteps(agentArtifactPrefix, c.getActionPin)...)
	steps = c.appendSafeOutputsPatchCheckoutSteps(steps, data, agentArtifactPrefix)
	steps = append(steps, generateGHESHostConfigurationStep())
	return c.appendUserProvidedSafeOutputSteps(steps, data)
}

func (c *Compiler) appendSafeOutputsSetupActionSteps(steps []string, data *WorkflowData) []string {
	setupActionRef := c.resolveActionReference("./actions/setup", data)
	if setupActionRef != "" || c.actionMode.IsScript() {
		// For dev mode (local action path), checkout the actions folder first
		steps = append(steps, c.generateCheckoutActionsFolder(data)...)

		// Enable artifact client flag if upload-artifact safe output is configured
		enableArtifactClient := data.SafeOutputs != nil && data.SafeOutputs.UploadArtifact != nil

		// Safe outputs job depends on agent job; reuse the agent's trace ID so all jobs share one OTLP trace
		safeOutputsTraceID := fmt.Sprintf("${{ needs.%s.outputs.setup-trace-id }}", constants.ActivationJobName)
		safeOutputsParentSpanID := setupParentSpanNeedsExpr(constants.ActivationJobName)
		steps = append(steps, c.generateSetupStep(data, setupActionRef, SetupActionDestination, enableArtifactClient, safeOutputsTraceID, safeOutputsParentSpanID)...)
	}
	return steps
}

func appendSafeOutputsTelemetryMaskSteps(steps []string, data *WorkflowData) []string {
	// Mask OTLP telemetry headers immediately after setup so authentication tokens cannot
	// leak into runner debug logs for any subsequent step in the safe outputs job.
	if isOTLPHeadersPresent(data) {
		steps = append(steps, generateOTLPHeadersMaskStep())
	}
	// Mask custom OTLP attribute values so user-supplied values cannot leak into runner logs.
	if isOTLPAttributesPresent(data) {
		steps = append(steps, generateOTLPAttributesMaskStep())
	}
	return steps
}

func (c *Compiler) appendSafeOutputsPatchCheckoutSteps(steps []string, data *WorkflowData, agentArtifactPrefix string) []string {
	// Add patch artifact download if create-pull-request or push-to-pull-request-branch is enabled
	// Both of these safe outputs require the patch file to apply changes
	// Download from unified agent artifact (prefixed in workflow_call context)
	if usesPatchesAndCheckouts(data.SafeOutputs) {
		consolidatedSafeOutputsJobLog.Print("Adding patch artifact download for create-pull-request or push-to-pull-request-branch")
		patchDownloadSteps := buildArtifactDownloadSteps(ArtifactDownloadConfig{
			ArtifactName: agentArtifactPrefix + constants.AgentArtifactName,
			DownloadPath: constants.TmpGhAwDirSlash,
			SetupEnvStep: false, // No environment variable needed, the script checks the file directly
			StepName:     "Download patch artifact",
		}, c.getActionPin)
		steps = append(steps, patchDownloadSteps...)

		// Add checkout and git config steps for PR operations. These mirror the agent job's
		// checkout layout exactly (same CheckoutManager generators); the base branch is
		// resolved by the JS handler at apply time, so no checkout-time base ref is needed.
		consolidatedSafeOutputsJobLog.Print("Adding shared checkout step for PR operations")
		checkoutSteps := c.buildSharedPRCheckoutSteps(data)
		steps = append(steps, checkoutSteps...)
	}
	return steps
}

func (c *Compiler) appendUserProvidedSafeOutputSteps(steps []string, data *WorkflowData) ([]string, error) {
	// Add user-provided steps after checkout/setup, before safe-output code
	if len(data.SafeOutputs.Steps) > 0 {
		consolidatedSafeOutputsJobLog.Printf("Adding %d user-provided steps to safe-outputs job", len(data.SafeOutputs.Steps))
		for i, step := range data.SafeOutputs.Steps {
			stepMap, ok := step.(map[string]any)
			if !ok {
				consolidatedSafeOutputsJobLog.Printf("Warning: safe-outputs step at index %d is not a valid step object (must be a map with properties like name, run, uses). Skipping this step.", i)
				continue
			}
			typedStep, err := MapToStep(stepMap)
			if err != nil {
				return nil, fmt.Errorf("failed to convert safe-outputs step at index %d to typed step: %w", i, err)
			}
			pinnedStep, err := applyActionPinToTypedStep(typedStep, data)
			if err != nil {
				return nil, fmt.Errorf("failed to pin action for safe-outputs step at index %d: %w", i, err)
			}
			stepYAML, err := ConvertStepToYAML(pinnedStep.ToMap())
			if err != nil {
				return nil, fmt.Errorf("failed to convert safe-outputs step at index %d to YAML: %w", i, err)
			}
			steps = append(steps, stepYAML)
		}
	}

	return steps, nil
}

// buildSafeOutputsHandlerOutputsAndActionSteps builds the handler-manager step (if needed),
// all job-level outputs derived from the handler, SARIF artifact upload, custom action steps,
// and the named convenience outputs for first-created items.
// It returns the collected steps, outputs map, and the list of safe-output step names registered.
func (c *Compiler) buildSafeOutputsHandlerOutputsAndActionSteps(data *WorkflowData, agentArtifactPrefix, markdownPath string) ([]string, map[string]string, []string, error) {
	var steps []string
	outputs := make(map[string]string)
	var safeOutputStepNames []string

	var err error
	steps, err = c.appendCustomSafeOutputScriptSteps(steps, data)
	if err != nil {
		return nil, nil, nil, err
	}
	steps = c.appendUploadArtifactStagingDownloadStep(steps, data, agentArtifactPrefix)
	steps, safeOutputStepNames, err = c.appendHandlerManagerSafeOutputSteps(steps, data, outputs, safeOutputStepNames)
	if err != nil {
		return nil, nil, nil, err
	}
	steps = c.appendSarifArtifactSafeOutputSteps(steps, data, outputs, agentArtifactPrefix)
	steps, safeOutputStepNames = c.appendCustomActionSafeOutputSteps(steps, data, markdownPath, safeOutputStepNames)
	addNamedSafeOutputJobOutputs(data, outputs)

	return steps, outputs, safeOutputStepNames, nil
}

func hasHandlerManagerSafeOutputTypes(safeOutputs *SafeOutputsConfig) bool {
	return safeOutputs.CreateIssues != nil ||
		safeOutputs.AddComments != nil ||
		safeOutputs.CreateDiscussions != nil ||
		safeOutputs.CloseIssues != nil ||
		safeOutputs.CloseDiscussions != nil ||
		safeOutputs.AddLabels != nil ||
		safeOutputs.RemoveLabels != nil ||
		safeOutputs.UpdateIssues != nil ||
		safeOutputs.UpdateDiscussions != nil ||
		safeOutputs.LinkSubIssue != nil ||
		safeOutputs.UpdateRelease != nil ||
		safeOutputs.CreatePullRequestReviewComments != nil ||
		safeOutputs.SubmitPullRequestReview != nil ||
		safeOutputs.ReplyToPullRequestReviewComment != nil ||
		safeOutputs.ResolvePullRequestReviewThread != nil ||
		safeOutputs.CreatePullRequests != nil ||
		safeOutputs.PushToPullRequestBranch != nil ||
		safeOutputs.UpdatePullRequests != nil ||
		safeOutputs.ClosePullRequests != nil ||
		safeOutputs.MarkPullRequestAsReadyForReview != nil ||
		safeOutputs.HideComment != nil ||
		safeOutputs.SetIssueType != nil ||
		safeOutputs.SetIssueField != nil ||
		safeOutputs.DispatchWorkflow != nil ||
		safeOutputs.CallWorkflow != nil ||
		safeOutputs.CreateCodeScanningAlerts != nil ||
		safeOutputs.AutofixCodeScanningAlert != nil ||
		safeOutputs.CreateCheckRun != nil ||
		safeOutputs.MissingTool != nil ||
		safeOutputs.MissingData != nil ||
		safeOutputs.AssignToAgent != nil ||
		safeOutputs.CreateAgentSessions != nil ||
		safeOutputs.UploadArtifact != nil ||
		len(safeOutputs.Scripts) > 0 ||
		len(safeOutputs.Actions) > 0
}

func (c *Compiler) appendCustomSafeOutputScriptSteps(steps []string, data *WorkflowData) ([]string, error) {
	if len(data.SafeOutputs.Scripts) == 0 {
		return steps, nil
	}
	consolidatedSafeOutputsJobLog.Printf("Adding setup step for %d custom safe-output script(s)", len(data.SafeOutputs.Scripts))
	scriptSetupSteps, err := buildCustomScriptFilesStep(data.SafeOutputs.Scripts)
	if err != nil {
		return nil, fmt.Errorf("failed to build custom script files step: %w", err)
	}
	return append(steps, scriptSetupSteps...), nil
}

func (c *Compiler) appendUploadArtifactStagingDownloadStep(steps []string, data *WorkflowData, agentArtifactPrefix string) []string {
	if data.SafeOutputs.UploadArtifact == nil {
		return steps
	}
	consolidatedSafeOutputsJobLog.Print("Adding upload-artifact staging download step")
	stagingArtifactName := agentArtifactPrefix + SafeOutputsUploadArtifactStagingArtifactName
	return append(steps,
		"      - name: Download upload-artifact staging\n",
		"        continue-on-error: true\n",
		fmt.Sprintf("        uses: %s\n", c.getActionPin("actions/download-artifact")),
		"        with:\n",
		fmt.Sprintf("          name: %s\n", stagingArtifactName),
		fmt.Sprintf("          path: %s\n", artifactStagingDirExpr),
	)
}

func (c *Compiler) appendHandlerManagerSafeOutputSteps(steps []string, data *WorkflowData, outputs map[string]string, safeOutputStepNames []string) ([]string, []string, error) {
	if !hasHandlerManagerSafeOutputTypes(data.SafeOutputs) {
		return steps, safeOutputStepNames, nil
	}
	consolidatedSafeOutputsJobLog.Print("Using handler manager for safe outputs")
	handlerManagerSteps, err := c.buildHandlerManagerStep(data)
	if err != nil {
		return nil, nil, err
	}
	steps = append(steps, handlerManagerSteps...)
	safeOutputStepNames = append(safeOutputStepNames, "process_safe_outputs")
	addHandlerManagerSafeOutputJobOutputs(data, outputs)
	return steps, safeOutputStepNames, nil
}

func addHandlerManagerSafeOutputJobOutputs(data *WorkflowData, outputs map[string]string) {
	outputs["process_safe_outputs_temporary_id_map"] = "${{ steps.process_safe_outputs.outputs.temporary_id_map }}"
	outputs["process_safe_outputs_processed_count"] = "${{ steps.process_safe_outputs.outputs.processed_count }}"
	outputs["create_discussion_errors"] = "${{ steps.process_safe_outputs.outputs.create_discussion_errors }}"
	outputs["create_discussion_error_count"] = "${{ steps.process_safe_outputs.outputs.create_discussion_error_count }}"
	outputs["code_push_failure_errors"] = "${{ steps.process_safe_outputs.outputs.code_push_failure_errors }}"
	outputs["code_push_failure_count"] = "${{ steps.process_safe_outputs.outputs.code_push_failure_count }}"
	addAssignmentSafeOutputJobOutputs(data, outputs)
	addUploadArtifactSafeOutputJobOutputs(data, outputs)
}

func addAssignmentSafeOutputJobOutputs(data *WorkflowData, outputs map[string]string) {
	if data.SafeOutputs.AssignToAgent != nil {
		consolidatedSafeOutputsJobLog.Print("Exposing assign_to_agent outputs from handler manager")
		outputs["assign_to_agent_assigned"] = "${{ steps.process_safe_outputs.outputs.assign_to_agent_assigned }}"
		outputs["assign_to_agent_assignment_errors"] = "${{ steps.process_safe_outputs.outputs.assign_to_agent_assignment_errors }}"
		outputs["assign_to_agent_assignment_error_count"] = "${{ steps.process_safe_outputs.outputs.assign_to_agent_assignment_error_count }}"
	}
	if data.SafeOutputs.CreateAgentSessions != nil {
		consolidatedSafeOutputsJobLog.Print("Exposing create_agent_session outputs from handler manager")
		outputs["create_agent_session_session_number"] = "${{ steps.process_safe_outputs.outputs.session_number }}"
		outputs["create_agent_session_session_url"] = "${{ steps.process_safe_outputs.outputs.session_url }}"
	}
}

func addUploadArtifactSafeOutputJobOutputs(data *WorkflowData, outputs map[string]string) {
	if data.SafeOutputs.UploadArtifact == nil {
		return
	}
	consolidatedSafeOutputsJobLog.Print("Exposing upload_artifact outputs from handler manager")
	cfg := data.SafeOutputs.UploadArtifact
	outputs["upload_artifact_count"] = "${{ steps.process_safe_outputs.outputs.upload_artifact_count }}"
	for i := range cfg.MaxUploads {
		outputs[fmt.Sprintf("upload_artifact_slot_%d_tmp_id", i)] = fmt.Sprintf("${{ steps.process_safe_outputs.outputs.slot_%d_tmp_id }}", i)
	}
}

func (c *Compiler) appendSarifArtifactSafeOutputSteps(steps []string, data *WorkflowData, outputs map[string]string, agentArtifactPrefix string) []string {
	if data.SafeOutputs.CreateCodeScanningAlerts == nil ||
		isHandlerStaged(c.trialMode || templatableBoolIsTrue(data.SafeOutputs.Staged), data.SafeOutputs.CreateCodeScanningAlerts.Staged) {
		return steps
	}
	consolidatedSafeOutputsJobLog.Print("Exposing sarif_file output for upload_code_scanning_sarif job")
	outputs["sarif_file"] = "${{ steps.process_safe_outputs.outputs.sarif_file }}"
	return append(steps, buildSarifArtifactUploadStep(agentArtifactPrefix, c.getActionPin)...)
}

func (c *Compiler) appendCustomActionSafeOutputSteps(steps []string, data *WorkflowData, markdownPath string, safeOutputStepNames []string) ([]string, []string) {
	if len(data.SafeOutputs.Actions) == 0 {
		return steps, safeOutputStepNames
	}
	c.resolveAllActions(data, markdownPath)
	steps = append(steps, c.buildActionSteps(data)...)
	for actionName := range data.SafeOutputs.Actions {
		normalizedName := stringutil.NormalizeSafeOutputIdentifier(actionName)
		safeOutputStepNames = append(safeOutputStepNames, "action_"+normalizedName)
	}
	return steps, safeOutputStepNames
}

func addNamedSafeOutputJobOutputs(data *WorkflowData, outputs map[string]string) {
	if data.SafeOutputs.AddReviewer != nil {
		outputs["add_reviewer_reviewers_added"] = "${{ steps.process_safe_outputs.outputs.reviewers_added }}"
	}
	if data.SafeOutputs.AssignMilestone != nil {
		outputs["assign_milestone_milestone_assigned"] = "${{ steps.process_safe_outputs.outputs.milestone_assigned }}"
	}
	if data.SafeOutputs.AssignToUser != nil {
		outputs["assign_to_user_assigned"] = "${{ steps.process_safe_outputs.outputs.assigned }}"
	}
	addCreatedItemSafeOutputJobOutputs(data, outputs)
}

func addCreatedItemSafeOutputJobOutputs(data *WorkflowData, outputs map[string]string) {
	if data.SafeOutputs.CreateIssues != nil {
		outputs["created_issue_number"] = "${{ steps.process_safe_outputs.outputs.created_issue_number }}"
		outputs["created_issue_url"] = "${{ steps.process_safe_outputs.outputs.created_issue_url }}"
	}
	if data.SafeOutputs.CreatePullRequests != nil {
		outputs["created_pr_number"] = "${{ steps.process_safe_outputs.outputs.created_pr_number }}"
		outputs["created_pr_url"] = "${{ steps.process_safe_outputs.outputs.created_pr_url }}"
	}
	if data.SafeOutputs.AddComments != nil {
		outputs["comment_id"] = "${{ steps.process_safe_outputs.outputs.comment_id }}"
		outputs["comment_url"] = "${{ steps.process_safe_outputs.outputs.comment_url }}"
	}
	if data.SafeOutputs.PushToPullRequestBranch != nil {
		outputs["push_commit_sha"] = "${{ steps.process_safe_outputs.outputs.push_commit_sha }}"
		outputs["push_commit_url"] = "${{ steps.process_safe_outputs.outputs.push_commit_url }}"
	}
	if data.SafeOutputs.CallWorkflow != nil {
		outputs["call_workflow_name"] = "${{ steps.process_safe_outputs.outputs.call_workflow_name }}"
		outputs["call_workflow_payload"] = "${{ steps.process_safe_outputs.outputs.call_workflow_payload }}"
	}
}

// buildSafeOutputsJobFromParts finalizes the step list (app-token insertion, token invalidation,
// items-manifest upload, dev-mode restore, script-mode cleanup), builds the job condition and
// dependency list, and assembles the Job struct for the safe_outputs job.
type buildSafeOutputsJobFromPartsOptions struct {
	data                   *WorkflowData
	mainJobName            string
	markdownPath           string
	agentArtifactPrefix    string
	steps                  []string
	outputs                map[string]string
	safeOutputStepNames    []string
	permissions            *Permissions
	threatDetectionEnabled bool
}

func (c *Compiler) buildSafeOutputsJobFromParts(
	opts buildSafeOutputsJobFromPartsOptions,
) (*Job, []string, error) {
	data := opts.data
	steps := opts.steps
	outputs := opts.outputs

	preambleTokenSteps := c.buildSafeOutputsPreambleTokenSteps(data, opts.permissions, outputs)
	steps = c.insertSafeOutputsPreambleTokenSteps(data, opts.agentArtifactPrefix, steps, preambleTokenSteps)
	steps = c.appendSafeOutputsFinalSteps(data, opts.agentArtifactPrefix, steps)

	jobCondition := buildSafeOutputsJobCondition(data, opts.threatDetectionEnabled)
	needs := c.buildSafeOutputsNeeds(data, opts.mainJobName, opts.threatDetectionEnabled)
	workflowID := GetWorkflowIDFromPath(opts.markdownPath)
	job := c.buildSafeOutputsJob(data, opts.permissions, outputs, steps, needs, jobCondition, workflowID)

	consolidatedSafeOutputsJobLog.Printf("Built consolidated safe outputs job with %d steps", len(opts.safeOutputStepNames))

	return job, opts.safeOutputStepNames, nil
}

func (c *Compiler) buildSafeOutputsPreambleTokenSteps(data *WorkflowData, permissions *Permissions, outputs map[string]string) []string {
	var preambleTokenSteps []string
	if data.SafeOutputs.GitHubApp != nil {
		outputs["app_token_minting_failed"] = "${{ steps.safe-outputs-app-token.outcome == 'failure' }}"
		var appTokenFallbackRepo string
		if hasWorkflowCallTrigger(data.On) {
			appTokenFallbackRepo = "${{ needs.activation.outputs.target_repo_name }}"
		}
		preambleTokenSteps = append(preambleTokenSteps, c.buildGitHubAppTokenMintStepForRepository(
			data.SafeOutputs.GitHubApp,
			permissions,
			appTokenFallbackRepo,
			inferSingleCheckoutRepositoryForGitHubAppOwner(data),
		)...)
	}
	if headApp := getSafeOutputsHeadApp(data.SafeOutputs); headApp != nil {
		headRepoSlug := getSafeOutputsHeadRepoSlug(data.SafeOutputs)
		preambleTokenSteps = append(preambleTokenSteps, c.buildGitHubAppTokenMintStepWithMeta(
			headApp,
			nil,
			headRepoNameFromSlug(headRepoSlug),
			headRepoSlug,
			"Generate GitHub App head token",
			"safe-outputs-head-app-token",
		)...)
	}
	return preambleTokenSteps
}

func (c *Compiler) insertSafeOutputsPreambleTokenSteps(data *WorkflowData, agentArtifactPrefix string, steps, preambleTokenSteps []string) []string {
	if len(preambleTokenSteps) == 0 {
		return steps
	}
	insertIndex := c.calculateSafeOutputsPreambleInsertionIndex(data, agentArtifactPrefix)
	for insertIndex < len(steps) && !strings.HasPrefix(steps[insertIndex], stepNameLinePrefix) {
		insertIndex++
	}
	if insertIndex == len(steps) {
		consolidatedSafeOutputsJobLog.Printf(
			"WARN: preamble-token insertion reached end of steps slice (len=%d); step ordering may be incorrect",
			len(steps),
		)
	}
	newSteps := append([]string{}, steps[:insertIndex]...)
	newSteps = append(newSteps, preambleTokenSteps...)
	return append(newSteps, steps[insertIndex:]...)
}

func (c *Compiler) calculateSafeOutputsPreambleInsertionIndex(data *WorkflowData, agentArtifactPrefix string) int {
	insertIndex := 0
	setupActionRef := c.resolveActionReference("./actions/setup", data)
	if setupActionRef != "" {
		insertIndex += len(c.generateCheckoutActionsFolder(data))
		countTraceID := fmt.Sprintf("${{ needs.%s.outputs.setup-trace-id }}", constants.ActivationJobName)
		countParentSpanID := setupParentSpanNeedsExpr(constants.ActivationJobName)
		enableArtifactClient := data.SafeOutputs != nil && data.SafeOutputs.UploadArtifact != nil
		insertIndex += len(c.generateSetupStep(data, setupActionRef, SetupActionDestination, enableArtifactClient, countTraceID, countParentSpanID))
	}
	if isOTLPHeadersPresent(data) {
		insertIndex += strings.Count(generateOTLPHeadersMaskStep(), stepNameLinePrefix)
	}
	if isOTLPAttributesPresent(data) {
		insertIndex += strings.Count(generateOTLPAttributesMaskStep(), stepNameLinePrefix)
	}
	insertIndex += len(buildAgentOutputDownloadSteps(agentArtifactPrefix, c.getActionPin))
	if data.SafeOutputs.UploadArtifact != nil {
		insertIndex += 6
	}
	if usesPatchesAndCheckouts(data.SafeOutputs) {
		patchDownloadSteps := buildArtifactDownloadSteps(ArtifactDownloadConfig{
			ArtifactName: agentArtifactPrefix + constants.AgentArtifactName,
			DownloadPath: constants.TmpGhAwDirSlash,
			SetupEnvStep: false,
			StepName:     "Download patch artifact",
		}, c.getActionPin)
		insertIndex += len(patchDownloadSteps)
	}
	return insertIndex
}

func (c *Compiler) appendSafeOutputsFinalSteps(data *WorkflowData, agentArtifactPrefix string, steps []string) []string {
	isStaged := c.trialMode || templatableBoolIsTrue(data.SafeOutputs.Staged)
	if !isStaged {
		steps = append(steps, buildSafeOutputItemsManifestUploadStep(agentArtifactPrefix, c.getActionPin)...)
	}
	if c.actionMode.IsDev() && usesPatchesAndCheckouts(data.SafeOutputs) {
		steps = append(steps, c.generateRestoreActionsSetupStep())
		consolidatedSafeOutputsJobLog.Print("Added restore actions folder step to safe_outputs job (dev mode with checkout)")
	}
	if c.actionMode.IsScript() {
		steps = append(steps, c.generateScriptModeCleanupStep())
	}
	return steps
}

func buildSafeOutputsJobCondition(data *WorkflowData, threatDetectionEnabled bool) ConditionNode {
	agentNotSkipped := BuildAnd(
		&NotNode{Child: BuildFunctionCall("cancelled")},
		BuildNotEquals(
			BuildPropertyAccess(fmt.Sprintf("needs.%s.result", constants.AgentJobName)),
			BuildStringLiteral("skipped"),
		),
	)
	if IsConditionalDetection(data.SafeOutputs) {
		return BuildAnd(
			BuildAnd(BuildFunctionCall("always"), agentNotSkipped),
			buildDetectionPassedCondition(),
		)
	}
	if threatDetectionEnabled {
		return BuildAnd(agentNotSkipped, buildDetectionSuccessCondition())
	}
	return agentNotSkipped
}

func (c *Compiler) buildSafeOutputsNeeds(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) []string {
	needs := []string{mainJobName}
	if threatDetectionEnabled {
		needs = append(needs, string(constants.DetectionJobName))
		consolidatedSafeOutputsJobLog.Print("Added detection job dependency to safe_outputs job")
	}
	needs = append(needs, string(constants.ActivationJobName))
	if data.LockForAgent {
		needs = append(needs, "unlock")
		consolidatedSafeOutputsJobLog.Print("Added unlock job dependency to safe_outputs job")
	}
	seenNeeds := make(map[string]struct{}, len(needs))
	for _, need := range needs {
		seenNeeds[need] = struct{}{}
	}
	needs = addExplicitSafeOutputsNeeds(data, needs, seenNeeds)
	return c.addPreActivationSafeOutputsNeed(data, needs, seenNeeds)
}

func addExplicitSafeOutputsNeeds(data *WorkflowData, needs []string, seenNeeds map[string]struct{}) []string {
	if data.SafeOutputs == nil {
		return needs
	}
	for _, need := range data.SafeOutputs.Needs {
		if setutil.Contains(seenNeeds, need) {
			continue
		}
		needs = append(needs, need)
		seenNeeds[need] = struct{}{}
		consolidatedSafeOutputsJobLog.Printf("Added explicit safe-outputs needs dependency to safe_outputs job: %s", need)
	}
	return needs
}

func (c *Compiler) addPreActivationSafeOutputsNeed(data *WorkflowData, needs []string, seenNeeds map[string]struct{}) []string {
	if data.SafeOutputs == nil || !messagesContainPreActivationRef(data.SafeOutputs.Messages) {
		return needs
	}
	if _, exists := c.jobManager.GetJob(string(constants.PreActivationJobName)); !exists {
		return needs
	}
	preActName := string(constants.PreActivationJobName)
	if !setutil.Contains(seenNeeds, preActName) {
		needs = append(needs, preActName)
		seenNeeds[preActName] = struct{}{}
		consolidatedSafeOutputsJobLog.Print("Added pre_activation dependency to safe_outputs job (messages reference pre_activation outputs)")
	}
	return needs
}

func (c *Compiler) buildSafeOutputsJob(data *WorkflowData, permissions *Permissions, outputs map[string]string, steps, needs []string, jobCondition ConditionNode, workflowID string) *Job {
	return &Job{
		Name:           "safe_outputs",
		If:             RenderCondition(jobCondition),
		RunsOn:         c.formatFrameworkJobRunsOn(data),
		Environment:    c.indentYAMLLines(resolveSafeOutputsEnvironment(data), "    "),
		Permissions:    permissions.RenderToYAML(),
		TimeoutMinutes: safeOutputsJobTimeoutMinutes(data),
		Concurrency:    c.buildSafeOutputsConcurrency(data),
		Env:            c.buildJobLevelSafeOutputEnvVars(data, workflowID),
		Steps:          steps,
		Outputs:        outputs,
		Needs:          needs,
	}
}

func (c *Compiler) buildSafeOutputsConcurrency(data *WorkflowData) string {
	if data.SafeOutputs.ConcurrencyGroup == "" {
		return ""
	}
	consolidatedSafeOutputsJobLog.Printf("Configuring safe_outputs job concurrency group: %s", data.SafeOutputs.ConcurrencyGroup)
	return c.indentYAMLLines(fmt.Sprintf("concurrency:\n  group: %q\n  cancel-in-progress: false", data.SafeOutputs.ConcurrencyGroup), "    ")
}

func safeOutputsJobTimeoutMinutes(data *WorkflowData) int {
	const defaultSafeOutputsTimeoutMinutes = 45
	if data.SafeOutputs.TimeoutMinutes > 0 {
		return data.SafeOutputs.TimeoutMinutes
	}
	return defaultSafeOutputsTimeoutMinutes
}

// buildJobLevelSafeOutputEnvVars builds environment variables that should be set at the job level
// for the consolidated safe_outputs job. These are variables that are common to all safe output steps.
func (c *Compiler) buildJobLevelSafeOutputEnvVars(data *WorkflowData, workflowID string) map[string]string {
	envVars := make(map[string]string)

	c.addWorkflowSafeOutputEnvVars(envVars, data, workflowID)
	c.addEngineSafeOutputEnvVars(envVars, data)
	addAgentOutputSafeOutputEnvVars(envVars)
	addCommandSafeOutputEnvVars(envVars, data)
	c.addSafeOutputModeEnvVars(envVars, data)
	c.addMessagesSafeOutputEnvVars(envVars, data)
	addDetectionSafeOutputEnvVars(envVars, data)

	return envVars
}

func (c *Compiler) addWorkflowSafeOutputEnvVars(envVars map[string]string, data *WorkflowData, workflowID string) {
	envVars["GH_AW_WORKFLOW_ID"] = fmt.Sprintf("%q", workflowID)
	envVars["GH_AW_CALLER_WORKFLOW_ID"] = fmt.Sprintf(`"${{ github.repository }}/%s"`, workflowID)
	envVars["GH_AW_WORKFLOW_NAME"] = fmt.Sprintf("%q", data.Name)
	if data.FrontmatterEmoji != "" {
		envVars["GH_AW_WORKFLOW_EMOJI"] = fmt.Sprintf("%q", data.FrontmatterEmoji)
	}
	if data.Source != "" {
		envVars["GH_AW_WORKFLOW_SOURCE"] = fmt.Sprintf("%q", data.Source)
		sourceURL := buildSourceURL(data.Source)
		if sourceURL != "" {
			envVars["GH_AW_WORKFLOW_SOURCE_URL"] = fmt.Sprintf("%q", sourceURL)
		}
	} else if localURL := buildLocalWorkflowSourceURL(c.markdownPath); localURL != "" {
		envVars["GH_AW_WORKFLOW_SOURCE_URL"] = fmt.Sprintf("%q", localURL)
	}
	if data.TrackerID != "" {
		envVars["GH_AW_TRACKER_ID"] = fmt.Sprintf("%q", data.TrackerID)
	}
	if utcOffset := c.getCompiledProjectUTCOffset(); utcOffset != "" {
		envVars["GH_AW_PROJECT_UTC"] = fmt.Sprintf("%q", utcOffset)
	}
}

func (c *Compiler) addEngineSafeOutputEnvVars(envVars map[string]string, data *WorkflowData) {
	if data.EngineConfig == nil {
		return
	}
	if data.EngineConfig.ID != "" {
		envVars["GH_AW_ENGINE_ID"] = fmt.Sprintf("%q", data.EngineConfig.ID)
	}
	if data.EngineConfig.Version != "" {
		envVars["GH_AW_ENGINE_VERSION"] = fmt.Sprintf("%q", data.EngineConfig.Version)
	}
	if data.EngineConfig.Model != "" {
		envVars["GH_AW_ENGINE_MODEL"] = fmt.Sprintf("%q", data.EngineConfig.Model)
	} else {
		envVars["GH_AW_ENGINE_MODEL"] = fmt.Sprintf("${{ needs.%s.outputs.model }}", constants.AgentJobName)
	}
}

func addAgentOutputSafeOutputEnvVars(envVars map[string]string) {
	envVars["GH_AW_EFFECTIVE_TOKENS"] = fmt.Sprintf("${{ needs.%s.outputs.effective_tokens }}", constants.AgentJobName)
	envVars["GH_AW_AIC"] = fmt.Sprintf("${{ needs.%s.outputs.aic }}", constants.AgentJobName)
	envVars["GH_AW_AMBIENT_CONTEXT"] = fmt.Sprintf("${{ needs.%s.outputs.ambient_context }}", constants.AgentJobName)
	envVars["GH_AW_AGENT_AIC"] = fmt.Sprintf("${{ needs.%s.outputs.aic }}", constants.AgentJobName)
}

func addCommandSafeOutputEnvVars(envVars map[string]string, data *WorkflowData) {
	if len(data.Command) > 0 {
		if commandsJSON, err := json.Marshal(data.Command); err == nil {
			envVars["GH_AW_COMMANDS"] = fmt.Sprintf("%q", string(commandsJSON))
		}
		if data.CommandPlaceholder != "" {
			envVars["GH_AW_COMMAND_PLACEHOLDER"] = fmt.Sprintf("%q", data.CommandPlaceholder)
		}
	}
	if len(data.LabelCommand) > 0 {
		if labelCommandsJSON, err := json.Marshal(data.LabelCommand); err == nil {
			envVars["GH_AW_LABEL_COMMANDS"] = fmt.Sprintf("%q", string(labelCommandsJSON))
		}
	}
}

func (c *Compiler) addSafeOutputModeEnvVars(envVars map[string]string, data *WorkflowData) {
	if data.SafeOutputs != nil {
		if value := resolveSafeOutputsStagedValue(c.trialMode, data.SafeOutputs.Staged); value != nil {
			if isExpression(*value) {
				envVars["GH_AW_SAFE_OUTPUTS_STAGED"] = *value
			} else {
				envVars["GH_AW_SAFE_OUTPUTS_STAGED"] = "\"true\""
			}
		}
	}
	if c.trialMode && c.trialLogicalRepoSlug != "" {
		envVars["GH_AW_TARGET_REPO_SLUG"] = fmt.Sprintf("%q", c.trialLogicalRepoSlug)
	}
}

func (c *Compiler) addMessagesSafeOutputEnvVars(envVars map[string]string, data *WorkflowData) {
	if data.SafeOutputs == nil || data.SafeOutputs.Messages == nil {
		return
	}
	messagesJSON, err := serializeMessagesConfig(data.SafeOutputs.Messages)
	if err != nil {
		consolidatedSafeOutputsJobLog.Printf("Warning: failed to serialize messages config: %v", err)
	} else if messagesJSON != "" {
		envVars["GH_AW_SAFE_OUTPUT_MESSAGES"] = fmt.Sprintf("%q", messagesJSON)
	}
}

func addDetectionSafeOutputEnvVars(envVars map[string]string, data *WorkflowData) {
	if !IsDetectionJobEnabled(data.SafeOutputs) {
		return
	}
	envVars["GH_AW_DETECTION_CONCLUSION"] = fmt.Sprintf("${{ needs.%s.outputs.detection_conclusion }}", constants.DetectionJobName)
	envVars["GH_AW_DETECTION_REASON"] = fmt.Sprintf("${{ needs.%s.outputs.detection_reason }}", constants.DetectionJobName)
	envVars["GH_AW_THREAT_DETECTION_AIC"] = fmt.Sprintf("${{ needs.%s.outputs.aic }}", constants.DetectionJobName)
}

// resolveSafeOutputsEnvironment resolves the effective GitHub deployment environment for
// safe-output jobs. If safe-outputs.environment is explicitly set, it takes precedence.
// Otherwise the top-level environment: field is propagated so that environment-scoped
// secrets are accessible in all safe-output jobs.
func resolveSafeOutputsEnvironment(data *WorkflowData) string {
	if data.SafeOutputs != nil && data.SafeOutputs.Environment != "" {
		return data.SafeOutputs.Environment
	}
	return data.Environment
}

// buildSafeOutputItemsManifestUploadStep builds the step that uploads the safe output
// items manifest and temporary ID map as a separate artifact. The step always runs
// (if: always()) so the files are available to the audit command even if some safe
// output steps fail.
// The files are uploaded as a dedicated "safe-outputs-items" artifact (not merged into the
// "agent" artifact) to avoid a 409 Conflict when both the agent job and safe_outputs job
// try to upload an artifact with the same name in the same workflow run.
// prefix is prepended to the artifact name; use empty string for non-workflow_call workflows.
// pinAction resolves the upload-artifact action reference; pass c.getActionPin from Compiler methods.
func buildSafeOutputItemsManifestUploadStep(prefix string, pinAction func(string) string) []string {
	return []string{
		"      - name: Upload Safe Outputs Items\n",
		"        if: always()\n",
		fmt.Sprintf("        uses: %s\n", pinAction("actions/upload-artifact")),
		"        with:\n",
		fmt.Sprintf("          name: %s%s\n", prefix, constants.SafeOutputItemsArtifactName),
		"          path: |\n",
		"            /tmp/gh-aw/safe-output-items.jsonl\n",
		fmt.Sprintf("            /tmp/gh-aw/%s\n", constants.TemporaryIdMapFilename),
		"          if-no-files-found: ignore\n",
	}
}

// buildSarifArtifactUploadStep builds the step that uploads the SARIF file generated by
// the create_code_scanning_alert handler as a GitHub Actions artifact.
//
// The SARIF file only exists in the safe_outputs job workspace.  The dedicated
// upload_code_scanning_sarif job runs in a completely separate, fresh workspace so it
// cannot access the file via a job-output path string alone — it must download the
// artifact first.
//
// The step is conditional on the sarif_file output being non-empty (i.e. the handler
// actually produced findings), so it is skipped on clean runs.
// prefix is prepended to the artifact name for workflow_call contexts.
// pinAction resolves the upload-artifact action reference; pass c.getActionPin from Compiler methods.
func buildSarifArtifactUploadStep(prefix string, pinAction func(string) string) []string {
	return []string{
		"      - name: Upload SARIF artifact\n",
		"        if: steps.process_safe_outputs.outputs.sarif_file != ''\n",
		fmt.Sprintf("        uses: %s\n", pinAction("actions/upload-artifact")),
		"        with:\n",
		fmt.Sprintf("          name: %s%s\n", prefix, constants.SarifArtifactName),
		"          path: ${{ steps.process_safe_outputs.outputs.sarif_file }}\n",
		"          if-no-files-found: error\n",
		"          retention-days: 1\n",
	}
}

// scriptNameToHandlerName converts a script name like "post-slack-message" to a
// JavaScript function name like "handlePostSlackMessage".
func scriptNameToHandlerName(scriptName string) string {
	parts := strings.FieldsFunc(scriptName, func(r rune) bool {
		return r == '-' || r == '_'
	})
	var sb strings.Builder
	sb.WriteString("handle")
	for _, part := range parts {
		if part != "" {
			sb.WriteString(strings.ToUpper(part[:1]) + part[1:])
		}
	}
	if sb.Len() == len("handle") {
		// Fallback: use the script name as-is when parts are empty
		if scriptName == "" {
			sb.WriteString("Unknown")
		} else {
			sb.WriteString(strings.ToUpper(scriptName[:1]) + scriptName[1:])
		}
	}
	return sb.String()
}

// generateSafeOutputScriptContent generates a complete JavaScript module for a custom safe-output
// script handler. Users write only the handler body (the code that runs inside the async handler
// function for each item), and the compiler generates the full outer wrapper including:
//   - Config input destructuring: const { channel, message } = config;
//   - Handler function: return async function handleX(item, resolvedTemporaryIds) { ... }
//   - The module.exports boilerplate
func generateSafeOutputScriptContent(scriptName string, scriptConfig *SafeScriptConfig) string {
	var sb strings.Builder
	sb.WriteString("// @ts-check\n")
	sb.WriteString("/// <reference types=\"./safe-output-script\" />\n")
	sb.WriteString("// Auto-generated safe-output script handler: " + scriptName + "\n\n")
	sb.WriteString("const { sanitizeContent } = require(\"./sanitize_content.cjs\");\n\n")
	sb.WriteString("/** @type {import('./types/safe-output-script').SafeOutputScriptMain} */\n")
	sb.WriteString("async function main(config = {}) {\n")

	// Auto-destructure all declared input names from config (provides access to
	// static YAML config values such as defaults).
	if len(scriptConfig.Inputs) > 0 {
		inputNames := make([]string, 0, len(scriptConfig.Inputs))
		for name := range scriptConfig.Inputs {
			safeName := stringutil.SanitizeParameterName(name)
			if safeName != name {
				inputNames = append(inputNames, name+": "+safeName)
			} else {
				inputNames = append(inputNames, name)
			}
		}
		sort.Strings(inputNames)
		sb.WriteString("  const { " + strings.Join(inputNames, ", ") + " } = config;\n")
	}

	// Generate the handler function that receives each item at runtime.
	handlerName := scriptNameToHandlerName(scriptName)
	sb.WriteString("  return async function " + handlerName + "(item, resolvedTemporaryIds, temporaryIdMap) {\n")
	// Indent each line of the user's handler body by 4 spaces
	for line := range strings.SplitSeq(scriptConfig.Script, "\n") {
		sb.WriteString("    " + line + "\n")
	}
	sb.WriteString("  };\n")
	sb.WriteString("}\n")
	sb.WriteString("module.exports = { main };\n")
	return sb.String()
}

// buildCustomScriptFilesStep generates a run step that writes inline safe-output script files
// to the setup action destination folder so they can be required by the handler manager.
// Users write only the handler body; the compiler wraps it with config destructuring,
// the handler function, and module.exports boilerplate.
// Each script is written using a heredoc to avoid shell quoting issues.
func buildCustomScriptFilesStep(scripts map[string]*SafeScriptConfig) ([]string, error) {
	if len(scripts) == 0 {
		return nil, nil
	}

	// Sort script names for deterministic output
	scriptNames := sliceutil.SortedKeys(scripts)

	var steps []string
	steps = append(steps, "      - name: Configure Safe Outputs Custom Scripts\n")
	steps = append(steps, "        run: |\n")

	for _, scriptName := range scriptNames {
		scriptConfig := scripts[scriptName]
		normalizedName := stringutil.NormalizeSafeOutputIdentifier(scriptName)
		filename := safeOutputScriptFilename(normalizedName)
		filePath := SetupActionDestinationShell + "/" + filename
		scriptContent := generateSafeOutputScriptContent(scriptName, scriptConfig)
		delimiter := GenerateHeredocDelimiterFromContent("SAFE_OUTPUT_SCRIPT_"+strings.ToUpper(normalizedName), scriptContent)

		if err := ValidateHeredocContent(scriptContent, delimiter); err != nil {
			return nil, fmt.Errorf("safe-output script %q: %w", scriptName, err)
		}

		steps = append(steps, fmt.Sprintf("          cat > \"%s\" << '%s'\n", filePath, delimiter))
		for line := range strings.SplitSeq(scriptContent, "\n") {
			steps = append(steps, "          "+line+"\n")
		}
		steps = append(steps, "          "+delimiter+"\n")
	}

	return steps, nil
}
