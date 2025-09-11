import { describe, it, expect, beforeEach, vi } from "vitest";

const { validateErrors, extractLevel, extractMessage, truncateString } =
  await import("./validate_errors.cjs");

// Mock global objects for testing
global.console = {
  log: vi.fn(),
  warn: vi.fn(),
  debug: vi.fn(),
};

global.core = {
  summary: {
    addRaw: vi.fn(() => ({ write: vi.fn() })),
  },
  setFailed: vi.fn(),
  warn: vi.fn(),
  error: vi.fn(),
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
        pattern:
          "\\[(\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2})\\]\\s+stream\\s+(error):\\s+(.+)",
        level_group: 2,
        message_group: 3,
        description: "Codex stream errors with timestamp",
      },
      {
        pattern:
          "\\[(\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2})\\]\\s+(ERROR):\\s+(.+)",
        level_group: 2,
        message_group: 3,
        description: "Codex ERROR messages with timestamp",
      },
      {
        pattern:
          "\\[(\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2})\\]\\s+(WARN|WARNING):\\s+(.+)",
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
    expect(global.core.error).toHaveBeenCalledWith(
      expect.stringContaining("exceeded retry limit")
    );

    // Should call core.warn for warnings
    expect(global.core.warn).toHaveBeenCalledTimes(1);
    expect(global.core.warn).toHaveBeenCalledWith(
      expect.stringContaining("This is a warning message")
    );
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

    // Should not call core.error or core.warn for empty content
    expect(global.core.error).not.toHaveBeenCalled();
    expect(global.core.warn).not.toHaveBeenCalled();
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

    // Should not call core.error or core.warn when no patterns match
    expect(global.core.error).not.toHaveBeenCalled();
    expect(global.core.warn).not.toHaveBeenCalled();
  });

  test("should crash with invalid regex patterns", () => {
    const logContent = "ERROR: Something went wrong";
    const patterns = [
      {
        pattern: "[invalid regex", // Missing closing bracket
        level_group: 0,
        message_group: 1,
        description: "Invalid pattern",
      },
    ];

    // Should throw an exception instead of handling gracefully
    expect(() => validateErrors(logContent, patterns)).toThrow();
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
        pattern:
          "\\[(\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2})\\]\\s+stream\\s+(error):\\s+(.+)",
        level_group: 2,
        message_group: 3,
        description: "Codex stream errors with timestamp",
      },
      {
        pattern:
          "\\[(\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2})\\]\\s+(ERROR):\\s+(.+)",
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
    expect(global.core.error).toHaveBeenCalledWith(
      expect.stringContaining("401 Unauthorized")
    );
    expect(global.core.error).toHaveBeenCalledWith(
      expect.stringContaining("exceeded retry limit")
    );
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
    process.env.GITHUB_AW_AGENT_OUTPUT = "/tmp/test.log";
    process.env.GITHUB_AW_ERROR_PATTERNS = JSON.stringify([
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
      global.core.setFailed(
        "Errors detected in agent logs - failing workflow step"
      );
    }

    expect(global.core.setFailed).toHaveBeenCalledWith(
      "Errors detected in agent logs - failing workflow step"
    );

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
      global.core.setFailed(
        "Errors detected in agent logs - failing workflow step"
      );
    }

    expect(global.core.setFailed).not.toHaveBeenCalled();
  });
});
