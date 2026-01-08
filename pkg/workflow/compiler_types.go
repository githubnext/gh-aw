package workflow

import (
	"os"

	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/parser"
)

var logTypes = logger.New("workflow:compiler_types")

// FileTracker interface for tracking files created during compilation
type FileTracker interface {
	TrackCreated(filePath string)
}

// Compiler handles converting markdown workflows to GitHub Actions YAML
type Compiler struct {
	verbose              bool
	engineOverride       string
	customOutput         string              // If set, output will be written to this path instead of default location
	version              string              // Version of the extension
	skipValidation       bool                // If true, skip schema validation
	noEmit               bool                // If true, validate without generating lock files
	strictMode           bool                // If true, enforce strict validation requirements
	trialMode            bool                // If true, suppress safe outputs for trial mode execution
	trialLogicalRepoSlug string              // If set in trial mode, the logical repository to checkout
	refreshStopTime      bool                // If true, regenerate stop-after times instead of preserving existing ones
	markdownPath         string              // Path to the markdown file being compiled (for context in dynamic tool generation)
	actionMode           ActionMode          // Mode for generating JavaScript steps (inline vs custom actions)
	actionTag            string              // Override action SHA or tag for actions/setup (when set, overrides actionMode to release)
	jobManager           *JobManager         // Manages jobs and dependencies
	engineRegistry       *EngineRegistry     // Registry of available agentic engines
	fileTracker          FileTracker         // Optional file tracker for tracking created files
	warningCount         int                 // Number of warnings encountered during compilation
	stepOrderTracker     *StepOrderTracker   // Tracks step ordering for validation
	actionCache          *ActionCache        // Shared cache for action pin resolutions across all workflows
	actionResolver       *ActionResolver     // Shared resolver for action pins across all workflows
	importCache          *parser.ImportCache // Shared cache for imported workflow files
	workflowIdentifier   string              // Identifier for the current workflow being compiled (for schedule scattering)
	scheduleWarnings     []string            // Accumulated schedule warnings for this compiler instance
	repositorySlug       string              // Repository slug (owner/repo) used as seed for scattering
	artifactManager      *ArtifactManager    // Tracks artifact uploads/downloads for validation
}

// NewCompiler creates a new workflow compiler with optional configuration
func NewCompiler(verbose bool, engineOverride string, version string) *Compiler {
	c := &Compiler{
		verbose:          verbose,
		engineOverride:   engineOverride,
		version:          version,
		skipValidation:   true,          // Skip validation by default for now since existing workflows don't fully comply
		actionMode:       ActionModeDev, // Default to dev mode (local action paths)
		jobManager:       NewJobManager(),
		engineRegistry:   GetGlobalEngineRegistry(),
		stepOrderTracker: NewStepOrderTracker(),
		artifactManager:  NewArtifactManager(),
	}

	return c
}

// NewCompilerWithCustomOutput creates a new workflow compiler with custom output path
func NewCompilerWithCustomOutput(verbose bool, engineOverride string, customOutput string, version string) *Compiler {
	c := &Compiler{
		verbose:          verbose,
		engineOverride:   engineOverride,
		customOutput:     customOutput,
		version:          version,
		skipValidation:   true,          // Skip validation by default for now since existing workflows don't fully comply
		actionMode:       ActionModeDev, // Default to dev mode (local action paths)
		jobManager:       NewJobManager(),
		engineRegistry:   GetGlobalEngineRegistry(),
		stepOrderTracker: NewStepOrderTracker(),
		artifactManager:  NewArtifactManager(),
	}

	return c
}

// SetSkipValidation configures whether to skip schema validation
func (c *Compiler) SetSkipValidation(skip bool) {
	c.skipValidation = skip
}

// SetNoEmit configures whether to validate without generating lock files
func (c *Compiler) SetNoEmit(noEmit bool) {
	c.noEmit = noEmit
}

// SetFileTracker sets the file tracker for tracking created files
func (c *Compiler) SetFileTracker(tracker FileTracker) {
	c.fileTracker = tracker
}

// SetTrialMode configures whether to run in trial mode (suppresses safe outputs)
func (c *Compiler) SetTrialMode(trialMode bool) {
	c.trialMode = trialMode
}

// SetTrialLogicalRepoSlug configures the target repository for trial mode
func (c *Compiler) SetTrialLogicalRepoSlug(repo string) {
	c.trialLogicalRepoSlug = repo
}

