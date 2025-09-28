import { describe, it, expect } from "vitest";

const { 
  convertGoPatternToJS, 
  createJSRegexFromGoPattern, 
  testGoPatternCompatibility 
} = await import("./regex_compatibility.cjs");

describe("convertGoPatternToJS", () => {
  it("should convert case-insensitive Go patterns", () => {
    const result = convertGoPatternToJS("(?i)test pattern");
    expect(result.pattern).toBe("test pattern");
    expect(result.flags).toBe("gi");
  });

  it("should leave regular patterns unchanged", () => {
    const result = convertGoPatternToJS("regular pattern");
    expect(result.pattern).toBe("regular pattern");
    expect(result.flags).toBe("g");
  });

  it("should handle complex patterns with (?i)", () => {
    const result = convertGoPatternToJS("(?i)access denied.*user.*not authorized");
    expect(result.pattern).toBe("access denied.*user.*not authorized");
    expect(result.flags).toBe("gi");
  });

  it("should handle patterns without (?i) prefix", () => {
    const result = convertGoPatternToJS("\\[(\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2})\\]\\s+(ERROR):\\s+(.+)");
    expect(result.pattern).toBe("\\[(\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2})\\]\\s+(ERROR):\\s+(.+)");
    expect(result.flags).toBe("g");
  });
});

describe("createJSRegexFromGoPattern", () => {
  it("should create valid RegExp from Go pattern", () => {
    const regex = createJSRegexFromGoPattern("(?i)test");
    expect(regex).toBeInstanceOf(RegExp);
    expect(regex.flags).toContain("i");
    expect(regex.flags).toContain("g");
  });

  it("should create working case-insensitive regex", () => {
    const regex = createJSRegexFromGoPattern("(?i)ERROR");
    expect(regex.test("error")).toBe(true);
    // Reset regex position due to global flag
    regex.lastIndex = 0;
    expect(regex.test("Error")).toBe(true);
    regex.lastIndex = 0;
    expect(regex.test("ERROR")).toBe(true);
  });

  it("should create working case-sensitive regex", () => {
    const regex = createJSRegexFromGoPattern("ERROR");
    expect(regex.test("ERROR")).toBe(true);
    regex.lastIndex = 0;
    expect(regex.test("error")).toBe(false);
    regex.lastIndex = 0;
    expect(regex.test("Error")).toBe(false);
  });
});

describe("testGoPatternCompatibility", () => {
  it("should report Go (?i) patterns as compatible after conversion", () => {
    const result = testGoPatternCompatibility("(?i)unauthorized");
    expect(result.compatible).toBe(true);
    expect(result.convertedPattern).toBe("unauthorized");
    expect(result.flags).toBe("gi");
  });

  it("should report regular patterns as compatible", () => {
    const result = testGoPatternCompatibility("\\[(\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2})\\]\\s+(ERROR):\\s+(.+)");
    expect(result.compatible).toBe(true);
    expect(result.convertedPattern).toBe("\\[(\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2})\\]\\s+(ERROR):\\s+(.+)");
    expect(result.flags).toBe("g");
  });

  it("should report truly invalid patterns as incompatible", () => {
    const result = testGoPatternCompatibility("[invalid regex");
    expect(result.compatible).toBe(false);
    expect(result.error).toBeTruthy();
  });
});

