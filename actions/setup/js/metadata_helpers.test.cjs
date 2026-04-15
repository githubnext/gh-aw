// @ts-check
import { describe, it, expect } from "vitest";

const { renderMetadataBlock, parseMetadataFromBody, METADATA_FENCE_LANG } = await import("./metadata_helpers.cjs");

describe("metadata_helpers", () => {
  describe("renderMetadataBlock", () => {
    it("should render a simple metadata object as a fenced code block", () => {
      const result = renderMetadataBlock({ verdict: "APPROVE", count: 5 });
      expect(result).toBe('```aw-metadata\n{"verdict":"APPROVE","count":5}\n```');
    });

    it("should return empty string for null", () => {
      expect(renderMetadataBlock(null)).toBe("");
    });

    it("should return empty string for undefined", () => {
      expect(renderMetadataBlock(undefined)).toBe("");
    });

    it("should return empty string for an empty object", () => {
      expect(renderMetadataBlock({})).toBe("");
    });

    it("should return empty string for an array", () => {
      // @ts-expect-error testing invalid input
      expect(renderMetadataBlock(["a", "b"])).toBe("");
    });

    it("should handle boolean, number and null values", () => {
      const result = renderMetadataBlock({ passed: true, score: 42, note: null });
      const parsed = JSON.parse(result.split("\n")[1]);
      expect(parsed.passed).toBe(true);
      expect(parsed.score).toBe(42);
      expect(parsed.note).toBeNull();
    });

    it("should produce a block with the correct fence language tag", () => {
      const result = renderMetadataBlock({ key: "value" });
      expect(result.startsWith("```" + METADATA_FENCE_LANG)).toBe(true);
      expect(result.endsWith("```")).toBe(true);
    });
  });

  describe("parseMetadataFromBody", () => {
    it("should parse metadata back from a rendered body", () => {
      const body = 'Some content\n\n```aw-metadata\n{"verdict":"APPROVE","count":5}\n```\n\nFooter';
      const result = parseMetadataFromBody(body);
      expect(result).toEqual({ verdict: "APPROVE", count: 5 });
    });

    it("should return null when no metadata block is present", () => {
      expect(parseMetadataFromBody("Just a regular comment body")).toBeNull();
    });

    it("should return null for null input", () => {
      // @ts-expect-error testing invalid input
      expect(parseMetadataFromBody(null)).toBeNull();
    });

    it("should return null for empty string", () => {
      expect(parseMetadataFromBody("")).toBeNull();
    });

    it("should return null when embedded JSON is malformed", () => {
      const body = "```aw-metadata\n{invalid json}\n```";
      expect(parseMetadataFromBody(body)).toBeNull();
    });

    it("should round-trip through render and parse", () => {
      const original = { verdict: "APPROVE", criteria_passed: 5, approved: true, note: null };
      const rendered = renderMetadataBlock(original);
      const parsed = parseMetadataFromBody("Review done\n\n" + rendered + "\n\nFooter");
      expect(parsed).toEqual(original);
    });

    it("should return null when embedded JSON is an array, not object", () => {
      const body = "```aw-metadata\n[1,2,3]\n```";
      expect(parseMetadataFromBody(body)).toBeNull();
    });
  });
});
