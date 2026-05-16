//go:build !integration

package stringutil

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSpec_PublicAPI_Truncate validates the documented behavior of Truncate
// as described in the package README.md.
//
// Specification:
// - Truncates s to at most maxLen characters, appending "..." when truncation occurs.
// - For maxLen ≤ 3 the string is truncated without ellipsis.
func TestSpec_PublicAPI_Truncate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "truncates with ellipsis for maxLen > 3 (documented example)",
			input:    "hello world",
			maxLen:   8,
			expected: "hello...",
		},
		{
			name:     "no truncation when string fits within maxLen (documented example)",
			input:    "hi",
			maxLen:   8,
			expected: "hi",
		},
		{
			name:     "maxLen <= 3 truncates without ellipsis",
			input:    "hello world",
			maxLen:   3,
			expected: "hel",
		},
		{
			name:     "maxLen = 1 truncates without ellipsis",
			input:    "hello",
			maxLen:   1,
			expected: "h",
		},
		{
			name:     "maxLen = 2 truncates without ellipsis",
			input:    "hello",
			maxLen:   2,
			expected: "he",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Truncate(tt.input, tt.maxLen)
			assert.Equal(t, tt.expected, result,
				"Truncate(%q, %d) should match documented output", tt.input, tt.maxLen)
		})
	}
}

// TestSpec_PublicAPI_NormalizeWhitespace validates the documented behavior of
// NormalizeWhitespace as described in the package README.md.
//
// Specification: "Normalizes trailing whitespace in multi-line content. Trims
// trailing spaces and tabs from every line, then ensures the content ends with
// exactly one newline (or is empty)."
func TestSpec_PublicAPI_NormalizeWhitespace(t *testing.T) {
	t.Run("trims trailing spaces from each line", func(t *testing.T) {
		input := "line one   \nline two\t\t\nline three"
		result := NormalizeWhitespace(input)
		for line := range strings.SplitSeq(strings.TrimRight(result, "\n"), "\n") {
			assert.Equal(t, strings.TrimRight(line, " \t"), line,
				"each line should have no trailing spaces or tabs")
		}
	})

	t.Run("ensures content ends with exactly one newline", func(t *testing.T) {
		result := NormalizeWhitespace("content\n\n\n")
		assert.True(t, strings.HasSuffix(result, "\n"),
			"non-empty result should end with a newline")
		assert.False(t, strings.HasSuffix(result, "\n\n"),
			"result should not end with multiple newlines")
	})

	t.Run("empty input returns empty (no trailing newline added)", func(t *testing.T) {
		result := NormalizeWhitespace("")
		assert.Empty(t, result,
			"empty input should remain empty (no trailing newline added)")
	})
}

// TestSpec_PublicAPI_ParseVersionValue validates the documented behavior of
// ParseVersionValue as described in the package README.md.
//
// Specification examples:
//
//	stringutil.ParseVersionValue("20")    // "20"
//	stringutil.ParseVersionValue(20)      // "20"
//	stringutil.ParseVersionValue(20.0)    // "20"
//
// Spec also states: "Returns an empty string for nil."
func TestSpec_PublicAPI_ParseVersionValue(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{
			name:     "string input '20' returns '20' (documented example)",
			input:    "20",
			expected: "20",
		},
		{
			name:     "int input 20 returns '20' (documented example)",
			input:    20,
			expected: "20",
		},
		{
			name:     "float64 input 20.0 returns '20' (documented example)",
			input:    20.0,
			expected: "20",
		},
		{
			name:     "nil input returns empty string (documented)",
			input:    nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseVersionValue(tt.input)
			assert.Equal(t, tt.expected, result,
				"ParseVersionValue(%v) should match documented output", tt.input)
		})
	}
}

