import { describe, it, expect, beforeEach, vi } from "vitest";

// Mock the global objects that GitHub Actions provides
const mockCore = {
  info: vi.fn(),
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(),
  },
};

// Set up global mocks before importing the module
globalThis.core = mockCore;

const { generateStagedPreview } = await import("./staged_preview.cjs");

describe("staged_preview.cjs", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("generateStagedPreview", () => {
    it("should generate preview with single item", async () => {
      const options = {
        title: "Create Issues",
        description: "The following issues would be created if staged mode was disabled:",
        items: [
          {
            title: "Test Issue",
            body: "Test body",
            labels: ["bug", "enhancement"],
          },
        ],
        renderItem: (item, index) => {
          let content = `#### Issue ${index + 1}\n`;
          content += `**Title:** ${item.title || "No title provided"}\n\n`;
          if (item.body) {
            content += `**Body:**\n${item.body}\n\n`;
          }
          if (item.labels && item.labels.length > 0) {
            content += `**Labels:** ${item.labels.join(", ")}\n\n`;
          }
          return content;
        },
      };

      await generateStagedPreview(options);

      expect(mockCore.summary.addRaw).toHaveBeenCalledTimes(1);
      expect(mockCore.summary.write).toHaveBeenCalledTimes(1);

      const summaryContent = mockCore.summary.addRaw.mock.calls[0][0];
      expect(summaryContent).toContain("## 🎭 Staged Mode: Create Issues Preview");
      expect(summaryContent).toContain("The following issues would be created if staged mode was disabled:");
      expect(summaryContent).toContain("#### Issue 1");
      expect(summaryContent).toContain("**Title:** Test Issue");
      expect(summaryContent).toContain("**Body:**\nTest body");
      expect(summaryContent).toContain("**Labels:** bug, enhancement");
      expect(summaryContent).toContain("---");

      // Verify that summary content is logged to core.info
      expect(mockCore.info).toHaveBeenCalledWith(summaryContent);
      expect(mockCore.info).toHaveBeenCalledWith("📝 Create Issues preview written to step summary");
    });

    it("should generate preview with multiple items", async () => {
      const options = {
        title: "Update Issues",
        description: "The following issue updates would be applied if staged mode was disabled:",
        items: [
          { issue_number: 1, title: "New Title 1", status: "open" },
          { issue_number: 2, title: "New Title 2", status: "closed" },
          { issue_number: 3, body: "New Body 3" },
        ],
        renderItem: (item, index) => {
          let content = `#### Issue Update ${index + 1}\n`;
          if (item.issue_number) {
            content += `**Target Issue:** #${item.issue_number}\n\n`;
          }
          if (item.title !== undefined) {
            content += `**New Title:** ${item.title}\n\n`;
          }
          if (item.body !== undefined) {
            content += `**New Body:**\n${item.body}\n\n`;
          }
          if (item.status !== undefined) {
            content += `**New Status:** ${item.status}\n\n`;
          }
          return content;
        },
      };

      await generateStagedPreview(options);

      const summaryContent = mockCore.summary.addRaw.mock.calls[0][0];
      expect(summaryContent).toContain("## 🎭 Staged Mode: Update Issues Preview");
      expect(summaryContent).toContain("#### Issue Update 1");
      expect(summaryContent).toContain("**Target Issue:** #1");
      expect(summaryContent).toContain("**New Title:** New Title 1");
      expect(summaryContent).toContain("**New Status:** open");
      expect(summaryContent).toContain("#### Issue Update 2");
      expect(summaryContent).toContain("**Target Issue:** #2");
      expect(summaryContent).toContain("**New Status:** closed");
      expect(summaryContent).toContain("#### Issue Update 3");
      expect(summaryContent).toContain("**New Body:**\nNew Body 3");

      // Check that all items are separated by dividers
      const dividerCount = (summaryContent.match(/---/g) || []).length;
      expect(dividerCount).toBe(3);
    });

    it("should handle add labels preview", async () => {
      const options = {
        title: "Add Labels",
        description: "The following labels would be added if staged mode was disabled:",
        items: [
          {
            item_number: 42,
            labels: ["bug", "enhancement", "good first issue"],
          },
        ],
        renderItem: item => {
          let content = "";
          if (item.item_number) {
            content += `**Target Issue:** #${item.item_number}\n\n`;
          } else {
            content += `**Target:** Current issue/PR\n\n`;
          }
          if (item.labels && item.labels.length > 0) {
            content += `**Labels to add:** ${item.labels.join(", ")}\n\n`;
          }
          return content;
        },
      };

      await generateStagedPreview(options);

      const summaryContent = mockCore.summary.addRaw.mock.calls[0][0];
      expect(summaryContent).toContain("## 🎭 Staged Mode: Add Labels Preview");
      expect(summaryContent).toContain("**Target Issue:** #42");
      expect(summaryContent).toContain("**Labels to add:** bug, enhancement, good first issue");
    });

    it("should handle PR review comments preview", async () => {
      const options = {
        title: "Create PR Review Comments",
        description: "The following review comments would be created if staged mode was disabled:",
        items: [
          {
            pull_request_number: 123,
            path: "src/main.js",
            line: 42,
            start_line: 40,
            side: "RIGHT",
            body: "This needs improvement",
          },
          {
            path: "src/utils.js",
            line: 10,
            side: "LEFT",
            body: "Consider refactoring",
          },
        ],
        renderItem: (item, index) => {
          const getRepositoryUrl = () => "https://github.com/test/repo";
          let content = `#### Review Comment ${index + 1}\n`;
          if (item.pull_request_number) {
            const repoUrl = getRepositoryUrl();
            const pullUrl = `${repoUrl}/pull/${item.pull_request_number}`;
            content += `**Target PR:** [#${item.pull_request_number}](${pullUrl})\n\n`;
          } else {
            content += `**Target:** Current PR\n\n`;
          }
          content += `**File:** ${item.path || "No path provided"}\n\n`;
          content += `**Line:** ${item.line || "No line provided"}\n\n`;
          if (item.start_line) {
            content += `**Start Line:** ${item.start_line}\n\n`;
          }
          content += `**Side:** ${item.side || "RIGHT"}\n\n`;
          content += `**Body:**\n${item.body || "No content provided"}\n\n`;
          return content;
        },
      };

      await generateStagedPreview(options);

      const summaryContent = mockCore.summary.addRaw.mock.calls[0][0];
      expect(summaryContent).toContain("## 🎭 Staged Mode: Create PR Review Comments Preview");
      expect(summaryContent).toContain("#### Review Comment 1");
      expect(summaryContent).toContain("**Target PR:** [#123](https://github.com/test/repo/pull/123)");
      expect(summaryContent).toContain("**File:** src/main.js");
      expect(summaryContent).toContain("**Line:** 42");
      expect(summaryContent).toContain("**Start Line:** 40");
      expect(summaryContent).toContain("**Side:** RIGHT");
      expect(summaryContent).toContain("**Body:**\nThis needs improvement");
      expect(summaryContent).toContain("#### Review Comment 2");
      expect(summaryContent).toContain("**Target:** Current PR");
      expect(summaryContent).toContain("**File:** src/utils.js");
      expect(summaryContent).toContain("**Side:** LEFT");
    });

    it("should handle push to PR branch preview with complex data", async () => {
      const options = {
        title: "Push to PR Branch",
        description: "The following changes would be pushed if staged mode was disabled:",
        items: [
          {
            target: "feature-branch",
            commit_message: "Update implementation",
            has_patch: true,
            patch_size: 150,
          },
        ],
        renderItem: item => {
          let content = "";
          content += `**Target:** ${item.target}\n\n`;
          if (item.commit_message) {
            content += `**Commit Message:** ${item.commit_message}\n\n`;
          }
          if (item.has_patch) {
            content += `**Changes:** Patch file exists with ${item.patch_size} lines\n\n`;
          }
          return content;
        },
      };

      await generateStagedPreview(options);

      const summaryContent = mockCore.summary.addRaw.mock.calls[0][0];
      expect(summaryContent).toContain("## 🎭 Staged Mode: Push to PR Branch Preview");
      expect(summaryContent).toContain("**Target:** feature-branch");
      expect(summaryContent).toContain("**Commit Message:** Update implementation");
      expect(summaryContent).toContain("**Changes:** Patch file exists with 150 lines");
    });

    it("should handle empty items array", async () => {
      const options = {
        title: "Test Preview",
        description: "Nothing to preview:",
        items: [],
        renderItem: () => "",
      };

      await generateStagedPreview(options);

      const summaryContent = mockCore.summary.addRaw.mock.calls[0][0];
      expect(summaryContent).toContain("## 🎭 Staged Mode: Test Preview Preview");
      expect(summaryContent).toContain("Nothing to preview:");
      // Should not have any dividers since there are no items
      expect(summaryContent).not.toContain("---");
    });

    it("should handle custom renderItem function with no divider", async () => {
      const options = {
        title: "Custom Preview",
        description: "Custom items:",
        items: [{ name: "item1" }, { name: "item2" }],
        renderItem: item => {
          return `- ${item.name}\n`;
        },
      };

      await generateStagedPreview(options);

      const summaryContent = mockCore.summary.addRaw.mock.calls[0][0];
      expect(summaryContent).toContain("- item1");
      expect(summaryContent).toContain("- item2");
      // Dividers should still be added after each item by the function
      const dividerCount = (summaryContent.match(/---/g) || []).length;
      expect(dividerCount).toBe(2);
    });

    it("should properly chain summary methods", async () => {
      const options = {
        title: "Test",
        description: "Test description",
        items: [{ test: "data" }],
        renderItem: () => "Test content\n",
      };

      await generateStagedPreview(options);

      expect(mockCore.summary.addRaw).toHaveBeenCalledTimes(1);
      expect(mockCore.summary.addRaw).toHaveReturnedWith(mockCore.summary);
      expect(mockCore.summary.write).toHaveBeenCalledTimes(1);
    });

    it("should include index in renderItem callback", async () => {
      const renderItemSpy = vi.fn((item, index) => `Item ${index + 1}\n`);

      const options = {
        title: "Index Test",
        description: "Testing index parameter",
        items: [{ id: 1 }, { id: 2 }, { id: 3 }],
        renderItem: renderItemSpy,
      };

      await generateStagedPreview(options);

      expect(renderItemSpy).toHaveBeenCalledTimes(3);
      expect(renderItemSpy).toHaveBeenNthCalledWith(1, { id: 1 }, 0);
      expect(renderItemSpy).toHaveBeenNthCalledWith(2, { id: 2 }, 1);
      expect(renderItemSpy).toHaveBeenNthCalledWith(3, { id: 3 }, 2);
    });
  });
});
