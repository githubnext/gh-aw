package cli

// CompileConfig holds configuration options for compiling workflows
type CompileConfig struct {
	MarkdownFiles          []string // Files to compile (empty for all files)
	Verbose                bool     // Enable verbose output
	EngineOverride         string   // Override AI engine setting
	Validate               bool     // Enable schema validation
	Watch                  bool     // Enable watch mode
	WorkflowDir            string   // Custom workflow directory
	SkipInstructions       bool     // Deprecated: Instructions are no longer written during compilation
	NoEmit                 bool     // Validate without generating lock files
	Purge                  bool     // Remove orphaned lock files
	TrialMode              bool     // Enable trial mode (suppress safe outputs)
	TrialLogicalRepoSlug   string   // Target repository for trial mode
	Strict                 bool     // Enable strict mode validation
	Dependabot             bool     // Generate Dependabot manifests for npm dependencies
	ForceOverwrite         bool     // Force overwrite of existing files (dependabot.yml)
	RefreshStopTime        bool     // Force regeneration of stop-after times instead of preserving existing ones
	ForceRefreshActionPins bool     // Force refresh of action pins by clearing cache and resolving from GitHub API
	Zizmor                 bool     // Run zizmor security scanner on generated .lock.yml files
	Poutine                bool     // Run poutine security scanner on generated .lock.yml files
	Actionlint             bool     // Run actionlint linter on generated .lock.yml files
	JSONOutput             bool     // Output validation results as JSON
	ActionMode             string   // Action script inlining mode: inline, dev, or release
	ActionTag              string   // Override action SHA or tag for actions/setup (overrides action-mode to release)
	ActionsRepo            string   // Override the external actions repository (default: github/gh-aw-actions)
	Stats                  bool     // Display statistics table sorted by file size
	FailFast               bool     // Stop at first error instead of collecting all errors
}

// WorkflowFailure represents a failed workflow with its error count
type WorkflowFailure struct {
	Path          string   // File path of the workflow
	ErrorCount    int      // Number of errors in this workflow
	ErrorMessages []string // Actual error messages to display to the user
}

// CompilationStats tracks the results of workflow compilation
type CompilationStats struct {
	Total           int
	Errors          int
	Warnings        int
	FailedWorkflows []string          // Names of workflows that failed compilation (deprecated, use FailedWorkflowDetails)
	FailureDetails  []WorkflowFailure // Detailed information about failed workflows
}

// CompileValidationError represents a single validation error or warning
type CompileValidationError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Line    int    `json:"line,omitempty"`
}

// ValidationResult represents the validation result for a single workflow
type ValidationResult struct {
	Workflow     string                   `json:"workflow"`
	Valid        bool                     `json:"valid"`
	Errors       []CompileValidationError `json:"errors"`
	Warnings     []CompileValidationError `json:"warnings"`
	CompiledFile string                   `json:"compiled_file,omitempty"`
	Labels       []string                 `json:"labels,omitempty"` // Labels referenced in safe-outputs configurations
}
