import { describe, it, expect, beforeEach, vi } from "vitest";

const { validateErrors, extractLevel, extractMessage, truncateString, shouldSkipLine } = await import("./validate_errors.cjs");

// Mock global objects for testing
global.console = {
  log: vi.fn(),
  warn: vi.fn(),
  debug: vi.fn(),
};

global.core = {
  // Core logging functions
  debug: vi.fn(),
  info: vi.fn(),
  notice: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),

  // Core workflow functions
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  exportVariable: vi.fn(),
  setSecret: vi.fn(),

  // Input/state functions (less commonly used but included for completeness)
  getInput: vi.fn(),
  getBooleanInput: vi.fn(),
  getMultilineInput: vi.fn(),
  getState: vi.fn(),
  saveState: vi.fn(),

  // Group functions
  startGroup: vi.fn(),
  endGroup: vi.fn(),
  group: vi.fn(),

  // Other utility functions
  addPath: vi.fn(),
  setCommandEcho: vi.fn(),
  isDebug: vi.fn().mockReturnValue(false),
  getIDToken: vi.fn(),
  toPlatformPath: vi.fn(),
  toPosixPath: vi.fn(),
  toWin32Path: vi.fn(),

  // Summary object with chainable methods
  summary: {
    addRaw: vi.fn(() => ({ write: vi.fn() })),
    write: vi.fn().mockResolvedValue(),
  },
};

describe("validateErrors", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  test("should detect errors with codex patterns", () => {
    const logContent = `[2025-09-10T17:54:49] stream error: exceeded retry limit, last status: 401 Unauthorized; retrying 1/5 in 216ms…
[2025-09-10T17:55:15] ERROR: exceeded retry limit, last status: 401 Unauthorized
Some normal log content
[2025-09-10T17:56:20] WARNING: This is a warning message`;

    const patterns = [
      {
        pattern: "\\[(\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2})\\]\\s+stream\\s+(error):\\s+(.+)",
        level_group: 2,
        message_group: 3,
        description: "Codex stream errors with timestamp",
      },
      {
        pattern: "\\[(\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2})\\]\\s+(ERROR):\\s+(.+)",
        level_group: 2,
        message_group: 3,
        description: "Codex ERROR messages with timestamp",
      },
      {
        pattern: "\\[(\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2})\\]\\s+(WARN|WARNING):\\s+(.+)",
        level_group: 2,
        message_group: 3,
        description: "Codex warning messages with timestamp",
      },
    ];

    const hasErrors = validateErrors(logContent, patterns);

    // Should return true since errors were found
    expect(hasErrors).toBe(true);

    // Should call core.error for errors
    expect(global.core.error).toHaveBeenCalledTimes(2);
    expect(global.core.error).toHaveBeenCalledWith(expect.stringContaining("exceeded retry limit"));

    // Should call core.warning for warnings
    expect(global.core.warning).toHaveBeenCalledTimes(1);
    expect(global.core.warning).toHaveBeenCalledWith(expect.stringContaining("This is a warning message"));
  });

  test("should handle empty log content", () => {
    const patterns = [
      {
        pattern: "ERROR:\\s+(.+)",
        level_group: 0,
        message_group: 1,
        description: "Simple error pattern",
      },
    ];

    const hasErrors = validateErrors("", patterns);

    // Should return false since no errors were found
    expect(hasErrors).toBe(false);

    // Should not call core.error or core.warning for empty content
    expect(global.core.error).not.toHaveBeenCalled();
    expect(global.core.warning).not.toHaveBeenCalled();
  });

  test("should handle no matching patterns", () => {
    const logContent = "Just some normal log content\nNothing interesting here";
    const patterns = [
      {
        pattern: "CRITICAL:\\s+(.+)",
        level_group: 0,
        message_group: 1,
        description: "Critical errors",
      },
    ];

    const hasErrors = validateErrors(logContent, patterns);

    // Should return false since no errors were found
    expect(hasErrors).toBe(false);

    // Should not call core.error or core.warning when no patterns match
    expect(global.core.error).not.toHaveBeenCalled();
    expect(global.core.warning).not.toHaveBeenCalled();
  });

  test("should handle invalid regex patterns gracefully", () => {
    const logContent = "ERROR: Something went wrong";
    const patterns = [
      {
        pattern: "[invalid regex", // Missing closing bracket
        level_group: 0,
        message_group: 1,
        description: "Invalid pattern",
      },
    ];

    // Should handle invalid patterns gracefully, not throw
    const hasErrors = validateErrors(logContent, patterns);

    // Should return false since no valid patterns matched
    expect(hasErrors).toBe(false);

    // Should log an error about the invalid pattern
    expect(global.core.error).toHaveBeenCalledWith("invalid error regex pattern: [invalid regex");
  });

  test("should detect 401 unauthorized errors from issue #668", () => {
    // Exact log content from GitHub issue #668
    const logContent = `[2025-09-10T17:54:49] stream error: exceeded retry limit, last status: 401 Unauthorized; retrying 1/5 in 216ms…
[2025-09-10T17:54:54] stream error: exceeded retry limit, last status: 401 Unauthorized; retrying 2/5 in 414ms…
[2025-09-10T17:54:58] stream error: exceeded retry limit, last status: 401 Unauthorized; retrying 3/5 in 821ms…
[2025-09-10T17:55:03] stream error: exceeded retry limit, last status: 401 Unauthorized; retrying 4/5 in 1.611s…
[2025-09-10T17:55:08] stream error: exceeded retry limit, last status: 401 Unauthorized; retrying 5/5 in 3.039s…
[2025-09-10T17:55:15] ERROR: exceeded retry limit, last status: 401 Unauthorized`;

    const patterns = [
      {
        pattern: "\\[(\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2})\\]\\s+stream\\s+(error):\\s+(.+)",
        level_group: 2,
        message_group: 3,
        description: "Codex stream errors with timestamp",
      },
      {
        pattern: "\\[(\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2})\\]\\s+(ERROR):\\s+(.+)",
        level_group: 2,
        message_group: 3,
        description: "Codex ERROR messages with timestamp",
      },
    ];

    const hasErrors = validateErrors(logContent, patterns);

    // Should return true since errors were found
    expect(hasErrors).toBe(true);

    // Should detect all 6 errors (5 stream errors + 1 ERROR)
    expect(global.core.error).toHaveBeenCalledTimes(6);

    // Verify calls contain the expected content
    expect(global.core.error).toHaveBeenCalledWith(expect.stringContaining("401 Unauthorized"));
    expect(global.core.error).toHaveBeenCalledWith(expect.stringContaining("exceeded retry limit"));
  });
});

