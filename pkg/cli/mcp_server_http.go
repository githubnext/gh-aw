package cli

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// sanitizeForLog removes newline and carriage return characters from user input
// to prevent log injection attacks where malicious users could forge log entries.
func sanitizeForLog(input string) string {
	// Remove both \n and \r to prevent log injection
	sanitized := strings.ReplaceAll(input, "\n", "")
	sanitized = strings.ReplaceAll(sanitized, "\r", "")
	return sanitized
}

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func loggingHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer wrapper to capture status code.
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Sanitize user-controlled input before logging to prevent log injection
		sanitizedPath := sanitizeForLog(r.URL.Path)

		// Log request details.
		log.Printf("[REQUEST] %s | %s | %s %s",
			start.Format(time.RFC3339),
			r.RemoteAddr,
			r.Method,
			sanitizedPath)

		// Call the actual handler.
		handler.ServeHTTP(wrapped, r)

		// Log response details.
		duration := time.Since(start)
		log.Printf("[RESPONSE] %s | %s | %s %s | Status: %d | Duration: %v",
			time.Now().Format(time.RFC3339),
			r.RemoteAddr,
			r.Method,
			sanitizedPath,
			wrapped.statusCode,
			duration)
	})
}

// runHTTPServer runs the MCP server with HTTP/SSE transport
func runHTTPServer(server *mcp.Server, port int) error {
	mcpLog.Printf("Creating HTTP server on port %d", port)

	// Create the streamable HTTP handler.
	handler := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{
		SessionTimeout: 2 * time.Hour, // Close idle sessions after 2 hours
		Logger:         logger.NewSlogLoggerWithHandler(mcpLog),
	})

	handlerWithLogging := loggingHandler(handler)

	// Create HTTP server
	addr := fmt.Sprintf(":%d", port)
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           handlerWithLogging,
		ReadHeaderTimeout: MCPServerHTTPTimeout,
	}

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Starting MCP server on http://localhost%s", addr)))
	mcpLog.Printf("HTTP server listening on %s", addr)

	// Run the HTTP server
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		mcpLog.Printf("HTTP server failed: %v", err)
		return fmt.Errorf("HTTP server failed: %w", err)
	}

	return nil
}
