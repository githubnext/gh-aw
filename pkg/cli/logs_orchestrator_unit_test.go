//go:build !integration

package cli

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestIsDeadlineExceeded verifies that the helper correctly identifies
// context.DeadlineExceeded and returns false for other cases (including nil error).
func TestIsDeadlineExceeded(t *testing.T) {
	t.Run("deadline exceeded context", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
		defer cancel()
		time.Sleep(time.Millisecond) // ensure deadline has fired
		assert.True(t, isDeadlineExceeded(ctx), "expected true for DeadlineExceeded context")
	})

	t.Run("cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		assert.False(t, isDeadlineExceeded(ctx), "expected false for cancelled (not deadline) context")
	})

	t.Run("active context", func(t *testing.T) {
		ctx := context.Background()
		assert.False(t, isDeadlineExceeded(ctx), "expected false for active (non-cancelled) context")
	})
}