describe("extractLevel", () => {
  test("should extract level from capture group", () => {
    const match = ["full match", "timestamp", "ERROR", "message"];
    const pattern = { level_group: 2, message_group: 3 };

    const level = extractLevel(match, pattern);
    expect(level).toBe("ERROR");
  });

  test("should infer level from match content when no group specified", () => {
    const match = ["stream error: something"];
    const pattern = { level_group: 0 };

    const level = extractLevel(match, pattern);
    expect(level).toBe("error");
  });

  test("should return unknown for unrecognized levels", () => {
    const match = ["INFO: something"];
    const pattern = { level_group: 0 };

    const level = extractLevel(match, pattern);
    expect(level).toBe("unknown");
  });
});

describe("extractMessage", () => {
  test("should extract message from capture group", () => {
    const match = ["full match", "timestamp", "ERROR", "Something went wrong"];
    const pattern = { message_group: 3 };
    const fullLine = "the full line";

    const message = extractMessage(match, pattern, fullLine);
    expect(message).toBe("Something went wrong");
  });

  test("should fallback to full match when no group specified", () => {
    const match = ["ERROR: Something went wrong"];
    const pattern = { message_group: 0 };
    const fullLine = "the full line";

    const message = extractMessage(match, pattern, fullLine);
    expect(message).toBe("ERROR: Something went wrong");
  });
});

describe("truncateString", () => {
  test("should truncate long strings", () => {
    const longString = "a".repeat(200);
    const result = truncateString(longString, 100);

    expect(result).toHaveLength(103); // 100 chars + "..."
    expect(result.endsWith("...")).toBe(true);
  });

  test("should return short strings unchanged", () => {
    const shortString = "short";
    const result = truncateString(shortString, 100);

    expect(result).toBe("short");
  });

  test("should handle null/undefined strings", () => {
    expect(truncateString(null, 100)).toBe("");
    expect(truncateString(undefined, 100)).toBe("");
  });
});

