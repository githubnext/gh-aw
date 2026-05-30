// This file provides heredoc validation functions used during workflow compilation.
//
// These exported functions validate heredoc delimiters and content to prevent
// injection attacks when embedding user-influenced content in shell heredocs.

package workflow

import (
	"errors"
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var heredocLog = logger.New("workflow:heredoc_validation")

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
	heredocLog.Printf("Validating heredoc content against delimiter %q (content length: %d)", delimiter, len(content))
	if delimiter == "" {
		return errors.New("heredoc delimiter cannot be empty")
	}
	if err := ValidateHeredocDelimiter(delimiter); err != nil {
		return err
	}
	if strings.Contains(content, delimiter) {
		heredocLog.Printf("Heredoc injection detected: delimiter %q found in content", delimiter)
		return fmt.Errorf("content contains heredoc delimiter %q — possible injection attempt", delimiter)
	}
	return nil
}

// ValidateHeredocDelimiter checks that a delimiter is safe for use inside
// single-quoted heredoc syntax (<< 'DELIM'). Rejects delimiters containing
// single quotes, newlines, carriage returns, or non-printable characters
// that could break the generated shell/YAML.
func ValidateHeredocDelimiter(delimiter string) error {
	heredocLog.Printf("Validating heredoc delimiter %q", delimiter)
	for _, r := range delimiter {
		switch {
		case r == '\'':
			heredocLog.Printf("Invalid delimiter %q: contains single quote", delimiter)
			return fmt.Errorf("heredoc delimiter %q contains single quote", delimiter)
		case r == '\n', r == '\r':
			heredocLog.Printf("Invalid delimiter %q: contains newline", delimiter)
			return fmt.Errorf("heredoc delimiter %q contains newline", delimiter)
		case r < 0x20 && r != '\t':
			heredocLog.Printf("Invalid delimiter %q: contains non-printable character %U", delimiter, r)
			return fmt.Errorf("heredoc delimiter %q contains non-printable character %U", delimiter, r)
		}
	}
	return nil
}
