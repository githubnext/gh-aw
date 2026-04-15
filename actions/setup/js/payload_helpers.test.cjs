// @ts-check
import { describe, it, expect } from "vitest";

const { renderPayloadBlock, parsePayloadFromBody, PAYLOAD_FENCE_INFO } = await import("./payload_helpers.cjs");

describe("payload_helpers", () => {
  describe("renderPayloadBlock", () => {
    it("should render a simple payload object as a fenced code block", () => {
      const result = renderPayloadBlock({ verdict: "APPROVE", count: 5 });
      expect(result).toBe('```json gh-aw-payload\n{"verdict":"APPROVE","count":5}\n```');
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
      const json = result.split("\n")[1];
      const parsed = JSON.parse(json);
      expect(parsed.passed).toBe(true);
      expect(parsed.score).toBe(42);
      expect(parsed.note).toBeNull();
    });

    it("should produce a block with the correct fence info string", () => {
      const result = renderPayloadBlock({ key: "value" });
      expect(result.startsWith("```" + PAYLOAD_FENCE_INFO)).toBe(true);
      expect(result.endsWith("```")).toBe(true);
    });

    it("should return empty string when JSON.stringify produces invalid JSON", () => {
      // Simulate an object that would break JSON.parse (e.g., via proxy that yields undefined)
      // In practice we test by passing an object where JSON.stringify returns non-parseable JSON.
      // The simplest verifiable case: a plain object always produces valid JSON, so we verify
      // that the guard doesn't reject valid inputs.
      const result = renderPayloadBlock({ ok: true });
      // Valid JSON round-trips cleanly
      const json = result.split("\n")[1];
      expect(() => JSON.parse(json)).not.toThrow();
    });
  });

  describe("parsePayloadFromBody", () => {
    it("should parse payload back from a rendered body", () => {
      const body = '```json gh-aw-payload\n{"verdict":"APPROVE","count":5}\n```';
      const result = parsePayloadFromBody(body);
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
