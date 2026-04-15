// @ts-check
import { describe, it, expect } from "vitest";

const { renderPayloadBlock, parsePayloadFromBody, PAYLOAD_FENCE_INFO } = await import("./payload_helpers.cjs");

describe("payload_helpers", () => {
  describe("renderPayloadBlock", () => {
    it("should render a simple payload object as a details section with pretty-printed JSON", () => {
      const result = renderPayloadBlock({ verdict: "APPROVE", count: 5 });
      expect(result).toBe('<details>\n<summary>payload</summary>\n\n```json gh-aw-payload\n{\n  "verdict": "APPROVE",\n  "count": 5\n}\n```\n\n</details>');
    });

    it("should return empty string for null", () => {
      expect(renderPayloadBlock(null)).toBe("");
    });

    it("should return empty string for undefined", () => {
      expect(renderPayloadBlock(undefined)).toBe("");
    });

    it("should return empty string for an empty object", () => {
      expect(renderPayloadBlock({})).toBe("");
    });

    it("should return empty string for an array", () => {
      // @ts-expect-error testing invalid input
      expect(renderPayloadBlock(["a", "b"])).toBe("");
    });

    it("should handle boolean, number and null values", () => {
      const result = renderPayloadBlock({ passed: true, score: 42, note: null });
      // Extract the JSON content between the fences
      const fenceStart = result.indexOf("```json gh-aw-payload\n") + "```json gh-aw-payload\n".length;
      const fenceEnd = result.lastIndexOf("\n```");
      const parsed = JSON.parse(result.slice(fenceStart, fenceEnd));
      expect(parsed.passed).toBe(true);
      expect(parsed.score).toBe(42);
      expect(parsed.note).toBeNull();
    });

    it("should produce a block with the correct fence info string", () => {
      const result = renderPayloadBlock({ key: "value" });
      expect(result).toContain("```" + PAYLOAD_FENCE_INFO);
      expect(result).toContain("```\n\n</details>");
    });

    it("should wrap payload in a details/summary section", () => {
      const result = renderPayloadBlock({ ok: true });
      expect(result).toContain("<details>");
      expect(result).toContain("<summary>payload</summary>");
      expect(result).toContain("</details>");
    });

    it("should always roundtrip JSON to normalize: extracted JSON is valid and parseable", () => {
      const result = renderPayloadBlock({ ok: true });
      const fenceStart = result.indexOf("```json gh-aw-payload\n") + "```json gh-aw-payload\n".length;
      const fenceEnd = result.lastIndexOf("\n```");
      expect(() => JSON.parse(result.slice(fenceStart, fenceEnd))).not.toThrow();
    });

    it("should pretty-print JSON with indentation", () => {
      const result = renderPayloadBlock({ a: 1, b: "two" });
      // Pretty-printed JSON contains newlines and spaces inside the fence
      expect(result).toContain('  "a": 1');
      expect(result).toContain('  "b": "two"');
    });
  });

  describe("parsePayloadFromBody", () => {
    it("should parse payload back from a compact single-line fenced block (backward compat)", () => {
      const body = '```json gh-aw-payload\n{"verdict":"APPROVE","count":5}\n```';
      const result = parsePayloadFromBody(body);
      expect(result).toEqual({ verdict: "APPROVE", count: 5 });
    });

    it("should parse payload from a pretty-printed fenced block", () => {
      const body = '```json gh-aw-payload\n{\n  "verdict": "APPROVE",\n  "count": 5\n}\n```';
      const result = parsePayloadFromBody(body);
      expect(result).toEqual({ verdict: "APPROVE", count: 5 });
    });

    it("should parse payload from a details-wrapped block (new render format)", () => {
      const rendered = renderPayloadBlock({ verdict: "APPROVE", count: 5 });
      const result = parsePayloadFromBody(rendered);
      expect(result).toEqual({ verdict: "APPROVE", count: 5 });
    });

    it("should return null when no payload block is present", () => {
      expect(parsePayloadFromBody("Just a regular comment body")).toBeNull();
    });

    it("should return null for null input", () => {
      // @ts-expect-error testing invalid input
      expect(parsePayloadFromBody(null)).toBeNull();
    });

    it("should return null for empty string", () => {
      expect(parsePayloadFromBody("")).toBeNull();
    });

    it("should return null when embedded JSON is malformed", () => {
      const body = "```json gh-aw-payload\n{invalid json}\n```";
      expect(parsePayloadFromBody(body)).toBeNull();
    });

    it("should round-trip through render and parse", () => {
      const original = { verdict: "APPROVE", criteria_passed: 5, approved: true, note: null };
      const rendered = renderPayloadBlock(original);
      const parsed = parsePayloadFromBody("Review done\n\n" + rendered + "\n\nFooter");
      expect(parsed).toEqual(original);
    });

    it("should return null when embedded JSON is an array, not object", () => {
      const body = "```json gh-aw-payload\n[1,2,3]\n```";
      expect(parsePayloadFromBody(body)).toBeNull();
    });

    it("should parse payload with CRLF line endings", () => {
      const body = 'Some content\r\n\r\n```json gh-aw-payload\r\n{"verdict":"APPROVE"}\r\n```\r\n\r\nFooter';
      const result = parsePayloadFromBody(body);
      expect(result).toEqual({ verdict: "APPROVE" });
    });
  });
});
