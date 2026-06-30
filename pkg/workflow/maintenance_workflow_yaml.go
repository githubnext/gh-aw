package workflow

import (
	"context"
	"strings"

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
	compileGitHubToken  string
	createCompilePR     bool
	copilotOrgBilling   bool // all Copilot workflows use copilot-requests: write (GITHUB_TOKEN); COPILOT_GITHUB_TOKEN is not required
}

// buildMaintenanceWorkflowYAML generates the complete YAML content for the
// agentics-maintenance.yml workflow. It is called by GenerateMaintenanceWorkflow
// after the cron schedule and setup parameters have been resolved.
func buildMaintenanceWorkflowYAML(
	ctx context.Context,
	opts buildMaintenanceWorkflowYAMLOptions,
) string {
	maintenanceWorkflowYAMLLog.Printf("Building maintenance workflow YAML: actionMode=%s minExpiresDays=%d cronSchedule=%q defaultBranch=%q disableLabelTrigger=%v createCompilePR=%v copilotOrgBilling=%v", opts.actionMode, opts.minExpiresDays, opts.cronSchedule, opts.defaultBranch, opts.disableLabelTrigger, opts.createCompilePR, opts.copilotOrgBilling)

	setupActionRef := ResolveSetupActionReference(ctx, opts.actionMode, opts.version, opts.actionTag, opts.resolver)

	var yaml strings.Builder
	yaml.WriteString(buildMaintenanceWorkflowPreamble(opts))
	yaml.WriteString(buildCloseExpiredEntitiesJob(opts, setupActionRef))
	yaml.WriteString(buildCleanupCacheMemoryJob(opts, setupActionRef))
	yaml.WriteString(buildRunOperationJob(ctx, opts, setupActionRef))
	yaml.WriteString(buildUpdatePRBranchesJob(opts, setupActionRef))
	yaml.WriteString(buildApplySafeOutputsJob(opts, setupActionRef))
	yaml.WriteString(buildCreateLabelsJob(ctx, opts, setupActionRef))
	yaml.WriteString(buildActivityReportJob(ctx, opts, setupActionRef))
	yaml.WriteString(buildForecastReportJob(ctx, opts, setupActionRef))
	yaml.WriteString(buildCloseAgenticWorkflowsIssuesJob(opts, setupActionRef))
	yaml.WriteString(buildValidateWorkflowsJob(ctx, opts, setupActionRef))

	if !opts.disableLabelTrigger {
		maintenanceWorkflowYAMLLog.Print("Adding label-triggered jobs: label_disable_agentic_workflow and label_apply_safe_outputs")
		yaml.WriteString(buildLabelDisableAgenticWorkflowJob(opts, setupActionRef))
		yaml.WriteString(buildLabelApplySafeOutputsJob(opts, setupActionRef))
	}

	if opts.actionMode == ActionModeDev {
		maintenanceWorkflowYAMLLog.Printf("Adding dev-only jobs: compile-workflows and secret-validation")
		yaml.WriteString(buildDevCompileWorkflowsJob(ctx, opts, setupActionRef))
		yaml.WriteString(buildDevSecretValidationJob(opts, setupActionRef))
	}

	return yaml.String()
}
