//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractRoles_OnRoles(t *testing.T) {
	compiler := &Compiler{}

	frontmatter := map[string]any{
		"on": map[string]any{
			"issues": map[string]any{
				"types": []any{"opened"},
			},
			"roles": []any{"admin", "write"},
		},
	}

	roles := compiler.extractRoles(frontmatter)

	assert.Equal(t, []string{"admin", "write"}, roles)
}

func TestExtractRoles_OnRolesString(t *testing.T) {
	compiler := &Compiler{}

	frontmatter := map[string]any{
		"on": map[string]any{
			"issues": map[string]any{
				"types": []any{"opened"},
			},
			"roles": "all",
		},
	}

	roles := compiler.extractRoles(frontmatter)

	assert.Equal(t, []string{"all"}, roles)
}

func TestExtractRoles_TopLevelRoles_NotSupported(t *testing.T) {
	compiler := &Compiler{}

	// Top-level roles is no longer supported - should return defaults
	frontmatter := map[string]any{
		"on": map[string]any{
			"issues": map[string]any{
				"types": []any{"opened"},
			},
		},
		"roles": []any{"admin", "maintainer", "write"},
	}

	roles := compiler.extractRoles(frontmatter)

	// Should return defaults since top-level roles is not recognized
	assert.Equal(t, []string{"admin", "maintainer", "write"}, roles)
}

func TestExtractRoles_TopLevelRoles_StringArray_NotSupported(t *testing.T) {
	compiler := &Compiler{}

	// Top-level roles is no longer supported - should return defaults
	frontmatter := map[string]any{
		"on": map[string]any{
			"issues": map[string]any{
				"types": []any{"opened"},
			},
		},
		"roles": []string{"admin", "write"},
	}

	roles := compiler.extractRoles(frontmatter)

	// Should return defaults since top-level roles is not recognized
	assert.Equal(t, []string{"admin", "maintainer", "write"}, roles)
}

func TestExtractRoles_OnlyOnRolesSupported(t *testing.T) {
	compiler := &Compiler{}

	// Only on.roles is supported now
	frontmatter := map[string]any{
		"on": map[string]any{
			"issues": map[string]any{
				"types": []any{"opened"},
			},
			"roles": []any{"admin"},
		},
		"roles": []any{"admin", "maintainer", "write"},
	}

	roles := compiler.extractRoles(frontmatter)

	// Should use on.roles, ignoring top-level
	assert.Equal(t, []string{"admin"}, roles)
}

func TestExtractRoles_Default(t *testing.T) {
	compiler := &Compiler{}

	frontmatter := map[string]any{
		"on": map[string]any{
			"issues": map[string]any{
				"types": []any{"opened"},
			},
		},
	}

	roles := compiler.extractRoles(frontmatter)

	assert.Equal(t, []string{"admin", "maintainer", "write"}, roles)
}

func TestExtractRoles_AllValue(t *testing.T) {
	compiler := &Compiler{}

	frontmatter := map[string]any{
		"on": map[string]any{
			"issues": map[string]any{
				"types": []any{"opened"},
			},
			"roles": "all",
		},
	}

	roles := compiler.extractRoles(frontmatter)

	assert.Equal(t, []string{"all"}, roles)
}
