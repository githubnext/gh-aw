// @ts-check
import { describe, it, expect, afterEach, vi } from "vitest";
import { createRequire } from "module";

const req = createRequire(import.meta.url);
const { getActionInput } = req("./action_input_utils.cjs");

describe("getActionInput", () => {
  afterEach(() => {
    vi.unstubAllEnvs();
  });

  describe("underscore form (INPUT_<NAME>)", () => {
    it("returns the underscore form value when set", () => {
      vi.stubEnv("INPUT_JOB_NAME", "agent");
      expect(getActionInput("JOB_NAME")).toBe("agent");
    });

    it("trims leading and trailing whitespace from the underscore form", () => {
      vi.stubEnv("INPUT_JOB_NAME", "  agent  ");
      expect(getActionInput("JOB_NAME")).toBe("agent");
    });

    it("returns empty string when underscore form is whitespace-only", () => {
      vi.stubEnv("INPUT_JOB_NAME", "   ");
      vi.stubEnv("INPUT_JOB-NAME", "");
      expect(getActionInput("JOB_NAME")).toBe("");
    });
  });

  describe("hyphen form (INPUT_<NAM-E>)", () => {
    it("returns the hyphen form value when only the hyphen form is set", () => {
      vi.stubEnv("INPUT_JOB-NAME", "agent");
      expect(getActionInput("JOB_NAME")).toBe("agent");
    });

    it("trims leading and trailing whitespace from the hyphen form", () => {
      vi.stubEnv("INPUT_JOB-NAME", "  trimmed  ");
      expect(getActionInput("JOB_NAME")).toBe("trimmed");
    });
  });

  describe("precedence", () => {
    it("prefers the underscore form over the hyphen form when both are set", () => {
      vi.stubEnv("INPUT_JOB_NAME", "underscore-value");
      vi.stubEnv("INPUT_JOB-NAME", "hyphen-value");
      expect(getActionInput("JOB_NAME")).toBe("underscore-value");
    });

    it("falls back to hyphen form when underscore form is an empty string", () => {
      vi.stubEnv("INPUT_JOB_NAME", "");
      vi.stubEnv("INPUT_JOB-NAME", "hyphen-fallback");
      expect(getActionInput("JOB_NAME")).toBe("hyphen-fallback");
    });
  });

  describe("absent variables", () => {
    it("returns empty string when neither form is set", () => {
      expect(getActionInput("JOB_NAME")).toBe("");
    });

    it("returns empty string for an unrecognised input name", () => {
      expect(getActionInput("NONEXISTENT_INPUT_XYZ")).toBe("");
    });
  });

  describe("input names", () => {
    it("handles input names with multiple underscores", () => {
      vi.stubEnv("INPUT_MY_LONG_INPUT_NAME", "multi-underscore");
      expect(getActionInput("MY_LONG_INPUT_NAME")).toBe("multi-underscore");
    });

    it("converts all underscores in the name to hyphens for the fallback form", () => {
      vi.stubEnv("INPUT_MY-LONG-INPUT-NAME", "hyphen-multi");
      expect(getActionInput("MY_LONG_INPUT_NAME")).toBe("hyphen-multi");
    });

    it("handles single-word input names (no underscores)", () => {
      vi.stubEnv("INPUT_TOKEN", "secret-token");
      expect(getActionInput("TOKEN")).toBe("secret-token");
    });
  });
});
