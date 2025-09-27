import { describe, it, expect, beforeEach, vi } from "vitest";
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

  // Input/state functions
  getInput: vi.fn(),
  getBooleanInput: vi.fn(),
  getMultilineInput: vi.fn(),
  getState: vi.fn(),
  saveState: vi.fn(),

  // Group functions
  startGroup: vi.fn(),
  endGroup: vi.fn(),
  group: vi.fn(),

  // Summary object with chainable methods
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(),
  },
};

const mockContext = {
  repo: {
    owner: "test-owner",
    repo: "test-repo",
  },
};

// Set up global mocks
global.core = mockCore;
global.context = mockContext;

// Mock child_process execSync
const mockExecSync = vi.fn();
vi.mock("child_process", () => ({
  execSync: mockExecSync,
}));

// Mock fs functions
vi.mock("fs", () => ({
  readFileSync: vi.fn(),
  writeFileSync: vi.fn(),
  existsSync: vi.fn(),
  mkdirSync: vi.fn(),
}));

describe("edit_wiki.cjs", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    delete process.env.GITHUB_AW_AGENT_OUTPUT;
    delete process.env.GITHUB_AW_SAFE_OUTPUTS_STAGED;
    delete process.env.GITHUB_WORKFLOW_NAME;
    delete process.env.GITHUB_AW_WIKI_ALLOWED_PATHS;
    delete process.env.GITHUB_AW_WIKI_MAX;
  });

  describe("sanitizeMarkdown", () => {
    let sanitizeMarkdown;

    beforeEach(async () => {
      // Load the edit_wiki.cjs file and extract the sanitizeMarkdown function
      const editWikiPath = path.join(__dirname, "edit_wiki.cjs");
      const content = await fs.promises.readFile(editWikiPath, "utf8");

      // Extract and eval the sanitizeMarkdown function
      const funcMatch = content.match(/function sanitizeMarkdown\(content\)\s*{[^}]*(?:{[^}]*}[^}]*)*}/);
      if (funcMatch) {
        eval(funcMatch[0]);
        sanitizeMarkdown = global.sanitizeMarkdown || eval("(" + funcMatch[0] + ")");
      }
    });

    it("should remove script tags", () => {
      const input = "# Hello\n<script>alert('xss')</script>\nWorld";
      const result = sanitizeMarkdown(input);
      expect(result).not.toContain("<script>");
      expect(result).not.toContain("alert");
      expect(result).toContain("# Hello");
      expect(result).toContain("World");
    });

    it("should remove dangerous HTML attributes", () => {
      const input = '<div onclick="alert()" onmouseover="steal()">Content</div>';
      const result = sanitizeMarkdown(input);
      expect(result).not.toContain("onclick");
      expect(result).not.toContain("onmouseover");
      expect(result).toContain("Content");
    });

    it("should remove form elements", () => {
      const input = "# Form\n<form><input type='password'/></form>";
      const result = sanitizeMarkdown(input);
      expect(result).not.toContain("<form>");
      expect(result).not.toContain("<input>");
      expect(result).toContain("# Form");
    });

    it("should limit content length", () => {
      const longContent = "x".repeat(600 * 1024); // 600KB content
      const result = sanitizeMarkdown(longContent);
      expect(result.length).toBeLessThan(600 * 1024);
      expect(result).toContain("*Content truncated due to length limits*");
    });

    it("should handle null and undefined input", () => {
      expect(sanitizeMarkdown(null)).toBe("");
      expect(sanitizeMarkdown(undefined)).toBe("");
      expect(sanitizeMarkdown("")).toBe("");
    });

    it("should normalize line endings", () => {
      const input = "Line 1\r\nLine 2\rLine 3\n";
      const result = sanitizeMarkdown(input);
      expect(result).toBe("Line 1\nLine 2\nLine 3\n");
    });
  });

  describe("validatePagePath", () => {
    let validatePagePath;

    beforeEach(async () => {
      // Load the edit_wiki.cjs file and extract the validatePagePath function
      const editWikiPath = path.join(__dirname, "edit_wiki.cjs");
      const content = await fs.promises.readFile(editWikiPath, "utf8");

      // Extract and eval the validatePagePath function
      const funcMatch = content.match(/function validatePagePath\([^)]*\)\s*{[^}]*(?:{[^}]*}[^}]*)*}/);
      if (funcMatch) {
        eval(funcMatch[0]);
        validatePagePath = global.validatePagePath || eval("(" + funcMatch[0] + ")");
      }
    });

    it("should validate paths against allowed patterns", () => {
      const allowedPaths = ["docs/", "wiki/"];

      expect(validatePagePath("docs/readme", allowedPaths, "workflow")).toBe(true);
      expect(validatePagePath("wiki/help", allowedPaths, "workflow")).toBe(true);
      expect(validatePagePath("docs/api/guide", allowedPaths, "workflow")).toBe(true);

      expect(validatePagePath("src/code", allowedPaths, "workflow")).toBe(false);
      expect(validatePagePath("config", allowedPaths, "workflow")).toBe(false);
    });

    it("should use workflow name as default when no allowed paths", () => {
      expect(validatePagePath("workflow/page", [], "workflow")).toBe(true);
      expect(validatePagePath("workflow/sub/page", [], "workflow")).toBe(true);
      expect(validatePagePath("other/page", [], "workflow")).toBe(false);
    });

    it("should handle path normalization", () => {
      const allowedPaths = ["docs/"];

      expect(validatePagePath("/docs/readme", allowedPaths, "workflow")).toBe(true);
      expect(validatePagePath("docs/readme/", allowedPaths, "workflow")).toBe(true);
      expect(validatePagePath("//docs//readme//", allowedPaths, "workflow")).toBe(true);
    });

    it("should reject invalid paths", () => {
      expect(validatePagePath("", ["docs/"], "workflow")).toBe(false);
      expect(validatePagePath(null, ["docs/"], "workflow")).toBe(false);
      expect(validatePagePath(undefined, ["docs/"], "workflow")).toBe(false);
    });
  });

  describe("main function", () => {
    beforeEach(() => {
      // Reset mocks
      vi.clearAllMocks();
      mockExecSync.mockImplementation(() => "");
      fs.existsSync.mockReturnValue(true);
    });

    it("should handle missing agent output", async () => {
      delete process.env.GITHUB_AW_AGENT_OUTPUT;

      // Import and run the edit_wiki script
      await import("./edit_wiki.cjs");

      expect(mockCore.info).toHaveBeenCalledWith("No GITHUB_AW_AGENT_OUTPUT environment variable found");
    });

    it("should handle empty agent output", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = "";

      await import("./edit_wiki.cjs");

      expect(mockCore.info).toHaveBeenCalledWith("Agent output content is empty");
    });

    it("should handle invalid JSON in agent output", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = "invalid json";

      await import("./edit_wiki.cjs");

      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Error parsing agent output JSON"));
    });

    it("should handle no edit-wiki items", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          { type: "create-issue", title: "Test" },
          { type: "add-comment", body: "Comment" },
        ],
      });

      await import("./edit_wiki.cjs");

      expect(mockCore.info).toHaveBeenCalledWith("No edit-wiki items found in agent output");
    });

    it("should handle staged mode", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [{ type: "edit-wiki", path: "docs/test", content: "# Test Page" }],
      });
      process.env.GITHUB_AW_SAFE_OUTPUTS_STAGED = "true";
      process.env.GITHUB_WORKFLOW_NAME = "test-workflow";

      await import("./edit_wiki.cjs");

      expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("🎭 Staged Mode: Edit Wiki Pages Preview"));
      expect(mockCore.summary.write).toHaveBeenCalled();
      expect(mockCore.info).toHaveBeenCalledWith("📝 Wiki edit preview written to step summary");
    });

    it("should validate wiki items", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          { type: "edit-wiki", path: "", content: "# Test" }, // Missing path
          { type: "edit-wiki", path: "docs/test", content: "" }, // Missing content
          { type: "edit-wiki", path: "restricted/path", content: "# Test" }, // Restricted path
          { type: "edit-wiki", path: "test-workflow/valid", content: "# Valid" }, // Valid
        ],
      });
      process.env.GITHUB_WORKFLOW_NAME = "test-workflow";

      await import("./edit_wiki.cjs");

      expect(mockCore.warning).toHaveBeenCalledWith("Wiki edit 1: Missing path, skipping");
      expect(mockCore.warning).toHaveBeenCalledWith("Wiki edit 2: Missing content, skipping");
      expect(mockCore.warning).toHaveBeenCalledWith("Wiki edit 3: Path 'restricted/path' not allowed, skipping");
    });

    it("should respect max limit", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          { type: "edit-wiki", path: "test-workflow/page1", content: "# Page 1" },
          { type: "edit-wiki", path: "test-workflow/page2", content: "# Page 2" },
          { type: "edit-wiki", path: "test-workflow/page3", content: "# Page 3" },
        ],
      });
      process.env.GITHUB_WORKFLOW_NAME = "test-workflow";
      process.env.GITHUB_AW_WIKI_MAX = "2";

      await import("./edit_wiki.cjs");

      // Should only process 2 items due to max limit
      expect(mockCore.info).toHaveBeenCalledWith("Processing 2 valid wiki edit(s)");
    });
  });
});