describe("Engine Pattern Compatibility Tests", () => {
  // Test actual patterns from the engines - these should match the Go implementations exactly
  const codexPatterns = [
    "\\[(\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2})\\]\\s+stream\\s+(error):\\s+(.+)",
    "\\[(\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2})\\]\\s+(ERROR):\\s+(.+)",
    "\\[(\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2})\\]\\s+(WARN|WARNING):\\s+(.+)",
    "(?i)access denied.*only authorized.*can trigger.*workflow",
    "(?i)access denied.*user.*not authorized",
    "(?i)repository permission check failed",
    "(?i)configuration error.*required permissions not specified",
    "(?i)permission.*denied",
    "(?i)unauthorized",
    "(?i)forbidden",
    "(?i)access.*restricted",
    "(?i)insufficient.*permission",
    "(?i)failed in.*permission",
    "(?i)error in.*permission"
  ];

  const copilotPatterns = [
    "(\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}\\.\\d{3}Z)\\s+\\[(ERROR)\\]\\s+(.+)",
    "(\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}\\.\\d{3}Z)\\s+\\[(WARN|WARNING)\\]\\s+(.+)",
    "\\[(\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}\\.\\d{3}Z)\\]\\s+(CRITICAL|ERROR):\\s+(.+)",
    "(Error):\\s+(.+)",
    "npm ERR!\\s+(.+)",
    "(Warning):\\s+(.+)",
    "(Fatal error):\\s+(.+)",
    "copilot:\\s+(error):\\s+(.+)",
    "(?i)access denied.*only authorized.*can trigger.*workflow",
    "(?i)access denied.*user.*not authorized",
    "(?i)repository permission check failed",
    "(?i)configuration error.*required permissions not specified",
    "(?i)permission.*denied",
    "(?i)unauthorized",
    "(?i)forbidden",
    "(?i)access.*restricted",
    "(?i)insufficient.*permission"
  ];

  const claudePatterns = [
    "(?i)access denied.*only authorized.*can trigger.*workflow",
    "(?i)access denied.*user.*not authorized",
    "(?i)repository permission check failed",
    "(?i)configuration error.*required permissions not specified",
    "(?i)permission.*denied",
    "(?i)unauthorized",
    "(?i)forbidden",
    "(?i)access.*restricted",
    "(?i)insufficient.*permission"
  ];

  it("should make all CodexEngine patterns JavaScript compatible", () => {
    codexPatterns.forEach((pattern, index) => {
      const result = testGoPatternCompatibility(pattern);
      expect(result.compatible).toBe(true, `CodexEngine pattern ${index + 1} should be compatible: ${pattern}`);
    });
  });

  it("should make all CopilotEngine patterns JavaScript compatible", () => {
    copilotPatterns.forEach((pattern, index) => {
      const result = testGoPatternCompatibility(pattern);
      expect(result.compatible).toBe(true, `CopilotEngine pattern ${index + 1} should be compatible: ${pattern}`);
    });
  });

  it("should make all ClaudeEngine patterns JavaScript compatible", () => {
    claudePatterns.forEach((pattern, index) => {
      const result = testGoPatternCompatibility(pattern);
      expect(result.compatible).toBe(true, `ClaudeEngine pattern ${index + 1} should be compatible: ${pattern}`);
    });
  });
});

describe("Pattern Functionality Tests", () => {
  it("should correctly match case-insensitive permission patterns", () => {
    const testCases = [
      { pattern: "(?i)unauthorized", text: "UNAUTHORIZED", shouldMatch: true },
      { pattern: "(?i)unauthorized", text: "unauthorized", shouldMatch: true },
      { pattern: "(?i)unauthorized", text: "Unauthorized", shouldMatch: true },
      { pattern: "(?i)forbidden", text: "FORBIDDEN", shouldMatch: true },
      { pattern: "(?i)permission.*denied", text: "Permission denied", shouldMatch: true },
      { pattern: "(?i)permission.*denied", text: "PERMISSION IS DENIED", shouldMatch: true }
    ];

    testCases.forEach(({ pattern, text, shouldMatch }) => {
      const regex = createJSRegexFromGoPattern(pattern);
      regex.lastIndex = 0; // Reset global regex position
      expect(regex.test(text)).toBe(shouldMatch, `Pattern "${pattern}" should ${shouldMatch ? 'match' : 'not match'} "${text}"`);
    });
  });

  it("should correctly match timestamped error patterns", () => {
    const testCases = [
      {
        pattern: "\\[(\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2})\\]\\s+(ERROR):\\s+(.+)",
        text: "[2025-01-10T12:34:56] ERROR: Something went wrong",
        shouldMatch: true,
        expectedGroups: ["2025-01-10T12:34:56", "ERROR", "Something went wrong"]
      },
      {
        pattern: "\\[(\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2})\\]\\s+stream\\s+(error):\\s+(.+)",
        text: "[2025-01-10T12:34:56] stream error: exceeded retry limit",
        shouldMatch: true,
        expectedGroups: ["2025-01-10T12:34:56", "error", "exceeded retry limit"]
      }
    ];

    testCases.forEach(({ pattern, text, shouldMatch, expectedGroups }) => {
      const regex = createJSRegexFromGoPattern(pattern);
      regex.lastIndex = 0; // Reset global regex position
      const match = regex.exec(text);
      
      if (shouldMatch) {
        expect(match).toBeTruthy(`Pattern "${pattern}" should match "${text}"`);
        if (expectedGroups && match) {
          expectedGroups.forEach((expectedGroup, index) => {
            expect(match[index + 1]).toBe(expectedGroup, `Group ${index + 1} should be "${expectedGroup}"`);
          });
        }
      } else {
        expect(match).toBeFalsy(`Pattern "${pattern}" should not match "${text}"`);
      }
    });
  });
});