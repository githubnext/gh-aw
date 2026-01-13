package campaign

// CampaignSpec defines a first-class campaign configuration loaded from
// YAML frontmatter in Markdown files.
//
// Files are discovered from the local repository under:
//
//	.github/workflows/*.campaign.md
//
// This provides a thin, declarative layer on top of existing agentic
// workflows and repo-memory conventions.
type CampaignSpec struct {
	ID          string `yaml:"id" json:"id" console:"header:ID"`
	Name        string `yaml:"name" json:"name" console:"header:Name,maxlen:30"`
	Description string `yaml:"description,omitempty" json:"description,omitempty" console:"header:Description,omitempty,maxlen:60"`

	// Objective is an optional outcome-owned statement describing what success means
	// for this campaign.
	Objective string `yaml:"objective,omitempty" json:"objective,omitempty" console:"header:Objective,omitempty,maxlen:60"`

	// KPIs is an optional list of KPIs used to measure progress toward the objective.
	// Recommended: 1 primary KPI plus up to 2 supporting KPIs.
	KPIs []CampaignKPI `yaml:"kpis,omitempty" json:"kpis,omitempty"`

	// ProjectURL points to the GitHub Project used as the primary campaign
	// dashboard.
	ProjectURL string `yaml:"project-url,omitempty" json:"project_url,omitempty" console:"header:Project URL,omitempty,maxlen:40"`

	// Version is an optional spec version string (for example: v1).
	// When omitted, it defaults to v1 during validation.
	Version string `yaml:"version,omitempty" json:"version,omitempty" console:"header:Version,omitempty"`

	// Workflows associates this campaign with one or more workflow IDs
	// (basename of the Markdown file without .md).
	Workflows []string `yaml:"workflows,omitempty" json:"workflows,omitempty" console:"header:Workflows,omitempty,maxlen:40"`

	// TrackerLabel is an optional label used to discover worker-created issues/PRs
	// (for example: campaign:security-q1-2025). When set, the discovery precomputation
	// step will search for items with this label.
	TrackerLabel string `yaml:"tracker-label,omitempty" json:"tracker_label,omitempty" console:"header:Tracker Label,omitempty,maxlen:40"`

	// MemoryPaths documents where this campaign writes its repo-memory
	// (for example: memory/campaigns/incident-response/**).
	MemoryPaths []string `yaml:"memory-paths,omitempty" json:"memory_paths,omitempty" console:"header:Memory Paths,omitempty,maxlen:40"`

	// MetricsGlob is an optional glob (relative to the repository root)
	// used to locate JSON metrics snapshots stored in the
	// memory/campaigns branch. When set, `gh aw campaign status` will
	// attempt to read the latest matching metrics file and surface a few
	// key fields.
	MetricsGlob string `yaml:"metrics-glob,omitempty" json:"metrics_glob,omitempty" console:"header:Metrics Glob,omitempty,maxlen:30"`

	// CursorGlob is an optional glob (relative to the repository root)
	// used to locate a durable cursor/checkpoint file stored in the
	// memory/campaigns branch. When set, generated coordinator workflows
	// will be instructed to continue incremental discovery from this cursor
	// and `gh aw campaign status` will surface its freshness.
	CursorGlob string `yaml:"cursor-glob,omitempty" json:"cursor_glob,omitempty" console:"header:Cursor Glob,omitempty,maxlen:30"`

	// Owners lists the primary human owners for this campaign.
	Owners []string `yaml:"owners,omitempty" json:"owners,omitempty" console:"header:Owners,omitempty,maxlen:30"`

	// ExecutiveSponsors lists executive stakeholders or sponsors who are
	// accountable for the outcome of this campaign.
	ExecutiveSponsors []string `yaml:"executive-sponsors,omitempty" json:"executive_sponsors,omitempty" console:"header:Executive Sponsors,omitempty,maxlen:30"`

	// RiskLevel is an optional free-form field (e.g. low/medium/high).
	RiskLevel string `yaml:"risk-level,omitempty" json:"risk_level,omitempty" console:"header:Risk Level,omitempty"`

	// State describes the lifecycle stage of the campaign definition.
	// Valid values are: planned, active, paused, completed, archived.
	State string `yaml:"state,omitempty" json:"state,omitempty" console:"header:State,omitempty"`

	// Tags provide free-form categorization for reporting (for example:
	// security, modernization, rollout).
	Tags []string `yaml:"tags,omitempty" json:"tags,omitempty" console:"header:Tags,omitempty,maxlen:30"`

	// AllowedSafeOutputs documents which safe-outputs operations this
	// campaign is expected to use (for example: create-issue,
	// create-pull-request). This is currently informational but can be
	// enforced by validation in the future.
	AllowedSafeOutputs []string `yaml:"allowed-safe-outputs,omitempty" json:"allowed_safe_outputs,omitempty" console:"header:Allowed Safe Outputs,omitempty,maxlen:30"`

	// ProjectGitHubToken is an optional GitHub token expression (e.g.,
	// ${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}) used for GitHub Projects v2
	// operations. When specified, this token is passed to the update-project
	// safe output configuration in the generated orchestrator workflow.
	ProjectGitHubToken string `yaml:"project-github-token,omitempty" json:"project_github_token,omitempty" console:"header:Project Token,omitempty,maxlen:30"`

	// Governance configures lightweight pacing and opt-out policies for campaign
	// orchestrator workflows. These guardrails are primarily enforced through
	// generated prompts and safe-output maxima.
	Governance *CampaignGovernancePolicy `yaml:"governance,omitempty" json:"governance,omitempty"`

	// ApprovalPolicy describes high-level approval expectations for this
	// campaign (for example: number of approvals and required roles).
	ApprovalPolicy *CampaignApprovalPolicy `yaml:"approval-policy,omitempty" json:"approval-policy,omitempty"`

	// ExecuteWorkflows enables the campaign to actively run the workflows
	// listed in the Workflows field. When true, the orchestrator will
	// execute workflows sequentially and can create new workflows if needed.
	// Default: false (passive discovery only).
	ExecuteWorkflows bool `yaml:"execute-workflows,omitempty" json:"execute_workflows,omitempty"`

	// Engine specifies the AI engine to use for the campaign orchestrator.
	// Valid values: copilot, claude, codex, custom.
	// Default: copilot (when not specified).
	Engine string `yaml:"engine,omitempty" json:"engine,omitempty" console:"header:Engine,omitempty"`

	// ConfigPath is populated at load time with the relative path of
	// the YAML file on disk, to help users locate definitions.
	ConfigPath string `yaml:"-" json:"config_path" console:"header:Config Path,maxlen:60"`
}

