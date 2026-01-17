import { describe, it, expect } from "vitest";
import { getErrorMessage, getErrorMessageWithoutPath } from "./error_helpers.cjs";

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

  describe("getErrorMessageWithoutPath", () => {
    it("should extract error code and description from EACCES error", () => {
      const error = new Error("EACCES: permission denied, open '/tmp/gh-aw/mcp-logs/mcp-gateway.log'");
      expect(getErrorMessageWithoutPath(error)).toBe("EACCES: permission denied");
    });

    it("should extract error code and description from ENOENT error", () => {
      const error = new Error("ENOENT: no such file or directory, open '/some/file.txt'");
      expect(getErrorMessageWithoutPath(error)).toBe("ENOENT: no such file or directory");
    });

    it("should extract error code and description from EISDIR error", () => {
      const error = new Error("EISDIR: illegal operation on a directory, read");
      expect(getErrorMessageWithoutPath(error)).toBe("EISDIR: illegal operation on a directory");
    });

    it("should handle error with single word after colon", () => {
      const error = new Error("EPERM: operation not permitted, unlink '/path/to/file'");
      expect(getErrorMessageWithoutPath(error)).toBe("EPERM: operation not permitted");
    });

    it("should return full message for non-filesystem errors", () => {
      const error = new Error("Network request failed");
      expect(getErrorMessageWithoutPath(error)).toBe("Network request failed");
    });

    it("should handle error messages without comma", () => {
      const error = new Error("UNKNOWN: unknown error");
      expect(getErrorMessageWithoutPath(error)).toBe("UNKNOWN: unknown error");
    });

    it("should handle plain string errors", () => {
      expect(getErrorMessageWithoutPath("Simple error")).toBe("Simple error");
    });

    it("should handle object with message property", () => {
      const error = { message: "EACCES: permission denied, write '/file'" };
      expect(getErrorMessageWithoutPath(error)).toBe("EACCES: permission denied");
    });
  });
});
