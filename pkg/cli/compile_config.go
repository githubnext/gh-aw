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

// sanitizeErrorMessage removes sensitive information from error messages before logging
// This prevents accidental exposure of secrets, tokens, and other sensitive data in logs
func sanitizeErrorMessage(message string) string {
	if message == "" {
		return message
	}

	// Pattern to match secret value assignments in YAML format
	// Matches patterns like: "SECRET_NAME: ${{ secrets.VALUE }}" or "key: sensitive_value"
	secretPatterns := []*regexp.Regexp{
		// Match GitHub secrets syntax: ${{ secrets.NAME }}
		regexp.MustCompile(`\$\{\{\s*secrets\.[^}]+\}\}`),
		// Match potential secret values after "secrets:" in YAML
		regexp.MustCompile(`(?mi)(secrets:\s*\n\s+\w+:\s*)[^\n]+`),
		// Match potential password/token fields
		regexp.MustCompile(`(?mi)(password|token|api_key|secret|credential):\s*[^\s\n]+`),
	}

	sanitized := message
	for _, pattern := range secretPatterns {
		sanitized = pattern.ReplaceAllString(sanitized, "[REDACTED]")
	}

	// Additional sanitization: if the message contains "secrets:" followed by data structures,
	// replace the entire secrets block
	if strings.Contains(strings.ToLower(sanitized), "secrets:") {
		lines := strings.Split(sanitized, "\n")
		var result []string
		inSecretsBlock := false

		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "secrets:") {
				result = append(result, strings.Split(line, "secrets:")[0]+"secrets: [REDACTED]")
				inSecretsBlock = true
				continue
			}

			// If we're in a secrets block and the line is indented, skip it
			if inSecretsBlock {
				// Check if this line is still part of the secrets block (indented)
				if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
					continue
				} else {
					inSecretsBlock = false
				}
			}

			result = append(result, line)
		}
		sanitized = strings.Join(result, "\n")
	}

	return sanitized
}
