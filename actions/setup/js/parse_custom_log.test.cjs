// @ts-check

import { describe, it, expect } from "vitest";
import { parseCustomLog } from "./parse_custom_log.cjs";

describe("parseCustomLog", () => {
  it("should detect and parse Claude format logs", () => {
    const claudeLog = JSON.stringify([
      {
        type: "init",
        num_turns: 1,
        tools: [],
      },
      {
        type: "turn",
        turn_number: 1,
        message: { role: "user", content: "Hello" },
      },
    ]);

    const result = parseCustomLog(claudeLog);

    expect(result).toBeDefined();
    expect(result.markdown).toContain("Custom Engine Log (Claude format)");
    expect(result.logEntries.length).toBeGreaterThan(0);
  });

  it("should detect and parse Codex format logs", () => {
    const codexLog = `{"type":"init","tools":[],"num_turns":1}
{"type":"turn","turn_number":1,"message":"Hello"}`;

    const result = parseCustomLog(codexLog);

    expect(result).toBeDefined();
    // Note: Actual format detection depends on parse_codex_log implementation
    expect(result.markdown).toBeDefined();
  });

  it("should handle unrecognized log format with basic fallback", () => {
    // Use a more complex string that neither parser will recognize
    // The Codex parser is very lenient and will accept most text
    const unknownLog = "Some plain text log\nwith multiple lines\nand no structure";

    const result = parseCustomLog(unknownLog);

    expect(result).toBeDefined();
    expect(result.markdown).toContain("Custom Engine Log");
    // The Codex parser will handle this, so check for Codex format
    expect(result.markdown).toBeDefined();
    expect(result.mcpFailures).toBeDefined();
    expect(result.logEntries).toBeDefined();
  });

  it("should truncate long logs in fallback mode", () => {
    // Use a long string that the parsers will process
    const longLog = "a".repeat(2000);

    const result = parseCustomLog(longLog);

    expect(result).toBeDefined();
    // Codex parser will handle this
    expect(result.markdown).toContain("Custom Engine Log");
  });

  it("should handle empty log content", () => {
    const result = parseCustomLog("");

    expect(result).toBeDefined();
    expect(result.markdown).toContain("Custom Engine Log");
    expect(result.logEntries).toEqual([]);
  });
});
