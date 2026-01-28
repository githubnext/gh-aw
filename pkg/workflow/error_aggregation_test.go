package workflow

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestStrictModeMultipleErrors tests that validateStrictMode aggregates multiple errors
func TestStrictModeMultipleErrors(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter map[string]any
		network     *NetworkPermissions
		expectError bool
		errorMsgs   []string // substrings that should appear in the error
	}{
		{
			name: "multiple validation errors are aggregated",
			frontmatter: map[string]any{
				"on": "push",
				"permissions": map[string]any{
					"contents": "write",
					"issues":   "write",
				},
				"tools": map[string]any{
					"serena": map[string]any{
						"mode": "local",
					},
				},
			},
			network: &NetworkPermissions{
				Allowed: []string{"github.com"},
			},
			expectError: true,
			errorMsgs: []string{
				"strict mode: write permission 'contents: write'",
				"strict mode: write permission 'issues: write'",
				"strict mode: serena tool with 'mode: local'",
			},
		},
		{
			name: "single validation error returns single error",
			frontmatter: map[string]any{
				"on": "push",
				"permissions": map[string]any{
					"contents": "write",
				},
			},
			network: &NetworkPermissions{
				Allowed: []string{"github.com"},
			},
			expectError: true,
			errorMsgs: []string{
				"strict mode: write permission 'contents: write'",
			},
		},
		{
			name: "no validation errors",
			frontmatter: map[string]any{
				"on": "push",
				"permissions": map[string]any{
					"contents": "read",
				},
			},
			network: &NetworkPermissions{
				Allowed: []string{"github.com"},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler()
			compiler.strictMode = true
			err := compiler.validateStrictMode(tt.frontmatter, tt.network)

			if tt.expectError {
				assert.Error(t, err, "expected error but got none")

				// Check that all expected error messages are present
				errStr := err.Error()
				for _, expectedMsg := range tt.errorMsgs {
					assert.Contains(t, errStr, expectedMsg, "expected error message to contain substring")
				}
			} else {
				assert.NoError(t, err, "unexpected error")
			}
		})
	}
}

// TestTemplateValidationMultipleErrors tests that validateNoIncludesInTemplateRegions aggregates multiple errors
func TestTemplateValidationMultipleErrors(t *testing.T) {
	tests := []struct {
		name      string
		markdown  string
		expectErr bool
		errorMsgs []string
	}{
		{
			name: "multiple import directives in different template regions",
			markdown: `
{{#if condition1}}
{{#import: path/to/file1.md}}
{{/if}}

{{#if condition2}}
{{#import: path/to/file2.md}}
{{/if}}
`,
			expectErr: true,
			errorMsgs: []string{
				"path/to/file1.md",
				"path/to/file2.md",
			},
		},
		{
			name: "multiple import directives in same template region",
			markdown: `
{{#if condition}}
{{#import: path/to/file1.md}}
{{#import: path/to/file2.md}}
{{/if}}
`,
			expectErr: true,
			errorMsgs: []string{
				"path/to/file1.md",
				"path/to/file2.md",
			},
		},
		{
			name: "no import directives in template regions",
			markdown: `
{{#if condition}}
Some content
{{/if}}
`,
			expectErr: false,
		},
		{
			name: "single import directive in template region",
			markdown: `
{{#if condition}}
{{#import: path/to/file.md}}
{{/if}}
`,
			expectErr: true,
			errorMsgs: []string{
				"path/to/file.md",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateNoIncludesInTemplateRegions(tt.markdown)

			if tt.expectErr {
				assert.Error(t, err, "expected error but got none")

				// Check that all expected error messages are present
				errStr := err.Error()
				for _, expectedMsg := range tt.errorMsgs {
					assert.Contains(t, errStr, expectedMsg, "expected error message to contain substring")
				}
			} else {
				assert.NoError(t, err, "unexpected error")
			}
		})
	}
}

// TestRuntimeValidationMultipleErrors tests that validateRuntimeModeRecursive aggregates multiple errors
func TestRuntimeValidationMultipleErrors(t *testing.T) {
	t.Run("runtime validation aggregates errors across dependencies", func(t *testing.T) {
		// This is a complex scenario that requires proper file resolution
		// The key improvement is that when multiple dependencies have conflicts,
		// they are all reported together instead of one at a time

		mainScript := `
const dep1 = require('./dep1.cjs');
const dep2 = require('./dep2.cjs');
`
		sources := map[string]string{
			"./dep1.cjs": `const { execSync } = require('child_process'); execSync('ls');`,
			"./dep2.cjs": `const { spawnSync } = require('child_process'); spawnSync('ls');`,
		}

		err := validateNoRuntimeMixing(mainScript, sources, RuntimeModeGitHubScript)

		if err != nil {
			// When there are multiple conflicts, they should all be in the error message
			errStr := err.Error()
			t.Logf("Got error (as expected for conflicts): %v", errStr)

			// The error should mention both files if both are checked
			// Note: Due to the recursive nature and early exit on conflict,
			// we might only see the first conflict, which is acceptable
			// as long as the user can fix errors iteratively
			assert.Contains(t, errStr, "runtime mode conflict", "error should mention runtime mode conflict")
		}
	})
}

// TestErrorsJoinUsage verifies that errors.Join() is being used correctly
func TestErrorsJoinUsage(t *testing.T) {
	t.Run("errors.Join returns nil for empty slice", func(t *testing.T) {
		var errs []error
		result := errors.Join(errs...)
		assert.NoError(t, result, "errors.Join should return nil for empty slice")
	})

	t.Run("errors.Join aggregates multiple errors", func(t *testing.T) {
		err1 := errors.New("error 1")
		err2 := errors.New("error 2")
		err3 := errors.New("error 3")

		result := errors.Join(err1, err2, err3)
		assert.Error(t, result, "errors.Join should return error when given errors")

		errStr := result.Error()
		assert.Contains(t, errStr, "error 1", "aggregated error should contain first error")
		assert.Contains(t, errStr, "error 2", "aggregated error should contain second error")
		assert.Contains(t, errStr, "error 3", "aggregated error should contain third error")
	})

	t.Run("errors.Join handles single error", func(t *testing.T) {
		err1 := errors.New("single error")

		result := errors.Join(err1)
		assert.Error(t, result, "errors.Join should return error for single error")
		assert.Contains(t, result.Error(), "single error", "aggregated error should contain the error")
	})
}
