package workflow

import (
	"os"

	actionpins "github.com/github/gh-aw/pkg/actionpins"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
)

var logTypes = logger.New("workflow:compiler_types")

// CompilerOption is a functional option for configuring a Compiler
type CompilerOption func(*Compiler)

// WithVerbose sets the verbose logging flag
func WithVerbose(verbose bool) CompilerOption {
	return func(c *Compiler) { c.verbose = verbose }
}

// WithEngineOverride sets the AI engine override
func WithEngineOverride(engine string) CompilerOption {
	return func(c *Compiler) { c.engineOverride = engine }
}

// WithSkipValidation configures whether to skip schema validation
func WithSkipValidation(skip bool) CompilerOption {
	return func(c *Compiler) { c.skipValidation = skip }
}

// WithNoEmit configures whether to validate without generating lock files
func WithNoEmit(noEmit bool) CompilerOption {
	return func(c *Compiler) { c.noEmit = noEmit }
}

// WithFailFast configures whether to stop at first validation error
func WithFailFast(failFast bool) CompilerOption {
	return func(c *Compiler) { c.failFast = failFast }
}

// WithWorkflowIdentifier sets the identifier for the current workflow being compiled
func WithWorkflowIdentifier(identifier string) CompilerOption {
	return func(c *Compiler) { c.workflowIdentifier = identifier }
}

// WithVersion sets the compiler version, used to determine action mode and version-specific behavior
func WithVersion(version string) CompilerOption {
	return func(c *Compiler) { c.version = version }
}

// FileCreationTracker interface for tracking files created during compilation
type FileCreationTracker interface {
	TrackCreated(filePath string)
}

// Compiler handles converting markdown workflows to GitHub Actions YAML
type Compiler struct {
	verbose                 bool
	quiet                   bool // If true, suppress success messages (for interactive mode)
	engineOverride          string
	customOutput            string                   // If set, output will be written to this path instead of default location
	version                 string                   // Version of the extension
	skipValidation          bool                     // If true, skip schema validation
	noEmit                  bool                     // If true, validate without generating lock files
	strictMode              bool                     // If true, enforce strict validation requirements
	allowActionRefs         bool                     // If true, unresolved action refs are warnings instead of errors
	approve                 bool                     // If true, approve safe update changes (skip safe update enforcement)
	trialMode               bool                     // If true, suppress safe outputs for trial mode execution
	trialLogicalRepoSlug    string                   // If set in trial mode, the logical repository to checkout
	refreshStopTime         bool                     // If true, regenerate stop-after times instead of preserving existing ones
	forceRefreshActionPins  bool                     // If true, clear action cache and resolve all actions from GitHub API
	failFast                bool                     // If true, stop at first validation error instead of collecting all errors
	actionCacheCleared      bool                     // Tracks if action cache has already been cleared (for forceRefreshActionPins)
	markdownPath            string                   // Path to the markdown file being compiled (for context in dynamic tool generation)
	actionMode              ActionMode               // Mode for generating JavaScript steps (inline vs custom actions)
	actionTag               string                   // Override action SHA or tag for actions/setup (when set, overrides actionMode to release)
	actionsRepo             string                   // Override the external actions repository (default: github/gh-aw-actions)
	jobManager              *JobManager              // Manages jobs and dependencies
	engineRegistry          *EngineRegistry          // Registry of available agentic engines
	engineCatalog           *EngineCatalog           // Catalog of engine definitions backed by the registry
	fileTracker             FileCreationTracker      // Optional file tracker for tracking created files
	warningCount            int                      // Number of warnings encountered during compilation
	stepOrderTracker        *StepOrderTracker        // Tracks step ordering for validation
	actionCache             *ActionCache             // Shared cache for action pin resolutions across all workflows
	actionResolver          *ActionResolver          // Shared resolver for action pins across all workflows
	actionPinWarnings       map[string]bool          // Shared cache of already-warned action pin failures (key: "repo@version")
	importCache             *parser.ImportCache      // Shared cache for imported workflow files
	workflowIdentifier      string                   // Identifier for the current workflow being compiled (for schedule scattering)
	scheduleWarnings        []string                 // Accumulated schedule warnings for this compiler instance
	safeUpdateWarnings      []string                 // Accumulated safe update warnings (new secrets/actions requiring review)
	repositorySlug          string                   // Repository slug (owner/repo) used as seed for scattering
	artifactManager         *ArtifactManager         // Tracks artifact uploads/downloads for validation
	scheduleFriendlyFormats map[int]string           // Maps schedule item index to friendly format string for current workflow
	gitRoot                 string                   // Git repository root directory (if set, used for action cache path)
	repoConfig              *RepoConfig              // Cached repository-level aw.json config
	repoConfigErr           error                    // Cached repo config load error
	repoConfigLoaded        bool                     // True once repo config has been loaded (success or failure)
	contentOverride         string                   // If set, use this content instead of reading from disk (for Wasm/in-memory compilation)
	skipHeader              bool                     // If true, skip ASCII art header in generated YAML (for Wasm/editor mode)
	inlinePrompt            bool                     // If true, inline markdown content in YAML instead of using runtime-import macros (for Wasm builds)
	priorManifests          map[string]*GHAWManifest // Pre-cached manifests keyed by lock file path; takes precedence over git HEAD / filesystem reads
	requireDocker           bool                     // If true, fail validation when Docker is not available instead of silently skipping
}

