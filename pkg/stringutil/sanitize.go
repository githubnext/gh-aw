package stringutil

import (
	"regexp"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var sanitizeLog = logger.New("stringutil:sanitize")

var multipleHyphens = regexp.MustCompile(`-+`)

// Regex patterns for detecting potential secret key names
var (
	// Match uppercase snake_case identifiers that look like secret names (e.g., MY_SECRET_KEY, GITHUB_TOKEN, API_KEY)
	// Excludes common workflow-related keywords
	secretNamePattern = regexp.MustCompile(`\b([A-Z][A-Z0-9]*_[A-Z0-9_]+)\b`)

	// Match PascalCase identifiers ending with security-related suffixes (e.g., GitHubToken, ApiKey, DeploySecret)
	pascalCaseSecretPattern = regexp.MustCompile(`\b([A-Z][a-z0-9]*(?:[A-Z][a-z0-9]*)*(?:Token|Key|Secret|Password|Credential|Auth))\b`)

	// Common non-sensitive workflow keywords to exclude from redaction
	commonWorkflowKeywords = map[string]struct{}{
		"GITHUB":            {},
		"ACTIONS":           {},
		"WORKFLOW":          {},
		"RUNNER":            {},
		"JOB":               {},
		"STEP":              {},
		"MATRIX":            {},
		"ENV":               {},
		"PATH":              {},
		"HOME":              {},
		"SHELL":             {},
		"INPUTS":            {},
		"OUTPUTS":           {},
		"NEEDS":             {},
		"STRATEGY":          {},
		"CONCURRENCY":       {},
		"IF":                {},
		"WITH":              {},
		"USES":              {},
		"RUN":               {},
		"WORKING_DIRECTORY": {},
		"CONTINUE_ON_ERROR": {},
		"TIMEOUT_MINUTES":   {},
	}
)

// SanitizeOptions configures the behavior of the SanitizeName function.
type SanitizeOptions struct {
	// PreserveSpecialChars is a list of special characters to preserve during sanitization.
	// Common characters include '.', '_'. If nil or empty, only alphanumeric and hyphens are preserved.
	PreserveSpecialChars []rune

	// TrimHyphens controls whether leading and trailing hyphens are removed from the result.
	// When true, hyphens at the start and end of the sanitized name are trimmed.
	TrimHyphens bool

	// DefaultValue is returned when the sanitized name is empty after all transformations.
	// If empty string, no default is applied.
	DefaultValue string
}

// SanitizeName sanitizes a string for use as an identifier, file name, or similar context.
// It provides configurable behavior through the SanitizeOptions parameter.
func SanitizeName(name string, opts *SanitizeOptions) string {
	logSanitizeInput(name, opts)

	// Handle nil options
	if opts == nil {
		opts = &SanitizeOptions{}
	}

	result := normalizeSanitizeSeparators(strings.ToLower(name), opts)
	result = applySanitizePattern(result, buildSanitizePreservePattern(opts), len(opts.PreserveSpecialChars) > 0)

	// Consolidate multiple consecutive hyphens into a single hyphen
	result = multipleHyphens.ReplaceAllString(result, "-")

	// Optionally trim leading/trailing hyphens
	if opts.TrimHyphens {
		result = strings.Trim(result, "-")
	}

	// Return default value if result is empty
	if result == "" && opts.DefaultValue != "" {
		sanitizeLog.Printf("Sanitized name is empty, using default: %q", opts.DefaultValue)
		return opts.DefaultValue
	}

	sanitizeLog.Printf("Sanitized name result: %q", result)
	return result
}

// logSanitizeInput logs input parameters when debug logging is enabled.
func logSanitizeInput(name string, opts *SanitizeOptions) {
	if !sanitizeLog.Enabled() {
		return
	}
	preserveCount := 0
	trimHyphens := false
	if opts != nil {
		preserveCount = len(opts.PreserveSpecialChars)
		trimHyphens = opts.TrimHyphens
	}
	sanitizeLog.Printf("Sanitizing name: input=%q, preserve_chars=%d, trim_hyphens=%t",
		name, preserveCount, trimHyphens)
}

// normalizeSanitizeSeparators converts common separators to hyphens and optionally
// converts underscores when they are not in the preserve list.
func normalizeSanitizeSeparators(result string, opts *SanitizeOptions) string {
	result = strings.ReplaceAll(result, ":", "-")
	result = strings.ReplaceAll(result, "\\", "-")
	result = strings.ReplaceAll(result, "/", "-")
	result = strings.ReplaceAll(result, " ", "-")
	if !slices.Contains(opts.PreserveSpecialChars, '_') {
		result = strings.ReplaceAll(result, "_", "-")
	}
	return result
}

// buildSanitizePreservePattern builds a regex character class of allowed characters.
func buildSanitizePreservePattern(opts *SanitizeOptions) string {
	var preserveChars strings.Builder
	preserveChars.WriteString("a-z0-9-") // Always preserve alphanumeric and hyphens
	for _, char := range opts.PreserveSpecialChars {
		switch char {
		case '.', '_':
			preserveChars.WriteRune(char)
		}
	}
	return preserveChars.String()
}

// applySanitizePattern removes or replaces characters not in the allowed set.
// When the caller has requested preservation of special chars, unwanted chars are
// replaced with hyphens; otherwise they are removed entirely.
func applySanitizePattern(result, allowedChars string, preserveSpecialChars bool) string {
	pattern := regexp.MustCompile(`[^` + allowedChars + `]+`)
	if preserveSpecialChars {
		return pattern.ReplaceAllString(result, "-")
	}
	return pattern.ReplaceAllString(result, "")
}

// SanitizeErrorMessage removes potential secret key names from error messages to prevent
// information disclosure via logs. This prevents exposing details about an organization's
// security infrastructure by redacting secret key names that might appear in error messages.
func SanitizeErrorMessage(message string) string {
	if message == "" {
		return message
	}

	sanitizeLog.Printf("Sanitizing error message: length=%d", len(message))

	// Redact uppercase snake_case patterns (e.g., MY_SECRET_KEY, API_TOKEN)
	sanitized := secretNamePattern.ReplaceAllStringFunc(message, func(match string) string {
		// Don't redact common workflow keywords
		if _, ok := commonWorkflowKeywords[match]; ok {
			return match
		}
		// Don't redact gh-aw public configuration variables (e.g., GH_AW_SKIP_NPX_VALIDATION)
		if strings.HasPrefix(match, "GH_AW_") {
			return match
		}
		sanitizeLog.Printf("Redacted snake_case secret pattern: %s", match)
		return "[REDACTED]"
	})

	// Redact PascalCase patterns ending with security suffixes (e.g., GitHubToken, ApiKey)
	sanitized = pascalCaseSecretPattern.ReplaceAllString(sanitized, "[REDACTED]")

	if sanitized != message {
		sanitizeLog.Print("Error message sanitization applied redactions")
	}

	return sanitized
}

// SanitizeIdentifierName sanitizes a name for use as a programming-language identifier
// by replacing disallowed characters with underscores.
//
// Use this function for code identifiers (for example JavaScript and Python variable
// names). It preserves [a-zA-Z0-9_] plus optional extraAllowed runes and prepends
// an underscore if the result would otherwise start with a digit.
//
// This function enforces only character-level sanitization. In particular, it returns
// the empty string unchanged for empty input and does not check language-specific
// constraints such as reserved keywords.
//
// For workflow artifact and user-agent identifiers, use workflow.SanitizeArtifactIdentifier
// instead, which produces hyphen-separated lowercase output.
func SanitizeIdentifierName(name string, extraAllowed func(rune) bool) string {
	result := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			return r
		}
		if extraAllowed != nil && extraAllowed(r) {
			return r
		}
		return '_'
	}, name)

	// Ensure it doesn't start with a number
	if len(result) > 0 && result[0] >= '0' && result[0] <= '9' {
		result = "_" + result
	}

	return result
}