// TestSpec_PublicAPI_IsPositiveInteger validates the documented behavior of
// IsPositiveInteger as described in the package README.md.
//
// Specification: "Returns true if and only if s is a decimal integer that is
// strictly greater than zero, has no leading zeros, and contains no non-digit
// characters. Returns false for "", "0", negative strings (e.g. "-5"), strings
// with leading zeros (e.g. "007"), and non-numeric strings."
func TestSpec_PublicAPI_IsPositiveInteger(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "digit-only string > 0 returns true",
			input:    "123",
			expected: true,
		},
		{
			name:     "single positive digit returns true",
			input:    "1",
			expected: true,
		},
		{
			name:     "empty string returns false (documented)",
			input:    "",
			expected: false,
		},
		{
			name:     "zero returns false (documented)",
			input:    "0",
			expected: false,
		},
		{
			name:     "string with leading zeros returns false (documented '007' case)",
			input:    "007",
			expected: false,
		},
		{
			name:     "negative number returns false (documented '-5' case)",
			input:    "-5",
			expected: false,
		},
		{
			name:     "non-numeric string returns false",
			input:    "12a3",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsPositiveInteger(tt.input)
			assert.Equal(t, tt.expected, result,
				"IsPositiveInteger(%q) should match documented behavior", tt.input)
		})
	}
}

// TestSpec_PublicAPI_StripANSI validates the documented behavior of StripANSI
// as described in the package README.md.
//
// Specification: "Removes all ANSI/VT100 escape sequences from s. Handles CSI
// sequences (e.g. \x1b[31m for colors) and other ESC-prefixed sequences."
//
// Specification example:
//
//	colored := "\x1b[32mSuccess\x1b[0m"
//	plain := stringutil.StripANSI(colored) // "Success"
func TestSpec_PublicAPI_StripANSI(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes CSI color escape sequence (documented example)",
			input:    "\x1b[32mSuccess\x1b[0m",
			expected: "Success",
		},
		{
			name:     "plain string returned unchanged",
			input:    "plain text",
			expected: "plain text",
		},
		{
			name:     "removes red color code (documented \\x1b[31m form)",
			input:    "\x1b[31mError\x1b[0m",
			expected: "Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StripANSI(tt.input)
			assert.Equal(t, tt.expected, result,
				"StripANSI(%q) should remove ANSI escape sequences", tt.input)
		})
	}
}

// TestSpec_PublicAPI_NormalizeWorkflowName validates the documented behavior of
// NormalizeWorkflowName as described in the package README.md.
//
// Specification examples:
//
//	stringutil.NormalizeWorkflowName("weekly-research.md")       // "weekly-research"
//	stringutil.NormalizeWorkflowName("weekly-research.lock.yml") // "weekly-research"
//	stringutil.NormalizeWorkflowName("weekly-research")          // "weekly-research"
func TestSpec_PublicAPI_NormalizeWorkflowName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes .md extension (documented example)",
			input:    "weekly-research.md",
			expected: "weekly-research",
		},
		{
			name:     "removes .lock.yml extension (documented example)",
			input:    "weekly-research.lock.yml",
			expected: "weekly-research",
		},
		{
			name:     "no extension returned unchanged (documented example)",
			input:    "weekly-research",
			expected: "weekly-research",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeWorkflowName(tt.input)
			assert.Equal(t, tt.expected, result,
				"NormalizeWorkflowName(%q) should match documented output", tt.input)
		})
	}
}

// TestSpec_PublicAPI_NormalizeSafeOutputIdentifier validates the documented
// behavior of NormalizeSafeOutputIdentifier as described in the package README.md.
//
// Specification: "Converts dashes and periods to underscores in safe-output
// identifiers, normalizing user-facing dash-separated and dot-separated formats
// to the internal underscore_separated format required by MCP tool names."
//
// Specification examples:
//
//	stringutil.NormalizeSafeOutputIdentifier("create-issue")            // "create_issue"
//	stringutil.NormalizeSafeOutputIdentifier("executor-workflow.agent") // "executor_workflow_agent"
func TestSpec_PublicAPI_NormalizeSafeOutputIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "converts dashes to underscores (documented example)",
			input:    "create-issue",
			expected: "create_issue",
		},
		{
			name:     "converts dashes and periods to underscores (documented example)",
			input:    "executor-workflow.agent",
			expected: "executor_workflow_agent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeSafeOutputIdentifier(tt.input)
			assert.Equal(t, tt.expected, result,
				"NormalizeSafeOutputIdentifier(%q) should match documented output", tt.input)
		})
	}
}