// CampaignKPI defines a single KPI used for campaign measurement.
type CampaignKPI struct {
	// ID is an optional stable identifier for this KPI.
	ID string `yaml:"id,omitempty" json:"id,omitempty"`

	// Name is a human-readable KPI name.
	Name string `yaml:"name" json:"name"`

	// Priority indicates whether this KPI is the primary KPI or a supporting KPI.
	// Expected values: primary, supporting.
	Priority string `yaml:"priority,omitempty" json:"priority,omitempty"`

	// Unit is an optional unit string (e.g., percent, days, count).
	Unit string `yaml:"unit,omitempty" json:"unit,omitempty"`

	// Baseline is the baseline KPI value.
	Baseline float64 `yaml:"baseline" json:"baseline"`

	// Target is the target KPI value.
	Target float64 `yaml:"target" json:"target"`

	// TimeWindowDays is the rolling time window (in days) used to compute the KPI.
	TimeWindowDays int `yaml:"time-window-days" json:"time-window-days"`

	// Direction indicates whether improvement means increasing or decreasing.
	// Expected values: increase, decrease.
	Direction string `yaml:"direction,omitempty" json:"direction,omitempty"`

	// Source describes the signal source used to compute the KPI.
	// Expected values: ci, pull_requests, code_security, custom.
	Source string `yaml:"source,omitempty" json:"source,omitempty"`
}

// CampaignGovernancePolicy captures lightweight pacing and opt-out policies.
// This is intentionally scoped to what gh-aw can apply safely and consistently
// via prompts and safe-output job limits.
type CampaignGovernancePolicy struct {
	// MaxNewItemsPerRun caps how many new items (issues/PRs) the launcher should
	// add to the Project board per run. 0 means "use defaults".
	MaxNewItemsPerRun int `yaml:"max-new-items-per-run,omitempty" json:"max_new_items_per_run,omitempty"`

	// MaxDiscoveryItemsPerRun caps how many candidate issues/PRs the launcher
	// and orchestrator may scan during discovery in a single run.
	// 0 means "use defaults".
	MaxDiscoveryItemsPerRun int `yaml:"max-discovery-items-per-run,omitempty" json:"max_discovery_items_per_run,omitempty"`

	// MaxDiscoveryPagesPerRun caps how many pages of results the launcher and
	// orchestrator may fetch in a single run.
	// 0 means "use defaults".
	MaxDiscoveryPagesPerRun int `yaml:"max-discovery-pages-per-run,omitempty" json:"max_discovery_pages_per_run,omitempty"`

	// OptOutLabels is a list of labels that opt an issue/PR out of campaign
	// tracking. Items with any of these labels should be ignored by launcher/
	// orchestrator.
	OptOutLabels []string `yaml:"opt-out-labels,omitempty" json:"opt_out_labels,omitempty"`

	// DoNotDowngradeDoneItems prevents moving Project status backwards (e.g.
	// Done -> In Progress) if the underlying issue/PR is reopened.
	DoNotDowngradeDoneItems *bool `yaml:"do-not-downgrade-done-items,omitempty" json:"do_not_downgrade_done_items,omitempty"`

	// MaxProjectUpdatesPerRun controls the update-project safe-output maximum
	// for generated coordinator workflows. 0 means "use defaults".
	MaxProjectUpdatesPerRun int `yaml:"max-project-updates-per-run,omitempty" json:"max_project_updates_per_run,omitempty"`

	// MaxCommentsPerRun controls the add-comment safe-output maximum for
	// generated coordinator workflows. 0 means "use defaults".
	MaxCommentsPerRun int `yaml:"max-comments-per-run,omitempty" json:"max_comments_per_run,omitempty"`
}

