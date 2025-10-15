package cli

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/githubnext/gh-aw/pkg/console"
)

// RepeatOptions contains configuration for the repeat functionality
type RepeatOptions struct {
	// Number of times to repeat execution (0 = run once)
	RepeatCount int
	// Message to display when starting repeat mode
	StartMessage string
	// Message to display on each repeat iteration (optional, uses default if empty)
	RepeatMessage string
	// Function to execute on each iteration
	ExecuteFunc func() error
	// Function to execute on cleanup/exit (optional)
	CleanupFunc func()
	// Whether to use stderr for informational messages (default: true)
	UseStderr bool
}

// ExecuteWithRepeat runs a function once, and optionally repeats it the specified number of times
// with graceful signal handling for shutdown.
func ExecuteWithRepeat(options RepeatOptions) error {
	// Run the function once
	if err := options.ExecuteFunc(); err != nil {
		return err
	}

	// If no repeat specified, we're done
	if options.RepeatCount <= 0 {
		return nil
	}

	// Set up repeat mode
	output := os.Stdout
	if options.UseStderr {
		output = os.Stderr
	}

	// Use provided start message or default
	startMsg := options.StartMessage
	if startMsg == "" {
		startMsg = fmt.Sprintf("Repeating %d more times. Press Ctrl+C to stop.", options.RepeatCount)
	}
	fmt.Fprintln(output, console.FormatInfoMessage(startMsg))

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Run the specified number of additional times
	for i := 1; i <= options.RepeatCount; i++ {
		select {
		case <-sigChan:
			fmt.Fprintln(output, console.FormatInfoMessage("Received interrupt signal, stopping repeat..."))

			// Execute cleanup function if provided
			if options.CleanupFunc != nil {
				options.CleanupFunc()
			}

			return nil
		default:
			// Use provided repeat message or default
			repeatMsg := options.RepeatMessage
			if repeatMsg == "" {
				repeatMsg = fmt.Sprintf("Running repetition %d/%d", i, options.RepeatCount)
			} else {
				// If message contains timestamp placeholder, replace it with current time
				if strings.Contains(repeatMsg, "%s") {
					repeatMsg = fmt.Sprintf(repeatMsg, time.Now().Format("2006-01-02 15:04:05"))
				}
			}
			fmt.Fprintln(output, console.FormatInfoMessage(repeatMsg))

			if err := options.ExecuteFunc(); err != nil {
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Error during repeat %d/%d: %v", i, options.RepeatCount, err)))
				// Continue running on error during repeat
			}
		}
	}

	return nil
}
