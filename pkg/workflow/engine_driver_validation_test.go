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