// SanitizeParameterName converts a parameter name to a safe JavaScript identifier
// by replacing non-alphanumeric characters with underscores.
//
// This function ensures that parameter names from workflows can be used safely
// in JavaScript code by:
// 1. Replacing any non-alphanumeric characters (except $ and _) with underscores
// 2. Prepending an underscore if the name starts with a number
//
// Valid characters: a-z, A-Z, 0-9 (not at start), _, $
//
// Examples:
//
//	SanitizeParameterName("my-param")        // returns "my_param"
//	SanitizeParameterName("my.param")        // returns "my_param"
//	SanitizeParameterName("123param")        // returns "_123param"
//	SanitizeParameterName("valid_name")      // returns "valid_name"
//	SanitizeParameterName("$special")        // returns "$special"
func SanitizeParameterName(name string) string {
	return SanitizeIdentifierName(name, func(r rune) bool { return r == '$' })
}

// SanitizePythonVariableName converts a parameter name to a valid Python identifier
// by replacing non-alphanumeric characters with underscores.
//
// This function ensures that parameter names from workflows can be used safely
// in Python code by:
// 1. Replacing any non-alphanumeric characters (except _) with underscores
// 2. Prepending an underscore if the name starts with a number
//
// Valid characters: a-z, A-Z, 0-9 (not at start), _
// Note: Python does not allow $ in identifiers (unlike JavaScript)
//
// Examples:
//
//	SanitizePythonVariableName("my-param")        // returns "my_param"
//	SanitizePythonVariableName("my.param")        // returns "my_param"
//	SanitizePythonVariableName("123param")        // returns "_123param"
//	SanitizePythonVariableName("valid_name")      // returns "valid_name"
func SanitizePythonVariableName(name string) string {
	return SanitizeIdentifierName(name, nil)
}

