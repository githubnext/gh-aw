// @ts-check
import { describe, it, expect, beforeEach, vi } from "vitest";
import { createRequire } from "module";
import fs from "fs";
import os from "os";
import path from "path";

const require = createRequire(import.meta.url);

// Create a temporary template file used by "new issue" tests
const tmpDir = os.tmpdir();
const testTemplatePath = path.join(tmpDir, "test_missing_issue_template.md");
fs.writeFileSync(testTemplatePath, "# Missing Items\n\n{{test_list}}\n");

// Mock globals before importing the module
const mockCore = {
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setOutput: vi.fn(),
  setFailed: vi.fn(),
};

const mockGithub = {
  rest: {
    search: {
      issuesAndPullRequests: vi.fn(),
    },
    issues: {
      create: vi.fn(),
      createComment: vi.fn(),
    },
  },
};

const mockContext = {
  repo: { owner: "test-owner", repo: "test-repo" },
};

globalThis.core = mockCore;
globalThis.github = mockGithub;
globalThis.context = mockContext;

const { buildMissingIssueHandler } = require("./missing_issue_helpers.cjs");

/**
 * Helper to build a minimal handler options object for testing
 * @param {Partial<Object>} overrides
 */
function makeOptions(overrides = {}) {
  return {
    handlerType: "create_test_issue",
    defaultTitlePrefix: "[test prefix]",
    itemsField: "test_items",
    templatePath: testTemplatePath,
    templateListKey: "test_list",
    buildCommentHeader: runUrl => [`## Test Header`, ``, `Items from [run](${runUrl}):`, ``],
    renderCommentItem: (item, index) => [`### ${index + 1}. ${item.name}`, `**Reason:** ${item.reason}`, ``],
    renderIssueItem: (item, index) => [`#### ${index + 1}. ${item.name}`, `**Reason:** ${item.reason}`, `**Reported at:** ${item.timestamp}`, ``],
    ...overrides,
  };
}

const defaultMessage = {
  workflow_name: "My Workflow",
  workflow_source: "my-workflow.md",
  workflow_source_url: "https://github.com/owner/repo/blob/main/my-workflow.md",
  run_url: "https://github.com/owner/repo/actions/runs/123",
  test_items: [{ name: "item-one", reason: "not found", timestamp: "2026-01-01T00:00:00Z" }],
};

