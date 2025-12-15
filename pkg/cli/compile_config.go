package cli

import (
	"regexp"
	"strings"
)

// CompileConfig holds configuration options for compiling workflows
type CompileConfig struct {
	MarkdownFiles        []string // Files to compile (empty for all files)
	Verbose              bool     // Enable verbose output
	EngineOverride       string   // Override AI engine setting
	Validate             bool     // Enable schema validation
	Watch                bool     // Enable watch mode
	WorkflowDir          string   // Custom workflow directory
	SkipInstructions     bool     // Deprecated: Instructions are no longer written during compilation
	NoEmit               bool     // Validate without generating lock files
	Purge                bool     // Remove orphaned lock files
	TrialMode            bool     // Enable trial mode (suppress safe outputs)
	TrialLogicalRepoSlug string   // Target repository for trial mode
	Strict               bool     // Enable strict mode validation
	Dependabot           bool     // Generate Dependabot manifests for npm dependencies
	ForceOverwrite       bool     // Force overwrite of existing files (dependabot.yml)
	Zizmor               bool     // Run zizmor security scanner on generated .lock.yml files
	Poutine              bool     // Run poutine security scanner on generated .lock.yml files
	Actionlint           bool     // Run actionlint linter on generated .lock.yml files
	JSONOutput           bool     // Output validation results as JSON
	RefreshStopTime      bool     // Force regeneration of stop-after times instead of preserving existing ones
	ActionMode           string   // Action script inlining mode: inline, dev, or release
}

// CompilationStats tracks the results of workflow compilation
type CompilationStats struct {
	Total           int
	Errors          int
	Warnings        int
	FailedWorkflows []string // Names of workflows that failed compilation
}

// ValidationError represents a single validation error or warning
type ValidationError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Line    int    `json:"line,omitempty"`
}

// ValidationResult represents the validation result for a single workflow
type ValidationResult struct {
	Workflow     string            `json:"workflow"`
	Valid        bool              `json:"valid"`
	Errors       []ValidationError `json:"errors"`
	Warnings     []ValidationError `json:"warnings"`
	CompiledFile string            `json:"compiled_file,omitempty"`
}

// sanitizeErrorMessage redacts sensitive information from error messages before logging.
// This prevents secrets, tokens, and other sensitive data from being exposed in logs.
func sanitizeErrorMessage(message string) string {
	// Common patterns for sensitive data - ordered by specificity (most specific first)
	patterns := []struct {
		regex       *regexp.Regexp
		replacement string
	}{
		// GitHub tokens (ghp_, gho_, ghs_, ghu_, ghr_)
		{regexp.MustCompile(`gh[pousr]_[a-zA-Z0-9]{36,}`), "[REDACTED_GITHUB_TOKEN]"},
		// AWS keys
		{regexp.MustCompile(`AKIA[0-9A-Z]{16}`), "[REDACTED_AWS_KEY]"},
		// Private keys
		{regexp.MustCompile(`-----BEGIN [A-Z ]+ PRIVATE KEY-----[\s\S]*?-----END [A-Z ]+ PRIVATE KEY-----`), "[REDACTED_PRIVATE_KEY]"},
		// Password/secret/key patterns in error messages with values
		{regexp.MustCompile(`(?i)(password|passwd|pwd|token|secret|key|apikey|api_key)[\s:=]+["']?([^\s"',}]+)["']?`), "$1=[REDACTED]"},
	}

	sanitized := message
	for _, pattern := range patterns {
		sanitized = pattern.regex.ReplaceAllString(sanitized, pattern.replacement)
	}

	// Additional cleanup: truncate very long error messages that might contain dumps
	if len(sanitized) > 1000 {
		sanitized = sanitized[:1000] + "... [truncated for security]"
	}

	return strings.TrimSpace(sanitized)
}