// SanitizeToolID removes common MCP prefixes and suffixes from tool IDs.
// This cleans up tool identifiers by removing redundant MCP-related naming patterns.
//
// The function:
// 1. Removes "mcp-" prefix
// 2. Removes "-mcp" suffix
// 3. Returns the original ID if the result would be empty
//
// Examples:
//
//	SanitizeToolID("notion-mcp")        // returns "notion"
//	SanitizeToolID("mcp-notion")        // returns "notion"
//	SanitizeToolID("some-mcp-server")   // returns "some-mcp-server" (middle occurrence unchanged)
//	SanitizeToolID("github")            // returns "github" (unchanged)
//	SanitizeToolID("mcp")               // returns "mcp" (prevents empty result)
func SanitizeToolID(toolID string) string {
	cleaned := toolID

	// Remove "mcp-" prefix
	cleaned = strings.TrimPrefix(cleaned, "mcp-")

	// Remove "-mcp" suffix
	cleaned = strings.TrimSuffix(cleaned, "-mcp")

	// If the result is empty, use the original
	if cleaned == "" {
		return toolID
	}

	return cleaned
}

// SanitizeForFilename converts a repository slug (owner/repo) to a filename-safe string.
// Replaces "/" with "-" and any remaining non-alphanumeric characters (except "-", "_", ".")
// with "-". Returns "clone-mode" if the slug is empty.
//
// Examples:
//
//	SanitizeForFilename("owner/repo")     // returns "owner-repo"
//	SanitizeForFilename("my.org/my_repo") // returns "my.org-my_repo"
//	SanitizeForFilename("")               // returns "clone-mode"
func SanitizeForFilename(slug string) string {
	if slug == "" {
		return "clone-mode"
	}
	var sb strings.Builder
	for _, r := range strings.ReplaceAll(slug, "/", "-") {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			sb.WriteRune(r)
		} else {
			sb.WriteRune('-')
		}
	}
	return sb.String()
}
