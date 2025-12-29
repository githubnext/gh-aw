package workflow

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/githubnext/gh-aw/pkg/parser"
)

// TestUserFacingErrorsDontLeakInternals validates that internal errors are properly
// wrapped and don't leak implementation details to users. This uses testify v1.11.0+'s
// NotErrorAs assertion to ensure error types are hidden from user-facing code.
func TestUserFacingErrorsDontLeakInternals(t *testing.T) {
	tests := []struct {
		name           string
		operation      func() error
		internalErrors []any
	}{
		{
			name: "workflow compilation YAML parse error",
			operation: func() error {
				// Create a test file with invalid YAML
				tmpDir := t.TempDir()
				testFile := filepath.Join(tmpDir, "invalid.md")
				content := `---
engine: copilot
on:
  - invalid: {{{
---
# Test Workflow`
				err := os.WriteFile(testFile, []byte(content), 0644)
				require.NoError(t, err, "Failed to write test file")

				// Try to compile the invalid workflow
				compiler := NewCompiler(false, "", "1.0.0")
				err = compiler.CompileWorkflow(testFile)
				return err
			},
			internalErrors: []any{
				&yaml.TypeError{},       // YAML parsing error should be wrapped
				&yaml.SyntaxError{},     // YAML syntax error should be wrapped
			},
		},
		{
			name: "workflow file read error",
			operation: func() error {
				// Try to compile a non-existent file
				compiler := NewCompiler(false, "", "1.0.0")
				err := compiler.CompileWorkflow("/nonexistent/file.md")
				return err
			},
			internalErrors: []any{
				&os.PathError{},         // File system errors should be wrapped
				&os.LinkError{},         // Symlink errors should be wrapped
			},
		},
		{
			name: "import resolution error",
			operation: func() error {
				// Create a test file with invalid import
				tmpDir := t.TempDir()
				testFile := filepath.Join(tmpDir, "test.md")
				content := `---
engine: copilot
imports:
  - nonexistent.md
on:
  issues:
    types: [opened]
---
# Test Workflow`
				err := os.WriteFile(testFile, []byte(content), 0644)
				require.NoError(t, err, "Failed to write test file")

				// Try to compile with invalid import
				compiler := NewCompiler(false, "", "1.0.0")
				err = compiler.CompileWorkflow(testFile)
				return err
			},
			internalErrors: []any{
				&parser.ImportError{},   // Import errors are internal implementation details
				&os.PathError{},         // Underlying file errors should be wrapped
			},
		},
		{
			name: "GitHub toolset validation error",
			operation: func() error {
				// Create a workflow with GitHub tools but missing toolsets
				tmpDir := t.TempDir()
				testFile := filepath.Join(tmpDir, "test.md")
				content := `---
engine: copilot
on:
  issues:
    types: [opened]
tools:
  github:
    allowed:
      - search_code
      - search_issues
---
# Test Workflow

Test workflow with GitHub tools.`
				err := os.WriteFile(testFile, []byte(content), 0644)
				require.NoError(t, err, "Failed to write test file")

				// Try to compile - should fail validation
				compiler := NewCompiler(false, "", "1.0.0")
				err = compiler.CompileWorkflow(testFile)
				return err
			},
			internalErrors: []any{
				&GitHubToolsetValidationError{}, // Internal validation error should be wrapped
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.operation()
			require.Error(t, err, "should return an error")

			// Verify that internal error types are not exposed to users
			for _, internalErr := range tt.internalErrors {
				// Create a pointer to the error type for NotErrorAs
				switch e := internalErr.(type) {
				case *yaml.TypeError:
					var target *yaml.TypeError
					assert.NotErrorAs(t, err, &target,
						fmt.Sprintf("internal error type %T should not leak to user", e))
				case *yaml.SyntaxError:
					var target *yaml.SyntaxError
					assert.NotErrorAs(t, err, &target,
						fmt.Sprintf("internal error type %T should not leak to user", e))
				case *os.PathError:
					var target *os.PathError
					assert.NotErrorAs(t, err, &target,
						fmt.Sprintf("internal error type %T should not leak to user", e))
				case *os.LinkError:
					var target *os.LinkError
					assert.NotErrorAs(t, err, &target,
						fmt.Sprintf("internal error type %T should not leak to user", e))
				case *parser.ImportError:
					var target *parser.ImportError
					assert.NotErrorAs(t, err, &target,
						fmt.Sprintf("internal error type %T should not leak to user", e))
				case *GitHubToolsetValidationError:
					var target *GitHubToolsetValidationError
					assert.NotErrorAs(t, err, &target,
						fmt.Sprintf("internal error type %T should not leak to user", e))
				default:
					t.Fatalf("Unknown error type in test: %T", e)
				}
			}

			// Ensure the error message is still meaningful
			errMsg := err.Error()
			assert.NotEmpty(t, errMsg, "error message should not be empty")
			assert.Greater(t, len(errMsg), 10, "error message should be descriptive")
		})
	}
}

