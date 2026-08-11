//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHasBashExplicitRestriction_WildcardIsSafe(t *testing.T) {
	assert.False(t, HasBashExplicitRestriction(map[string]any{"bash": []any{"*"}}))
	assert.False(t, HasBashExplicitRestriction(map[string]any{"bash": []any{":*"}}))
}

func TestHasBashExplicitRestriction_AbsentIsSafe(t *testing.T) {
	assert.False(t, HasBashExplicitRestriction(nil))
	assert.False(t, HasBashExplicitRestriction(map[string]any{}))
	assert.False(t, HasBashExplicitRestriction(map[string]any{"bash": true}))
	assert.False(t, HasBashExplicitRestriction(map[string]any{"bash": nil}))
}

func TestHasBashExplicitRestriction_FalseIsRestriction(t *testing.T) {
	assert.True(t, HasBashExplicitRestriction(map[string]any{"bash": false}))
}

func TestHasBashExplicitRestriction_EmptyListIsRestriction(t *testing.T) {
	assert.True(t, HasBashExplicitRestriction(map[string]any{"bash": []any{}}))
}

func TestHasBashExplicitRestriction_NamedListIsRestriction(t *testing.T) {
	assert.True(t, HasBashExplicitRestriction(map[string]any{"bash": []any{"ls", "cat"}}))
}

func TestHasBashExplicitRestriction_MixedWildcardAmongNamesIsSafe(t *testing.T) {
	assert.False(t, HasBashExplicitRestriction(map[string]any{"bash": []any{"ls", "*", "cat"}}))
	assert.False(t, HasBashExplicitRestriction(map[string]any{"bash": []any{"ls", ":*", "cat"}}))
}
