package console

import (
	"fmt"
	"os"
)

// LogVerbose outputs a verbose message to stderr only when verbose mode is enabled.
// This is a convenience helper to avoid repetitive if-verbose checks throughout the codebase.
func LogVerbose(verbose bool, message string) {
	if verbose {
		fmt.Fprintln(os.Stderr, FormatVerboseMessage(message))
	}
}
