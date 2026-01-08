package workflow

import (
	"testing"
)

// TestCopilotEngineDetectsRealWorldErrors tests error detection using actual log patterns from GitHub Actions run 18182120048
func TestCopilotEngineDetectsRealWorldErrors(t *testing.T) {
	engine := NewCopilotEngine()
	patterns := engine.GetErrorPatterns()

	// Skip if engine has no patterns (patterns are now in JavaScript)
	if len(patterns) == 0 {
		t.Skip("Engine has no Go patterns - patterns are now defined in JavaScript")
	}

	// Actual log content from the problematic workflow run
	testLog := `✓ Run make fmt to format code
   $ make fmt
   ↪ 2 lines...
●

   Good! Formatting completed. Now let's run lint to check for any issues:

✓ Run make lint to check for linting issues
   $ make lint
   ↪ 10 lines...
●

   Excellent! No linting issues found. Now let's recompile workflows:

✓ Run make recompile to recompile workflows
   $ make recompile
   ↪ 223 lines...
●

   Great! Workflows recompiled successfully. Now let's run tests to ensure nothing is broken:

✓ Run make test to run all tests
   $ make test
   ↪ 170 lines...
●

   Perfect! All tests passed. Now let's check if there are any changes to commit:

✓ Check git status for changes
   $ git status
   ↪ 5 lines...
●

   Excellent! The codebase is already completely tidy. Let me verify this one more time by running
   'make agent-finish' to ensure everything is in order:

✓ Run make agent-finish to ensure everything is correct
   $ make agent-finish
   ↪ 219 lines...
●

   There's a minor issue with the JavaScript tests failing because vitest is not found, but this is
    not something we introduced - it's a pre-existing issue in the environment. Let me check if
   this is a development dependency issue:

✗ Check if vitest needs to be installed
   $ cd pkg/workflow/js && npm install && cd -
   Permission denied and could not request permission from user
●

   Let me try a different approach - check the dev dependencies:

✗ Install dev dependencies
   $ cd pkg/workflow/js && npm ci --include=dev && cd -
   Permission denied and could not request permission from user
●

   Let me check if we can run the deps-dev target from the Makefile:

✗ Install dev dependencies via Makefile
   $ make deps-dev 2>&1 | head -50
   Permission denied and could not request permission from user`

	// Test that errors are detected in the log
	errors := CountErrorsAndWarningsWithPatterns(testLog, patterns)

	errorCount := CountErrors(errors)
	warningCount := CountWarnings(errors)

	// Log all detected errors and warnings for debugging
	t.Logf("Detected %d errors and %d warnings in the workflow log", errorCount, warningCount)
	for i, err := range errors {
		t.Logf("  %d. [%s] Line %d: %s", i+1, err.Type, err.Line, err.Message)
	}

	if errorCount == 0 {
		t.Error("Expected to detect errors in the log, but found none")
		t.Log("Log content contains:")
		t.Log("  - ✗ symbols indicating failed commands")
		t.Log("These should be detected as errors by the Copilot error patterns")
	}

	// We should detect at least 3 errors (3 failed commands with ✗ symbol)
	// Note: "Permission denied and could not request permission from user" should be warnings, not errors
	if errorCount < 3 {
		t.Errorf("Expected at least 3 errors, but detected %d", errorCount)
	}

	// We should detect warnings for "Permission denied" messages
	if warningCount < 3 {
		t.Errorf("Expected at least 3 warnings for 'Permission denied' messages, but detected %d", warningCount)
	}
}

// TestCopilotEngineDetectsCommandNotFoundInLogs tests detection of command not found errors
func TestCopilotEngineDetectsCommandNotFoundInLogs(t *testing.T) {
	engine := NewCopilotEngine()
	patterns := engine.GetErrorPatterns()

	// Skip if engine has no patterns (patterns are now in JavaScript)
	if len(patterns) == 0 {
		t.Skip("Engine has no Go patterns - patterns are now defined in JavaScript")
	}

	// Simulate log with command not found error
	testLog := `Running make test
sh: 1: vitest: not found
make: *** [Makefile:100: test-js] Error 127
make: *** Waiting for unfinished jobs....`

	errors := CountErrorsAndWarningsWithPatterns(testLog, patterns)

	errorCount := CountErrors(errors)
	if errorCount == 0 {
		t.Error("Expected to detect 'vitest: not found' error, but found none")
	}

	t.Logf("Successfully detected %d errors including command not found", errorCount)
}

// TestCopilotEngineDetectsNodeModuleNotFound tests detection of Node.js module errors
func TestCopilotEngineDetectsNodeModuleNotFound(t *testing.T) {
	engine := NewCopilotEngine()
	patterns := engine.GetErrorPatterns()

	// Skip if engine has no patterns (patterns are now in JavaScript)
	if len(patterns) == 0 {
		t.Skip("Engine has no Go patterns - patterns are now defined in JavaScript")
	}

	testLog := `node:internal/modules/cjs/loader:1147
  throw err;
  ^

Error: Cannot find module 'vitest'
Require stack:
- /home/runner/work/gh-aw/gh-aw/pkg/workflow/js/test.js
    at Module._resolveFilename (node:internal/modules/cjs/loader:1144:15)`

	errors := CountErrorsAndWarningsWithPatterns(testLog, patterns)

	errorCount := CountErrors(errors)
	if errorCount == 0 {
		t.Error("Expected to detect 'Cannot find module' error, but found none")
	}

	t.Logf("Successfully detected %d errors including module not found", errorCount)
}