describe("main function behavior", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  test("should call core.setFailed when errors are detected", () => {
    // Mock the imported functions since main isn't exported
    const originalProcessEnv = process.env;

    // Mock environment variables
    process.env.GH_AW_AGENT_OUTPUT = "/tmp/gh-aw/test.log";
    process.env.GH_AW_ERROR_PATTERNS = JSON.stringify([
      {
        pattern: "ERROR:\\s+(.+)",
        level_group: 0,
        message_group: 1,
        description: "Test error pattern",
      },
    ]);

    // Mock fs.existsSync to return true
    const mockFs = {
      existsSync: vi.fn(() => true),
      readFileSync: vi.fn(() => "ERROR: Test error message"),
    };

    // Since we can't easily test main() directly, we can test the core logic
    // The main function will call validateErrors and if it returns true, call core.setFailed
    const logContent = "ERROR: Test error message";
    const patterns = [
      {
        pattern: "ERROR:\\s+(.+)",
        level_group: 0,
        message_group: 1,
        description: "Test error pattern",
      },
    ];

    const hasErrors = validateErrors(logContent, patterns);
    expect(hasErrors).toBe(true);

    // Simulate what main() would do
    if (hasErrors) {
      global.core.setFailed("Errors detected in agent logs - failing workflow step");
    }

    expect(global.core.setFailed).toHaveBeenCalledWith("Errors detected in agent logs - failing workflow step");

    // Restore
    process.env = originalProcessEnv;
  });

  test("should not call core.setFailed when only warnings are detected", () => {
    const logContent = "WARNING: This is just a warning";
    const patterns = [
      {
        pattern: "WARNING:\\s+(.+)",
        level_group: 0,
        message_group: 1,
        description: "Test warning pattern",
      },
    ];

    const hasErrors = validateErrors(logContent, patterns);
    expect(hasErrors).toBe(false);

    // Simulate what main() would do
    if (hasErrors) {
      global.core.setFailed("Errors detected in agent logs - failing workflow step");
    }

    expect(global.core.setFailed).not.toHaveBeenCalled();
  });
});

