package console

import (
	"fmt"
	"io"
	"os"
)

// stderr is the writer used by all Print* helpers. Tests may replace it with a
// bytes.Buffer to capture output without touching OS file descriptors.
// Tests must not call t.Parallel() as this variable is not concurrency-safe.
var stderr io.Writer = os.Stderr

// PrintError formats and prints a compiler error to stderr.
// FormatError already includes a trailing newline, so Fprint is used to avoid
// emitting a spurious blank line.
func PrintError(err CompilerError) {
	fmt.Fprint(stderr, FormatError(err))
}

// PrintSuccessMessage formats and prints a success message to stderr.
func PrintSuccessMessage(message string) {
	fmt.Fprintln(stderr, FormatSuccessMessageStderr(message))
}

// PrintInfoMessage formats and prints an info message to stderr.
func PrintInfoMessage(message string) {
	fmt.Fprintln(stderr, FormatInfoMessageStderr(message))
}

// PrintTableHeaderStderr formats and prints a table header to stderr.
func PrintTableHeaderStderr(text string) {
	fmt.Fprintln(stderr, FormatTableHeaderStderr(text))
}

// PrintWarningMessage formats and prints a warning message to stderr.
func PrintWarningMessage(message string) {
	fmt.Fprintln(stderr, FormatWarningMessageStderr(message))
}

// PrintErrorMessage formats and prints a simple error message to stderr.
func PrintErrorMessage(message string) {
	fmt.Fprintln(stderr, FormatErrorMessage(message))
}

// PrintErrorTextStderr formats and prints error-styled text to stderr.
func PrintErrorTextStderr(text string) {
	fmt.Fprintln(stderr, FormatErrorTextStderr(text))
}

// PrintCommandMessage formats and prints a command message to stderr.
func PrintCommandMessage(command string) {
	fmt.Fprintln(stderr, FormatCommandMessageStderr(command))
}

// PrintProgressMessage formats and prints a progress message to stderr.
func PrintProgressMessage(message string) {
	fmt.Fprintln(stderr, FormatProgressMessageStderr(message))
}

// PrintPromptMessage formats and prints a prompt message to stderr.
func PrintPromptMessage(message string) {
	fmt.Fprintln(stderr, FormatPromptMessageStderr(message))
}

// PrintVerboseMessage formats and prints a verbose message to stderr.
func PrintVerboseMessage(message string) {
	fmt.Fprintln(stderr, FormatVerboseMessageStderr(message))
}

// PrintListItem formats and prints a list item to stderr.
func PrintListItem(item string) {
	fmt.Fprintln(stderr, FormatListItemStderr(item))
}

// PrintSectionHeader formats and prints a section header to stderr.
func PrintSectionHeader(header string) {
	fmt.Fprintln(stderr, FormatSectionHeaderStderr(header))
}

// PrintErrorChain formats and prints an error chain to stderr.
func PrintErrorChain(err error) {
	fmt.Fprintln(stderr, FormatErrorChain(err))
}
