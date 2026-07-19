package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
)

var retryLog = logger.New("cli:retry")

// RepeatOptions contains configuration for the repeat functionality
type RepeatOptions struct {
	// Context for cancellation (optional, but recommended for proper Ctrl-C handling)
	Ctx context.Context
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
	retryLog.Printf("Executing function with repeat count: %d", options.RepeatCount)
	if err := options.ExecuteFunc(); err != nil {
		retryLog.Printf("Initial execution failed: %v", err)
		return err
	}
	if options.RepeatCount <= 0 {
		retryLog.Print("No repeat requested, execution complete")
		return nil
	}

	output := ExecuteWithRepeatOutput(options)
	ctx := ExecuteWithRepeatContext(options)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	runCleanup := func() {
		if options.CleanupFunc != nil {
			retryLog.Print("Executing cleanup function")
			options.CleanupFunc()
		}
	}
	for i := 1; i <= options.RepeatCount; i++ {
		if err := ExecuteWithRepeatIteration(ctx, sigChan, output, options, i, runCleanup); err != nil {
			return err
		}
	}
	retryLog.Printf("Completed all %d iterations successfully", options.RepeatCount)
	return nil
}

func ExecuteWithRepeatOutput(options RepeatOptions) *os.File {
	retryLog.Printf("Starting repeat mode for %d iterations", options.RepeatCount)
	output := os.Stdout
	if options.UseStderr {
		output = os.Stderr
	}
	startMsg := options.StartMessage
	if startMsg == "" {
		startMsg = fmt.Sprintf("Repeating %d more times. Press Ctrl+C to stop.", options.RepeatCount)
	}
	fmt.Fprintln(output, console.FormatInfoMessage(startMsg))
	return output
}

func ExecuteWithRepeatContext(options RepeatOptions) context.Context {
	ctx := options.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	return ctx
}

func ExecuteWithRepeatIteration(ctx context.Context, sigChan <-chan os.Signal, output *os.File, options RepeatOptions, i int, runCleanup func()) error {
	select {
	case <-ctx.Done():
		retryLog.Printf("Context cancelled at iteration %d/%d", i, options.RepeatCount)
		fmt.Fprintln(output, console.FormatInfoMessage("Received interrupt signal, stopping repeat..."))
		runCleanup()
		return ctx.Err()
	case <-sigChan:
		retryLog.Printf("Interrupt signal received at iteration %d/%d", i, options.RepeatCount)
		fmt.Fprintln(output, console.FormatInfoMessage("Received interrupt signal, stopping repeat..."))
		runCleanup()
		return context.Canceled
	default:
		retryLog.Printf("Starting iteration %d/%d", i, options.RepeatCount)
		ExecuteWithRepeatRunIteration(output, options, i)
		return nil
	}
}

func ExecuteWithRepeatRunIteration(output *os.File, options RepeatOptions, i int) {
	repeatMsg := options.RepeatMessage
	if repeatMsg == "" {
		repeatMsg = fmt.Sprintf("Running repetition %d/%d", i, options.RepeatCount)
	} else if strings.Contains(repeatMsg, "%s") {
		repeatMsg = fmt.Sprintf(repeatMsg, time.Now().Format("2006-01-02 15:04:05"))
	}
	fmt.Fprintln(output, console.FormatInfoMessage(repeatMsg))
	if err := options.ExecuteFunc(); err != nil {
		retryLog.Printf("Error during iteration %d: %v", i, err)
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Error during repeat %d/%d: %v", i, options.RepeatCount, err)))
	}
}
