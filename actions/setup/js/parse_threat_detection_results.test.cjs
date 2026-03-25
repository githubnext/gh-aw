// @ts-check

import { describe, it, expect } from "vitest";

// Set up a minimal global core so the CJS module loads without errors
global.core = {
  info: () => {},
  warning: () => {},
  error: () => {},
  setFailed: () => {},
  setOutput: () => {},
};

const { parseThreatDetectionResult } = require("./parse_threat_detection_results.cjs");

describe("parse_threat_detection_results.cjs", () => {
  describe("default verdict (builtin values)", () => {
    it("returns all false fields when content has no THREAT_DETECTION_RESULT line", () => {
      const result = parseThreatDetectionResult("agent ran successfully\nno threats here");

      expect(result.prompt_injection).toBe(false);
      expect(result.secret_leak).toBe(false);
      expect(result.malicious_patch).toBe(false);
      expect(result.reasons).toEqual([]);
    });

    it("returns all false fields for empty content", () => {
      const result = parseThreatDetectionResult("");

      expect(result.prompt_injection).toBe(false);
      expect(result.secret_leak).toBe(false);
      expect(result.malicious_patch).toBe(false);
      expect(result.reasons).toEqual([]);
    });
  });

  describe("parsing the expected THREAT_DETECTION_RESULT output", () => {
    it("parses the canonical all-false result line correctly", () => {
      const content = 'THREAT_DETECTION_RESULT:{"prompt_injection":false,"secret_leak":false,"malicious_patch":false,"reasons":[]}';

      const result = parseThreatDetectionResult(content);

      expect(result.prompt_injection).toBe(false);
      expect(result.secret_leak).toBe(false);
      expect(result.malicious_patch).toBe(false);
      expect(result.reasons).toEqual([]);
    });

    it("detects prompt injection when set to true", () => {
      const content = 'THREAT_DETECTION_RESULT:{"prompt_injection":true,"secret_leak":false,"malicious_patch":false,"reasons":["Prompt injection detected"]}';

      const result = parseThreatDetectionResult(content);

      expect(result.prompt_injection).toBe(true);
      expect(result.secret_leak).toBe(false);
      expect(result.malicious_patch).toBe(false);
      expect(result.reasons).toEqual(["Prompt injection detected"]);
    });

    it("detects secret leak when set to true", () => {
      const content = 'THREAT_DETECTION_RESULT:{"prompt_injection":false,"secret_leak":true,"malicious_patch":false,"reasons":["Secret found in output"]}';

      const result = parseThreatDetectionResult(content);

      expect(result.prompt_injection).toBe(false);
      expect(result.secret_leak).toBe(true);
      expect(result.malicious_patch).toBe(false);
      expect(result.reasons).toEqual(["Secret found in output"]);
    });

    it("detects malicious patch when set to true", () => {
      const content = 'THREAT_DETECTION_RESULT:{"prompt_injection":false,"secret_leak":false,"malicious_patch":true,"reasons":["Patch modifies security controls"]}';

      const result = parseThreatDetectionResult(content);

      expect(result.prompt_injection).toBe(false);
      expect(result.secret_leak).toBe(false);
      expect(result.malicious_patch).toBe(true);
      expect(result.reasons).toEqual(["Patch modifies security controls"]);
    });

    it("parses multiple reasons correctly", () => {
      const content = 'THREAT_DETECTION_RESULT:{"prompt_injection":true,"secret_leak":true,"malicious_patch":false,"reasons":["Reason one","Reason two"]}';

      const result = parseThreatDetectionResult(content);

      expect(result.prompt_injection).toBe(true);
      expect(result.secret_leak).toBe(true);
      expect(result.reasons).toEqual(["Reason one", "Reason two"]);
    });

    it("finds the result line embedded among other lines", () => {
      const content = ["Starting threat detection analysis...", "Checking for prompt injection...", 'THREAT_DETECTION_RESULT:{"prompt_injection":false,"secret_leak":false,"malicious_patch":false,"reasons":[]}', "Analysis complete."].join(
        "\n"
      );

      const result = parseThreatDetectionResult(content);

      expect(result.prompt_injection).toBe(false);
      expect(result.secret_leak).toBe(false);
      expect(result.malicious_patch).toBe(false);
    });

    it("trims leading and trailing whitespace from the result line", () => {
      const content = '  THREAT_DETECTION_RESULT:{"prompt_injection":false,"secret_leak":false,"malicious_patch":false,"reasons":[]}  ';

      const result = parseThreatDetectionResult(content);

      expect(result.prompt_injection).toBe(false);
      expect(result.secret_leak).toBe(false);
      expect(result.malicious_patch).toBe(false);
    });

    it("uses the first THREAT_DETECTION_RESULT line and ignores subsequent ones", () => {
      const content = [
        'THREAT_DETECTION_RESULT:{"prompt_injection":false,"secret_leak":false,"malicious_patch":false,"reasons":[]}',
        'THREAT_DETECTION_RESULT:{"prompt_injection":true,"secret_leak":true,"malicious_patch":true,"reasons":["should be ignored"]}',
      ].join("\n");

      const result = parseThreatDetectionResult(content);

      expect(result.prompt_injection).toBe(false);
      expect(result.secret_leak).toBe(false);
      expect(result.malicious_patch).toBe(false);
    });

    it("preserves builtin false values when result JSON explicitly sets them to false", () => {
      const content = 'THREAT_DETECTION_RESULT:{"prompt_injection":false,"secret_leak":false,"malicious_patch":false,"reasons":[]}';

      const result = parseThreatDetectionResult(content);

      // Explicitly verify each builtin field is false (not truthy, not null, not undefined)
      expect(result.prompt_injection).toStrictEqual(false);
      expect(result.secret_leak).toStrictEqual(false);
      expect(result.malicious_patch).toStrictEqual(false);
    });
  });
});
