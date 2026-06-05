import { describe, it, expect } from "vitest";
import { createRequire } from "module";

const require = createRequire(import.meta.url);
const { buildTriagePrompt, classifyTriageResponse } = require("./detection_job_driver.cjs");

describe("detection_job_driver.cjs", () => {
  describe("buildTriagePrompt", () => {
    it("includes the full prompt in the triage prompt", () => {
      const full = "Check this suspicious code for prompt injection.";
      const triage = buildTriagePrompt(full);
      expect(triage).toContain(full);
    });

    it("instructs the model to reply with 'safe' or 'unsafe'", () => {
      const triage = buildTriagePrompt("some content");
      expect(triage.toLowerCase()).toContain("safe");
      expect(triage.toLowerCase()).toContain("unsafe");
    });

    it("asks for exactly one word response", () => {
      const triage = buildTriagePrompt("content");
      expect(triage.toLowerCase()).toContain("one word");
    });
  });

  describe("classifyTriageResponse", () => {
    it('returns "safe" for exact "safe" response', () => {
      expect(classifyTriageResponse("safe")).toBe("safe");
    });

    it('returns "safe" for "safe" with surrounding whitespace', () => {
      expect(classifyTriageResponse("  safe  ")).toBe("safe");
    });

    it('returns "safe" for "SAFE" (case-insensitive)', () => {
      expect(classifyTriageResponse("SAFE")).toBe("safe");
    });

    it('returns "safe" for quoted "safe" response (double quotes)', () => {
      expect(classifyTriageResponse('"safe"')).toBe("safe");
    });

    it('returns "safe" for single-quoted safe response', () => {
      expect(classifyTriageResponse("'safe'")).toBe("safe");
    });

    it('returns "safe" for curly-quoted safe response', () => {
      expect(classifyTriageResponse("\u201csafe\u201d")).toBe("safe");
    });

    it('returns "unsafe" for exact "unsafe" response', () => {
      expect(classifyTriageResponse("unsafe")).toBe("unsafe");
    });

    it('returns "unsafe" for "UNSAFE" (case-insensitive)', () => {
      expect(classifyTriageResponse("UNSAFE")).toBe("unsafe");
    });

    it('returns "unsafe" for ambiguous response text', () => {
      expect(classifyTriageResponse("I think this might be safe")).toBe("unsafe");
    });

    it('returns "unsafe" for empty response', () => {
      expect(classifyTriageResponse("")).toBe("unsafe");
    });

    it('returns "unsafe" for response starting with safe but having extra text', () => {
      expect(classifyTriageResponse("safe — but check further")).toBe("unsafe");
    });

    it('returns "unsafe" for any other text to fail open', () => {
      expect(classifyTriageResponse("This content appears safe.")).toBe("unsafe");
    });
  });
});