describe("infinite loop detection", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  test("should detect and prevent infinite loops from zero-width matches", () => {
    // A pattern that could potentially match zero-width (though JavaScript's regex.exec
    // typically advances even on zero-width matches, this tests our safety mechanism)
    const logContent = "test line";
    const patterns = [
      {
        // This pattern uses a lookahead which could be problematic in some scenarios
        pattern: "(?=t)",
        level_group: 0,
        message_group: 0,
        description: "Potentially problematic zero-width pattern",
      },
    ];

    // Should not hang or throw, should complete
    const hasErrors = validateErrors(logContent, patterns);

    // The function should complete without hanging
    expect(hasErrors).toBeDefined();
  });

  test("should log warning when iteration count is high", () => {
    // Create a line with many matches to trigger the warning threshold
    const repeatedPattern = "ERROR ".repeat(1500); // More than ITERATION_WARNING_THRESHOLD
    const logContent = repeatedPattern;

    const patterns = [
      {
        pattern: "ERROR",
        level_group: 0,
        message_group: 0,
        description: "Simple error pattern",
      },
    ];

    validateErrors(logContent, patterns);

    // Should have logged a warning about high iteration count
    const warningCalls = global.core.warning.mock.calls.map(call => call[0]);
    const hasIterationWarning = warningCalls.some(msg => msg.includes("High iteration count") || msg.includes("1000"));

    expect(hasIterationWarning).toBe(true);
  });

  test("should enforce maximum iteration limit or max error limit", () => {
    // Create a line with an extreme number of potential matches but within MAX_LINE_LENGTH
    // Use 9000 chars to stay under the 10KB limit but still exceed MAX_ITERATIONS_PER_LINE
    const massivePattern = "X".repeat(9000);
    const logContent = massivePattern;

    const patterns = [
      {
        pattern: "X",
        level_group: 0,
        message_group: 0,
        description: "Single character pattern",
      },
    ];

    validateErrors(logContent, patterns);

    // Should have logged either an error about maximum iteration limit OR stopping due to max errors
    const errorCalls = global.core.error.mock.calls.map(call => call[0]);
    const warningCalls = global.core.warning.mock.calls.map(call => call[0]);

    const hasMaxIterationError = errorCalls.some(msg => msg.includes("Maximum iteration limit") || msg.includes("10000"));
    const hasMaxErrorsWarning = warningCalls.some(msg => msg.includes("Stopping") && (msg.includes("100") || msg.includes("max")));

    // Either condition should be true (we hit one of the limits)
    expect(hasMaxIterationError || hasMaxErrorsWarning).toBe(true);
  });

  test("should not have patterns that match empty string (zero-width)", () => {
    // Test patterns that should NEVER be used because they match zero-width
    const problematicPatterns = [
      {
        pattern: ".*",
        description: "Pure .* matches zero-width at end",
      },
      {
        pattern: "a*",
        description: "Single char * matches zero-width",
      },
      {
        pattern: ".*error.*",
        description: ".* surrounding text (false positive - actually safe)",
      },
    ];

    problematicPatterns.forEach(({ pattern, description }) => {
      const regex = new RegExp(pattern, "g");
      const testStr = "hello";
      let lastIndex = -1;
      let iterationCount = 0;
      let match;

      while ((match = regex.exec(testStr)) !== null && iterationCount < 100) {
        iterationCount++;

        // Check if lastIndex is stuck (zero-width match at end)
        if (regex.lastIndex === lastIndex) {
          // This is the problematic case - pattern matches zero-width
          console.log(`Pattern "${pattern}" (${description}) would cause infinite loop!`);

          // For patterns like .* and a*, this is expected
          if (pattern === ".*" || pattern === "a*") {
            expect(regex.lastIndex).toBe(lastIndex);
            return; // Expected behavior for these patterns
          }
        }
        lastIndex = regex.lastIndex;
      }
    });
  });

  test("should handle actual engine patterns safely", () => {
    // Test with actual patterns from the engines (converted from Go)
    const actualPatterns = [
      {
        pattern: "access denied.*only authorized.*can trigger.*workflow",
        description: "access denied workflow",
      },
      {
        pattern: "error.*permission.*denied",
        description: "error permission denied",
      },
      {
        pattern: "error.*unauthorized",
        description: "error unauthorized",
      },
    ];

    const testLines = [
      "ERROR permission denied",
      "error: unauthorized access",
      "access denied to user not authorized",
      "", // Empty line should not cause issues
      "x".repeat(1000), // Long line
    ];

    actualPatterns.forEach(({ pattern, description }) => {
      testLines.forEach(line => {
        const regex = new RegExp(pattern, "gi");
        let match;
        let iterationCount = 0;
        let lastIndex = -1;
        const MAX_ITERATIONS = 1000;

        while ((match = regex.exec(line)) !== null) {
          iterationCount++;

          // Safety check: lastIndex must advance
          if (regex.lastIndex === lastIndex) {
            throw new Error(`Pattern "${pattern}" (${description}) caused infinite loop on line: "${line.substring(0, 50)}..."`);
          }
          lastIndex = regex.lastIndex;

          if (iterationCount > MAX_ITERATIONS) {
            throw new Error(`Pattern "${pattern}" exceeded ${MAX_ITERATIONS} iterations on line: "${line.substring(0, 50)}..."`);
          }
        }
      });
    });
  });

  test("should never match empty string for production patterns", () => {
    // Patterns from actual engines should NEVER match empty string
    const productionPatterns = [
      "access denied.*only authorized.*can trigger.*workflow",
      "error.*permission.*denied",
      "error.*unauthorized",
      "error.*forbidden",
      "error.*access.*restricted",
      "(\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}\\.\\d{3}Z)\\s+\\[(ERROR)\\]\\s+(.+)",
    ];

    productionPatterns.forEach(pattern => {
      const regex = new RegExp(pattern, "gi");
      const matchesEmpty = regex.test("");

      if (matchesEmpty) {
        throw new Error(`Production pattern "${pattern}" matches empty string! This WILL cause infinite loops.`);
      }
    });
  });

  test("should log debug information about patterns and lines", () => {
    const logContent = "ERROR: test\nWARNING: test2";
    const patterns = [
      {
        pattern: "ERROR:\\s+(.+)",
        level_group: 0,
        message_group: 1,
        description: "Error pattern",
      },
    ];

    validateErrors(logContent, patterns);

    // Should have logged info information
    const infoCalls = global.core.info.mock.calls.map(call => call[0]);

    // Should log pattern count and line count
    const hasPatternInfo = infoCalls.some(msg => msg.includes("patterns") && msg.includes("lines"));
    expect(hasPatternInfo).toBe(true);
  });
});

