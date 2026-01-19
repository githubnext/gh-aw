package cli

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
)

// TestSendProgressWithNilToken tests that sendProgress handles nil tokens gracefully
func TestSendProgressWithNilToken(t *testing.T) {
	// Create a minimal request without a progress token
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{},
	}

	// Verify no progress token is set
	assert.Nil(t, req.Params.GetProgressToken(), "Progress token should be nil")

	// This should not panic or cause errors
	ctx := context.Background()
	assert.NotPanics(t, func() {
		sendProgress(ctx, req, "Test message", 1, 5)
	}, "sendProgress should handle nil token gracefully")
}

// TestSendProgressDoesNotPanic tests that sendProgress doesn't panic
func TestSendProgressDoesNotPanic(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		progress float64
		total    float64
	}{
		{
			name:     "basic progress",
			message:  "Processing data",
			progress: 1,
			total:    5,
		},
		{
			name:     "zero progress",
			message:  "Starting",
			progress: 0,
			total:    10,
		},
		{
			name:     "complete progress",
			message:  "Done",
			progress: 5,
			total:    5,
		},
		{
			name:     "unknown total",
			message:  "Processing",
			progress: 5,
			total:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a minimal request without a progress token
			req := &mcp.CallToolRequest{
				Params: &mcp.CallToolParamsRaw{},
			}

			// This should not panic even without a valid session
			ctx := context.Background()
			assert.NotPanics(t, func() {
				sendProgress(ctx, req, tt.message, tt.progress, tt.total)
			}, "sendProgress should not panic")
		})
	}
}


