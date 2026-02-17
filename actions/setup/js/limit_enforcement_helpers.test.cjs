// @ts-check
import { describe, it, expect } from "vitest";

const { enforceArrayLimit, tryEnforceArrayLimit } = require("./limit_enforcement_helpers.cjs");

describe("limit_enforcement_helpers", () => {
  describe("enforceArrayLimit", () => {
    it("should not throw when array is under limit", () => {
      expect(() => enforceArrayLimit([1, 2, 3], 5, "items")).not.toThrow();
    });

    it("should not throw when array equals limit", () => {
      expect(() => enforceArrayLimit([1, 2, 3, 4, 5], 5, "items")).not.toThrow();
    });

    it("should throw E003 error when array exceeds limit", () => {
      expect(() => enforceArrayLimit([1, 2, 3, 4, 5, 6], 5, "items")).toThrow("E003");
      expect(() => enforceArrayLimit([1, 2, 3, 4, 5, 6], 5, "items")).toThrow("Cannot add more than 5 items");
      expect(() => enforceArrayLimit([1, 2, 3, 4, 5, 6], 5, "items")).toThrow("received 6");
    });

    it("should not throw when array is null or undefined", () => {
      expect(() => enforceArrayLimit(null, 5, "items")).not.toThrow();
      expect(() => enforceArrayLimit(undefined, 5, "items")).not.toThrow();
    });

    it("should not throw when array is not an array", () => {
      // @ts-ignore - testing invalid input
      expect(() => enforceArrayLimit("not an array", 5, "items")).not.toThrow();
      // @ts-ignore - testing invalid input
      expect(() => enforceArrayLimit(123, 5, "items")).not.toThrow();
    });

    it("should use parameter name in error message", () => {
      expect(() => enforceArrayLimit([1, 2, 3], 2, "labels")).toThrow("labels");
      expect(() => enforceArrayLimit([1, 2, 3, 4], 2, "assignees")).toThrow("assignees");
    });
  });

  describe("tryEnforceArrayLimit", () => {
    it("should return success when array is under limit", () => {
      const result = tryEnforceArrayLimit([1, 2, 3], 5, "items");
      expect(result.success).toBe(true);
    });

    it("should return success when array equals limit", () => {
      const result = tryEnforceArrayLimit([1, 2, 3, 4, 5], 5, "items");
      expect(result.success).toBe(true);
    });

    it("should return error result when array exceeds limit", () => {
      const result = tryEnforceArrayLimit([1, 2, 3, 4, 5, 6], 5, "items");
      expect(result.success).toBe(false);
      expect(result.error).toContain("E003");
      expect(result.error).toContain("Cannot add more than 5 items");
      expect(result.error).toContain("received 6");
    });

    it("should return success when array is null or undefined", () => {
      const result1 = tryEnforceArrayLimit(null, 5, "items");
      expect(result1.success).toBe(true);

      const result2 = tryEnforceArrayLimit(undefined, 5, "items");
      expect(result2.success).toBe(true);
    });

    it("should include parameter name in error message", () => {
      const result = tryEnforceArrayLimit([1, 2, 3], 2, "labels");
      expect(result.success).toBe(false);
      expect(result.error).toContain("labels");
    });
  });
});
