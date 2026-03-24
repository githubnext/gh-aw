package parser

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/goccy/go-yaml"
)

var yamlErrorLog = logger.New("parser:yaml_error")

// Package-level compiled regex patterns for better performance
var (
	lineColPatternParser = regexp.MustCompile(`^\[(\d+):(\d+)\]`)
	definedAtPattern     = regexp.MustCompile(`already defined at \[(\d+):(\d+)\]`)
	sourceLinePattern    = regexp.MustCompile(`(?m)^(>?\s*)(\d+)(\s*\|)`)
)

// yamlErrorTranslations maps common goccy/go-yaml internal error messages to
// user-friendly plain-language descriptions with actionable fix guidance.
// Each pattern is matched case-insensitively against the parser message text only
// (not the surrounding source-context lines in yaml.FormatError() output).
//
// These translations are the single source of truth shared between the parser
// and workflow packages. See TranslateYAMLMessage for public access.
var yamlErrorTranslations = []struct {
	pattern     string
	replacement string
}{
	{
		"unexpected key name",
		"missing ':' after key — YAML mapping entries require 'key: value' format",
	},
	{
		"mapping value is not allowed in this context",
		"unexpected ':' — check indentation or if this key belongs in a mapping block",
	},
	{
		"mapping values are not allowed",
		"unexpected ':' — check indentation or if this key belongs in a mapping block",
	},
	{
		"string was used where mapping is expected",
		"expected a YAML mapping (key: value pairs) but got a plain string",
	},
	{
		"non-map value is specified",
		"expected a YAML mapping (key: value pairs) — did you forget a colon after the key?",
	},
	{
		"tab character cannot use as a map key directly",
		"tab character in key — YAML requires spaces for indentation, not tabs",
	},
	{
		"found character that cannot start any token",
		"invalid character — check indentation uses spaces, not tabs",
	},
	{
		"could not find expected ':'",
		"missing ':' in key-value pair",
	},
	{
		"did not find expected key",
		"incorrect indentation or missing key in mapping",
	},
}

// TranslateYAMLMessage translates a raw goccy/go-yaml parser message to a user-friendly
// description. It is the public entry point used by both the parser and workflow packages
// so that both code paths share a single translation table.
//
// The function performs a case-insensitive substring replacement of the first matching
// pattern, leaving any surrounding text intact. This is safe for ASCII patterns because
// strings.ToLower preserves byte positions exactly for ASCII characters.
func TranslateYAMLMessage(message string) string {
	lower := strings.ToLower(message)
	for _, t := range yamlErrorTranslations {
		if idx := strings.Index(lower, t.pattern); idx >= 0 {
			yamlErrorLog.Printf("Translating YAML message pattern %q", t.pattern)
			// Slice using idx from the lowercase string. Safe because all patterns are ASCII.
			return message[:idx] + t.replacement + message[idx+len(t.pattern):]
		}
	}
	return message
}

// translateYAMLError translates cryptic goccy/go-yaml parser messages to user-friendly descriptions.
// It operates on the full yaml.FormatError() output, which includes a header line and source context:
//
//	[line:col] original parser message
//	>  1 | some: yaml
//	       ^
//
// Only the parser message portion (the header line, after the "[line:col] " prefix) is translated.
// Source-context lines are left untouched to avoid accidentally replacing text inside user YAML content.
func translateYAMLError(formatted string) string {
	if formatted == "" {
		return formatted
	}

	// Split into the header line (which contains the parser message) and the rest (source context).
	var header, rest string
	if nl := strings.IndexByte(formatted, '\n'); nl >= 0 {
		header = formatted[:nl]
		rest = formatted[nl:]
	} else {
		header = formatted
		rest = ""
	}

	// Within the header, locate the parser message text after the "[line:col] " prefix.
	// If the prefix is absent (unusual), treat the entire header as the message.
	msgStart := strings.Index(header, "] ")
	var prefix, msg string
	if msgStart >= 0 {
		msgStart += len("] ")
		prefix = header[:msgStart]
		msg = header[msgStart:]
	} else {
		prefix = ""
		msg = header
	}

	// Translate only the message portion, leaving prefix and source context intact.
	translated := TranslateYAMLMessage(msg)

	return prefix + translated + rest
}

