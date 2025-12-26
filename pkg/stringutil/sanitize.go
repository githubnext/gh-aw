package stringutil

import (
	"regexp"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var sanitizeLog = logger.New("stringutil:sanitize")

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
		if commonWorkflowKeywords[match] {
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