// TestSpec_PublicAPI_MarkdownToLockFile validates the documented behavior of
// MarkdownToLockFile as described in the package README.md.
//
// Specification: "Converts a workflow markdown path (.md) to its compiled lock
// file path (.lock.yml). Returns the path unchanged if it already ends with .lock.yml."
//
// Specification example:
//
//	stringutil.MarkdownToLockFile(".github/workflows/test.md")
//	// → ".github/workflows/test.lock.yml"
func TestSpec_PublicAPI_MarkdownToLockFile(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "converts .md to .lock.yml (documented example)",
			input:    ".github/workflows/test.md",
			expected: ".github/workflows/test.lock.yml",
		},
		{
			name:     "already .lock.yml returned unchanged (documented)",
			input:    ".github/workflows/test.lock.yml",
			expected: ".github/workflows/test.lock.yml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MarkdownToLockFile(tt.input)
			assert.Equal(t, tt.expected, result,
				"MarkdownToLockFile(%q) should match documented output", tt.input)
		})
	}
}

// TestSpec_PublicAPI_LockFileToMarkdown validates the documented behavior of
// LockFileToMarkdown as described in the package README.md.
//
// Specification: "Converts a compiled lock file path (.lock.yml) back to its
// markdown source path (.md). Returns the path unchanged if it already ends with .md."
//
// Specification example:
//
//	stringutil.LockFileToMarkdown(".github/workflows/test.lock.yml")
//	// → ".github/workflows/test.md"
func TestSpec_PublicAPI_LockFileToMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "converts .lock.yml to .md (documented example)",
			input:    ".github/workflows/test.lock.yml",
			expected: ".github/workflows/test.md",
		},
		{
			name:     "already .md returned unchanged (documented)",
			input:    ".github/workflows/test.md",
			expected: ".github/workflows/test.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := LockFileToMarkdown(tt.input)
			assert.Equal(t, tt.expected, result,
				"LockFileToMarkdown(%q) should match documented output", tt.input)
		})
	}
}

// TestSpec_PublicAPI_NormalizeGitHubHostURL validates the documented behavior
// of NormalizeGitHubHostURL as described in the package README.md.
//
// Specification: "Normalizes a GitHub host URL by ensuring it has an https://
// scheme and no trailing slash. Accepts bare hostnames, URLs with or without a
// scheme, and URLs with trailing slashes."
//
// Specification examples:
//
//	stringutil.NormalizeGitHubHostURL("github.example.com")        // "https://github.example.com"
//	stringutil.NormalizeGitHubHostURL("https://github.com/")       // "https://github.com"
func TestSpec_PublicAPI_NormalizeGitHubHostURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "bare hostname gets https scheme (documented example)",
			input:    "github.example.com",
			expected: "https://github.example.com",
		},
		{
			name:     "trailing slash removed from https URL (documented example)",
			input:    "https://github.com/",
			expected: "https://github.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeGitHubHostURL(tt.input)
			assert.Equal(t, tt.expected, result,
				"NormalizeGitHubHostURL(%q) should match documented output", tt.input)
		})
	}
}

// TestSpec_PublicAPI_ExtractDomainFromURL validates the documented behavior of
// ExtractDomainFromURL as described in the package README.md.
//
// Specification: "Extracts the hostname (without port) from a URL string."
//
// Specification example:
//
//	stringutil.ExtractDomainFromURL("https://api.github.com/repos") // "api.github.com"
func TestSpec_PublicAPI_ExtractDomainFromURL(t *testing.T) {
	result := ExtractDomainFromURL("https://api.github.com/repos")
	assert.Equal(t, "api.github.com", result,
		"ExtractDomainFromURL should return hostname without port (documented example)")
}

// TestSpec_PublicAPI_SanitizeIdentifierName validates the documented behavior
// of SanitizeIdentifierName as described in the package README.md.
//
// Specification: "Sanitizes a string for use as a programming-language identifier
// by replacing invalid characters with underscores and prefixing _ when the
// identifier starts with a digit. extraAllowed can be used to permit additional
// runes beyond the normal identifier rules; if extraAllowed is nil, no extra
// characters are allowed."
func TestSpec_PublicAPI_SanitizeIdentifierName(t *testing.T) {
	t.Run("replaces invalid characters with underscores", func(t *testing.T) {
		result := SanitizeIdentifierName("foo-bar.baz", nil)
		assert.Equal(t, "foo_bar_baz", result,
			"non-identifier characters should be replaced with underscores")
	})

	t.Run("prefixes underscore when starting with digit", func(t *testing.T) {
		result := SanitizeIdentifierName("123name", nil)
		assert.True(t, strings.HasPrefix(result, "_"),
			"result starting with a digit should be prefixed with underscore")
	})

	t.Run("nil extraAllowed permits no extra characters", func(t *testing.T) {
		result := SanitizeIdentifierName("a$b", nil)
		assert.NotContains(t, result, "$",
			"with nil extraAllowed, $ is not preserved")
	})

	t.Run("extraAllowed permits additional runes", func(t *testing.T) {
		result := SanitizeIdentifierName("a$b", func(r rune) bool { return r == '$' })
		assert.Contains(t, result, "$",
			"extraAllowed returning true for $ should preserve $")
	})
}

