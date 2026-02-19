package parser

import (
	"github.com/github/gh-aw/pkg/stringutil"
)

// StripANSI removes ANSI escape codes from a string.
// This is a thin wrapper around stringutil.StripANSI for backward compatibility.
// The comprehensive implementation lives in pkg/stringutil/ansi.go.
func StripANSI(s string) string {
	return stringutil.StripANSI(s)
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
