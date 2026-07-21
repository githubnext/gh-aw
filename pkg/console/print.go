package console

import (
	"fmt"
	"os"
)

// PrintError formats and prints a compiler error to stderr.
func PrintError(err CompilerError) (int, error) {
	return fmt.Fprintln(os.Stderr, FormatError(err))
}

// PrintSuccessMessage formats and prints a success message to stderr.
func PrintSuccessMessage(message string) (int, error) {
	return fmt.Fprintln(os.Stderr, FormatSuccessMessage(message))
}

// PrintInfoMessage formats and prints an info message to stderr.
func PrintInfoMessage(message string) (int, error) {
	return fmt.Fprintln(os.Stderr, FormatInfoMessage(message))
}

// PrintTableHeaderStderr formats and prints a table header to stderr.
func PrintTableHeaderStderr(text string) (int, error) {
	return fmt.Fprintln(os.Stderr, FormatTableHeaderStderr(text))
}

// PrintWarningMessage formats and prints a warning message to stderr.
func PrintWarningMessage(message string) (int, error) {
	return fmt.Fprintln(os.Stderr, FormatWarningMessage(message))
}

// PrintErrorMessage formats and prints a simple error message to stderr.
func PrintErrorMessage(message string) (int, error) {
	return fmt.Fprintln(os.Stderr, FormatErrorMessage(message))
}

// PrintErrorTextStderr formats and prints error-styled text to stderr.
func PrintErrorTextStderr(text string) (int, error) {
	return fmt.Fprintln(os.Stderr, FormatErrorTextStderr(text))
}

// PrintCommandMessage formats and prints a command message to stderr.
func PrintCommandMessage(command string) (int, error) {
	return fmt.Fprintln(os.Stderr, FormatCommandMessage(command))
}

// PrintProgressMessage formats and prints a progress message to stderr.
func PrintProgressMessage(message string) (int, error) {
	return fmt.Fprintln(os.Stderr, FormatProgressMessage(message))
}

// PrintPromptMessage formats and prints a prompt message to stderr.
func PrintPromptMessage(message string) (int, error) {
	return fmt.Fprintln(os.Stderr, FormatPromptMessage(message))
}

// PrintVerboseMessage formats and prints a verbose message to stderr.
func PrintVerboseMessage(message string) (int, error) {
	return fmt.Fprintln(os.Stderr, FormatVerboseMessage(message))
}

// PrintListItem formats and prints a list item to stderr.
func PrintListItem(item string) (int, error) {
	return fmt.Fprintln(os.Stderr, FormatListItem(item))
}

// PrintSectionHeader formats and prints a section header to stderr.
func PrintSectionHeader(header string) (int, error) {
	return fmt.Fprintln(os.Stderr, FormatSectionHeader(header))
}

// PrintErrorChain formats and prints an error chain to stderr.
func PrintErrorChain(err error) (int, error) {
	return fmt.Fprintln(os.Stderr, FormatErrorChain(err))
}
