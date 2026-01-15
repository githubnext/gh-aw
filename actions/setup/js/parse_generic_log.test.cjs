import { describe, it, expect, beforeEach, vi } from "vitest";

describe("parse_generic_log.cjs", () => {
  let parseGenericLog;

  beforeEach(async () => {
    // Use runtime import to load CommonJS module
    const module = await import("./parse_generic_log.cjs");
    parseGenericLog = module.parseGenericLog;
  });

  it("should wrap simple log content in markdown code block", () => {
    const logContent = "This is a simple log message\nAnother line";
    const result = parseGenericLog(logContent);

    expect(result.markdown).toContain("## Agent Log");
    expect(result.markdown).toContain("```");
    expect(result.markdown).toContain(logContent);
    expect(result.mcpFailures).toEqual([]);
    expect(result.maxTurnsHit).toBe(false);
    expect(result.logEntries).toEqual([]);
  });

  it("should handle empty log content", () => {
    const logContent = "";
    const result = parseGenericLog(logContent);

    expect(result.markdown).toContain("## Agent Log");
    expect(result.markdown).toContain("```");
    expect(result.mcpFailures).toEqual([]);
    expect(result.maxTurnsHit).toBe(false);
    expect(result.logEntries).toEqual([]);
  });

  it("should preserve multiline log content", () => {
    const logContent = `Line 1
Line 2
Line 3`;
    const result = parseGenericLog(logContent);

    expect(result.markdown).toContain("Line 1");
    expect(result.markdown).toContain("Line 2");
    expect(result.markdown).toContain("Line 3");
  });

  it("should handle log content with special characters", () => {
    const logContent = "Error: Invalid $variable in `command`\nStack trace: ...";
    const result = parseGenericLog(logContent);

    expect(result.markdown).toContain("Error: Invalid $variable");
    expect(result.markdown).toContain("Stack trace: ...");
  });
});
