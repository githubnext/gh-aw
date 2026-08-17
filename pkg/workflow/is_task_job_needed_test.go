//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestIsActivationJobNeeded verifies that isActivationJobNeeded always returns true.
// As documented in compiler_jobs.go, the activation job is unconditionally required to
// perform the timestamp check, and it also handles command configuration, text output,
// runtime If conditions, and consolidated permission checks.
func TestIsActivationJobNeeded(t *testing.T) {
	compiler := NewCompiler()

	assert.True(t, compiler.isActivationJobNeeded(), "activation job is always needed")
}
