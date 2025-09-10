import { describe, it, expect, beforeEach, vi } from "vitest";

const {
  validateErrors,
  extractLevel,
  extractMessage,
  generateValidationSummary,
  truncateString,
} = await import("./validate_errors.cjs");

// Mock global objects for testing
global.console = {
  log: vi.fn(),
  warn: vi.fn(),
};

global.core = {
  summary: {
    addRaw: vi.fn(() => ({ write: vi.fn() })),
  },
  setFailed: vi.fn(),
  warn: vi.fn(),
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

    const result = validateErrors(logContent, patterns);

    expect(result).toBeDefined();
    expect(result).toContain("Log Validation Results");
    expect(result).toContain("🚨 **2** error(s)");
    expect(result).toContain("⚠️ **1** warning(s)");
    expect(result).toContain("exceeded retry limit");
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

    const result = validateErrors("", patterns);
    expect(result).toBeNull();
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

    const result = validateErrors(logContent, patterns);
    expect(result).toBeNull();
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

    // Should not throw an exception
    const result = validateErrors(logContent, patterns);
    expect(global.core.warn).toHaveBeenCalled();
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

describe("generateValidationSummary", () => {
  test("should return null when no issues found", () => {
    const result = generateValidationSummary([], []);
    expect(result).toBeNull();
  });

  test("should generate proper markdown for errors and warnings", () => {
    const errors = [
      {
        line: 1,
        level: "error",
        message: "Critical failure",
        pattern: "Error pattern",
        rawLine: "[2025-09-10T17:54:49] ERROR: Critical failure",
      },
    ];

    const warnings = [
      {
        line: 2,
        level: "warning",
        message: "Minor issue",
        pattern: "Warning pattern",
        rawLine: "[2025-09-10T17:54:50] WARNING: Minor issue",
      },
    ];

    const result = generateValidationSummary(errors, warnings);

    expect(result).toContain("## 🔍 Log Validation Results");
    expect(result).toContain("🚨 **1** error(s)");
    expect(result).toContain("⚠️ **1** warning(s)");
    expect(result).toContain("### 🚨 Errors");
    expect(result).toContain("### ⚠️ Warnings");
    expect(result).toContain("Critical failure");
    expect(result).toContain("Minor issue");
    expect(result).toContain("### 💡 Recommendations");
  });

  test("should handle warnings only", () => {
    const warnings = [
      {
        line: 1,
        level: "warning",
        message: "Just a warning",
        pattern: "Warning pattern",
        rawLine: "WARNING: Just a warning",
      },
    ];

    const result = generateValidationSummary([], warnings);

    expect(result).toContain("⚠️ **1** warning(s)");
    expect(result).not.toContain("🚨");
    expect(result).not.toContain("### 💡 Recommendations");
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
