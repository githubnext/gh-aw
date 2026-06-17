import { describe, it, expect } from "vitest";
import { getErrorMessage, isHtmlContent, isLockedError, isRateLimitError } from "./error_helpers.cjs";

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

    it("should sanitize HTML DOCTYPE error response with status", () => {
      const html = "<!DOCTYPE html>\n<html><head><title>Unicorn!</title></head><body>...</body></html>";
      const error = new Error(html);
      /** @type {any} */ error.status = 504;
      expect(getErrorMessage(error)).toBe("GitHub returned an unexpected HTML response (HTTP 504)");
    });

    it("should sanitize HTML DOCTYPE error response without status", () => {
      const html = "<!DOCTYPE html>\n<html><head><title>Unicorn!</title></head><body>...</body></html>";
      const error = new Error(html);
      expect(getErrorMessage(error)).toBe("GitHub returned an unexpected HTML response");
    });

    it("should sanitize bare <html> error response with status", () => {
      const html = "<html><head><title>Service Unavailable</title></head><body>...</body></html>";
      const error = { message: html, status: 503 };
      expect(getErrorMessage(error)).toBe("GitHub returned an unexpected HTML response (HTTP 503)");
    });

    it("should sanitize html with leading whitespace", () => {
      const html = "  \n<!DOCTYPE html><html>...</html>";
      const error = new Error(html);
      expect(getErrorMessage(error)).toBe("GitHub returned an unexpected HTML response");
    });

    it("should sanitize raw HTML string throw", () => {
      const html = "<!DOCTYPE html><html><body>Unicorn</body></html>";
      expect(getErrorMessage(html)).toBe("GitHub returned an unexpected HTML response");
    });

    it("should not sanitize plain-text error messages that happen to mention html", () => {
      const error = new Error("Validation failed: invalid html content provided");
      expect(getErrorMessage(error)).toBe("Validation failed: invalid html content provided");
    });
  });

  describe("isHtmlContent", () => {
    it("should return true for DOCTYPE HTML string", () => {
      expect(isHtmlContent("<!DOCTYPE html><html></html>")).toBe(true);
    });

    it("should return true for bare html tag", () => {
      expect(isHtmlContent("<html><head></head><body></body></html>")).toBe(true);
    });

    it("should return true with leading whitespace", () => {
      expect(isHtmlContent("\n  <!DOCTYPE html>...")).toBe(true);
    });

    it("should return true for case-insensitive DOCTYPE", () => {
      expect(isHtmlContent("<!doctype HTML><html></html>")).toBe(true);
    });

    it("should return false for plain text", () => {
      expect(isHtmlContent("Resource not accessible by integration")).toBe(false);
    });

    it("should return false for JSON-like content", () => {
      expect(isHtmlContent('{"message":"Not Found","documentation_url":"..."}')).toBe(false);
    });

    it("should return false for empty string", () => {
      expect(isHtmlContent("")).toBe(false);
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

  describe("isRateLimitError", () => {
    it("should return true for 'API rate limit exceeded' message", () => {
      expect(isRateLimitError(new Error("API rate limit exceeded for installation"))).toBe(true);
    });

    it("should return true for 'rate limit exceeded' message", () => {
      expect(isRateLimitError(new Error("rate limit exceeded: please retry after 60 seconds"))).toBe(true);
    });

    it("should return true for mixed-case 'API Rate Limit' message", () => {
      expect(isRateLimitError(new Error("API Rate Limit exceeded"))).toBe(true);
    });

    it("should return false for unrelated API errors", () => {
      expect(isRateLimitError(new Error("Network connection error"))).toBe(false);
    });

    it("should return false for null error", () => {
      expect(isRateLimitError(null)).toBe(false);
    });

    it("should return false for undefined error", () => {
      expect(isRateLimitError(undefined)).toBe(false);
    });

    it("should return false for non-rate-limit 403 errors", () => {
      const error = new Error("Forbidden: insufficient permissions");
      expect(isRateLimitError(error)).toBe(false);
    });
  });
});
