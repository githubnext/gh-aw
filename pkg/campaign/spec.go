package campaign

// CampaignSpec defines a first-class campaign configuration loaded from
// YAML frontmatter in Markdown files.
//
// Files are discovered from the local repository under:
//
//	campaigns/*.campaign.md
//
// This provides a thin, declarative layer on top of existing agentic
// workflows and repo-memory conventions.
type CampaignSpec struct {
	ID          string `yaml:"id" json:"id" console:"header:ID"`
	Name        string `yaml:"name" json:"name" console:"header:Name"`
	Description string `yaml:"description,omitempty" json:"description,omitempty" console:"header:Description,omitempty"`

	// Version is an optional spec version string (for example: v1).
	// When omitted, it defaults to v1 during validation.
	Version string `yaml:"version,omitempty" json:"version,omitempty" console:"header:Version,omitempty"`

	// Workflows associates this campaign with one or more workflow IDs
	// (basename of the Markdown file without .md).
	Workflows []string `yaml:"workflows,omitempty" json:"workflows,omitempty" console:"header:Workflows,omitempty"`

	// MemoryPaths documents where this campaign writes its repo-memory
	// (for example: memory/campaigns/incident-*/**).
	MemoryPaths []string `yaml:"memory_paths,omitempty" json:"memory_paths,omitempty" console:"header:Memory Paths,omitempty"`

	// MetricsGlob is an optional glob (relative to the repository root)
	// used to locate JSON metrics snapshots stored in the
	// memory/campaigns branch. When set, `gh aw campaign status` will
	// attempt to read the latest matching metrics file and surface a few
	// key fields.
	MetricsGlob string `yaml:"metrics_glob,omitempty" json:"metrics_glob,omitempty" console:"header:Metrics Glob,omitempty"`

	// Owners lists the primary human owners for this campaign.
	Owners []string `yaml:"owners,omitempty" json:"owners,omitempty" console:"header:Owners,omitempty"`

	// ExecutiveSponsors lists executive stakeholders or sponsors who are
	// accountable for the outcome of this campaign.
	ExecutiveSponsors []string `yaml:"executive_sponsors,omitempty" json:"executive_sponsors,omitempty" console:"header:Executive Sponsors,omitempty"`

	// RiskLevel is an optional free-form field (e.g. low/medium/high).
	RiskLevel string `yaml:"risk_level,omitempty" json:"risk_level,omitempty" console:"header:Risk Level,omitempty"`

	// TrackerLabel describes the label used to associate issues/PRs with
	// this campaign (for example: campaign:incident-response).
	TrackerLabel string `yaml:"tracker_label,omitempty" json:"tracker_label,omitempty" console:"header:Tracker Label,omitempty"`

	// State describes the lifecycle stage of the campaign definition.
	// Valid values are: planned, active, paused, completed, archived.
	State string `yaml:"state,omitempty" json:"state,omitempty" console:"header:State,omitempty"`

	// Tags provide free-form categorization for reporting (for example:
	// security, modernization, rollout).
	Tags []string `yaml:"tags,omitempty" json:"tags,omitempty" console:"header:Tags,omitempty"`

	// AllowedSafeOutputs documents which safe-outputs operations this
	// campaign is expected to use (for example: create-issue,
	// create-pull-request). This is currently informational but can be
	// enforced by validation in the future.
	AllowedSafeOutputs []string `yaml:"allowed_safe_outputs,omitempty" json:"allowed_safe_outputs,omitempty" console:"header:Allowed Safe Outputs,omitempty"`

	// ApprovalPolicy describes high-level approval expectations for this
	// campaign (for example: number of approvals and required roles).
	ApprovalPolicy *CampaignApprovalPolicy `yaml:"approval_policy,omitempty" json:"approval_policy,omitempty"`

	// ConfigPath is populated at load time with the relative path of
	// the YAML file on disk, to help users locate definitions.
	ConfigPath string `yaml:"-" json:"config_path" console:"header:Config Path"`
}

// CampaignApprovalPolicy captures basic approval expectations for a
// campaign. It is intentionally lightweight and advisory; enforcement
// is left to workflows and organizational process.
type CampaignApprovalPolicy struct {
	RequiredApprovals int      `yaml:"required_approvals,omitempty" json:"required_approvals,omitempty"`
	RequiredRoles     []string `yaml:"required_roles,omitempty" json:"required_roles,omitempty"`
	ChangeControl     bool     `yaml:"change_control,omitempty" json:"change_control,omitempty"`
}

// CampaignRuntimeStatus represents the live status of a campaign, including
// compiled workflow state and basic issue/PR counts derived from the tracker
// label.
type CampaignRuntimeStatus struct {
	ID           string   `json:"id" console:"header:ID"`
	Name         string   `json:"name" console:"header:Name"`
	TrackerLabel string   `json:"tracker_label,omitempty" console:"header:Tracker Label,omitempty"`
	Workflows    []string `json:"workflows,omitempty" console:"header:Workflows,omitempty"`
	Compiled     string   `json:"compiled" console:"header:Compiled"`

	IssuesOpen   int `json:"issues_open,omitempty" console:"header:Issues Open,omitempty"`
	IssuesClosed int `json:"issues_closed,omitempty" console:"header:Issues Closed,omitempty"`
	PRsOpen      int `json:"prs_open,omitempty" console:"header:PRs Open,omitempty"`
	PRsMerged    int `json:"prs_merged,omitempty" console:"header:PRs Merged,omitempty"`

	// Optional metrics from repo-memory (when MetricsGlob is set and a
	// matching JSON snapshot is found on the memory/campaigns branch).
	MetricsTasksTotal          int     `json:"metrics_tasks_total,omitempty" console:"header:Tasks Total,omitempty"`
	MetricsTasksCompleted      int     `json:"metrics_tasks_completed,omitempty" console:"header:Tasks Completed,omitempty"`
	MetricsVelocityPerDay      float64 `json:"metrics_velocity_per_day,omitempty" console:"header:Velocity/Day,omitempty"`
	MetricsEstimatedCompletion string  `json:"metrics_estimated_completion,omitempty" console:"header:ETA,omitempty"`
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
	Date                string  `json:"date,omitempty"`
	CampaignID          string  `json:"campaign_id,omitempty"`
	TasksTotal          int     `json:"tasks_total,omitempty"`
	TasksCompleted      int     `json:"tasks_completed,omitempty"`
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
