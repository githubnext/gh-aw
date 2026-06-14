//go:build !integration

package cli

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestWaitForServerReady_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := waitForServerReady(ctx, 65535, 100*time.Millisecond, false)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation error, got %v", err)
	}
}

func TestWaitForServerReady_Timeout(t *testing.T) {
	err := waitForServerReady(t.Context(), 65535, 10*time.Millisecond, false)
	if !errors.Is(err, errMCPScriptsServerStartupTimeout) {
		t.Fatalf("expected timeout error, got %v", err)
	}
}