// FormatYAMLError formats a YAML error with source code context using yaml.FormatError()
// frontmatterLineOffset is the line number where the frontmatter content begins in the document (1-based)
// Returns the formatted error string with line numbers adjusted for frontmatter position
func FormatYAMLError(err error, frontmatterLineOffset int, sourceYAML string) string {
	yamlErrorLog.Printf("Formatting YAML error with yaml.FormatError(): offset=%d", frontmatterLineOffset)

	// Use goccy/go-yaml's native FormatError for consistent formatting with source context
	// colored=false to avoid ANSI escape codes, inclSource=true to include source lines
	formatted := yaml.FormatError(err, false, true)

	// Translate cryptic parser messages to user-friendly descriptions (header line only)
	formatted = translateYAMLError(formatted)

	// Adjust line numbers in the formatted output to account for frontmatter position
	if frontmatterLineOffset > 1 {
		formatted = adjustLineNumbersInFormattedError(formatted, frontmatterLineOffset-1)
	}

	return formatted
}

// adjustLineNumbersInFormattedError adjusts line numbers in yaml.FormatError() output
// by adding the specified offset to all line numbers
func adjustLineNumbersInFormattedError(formatted string, offset int) string {
	if offset == 0 {
		return formatted
	}

	yamlErrorLog.Printf("Adjusting YAML error line numbers with offset: +%d", offset)

	// Pattern to match line numbers in the format:
	// [line:col] at the start
	// "   1 | content" in the source context
	// ">  2 | content" with the error marker

	// Adjust [line:col] format at the start
	formatted = lineColPatternParser.ReplaceAllStringFunc(formatted, func(match string) string {
		var line, col int
		if _, err := fmt.Sscanf(match, "[%d:%d]", &line, &col); err == nil {
			return fmt.Sprintf("[%d:%d]", line+offset, col)
		}
		return match
	})

	// Adjust line numbers in "already defined at [line:col]" references
	formatted = definedAtPattern.ReplaceAllStringFunc(formatted, func(match string) string {
		var line, col int
		if _, err := fmt.Sscanf(match, "already defined at [%d:%d]", &line, &col); err == nil {
			return fmt.Sprintf("already defined at [%d:%d]", line+offset, col)
		}
		return match
	})

	// Adjust line numbers in source context lines (both "   1 |" and ">  1 |" formats)
	formatted = sourceLinePattern.ReplaceAllStringFunc(formatted, func(match string) string {
		var line int
		if strings.Contains(match, ">") {
			if _, err := fmt.Sscanf(match, "> %d |", &line); err == nil {
				return fmt.Sprintf(">%3d |", line+offset)
			}
		} else {
			if _, err := fmt.Sscanf(match, "%d |", &line); err == nil {
				return fmt.Sprintf("%4d |", line+offset)
			}
		}
		// If we can't parse it, extract parts manually
		parts := strings.Split(match, "|")
		if len(parts) == 2 {
			prefix := strings.TrimRight(parts[0], "0123456789")
			lineStr := strings.Trim(parts[0][len(prefix):], " ")
			if n, err := fmt.Sscanf(lineStr, "%d", &line); err == nil && n == 1 {
				if strings.Contains(prefix, ">") {
					return fmt.Sprintf(">%3d |", line+offset)
				}
				return fmt.Sprintf("%4d |", line+offset)
			}
		}
		return match
	})

	return formatted
}

// ExtractYAMLError extracts line and column information from YAML parsing errors
// frontmatterLineOffset is the line number where the frontmatter content begins in the document (1-based)
// This allows proper line number reporting when frontmatter is not at the beginning of the document
//
// NOTE: This function is kept for backward compatibility. New code should use FormatYAMLError()
// which leverages yaml.FormatError() for better error messages with source context.

// extractFromGoccyFormat extracts line/column from goccy/go-yaml's [line:column] message format

// extractFromStringParsing provides fallback string parsing for other YAML libraries
