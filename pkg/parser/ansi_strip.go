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