// NewCompiler creates a new workflow compiler with functional options.
// By default, it auto-detects the version and action mode.
// Common options: WithVerbose, WithEngineOverride, WithNoEmit, WithSkipValidation
func NewCompiler(opts ...CompilerOption) *Compiler {
	// Get the current compiler version (set by SetVersion during CLI initialization)
	version := GetVersion()

	// Auto-detect git repository root for action cache path resolution
	// This ensures actions-lock.json is created at repo root regardless of CWD
	gitRoot := findGitRoot()

	// Create compiler with defaults
	c := &Compiler{
		verbose:           false,
		engineOverride:    "",
		version:           version,
		skipValidation:    true,                      // Skip validation by default for now since existing workflows don't fully comply
		actionMode:        DetectActionMode(version), // Auto-detect action mode based on version
		jobManager:        NewJobManager(),
		engineRegistry:    GetGlobalEngineRegistry(),
		engineCatalog:     NewEngineCatalog(GetGlobalEngineRegistry()),
		stepOrderTracker:  NewStepOrderTracker(),
		artifactManager:   NewArtifactManager(),
		actionPinWarnings: make(map[string]bool), // Initialize warning cache
		gitRoot:           gitRoot,               // Auto-detected git root
	}

	// Apply functional options
	for _, opt := range opts {
		opt(c)
	}
	// Auto-detect action mode based on version in case version has been update
	c.actionMode = DetectActionMode(c.version)

	logTypes.Printf("Created compiler: version=%s, actionMode=%s, skipValidation=%t, strictMode=%t", c.version, c.actionMode, c.skipValidation, c.strictMode)

	return c
}

// SetSkipValidation configures whether to skip schema validation
func (c *Compiler) SetSkipValidation(skip bool) {
	c.skipValidation = skip
}

// SetRequireDocker configures whether Docker must be available for container image validation.
// When true, validation fails with an error if Docker is not installed or the daemon is not running.
// When false (default), validation is silently skipped when Docker is unavailable.
func (c *Compiler) SetRequireDocker(require bool) {
	c.requireDocker = require
}

// SetQuiet configures whether to suppress success messages (for interactive mode)
func (c *Compiler) SetQuiet(quiet bool) {
	c.quiet = quiet
}

// SetNoEmit configures whether to validate without generating lock files
func (c *Compiler) SetNoEmit(noEmit bool) {
	c.noEmit = noEmit
}

// SetApprove configures whether to skip safe update enforcement via the CLI --approve flag.
// When true, safe update enforcement is disabled regardless of strict mode setting,
// approving all changes.
func (c *Compiler) SetApprove(approve bool) {
	c.approve = approve
}