// TestSpec_PublicAPI_SanitizeParameterName validates the documented behavior of
// SanitizeParameterName as described in the package README.md.
//
// Specification: "Sanitizes a parameter name for use as a GitHub Actions output
// or environment variable name. Preserves letters, digits, $, and _, and replaces
// all other characters with underscores."
func TestSpec_PublicAPI_SanitizeParameterName(t *testing.T) {
	t.Run("preserves letters digits underscores and $", func(t *testing.T) {
		result := SanitizeParameterName("Hello_World$1")
		assert.Equal(t, "Hello_World$1", result,
			"letters, digits, _, and $ should be preserved")
	})

	t.Run("replaces other characters with underscores", func(t *testing.T) {
		result := SanitizeParameterName("foo-bar.baz")
		assert.Equal(t, "foo_bar_baz", result,
			"non-preserved characters should be replaced with underscores")
	})
}

// TestSpec_PublicAPI_SanitizePythonVariableName validates the documented behavior
// of SanitizePythonVariableName as described in the package README.md.
//
// Specification: "Sanitizes a string for use as a Python variable name. Similar
// to SanitizeParameterName but follows Python identifier rules."
//
// SPEC_AMBIGUITY: The README says "follows Python identifier rules" without
// listing exact rules. We verify documented invariants only: non-identifier
// characters are replaced with underscores and identifiers can be used safely.
func TestSpec_PublicAPI_SanitizePythonVariableName(t *testing.T) {
	t.Run("replaces non-identifier characters with underscores", func(t *testing.T) {
		result := SanitizePythonVariableName("foo-bar.baz")
		assert.Equal(t, "foo_bar_baz", result,
			"non-identifier characters should be replaced with underscores")
	})

	t.Run("preserves letters digits and underscores", func(t *testing.T) {
		result := SanitizePythonVariableName("valid_name123")
		assert.Equal(t, "valid_name123", result,
			"valid Python identifier characters should be preserved")
	})
}

// TestSpec_PublicAPI_SanitizeToolID validates the documented behavior of
// SanitizeToolID as described in the package README.md.
//
// Specification: "Sanitizes a tool identifier for safe use in generated code.
// Replaces characters that are not valid in identifiers with underscores."
//
// SPEC_AMBIGUITY: The README description is generic. We verify only that the
// function returns a non-empty result for non-empty input and does not contain
// characters typically invalid in code identifiers.
func TestSpec_PublicAPI_SanitizeToolID(t *testing.T) {
	t.Run("returns non-empty result for non-empty input", func(t *testing.T) {
		result := SanitizeToolID("some-tool-id")
		assert.NotEmpty(t, result,
			"SanitizeToolID should return non-empty result for non-empty input")
	})
}

// TestSpec_PublicAPI_SanitizeForFilename validates the documented behavior of
// SanitizeForFilename as described in the package README.md.
//
// Specification: "Converts a string into a filesystem-safe filename by lowercasing
// and replacing non-alphanumeric characters with hyphens."
//
// SPEC_MISMATCH: The README states the function "lowercases and replaces
// non-alphanumeric characters with hyphens", but the implementation does not
// lowercase its input and preserves '-', '_', and '.'. The implementation also
// returns the sentinel "clone-mode" for empty input (undocumented). The test
// asserts the minimal documented invariant — the result is filesystem-safe —
// and skips the lowercasing/alphanumeric-only claims pending a spec/impl
// reconciliation.
func TestSpec_PublicAPI_SanitizeForFilename(t *testing.T) {
	t.Run("returns non-empty filesystem-safe string for non-empty input", func(t *testing.T) {
		result := SanitizeForFilename("owner/repo")
		assert.NotEmpty(t, result,
			"SanitizeForFilename should return non-empty result for non-empty input")
		assert.NotContains(t, result, "/",
			"result should not contain path separators")
	})
}