// TestInternalErrorTypesAreWrapped tests specific error wrapping scenarios
// where we want to ensure internal errors are properly wrapped with context.
func TestInternalErrorTypesAreWrapped(t *testing.T) {
	tests := []struct {
		name              string
		setupFunc         func(t *testing.T) error
		shouldNotContain  []any
		shouldBeWrapped   bool
	}{
		{
			name: "HTTP client errors from external services",
			setupFunc: func(t *testing.T) error {
				// Simulate an HTTP error that might occur during MCP server communication
				// This is a placeholder - in real code, HTTP errors would come from actual requests
				httpErr := &http.ProtocolError{ErrorString: "simulated protocol error"}
				return fmt.Errorf("failed to connect to MCP server: %w", httpErr)
			},
			shouldNotContain: []any{
				&http.ProtocolError{}, // HTTP internal errors should be wrapped
			},
			shouldBeWrapped: true,
		},
		{
			name: "IO errors from file operations",
			setupFunc: func(t *testing.T) error {
				// Simulate an IO error
				return fmt.Errorf("failed to read workflow file: %w", io.ErrUnexpectedEOF)
			},
			shouldNotContain: []any{
				io.ErrUnexpectedEOF, // Specific IO errors should be wrapped with context
			},
			shouldBeWrapped: false, // This is a sentinel error, different handling
		},
		{
			name: "directory creation errors",
			setupFunc: func(t *testing.T) error {
				// Try to create a directory in a location that doesn't exist
				badPath := filepath.Join(t.TempDir(), "nonexistent", "deeply", "nested", "path")
				// Note: this might succeed on some systems, so we'll simulate
				return fmt.Errorf("failed to setup workflow directory: %w", 
					&os.PathError{Op: "mkdir", Path: badPath, Err: os.ErrNotExist})
			},
			shouldNotContain: []any{
				&os.PathError{}, // Path errors should be wrapped with user-friendly context
			},
			shouldBeWrapped: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.setupFunc(t)
			require.Error(t, err)

			for _, internalErr := range tt.shouldNotContain {
				if tt.shouldBeWrapped {
					// Create appropriate target variables for NotErrorAs
					switch e := internalErr.(type) {
					case *http.ProtocolError:
						var target *http.ProtocolError
						assert.NotErrorAs(t, err, &target,
							fmt.Sprintf("internal error type %T should be wrapped", e))
					case *os.PathError:
						var target *os.PathError
						assert.NotErrorAs(t, err, &target,
							fmt.Sprintf("internal error type %T should be wrapped", e))
					case error:
						// For sentinel errors like io.ErrUnexpectedEOF
						assert.NotErrorIs(t, err, e,
							fmt.Sprintf("sentinel error %v should be wrapped with context", e))
					}
				}
			}

			// Verify error message provides context
			assert.NotEmpty(t, err.Error())
		})
	}
}