describe("missing_issue_helpers.cjs - buildMissingIssueHandler", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("validation", () => {
    it("should return error when workflow_name is missing", async () => {
      const handler = await buildMissingIssueHandler(makeOptions())({});
      const result = await handler({ test_items: [{ name: "x", reason: "y" }] });

      expect(result.success).toBe(false);
      expect(result.error).toBe("Missing required field: workflow_name");
      expect(mockCore.warning).toHaveBeenCalledWith("Missing required field: workflow_name");
    });

    it("should return error when items field is missing", async () => {
      const handler = await buildMissingIssueHandler(makeOptions())({});
      const result = await handler({ workflow_name: "Test Workflow" });

      expect(result.success).toBe(false);
      expect(result.error).toBe("Missing or empty test_items array");
    });

    it("should return error when items array is empty", async () => {
      const handler = await buildMissingIssueHandler(makeOptions())({});
      const result = await handler({ workflow_name: "Test Workflow", test_items: [] });

      expect(result.success).toBe(false);
      expect(result.error).toBe("Missing or empty test_items array");
    });

    it("should return error when items field is not an array", async () => {
      const handler = await buildMissingIssueHandler(makeOptions())({});
      const result = await handler({ workflow_name: "Test Workflow", test_items: "not-an-array" });

      expect(result.success).toBe(false);
      expect(result.error).toBe("Missing or empty test_items array");
    });
  });

  describe("max count enforcement", () => {
    it("should skip processing when max count is reached", async () => {
      mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
        data: { total_count: 0, items: [] },
      });
      mockGithub.rest.issues.create.mockResolvedValue({
        data: { number: 1, html_url: "https://github.com/owner/repo/issues/1" },
      });

      const handler = await buildMissingIssueHandler(makeOptions())({ max: 1 });

      // First call should succeed (or fail with a non-max error)
      await handler(defaultMessage);

      // Second call should be rejected due to max count
      const result = await handler(defaultMessage);
      expect(result.success).toBe(false);
      expect(result.error).toBe("Max count of 1 reached");
      expect(mockCore.warning).toHaveBeenCalledWith("Skipping create_test_issue: max count of 1 reached");
    });

    it("should allow higher max counts", async () => {
      mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
        data: {
          total_count: 1,
          items: [{ number: 42, html_url: "https://github.com/owner/repo/issues/42" }],
        },
      });
      mockGithub.rest.issues.createComment.mockResolvedValue({ data: {} });

      const handler = await buildMissingIssueHandler(makeOptions())({ max: 3 });

      // First two calls should not hit the limit
      await handler(defaultMessage);
      await handler(defaultMessage);

      // processedCount is now 2, max is 3 - third should still work
      const result = await handler(defaultMessage);
      expect(result.success).toBe(true);
    });
  });

  describe("config extraction", () => {
    it("should use defaultTitlePrefix when no config.title_prefix provided", async () => {
      mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
        data: { total_count: 0, items: [] },
      });
      mockGithub.rest.issues.create.mockResolvedValue({
        data: { number: 10, html_url: "https://github.com/owner/repo/issues/10" },
      });

      const handler = await buildMissingIssueHandler(makeOptions({ defaultTitlePrefix: "[custom prefix]" }))({});
      await handler(defaultMessage);

      expect(mockGithub.rest.search.issuesAndPullRequests).toHaveBeenCalledWith(expect.objectContaining({ q: expect.stringContaining("[custom prefix] My Workflow") }));
    });

    it("should use config.title_prefix when provided", async () => {
      mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
        data: { total_count: 0, items: [] },
      });
      mockGithub.rest.issues.create.mockResolvedValue({
        data: { number: 10, html_url: "https://github.com/owner/repo/issues/10" },
      });

      const handler = await buildMissingIssueHandler(makeOptions())({ title_prefix: "[override]" });
      await handler(defaultMessage);

      expect(mockGithub.rest.search.issuesAndPullRequests).toHaveBeenCalledWith(expect.objectContaining({ q: expect.stringContaining("[override] My Workflow") }));
    });
  });

  describe("existing issue - add comment", () => {
    it("should add comment to existing issue and return updated action", async () => {
      mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
        data: {
          total_count: 1,
          items: [{ number: 99, html_url: "https://github.com/owner/repo/issues/99" }],
        },
      });
      mockGithub.rest.issues.createComment.mockResolvedValue({ data: {} });

      const handler = await buildMissingIssueHandler(makeOptions())({});
      const result = await handler(defaultMessage);

      expect(result.success).toBe(true);
      expect(result.action).toBe("updated");
      expect(result.issue_number).toBe(99);
      expect(mockGithub.rest.issues.createComment).toHaveBeenCalledWith(
        expect.objectContaining({
          owner: "test-owner",
          repo: "test-repo",
          issue_number: 99,
          body: expect.stringContaining("## Test Header"),
        })
      );
    });

    it("should include rendered item in comment body", async () => {
      mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
        data: {
          total_count: 1,
          items: [{ number: 55, html_url: "https://github.com/owner/repo/issues/55" }],
        },
      });
      mockGithub.rest.issues.createComment.mockResolvedValue({ data: {} });

      const handler = await buildMissingIssueHandler(makeOptions())({});
      await handler(defaultMessage);

      const commentBody = mockGithub.rest.issues.createComment.mock.calls[0][0].body;
      expect(commentBody).toContain("### 1. item-one");
      expect(commentBody).toContain("**Reason:** not found");
    });
  });

  describe("new issue - create issue", () => {
    it("should create a new issue when no existing issue found", async () => {
      mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
        data: { total_count: 0, items: [] },
      });
      mockGithub.rest.issues.create.mockResolvedValue({
        data: { number: 77, html_url: "https://github.com/owner/repo/issues/77" },
      });

      const handler = await buildMissingIssueHandler(makeOptions())({});
      const result = await handler(defaultMessage);

      expect(result.success).toBe(true);
      expect(result.action).toBe("created");
      expect(result.issue_number).toBe(77);
      expect(mockGithub.rest.issues.create).toHaveBeenCalled();
    });

    it("should apply labels when configured", async () => {
      mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
        data: { total_count: 0, items: [] },
      });
      mockGithub.rest.issues.create.mockResolvedValue({
        data: { number: 77, html_url: "https://github.com/owner/repo/issues/77" },
      });

      const handler = await buildMissingIssueHandler(makeOptions())({ labels: "bug,help wanted" });
      await handler(defaultMessage);

      expect(mockGithub.rest.issues.create).toHaveBeenCalledWith(expect.objectContaining({ labels: ["bug", "help wanted"] }));
    });

    it("should apply array labels when configured", async () => {
      mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
        data: { total_count: 0, items: [] },
      });
      mockGithub.rest.issues.create.mockResolvedValue({
        data: { number: 77, html_url: "https://github.com/owner/repo/issues/77" },
      });

      const handler = await buildMissingIssueHandler(makeOptions())({ labels: ["bug", "needs-triage"] });
      await handler(defaultMessage);

      expect(mockGithub.rest.issues.create).toHaveBeenCalledWith(expect.objectContaining({ labels: ["bug", "needs-triage"] }));
    });
  });

  describe("error handling", () => {
    it("should return error result when GitHub API throws", async () => {
      mockGithub.rest.search.issuesAndPullRequests.mockRejectedValue(new Error("API rate limit exceeded"));

      const handler = await buildMissingIssueHandler(makeOptions())({});
      const result = await handler(defaultMessage);

      expect(result.success).toBe(false);
      expect(result.error).toContain("API rate limit exceeded");
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Failed to create or update issue"));
    });
  });
});
