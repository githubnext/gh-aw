//go:build !integration

package workflow

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflowData_PinContext_SkipHardcodedFallback(t *testing.T) {
	originalDefaultHost := getDefaultGHHost()
	t.Cleanup(func() {
		SetDefaultGHHost(originalDefaultHost)
	})

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

	t.Run("GH_HOST=github.com wins over non-github.com default host", func(t *testing.T) {
		t.Setenv("GH_HOST", "github.com")
		SetDefaultGHHost("myorg.ghe.com")
		t.Cleanup(func() { SetDefaultGHHost("") })

		d := &WorkflowData{}
		ctx := d.PinContext()

		require.NotNil(t, ctx)
		assert.False(t, ctx.SkipHardcodedFallback, "Expected SkipHardcodedFallback to be false when GH_HOST=github.com even if default host is GHE")
	})

	t.Run("does not set SkipHardcodedFallback when GH_HOST is not set", func(t *testing.T) {
		require.NoError(t, os.Unsetenv("GH_HOST"))
		SetDefaultGHHost("")

		d := &WorkflowData{}
		ctx := d.PinContext()

		require.NotNil(t, ctx)
		assert.False(t, ctx.SkipHardcodedFallback, "Expected SkipHardcodedFallback to be false when GH_HOST is not set")
	})

	t.Run("sets SkipHardcodedFallback when default GH host is a non-github.com host", func(t *testing.T) {
		require.NoError(t, os.Unsetenv("GH_HOST"))
		SetDefaultGHHost("myorg.ghe.com")

		d := &WorkflowData{}
		ctx := d.PinContext()

		require.NotNil(t, ctx)
		assert.True(t, ctx.SkipHardcodedFallback, "Expected SkipHardcodedFallback to be true when default GH host is a GHE host")
	})

	t.Run("does not set SkipHardcodedFallback when default GH host is github.com", func(t *testing.T) {
		require.NoError(t, os.Unsetenv("GH_HOST"))
		SetDefaultGHHost("github.com")

		d := &WorkflowData{}
		ctx := d.PinContext()

		require.NotNil(t, ctx)
		assert.False(t, ctx.SkipHardcodedFallback, "Expected SkipHardcodedFallback to be false when default GH host is github.com")
	})

	t.Run("returns nil for nil WorkflowData", func(t *testing.T) {
		var d *WorkflowData
		ctx := d.PinContext()
		assert.Nil(t, ctx)
	})
}
