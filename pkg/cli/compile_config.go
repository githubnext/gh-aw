package cli

import (
	"regexp"
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
	Stats                bool     // Display statistics table sorted by file size
}

// WorkflowFailure represents a failed workflow with its error count
type WorkflowFailure struct {
	Path       string // File path of the workflow
	ErrorCount int    // Number of errors in this workflow
}

// CompilationStats tracks the results of workflow compilation
type CompilationStats struct {
	Total           int
	Errors          int
	Warnings        int
	FailedWorkflows []string          // Names of workflows that failed compilation (deprecated, use FailedWorkflowDetails)
	FailureDetails  []WorkflowFailure // Detailed information about failed workflows
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

// Regex patterns for detecting potential secret key names
var (
	// Match uppercase snake_case identifiers that look like secret names (e.g., MY_SECRET_KEY, GITHUB_TOKEN, API_KEY)
	// Excludes common workflow-related keywords
	secretNamePattern = regexp.MustCompile(`\b([A-Z][A-Z0-9]*_[A-Z0-9_]+)\b`)

	// Match PascalCase identifiers ending with security-related suffixes (e.g., GitHubToken, ApiKey, DeploySecret)
	pascalCaseSecretPattern = regexp.MustCompile(`\b([A-Z][a-z0-9]*(?:[A-Z][a-z0-9]*)*(?:Token|Key|Secret|Password|Credential|Auth))\b`)

	// Common non-sensitive workflow keywords to exclude from redaction
	commonWorkflowKeywords = map[string]bool{
		"GITHUB":            true,
		"ACTIONS":           true,
		"WORKFLOW":          true,
		"RUNNER":            true,
		"JOB":               true,
		"STEP":              true,
		"MATRIX":            true,
		"ENV":               true,
		"PATH":              true,
		"HOME":              true,
		"SHELL":             true,
		"INPUTS":            true,
		"OUTPUTS":           true,
		"NEEDS":             true,
		"STRATEGY":          true,
		"CONCURRENCY":       true,
		"IF":                true,
		"WITH":              true,
		"USES":              true,
		"RUN":               true,
		"WORKING_DIRECTORY": true,
		"CONTINUE_ON_ERROR": true,
		"TIMEOUT_MINUTES":   true,
	}
)

// sanitizeErrorMessage removes potential secret key names from error messages to prevent
// information disclosure via logs. This prevents exposing details about an organization's
// security infrastructure by redacting secret key names that might appear in error messages.
func sanitizeErrorMessage(message string) string {
	if message == "" {
		return message
	}

	// Redact uppercase snake_case patterns (e.g., MY_SECRET_KEY, API_TOKEN)
	sanitized := secretNamePattern.ReplaceAllStringFunc(message, func(match string) string {
		// Don't redact common workflow keywords
		if commonWorkflowKeywords[match] {
			return match
		}
		return "[REDACTED]"
	})

	// Redact PascalCase patterns ending with security suffixes (e.g., GitHubToken, ApiKey)
	sanitized = pascalCaseSecretPattern.ReplaceAllString(sanitized, "[REDACTED]")

	return sanitized
}

// sanitizeValidationResults creates a sanitized copy of validation results with all
// error and warning messages sanitized to remove potential secret key names.
// This is applied at the JSON output boundary to ensure no sensitive information
// is leaked regardless of where error messages originated.
func sanitizeValidationResults(results []ValidationResult) []ValidationResult {
	if results == nil {
		return nil
	}

	sanitized := make([]ValidationResult, len(results))
	for i, result := range results {
		sanitized[i] = ValidationResult{
			Workflow:     result.Workflow,
			Valid:        result.Valid,
			CompiledFile: result.CompiledFile,
			Errors:       make([]ValidationError, len(result.Errors)),
			Warnings:     make([]ValidationError, len(result.Warnings)),
		}

		// Sanitize all error messages
		for j, err := range result.Errors {
			sanitized[i].Errors[j] = ValidationError{
				Type:    err.Type,
				Message: sanitizeErrorMessage(err.Message),
				Line:    err.Line,
			}
		}

		// Sanitize all warning messages
		for j, warn := range result.Warnings {
			sanitized[i].Warnings[j] = ValidationError{
				Type:    warn.Type,
				Message: sanitizeErrorMessage(warn.Message),
				Line:    warn.Line,
			}
		}
	}

	return sanitized
}
