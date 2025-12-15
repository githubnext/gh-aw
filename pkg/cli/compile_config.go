package cli

import "regexp"

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

// sanitizeErrorMessage removes sensitive information from error messages before logging
// to prevent exposure of secrets, tokens, and other sensitive data (CWE-312, CWE-315, CWE-359).
//
// This function uses regex patterns to detect and redact:
// - Secret references (secrets.*, secret:, password:, token:, api_key:, credential:)
// - GitHub token patterns (ghp_*, ghs_*, etc.)
// - Potential API keys (long alphanumeric strings in quotes)
//
// The function preserves the structure of error messages for debugging while removing values.
func sanitizeErrorMessage(message string) string {
	// Import regexp at the top of the file if not already present
	// Pattern 1: Match secret references and redact their values
	// Examples: "secret: abc123" -> "secret: [REDACTED]"
	//           "password: xyz789" -> "password: [REDACTED]"
	patterns := []struct {
		regex       string
		replacement string
	}{
		// Match patterns like "secret: value", "password: value", etc.
		{`(?i)(secret|password|token|api_key|credential|key)(\s*[:=]\s*)([^\s,}\]]+)`, `$1$2[REDACTED]`},
		// Match GitHub token patterns (ghp_, ghs_, gho_, ghu_, ghr_)
		{`gh[psouhrt]_[A-Za-z0-9_]+`, `[REDACTED_TOKEN]`},
		// Match quoted strings that look like API keys (long alphanumeric strings)
		{`["']([A-Za-z0-9_\-]{32,})["']`, `"[REDACTED]"`},
		// Match ${{ secrets.* }} patterns
		{`\$\{\{\s*secrets\.[^\}]+\}\}`, `${{ secrets.[REDACTED] }}`},
	}

	result := message
	for _, p := range patterns {
		re := regexp.MustCompile(p.regex)
		result = re.ReplaceAllString(result, p.replacement)
	}

	return result
}
