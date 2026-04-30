// This file provides heredoc validation functions used during workflow compilation.
//
// These exported functions validate heredoc delimiters and content to prevent
// injection attacks when embedding user-influenced content in shell heredocs.

package workflow

import (
	"errors"
	"fmt"
	"strings"
)

// ValidateHeredocContent checks that content does not contain the heredoc delimiter
// anywhere (substring match). The check is intentionally stricter than what shell
// heredocs require (delimiter on its own line) — rejecting any occurrence eliminates
// ambiguity and avoids edge cases around whitespace or partial-line matches.
//
// Callers that wrap user-influenced content (e.g. the markdown body, frontmatter scripts)
// MUST call ValidateHeredocContent before embedding that content in a heredoc.
//
// In practice, hitting this error requires finding a fixed-point where the content
// (which is part of the frontmatter hash input) produces a hash that generates a
// delimiter that also appears in the content — computationally infeasible with
// HMAC-SHA256. This check exists as defense-in-depth.
func ValidateHeredocContent(content, delimiter string) error {
	if delimiter == "" {
		return errors.New("heredoc delimiter cannot be empty")
	}
	if err := ValidateHeredocDelimiter(delimiter); err != nil {
		return err
	}
	if strings.Contains(content, delimiter) {
		return fmt.Errorf("content contains heredoc delimiter %q — possible injection attempt", delimiter)
	}
	return nil
}

// ValidateHeredocDelimiter checks that a delimiter is safe for use inside
// single-quoted heredoc syntax (<< 'DELIM'). Rejects delimiters containing
// single quotes, newlines, carriage returns, or non-printable characters
// that could break the generated shell/YAML.
func ValidateHeredocDelimiter(delimiter string) error {
	for _, r := range delimiter {
		switch {
		case r == '\'':
			return fmt.Errorf("heredoc delimiter %q contains single quote", delimiter)
		case r == '\n', r == '\r':
			return fmt.Errorf("heredoc delimiter %q contains newline", delimiter)
		case r < 0x20 && r != '\t':
			return fmt.Errorf("heredoc delimiter %q contains non-printable character %U", delimiter, r)
		}
	}
	return nil
}
