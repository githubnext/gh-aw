import { describe, it, expect } from "vitest";
import { parseJsonlContent } from "./jsonl_helpers.cjs";

describe("jsonl_helpers", () => {
  describe("parseJsonlContent", () => {
    it("returns parsed JSON entries and skips malformed lines", () => {
      const parsed = parseJsonlContent(['{"event":"token_steering"}', "not-json", "", "   ", '{"event":"request"}'].join("\n"));

      expect(parsed).toEqual([{ event: "token_steering" }, { event: "request" }]);
    });

    it("returns empty array for non-string or empty content", () => {
      expect(parseJsonlContent("")).toEqual([]);
      expect(parseJsonlContent(/** @type {any} */ null)).toEqual([]);
      expect(parseJsonlContent(/** @type {any} */ undefined)).toEqual([]);
    });

    it("supports optional line pre-filtering before JSON parsing", () => {
      const parsed = parseJsonlContent(['{"event":"token_steering"}', '{"event":"request"}', '{"event":"model_steering"}'].join("\n"), line => line.includes("steering"));

      expect(parsed).toEqual([{ event: "token_steering" }, { event: "model_steering" }]);
    });
  });
});
