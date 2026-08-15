//go:build !integration

package ctxutil_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/github/gh-aw/pkg/ctxutil"
)

// TestSpec_PublicAPI_OrBackground validates the documented behavior of
// OrBackground as described in the ctxutil README.md specification.
func TestSpec_PublicAPI_OrBackground(t *testing.T) {
	type key string
	const testKey key = "spec-key"

	t.Run("returns context.Background() when ctx is nil", func(t *testing.T) {
		var nilCtx context.Context

		got := ctxutil.OrBackground(nilCtx)
		assert.NotNil(t, got, "OrBackground should never return a nil context")
		assert.Equal(t, context.Background(), got, "OrBackground should fall back to context.Background() for a nil input")
	})

	t.Run("returns the context unchanged when ctx is non-nil", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), testKey, "value")

		got := ctxutil.OrBackground(ctx)

		assert.Same(t, ctx, got, "OrBackground should return the exact same non-nil context instance")
		assert.Equal(t, "value", got.Value(testKey), "OrBackground should preserve values carried by the input context")
	})

	t.Run("preserves cancellation of the input context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		got := ctxutil.OrBackground(ctx)
		cancel()

		assert.ErrorIs(t, got.Err(), context.Canceled, "OrBackground should not detach the input context from its cancellation")
	})
}