// SetStrictMode configures whether to enable strict validation mode
func (c *Compiler) SetStrictMode(strict bool) {
	c.strictMode = strict
}

// SetRefreshStopTime configures whether to force regeneration of stop-after times
func (c *Compiler) SetRefreshStopTime(refresh bool) {
	c.refreshStopTime = refresh
}

// SetActionMode configures the action mode for JavaScript step generation
func (c *Compiler) SetActionMode(mode ActionMode) {
	c.actionMode = mode
}

// GetActionMode returns the current action mode
func (c *Compiler) GetActionMode() ActionMode {
	return c.actionMode
}

// SetActionTag sets the action tag override for actions/setup
func (c *Compiler) SetActionTag(tag string) {
	c.actionTag = tag
}

// GetActionTag returns the action tag override (empty if not set)
func (c *Compiler) GetActionTag() string {
	return c.actionTag
}

// GetVersion returns the version string used by the compiler
func (c *Compiler) GetVersion() string {
	return c.version
}

// IncrementWarningCount increments the warning counter
func (c *Compiler) IncrementWarningCount() {
	c.warningCount++
}

// GetWarningCount returns the current warning count
func (c *Compiler) GetWarningCount() int {
	return c.warningCount
}

// ResetWarningCount resets the warning counter to zero
func (c *Compiler) ResetWarningCount() {
	c.warningCount = 0
}

// SetWorkflowIdentifier sets the identifier for the current workflow being compiled
// This is used for deterministic schedule scattering
func (c *Compiler) SetWorkflowIdentifier(identifier string) {
	c.workflowIdentifier = identifier
}

// GetWorkflowIdentifier returns the current workflow identifier
func (c *Compiler) GetWorkflowIdentifier() string {
	return c.workflowIdentifier
}

// SetRepositorySlug sets the repository slug for schedule scattering
func (c *Compiler) SetRepositorySlug(slug string) {
	c.repositorySlug = slug
}

// GetRepositorySlug returns the repository slug
func (c *Compiler) GetRepositorySlug() string {
	return c.repositorySlug
}

// GetScheduleWarnings returns all accumulated schedule warnings for this compiler instance
func (c *Compiler) GetScheduleWarnings() []string {
	return c.scheduleWarnings
}

// getSharedActionResolver returns the shared action resolver, initializing it on first use
// This ensures all workflows compiled by this compiler instance share the same in-memory cache
func (c *Compiler) getSharedActionResolver() (*ActionCache, *ActionResolver) {
	if c.actionCache == nil {
		// Initialize cache and resolver on first use
		cwd, err := os.Getwd()
		if err != nil {
			cwd = "."
		}
		c.actionCache = NewActionCache(cwd)
		_ = c.actionCache.Load() // Ignore errors if cache doesn't exist
		c.actionResolver = NewActionResolver(c.actionCache)
		logTypes.Print("Initialized shared action cache and resolver for compiler")
	}
	return c.actionCache, c.actionResolver
}

// getSharedImportCache returns the shared import cache, initializing it on first use
// This ensures all workflows compiled by this compiler instance share the same import cache
func (c *Compiler) getSharedImportCache() *parser.ImportCache {
	if c.importCache == nil {
		// Initialize cache on first use
		cwd, err := os.Getwd()
		if err != nil {
			cwd = "."
		}
		c.importCache = parser.NewImportCache(cwd)
		logTypes.Print("Initialized shared import cache for compiler")
	}
	return c.importCache
}

// GetSharedActionCache returns the shared action cache used by this compiler instance.
// The cache is lazily initialized on first access and shared across all workflows.
// This allows action SHA validation and other operations to reuse cached resolutions.
func (c *Compiler) GetSharedActionCache() *ActionCache {
	cache, _ := c.getSharedActionResolver()
	return cache
}

// GetArtifactManager returns the artifact manager for tracking uploads/downloads
func (c *Compiler) GetArtifactManager() *ArtifactManager {
	if c.artifactManager == nil {
		c.artifactManager = NewArtifactManager()
	}
	return c.artifactManager
}

// SkipIfMatchConfig holds the configuration for skip-if-match conditions
type SkipIfMatchConfig struct {
	Query string // GitHub search query to check before running workflow
	Max   int    // Maximum number of matches before skipping (defaults to 1)
}

