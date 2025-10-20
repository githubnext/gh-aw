import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";

// Mock the global objects that GitHub Actions provides
const mockCore = {
  // Core logging functions
  debug: vi.fn(),
  info: vi.fn(),
  notice: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),

  // Core workflow functions
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  exportVariable: vi.fn(),
  setSecret: vi.fn(),

  // Input/state functions (less commonly used but included for completeness)
  getInput: vi.fn(),
  getBooleanInput: vi.fn(),
  getMultilineInput: vi.fn(),
  getState: vi.fn(),
  saveState: vi.fn(),

  // Group functions
  startGroup: vi.fn(),
  endGroup: vi.fn(),
  group: vi.fn(),

  // Other utility functions
  addPath: vi.fn(),
  setCommandEcho: vi.fn(),
  isDebug: vi.fn().mockReturnValue(false),
  getIDToken: vi.fn(),
  toPlatformPath: vi.fn(),
  toPosixPath: vi.fn(),
  toWin32Path: vi.fn(),

  // Summary object with chainable methods
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(),
  },
};

const mockGithub = {
  rest: {
    repos: {
      getCollaboratorPermissionLevel: vi.fn(),
    },
  },
};

const mockContext = {
  actor: "test-user",
  repo: {
    owner: "test-owner",
    repo: "test-repo",
  },
  eventName: "issues",
  payload: {},
};

// Set up global variables
global.core = mockCore;
global.github = mockGithub;
global.context = mockContext;