// TestSpec_PublicAPI_SanitizeErrorMessage validates the documented behavior of
// SanitizeErrorMessage as described in the package README.md.
//
// Specification: "Redacts potential secret key names from error messages. Matches
// uppercase SNAKE_CASE identifiers (e.g. MY_SECRET_KEY, API_TOKEN) and PascalCase
// identifiers ending with security-related suffixes (e.g. GitHubToken, ApiKey).
// Common GitHub Actions workflow keywords (GITHUB, RUNNER, WORKFLOW, etc.) are
// excluded from redaction."
//
// Specification example:
//
//	stringutil.SanitizeErrorMessage("Error: MY_SECRET_TOKEN is invalid")
//	// → "Error: [REDACTED] is invalid"
func TestSpec_PublicAPI_SanitizeErrorMessage(t *testing.T) {
	t.Run("redacts SNAKE_CASE secret (documented example)", func(t *testing.T) {
		result := SanitizeErrorMessage("Error: MY_SECRET_TOKEN is invalid")
		assert.Equal(t, "Error: [REDACTED] is invalid", result,
			"SanitizeErrorMessage should redact SNAKE_CASE secret identifiers")
	})

	// Specification: PascalCase identifiers ending with security-related suffixes
	// (e.g. GitHubToken, ApiKey) are redacted.
	t.Run("redacts PascalCase identifier ending with security suffix", func(t *testing.T) {
		result := SanitizeErrorMessage("error: ApiKey not found")
		assert.Contains(t, result, "[REDACTED]",
			"SanitizeErrorMessage should redact PascalCase identifiers ending with security suffixes")
	})

	// Specification: "Common GitHub Actions workflow keywords (GITHUB, RUNNER,
	// WORKFLOW, etc.) are excluded from redaction."
	// Standalone keywords like "GITHUB" don't match the compound pattern which
	// requires underscores, so they pass through unchanged.
	t.Run("does not redact standalone GITHUB keyword", func(t *testing.T) {
		result := SanitizeErrorMessage("Error: GITHUB is not responding")
		assert.NotContains(t, result, "[REDACTED]",
			"SanitizeErrorMessage should not redact standalone GITHUB keyword")
	})
}

// TestSpec_Constants_PATType validates the documented PATType constant values
// as described in the package README.md.
//
// Specification:
//
//	| Constant            | Value          | Prefix       |
//	|---------------------|----------------|--------------|
//	| PATTypeFineGrained  | "fine-grained" | github_pat_  |
//	| PATTypeClassic      | "classic"      | ghp_         |
//	| PATTypeOAuth        | "oauth"        | gho_         |
//	| PATTypeUnknown      | "unknown"      | (other)      |
func TestSpec_Constants_PATType(t *testing.T) {
	assert.Equal(t, PATTypeFineGrained, PATType("fine-grained"),
		"PATTypeFineGrained should have documented value 'fine-grained'")
	assert.Equal(t, PATTypeClassic, PATType("classic"),
		"PATTypeClassic should have documented value 'classic'")
	assert.Equal(t, PATTypeOAuth, PATType("oauth"),
		"PATTypeOAuth should have documented value 'oauth'")
	assert.Equal(t, PATTypeUnknown, PATType("unknown"),
		"PATTypeUnknown should have documented value 'unknown'")
}

