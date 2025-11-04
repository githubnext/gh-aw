import { describe, it, expect } from "vitest";

// Import the function to test
const { sanitizeLabelContent } = require("./sanitize_label_content.cjs");

describe("sanitize_label_content.cjs", () => {
  describe("sanitizeLabelContent", () => {
    it("should return empty string for null input", () => {
      expect(sanitizeLabelContent(null)).toBe("");
    });

    it("should return empty string for undefined input", () => {
      expect(sanitizeLabelContent(undefined)).toBe("");
    });

    it("should return empty string for non-string input", () => {
      expect(sanitizeLabelContent(123)).toBe("");
      expect(sanitizeLabelContent({})).toBe("");
      expect(sanitizeLabelContent([])).toBe("");
    });

    it("should trim whitespace from input", () => {
      expect(sanitizeLabelContent("  test  ")).toBe("test");
      expect(sanitizeLabelContent("\n\ttest\n\t")).toBe("test");
    });

    it("should remove control characters", () => {
      const input = "test\x00\x01\x02\x03\x04\x05\x06\x07\x08label";
      expect(sanitizeLabelContent(input)).toBe("testlabel");
    });

    it("should remove DEL character (0x7F)", () => {
      const input = "test\x7Flabel";
      expect(sanitizeLabelContent(input)).toBe("testlabel");
    });

    it("should preserve newline character", () => {
      const input = "test\nlabel";
      expect(sanitizeLabelContent(input)).toBe("test\nlabel");
    });

    it("should remove ANSI escape codes", () => {
      const input = "\x1b[31mred text\x1b[0m";
      expect(sanitizeLabelContent(input)).toBe("red text");
    });

    it("should remove various ANSI codes", () => {
      const input = "\x1b[1;32mBold Green\x1b[0m\x1b[4mUnderline\x1b[0m";
      expect(sanitizeLabelContent(input)).toBe("Bold GreenUnderline");
    });

    it("should neutralize @mentions by wrapping in backticks", () => {
      expect(sanitizeLabelContent("Hello @user")).toBe("Hello `@user`");
      expect(sanitizeLabelContent("@user said something")).toBe("`@user` said something");
    });

    it("should neutralize @org/team mentions", () => {
      expect(sanitizeLabelContent("Hello @myorg/myteam")).toBe("Hello `@myorg/myteam`");
    });

    it("should not neutralize @mentions already in backticks", () => {
      const input = "Already `@user` handled";
      expect(sanitizeLabelContent(input)).toBe("Already `@user` handled");
    });

    it("should neutralize multiple @mentions", () => {
      const input = "@user1 and @user2 are here";
      expect(sanitizeLabelContent(input)).toBe("`@user1` and `@user2` are here");
    });

    it("should remove HTML special characters", () => {
      expect(sanitizeLabelContent("test<>&'\"label")).toBe("testlabel");
    });

    it("should remove less-than signs", () => {
      expect(sanitizeLabelContent("a < b")).toBe("a  b");
    });

    it("should remove greater-than signs", () => {
      expect(sanitizeLabelContent("a > b")).toBe("a  b");
    });

    it("should remove ampersands", () => {
      expect(sanitizeLabelContent("test & label")).toBe("test  label");
    });

    it("should remove single and double quotes", () => {
      expect(sanitizeLabelContent('test\'s "label"')).toBe("tests label");
    });

    it("should handle complex input with multiple sanitizations", () => {
      const input = "  @user \x1b[31mred\x1b[0m <tag> test&label  ";
      expect(sanitizeLabelContent(input)).toBe("`@user` red tag testlabel");
    });

    it("should handle empty string input", () => {
      expect(sanitizeLabelContent("")).toBe("");
    });

    it("should handle whitespace-only input", () => {
      expect(sanitizeLabelContent("   \n\t  ")).toBe("");
    });

    it("should preserve normal alphanumeric characters", () => {
      expect(sanitizeLabelContent("bug123")).toBe("bug123");
      expect(sanitizeLabelContent("feature-request")).toBe("feature-request");
    });

    it("should preserve hyphens and underscores", () => {
      expect(sanitizeLabelContent("test-label_123")).toBe("test-label_123");
    });

    it("should handle consecutive control characters", () => {
      const input = "test\x00\x01\x02\x03\x04\x05label";
      expect(sanitizeLabelContent(input)).toBe("testlabel");
    });

    it("should handle @mentions at various positions", () => {
      expect(sanitizeLabelContent("start @user end")).toBe("start `@user` end");
      expect(sanitizeLabelContent("@user at start")).toBe("`@user` at start");
      expect(sanitizeLabelContent("at end @user")).toBe("at end `@user`");
    });

    it("should not treat email-like patterns as @mentions after alphanumerics", () => {
      const input = "email@example.com";
      // The regex has [^\w`] which requires non-word character before @
      // so 'email@' won't match because 'l' is a word character
      expect(sanitizeLabelContent(input)).toBe("email@example.com");
    });

    it("should handle username edge cases", () => {
      // Valid GitHub usernames can be 1-39 chars, alphanumeric + hyphens
      expect(sanitizeLabelContent("@a")).toBe("`@a`");
      expect(sanitizeLabelContent("@user-name-123")).toBe("`@user-name-123`");
    });

    it("should combine all sanitization rules correctly", () => {
      const input = '  \x1b[31m@user\x1b[0m says <hello> & "goodbye"  ';
      expect(sanitizeLabelContent(input)).toBe("`@user` says hello  goodbye");
    });
  });
});