describe("compute_text.cjs", () => {
  let computeTextScript;
  let sanitizeContentFunction;

  beforeEach(() => {
    // Reset all mocks
    vi.clearAllMocks();

    // Reset context
    mockContext.eventName = "issues";
    mockContext.payload = {};

    // Reset environment variables
    delete process.env.GH_AW_ALLOWED_DOMAINS;

    // Read the script content
    const scriptPath = path.join(process.cwd(), "compute_text.cjs");
    computeTextScript = fs.readFileSync(scriptPath, "utf8");

    // Extract sanitizeContent function for unit testing
    // We need to eval the script to get access to the function
    const scriptWithExport = computeTextScript.replace(
      "await main();",
      "global.testSanitizeContent = sanitizeContent; global.testMain = main;"
    );
    eval(scriptWithExport);
    sanitizeContentFunction = global.testSanitizeContent;
  });

  describe("sanitizeContent function", () => {
    it("should handle null and undefined inputs", () => {
      expect(sanitizeContentFunction(null)).toBe("");
      expect(sanitizeContentFunction(undefined)).toBe("");
      expect(sanitizeContentFunction("")).toBe("");
    });

    it("should neutralize @mentions by wrapping in backticks", () => {
      const input = "Hello @user and @org/team";
      const result = sanitizeContentFunction(input);
      expect(result).toContain("`@user`");
      expect(result).toContain("`@org/team`");
    });

    it("should neutralize bot trigger phrases", () => {
      const input = "This fixes #123 and closes #456";
      const result = sanitizeContentFunction(input);
      expect(result).toContain("`fixes #123`");
      expect(result).toContain("`closes #456`");
    });

    it("should remove control characters", () => {
      const input = "Hello\x00\x01\x08world\x7F";
      const result = sanitizeContentFunction(input);
      expect(result).toBe("Helloworld");
    });

    it("should convert XML tags to parentheses format", () => {
      const input = 'Test <tag>content</tag> & "quotes"';
      const result = sanitizeContentFunction(input);
      expect(result).toContain("(tag)content(/tag)");
      expect(result).toContain("&");
      expect(result).toContain('"quotes"');
    });

    it("should handle self-closing XML tags without whitespace", () => {
      const input = 'Self-closing: <br/> <img src="test.jpg"/> <meta charset="utf-8"/>';
      const result = sanitizeContentFunction(input);
      expect(result).toContain("(br/)");
      expect(result).toContain('(img src="test.jpg"/)');
      expect(result).toContain('(meta charset="utf-8"/)');
    });

    it("should handle self-closing XML tags with whitespace", () => {
      const input = 'With spaces: <br /> <img src="test.jpg" /> <meta charset="utf-8" />';
      const result = sanitizeContentFunction(input);
      expect(result).toContain("(br /)");
      expect(result).toContain('(img src="test.jpg" /)');
      expect(result).toContain('(meta charset="utf-8" /)');
    });

    it("should handle XML tags with various whitespace patterns", () => {
      const input = 'Various: <div\tclass="test">content</div> <span\n  id="test">text</span>';
      const result = sanitizeContentFunction(input);
      expect(result).toContain('(div\tclass="test")content(/div)');
      expect(result).toContain('(span\n  id="test")text(/span)');
    });

    it("should redact non-https protocols", () => {
      const input = "Visit http://example.com or ftp://files.com";
      const result = sanitizeContentFunction(input);
      expect(result).toContain("(redacted)");
      expect(result).not.toContain("http://example.com");
    });

    it("should allow github.com domains", () => {
      const input = "Visit https://github.com/user/repo";
      const result = sanitizeContentFunction(input);
      expect(result).toContain("https://github.com/user/repo");
    });

    it("should redact unknown domains", () => {
      const input = "Visit https://evil.com/malware";
      const result = sanitizeContentFunction(input);
      expect(result).toContain("(redacted)");
      expect(result).not.toContain("evil.com");
    });

    it("should truncate long content", () => {
      const longContent = "a".repeat(600000); // Exceed 524288 limit
      const result = sanitizeContentFunction(longContent);
      expect(result.length).toBeLessThan(600000);
      expect(result).toContain("[Content truncated due to length]");
    });

    it("should truncate too many lines", () => {
      const manyLines = Array(70000).fill("line").join("\n"); // Exceed 65000 limit
      const result = sanitizeContentFunction(manyLines);
      expect(result.split("\n").length).toBeLessThan(70000);
      expect(result).toContain("[Content truncated due to line count]");
    });

    it("should remove ANSI escape sequences", () => {
      const input = "Hello \u001b[31mred\u001b[0m world";
      const result = sanitizeContentFunction(input);
      // ANSI sequences should be removed, allowing for possible differences in regex matching
      expect(result).toMatch(/Hello.*red.*world/);
      expect(result).not.toMatch(/\u001b\[/);
    });

    it("should respect custom allowed domains", () => {
      process.env.GH_AW_ALLOWED_DOMAINS = "example.com,trusted.org";
      const input = "Visit https://example.com and https://trusted.org and https://evil.com";
      const result = sanitizeContentFunction(input);
      expect(result).toContain("https://example.com");
      expect(result).toContain("https://trusted.org");
      expect(result).toContain("(redacted)"); // for evil.com
    });
  });

  describe("main function", () => {
    let testMain;

    beforeEach(() => {
      // Set up default successful permission check
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
        data: { permission: "admin" },
      });

      // Get the main function from global scope
      testMain = global.testMain;
    });

    it("should extract text from issue payload", async () => {
      mockContext.eventName = "issues";
      mockContext.payload = {
        issue: {
          title: "Test Issue",
          body: "Issue description",
        },
      };

      await testMain();

      expect(mockCore.setOutput).toHaveBeenCalledWith("text", "Test Issue\n\nIssue description");
    });

    it("should extract text from pull request payload", async () => {
      mockContext.eventName = "pull_request";
      mockContext.payload = {
        pull_request: {
          title: "Test PR",
          body: "PR description",
        },
      };

      await testMain();

      expect(mockCore.setOutput).toHaveBeenCalledWith("text", "Test PR\n\nPR description");
    });

    it("should extract text from issue comment payload", async () => {
      mockContext.eventName = "issue_comment";
      mockContext.payload = {
        comment: {
          body: "This is a comment",
        },
      };

      await testMain();

      expect(mockCore.setOutput).toHaveBeenCalledWith("text", "This is a comment");
    });

    it("should extract text from pull request target payload", async () => {
      mockContext.eventName = "pull_request_target";
      mockContext.payload = {
        pull_request: {
          title: "Test PR Target",
          body: "PR target description",
        },
      };

      await testMain();

      expect(mockCore.setOutput).toHaveBeenCalledWith("text", "Test PR Target\n\nPR target description");
    });

    it("should extract text from pull request review comment payload", async () => {
      mockContext.eventName = "pull_request_review_comment";
      mockContext.payload = {
        comment: {
          body: "Review comment",
        },
      };

      await testMain();

      expect(mockCore.setOutput).toHaveBeenCalledWith("text", "Review comment");
    });

    it("should extract text from pull request review payload", async () => {
      mockContext.eventName = "pull_request_review";
      mockContext.payload = {
        review: {
          body: "Review body",
        },
      };

      await testMain();

      expect(mockCore.setOutput).toHaveBeenCalledWith("text", "Review body");
    });

    it("should extract text from discussion payload", async () => {
      mockContext.eventName = "discussion";
      mockContext.payload = {
        discussion: {
          title: "Test Discussion",
          body: "Discussion description",
        },
      };

      await testMain();

      expect(mockCore.setOutput).toHaveBeenCalledWith("text", "Test Discussion\n\nDiscussion description");
    });

    it("should extract text from discussion comment payload", async () => {
      mockContext.eventName = "discussion_comment";
      mockContext.payload = {
        comment: {
          body: "Discussion comment text",
        },
      };

      await testMain();

      expect(mockCore.setOutput).toHaveBeenCalledWith("text", "Discussion comment text");
    });

    it("should handle unknown event types", async () => {
      mockContext.eventName = "unknown_event";
      mockContext.payload = {};

      await testMain();

      expect(mockCore.setOutput).toHaveBeenCalledWith("text", "");
    });

    it("should deny access for non-admin/maintain users", async () => {
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
        data: { permission: "read" },
      });

      mockContext.eventName = "issues";
      mockContext.payload = {
        issue: {
          title: "Test Issue",
          body: "Issue description",
        },
      };

      await testMain();

      expect(mockCore.setOutput).toHaveBeenCalledWith("text", "");
    });

    it("should sanitize extracted text before output", async () => {
      mockContext.eventName = "issues";
      mockContext.payload = {
        issue: {
          title: "Test @user fixes #123",
          body: "Visit https://evil.com",
        },
      };

      await testMain();

      const outputCall = mockCore.setOutput.mock.calls[0];
      expect(outputCall[1]).toContain("`@user`");
      expect(outputCall[1]).toContain("`fixes #123`");
      expect(outputCall[1]).toContain("(redacted)");
    });

    it("should handle missing title and body gracefully", async () => {
      mockContext.eventName = "issues";
      mockContext.payload = {
        issue: {}, // No title or body
      };

      await testMain();

      // Since empty strings get sanitized/trimmed, expect empty string
      expect(mockCore.setOutput).toHaveBeenCalledWith("text", "");
    });

    it("should handle null values in payload", async () => {
      mockContext.eventName = "issue_comment";
      mockContext.payload = {
        comment: {
          body: null,
        },
      };

      await testMain();

      expect(mockCore.setOutput).toHaveBeenCalledWith("text", "");
    });
  });
});
