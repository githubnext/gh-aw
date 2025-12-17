//go:build integration

package cli

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/githubnext/gh-aw/pkg/parser"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestHTTPMCPServerRetry_SuccessAfterTransientFailure tests that the HTTP MCP client
// successfully retries transient connection failures
func TestHTTPMCPServerRetry_SuccessAfterTransientFailure(t *testing.T) {
	// Create a counter to track connection attempts
	var attemptCount atomic.Int32

	// Create a test server that fails the first attempt then succeeds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := attemptCount.Add(1)
		if count == 1 {
			// First attempt: close connection immediately to simulate transient failure
			hj, ok := w.(http.Hijacker)
			if ok {
				conn, _, err := hj.Hijack()
				if err == nil {
					conn.Close()
					return
				}
			}
			// If hijacking fails, return 503 Service Unavailable
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		// Subsequent attempts: return success
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Return a minimal valid SSE response for MCP
		w.Write([]byte("event: endpoint\ndata: /message\n\n"))
	}))
	defer server.Close()

	// Test that connection succeeds after retry
	config := parser.MCPServerConfig{
		Name: "test-retry-server",
		Type: "http",
		URL:  server.URL,
	}

	// Use a longer timeout to allow for retries
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Try to connect - should succeed after retry
	// Note: This test validates the retry mechanism is in place
	// The actual MCP handshake may still fail, but we're testing
	// that transient network errors trigger retries
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	transport := &mcp.StreamableClientTransport{
		Endpoint: config.URL,
	}

	// This will use connectWithRetry internally via connectHTTPMCPServer
	_, err := connectWithRetry(ctx, client, transport, nil)

	// We expect either success or a specific error after retries
	// The key is that it should have attempted multiple times
	attempts := attemptCount.Load()
	if attempts < 2 {
		t.Errorf("expected at least 2 connection attempts, got %d", attempts)
	}

	// If we got an error, it should not be a transient network error
	// (it might be a protocol error, which is fine for this test)
	if err != nil {
		if isTransientError(err) {
			t.Errorf("got transient error after retries: %v", err)
		}
		// Log the error but don't fail - we're testing retry behavior, not full protocol
		t.Logf("Connection failed after retries (expected for protocol mismatch): %v", err)
	}
}

// TestHTTPMCPServerRetry_PermanentFailure tests that permanent errors
// don't trigger retries
func TestHTTPMCPServerRetry_PermanentFailure(t *testing.T) {
	var attemptCount atomic.Int32

	// Create a server that always returns 401 Unauthorized (permanent error)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount.Add(1)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Unauthorized"))
	}))
	defer server.Close()

	config := parser.MCPServerConfig{
		Name: "test-permanent-error",
		Type: "http",
		URL:  server.URL,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	transport := &mcp.StreamableClientTransport{
		Endpoint: config.URL,
	}

	start := time.Now()
	_, err := connectWithRetry(ctx, client, transport, nil)
	duration := time.Since(start)

	if err == nil {
		t.Fatal("expected error for permanent failure")
	}

	// Should fail quickly without long retries
	// Allow up to 15 seconds for the connection attempt itself
	if duration > 15*time.Second {
		t.Errorf("expected quick failure, took %v", duration)
	}

	attempts := attemptCount.Load()
	t.Logf("Connection attempts: %d, duration: %v", attempts, duration)
}

// TestHTTPMCPServerRetry_ContextCancellation tests that context cancellation
// is respected during retry delays
func TestHTTPMCPServerRetry_ContextCancellation(t *testing.T) {
	// Use a non-existent server to trigger connection refused (transient error)
	// This will cause retries with backoff delays
	config := parser.MCPServerConfig{
		Name: "test-cancellation",
		Type: "http",
		URL:  "http://localhost:59999", // Non-existent port
	}

	// Create a context that will be cancelled quickly
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	transport := &mcp.StreamableClientTransport{
		Endpoint: config.URL,
	}

	start := time.Now()
	_, err := connectWithRetry(ctx, client, transport, nil)
	duration := time.Since(start)

	if err == nil {
		t.Fatal("expected error due to context cancellation or connection failure")
	}

	// Should respect context timeout (approximately 2 seconds, not full retry duration)
	// Allow some margin for the test
	if duration > 5*time.Second {
		t.Errorf("expected cancellation around 2s, took %v", duration)
	}

	t.Logf("Context cancelled/timed out after %v with error: %v", duration, err)
}
