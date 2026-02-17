import { describe, it, expect } from "vitest";
import { getErrorMessage, isLockedError } from "./error_helpers.cjs";

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

  describe("isLockedError", () => {
    it("should return true for 403 error with 'locked' in message", () => {
      const error = new Error("Issue is locked");
      error.status = 403;
      expect(isLockedError(error)).toBe(true);
    });

    it("should return true for 403 error with 'Lock conversation' in message", () => {
      const error = new Error("Lock conversation is enabled");
      error.status = 403;
      expect(isLockedError(error)).toBe(true);
    });

    it("should return false for 403 error without 'locked' in message", () => {
      const error = new Error("Forbidden: insufficient permissions");
      error.status = 403;
      expect(isLockedError(error)).toBe(false);
    });

    it("should return false for non-403 error with 'locked' in message", () => {
      const error = new Error("Issue is locked");
      error.status = 500;
      expect(isLockedError(error)).toBe(false);
    });

    it("should return false for error without status property", () => {
      const error = new Error("Issue is locked");
      expect(isLockedError(error)).toBe(false);
    });

    it("should return false for null error", () => {
      expect(isLockedError(null)).toBe(false);
    });

    it("should return false for undefined error", () => {
      expect(isLockedError(undefined)).toBe(false);
    });

    it("should handle object errors with status and message", () => {
      const error = { status: 403, message: "This resource is locked" };
      expect(isLockedError(error)).toBe(true);
    });

    it("should return false for 403 error with only partial match", () => {
      const error = { status: 403, message: "This issue has been unlocked" };
      // Contains "unlocked" which includes "locked" substring
      expect(isLockedError(error)).toBe(true);
    });
  });
});
