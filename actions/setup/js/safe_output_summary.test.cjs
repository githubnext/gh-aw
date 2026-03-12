import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import * as fs from "fs";
import * as path from "path";
import * as os from "os";

// Mock the global objects that GitHub Actions provides
const mockCore = {
  info: vi.fn(),
  debug: vi.fn(),
  warning: vi.fn(),
  startGroup: vi.fn(),
  endGroup: vi.fn(),
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(undefined),
  },
};

// Set up global mocks before importing the module
globalThis.core = mockCore;

const { generateSafeOutputSummary, writeSafeOutputSummaries } = await import("./safe_output_summary.cjs");

describe("safe_output_summary", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("generateSafeOutputSummary", () => {
    it("should generate summary for successful create_issue", () => {
      const options = {
        type: "create_issue",
        messageIndex: 1,
        success: true,
        result: {
          repo: "owner/repo",
          number: 123,
          url: "https://github.com/owner/repo/issues/123",
          temporaryId: "issue-1",
        },
        message: {
          title: "Test Issue",
          body: "This is a test issue body",
          labels: ["bug", "enhancement"],
        },
      };

      const summary = generateSafeOutputSummary(options);

      expect(summary).toContain("<details>");
      expect(summary).toContain("</details>");
      expect(summary).toContain("✅");
      expect(summary).toContain("Create Issue");
      expect(summary).toContain("Message 1");
      expect(summary).toContain("owner/repo#123");
      expect(summary).toContain("https://github.com/owner/repo/issues/123");
      expect(summary).toContain("issue-1");
      expect(summary).toContain("Test Issue");
      expect(summary).toContain("bug, enhancement");
    });

    it("should generate summary for failed message with error", () => {
      const options = {
        type: "create_project",
        messageIndex: 2,
        success: false,
        result: null,
        message: {
          title: "Test Project",
        },
        error: "Failed to create project: permission denied",
      };

      const summary = generateSafeOutputSummary(options);

      expect(summary).toContain("❌");
      expect(summary).toContain("Failed");
      expect(summary).toContain("Create Project");
      expect(summary).toContain("Message 2");
      expect(summary).toContain("permission denied");
    });

    it("should truncate long body content", () => {
      const longBody = "a".repeat(1000);

      const options = {
        type: "create_discussion",
        messageIndex: 3,
        success: true,
        result: {
          repo: "owner/repo",
          number: 456,
        },
        message: {
          title: "Test Discussion",
          body: longBody,
        },
      };

      const summary = generateSafeOutputSummary(options);

      expect(summary).toContain("Body Preview");
      expect(summary).toContain("...");
      expect(summary.length).toBeLessThan(longBody.length + 1000);
    });

    it("should use 6-backtick fences for body content containing backticks", () => {
      const bodyWithBackticks = "Here is some code:\n```javascript\nconsole.log('hello');\n```\nEnd of body.";

      const options = {
        type: "create_issue",
        messageIndex: 1,
        success: true,
        result: {
          repo: "owner/repo",
          number: 123,
        },
        message: {
          title: "Issue with code",
          body: bodyWithBackticks,
        },
      };

      const summary = generateSafeOutputSummary(options);

      // Should use 6-backtick fences to avoid breaking when body contains triple backticks
      expect(summary).toContain("``````\n");
      expect(summary).toContain("Body Preview");
      expect(summary).toContain("```javascript");
    });

    it("should use 6-backtick fences for error message details containing backticks", () => {
      const messageWithBackticks = {
        title: "Test Issue",
        body: "Code: ```\nconsole.log('test');\n```",
      };

      const options = {
        type: "create_issue",
        messageIndex: 1,
        success: false,
        result: null,
        message: messageWithBackticks,
        error: "Failed to create issue",
      };

      const summary = generateSafeOutputSummary(options);

      // Should use 6-backtick fences for message details JSON to avoid rendering issues
      expect(summary).toContain("``````json\n");
      expect(summary).toContain("Message Details");
    });

    it("should handle project-specific results", () => {
      const options = {
        type: "create_project",
        messageIndex: 4,
        success: true,
        result: {
          projectUrl: "https://github.com/orgs/owner/projects/123",
        },
        message: {
          title: "Test Project",
        },
      };

      const summary = generateSafeOutputSummary(options);

      expect(summary).toContain("Project URL");
      expect(summary).toContain("https://github.com/orgs/owner/projects/123");
    });

    it("should display secrecy field when present in message", () => {
      const options = {
        type: "create_issue",
        messageIndex: 1,
        success: true,
        result: {
          repo: "owner/repo",
          number: 123,
        },
        message: {
          title: "Secure Issue",
          body: "Sensitive content",
          secrecy: "private",
        },
      };

      const summary = generateSafeOutputSummary(options);

      expect(summary).toContain("Secrecy:");
      expect(summary).toContain("private");
    });

    it("should display integrity field when present in message", () => {
      const options = {
        type: "create_issue",
        messageIndex: 1,
        success: true,
        result: {
          repo: "owner/repo",
          number: 123,
        },
        message: {
          title: "Trusted Issue",
          body: "Verified content",
          integrity: "high",
        },
      };

      const summary = generateSafeOutputSummary(options);

      expect(summary).toContain("Integrity:");
      expect(summary).toContain("high");
    });

    it("should display both secrecy and integrity fields when present", () => {
      const options = {
        type: "add_comment",
        messageIndex: 2,
        success: true,
        result: {
          repo: "owner/repo",
          number: 456,
        },
        message: {
          body: "A comment",
          secrecy: "internal",
          integrity: "medium",
        },
      };

      const summary = generateSafeOutputSummary(options);

      expect(summary).toContain("Secrecy:");
      expect(summary).toContain("internal");
      expect(summary).toContain("Integrity:");
      expect(summary).toContain("medium");
    });

    it("should not display secrecy or integrity when absent from message", () => {
      const options = {
        type: "create_issue",
        messageIndex: 1,
        success: true,
        result: {
          repo: "owner/repo",
          number: 123,
        },
        message: {
          title: "Normal Issue",
          body: "Normal content",
        },
      };

      const summary = generateSafeOutputSummary(options);

      expect(summary).not.toContain("Secrecy:");
      expect(summary).not.toContain("Integrity:");
    });

    it("should display secrecy and integrity fields even when operation fails", () => {
      const options = {
        type: "create_issue",
        messageIndex: 1,
        success: false,
        result: null,
        message: {
          title: "Failed Issue",
          secrecy: "public",
          integrity: "low",
        },
        error: "Permission denied",
      };

      const summary = generateSafeOutputSummary(options);

      expect(summary).toContain("Secrecy:");
      expect(summary).toContain("public");
      expect(summary).toContain("Integrity:");
      expect(summary).toContain("low");
    });

    it("should show fallback issue status when create_pull_request falls back to issue", () => {
      const options = {
        type: "create_pull_request",
        messageIndex: 1,
        success: true,
        result: {
          fallback_used: true,
          issue_number: 42,
          issue_url: "https://github.com/owner/repo/issues/42",
          branch_name: "fix-branch",
          repo: "owner/repo",
        },
        message: {
          title: "Fix: upgrade dependencies",
          body: "This upgrades the dependencies.",
        },
      };

      const summary = generateSafeOutputSummary(options);

      // Should use warning emoji and "Fallback Issue Created" status
      expect(summary).toContain("⚠️");
      expect(summary).toContain("Fallback Issue Created");
      expect(summary).toContain("Message 1");
      // Should NOT show "Success"
      expect(summary).not.toContain("✅");
      // Should show fallback explanation
      expect(summary).toContain("protected file");
      expect(summary).toContain("https://github.com/owner/repo/issues/42");
      expect(summary).toContain("owner/repo#42");
      expect(summary).toContain("fix-branch");
      expect(summary).toContain("Fix: upgrade dependencies");
    });

    it("should show fallback issue status when push_to_pull_request_branch falls back to issue", () => {
      const options = {
        type: "push_to_pull_request_branch",
        messageIndex: 2,
        success: true,
        result: {
          fallback_used: true,
          issue_number: 99,
          issue_url: "https://github.com/owner/repo/issues/99",
        },
        message: {
          body: "Pushing to PR branch.",
        },
      };

      const summary = generateSafeOutputSummary(options);

      expect(summary).toContain("⚠️");
      expect(summary).toContain("Fallback Issue Created");
      expect(summary).toContain("https://github.com/owner/repo/issues/99");
      expect(summary).not.toContain("✅");
    });

    it("should show normal success for create_pull_request when no fallback", () => {
      const options = {
        type: "create_pull_request",
        messageIndex: 1,
        success: true,
        result: {
          pull_request_number: 5,
          pull_request_url: "https://github.com/owner/repo/pull/5",
          url: "https://github.com/owner/repo/pull/5",
          repo: "owner/repo",
          number: 5,
        },
        message: { title: "My PR" },
      };

      const summary = generateSafeOutputSummary(options);

      expect(summary).toContain("✅");
      expect(summary).toContain("Success");
      expect(summary).not.toContain("⚠️");
      expect(summary).not.toContain("Fallback");
    });
  });

  describe("writeSafeOutputSummaries", () => {
    it("should write summaries for multiple results", async () => {
      const results = [
        {
          type: "create_issue",
          messageIndex: 0,
          success: true,
          result: {
            repo: "owner/repo",
            number: 123,
            url: "https://github.com/owner/repo/issues/123",
          },
        },
        {
          type: "create_project",
          messageIndex: 1,
          success: true,
          result: {
            projectUrl: "https://github.com/orgs/owner/projects/456",
          },
        },
      ];

      const messages = [{ title: "Issue 1", body: "Body 1" }, { title: "Project 1" }];

      await writeSafeOutputSummaries(results, messages);

      expect(mockCore.summary.addRaw).toHaveBeenCalledTimes(1);
      expect(mockCore.summary.write).toHaveBeenCalledTimes(1);
      expect(mockCore.info).toHaveBeenCalledWith("📝 Safe output summaries written to step summary");

      const summaryContent = mockCore.summary.addRaw.mock.calls[0][0];
      expect(summaryContent).toContain("Safe Output Processing Summary");
      expect(summaryContent).toContain("Processed 2 safe-output message(s)");
      expect(summaryContent).toContain("Create Issue");
      expect(summaryContent).toContain("Create Project");
    });

    it("should skip results handled by standalone steps", async () => {
      const results = [
        {
          type: "create_issue",
          messageIndex: 0,
          success: true,
          result: { repo: "owner/repo", number: 123 },
        },
        {
          type: "noop",
          messageIndex: 1,
          success: false,
          skipped: true,
          reason: "Handled by standalone step",
        },
      ];

      const messages = [{ title: "Issue 1" }, { message: "Noop message" }];

      await writeSafeOutputSummaries(results, messages);

      const summaryContent = mockCore.summary.addRaw.mock.calls[0][0];
      expect(summaryContent).toContain("Create Issue");
      expect(summaryContent).not.toContain("Noop");
    });

    it("should handle empty results", async () => {
      await writeSafeOutputSummaries([], []);

      expect(mockCore.summary.addRaw).not.toHaveBeenCalled();
      expect(mockCore.summary.write).not.toHaveBeenCalled();
    });

    it("should handle write failures gracefully", async () => {
      mockCore.summary.write.mockRejectedValueOnce(new Error("Write failed"));

      const results = [
        {
          type: "create_issue",
          messageIndex: 0,
          success: true,
          result: { repo: "owner/repo", number: 123 },
        },
      ];

      const messages = [{ title: "Issue 1" }];

      await writeSafeOutputSummaries(results, messages);

      expect(mockCore.warning).toHaveBeenCalledWith("Failed to write safe output summaries: Write failed");
    });

    it("should log raw .jsonl content when safe outputs file exists", async () => {
      // Create a temporary .jsonl file
      const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "test-safe-outputs-"));
      const jsonlFile = path.join(tempDir, "outputs.jsonl");
      const jsonlContent = '{"type":"create_issue","title":"Test Issue"}\n{"type":"add_comment","body":"Test comment"}';
      fs.writeFileSync(jsonlFile, jsonlContent, "utf8");

      // Set environment variable
      const originalEnv = process.env.GH_AW_SAFE_OUTPUTS;
      process.env.GH_AW_SAFE_OUTPUTS = jsonlFile;

      try {
        const results = [
          {
            type: "create_issue",
            messageIndex: 0,
            success: true,
            result: { repo: "owner/repo", number: 123 },
          },
        ];

        const messages = [{ title: "Issue 1" }];

        await writeSafeOutputSummaries(results, messages);

        // Verify that displayFileContent was called (which uses core.startGroup and core.endGroup)
        expect(mockCore.startGroup).toHaveBeenCalled();
        expect(mockCore.endGroup).toHaveBeenCalled();

        // Verify that the group title includes the file name and size
        const startGroupCalls = mockCore.startGroup.mock.calls.map(call => call[0]);
        expect(startGroupCalls.some(call => call.includes("safe-outputs.jsonl"))).toBe(true);

        // Verify that content lines were logged
        const infoCalls = mockCore.info.mock.calls.map(call => call[0]);
        expect(infoCalls.length).toBeGreaterThan(0);
      } finally {
        // Cleanup
        process.env.GH_AW_SAFE_OUTPUTS = originalEnv;
        fs.rmSync(tempDir, { recursive: true, force: true });
      }
    });

    it("should handle missing safe outputs file gracefully", async () => {
      // Set environment variable to a non-existent file
      const originalEnv = process.env.GH_AW_SAFE_OUTPUTS;
      process.env.GH_AW_SAFE_OUTPUTS = "/non/existent/file.jsonl";

      try {
        const results = [
          {
            type: "create_issue",
            messageIndex: 0,
            success: true,
            result: { repo: "owner/repo", number: 123 },
          },
        ];

        const messages = [{ title: "Issue 1" }];

        await writeSafeOutputSummaries(results, messages);

        // Should not throw and should still write summary
        expect(mockCore.summary.write).toHaveBeenCalledTimes(1);
        expect(mockCore.info).toHaveBeenCalledWith("📝 Safe output summaries written to step summary");
      } finally {
        // Cleanup
        process.env.GH_AW_SAFE_OUTPUTS = originalEnv;
      }
    });

    it("should skip logging when safe outputs file is empty", async () => {
      // Create a temporary empty .jsonl file
      const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "test-safe-outputs-"));
      const jsonlFile = path.join(tempDir, "outputs.jsonl");
      fs.writeFileSync(jsonlFile, "", "utf8");

      // Set environment variable
      const originalEnv = process.env.GH_AW_SAFE_OUTPUTS;
      process.env.GH_AW_SAFE_OUTPUTS = jsonlFile;

      try {
        const results = [
          {
            type: "create_issue",
            messageIndex: 0,
            success: true,
            result: { repo: "owner/repo", number: 123 },
          },
        ];

        const messages = [{ title: "Issue 1" }];

        await writeSafeOutputSummaries(results, messages);

        // Should not log empty content or start a log group
        expect(mockCore.startGroup).not.toHaveBeenCalled();
        expect(mockCore.endGroup).not.toHaveBeenCalled();
      } finally {
        // Cleanup
        process.env.GH_AW_SAFE_OUTPUTS = originalEnv;
        fs.rmSync(tempDir, { recursive: true, force: true });
      }
    });

    it("should truncate large safe outputs file content", async () => {
      // Create a temporary .jsonl file with large content (> 5000 bytes)
      const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "test-safe-outputs-"));
      const jsonlFile = path.join(tempDir, "outputs.jsonl");

      // Create content larger than 5000 bytes
      const largeEntry = { type: "create_issue", title: "Test", body: "a".repeat(5000) };
      const jsonlContent = JSON.stringify(largeEntry) + "\n" + JSON.stringify(largeEntry);
      fs.writeFileSync(jsonlFile, jsonlContent, "utf8");

      // Set environment variable
      const originalEnv = process.env.GH_AW_SAFE_OUTPUTS;
      process.env.GH_AW_SAFE_OUTPUTS = jsonlFile;

      try {
        const results = [
          {
            type: "create_issue",
            messageIndex: 0,
            success: true,
            result: { repo: "owner/repo", number: 123 },
          },
        ];

        const messages = [{ title: "Issue 1" }];

        await writeSafeOutputSummaries(results, messages);

        // Verify that displayFileContent was called (which uses core.startGroup and core.endGroup)
        expect(mockCore.startGroup).toHaveBeenCalled();
        expect(mockCore.endGroup).toHaveBeenCalled();

        // Verify that the group title includes the file name
        const startGroupCalls = mockCore.startGroup.mock.calls.map(call => call[0]);
        expect(startGroupCalls.some(call => call.includes("safe-outputs.jsonl"))).toBe(true);

        // Verify that truncation message was logged (displayFileContent shows truncation info)
        const infoCalls = mockCore.info.mock.calls.map(call => call[0]);
        expect(infoCalls.some(call => call.includes("truncated") || call.includes("..."))).toBe(true);
      } finally {
        // Cleanup
        process.env.GH_AW_SAFE_OUTPUTS = originalEnv;
        fs.rmSync(tempDir, { recursive: true, force: true });
      }
    });
  });
});
