package workflow

import (
	"context"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var maintenanceWorkflowYAMLLog = logger.New("workflow:maintenance_workflow_yaml")

const maintenanceWorkflowCustomInstructions = `This file defines the generated agentic maintenance workflow for this repository.
It runs scheduled cleanup for expiring safe outputs and supports manual maintenance operations.

This workflow is generated automatically when workflows use expiring safe outputs
or when repository maintenance features are enabled in .github/workflows/aw.json.

To disable maintenance workflow generation, set in .github/workflows/aw.json:
  {"maintenance": false}

Agentic maintenance docs:
  https://github.github.com/gh-aw/reference/ephemerals/#manual-maintenance-operations`

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

// buildMaintenanceWorkflowYAML generates the complete YAML content for the
// agentics-maintenance.yml workflow. It is called by GenerateMaintenanceWorkflow
// after the cron schedule and setup parameters have been resolved.
func buildMaintenanceWorkflowYAML(ctx context.Context, opts buildMaintenanceWorkflowYAMLOptions) string {
	maintenanceWorkflowYAMLLog.Printf("Building maintenance workflow YAML: actionMode=%s minExpiresDays=%d cronSchedule=%q defaultBranch=%q disableLabelTrigger=%v createCompilePR=%v copilotOrgBilling=%v", opts.actionMode, opts.minExpiresDays, opts.cronSchedule, opts.defaultBranch, opts.disableLabelTrigger, opts.createCompilePR, opts.copilotOrgBilling)
	labelDisableJobEnabled := !opts.disableLabelTrigger && !opts.maintenanceConfig.IsJobDisabled("label_disable_agentic_workflow")
	labelApplySafeOutputsJobEnabled := !opts.disableLabelTrigger && !opts.maintenanceConfig.IsJobDisabled("label_apply_safe_outputs")
	setupActionRef := ResolveSetupActionReference(ctx, opts.actionMode, opts.version, opts.actionTag, opts.resolver)

	var yaml strings.Builder
	yaml.WriteString(GenerateWorkflowHeader("", "pkg/workflow/maintenance_workflow.go", maintenanceWorkflowCustomInstructions))

	builder := newMaintenanceWorkflowYAMLBuilder(ctx, opts, &yaml, setupActionRef, labelDisableJobEnabled, labelApplySafeOutputsJobEnabled)
	builder.writeMaintenanceWorkflowTriggersSection()
	builder.writeCloseExpiredJobs()
	builder.writeCleanupCacheJob()
	builder.writeRunOperationJob()
	builder.writeUpdatePRBranchesJob()
	builder.writeApplySafeOutputsJob()
	builder.writeCreateLabelsJob()
	builder.writeActivityReportJob()
	builder.writeForecastReportJob()
	builder.writeCloseAWIssuesJob()
	builder.writeValidateWorkflowsJob()
	builder.writeLabelTriggeredJobs()
	builder.writeDevModeJobs()
	return yaml.String()
}
