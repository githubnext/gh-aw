//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompilerOptionsAppliedByNewCompiler(t *testing.T) {
	c := NewCompiler(
		WithVerbose(true),
		WithEngineOverride("claude"),
		WithSkipValidation(false),
		WithNoEmit(true),
		WithFailFast(true),
		WithWorkflowIdentifier("my-workflow"),
		WithVersion("v1.2.3"),
	)

	assert.True(t, c.verbose)
	assert.Equal(t, "claude", c.engineOverride)
	assert.False(t, c.skipValidation)
	assert.True(t, c.noEmit)
	assert.True(t, c.failFast)
	assert.Equal(t, "my-workflow", c.workflowIdentifier)
	assert.Equal(t, "v1.2.3", c.GetVersion())
}

func TestNewCompilerDefaults(t *testing.T) {
	c := NewCompiler()

	assert.False(t, c.verbose)
	assert.Empty(t, c.engineOverride)
	assert.True(t, c.skipValidation)
	assert.NotNil(t, c.ctx)
	assert.Equal(t, DetectActionMode(c.GetVersion()), c.GetActionMode())
}
