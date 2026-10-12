import { describe, expect, it } from "vitest";
import { createRequire } from "module";

const require = createRequire(import.meta.url);
const { redactDiagnosticText, redactAndBoundDiagnostics, redactMCPScriptLogMessage } = require("./diagnostic_sanitization.cjs");
const { sanitizeContent } = require("./sanitize_content.cjs");

describe("diagnostic sanitization", () => {
  it.each([
    "[go] Invoking handler with args:",
    "[go] JavaScript handler args:",
    "[go] Python handler args:",
    "[go] Go handler args:",
    "[go] Shell handler args:",
    "[go] Input JSON (300 bytes):",
    "[go] Set env: INPUT_ARGS=",
    "[go] stdout:",
    "[go] stderr:",
    "[go] Serialized result:",
    "[go] Output file content:",
    "[go] Parsed output:",
    "[go] Shell handler completed, outputs:",
    "Error for notification:",
    "[go] JavaScript script error:",
    "[go] Handler threw error:",
    "[go] Stack trace:",
    "recv:",
    "send:",
    "Parse error:",
    "callBackendTool: serverID=mcpscripts, toolName=go, args=",
  ])("withholds the whole payload after %s, even with malformed JSON or line breaks", prefix => {
    const result = redactMCPScriptLogMessage(`${prefix} {"args":["private-input"]}\nprivate-second-line`);
    expect(result).not.toContain("private");
    expect(result).toContain("[REDACTED]");
  });

  it("retains MCP script execution metadata and uses the existing credential redactor", () => {
    expect(redactMCPScriptLogMessage("[go] Executing JavaScript script...")).toBe("[go] Executing JavaScript script...");
    const token = `ghp_${"a".repeat(36)}`;
    expect(redactMCPScriptLogMessage(`Metadata: ${token}`)).not.toContain(token);
  });

  it.each(["maskedValues", "secrets"])("redacts the whole %s value before a built-in pattern can consume its prefix", key => {
    const secret = `sk-proj-${"a".repeat(64)}${"b".repeat(16)}`;
    const options = { secrets: [], [key]: [secret] };
    expect(redactDiagnosticText(`Error: ${secret}`, options)).toBe("Error: ***");
    expect(redactAndBoundDiagnostics(`Error: ${secret}`, options)).not.toContain("b".repeat(16));
  });

  it("prefers longer registered secrets over overlapping prefixes", () => {
    expect(redactDiagnosticText("Error: prefix-suffix", { secrets: ["prefix"], maskedValues: ["prefix-suffix"] })).toBe("Error: ***");
  });

  it.each(["&amp;#38;#96;", "&\x00#96;", "&<!-- hidden -->#96;", "&\x1b[0m#96;"])("normalizes encoded delimiters before a later sanitization pass: %s", entity => {
    const input = entity.repeat(3);
    const result = redactAndBoundDiagnostics(input);
    expect(result).toContain("```");
    expect(sanitizeContent(result)).toBe(result);
  });

  it("redacts secrets exposed by entity normalization", () => {
    const result = redactAndBoundDiagnostics("Error: private&#45;value", { secrets: ["private-value"] });
    expect(result).not.toContain("private");
    expect(result).toContain("***");
  });

  it("bounds oversized diagnostics and preserves truncation notice", () => {
    const result = redactAndBoundDiagnostics("x".repeat(9000));
    expect(result).toContain("[Content truncated due to length]");
    expect(result.length).toBeLessThan(8100);
  });
});
