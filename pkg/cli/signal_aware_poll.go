package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
)

var pollLog = logger.New("cli:signal_aware_poll")

// ErrInterrupted is returned when polling is interrupted by a signal or context cancellation
var ErrInterrupted = errors.New("interrupted by user")

// PollResult represents the result of a polling operation
type PollResult int

const (
	// PollContinue indicates polling should continue
	PollContinue PollResult = iota
	// PollSuccess indicates polling completed successfully
	PollSuccess
	// PollFailure indicates polling failed
	PollFailure
)

// PollOptions contains configuration for signal-aware polling
type PollOptions struct {
	// Context for cancellation (optional, but recommended for proper Ctrl-C handling)
	Ctx context.Context
	// Interval between poll attempts
	PollInterval time.Duration
	// Timeout for the entire polling operation
	Timeout time.Duration
	// Function to call on each poll iteration.
	// The ctx passed to PollFunc is the same context used by the poll loop, so callers can
	// pass it to context-aware operations (e.g. RunGHContext) to abort mid-call on Ctrl-C.
	// Should return PollContinue to keep polling, PollSuccess to succeed, or PollFailure to fail.
	PollFunc func(ctx context.Context) (PollResult, error)
	// Message to display when polling starts (optional)
	StartMessage string
	// Message to display on each poll iteration (optional)
	ProgressMessage string
	// Message to display on successful completion (optional)
	SuccessMessage string
	// Whether to show verbose progress messages
	Verbose bool
}

// PollWithSignalHandling polls with a function until it succeeds, fails, times out, or receives an interrupt signal
// This provides a reusable pattern for any operation that needs to poll with graceful Ctrl-C handling
func PollWithSignalHandling(options PollOptions) error {
	pollLog.Printf("Starting polling: interval=%v, timeout=%v", options.PollInterval, options.Timeout)
	if options.Verbose && options.StartMessage != "" {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(options.StartMessage))
	}
	ctx := pollWithSignalHandlingContext(options)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigChan)
	start := time.Now()
	ticker := time.NewTicker(options.PollInterval)
	defer ticker.Stop()

	initialResult, initialErr := options.PollFunc(ctx)
	if done, err := pollWithSignalHandlingResult(initialResult, initialErr, options); done || err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return pollWithSignalHandlingContextDone(ctx)
		case <-sigChan:
			pollLog.Print("Received interrupt signal")
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Received interrupt signal, stopping wait..."))
			return ErrInterrupted
		case <-ticker.C:
			if err := pollWithSignalHandlingTick(ctx, start, options); err != nil {
				return err
			}
		}
	}
}

func pollWithSignalHandlingContext(options PollOptions) context.Context {
	ctx := options.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	return ctx
}

func pollWithSignalHandlingResult(result PollResult, err error, options PollOptions) (bool, error) {
	switch result {
	case PollSuccess:
		if options.Verbose && options.SuccessMessage != "" {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(options.SuccessMessage))
		}
		return true, nil
	case PollFailure:
		return true, err
	default:
		return false, nil
	}
}

func pollWithSignalHandlingContextDone(ctx context.Context) error {
	pollLog.Printf("Context cancelled (%v), stopping poll", ctx.Err())
	msg := "Operation cancelled, stopping wait..."
	if err := ctx.Err(); err != nil {
		msg = fmt.Sprintf("Operation cancelled (%v), stopping wait...", err)
	}
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(msg))
	return ErrInterrupted
}

func pollWithSignalHandlingTick(ctx context.Context, start time.Time, options PollOptions) error {
	if options.Timeout > 0 && time.Since(start) > options.Timeout {
		pollLog.Printf("Timeout exceeded: %v", options.Timeout)
		return fmt.Errorf("operation timed out after %v", options.Timeout)
	}
	result, err := options.PollFunc(ctx)
	if done, err := pollWithSignalHandlingResult(result, err, options); done || err != nil {
		return err
	}
	if options.Verbose && options.ProgressMessage != "" {
		fmt.Fprintln(os.Stderr, console.FormatProgressMessage(options.ProgressMessage))
	}
	return nil
}
