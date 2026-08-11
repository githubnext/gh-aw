package workflow

import (
	"context"
)

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

// NewCompiler creates a new workflow compiler with functional options.
// By default, it auto-detects the version and action mode.
// Common options: WithVerbose, WithEngineOverride, WithNoEmit, WithSkipValidation
func NewCompiler(opts ...CompilerOption) *Compiler {
	// Get the current compiler version (set by SetVersion during CLI initialization)
	version := GetVersion()

	// Auto-detect git repository root for action cache path resolution
	// This ensures actions-lock.json is created at repo root regardless of CWD
	gitRoot := findGitRoot()

	engineRegistry := NewEngineRegistry()

	// Create compiler with defaults
	c := &Compiler{
		ctx:                     context.Background(), // Default context; override with WithContext
		verbose:                 false,
		engineOverride:          "",
		version:                 version,
		skipValidation:          true,                      // Skip validation by default for now since existing workflows don't fully comply
		actionMode:              DetectActionMode(version), // Auto-detect action mode based on version
		jobManager:              NewJobManager(),
		engineRegistry:          engineRegistry,
		engineCatalog:           NewEngineCatalog(engineRegistry),
		stepOrderTracker:        NewStepOrderTracker(),
		artifactManager:         NewArtifactManager(),
		actionPinWarnings:       make(map[string]bool), // Initialize warning cache
		priorManifests:          make(map[string]*GHAWManifest),
		ownerTypeCache:          make(map[string]string),        // Initialize owner-type cache (keyed by owner login)
		copilotRequestsTipShown: make(map[string]bool),          // Initialize one-time tip tracking (keyed by markdown path)
		featureUsage:            make(map[string]int),           // Initialize batch feature usage counts
		permissionWarningShown:  make(map[string]string),        // Initialize one-time permission warning tracking (keyed by markdown path)
		allowedDomainsCache:     make(map[string]allowedDomain), // Initialize allowed-domains cache (keyed by markdown path)
		gitRoot:                 gitRoot,                        // Auto-detected git root
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
