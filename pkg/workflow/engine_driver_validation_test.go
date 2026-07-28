//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidateEngineScriptFilename tests the validateEngineScriptFilename helper directly.
// The function validates that a script name is a safe Node.js basename (no paths, safe chars,
// correct extension). It is also exercised indirectly by TestValidateEngineHarnessScript.
func TestValidateEngineScriptFilename_ValidInputs(t *testing.T) {
	tests := []struct {
		name       string
		scriptName string
	}{
		{name: "simple js", scriptName: "script.js"},
		{name: "simple cjs", scriptName: "driver.cjs"},
		{name: "simple mjs", scriptName: "index.mjs"},
		{name: "hyphenated name", scriptName: "my-driver.js"},
		{name: "underscore name", scriptName: "my_script.js"},
		{name: "dotted name", scriptName: "a.b.js"},
		{name: "uppercase", scriptName: "MyScript.js"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEngineScriptFilename("engine.harness", tt.scriptName)
			require.NoError(t, err, "expected %q to be valid", tt.scriptName)
		})
	}
}

func TestValidateEngineScriptFilename_InvalidInputs(t *testing.T) {
	tests := []struct {
		name        string
		scriptName  string
		errContains string
	}{
		{
			name:        "leading whitespace",
			scriptName:  " script.js",
			errContains: "leading/trailing whitespace",
		},
		{
			name:        "trailing whitespace",
			scriptName:  "script.js ",
			errContains: "leading/trailing whitespace",
		},
		{
			name:        "absolute path",
			scriptName:  "/tmp/script.js",
			errContains: "safe basename",
		},
		{
			name:        "path separator slash",
			scriptName:  "subdir/script.js",
			errContains: "safe basename",
		},
		{
			name:        "path separator backslash",
			scriptName:  `sub\script.js`,
			errContains: "safe basename",
		},
		{
			name:        "directory traversal",
			scriptName:  "../script.js",
			errContains: "safe basename",
		},
		{
			name:        "unsupported extension py",
			scriptName:  "script.py",
			errContains: ".js, .cjs, or .mjs",
		},
		{
			name:        "unsupported extension ts",
			scriptName:  "script.ts",
			errContains: ".js, .cjs, or .mjs",
		},
		{
			name:        "no extension",
			scriptName:  "script",
			errContains: ".js, .cjs, or .mjs",
		},
		{
			name:        "shell metacharacter dollar",
			scriptName:  "$script.js",
			errContains: "safe basename",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEngineScriptFilename("engine.harness", tt.scriptName)
			require.Error(t, err, "expected %q to be invalid", tt.scriptName)
			assert.Contains(t, err.Error(), tt.errContains)
		})
	}
}

func TestValidateEngineHarnessScript(t *testing.T) {
	tests := []struct {
		name        string
		workflow    *WorkflowData
		expectError bool
		errorSubstr string
	}{
		{
			name: "no engine config",
			workflow: &WorkflowData{
				EngineConfig: nil,
			},
			expectError: false,
		},
		{
			name: "no harness configured",
			workflow: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot"},
			},
			expectError: false,
		},
		{
			name: "valid cjs harness on copilot",
			workflow: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot", HarnessScript: "custom_harness.cjs"},
			},
			expectError: false,
		},
		{
			name: "valid mjs harness on copilot",
			workflow: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot", HarnessScript: "custom_harness.mjs"},
			},
			expectError: false,
		},
		{
			name: "invalid extension",
			workflow: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot", HarnessScript: "harness.sh"},
			},
			expectError: true,
			errorSubstr: "must be a Node.js script",
		},
		{
			name: "harness configured for any engine",
			workflow: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "claude", HarnessScript: "driver.cjs"},
			},
			expectError: false,
		},
		{
			name: "invalid path traversal",
			workflow: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot", HarnessScript: "../driver.cjs"},
			},
			expectError: true,
			errorSubstr: "safe basename",
		},
		{
			name: "invalid path separator",
			workflow: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot", HarnessScript: "nested/driver.cjs"},
			},
			expectError: true,
			errorSubstr: "safe basename",
		},
		{
			name: "invalid shell metacharacter",
			workflow: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot", HarnessScript: "driver;rm -rf /.cjs"},
			},
			expectError: true,
			errorSubstr: "safe basename",
		},
		{
			name: "invalid leading whitespace",
			workflow: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot", HarnessScript: " driver.cjs"},
			},
			expectError: true,
			errorSubstr: "leading/trailing whitespace",
		},
		{
			name: "invalid leading hyphen",
			workflow: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot", HarnessScript: "-driver.cjs"},
			},
			expectError: true,
			errorSubstr: "safe basename",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler()
			err := compiler.validateEngineHarnessScript(tt.workflow)

			if tt.expectError {
				require.Error(t, err, "Expected validation error")
				if tt.errorSubstr != "" {
					require.ErrorContains(t, err, tt.errorSubstr, "Expected error substring mismatch")
				}
				return
			}

			assert.NoError(t, err, "Expected harness validation to pass")
		})
	}
}

