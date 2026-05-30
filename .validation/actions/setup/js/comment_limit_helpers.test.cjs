// @ts-check
import { describe, it, expect } from "vitest";

describe("comment_limit_helpers", () => {
  let MAX_COMMENT_LENGTH, MAX_MENTIONS, MAX_LINKS, enforceCommentLimits;

  // Import the module
  beforeEach(async () => {
    const module = await import("./comment_limit_helpers.cjs");
    MAX_COMMENT_LENGTH = module.MAX_COMMENT_LENGTH;
    MAX_MENTIONS = module.MAX_MENTIONS;
    MAX_LINKS = module.MAX_LINKS;
    enforceCommentLimits = module.enforceCommentLimits;
  });

  describe("constants", () => {
    it("should export MAX_COMMENT_LENGTH constant", () => {
      expect(MAX_COMMENT_LENGTH).toBe(65536);
    });

    it("should export MAX_MENTIONS constant", () => {
      expect(MAX_MENTIONS).toBe(10);
    });

    it("should export MAX_LINKS constant", () => {
      expect(MAX_LINKS).toBe(50);
    });
  });

  describe("enforceCommentLimits", () => {
    it("should accept valid comment body", () => {
      const validBody = "This is a valid comment with reasonable length";
      expect(() => enforceCommentLimits(validBody)).not.toThrow();
    });

    it("should accept comment at exactly maximum length", () => {
      const exactBody = "a".repeat(MAX_COMMENT_LENGTH);
      expect(() => enforceCommentLimits(exactBody)).not.toThrow();
    });

    it("should reject comment exceeding maximum length", () => {
      const longBody = "a".repeat(MAX_COMMENT_LENGTH + 1);
      expect(() => enforceCommentLimits(longBody)).toThrow(/E006.*maximum length/i);
    });

    it("should accept comment with exactly maximum mentions", () => {
      const mentions = Array.from({ length: MAX_MENTIONS }, (_, i) => `@user${i}`).join(" ");
      expect(() => enforceCommentLimits(mentions)).not.toThrow();
    });

    it("should reject comment exceeding maximum mentions", () => {
      const mentions = Array.from({ length: MAX_MENTIONS + 1 }, (_, i) => `@user${i}`).join(" ");
      expect(() => enforceCommentLimits(mentions)).toThrow(/E007.*mentions/i);
    });

    it("should accept comment with exactly maximum links", () => {
      const links = Array.from({ length: MAX_LINKS }, (_, i) => `https://example.com/${i}`).join(" ");
      expect(() => enforceCommentLimits(links)).not.toThrow();
    });

    it("should reject comment exceeding maximum links", () => {
      const links = Array.from({ length: MAX_LINKS + 1 }, (_, i) => `https://example.com/${i}`).join(" ");
      expect(() => enforceCommentLimits(links)).toThrow(/E008.*links/i);
    });

    it("should count both http and https links", () => {
      const httpsLinks = Array.from({ length: 30 }, (_, i) => `https://example.com/${i}`).join(" ");
      const httpLinks = Array.from({ length: 21 }, (_, i) => `http://example.org/${i}`).join(" ");
      const mixed = `${httpsLinks} ${httpLinks}`;
      expect(() => enforceCommentLimits(mixed)).toThrow(/E008.*51.*links/i);
    });

    it("should include actual values in error messages", () => {
      const longBody = "a".repeat(70000);
      try {
        enforceCommentLimits(longBody);
        expect.fail("Should have thrown error");
      } catch (error) {
        expect(error.message).toContain("E006");
        expect(error.message).toContain("65536"); // Max length
        expect(error.message).toContain("70000"); // Actual length
      }
    });

    it("should accept empty string", () => {
      expect(() => enforceCommentLimits("")).not.toThrow();
    });

    it("should handle comment with no mentions", () => {
      const body = "Comment without any user mentions";
      expect(() => enforceCommentLimits(body)).not.toThrow();
    });

    it("should handle comment with no links", () => {
      const body = "Comment without any links or URLs";
      expect(() => enforceCommentLimits(body)).not.toThrow();
    });
  });
});