// SkipIfNoMatchConfig holds the configuration for skip-if-no-match conditions
type SkipIfNoMatchConfig struct {
	Query string // GitHub search query to check before running workflow
	Min   int    // Minimum number of matches required to proceed (defaults to 1)
}

// WorkflowData holds all the data needed to generate a GitHub Actions workflow
type WorkflowData struct {
	Name                string
	TrialMode           bool           // whether the workflow is running in trial mode
	TrialLogicalRepo    string         // target repository slug for trial mode (owner/repo)
	FrontmatterName     string         // name field from frontmatter (for code scanning alert driver default)
	FrontmatterYAML     string         // raw frontmatter YAML content (rendered as comment in lock file for reference)
	Description         string         // optional description rendered as comment in lock file
	Source              string         // optional source field (owner/repo@ref/path) rendered as comment in lock file
	TrackerID           string         // optional tracker identifier for created assets (min 8 chars, alphanumeric + hyphens/underscores)
	ImportedFiles       []string       // list of files imported via imports field (rendered as comment in lock file)
	IncludedFiles       []string       // list of files included via @include directives (rendered as comment in lock file)
	ImportInputs        map[string]any // input values from imports with inputs (for github.aw.inputs.* substitution)
	On                  string
	Permissions         string
	Network             string // top-level network permissions configuration
	Concurrency         string // workflow-level concurrency configuration
	RunName             string
	Env                 string
	If                  string
	TimeoutMinutes      string
	CustomSteps         string
	PostSteps           string // steps to run after AI execution
	RunsOn              string
	Environment         string // environment setting for the main job
	Container           string // container setting for the main job
	Services            string // services setting for the main job
	Tools               map[string]any
	ParsedTools         *Tools // Structured tools configuration (NEW: parsed from Tools map)
	MarkdownContent     string
	AI                  string        // "claude" or "codex" (for backwards compatibility)
	EngineConfig        *EngineConfig // Extended engine configuration
	AgentFile           string        // Path to custom agent file (from imports)
	StopTime            string
	SkipIfMatch         *SkipIfMatchConfig   // skip-if-match configuration with query and max threshold
	SkipIfNoMatch       *SkipIfNoMatchConfig // skip-if-no-match configuration with query and min threshold
	ManualApproval      string               // environment name for manual approval from on: section
	Command             []string             // for /command trigger support - multiple command names
	CommandEvents       []string             // events where command should be active (nil = all events)
	CommandOtherEvents  map[string]any       // for merging command with other events
	AIReaction          string               // AI reaction type like "eyes", "heart", etc.
	LockForAgent        bool                 // whether to lock the issue during agent workflow execution
	Jobs                map[string]any       // custom job configurations with dependencies
	Cache               string               // cache configuration
	NeedsTextOutput     bool                 // whether the workflow uses ${{ needs.task.outputs.text }}
	NetworkPermissions  *NetworkPermissions  // parsed network permissions
	SandboxConfig       *SandboxConfig       // parsed sandbox configuration (AWF or SRT)
	SafeOutputs         *SafeOutputsConfig   // output configuration for automatic output routes
	SafeInputs          *SafeInputsConfig    // safe-inputs configuration for custom MCP tools
	Roles               []string             // permission levels required to trigger workflow
	Bots                []string             // allow list of bot identifiers that can trigger workflow
	CacheMemoryConfig   *CacheMemoryConfig   // parsed cache-memory configuration
	RepoMemoryConfig    *RepoMemoryConfig    // parsed repo-memory configuration
	SafetyPrompt        bool                 // whether to include XPIA safety prompt (default true)
	Runtimes            map[string]any       // runtime version overrides from frontmatter
	ToolsTimeout        int                  // timeout in seconds for tool/MCP operations (0 = use engine default)
	GitHubToken         string               // top-level github-token expression from frontmatter
	ToolsStartupTimeout int                  // timeout in seconds for MCP server startup (0 = use engine default)
	Features            map[string]any       // feature flags and configuration options from frontmatter (supports bool and string values)
	ActionCache         *ActionCache         // cache for action pin resolutions
	ActionResolver      *ActionResolver      // resolver for action pins
	StrictMode          bool                 // strict mode for action pinning
	SecretMasking       *SecretMaskingConfig // secret masking configuration
}

