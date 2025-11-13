import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

describe("sanitize_content.cjs", () => {
  let mockCore;
  let sanitizeContent;

  beforeEach(async () => {
    // Mock core actions methods
    mockCore = {
      debug: vi.fn(),
      info: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),
    };
    global.core = mockCore;

    // Import the module
    const module = await import("./sanitize_content.cjs");
    sanitizeContent = module.sanitizeContent;
  });

  afterEach(() => {
    delete global.core;
    delete process.env.GH_AW_ALLOWED_DOMAINS;
    delete process.env.GH_AW_COMMAND;
    delete process.env.GITHUB_SERVER_URL;
    delete process.env.GITHUB_API_URL;
  });

  describe("basic sanitization", () => {
    it("should return empty string for null or undefined input", () => {
      expect(sanitizeContent(null)).toBe("");
      expect(sanitizeContent(undefined)).toBe("");
    });

    it("should return empty string for non-string input", () => {
      expect(sanitizeContent(123)).toBe("");
      expect(sanitizeContent({})).toBe("");
      expect(sanitizeContent([])).toBe("");
    });

    it("should trim whitespace", () => {
      expect(sanitizeContent("  hello world  ")).toBe("hello world");
      expect(sanitizeContent("\n\thello\n\t")).toBe("hello");
    });

    it("should preserve normal text", () => {
      expect(sanitizeContent("Hello, this is normal text.")).toBe("Hello, this is normal text.");
    });
  });

  describe("command neutralization", () => {
    beforeEach(() => {
      process.env.GH_AW_COMMAND = "bot";
    });

    it("should neutralize command at start of text", () => {
      const result = sanitizeContent("/bot do something");
      expect(result).toBe("`/bot` do something");
    });

    it("should neutralize command after whitespace", () => {
      const result = sanitizeContent("  /bot do something");
      expect(result).toBe("`/bot` do something");
    });

    it("should not neutralize command in middle of text", () => {
      const result = sanitizeContent("hello /bot world");
      expect(result).toBe("hello /bot world");
    });

    it("should handle special regex characters in command name", () => {
      process.env.GH_AW_COMMAND = "my-bot+test";
      const result = sanitizeContent("/my-bot+test action");
      expect(result).toBe("`/my-bot+test` action");
    });

    it("should not neutralize when no command is set", () => {
      delete process.env.GH_AW_COMMAND;
      const result = sanitizeContent("/bot do something");
      expect(result).toBe("/bot do something");
    });
  });

  describe("@mention neutralization", () => {
    it("should neutralize @mentions", () => {
      const result = sanitizeContent("Hello @user");
      expect(result).toBe("Hello `@user`");
    });

    it("should neutralize @org/team mentions", () => {
      const result = sanitizeContent("Hello @myorg/myteam");
      expect(result).toBe("Hello `@myorg/myteam`");
    });

    it("should not neutralize @mentions already in backticks", () => {
      const result = sanitizeContent("Already `@user` mentioned");
      expect(result).toBe("Already `@user` mentioned");
    });

    it("should neutralize multiple @mentions", () => {
      const result = sanitizeContent("@user1 and @user2 are here");
      expect(result).toBe("`@user1` and `@user2` are here");
    });

    it("should not neutralize email addresses", () => {
      const result = sanitizeContent("Contact email@example.com");
      expect(result).toBe("Contact email@example.com");
    });
  });

  describe("XML comments removal", () => {
    it("should remove XML comments", () => {
      const result = sanitizeContent("Hello <!-- comment --> world");
      expect(result).toBe("Hello  world");
    });

    it("should remove malformed XML comments", () => {
      const result = sanitizeContent("Hello <!--! comment --!> world");
      expect(result).toBe("Hello  world");
    });

    it("should remove multiline XML comments", () => {
      const result = sanitizeContent("Hello <!-- multi\nline\ncomment --> world");
      expect(result).toBe("Hello  world");
    });
  });

  describe("XML/HTML tag conversion", () => {
    it("should convert opening tags to parentheses", () => {
      const result = sanitizeContent("Hello <div>world</div>");
      expect(result).toBe("Hello (div)world(/div)");
    });

    it("should convert tags with attributes to parentheses", () => {
      const result = sanitizeContent('<div class="test">content</div>');
      expect(result).toBe('(div class="test")content(/div)');
    });

    it("should preserve allowed safe tags", () => {
      const allowedTags = ["details", "summary", "code", "em", "b"];
      allowedTags.forEach(tag => {
        const result = sanitizeContent(`<${tag}>content</${tag}>`);
        expect(result).toBe(`<${tag}>content</${tag}>`);
      });
    });

    it("should convert self-closing tags", () => {
      const result = sanitizeContent("Hello <br/> world");
      expect(result).toBe("Hello (br/) world");
    });

    it("should handle CDATA sections", () => {
      const result = sanitizeContent("<![CDATA[<script>alert('xss')</script>]]>");
      expect(result).toBe("(![CDATA[(script)alert('xss')(/script)]])");
    });
  });

  describe("ANSI escape sequence removal", () => {
    it("should remove ANSI color codes", () => {
      const result = sanitizeContent("\x1b[31mred text\x1b[0m");
      expect(result).toBe("red text");
    });

    it("should remove various ANSI codes", () => {
      const result = sanitizeContent("\x1b[1;32mBold Green\x1b[0m");
      expect(result).toBe("Bold Green");
    });
  });

  describe("control character removal", () => {
    it("should remove control characters", () => {
      const result = sanitizeContent("test\x00\x01\x02\x03content");
      expect(result).toBe("testcontent");
    });

    it("should preserve newlines and tabs", () => {
      const result = sanitizeContent("test\ncontent\twith\ttabs");
      expect(result).toBe("test\ncontent\twith\ttabs");
    });

    it("should remove DEL character", () => {
      const result = sanitizeContent("test\x7Fcontent");
      expect(result).toBe("testcontent");
    });
  });

  describe("URL protocol sanitization", () => {
    it("should allow HTTPS URLs", () => {
      const result = sanitizeContent("Visit https://github.com");
      expect(result).toBe("Visit https://github.com");
    });

    it("should redact HTTP URLs", () => {
      const result = sanitizeContent("Visit http://example.com");
      expect(result).toContain("(redacted)");
      expect(mockCore.info).toHaveBeenCalled();
    });

    it("should redact javascript: URLs", () => {
      const result = sanitizeContent("Click javascript:alert('xss')");
      expect(result).toContain("(redacted)");
    });

    it("should redact data: URLs", () => {
      const result = sanitizeContent("Image data:image/png;base64,abc123");
      expect(result).toContain("(redacted)");
    });

    it("should preserve file paths with colons", () => {
      const result = sanitizeContent("C:\\path\\to\\file");
      expect(result).toBe("C:\\path\\to\\file");
    });

    it("should preserve namespace patterns", () => {
      const result = sanitizeContent("std::vector::push_back");
      expect(result).toBe("std::vector::push_back");
    });
  });

  describe("URL domain filtering", () => {
    it("should allow default GitHub domains", () => {
      const urls = [
        "https://github.com/repo",
        "https://api.github.com/endpoint",
        "https://raw.githubusercontent.com/file",
        "https://example.github.io/page",
      ];

      urls.forEach(url => {
        const result = sanitizeContent(`Visit ${url}`);
        expect(result).toBe(`Visit ${url}`);
      });
    });

    it("should redact disallowed domains", () => {
      const result = sanitizeContent("Visit https://evil.com/malicious");
      expect(result).toContain("(redacted)");
      expect(mockCore.info).toHaveBeenCalled();
    });

    it("should use custom allowed domains from environment", () => {
      process.env.GH_AW_ALLOWED_DOMAINS = "example.com,trusted.net";
      const result = sanitizeContent("Visit https://example.com/page");
      expect(result).toBe("Visit https://example.com/page");
    });

    it("should extract and allow GitHub Enterprise domains", () => {
      process.env.GITHUB_SERVER_URL = "https://github.company.com";
      const result = sanitizeContent("Visit https://github.company.com/repo");
      expect(result).toBe("Visit https://github.company.com/repo");
    });

    it("should allow subdomains of allowed domains", () => {
      const result = sanitizeContent("Visit https://subdomain.github.com/page");
      expect(result).toBe("Visit https://subdomain.github.com/page");
    });

    it("should log redacted domains", () => {
      sanitizeContent("Visit https://verylongdomainnamefortest.com/page");
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Redacted URL:"));
      expect(mockCore.debug).toHaveBeenCalledWith(expect.stringContaining("Redacted URL (full):"));
    });
  });

  describe("bot trigger neutralization", () => {
    it("should neutralize 'fixes #123' patterns", () => {
      const result = sanitizeContent("This fixes #123");
      expect(result).toBe("This `fixes #123`");
    });

    it("should neutralize 'closes #456' patterns", () => {
      const result = sanitizeContent("PR closes #456");
      expect(result).toBe("PR `closes #456`");
    });

    it("should neutralize 'resolves #789' patterns", () => {
      const result = sanitizeContent("This resolves #789");
      expect(result).toBe("This `resolves #789`");
    });

    it("should handle various bot trigger verbs", () => {
      const triggers = ["fix", "fixes", "close", "closes", "resolve", "resolves"];
      triggers.forEach(verb => {
        const result = sanitizeContent(`This ${verb} #123`);
        expect(result).toBe(`This \`${verb} #123\``);
      });
    });

    it("should neutralize alphanumeric issue references", () => {
      const result = sanitizeContent("fixes #abc123def");
      expect(result).toBe("`fixes #abc123def`");
    });
  });

  describe("content truncation", () => {
    it("should truncate content exceeding max length", () => {
      const longContent = "x".repeat(600000);
      const result = sanitizeContent(longContent);

      expect(result.length).toBeLessThan(longContent.length);
      expect(result).toContain("[Content truncated due to length]");
    });

    it("should truncate content exceeding max lines", () => {
      const manyLines = Array(70000).fill("line").join("\n");
      const result = sanitizeContent(manyLines);

      expect(result.split("\n").length).toBeLessThan(70000);
      expect(result).toContain("[Content truncated due to line count]");
    });

    it("should respect custom max length parameter", () => {
      const content = "x".repeat(200);
      const result = sanitizeContent(content, 100);

      expect(result.length).toBeLessThanOrEqual(100 + 50); // +50 for truncation message
      expect(result).toContain("[Content truncated");
    });

    it("should not truncate short content", () => {
      const shortContent = "This is a short message";
      const result = sanitizeContent(shortContent);

      expect(result).toBe(shortContent);
      expect(result).not.toContain("[Content truncated");
    });
  });

  describe("combined sanitization", () => {
    it("should apply all sanitizations correctly", () => {
      const input = `  
        <!-- comment -->
        Hello @user, visit https://github.com
        <script>alert('xss')</script>
        This fixes #123
        \x1b[31mRed text\x1b[0m
      `;

      const result = sanitizeContent(input);

      expect(result).not.toContain("<!-- comment -->");
      expect(result).toContain("`@user`");
      expect(result).toContain("https://github.com");
      expect(result).not.toContain("<script>");
      expect(result).toContain("(script)");
      expect(result).toContain("`fixes #123`");
      expect(result).not.toContain("\x1b[31m");
      expect(result).toContain("Red text");
    });

    it("should handle malicious XSS attempts", () => {
      const maliciousInputs = [
        '<img src=x onerror="alert(1)">',
        'javascript:alert(document.cookie)',
        '<svg onload="alert(1)">',
        'data:text/html,<script>alert(1)</script>',
      ];

      maliciousInputs.forEach(input => {
        const result = sanitizeContent(input);
        expect(result).not.toContain("<img");
        expect(result).not.toContain("javascript:");
        expect(result).not.toContain("<svg");
        expect(result).not.toContain("data:");
      });
    });

    it("should preserve allowed HTML in safe context", () => {
      const input = "<details><summary>Click here</summary>Content</details>";
      const result = sanitizeContent(input);

      expect(result).toBe(input);
    });
  });

  describe("edge cases", () => {
    it("should handle empty string", () => {
      expect(sanitizeContent("")).toBe("");
    });

    it("should handle whitespace-only input", () => {
      expect(sanitizeContent("   \n\t  ")).toBe("");
    });

    it("should handle content with only control characters", () => {
      const result = sanitizeContent("\x00\x01\x02\x03");
      expect(result).toBe("");
    });

    it("should handle content with multiple consecutive spaces", () => {
      const result = sanitizeContent("hello     world");
      expect(result).toBe("hello     world");
    });

    it("should handle Unicode characters", () => {
      const result = sanitizeContent("Hello 世界 🌍");
      expect(result).toBe("Hello 世界 🌍");
    });

    it("should handle URLs in query parameters", () => {
      const input = "https://github.com/redirect?url=https://github.com/target";
      const result = sanitizeContent(input);

      expect(result).toContain("github.com");
      expect(result).not.toContain("(redacted)");
    });

    it("should handle nested backticks", () => {
      const result = sanitizeContent("Already `@user` and @other");
      expect(result).toBe("Already `@user` and `@other`");
    });
  });
});
