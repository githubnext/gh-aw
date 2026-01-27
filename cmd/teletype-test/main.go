// Manual test program for teletype effect
// Run with: go run cmd/teletype-test/main.go
package main

import (
"fmt"
"os"

"github.com/githubnext/gh-aw/pkg/console"
)

func main() {
// Test basic teletype
fmt.Fprintln(os.Stderr, "=== Teletype Effect Demo ===")
fmt.Fprintln(os.Stderr, "")

// Welcome message
_ = console.TeletypeWriteln(os.Stderr, "🚀 Welcome to GitHub Agentic Workflows!")
fmt.Fprintln(os.Stderr, "")

// Description
_ = console.TeletypeWriteln(os.Stderr, "This tool will walk you through adding an automated workflow to your repository.")
fmt.Fprintln(os.Stderr, "")

// Success messages
_ = console.TeletypeWriteln(os.Stderr, console.FormatSuccessMessage("GitHub CLI authenticated"))
_ = console.TeletypeWriteln(os.Stderr, console.FormatSuccessMessage("Target repository: githubnext/gh-aw"))
_ = console.TeletypeWriteln(os.Stderr, console.FormatSuccessMessage("GitHub Actions is enabled"))
_ = console.TeletypeWriteln(os.Stderr, console.FormatSuccessMessage("Repository permissions verified"))
fmt.Fprintln(os.Stderr, "")

// Info message
_ = console.TeletypeWriteln(os.Stderr, console.FormatInfoMessage("This workflow helps automate daily repository status reports."))
fmt.Fprintln(os.Stderr, "")

// Multiple lines
_ = console.TeletypeWriteLines(os.Stderr,
"Setting up your workflow...",
"Configuring AI engine...",
"Creating pull request...",
)
fmt.Fprintln(os.Stderr, "")

// Final message
_ = console.TeletypeWriteln(os.Stderr, console.FormatSuccessMessage("✨ All done! Your workflow is ready to use."))
}