// BaseSafeOutputConfig holds common configuration fields for all safe output types
type BaseSafeOutputConfig struct {
	Max         int    `yaml:"max,omitempty"`          // Maximum number of items to create
	GitHubToken string `yaml:"github-token,omitempty"` // GitHub token for this specific output type
}

// SafeOutputsConfig holds configuration for automatic output routes
type SafeOutputsConfig struct {
	CreateIssues                    *CreateIssuesConfig                    `yaml:"create-issues,omitempty"`
	CreateDiscussions               *CreateDiscussionsConfig               `yaml:"create-discussions,omitempty"`
	UpdateDiscussions               *UpdateDiscussionsConfig               `yaml:"update-discussion,omitempty"`
	CloseDiscussions                *CloseDiscussionsConfig                `yaml:"close-discussions,omitempty"`
	CloseIssues                     *CloseIssuesConfig                     `yaml:"close-issue,omitempty"`
	ClosePullRequests               *ClosePullRequestsConfig               `yaml:"close-pull-request,omitempty"`
	MarkPullRequestAsReadyForReview *MarkPullRequestAsReadyForReviewConfig `yaml:"mark-pull-request-as-ready-for-review,omitempty"`
	AddComments                     *AddCommentsConfig                     `yaml:"add-comments,omitempty"`
	CreatePullRequests              *CreatePullRequestsConfig              `yaml:"create-pull-requests,omitempty"`
	CreatePullRequestReviewComments *CreatePullRequestReviewCommentsConfig `yaml:"create-pull-request-review-comments,omitempty"`
	CreateCodeScanningAlerts        *CreateCodeScanningAlertsConfig        `yaml:"create-code-scanning-alerts,omitempty"`
	AddLabels                       *AddLabelsConfig                       `yaml:"add-labels,omitempty"`
	AddReviewer                     *AddReviewerConfig                     `yaml:"add-reviewer,omitempty"`
	AssignMilestone                 *AssignMilestoneConfig                 `yaml:"assign-milestone,omitempty"`
	AssignToAgent                   *AssignToAgentConfig                   `yaml:"assign-to-agent,omitempty"`
	AssignToUser                    *AssignToUserConfig                    `yaml:"assign-to-user,omitempty"` // Assign users to issues
	UpdateIssues                    *UpdateIssuesConfig                    `yaml:"update-issues,omitempty"`
	UpdatePullRequests              *UpdatePullRequestsConfig              `yaml:"update-pull-request,omitempty"` // Update GitHub pull request title/body
	PushToPullRequestBranch         *PushToPullRequestBranchConfig         `yaml:"push-to-pull-request-branch,omitempty"`
	UploadAssets                    *UploadAssetsConfig                    `yaml:"upload-asset,omitempty"`
	UpdateRelease                   *UpdateReleaseConfig                   `yaml:"update-release,omitempty"`               // Update GitHub release descriptions
	CreateAgentSessions             *CreateAgentSessionConfig              `yaml:"create-agent-session,omitempty"`         // Create GitHub Copilot agent sessions
	UpdateProjects                  *UpdateProjectConfig                   `yaml:"update-project,omitempty"`               // Smart project board management (create/add/update)
	CopyProjects                    *CopyProjectsConfig                    `yaml:"copy-project,omitempty"`                 // Copy GitHub Projects V2
	CreateProjectStatusUpdates      *CreateProjectStatusUpdateConfig       `yaml:"create-project-status-update,omitempty"` // Create GitHub project status updates
	LinkSubIssue                    *LinkSubIssueConfig                    `yaml:"link-sub-issue,omitempty"`               // Link issues as sub-issues
	HideComment                     *HideCommentConfig                     `yaml:"hide-comment,omitempty"`                 // Hide comments
	DispatchWorkflow                *DispatchWorkflowConfig                `yaml:"dispatch-workflow,omitempty"`            // Dispatch workflow_dispatch events to other workflows
	MissingTool                     *MissingToolConfig                     `yaml:"missing-tool,omitempty"`                 // Optional for reporting missing functionality
	MissingData                     *MissingDataConfig                     `yaml:"missing-data,omitempty"`                 // Optional for reporting missing data required to achieve goals
	NoOp                            *NoOpConfig                            `yaml:"noop,omitempty"`                         // No-op output for logging only (always available as fallback)
	ThreatDetection                 *ThreatDetectionConfig                 `yaml:"threat-detection,omitempty"`             // Threat detection configuration
	Jobs                            map[string]*SafeJobConfig              `yaml:"jobs,omitempty"`                         // Safe-jobs configuration (moved from top-level)
	App                             *GitHubAppConfig                       `yaml:"app,omitempty"`                          // GitHub App credentials for token minting
	AllowedDomains                  []string                               `yaml:"allowed-domains,omitempty"`
	AllowGitHubReferences           []string                               `yaml:"allowed-github-references,omitempty"` // Allowed repositories for GitHub references (e.g., ["repo", "org/repo2"])
	Staged                          bool                                   `yaml:"staged,omitempty"`                    // If true, emit step summary messages instead of making GitHub API calls
	Env                             map[string]string                      `yaml:"env,omitempty"`                       // Environment variables to pass to safe output jobs
	GitHubToken                     string                                 `yaml:"github-token,omitempty"`              // GitHub token for safe output jobs
	MaximumPatchSize                int                                    `yaml:"max-patch-size,omitempty"`            // Maximum allowed patch size in KB (defaults to 1024)
	RunsOn                          string                                 `yaml:"runs-on,omitempty"`                   // Runner configuration for safe-outputs jobs
	Messages                        *SafeOutputMessagesConfig              `yaml:"messages,omitempty"`                  // Custom message templates for footer and notifications
	Mentions                        *MentionsConfig                        `yaml:"mentions,omitempty"`                  // Configuration for @mention filtering in safe outputs
}

