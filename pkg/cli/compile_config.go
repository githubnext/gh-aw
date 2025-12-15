package cli

import (
	"regexp"
)

// Regex patterns for detecting and redacting sensitive information in error messages
var (
	// Match GitHub secrets syntax: ${{ secrets.NAME }}
	secretsPattern = regexp.MustCompile(`\$\{\{\s*secrets\.[a-zA-Z_][a-zA-Z0-9_]*\s*\}\}`)

	// Match potential secret values: long alphanumeric strings that might be tokens/keys
	tokenPattern = regexp.MustCompile(`\b[a-zA-Z0-9_-]{20,}\b`)

	// Match credential-like key-value pairs in YAML/JSON
	credentialPattern = regexp.MustCompile(`(?i)(password|token|api[_-]?key|secret|credential|auth)["\s]*[:=]["\s]*[^\s"',}]+`)

	// Match GitHub token patterns
	githubTokenPattern = regexp.MustCompile(`\b(ghp|gho|ghu|ghs|ghr)_[a-zA-Z0-9]{36,}\b`)
)

// sanitizeErrorMessage removes or redacts sensitive information from error messages
// before they are added to ValidationError structures that will be logged or output as JSON.
func sanitizeErrorMessage(message string) string {
	if message == "" {
		return message
	}

	// Redact GitHub secrets syntax
	message = secretsPattern.ReplaceAllString(message, "${{ secrets.[REDACTED] }}")

	// Redact GitHub tokens
	message = githubTokenPattern.ReplaceAllString(message, "[REDACTED_TOKEN]")

	// Redact credential key-value pairs
	message = credentialPattern.ReplaceAllStringFunc(message, func(match string) string {
		// Keep the key name but redact the value
		parts := regexp.MustCompile(`[:=]`).Split(match, 2)
		if len(parts) == 2 {
			return parts[0] + ": [REDACTED]"
		}
		return "[REDACTED]"
	})

	// For very long strings that might contain secret dumps, truncate them
	if len(message) > 1000 {
		message = message[:1000] + "... [truncated]"
	}

	return message
}

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
