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

// sensitivePatterns contains patterns that might indicate sensitive information in error messages
var sensitivePatterns = []*regexp.Regexp{
	// Match potential secret/password/token patterns
	regexp.MustCompile(`(?i)(secret|password|token|key|credential|api[_-]?key)s?\s*[:=]\s*['"]?[^\s'"]+['"]?`),
	// Match GitHub token patterns
	regexp.MustCompile(`gh[ps]_[a-zA-Z0-9]{36,}`),
	// Match GitHub Actions secret expressions
	regexp.MustCompile(`\$\{\{\s*secrets\.[^}]+\}\}`),
	// Match quoted strings of 40+ chars preceded by secret-like keywords
	regexp.MustCompile(`(?i)(secret|password|token|key|credential|api[_-]?key)s?\s*[:=]\s*['\"][a-zA-Z0-9_\-]{40,}['\"]`),
}

// sanitizeErrorMessage removes potentially sensitive information from error messages
// before they are logged or returned in JSON output.
func sanitizeErrorMessage(msg string) string {
	sanitized := msg
	for _, pattern := range sensitivePatterns {
		// Replace sensitive patterns with a redacted placeholder
		sanitized = pattern.ReplaceAllStringFunc(sanitized, func(match string) string {
			// Try to preserve the structure of the message while redacting the value
			if strings.Contains(match, ":") || strings.Contains(match, "=") {
				idx := strings.IndexAny(match, ":=")
				if idx != -1 && idx+1 < len(match) {
					return match[:idx+1] + " [REDACTED]"
				}
			}
			return "[REDACTED]"
		})
	}
	return sanitized
}
