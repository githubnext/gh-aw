//go:build !integration

package workflow

import (
	"testing"
)

// TestValidateStrictTools_SerenaDockerMode tests that serena docker mode is allowed in strict mode
// (tools.serena is deprecated but still accepted with a warning, not a strict mode error)
func TestValidateStrictTools_SerenaDockerMode(t *testing.T) {
	compiler := NewCompiler()
	frontmatter := map[string]any{
		"on": "push",
		"tools": map[string]any{
			"serena": map[string]any{
				"mode": "docker",
				"languages": map[string]any{
					"go": map[string]any{},
				},
			},
		},
	}

	err := compiler.validateStrictTools(frontmatter)
	if err != nil {
		t.Errorf("Expected no error for serena docker mode in strict mode (only a warning), got: %v", err)
	}
}

// TestValidateStrictTools_SerenaNoMode tests that serena without mode is allowed in strict mode
// (tools.serena is deprecated but still accepted with a warning, not a strict mode error)
func TestValidateStrictTools_SerenaNoMode(t *testing.T) {
	compiler := NewCompiler()
	frontmatter := map[string]any{
		"on": "push",
		"tools": map[string]any{
			"serena": map[string]any{
				"languages": map[string]any{
					"go": map[string]any{},
				},
			},
		},
	}

	err := compiler.validateStrictTools(frontmatter)
	if err != nil {
		t.Errorf("Expected no error for serena without mode in strict mode (only a warning), got: %v", err)
	}
}

// TestValidateStrictTools_NoSerena tests that validation passes without serena
func TestValidateStrictTools_NoSerena(t *testing.T) {
	compiler := NewCompiler()
	frontmatter := map[string]any{
		"on": "push",
		"tools": map[string]any{
			"bash": []string{"*"},
		},
	}

	err := compiler.validateStrictTools(frontmatter)
	if err != nil {
		t.Errorf("Expected no error without serena tool, got: %v", err)
	}
}