// SafeOutputMessagesConfig holds custom message templates for safe-output footer and notification messages
type SafeOutputMessagesConfig struct {
	Footer                         string `yaml:"footer,omitempty" json:"footer,omitempty"`                                                    // Custom footer message template
	FooterInstall                  string `yaml:"footer-install,omitempty" json:"footerInstall,omitempty"`                                     // Custom installation instructions template
	FooterWorkflowRecompile        string `yaml:"footer-workflow-recompile,omitempty" json:"footerWorkflowRecompile,omitempty"`                // Custom footer template for workflow recompile issues
	FooterWorkflowRecompileComment string `yaml:"footer-workflow-recompile-comment,omitempty" json:"footerWorkflowRecompileComment,omitempty"` // Custom footer template for comments on workflow recompile issues
	StagedTitle                    string `yaml:"staged-title,omitempty" json:"stagedTitle,omitempty"`                                         // Custom staged mode title template
	StagedDescription              string `yaml:"staged-description,omitempty" json:"stagedDescription,omitempty"`                             // Custom staged mode description template
	RunStarted                     string `yaml:"run-started,omitempty" json:"runStarted,omitempty"`                                           // Custom workflow activation message template
	RunSuccess                     string `yaml:"run-success,omitempty" json:"runSuccess,omitempty"`                                           // Custom workflow success message template
	RunFailure                     string `yaml:"run-failure,omitempty" json:"runFailure,omitempty"`                                           // Custom workflow failure message template
	DetectionFailure               string `yaml:"detection-failure,omitempty" json:"detectionFailure,omitempty"`                               // Custom detection job failure message template
}

// MentionsConfig holds configuration for @mention filtering in safe outputs
type MentionsConfig struct {
	// Enabled can be:
	//   true: mentions always allowed (error in strict mode)
	//   false: mentions always escaped
	//   nil: use default behavior with team members and context
	Enabled *bool `yaml:"enabled,omitempty" json:"enabled,omitempty"`

	// AllowTeamMembers determines if team members can be mentioned (default: true)
	AllowTeamMembers *bool `yaml:"allow-team-members,omitempty" json:"allowTeamMembers,omitempty"`

	// AllowContext determines if mentions from event context are allowed (default: true)
	AllowContext *bool `yaml:"allow-context,omitempty" json:"allowContext,omitempty"`

	// Allowed is a list of user/bot names always allowed (bots not allowed by default)
	Allowed []string `yaml:"allowed,omitempty" json:"allowed,omitempty"`

	// Max is the maximum number of mentions per message (default: 50)
	Max *int `yaml:"max,omitempty" json:"max,omitempty"`
}

// SecretMaskingConfig holds configuration for secret redaction behavior
type SecretMaskingConfig struct {
	Steps []map[string]any `yaml:"steps,omitempty"` // Additional secret redaction steps to inject after built-in redaction
}