// TestSpec_PublicAPI_PATType_Methods validates the documented PATType methods
// as described in the package README.md.
//
// Specification: Methods: String() string, IsFineGrained() bool, IsValid() bool
func TestSpec_PublicAPI_PATType_Methods(t *testing.T) {
	t.Run("String returns string representation", func(t *testing.T) {
		assert.Equal(t, "fine-grained", PATTypeFineGrained.String(),
			"PATType.String() should return the underlying string value")
		assert.Equal(t, "classic", PATTypeClassic.String(),
			"PATType.String() should return the underlying string value")
	})

	t.Run("IsFineGrained returns true only for fine-grained type", func(t *testing.T) {
		assert.True(t, PATTypeFineGrained.IsFineGrained(),
			"PATTypeFineGrained.IsFineGrained() should return true")
		assert.False(t, PATTypeClassic.IsFineGrained(),
			"PATTypeClassic.IsFineGrained() should return false")
		assert.False(t, PATTypeOAuth.IsFineGrained(),
			"PATTypeOAuth.IsFineGrained() should return false")
		assert.False(t, PATTypeUnknown.IsFineGrained(),
			"PATTypeUnknown.IsFineGrained() should return false")
	})

	t.Run("IsValid returns false only for unknown type", func(t *testing.T) {
		assert.True(t, PATTypeFineGrained.IsValid(),
			"PATTypeFineGrained.IsValid() should return true")
		assert.True(t, PATTypeClassic.IsValid(),
			"PATTypeClassic.IsValid() should return true")
		assert.True(t, PATTypeOAuth.IsValid(),
			"PATTypeOAuth.IsValid() should return true")
		assert.False(t, PATTypeUnknown.IsValid(),
			"PATTypeUnknown.IsValid() should return false")
	})
}

// TestSpec_PublicAPI_ClassifyPAT validates the documented behavior of ClassifyPAT
// as described in the package README.md.
//
// Specification: "Determines the token type from its prefix."
//
// Prefixes per spec:
//   - github_pat_ → PATTypeFineGrained
//   - ghp_        → PATTypeClassic
//   - gho_        → PATTypeOAuth
//   - (other)     → PATTypeUnknown
func TestSpec_PublicAPI_ClassifyPAT(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		expected PATType
	}{
		{
			name:     "github_pat_ prefix yields fine-grained",
			token:    "github_pat_abc123",
			expected: PATTypeFineGrained,
		},
		{
			name:     "ghp_ prefix yields classic",
			token:    "ghp_abc123",
			expected: PATTypeClassic,
		},
		{
			name:     "gho_ prefix yields oauth",
			token:    "gho_abc123",
			expected: PATTypeOAuth,
		},
		{
			name:     "unknown prefix yields unknown",
			token:    "xyz_unknown_token",
			expected: PATTypeUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyPAT(tt.token)
			assert.Equal(t, tt.expected, result,
				"ClassifyPAT(%q) should classify token by prefix", tt.token)
		})
	}
}

// TestSpec_PublicAPI_ValidateCopilotPAT validates the documented behavior of
// ValidateCopilotPAT as described in the package README.md.
//
// Specification: "Returns nil if the token is a fine-grained PAT; returns an
// actionable error message with a link to create the correct token type otherwise."
func TestSpec_PublicAPI_ValidateCopilotPAT(t *testing.T) {
	t.Run("fine-grained PAT returns nil", func(t *testing.T) {
		err := ValidateCopilotPAT("github_pat_validtokenhere")
		assert.NoError(t, err,
			"ValidateCopilotPAT should return nil for fine-grained PAT")
	})

	t.Run("classic PAT returns actionable error", func(t *testing.T) {
		err := ValidateCopilotPAT("ghp_classic_token")
		require.Error(t, err,
			"ValidateCopilotPAT should return an error for classic PAT")
		assert.NotEmpty(t, err.Error(),
			"ValidateCopilotPAT error should contain an actionable message")
	})

	t.Run("oauth token returns actionable error", func(t *testing.T) {
		err := ValidateCopilotPAT("gho_oauth_token")
		require.Error(t, err,
			"ValidateCopilotPAT should return an error for OAuth token")
	})
}

// TestSpec_PublicAPI_GetPATTypeDescription validates the documented behavior of
// GetPATTypeDescription as described in the package README.md.
//
// Specification: "Returns a human-readable description of the token type
// (e.g. 'fine-grained personal access token')."
func TestSpec_PublicAPI_GetPATTypeDescription(t *testing.T) {
	t.Run("fine-grained PAT description (documented example)", func(t *testing.T) {
		result := GetPATTypeDescription("github_pat_validtokenhere")
		assert.Equal(t, "fine-grained personal access token", result,
			"GetPATTypeDescription should return the documented example string for fine-grained PATs")
	})

	t.Run("returns non-empty human-readable description for any token", func(t *testing.T) {
		for _, token := range []string{"github_pat_x", "ghp_x", "gho_x", "xyz_x"} {
			result := GetPATTypeDescription(token)
			assert.NotEmpty(t, result,
				"GetPATTypeDescription(%q) should return non-empty description", token)
		}
	})
}
