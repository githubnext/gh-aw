package workflow

import (
	"context"
	"os"

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
	ctx                     context.Context // Context for network operations (e.g. SHA resolution); defaults to context.Background()
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
	forceStaged             bool                     // If true, force all safe-outputs into staged mode
	trialMode               bool                     // If true, suppress safe outputs for trial mode execution
	trialLogicalRepoSlug    string                   // If set in trial mode, the logical repository to checkout
	useSamples              bool                     // If true, replace the agentic step with a deterministic samples replay driver (hidden feature)
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
	repositorySlugLocked    bool                     // If true, repositorySlug was set via --schedule-seed and must not be overridden by per-file detection
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
	ghesCompatFromCLI       bool                     // If true, GHES compat was requested via --ghes CLI flag (takes precedence over aw.json)
	ghesArtifactCompat      bool                     // If true, emit GHES-compatible v3.x pins for artifact actions instead of the latest v7/v8
	ownerTypeCache          map[string]string        // Cached GitHub owner type ("User"/"Organization"/"") keyed by owner login; not goroutine-safe (Compiler is used sequentially)
	copilotRequestsTipShown map[string]bool          // Tracks markdown paths that already emitted the copilot-requests enable tip in this compiler instance
	// modelPricingResolver is an optional callback for resolving per-token pricing of models that
	// are absent from the embedded models.json catalog. When non-nil it is called during
	// buildInitialWorkflowData for the workflow's configured model; any returned pricing is merged
	// into WorkflowData.ModelCosts so it is embedded in GH_AW_INFO_MODEL_COSTS in the lock.yml.
	// Injected by the cli package (which has access to the embedded catalog and models.dev download).
	modelPricingResolver func(ctx context.Context, provider, model string) (map[string]float64, bool)
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
		ctx:                     context.Background(), // Default context; override with WithContext
		verbose:                 false,
		engineOverride:          "",
		version:                 version,
		skipValidation:          true,                      // Skip validation by default for now since existing workflows don't fully comply
		actionMode:              DetectActionMode(version), // Auto-detect action mode based on version
		jobManager:              NewJobManager(),
		engineRegistry:          GetGlobalEngineRegistry(),
		engineCatalog:           NewEngineCatalog(GetGlobalEngineRegistry()),
		stepOrderTracker:        NewStepOrderTracker(),
		artifactManager:         NewArtifactManager(),
		actionPinWarnings:       make(map[string]bool), // Initialize warning cache
		priorManifests:          make(map[string]*GHAWManifest),
		ownerTypeCache:          make(map[string]string), // Initialize owner-type cache (keyed by owner login)
		copilotRequestsTipShown: make(map[string]bool),   // Initialize one-time tip tracking (keyed by markdown path)
		gitRoot:                 gitRoot,                 // Auto-detected git root
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

// SetContext sets the context used for network operations such as SHA resolution.
func (c *Compiler) SetContext(ctx context.Context) {
	c.ctx = ctx
}

// SetModelPricingResolver registers a callback used to resolve pricing for models that are
// not present in the embedded models.json catalog. The resolver receives the workflow's
// inference provider and model name; it should return per-token pricing (USD) and true when
// pricing is available, or (nil, false) when it is not. Injected by the cli package so that
// the compiler can fetch missing pricing from models.dev without a circular import.
func (c *Compiler) SetModelPricingResolver(fn func(ctx context.Context, provider, model string) (map[string]float64, bool)) {
	c.modelPricingResolver = fn
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

// SetForceStaged configures whether safe-outputs should always compile in staged mode.
func (c *Compiler) SetForceStaged(force bool) {
	c.forceStaged = force
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

// SetUseSamples configures whether to replace the agentic step with a
// deterministic replay driver that feeds `samples` entries to the safe-outputs
// MCP server via real `tools/call` JSON-RPC. Hidden feature used by
// `gh aw compile --use-samples`.
func (c *Compiler) SetUseSamples(use bool) {
	c.useSamples = use
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

// SetGHESCompat enables GHES artifact compatibility mode via the --ghes CLI flag.
// When true, the compiler emits GHES-compatible v3.x artifact action pins
// (upload-artifact@v3, download-artifact@v3) instead of the latest v7/v8.
// This flag takes precedence over the aw.json ghes field.
func (c *Compiler) SetGHESCompat(enabled bool) {
	c.ghesCompatFromCLI = enabled
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

// LockRepositorySlug marks the repository slug as explicitly set (e.g. via --schedule-seed)
// so that per-file git-remote detection cannot override it.
func (c *Compiler) LockRepositorySlug() {
	c.repositorySlugLocked = true
}

// IsRepositorySlugLocked reports whether the repository slug has been locked
// via LockRepositorySlug and must not be overridden by per-file detection.
func (c *Compiler) IsRepositorySlugLocked() bool {
	return c.repositorySlugLocked
}

// SetRepositorySlugIfUnlocked sets the repository slug only when it has not been
// locked via LockRepositorySlug.  This is the method per-file git-remote detection
// should call so that an explicit --schedule-seed flag is never overridden.
func (c *Compiler) SetRepositorySlugIfUnlocked(slug string) {
	if !c.repositorySlugLocked {
		c.SetRepositorySlug(slug)
	}
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
	if manifests == nil {
		manifests = make(map[string]*GHAWManifest)
	}
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

// GetSharedActionResolver returns the shared action resolver used by this compiler instance.
// The resolver is lazily initialized on first access and shared across all workflows.
// It tracks which cache keys were used during compilation, enabling orphaned-entry pruning.
func (c *Compiler) GetSharedActionResolver() *ActionResolver {
	_, resolver := c.getSharedActionResolver()
	return resolver
}

// SkipIfMatchConfig holds the configuration for skip-if-match conditions