// SetFileTracker sets the file tracker for tracking created files
func (c *Compiler) SetFileTracker(tracker FileCreationTracker) {
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

// SetAllowActionRefs configures whether unresolved action refs are warnings.
// When false (default), unresolved action refs are compiler errors.
func (c *Compiler) SetAllowActionRefs(allow bool) {
	c.allowActionRefs = allow
}

// SetRefreshStopTime configures whether to force regeneration of stop-after times
func (c *Compiler) SetRefreshStopTime(refresh bool) {
	c.refreshStopTime = refresh
}

// SetForceRefreshActionPins configures whether to force refresh of action pins
func (c *Compiler) SetForceRefreshActionPins(force bool) {
	c.forceRefreshActionPins = force
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

// SetActionsRepo sets the external actions repository override.
// When set, this overrides the default "github/gh-aw-actions" repository used in action mode.
func (c *Compiler) SetActionsRepo(repo string) {
	c.actionsRepo = repo
}

// effectiveActionsRepo returns the actions repository to use for action mode references.
// Returns the override if set, otherwise returns the default GitHubActionsOrgRepo constant.
func (c *Compiler) effectiveActionsRepo() string {
	if c.actionsRepo != "" {
		return c.actionsRepo
	}
	return GitHubActionsOrgRepo
}

// EffectiveActionsRepo returns the actions repository used for action mode references.
// Returns the override if set, otherwise returns the default GitHubActionsOrgRepo.
func (c *Compiler) EffectiveActionsRepo() string {
	return c.effectiveActionsRepo()
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

// SetRepositorySlug sets the repository slug for schedule scattering
func (c *Compiler) SetRepositorySlug(slug string) {
	c.repositorySlug = slug
}

// GetRepositorySlug returns the repository slug (owner/repo) set on this compiler instance.
func (c *Compiler) GetRepositorySlug() string {
	return c.repositorySlug
}

// GetScheduleWarnings returns all accumulated schedule warnings for this compiler instance
func (c *Compiler) GetScheduleWarnings() []string {
	return c.scheduleWarnings
}

// AddSafeUpdateWarning appends a safe update warning to the compiler's accumulated list.
// Callers should invoke this when a safe update violation is detected instead of
// returning a compilation error, so that compilation still succeeds and the agent
// receives actionable guidance.
func (c *Compiler) AddSafeUpdateWarning(warning string) {
	if c.safeUpdateWarnings == nil {
		c.safeUpdateWarnings = []string{}
	}
	c.safeUpdateWarnings = append(c.safeUpdateWarnings, warning)
}

// GetSafeUpdateWarnings returns all accumulated safe update warnings for this compiler instance.
func (c *Compiler) GetSafeUpdateWarnings() []string {
	return c.safeUpdateWarnings
}

// SetPriorManifests replaces the entire pre-cached manifest map.
func (c *Compiler) SetPriorManifests(manifests map[string]*GHAWManifest) {
	c.priorManifests = manifests
}

// getSharedActionResolver returns the shared action resolver, initializing it on first use
// This ensures all workflows compiled by this compiler instance share the same in-memory cache
func (c *Compiler) getSharedActionResolver() (*ActionCache, *ActionResolver) {
	if c.actionCache == nil {
		// Initialize cache and resolver on first use
		// Use git root if provided, otherwise fall back to current working directory
		baseDir := c.gitRoot
		if baseDir == "" {
			cwd, err := os.Getwd()
			if err != nil {
				cwd = "."
			}
			baseDir = cwd
		}
		c.actionCache = NewActionCache(baseDir)

		// Load existing cache unless force refresh is enabled
		if !c.forceRefreshActionPins {
			_ = c.actionCache.Load() // Ignore errors if cache doesn't exist
		} else {
			logTypes.Print("Force refresh action pins enabled: skipping cache load and will resolve all actions dynamically")
			// Mark as cleared since we skipped loading
			c.actionCacheCleared = true
		}

		c.actionResolver = NewActionResolver(c.actionCache)
		logTypes.Print("Initialized shared action cache and resolver for compiler")
	} else if c.forceRefreshActionPins && !c.actionCacheCleared {
		// If cache already exists but force refresh is set and we haven't cleared it yet, clear it once
		logTypes.Print("Force refresh action pins: clearing existing cache once for this run")
		c.actionCache.Entries = make(map[string]ActionCacheEntry)
		c.actionCacheCleared = true
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

// SkipIfMatchConfig holds the configuration for skip-if-match conditions
type SkipIfMatchConfig struct {
	Query string // GitHub search query to check before running workflow
	Max   int    // Maximum number of matches before skipping (defaults to 1)
	Scope string // Scope for the query: "none" disables auto repo:owner/repo scoping
	// Auth (github-token / github-app) is taken from on.github-token / on.github-app at the top level.
}

// SkipIfNoMatchConfig holds the configuration for skip-if-no-match conditions
type SkipIfNoMatchConfig struct {
	Query string // GitHub search query to check before running workflow
	Min   int    // Minimum number of matches required to proceed (defaults to 1)
	Scope string // Scope for the query: "none" disables auto repo:owner/repo scoping
	// Auth (github-token / github-app) is taken from on.github-token / on.github-app at the top level.
}

// SkipIfCheckFailingConfig holds the configuration for skip-if-check-failing conditions
type SkipIfCheckFailingConfig struct {
	Include      []string // check names to include (empty = all checks)
	Exclude      []string // check names to exclude
	Branch       string   // optional branch name to check (defaults to triggering ref or PR base branch)
	AllowPending bool     // if true, pending/in-progress checks are not treated as failing (default: treat pending as failing)
}
type WorkflowData struct {
	Name                         string
	WorkflowID                   string         // workflow identifier derived from markdown filename (basename without extension)
	TrialMode                    bool           // whether the workflow is running in trial mode
	TrialLogicalRepo             string         // target repository slug for trial mode (owner/repo)
	FrontmatterName              string         // name field from frontmatter (for code scanning alert driver default)
	FrontmatterYAML              string         // raw frontmatter YAML content (rendered as comment in lock file for reference)
	FrontmatterHash              string         // SHA-256 hash of frontmatter (computed before job building, used to derive stable heredoc delimiters)
	Description                  string         // optional description rendered as comment in lock file
	Source                       string         // optional source field (owner/repo@ref/path) rendered as comment in lock file
	Redirect                     string         // optional redirect field describing a moved workflow location
	TrackerID                    string         // optional tracker identifier for created assets (min 8 chars, alphanumeric + hyphens/underscores)
	ImportedFiles                []string       // list of files imported via imports field (rendered as comment in lock file)
	ImportedMarkdown             string         // Only imports WITH inputs (for compile-time substitution)
	ImportPaths                  []string       // Import file paths for runtime-import macro generation (imports without inputs)
	MainWorkflowMarkdown         string         // main workflow markdown without imports (for runtime-import)
	IncludedFiles                []string       // list of files included via @include directives (rendered as comment in lock file)
	ImportInputs                 map[string]any // input values from imports with inputs (for github.aw.inputs.* substitution)
	On                           string
	Permissions                  string
	Network                      string // top-level network permissions configuration
	Concurrency                  string // workflow-level concurrency configuration
	RunName                      string
	Env                          string
	EnvSources                   map[string]string // env var name → source ("(main workflow)" or import file path) for lock file header
	If                           string
	TimeoutMinutes               string
	CustomSteps                  string
	PreSteps                     string // steps to run at the very start of the agent job, before checkout
	PreAgentSteps                string // steps to run immediately before the agent execution step
	PostSteps                    string // steps to run after AI execution
	RunsOn                       string
	RunsOnSlim                   string // runner override for all framework/generated jobs (activation, safe-outputs, unlock, etc.)
	Environment                  string // environment setting for the main job
	Container                    string // container setting for the main job
	Services                     string // services setting for the main job
	Tools                        map[string]any
	ParsedTools                  *Tools // Structured tools configuration (NEW: parsed from Tools map)
	MarkdownContent              string
	AI                           string        // "claude" or "codex" (for backwards compatibility)
	EngineConfig                 *EngineConfig // Extended engine configuration
	AgentFile                    string        // Path to custom agent file (from imports)
	AgentImportSpec              string        // Original import specification for agent file (e.g., "owner/repo/path@ref")
	RepositoryImports            []string      // Repository-only imports (format: "owner/repo@ref") for .github folder merging
	StopTime                     string
	SkipIfMatch                  *SkipIfMatchConfig              // skip-if-match configuration with query and max threshold
	SkipIfNoMatch                *SkipIfNoMatchConfig            // skip-if-no-match configuration with query and min threshold
	SkipIfCheckFailing           *SkipIfCheckFailingConfig       // skip-if-check-failing configuration
	SkipRoles                    []string                        // roles to skip workflow for (e.g., [admin, maintainer, write])
	SkipBots                     []string                        // users to skip workflow for (e.g., [user1, user2])
	OnSteps                      []map[string]any                // steps to inject into the pre-activation job from on.steps
	OnPermissions                *Permissions                    // additional permissions for the pre-activation job from on.permissions
	OnNeeds                      []string                        // custom workflow jobs that pre_activation/activation should depend on from on.needs
	ManualApproval               string                          // environment name for manual approval from on: section
	Command                      []string                        // for /command trigger support - multiple command names
	CommandEvents                []string                        // events where command should be active (nil = all events)
	CommandOtherEvents           map[string]any                  // for merging command with other events
	LabelCommand                 []string                        // for label-command trigger support - label names that act as commands
	LabelCommandEvents           []string                        // events where label-command should be active (nil = all: issues, pull_request, discussion)
	LabelCommandOtherEvents      map[string]any                  // for merging label-command with other events
	LabelCommandRemoveLabel      bool                            // whether to automatically remove the triggering label (default: true)
	AIReaction                   string                          // AI reaction type like "eyes", "heart", etc.
	ReactionIssues               *bool                           // whether reactions are allowed on issues/issue_comment triggers (default: true)
	ReactionPullRequests         *bool                           // whether reactions are allowed on pull_request/pull_request_review_comment triggers (default: true)
	ReactionDiscussions          *bool                           // whether reactions are allowed on discussion/discussion_comment triggers (default: true)
	StatusComment                *bool                           // whether to post status comments (default: true when ai-reaction is set, false otherwise)
	StatusCommentIssues          *bool                           // whether status comments are allowed on issues/issue_comment triggers (default: true)
	StatusCommentPullRequests    *bool                           // whether status comments are allowed on pull_request/pull_request_review_comment triggers (default: true)
	StatusCommentDiscussions     *bool                           // whether status comments are allowed on discussion/discussion_comment triggers (default: true)
	ActivationGitHubToken        string                          // custom github token from on.github-token for reactions/comments
	ActivationGitHubApp          *GitHubAppConfig                // github app config from on.github-app for minting activation tokens
	TopLevelGitHubApp            *GitHubAppConfig                // top-level github-app fallback for all nested github-app token minting operations
	LockForAgent                 bool                            // whether to lock the issue during agent workflow execution
	Jobs                         map[string]any                  // custom job configurations with dependencies
	Cache                        string                          // cache configuration
	NeedsTextOutput              bool                            // whether the workflow uses ${{ needs.task.outputs.text }}
	NetworkPermissions           *NetworkPermissions             // parsed network permissions
	SandboxConfig                *SandboxConfig                  // parsed sandbox configuration (AWF or SRT)
	SafeOutputs                  *SafeOutputsConfig              // output configuration for automatic output routes
	MCPScripts                   *MCPScriptsConfig               // mcp-scripts configuration for custom MCP tools
	Roles                        []string                        // permission levels required to trigger workflow
	Bots                         []string                        // allow list of bot identifiers that can trigger workflow
	RateLimit                    *RateLimitConfig                // rate limiting configuration for workflow triggers
	CacheMemoryConfig            *CacheMemoryConfig              // parsed cache-memory configuration
	RepoMemoryConfig             *RepoMemoryConfig               // parsed repo-memory configuration
	Runtimes                     map[string]any                  // runtime version overrides from frontmatter
	ToolsTimeout                 string                          // timeout for tool/MCP operations: numeric string (seconds) or GitHub Actions expression (empty = use engine default)
	ToolsStartupTimeout          string                          // timeout for MCP server startup: numeric string (seconds) or GitHub Actions expression (empty = use engine default)
	Features                     map[string]any                  // feature flags and configuration options from frontmatter (supports bool and string values)
	ActionCache                  *ActionCache                    // cache for action pin resolutions
	ActionResolver               *ActionResolver                 // resolver for action pins
	DockerImages                 []string                        // container images collected at compile time (pinned refs when pins are cached)
	DockerImagePins              []GHAWManifestContainer         // full container pin info (image, digest, pinned_image) for manifest
	ActionResolutionFailures     []GHAWManifestResolutionFailure // unresolved action-ref pinning failures for lock manifest auditing
	StrictMode                   bool                            // strict mode for action pinning
	AllowActionRefs              bool                            // if true, unresolved action refs are warnings instead of errors
	SecretMasking                *SecretMaskingConfig            // secret masking configuration
	ParsedFrontmatter            *FrontmatterConfig              // cached parsed frontmatter configuration (for performance optimization)
	RawFrontmatter               map[string]any                  // raw parsed frontmatter map (for passing to hash functions without re-parsing)
	OTLPEndpoint                 string                          // resolved OTLP endpoint (from observability.otlp.endpoint, including imports; set by injectOTLPConfig)
	OTLPHeaders                  string                          // normalized OTLP headers in key=value,key=value format (from observability.otlp.headers, including imports; set by injectOTLPConfig)
	ResolvedMCPServers           map[string]any                  // fully merged mcp-servers from main workflow and all imports (for mcp inspect)
	ActionPinWarnings            map[string]bool                 // cache of already-warned action pin failures (key: "repo@version")
	ActionMode                   ActionMode                      // action mode for workflow compilation (dev, release, script)
	HasExplicitGitHubTool        bool                            // true if tools.github was explicitly configured in frontmatter
	InlinedImports               bool                            // if true, inline all imports at compile time (from inlined-imports frontmatter field)
	CheckoutConfigs              []*CheckoutConfig               // user-configured checkout settings from frontmatter
	CheckoutDisabled             bool                            // true when checkout: false is set in frontmatter
	HasDispatchItemNumber        bool                            // true when workflow_dispatch has item_number input (generated by label trigger shorthand)
	ConcurrencyJobDiscriminator  string                          // optional discriminator expression appended to job-level concurrency groups (from concurrency.job-discriminator)
	IsDetectionRun               bool                            // true when this WorkflowData is used for inline threat detection (not the main agent run)
	UpdateCheckDisabled          bool                            // true when check-for-updates: false is set in frontmatter (disables version check step in activation job)
	StaleCheckDisabled           bool                            // true when on.stale-check: false is set in frontmatter (disables frontmatter hash check step in activation job)
	EngineConfigSteps            []map[string]any                // steps returned by engine.RenderConfig — prepended before execution steps
	ServicePortExpressions       string                          // comma-separated ${{ job.services['<id>'].ports['<port>'] }} expressions for AWF --allow-host-service-ports
	RunInstallScripts            bool                            // true when run-install-scripts: true is set (globally or per node runtime); disables --ignore-scripts on generated npm install steps
	CachedPermissions            *Permissions                    // cached parsed Permissions object (for performance optimization); populated by applyDefaults after all permission mutations
	ConcurrencyGroupExpr         string                          // cached concurrency group expression extracted from Concurrency YAML (for performance optimization); populated by applyDefaults
	CachedAllowedDomainsStr      string                          // cached allowed-domains string for sanitization (for performance optimization); computed once and reused across multiple compilation steps
	CachedAllowedDomainsComputed bool                            // true once CachedAllowedDomainsStr has been set; distinguishes "computed empty" from "not yet computed"
}

// PinContext returns an actionpins.PinContext backed by this WorkflowData.
// It is used to pass the resolver and warnings state to pkg/actionpins functions
// without introducing an import cycle.
func (d *WorkflowData) PinContext() *actionpins.PinContext {
	if d == nil {
		return nil
	}
	if d.ActionPinWarnings == nil {
		d.ActionPinWarnings = make(map[string]bool)
	}
	ctx := &actionpins.PinContext{
		StrictMode:      d.StrictMode,
		EnforcePinned:   true,
		AllowActionRefs: d.AllowActionRefs,
		Warnings:        d.ActionPinWarnings,
		RecordResolutionFailure: func(f actionpins.ResolutionFailure) {
			d.ActionResolutionFailures = append(d.ActionResolutionFailures, GHAWManifestResolutionFailure{
				Repo:      f.Repo,
				Ref:       f.Ref,
				ErrorType: string(f.ErrorType),
			})
		},
	}
	// Only set Resolver if non-nil to avoid passing a typed nil interface value
	// (which would be non-nil in actionpins but crash on method call).
	if d.ActionResolver != nil {
		ctx.Resolver = d.ActionResolver
	}
	return ctx
}

// BaseSafeOutputConfig holds common configuration fields for all safe output types
type BaseSafeOutputConfig struct {
	Max         *string `yaml:"max,omitempty"`          // Maximum number of items to create (supports integer or GitHub Actions expression)
	GitHubToken string  `yaml:"github-token,omitempty"` // GitHub token for this specific output type
	Staged      bool    `yaml:"staged,omitempty"`       // If true, emit step summary messages instead of making GitHub API calls for this specific output type
}

// SafeOutputsConfig holds configuration for automatic output routes
type SafeOutputsConfig struct {
	CreateIssues                    *CreateIssuesConfig                    `yaml:"create-issue,omitempty"`
	CreateDiscussions               *CreateDiscussionsConfig               `yaml:"create-discussion,omitempty"`
	UpdateDiscussions               *UpdateDiscussionsConfig               `yaml:"update-discussion,omitempty"`
	CloseDiscussions                *CloseDiscussionsConfig                `yaml:"close-discussion,omitempty"`
	CloseIssues                     *CloseIssuesConfig                     `yaml:"close-issue,omitempty"`
	ClosePullRequests               *ClosePullRequestsConfig               `yaml:"close-pull-request,omitempty"`
	MarkPullRequestAsReadyForReview *MarkPullRequestAsReadyForReviewConfig `yaml:"mark-pull-request-as-ready-for-review,omitempty"`
	AddComments                     *AddCommentsConfig                     `yaml:"add-comment,omitempty"`
	CommentMemory                   *CommentMemoryConfig                   `yaml:"comment-memory,omitempty"` // Persist and update managed memory comments on issues/PRs
	CreatePullRequests              *CreatePullRequestsConfig              `yaml:"create-pull-request,omitempty"`
	CreatePullRequestReviewComments *CreatePullRequestReviewCommentsConfig `yaml:"create-pull-request-review-comment,omitempty"`
	SubmitPullRequestReview         *SubmitPullRequestReviewConfig         `yaml:"submit-pull-request-review,omitempty"`           // Submit a PR review with status (APPROVE, REQUEST_CHANGES, COMMENT)
	ReplyToPullRequestReviewComment *ReplyToPullRequestReviewCommentConfig `yaml:"reply-to-pull-request-review-comment,omitempty"` // Reply to existing review comments on PRs
	ResolvePullRequestReviewThread  *ResolvePullRequestReviewThreadConfig  `yaml:"resolve-pull-request-review-thread,omitempty"`   // Resolve a review thread on a pull request
	CreateCodeScanningAlerts        *CreateCodeScanningAlertsConfig        `yaml:"create-code-scanning-alerts,omitempty"`
	AutofixCodeScanningAlert        *AutofixCodeScanningAlertConfig        `yaml:"autofix-code-scanning-alert,omitempty"`
	AddLabels                       *AddLabelsConfig                       `yaml:"add-labels,omitempty"`
	RemoveLabels                    *RemoveLabelsConfig                    `yaml:"remove-labels,omitempty"`
	AddReviewer                     *AddReviewerConfig                     `yaml:"add-reviewer,omitempty"`
	AssignMilestone                 *AssignMilestoneConfig                 `yaml:"assign-milestone,omitempty"`
	AssignToAgent                   *AssignToAgentConfig                   `yaml:"assign-to-agent,omitempty"`
	AssignToUser                    *AssignToUserConfig                    `yaml:"assign-to-user,omitempty"`     // Assign users to issues
	UnassignFromUser                *UnassignFromUserConfig                `yaml:"unassign-from-user,omitempty"` // Remove assignees from issues
	UpdateIssues                    *UpdateIssuesConfig                    `yaml:"update-issue,omitempty"`
	UpdatePullRequests              *UpdatePullRequestsConfig              `yaml:"update-pull-request,omitempty"` // Update GitHub pull request title/body
	MergePullRequest                *MergePullRequestConfig                `yaml:"merge-pull-request,omitempty"`  // Merge pull requests under constrained policy checks
	PushToPullRequestBranch         *PushToPullRequestBranchConfig         `yaml:"push-to-pull-request-branch,omitempty"`
	UploadAssets                    *UploadAssetsConfig                    `yaml:"upload-asset,omitempty"`
	UploadArtifact                  *UploadArtifactConfig                  `yaml:"upload-artifact,omitempty"`              // Upload files as run-scoped GitHub Actions artifacts
	UpdateRelease                   *UpdateReleaseConfig                   `yaml:"update-release,omitempty"`               // Update GitHub release descriptions
	CreateAgentSessions             *CreateAgentSessionConfig              `yaml:"create-agent-session,omitempty"`         // Create GitHub Copilot coding agent sessions
	UpdateProjects                  *UpdateProjectConfig                   `yaml:"update-project,omitempty"`               // Smart project board management (create/add/update)
	CreateProjects                  *CreateProjectsConfig                  `yaml:"create-project,omitempty"`               // Create GitHub Projects V2
	CreateProjectStatusUpdates      *CreateProjectStatusUpdateConfig       `yaml:"create-project-status-update,omitempty"` // Create GitHub project status updates
	LinkSubIssue                    *LinkSubIssueConfig                    `yaml:"link-sub-issue,omitempty"`               // Link issues as sub-issues
	HideComment                     *HideCommentConfig                     `yaml:"hide-comment,omitempty"`                 // Hide comments
	SetIssueType                    *SetIssueTypeConfig                    `yaml:"set-issue-type,omitempty"`               // Set the type of an issue (empty string clears the type)
	DispatchWorkflow                *DispatchWorkflowConfig                `yaml:"dispatch-workflow,omitempty"`            // Dispatch workflow_dispatch events to other workflows
	DispatchRepository              *DispatchRepositoryConfig              `yaml:"dispatch_repository,omitempty"`          // Dispatch repository_dispatch events to external repositories
	CallWorkflow                    *CallWorkflowConfig                    `yaml:"call-workflow,omitempty"`                // Call reusable workflows via workflow_call fan-out
	MissingTool                     *MissingToolConfig                     `yaml:"missing-tool,omitempty"`                 // Optional for reporting missing functionality
	MissingData                     *MissingDataConfig                     `yaml:"missing-data,omitempty"`                 // Optional for reporting missing data required to achieve goals
	NoOp                            *NoOpConfig                            `yaml:"noop,omitempty"`                         // No-op output for logging only (always available as fallback)
	ReportIncomplete                *ReportIncompleteConfig                `yaml:"report-incomplete,omitempty"`            // Signal that the task could not be completed due to a tool or infrastructure failure
	ThreatDetection                 *ThreatDetectionConfig                 `yaml:"threat-detection,omitempty"`             // Threat detection configuration
	Jobs                            map[string]*SafeJobConfig              `yaml:"jobs,omitempty"`                         // Safe-jobs configuration (moved from top-level)
	Scripts                         map[string]*SafeScriptConfig           `yaml:"scripts,omitempty"`                      // Custom inline handlers that run in the safe-output handler loop
	GitHubApp                       *GitHubAppConfig                       `yaml:"github-app,omitempty"`                   // GitHub App credentials for token minting
	AllowedDomains                  []string                               `yaml:"allowed-domains,omitempty"`              // Allowed domains for URL redaction, unioned with network.allowed; supports ecosystem identifiers
	AllowGitHubReferences           []string                               `yaml:"allowed-github-references,omitempty"`    // Allowed repositories for GitHub references (e.g., ["repo", "org/repo2"])
	Staged                          bool                                   `yaml:"staged,omitempty"`                       // If true, emit step summary messages instead of making GitHub API calls
	Env                             map[string]string                      `yaml:"env,omitempty"`                          // Environment variables to pass to safe output jobs
	GitHubToken                     string                                 `yaml:"github-token,omitempty"`                 // GitHub token for safe output jobs
	MaximumPatchSize                int                                    `yaml:"max-patch-size,omitempty"`               // Maximum allowed patch size in KB (defaults to 1024)
	MaximumPatchFiles               int                                    `yaml:"max-patch-files,omitempty"`              // Maximum allowed unique files per create-pull-request patch (defaults to 100)
	RunsOn                          string                                 `yaml:"runs-on,omitempty"`                      // Runner configuration for safe-outputs jobs
	Messages                        *SafeOutputMessagesConfig              `yaml:"messages,omitempty"`                     // Custom message templates for footer and notifications
	Mentions                        *MentionsConfig                        `yaml:"mentions,omitempty"`                     // Configuration for @mention filtering in safe outputs
	Footer                          *bool                                  `yaml:"footer,omitempty"`                       // Global footer control - when false, omits visible footer from all safe outputs (XML markers still included)
	GroupReports                    bool                                   `yaml:"group-reports,omitempty"`                // If true, create parent "Failed runs" issue for agent failures (default: false)
	ReportFailureAsIssue            *bool                                  `yaml:"report-failure-as-issue,omitempty"`      // If false, disables creating failure tracking issues when workflows fail (default: true)
	FailureIssueRepo                string                                 `yaml:"failure-issue-repo,omitempty"`           // Repository to create failure issues in (format: "owner/repo"), defaults to current repo
	MaxBotMentions                  *string                                `yaml:"max-bot-mentions,omitempty"`             // Maximum bot trigger references (e.g. 'fixes #123') allowed before filtering. Default: 10. Supports integer or GitHub Actions expression.
	Steps                           []any                                  `yaml:"steps,omitempty"`                        // User-provided steps injected after setup/checkout and before safe-output code
	IDToken                         *string                                `yaml:"id-token,omitempty"`                     // Override id-token permission: "write" to force-add, "none" to disable auto-detection
	ConcurrencyGroup                string                                 `yaml:"concurrency-group,omitempty"`            // Concurrency group for the safe-outputs job (cancel-in-progress is always false)
	Needs                           []string                               `yaml:"needs,omitempty"`                        // Additional custom workflow jobs that safe_outputs should depend on
	Environment                     string                                 `yaml:"environment,omitempty"`                  // Override the GitHub deployment environment for the safe-outputs job (defaults to the top-level environment: field)
	Actions                         map[string]*SafeOutputActionConfig     `yaml:"actions,omitempty"`                      // Custom GitHub Actions mounted as safe output tools (resolved at compile time)
	AutoInjectedCreateIssue         bool                                   `yaml:"-"`                                      // Internal: true when create-issues was automatically injected by the compiler (not user-configured)
}

// SafeOutputMessagesConfig holds custom message templates for safe-output footer and notification messages
type SafeOutputMessagesConfig struct {
	Footer                         string `yaml:"footer,omitempty" json:"footer,omitempty"`                                                    // Custom footer message template
	FooterInstall                  string `yaml:"footer-install,omitempty" json:"footerInstall,omitempty"`                                     // Custom installation instructions template
	FooterWorkflowRecompile        string `yaml:"footer-workflow-recompile,omitempty" json:"footerWorkflowRecompile,omitempty"`                // Custom footer template for workflow recompile issues
	FooterWorkflowRecompileComment string `yaml:"footer-workflow-recompile-comment,omitempty" json:"footerWorkflowRecompileComment,omitempty"` // Custom footer template for comments on workflow recompile issues
	StagedTitle                    string `yaml:"staged-title,omitempty" json:"stagedTitle,omitempty"`                                         // Custom styled mode title template
	StagedDescription              string `yaml:"staged-description,omitempty" json:"stagedDescription,omitempty"`                             // Custom staged mode description template
	AppendOnlyComments             bool   `yaml:"append-only-comments,omitempty" json:"appendOnlyComments,omitempty"`                          // If true, post run status as new comments instead of updating the activation comment
	ActivationComments             string `yaml:"activation-comments,omitempty" json:"activationComments,omitempty"`                           // If "false", disable all activation/fallback comments entirely. Supports templatable boolean values (literal "true"/"false" or GitHub Actions expressions). Empty/unset preserves default enabled behavior.
	RunStarted                     string `yaml:"run-started,omitempty" json:"runStarted,omitempty"`                                           // Custom workflow activation message template
	RunSuccess                     string `yaml:"run-success,omitempty" json:"runSuccess,omitempty"`                                           // Custom workflow success message template
	RunFailure                     string `yaml:"run-failure,omitempty" json:"runFailure,omitempty"`                                           // Custom workflow failure message template
	DetectionFailure               string `yaml:"detection-failure,omitempty" json:"detectionFailure,omitempty"`                               // Custom detection job failure message template
	PullRequestCreated             string `yaml:"pull-request-created,omitempty" json:"pullRequestCreated,omitempty"`                          // Custom message template for pull request creation link. Placeholders: {item_number}, {item_url}
	IssueCreated                   string `yaml:"issue-created,omitempty" json:"issueCreated,omitempty"`                                       // Custom message template for issue creation link. Placeholders: {item_number}, {item_url}
	CommitPushed                   string `yaml:"commit-pushed,omitempty" json:"commitPushed,omitempty"`                                       // Custom message template for commit push link. Placeholders: {commit_sha}, {short_sha}, {commit_url}
	AgentFailureIssue              string `yaml:"agent-failure-issue,omitempty" json:"agentFailureIssue,omitempty"`                            // Custom footer template for agent failure tracking issues
	AgentFailureComment            string `yaml:"agent-failure-comment,omitempty" json:"agentFailureComment,omitempty"`                        // Custom footer template for comments on agent failure tracking issues
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
