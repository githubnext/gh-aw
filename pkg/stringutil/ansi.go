// Package stringutil provides utility functions for working with strings.
package stringutil

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var ansiLog = logger.New("stringutil:ansi")

// StripANSI removes ANSI escape codes from a string using a comprehensive byte scanner.
// It handles CSI sequences (\x1b[), OSC sequences (\x1b]), G0/G1 character set selections,
// keypad mode sequences, reset sequences, and other common 2-character escape sequences.
//
// This is more thorough than regex-based approaches and correctly handles edge cases
// such as incomplete sequences, nested sequences, and non-standard terminal sequences.
func StripANSI(s string) string {
	if s == "" {
		return s
	}

	ansiLog.Printf("StripANSI: input length=%d", len(s))

	var result strings.Builder
	result.Grow(len(s)) // Pre-allocate capacity for efficiency

	i := 0
	for i < len(s) {
		if s[i] != '\x1b' {
			result.WriteByte(s[i])
			i++
			continue
		}
		if i+1 >= len(s) {
			i++ // ESC at end of string, skip it
			continue
		}
		// Found ESC character, advance past the sequence
		i = skipEscapeSequence(s, i)
	}

	return result.String()
}

// skipEscapeSequence advances past a complete ANSI escape sequence starting at i.
// i must point at '\x1b' and i+1 must be within the string.
func skipEscapeSequence(s string, i int) int {
	switch s[i+1] {
	case '[':
		return skipCSISequence(s, i)
	case ']':
		return skipOSCSequence(s, i)
	case '(', ')':
		// G0/G1 character set selection: \x1b(char or \x1b)char
		i += 2 // Skip ESC and ( or )
		if i < len(s) {
			i++ // Skip the charset character
		}
		return i
	case '=', '>', 'c':
		// Application keypad, normal keypad, or reset: 2-char sequences
		return i + 2
	default:
		// Other 2-character escape sequences (\x1b7, \x1b8, \x1bD, etc.)
		if s[i+1] >= '0' && s[i+1] <= '~' {
			return i + 2
		}
		// Invalid or incomplete escape sequence, skip only ESC
		return i + 1
	}
}

// skipCSISequence advances past a CSI sequence (\x1b[...final_char).
// Parameters are in range 0x30-0x3F, intermediate chars 0x20-0x2F, final 0x40-0x7E.
func skipCSISequence(s string, i int) int {
	i += 2 // Skip ESC and [
	for i < len(s) {
		if isFinalCSIChar(s[i]) {
			return i + 1 // Skip the final character
		} else if isCSIParameterChar(s[i]) {
			i++ // Skip parameter/intermediate character
		} else {
			break // Invalid character, stop
		}
	}
	return i
}

// skipOSCSequence advances past an OSC sequence (\x1b]...terminator).
// Terminators: \x07 (BEL) or \x1b\\ (ST).
func skipOSCSequence(s string, i int) int {
	i += 2 // Skip ESC and ]
	for i < len(s) {
		if s[i] == '\x07' {
			return i + 1 // Skip BEL
		}
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '\\' {
			return i + 2 // Skip ESC and \
		}
		i++
	}
	return i
}

// isFinalCSIChar checks if a character is a valid CSI final character
// Final characters are in range 0x40-0x7E (@-~)
func isFinalCSIChar(b byte) bool {
	return b >= 0x40 && b <= 0x7E
}

// isCSIParameterChar checks if a character is a valid CSI parameter or intermediate character
// Parameter characters are in range 0x30-0x3F (0-?)
// Intermediate characters are in range 0x20-0x2F (space-/)
func isCSIParameterChar(b byte) bool {
	return (b >= 0x20 && b <= 0x2F) || (b >= 0x30 && b <= 0x3F)
}
