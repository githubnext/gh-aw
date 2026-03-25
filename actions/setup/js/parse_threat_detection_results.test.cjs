// @ts-check

import { describe, it, expect, vi, beforeEach } from "vitest";

// Set up a minimal global core so the CJS module loads without errors
global.core = {
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
};

const { parseThreatDetectionResult } = require("./parse_threat_detection_results.cjs");

describe("parse_threat_detection_results.cjs", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("default verdict when no result line is present (builtin values)", () => {
    it("returns found: false when content has no THREAT_DETECTION_RESULT line", () => {
      const { found } = parseThreatDetectionResult("agent ran successfully\nno threats here");

      expect(found).toBe(false);
    });

    it("returns all false verdict fields when no THREAT_DETECTION_RESULT line is present", () => {
      const { verdict } = parseThreatDetectionResult("agent ran successfully\nno threats here");

      expect(verdict.prompt_injection).toBe(false);
      expect(verdict.secret_leak).toBe(false);
      expect(verdict.malicious_patch).toBe(false);
      expect(verdict.reasons).toEqual([]);
    });

    it("returns found: false for empty content", () => {
      const { found } = parseThreatDetectionResult("");

      expect(found).toBe(false);
    });

    it("logs a warning when no result line is found", () => {
      parseThreatDetectionResult("no result here");

      expect(global.core.warning).toHaveBeenCalledWith(expect.stringContaining("No THREAT_DETECTION_RESULT line found"));
    });
  });

  describe("parsing the expected THREAT_DETECTION_RESULT output", () => {
    it("returns found: true for the canonical result line", () => {
      const content = 'THREAT_DETECTION_RESULT:{"prompt_injection":false,"secret_leak":false,"malicious_patch":false,"reasons":[]}';

      const { found } = parseThreatDetectionResult(content);

      expect(found).toBe(true);
    });

    it("parses the canonical all-false result line correctly", () => {
      const content = 'THREAT_DETECTION_RESULT:{"prompt_injection":false,"secret_leak":false,"malicious_patch":false,"reasons":[]}';

      const { verdict } = parseThreatDetectionResult(content);

      expect(verdict.prompt_injection).toBe(false);
      expect(verdict.secret_leak).toBe(false);
      expect(verdict.malicious_patch).toBe(false);
      expect(verdict.reasons).toEqual([]);
    });

    it("logs an info message when the result line is found", () => {
      const content = 'THREAT_DETECTION_RESULT:{"prompt_injection":false,"secret_leak":false,"malicious_patch":false,"reasons":[]}';

      parseThreatDetectionResult(content);

      expect(global.core.info).toHaveBeenCalledWith(expect.stringContaining("Found THREAT_DETECTION_RESULT line"));
    });

    it("detects prompt injection when set to true", () => {
      const content = 'THREAT_DETECTION_RESULT:{"prompt_injection":true,"secret_leak":false,"malicious_patch":false,"reasons":["Prompt injection detected"]}';

      const { verdict } = parseThreatDetectionResult(content);

      expect(verdict.prompt_injection).toBe(true);
      expect(verdict.secret_leak).toBe(false);
      expect(verdict.malicious_patch).toBe(false);
      expect(verdict.reasons).toEqual(["Prompt injection detected"]);
    });

    it("detects secret leak when set to true", () => {
      const content = 'THREAT_DETECTION_RESULT:{"prompt_injection":false,"secret_leak":true,"malicious_patch":false,"reasons":["Secret found in output"]}';

      const { verdict } = parseThreatDetectionResult(content);

      expect(verdict.prompt_injection).toBe(false);
      expect(verdict.secret_leak).toBe(true);
      expect(verdict.malicious_patch).toBe(false);
      expect(verdict.reasons).toEqual(["Secret found in output"]);
    });

    it("detects malicious patch when set to true", () => {
      const content = 'THREAT_DETECTION_RESULT:{"prompt_injection":false,"secret_leak":false,"malicious_patch":true,"reasons":["Patch modifies security controls"]}';

      const { verdict } = parseThreatDetectionResult(content);

      expect(verdict.prompt_injection).toBe(false);
      expect(verdict.secret_leak).toBe(false);
      expect(verdict.malicious_patch).toBe(true);
      expect(verdict.reasons).toEqual(["Patch modifies security controls"]);
    });

    it("parses multiple reasons correctly", () => {
      const content = 'THREAT_DETECTION_RESULT:{"prompt_injection":true,"secret_leak":true,"malicious_patch":false,"reasons":["Reason one","Reason two"]}';

      const { verdict } = parseThreatDetectionResult(content);

      expect(verdict.prompt_injection).toBe(true);
      expect(verdict.secret_leak).toBe(true);
      expect(verdict.reasons).toEqual(["Reason one", "Reason two"]);
    });

    it("finds the result line embedded among other lines", () => {
      const content = ["Starting threat detection analysis...", "Checking for prompt injection...", 'THREAT_DETECTION_RESULT:{"prompt_injection":false,"secret_leak":false,"malicious_patch":false,"reasons":[]}', "Analysis complete."].join(
        "\n"
      );

      const { verdict, found } = parseThreatDetectionResult(content);

      expect(found).toBe(true);
      expect(verdict.prompt_injection).toBe(false);
      expect(verdict.secret_leak).toBe(false);
      expect(verdict.malicious_patch).toBe(false);
    });

    it("trims leading and trailing whitespace from the result line", () => {
      const content = '  THREAT_DETECTION_RESULT:{"prompt_injection":false,"secret_leak":false,"malicious_patch":false,"reasons":[]}  ';

      const { verdict, found } = parseThreatDetectionResult(content);

      expect(found).toBe(true);
      expect(verdict.prompt_injection).toBe(false);
      expect(verdict.secret_leak).toBe(false);
      expect(verdict.malicious_patch).toBe(false);
    });

    it("uses the first THREAT_DETECTION_RESULT line and ignores subsequent ones", () => {
      const content = [
        'THREAT_DETECTION_RESULT:{"prompt_injection":false,"secret_leak":false,"malicious_patch":false,"reasons":[]}',
        'THREAT_DETECTION_RESULT:{"prompt_injection":true,"secret_leak":true,"malicious_patch":true,"reasons":["should be ignored"]}',
      ].join("\n");

      const { verdict } = parseThreatDetectionResult(content);

      expect(verdict.prompt_injection).toBe(false);
      expect(verdict.secret_leak).toBe(false);
      expect(verdict.malicious_patch).toBe(false);
    });

    it("preserves builtin false values when result JSON explicitly sets them to false", () => {
      const content = 'THREAT_DETECTION_RESULT:{"prompt_injection":false,"secret_leak":false,"malicious_patch":false,"reasons":[]}';

      const { verdict } = parseThreatDetectionResult(content);

      // Explicitly verify each builtin field is false (not truthy, not null, not undefined)
      expect(verdict.prompt_injection).toStrictEqual(false);
      expect(verdict.secret_leak).toStrictEqual(false);
      expect(verdict.malicious_patch).toStrictEqual(false);
    });
  });
});