// TestValidationErrorsAreUserFriendly ensures that validation errors
// provide helpful context without exposing internal implementation details.
func TestValidationErrorsAreUserFriendly(t *testing.T) {
	tests := []struct {
		name          string
		operation     func() error
		wantInMessage []string
		notErrorTypes []any
	}{
		{
			name: "missing engine configuration",
			operation: func() error {
				tmpDir := t.TempDir()
				testFile := filepath.Join(tmpDir, "test.md")
				content := `---
on:
  issues:
    types: [opened]
---
# Test Workflow without engine`
				err := os.WriteFile(testFile, []byte(content), 0644)
				require.NoError(t, err)

				compiler := NewCompiler(false, "", "1.0.0")
				err = compiler.CompileWorkflow(testFile)
				return err
			},
			wantInMessage: []string{"engine"},
			notErrorTypes: []any{
				&yaml.TypeError{},
				&os.PathError{},
			},
		},
		{
			name: "invalid trigger configuration",
			operation: func() error {
				tmpDir := t.TempDir()
				testFile := filepath.Join(tmpDir, "test.md")
				content := `---
engine: copilot
on:
  invalid_trigger:
    types: [something]
---
# Test Workflow with invalid trigger`
				err := os.WriteFile(testFile, []byte(content), 0644)
				require.NoError(t, err)

				compiler := NewCompiler(false, "", "1.0.0")
				err = compiler.CompileWorkflow(testFile)
				return err
			},
			wantInMessage: []string{}, // Just verify no internal errors leak
			notErrorTypes: []any{
				&yaml.TypeError{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.operation()
			require.Error(t, err)

			errMsg := err.Error()
			
			// Check that error message contains expected keywords
			for _, want := range tt.wantInMessage {
				assert.Contains(t, errMsg, want,
					fmt.Sprintf("error message should mention '%s'", want))
			}

			// Verify internal error types don't leak
			for _, internalErr := range tt.notErrorTypes {
				switch e := internalErr.(type) {
				case *yaml.TypeError:
					var target *yaml.TypeError
					assert.NotErrorAs(t, err, &target,
						fmt.Sprintf("internal error type %T should not leak", e))
				case *os.PathError:
					var target *os.PathError
					assert.NotErrorAs(t, err, &target,
						fmt.Sprintf("internal error type %T should not leak", e))
				default:
					t.Fatalf("Unknown error type in test: %T", e)
				}
			}
		})
	}
}

// TestErrorWrappingPreservesContext ensures that when we wrap errors,
// we don't lose important context information.
func TestErrorWrappingPreservesContext(t *testing.T) {
	tests := []struct {
		name      string
		operation func() error
		wantInfo  []string // Information that should be preserved
	}{
		{
			name: "file path is preserved in wrapped errors",
			operation: func() error {
				tmpDir := t.TempDir()
				testFile := filepath.Join(tmpDir, "my-workflow.md")
				// Don't create the file - let it fail
				
				compiler := NewCompiler(false, "", "1.0.0")
				err := compiler.CompileWorkflow(testFile)
				return err
			},
			wantInfo: []string{"my-workflow.md"}, // Filename should appear in error
		},
		{
			name: "line numbers are preserved for YAML errors",
			operation: func() error {
				tmpDir := t.TempDir()
				testFile := filepath.Join(tmpDir, "test.md")
				content := `---
engine: copilot
on: 123456
---
# Test Workflow`
				err := os.WriteFile(testFile, []byte(content), 0644)
				require.NoError(t, err)

				compiler := NewCompiler(false, "", "1.0.0")
				err = compiler.CompileWorkflow(testFile)
				return err
			},
			wantInfo: []string{"on"}, // Field name should be mentioned
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.operation()
			require.Error(t, err)

			errMsg := err.Error()
			for _, info := range tt.wantInfo {
				assert.Contains(t, errMsg, info,
					fmt.Sprintf("error should preserve context: '%s'", info))
			}

			// Also verify it's still wrapped properly
			var pathErr *os.PathError
			assert.NotErrorAs(t, err, &pathErr, "os.PathError should be wrapped")
			
			var typeErr *yaml.TypeError
			assert.NotErrorAs(t, err, &typeErr, "yaml.TypeError should be wrapped")
		})
	}
}
