//go:build !integration

package workflow

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflowData_PinContext_SkipHardcodedFallback(t *testing.T) {
	t.Run("sets SkipHardcodedFallback when GH_HOST is a non-github.com host", func(t *testing.T) {
		t.Setenv("GH_HOST", "myorg.ghe.com")

		d := &WorkflowData{}
		ctx := d.PinContext()

		require.NotNil(t, ctx)
		assert.True(t, ctx.SkipHardcodedFallback, "Expected SkipHardcodedFallback to be true when GH_HOST is a GHE host")
	})

	t.Run("does not set SkipHardcodedFallback when GH_HOST is github.com", func(t *testing.T) {
		t.Setenv("GH_HOST", "github.com")

		d := &WorkflowData{}
		ctx := d.PinContext()

		require.NotNil(t, ctx)
		assert.False(t, ctx.SkipHardcodedFallback, "Expected SkipHardcodedFallback to be false when GH_HOST is github.com")
	})

	t.Run("does not set SkipHardcodedFallback when GH_HOST is not set", func(t *testing.T) {
		require.NoError(t, os.Unsetenv("GH_HOST"))

		d := &WorkflowData{}
		ctx := d.PinContext()

		require.NotNil(t, ctx)
		assert.False(t, ctx.SkipHardcodedFallback, "Expected SkipHardcodedFallback to be false when GH_HOST is not set")
	})

	t.Run("returns nil for nil WorkflowData", func(t *testing.T) {
		var d *WorkflowData
		ctx := d.PinContext()
		assert.Nil(t, ctx)
	})
}
