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
    });
  });
});