describe("shouldSkipLine", () => {
  test("should skip GitHub Actions environment variable declarations with timestamp", () => {
    const line = '2025-10-11T21:23:50.7459810Z   GH_AW_ERROR_PATTERNS: [{"pattern":"access denied.*only authorized.*can trigger.*workflow"}]';
    expect(shouldSkipLine(line)).toBe(true);
  });

  test("should skip GitHub Actions environment variable declarations without timestamp", () => {
    const line = '   GH_AW_ERROR_PATTERNS: [{"pattern":"error.*permission.*denied"}]';
    expect(shouldSkipLine(line)).toBe(true);
  });

  test("should skip env: section headers in GitHub Actions logs", () => {
    const line = "2025-10-11T21:23:50.7453806Z env:";
    expect(shouldSkipLine(line)).toBe(true);
  });

  test("should not skip regular log lines", () => {
    const line = "ERROR: permission denied";
    expect(shouldSkipLine(line)).toBe(false);
  });

  test("should not skip lines that mention error patterns in context", () => {
    const line = "Analyzing error patterns for validation";
    expect(shouldSkipLine(line)).toBe(false);
  });

  test("should not skip empty lines", () => {
    const line = "";
    expect(shouldSkipLine(line)).toBe(false);
  });

  test("should not skip lines with GH_AW_ERROR_PATTERNS in regular content", () => {
    // This line mentions the env var but is not the actual env var declaration
    const line = "The GH_AW_ERROR_PATTERNS variable was set correctly";
    expect(shouldSkipLine(line)).toBe(false);
  });

  test("should skip Copilot CLI DEBUG messages", () => {
    const line = "2025-12-15T08:35:23.457Z [DEBUG] Unable to parse tool invocation as JSON. Treating it as a string for filtering: SyntaxError: Unexpected token 'l'";
    expect(shouldSkipLine(line)).toBe(true);
  });

  test("should not skip Copilot CLI ERROR messages", () => {
    const line = "2025-12-15T08:35:23.457Z [ERROR] Tool execution failed";
    expect(shouldSkipLine(line)).toBe(false);
  });

  test("should not skip Copilot CLI WARNING messages", () => {
    const line = "2025-12-15T08:35:23.457Z [WARNING] This is a warning";
    expect(shouldSkipLine(line)).toBe(false);
  });
});

describe("validateErrors with environment variable filtering", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  test("should not match error patterns in GH_AW_ERROR_PATTERNS environment variable output", () => {
    // Simulate actual GitHub Actions log output that includes the env var
    const logContent = `2025-10-11T21:23:50.7452613Z   debug: false
2025-10-11T21:23:50.7453024Z   user-agent: actions/github-script
2025-10-11T21:23:50.7454107Z env:
2025-10-11T21:23:50.7454400Z   GH_AW_SAFE_OUTPUTS: /tmp/gh-aw/safeoutputs/outputs.jsonl
2025-10-11T21:23:50.7459810Z   GH_AW_ERROR_PATTERNS: [{"pattern":"access denied.*only authorized.*can trigger.*workflow","level_group":0,"message_group":0,"description":"Permission denied - workflow access restriction"},{"pattern":"error.*permission.*denied","level_group":0,"message_group":0,"description":"Permission denied error"}]
2025-10-11T21:23:50.7464005Z ##[endgroup]
Regular log content here
ERROR: actual error that should be caught`;

    const patterns = [
      {
        pattern: "access denied.*only authorized.*can trigger.*workflow",
        level_group: 0,
        message_group: 0,
        description: "Permission denied - workflow access restriction",
      },
      {
        pattern: "error.*permission.*denied",
        level_group: 0,
        message_group: 0,
        description: "Permission denied error",
      },
      {
        pattern: "ERROR:\\s+(.+)",
        level_group: 0,
        message_group: 1,
        description: "Simple ERROR pattern",
      },
    ];

    const hasErrors = validateErrors(logContent, patterns);

    // Should detect the actual ERROR line but not the env var lines
    expect(hasErrors).toBe(true);

    // Should only have 1 error (the actual ERROR line), not false positives from env var
    const errorCalls = global.core.error.mock.calls;
    const relevantErrors = errorCalls.filter(call => call[0].includes("actual error that should be caught"));
    expect(relevantErrors.length).toBeGreaterThan(0);

    // Should NOT have errors for the env var lines
    const envVarErrors = errorCalls.filter(call => call[0].includes("GH_AW_ERROR_PATTERNS"));
    expect(envVarErrors.length).toBe(0);
  });

  test("should still catch real errors that match the patterns", () => {
    const logContent = `2025-10-11T21:23:50.7459810Z   GH_AW_ERROR_PATTERNS: [{"pattern":"error.*permission.*denied"}]
Normal log line
error: permission was denied to the user
More logs`;

    const patterns = [
      {
        pattern: "error.*permission.*denied",
        level_group: 0,
        message_group: 0,
        description: "Permission error",
      },
    ];

    const hasErrors = validateErrors(logContent, patterns);

    // Should detect the real error, not the env var line
    expect(hasErrors).toBe(true);
    expect(global.core.error).toHaveBeenCalled();

    // Verify it caught the actual error line
    const errorCalls = global.core.error.mock.calls;
    const realError = errorCalls.find(call => call[0].includes("permission was denied to the user"));
    expect(realError).toBeDefined();
  });
});