// CampaignApprovalPolicy captures basic approval expectations for a
// campaign. It is intentionally lightweight and advisory; enforcement
// is left to workflows and organizational process.
type CampaignApprovalPolicy struct {
	RequiredApprovals int      `yaml:"required-approvals,omitempty" json:"required-approvals,omitempty"`
	RequiredRoles     []string `yaml:"required-roles,omitempty" json:"required-roles,omitempty"`
	ChangeControl     bool     `yaml:"change-control,omitempty" json:"change-control,omitempty"`
}

// CampaignRuntimeStatus represents the live status of a campaign, including
// compiled workflow state and optional metrics/cursor info.
type CampaignRuntimeStatus struct {
	ID        string   `json:"id" console:"header:ID"`
	Name      string   `json:"name" console:"header:Name"`
	Workflows []string `json:"workflows,omitempty" console:"header:Workflows,omitempty"`
	Compiled  string   `json:"compiled" console:"header:Compiled"`

	// Optional metrics from repo-memory (when MetricsGlob is set and a
	// matching JSON snapshot is found on the memory/campaigns branch).
	MetricsTasksTotal          int     `json:"metrics_tasks_total,omitempty" console:"header:Tasks Total,omitempty"`
	MetricsTasksCompleted      int     `json:"metrics_tasks_completed,omitempty" console:"header:Tasks Completed,omitempty"`
	MetricsVelocityPerDay      float64 `json:"metrics_velocity_per_day,omitempty" console:"header:Velocity/Day,omitempty"`
	MetricsEstimatedCompletion string  `json:"metrics_estimated_completion,omitempty" console:"header:ETA,omitempty"`

	// Optional durable cursor/checkpoint info from repo-memory.
	CursorPath      string `json:"cursor_path,omitempty" console:"header:Cursor Path,omitempty,maxlen:40"`
	CursorUpdatedAt string `json:"cursor_updated_at,omitempty" console:"header:Cursor Updated,omitempty,maxlen:30"`
}

// CampaignMetricsSnapshot describes the JSON structure used by campaign
// metrics snapshots written into the memory/campaigns branch.
//
// This mirrors the example in the campaigns guide:
//
//	{
//	  "date": "2025-01-16",
//	  "campaign_id": "security-q1-2025",
//	  "tasks_total": 200,
//	  "tasks_completed": 15,
//	  "tasks_in_progress": 30,
//	  "tasks_blocked": 5,
//	  "velocity_per_day": 7.5,
//	  "estimated_completion": "2025-02-12"
//	}
type CampaignMetricsSnapshot struct {
	Date                string  `json:"date"`            // Required: YYYY-MM-DD format
	CampaignID          string  `json:"campaign_id"`     // Required: campaign identifier
	TasksTotal          int     `json:"tasks_total"`     // Required: total task count (>= 0)
	TasksCompleted      int     `json:"tasks_completed"` // Required: completed task count (>= 0)
	TasksInProgress     int     `json:"tasks_in_progress,omitempty"`
	TasksBlocked        int     `json:"tasks_blocked,omitempty"`
	VelocityPerDay      float64 `json:"velocity_per_day,omitempty"`
	EstimatedCompletion string  `json:"estimated_completion,omitempty"`
}

// CampaignValidationResult represents the result of validating a campaign spec.
type CampaignValidationResult struct {
	ID         string   `json:"id" console:"header:ID"`
	Name       string   `json:"name" console:"header:Name"`
	ConfigPath string   `json:"config_path" console:"header:Config Path"`
	Problems   []string `json:"problems,omitempty" console:"header:Problems,omitempty"`
}
