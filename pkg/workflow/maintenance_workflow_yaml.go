package workflow

import (
	"context"

	"github.com/github/gh-aw/pkg/logger"
)

var maintenanceWorkflowYAMLLog = logger.New("workflow:maintenance_workflow_yaml")

// buildMaintenanceWorkflowYAMLOptions configures the maintenance workflow YAML builder.
type buildMaintenanceWorkflowYAMLOptions struct {
	cronSchedule        string
	scheduleDesc        string
	minExpiresDays      int
	runsOnValue         string
	actionMode          ActionMode
	version             string
	actionTag           string
	resolver            SHAResolver
	configuredRunsOn    RunsOnValue
	defaultBranch       string
	disableLabelTrigger bool
	maintenanceConfig   *MaintenanceConfig
	compileGitHubToken  string
	createCompilePR     bool
	copilotOrgBilling   bool // all Copilot workflows use copilot-requests: write (GITHUB_TOKEN); COPILOT_GITHUB_TOKEN is not required
}

func buildMaintenanceWorkflowYAML(
	ctx context.Context,
	opts buildMaintenanceWorkflowYAMLOptions,
) string {
	maintenanceWorkflowYAMLLog.Printf("Building maintenance workflow YAML: actionMode=%s minExpiresDays=%d cronSchedule=%q defaultBranch=%q disableLabelTrigger=%v createCompilePR=%v copilotOrgBilling=%v",
		opts.actionMode, opts.minExpiresDays, opts.cronSchedule, opts.defaultBranch, opts.disableLabelTrigger, opts.createCompilePR, opts.copilotOrgBilling)

	labelDisableJobEnabled := !opts.disableLabelTrigger && !opts.maintenanceConfig.IsJobDisabled("label_disable_agentic_workflow")
	labelApplySafeOutputsJobEnabled := !opts.disableLabelTrigger && !opts.maintenanceConfig.IsJobDisabled("label_apply_safe_outputs")

	appliedRunURLValue := "${{ jobs.apply_safe_outputs.outputs.run_url }}"
	appliedRunURLDescription := "The run URL that safe outputs were applied from"
	if opts.maintenanceConfig.IsJobDisabled("apply_safe_outputs") {
		appliedRunURLValue = "${{ inputs.run_url }}"
		appliedRunURLDescription = "The run URL that safe outputs were applied from (workflow_call falls back to inputs.run_url when apply_safe_outputs is disabled; other triggers leave this empty)"
	}

	m := newMaintenanceYAMLBuilder(ctx, opts)
	m.writeWorkflowHeader()
	m.writeScheduleAndLabelTriggers(labelDisableJobEnabled, labelApplySafeOutputsJobEnabled)
	m.writeDispatchAndCallTriggers(appliedRunURLValue, appliedRunURLDescription)
	m.writeCloseExpiredEntitiesJobs()
	m.writeCleanupCacheMemoryJob()
	m.writeRunOperationJob(ctx)
	m.writeUpdatePRBranchesJob()
	m.writeApplySafeOutputsJob()
	m.writeCreateLabelsJob(ctx)
	m.writeActivityReportJobHeader(ctx)
	m.writeActivityReportLogSteps()
	m.writeForecastReportJobHeader(ctx)
	m.writeForecastReportRunSteps()
	m.writeCloseAgenticWorkflowsIssuesJob()
	m.writeValidateWorkflowsJob(ctx)

	if labelDisableJobEnabled || labelApplySafeOutputsJobEnabled {
		maintenanceWorkflowYAMLLog.Print("Adding label-triggered jobs")
		if labelDisableJobEnabled {
			m.writeLabelDisableJob()
		}
		if labelApplySafeOutputsJobEnabled {
			m.writeLabelApplySafeOutputsJob()
		}
	}

	if opts.actionMode == ActionModeDev {
		maintenanceWorkflowYAMLLog.Printf("Adding dev-only jobs: compile-workflows and secret-validation")
		m.writeCompileWorkflowsJob(ctx)
		m.writeSecretValidationJob()
	}

	return m.b.String()
}
