import { describe, it, expect } from "vitest";
import { getErrorMessage } from "./error_helpers.cjs";

describe("error_helpers", () => {
  describe("getErrorMessage", () => {
    it("should extract message from Error instance", () => {
      const error = new Error("Test error message");
      expect(getErrorMessage(error)).toBe("Test error message");
    });

    it("should extract message from object with message property", () => {
      const error = { message: "Custom error message" };
      expect(getErrorMessage(error)).toBe("Custom error message");
    });

    it("should handle objects with non-string message property", () => {
      const error = { message: 123 };
      expect(getErrorMessage(error)).toBe("[object Object]");
    });

    it("should convert string to string", () => {
      expect(getErrorMessage("Plain string error")).toBe("Plain string error");
    });

    it("should convert number to string", () => {
      expect(getErrorMessage(42)).toBe("42");
    });

    it("should convert null to string", () => {
      expect(getErrorMessage(null)).toBe("null");
    });

    it("should convert undefined to string", () => {
      expect(getErrorMessage(undefined)).toBe("undefined");
    });

    it("should handle object without message property", () => {
      const error = { code: "ERROR_CODE", status: 500 };
      expect(getErrorMessage(error)).toBe("[object Object]");
    });
  });
});
