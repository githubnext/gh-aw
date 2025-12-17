package cli

import (
	"context"
	"errors"
	"fmt"
	"net"
	"syscall"
	"time"

	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var mcpConnectRetryLog = logger.New("cli:mcp_connect_retry")

// connectWithRetry attempts to connect to an MCP server with exponential backoff retry logic.
// It retries up to 3 times (initial attempt + 2 retries) with delays of 1s, 2s between attempts.
// Only transient network errors trigger retries; permanent errors fail immediately.
func connectWithRetry(ctx context.Context, client *mcp.Client, transport mcp.Transport, connectOptions *mcp.ClientSessionOptions) (*mcp.ClientSession, error) {
	const maxAttempts = 3
	var lastErr error

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			mcpConnectRetryLog.Printf("Retry attempt %d/%d", attempt+1, maxAttempts)
		}

		session, err := client.Connect(ctx, transport, connectOptions)
		if err == nil {
			if attempt > 0 {
				mcpConnectRetryLog.Printf("Successfully connected after %d attempts", attempt+1)
			}
			return session, nil
		}

		lastErr = err
		mcpConnectRetryLog.Printf("Connection attempt %d failed: %v", attempt+1, err)

		// Check if error is transient and worth retrying
		if !isTransientError(err) {
			mcpConnectRetryLog.Printf("Error is not transient, failing immediately")
			return nil, err
		}

		// Don't retry on the last attempt
		if attempt == maxAttempts-1 {
			break
		}

		// Calculate backoff delay: 1s, 2s, 4s
		backoffDelay := time.Second * time.Duration(1<<attempt)
		mcpConnectRetryLog.Printf("Waiting %v before retry", backoffDelay)

		// Wait for backoff or context cancellation
		select {
		case <-ctx.Done():
			mcpConnectRetryLog.Print("Context cancelled during retry backoff")
			return nil, ctx.Err()
		case <-time.After(backoffDelay):
			// Continue to next retry attempt
		}
	}

	return nil, fmt.Errorf("failed to connect after %d attempts: %w", maxAttempts, lastErr)
}

// isTransientError determines if an error is transient and worth retrying.
// Returns true for network errors like connection refused, timeouts, and temporary DNS failures.
// Returns false for permanent errors like invalid configuration or authentication failures.
func isTransientError(err error) bool {
	if err == nil {
		return false
	}

	// Check for context errors - these are not transient
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Check for common transient network errors
	var netErr net.Error
	if errors.As(err, &netErr) {
		// Timeout errors are transient
		if netErr.Timeout() {
			return true
		}
	}

	// Check for connection refused (ECONNREFUSED)
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		// Connection refused is transient - server might be starting up
		if errors.Is(opErr.Err, syscall.ECONNREFUSED) {
			return true
		}
		// Network unreachable is transient
		if errors.Is(opErr.Err, syscall.ENETUNREACH) {
			return true
		}
		// Host unreachable is transient
		if errors.Is(opErr.Err, syscall.EHOSTUNREACH) {
			return true
		}
	}

	// Check for syscall errors directly
	if errors.Is(err, syscall.ECONNREFUSED) {
		return true
	}
	if errors.Is(err, syscall.ENETUNREACH) {
		return true
	}
	if errors.Is(err, syscall.EHOSTUNREACH) {
		return true
	}
	if errors.Is(err, syscall.ECONNRESET) {
		return true
	}

	// Default to non-transient for unknown errors
	return false
}