func TestValidateEngineCopilotSDKDriver_Copilot(t *testing.T) {
	tests := []struct {
		name        string
		workflow    *WorkflowData
		expectError bool
		errorSubstr string
	}{
		{
			name: "no engine config",
			workflow: &WorkflowData{
				EngineConfig: nil,
			},
			expectError: false,
		},
		{
			name: "no copilot sdk driver configured",
			workflow: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot"},
			},
			expectError: false,
		},
		{
			name: "valid cjs copilot sdk driver",
			workflow: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot", Driver: "custom_copilot_sdk_driver.cjs"},
			},
			expectError: false,
		},
		{
			name: "valid mjs copilot sdk driver",
			workflow: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot", Driver: "custom_copilot_sdk_driver.mjs"},
			},
			expectError: false,
		},
		{
			name: "invalid extension",
			workflow: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot", Driver: "driver.sh"},
			},
			expectError: true,
			errorSubstr: "unsupported extension",
		},
		{
			name: "invalid path traversal",
			workflow: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot", Driver: "../driver.cjs"},
			},
			expectError: true,
			errorSubstr: "relative path",
		},
		{
			name: "invalid path separator leading slash",
			workflow: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot", Driver: "/abs/driver.cjs"},
			},
			expectError: true,
			errorSubstr: "relative path",
		},
		{
			name: "valid relative path driver",
			workflow: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot", Driver: "nested/driver.cjs"},
			},
			expectError: false,
		},
		{
			name: "valid github drivers path driver",
			workflow: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot", Driver: ".github/drivers/my_driver.py"},
			},
			expectError: false,
		},
		{
			name: "invalid shell metacharacter",
			workflow: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot", Driver: "driver;rm -rf /.cjs"},
			},
			expectError: true,
			errorSubstr: "metacharacter",
		},
		{
			name: "invalid leading whitespace",
			workflow: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot", Driver: " driver.cjs"},
			},
			expectError: true,
			errorSubstr: "leading/trailing whitespace",
		},
		{
			name: "invalid leading hyphen",
			workflow: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot", Driver: "-driver.cjs"},
			},
			expectError: true,
			errorSubstr: "metacharacter",
		},
		{
			name: "valid python driver",
			workflow: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot", Driver: "my_driver.py"},
			},
			expectError: false,
		},
		{
			name: "valid typescript driver",
			workflow: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot", Driver: "my_driver.ts"},
			},
			expectError: false,
		},
		{
			name: "valid mts typescript driver",
			workflow: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot", Driver: "my_driver.mts"},
			},
			expectError: false,
		},
		{
			name: "valid ruby driver",
			workflow: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot", Driver: "my_driver.rb"},
			},
			expectError: false,
		},
		{
			name: "valid arbitrary command no extension",
			workflow: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot", Driver: "my-copilot-driver"},
			},
			expectError: false,
		},
		{
			name: "valid arbitrary command underscore no extension",
			workflow: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot", Driver: "my_driver"},
			},
			expectError: false,
		},
		{
			name: "invalid java extension",
			workflow: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot", Driver: "driver.java"},
			},
			expectError: true,
			errorSubstr: "unsupported extension",
		},
		{
			name: "invalid consecutive slashes",
			workflow: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot", Driver: ".github//drivers/driver.py"},
			},
			expectError: true,
			errorSubstr: "empty path segments",
		},
		{
			name: "invalid trailing slash",
			workflow: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot", Driver: ".github/drivers/"},
			},
			expectError: true,
			errorSubstr: "empty path segments",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler()
			err := compiler.validateEngineDriver(tt.workflow)

			if tt.expectError {
				require.Error(t, err, "Expected validation error")
				if tt.errorSubstr != "" {
					require.ErrorContains(t, err, tt.errorSubstr, "Expected error substring mismatch")
				}
				return
			}

			assert.NoError(t, err, "Expected driver validation to pass")
		})
	}
}
