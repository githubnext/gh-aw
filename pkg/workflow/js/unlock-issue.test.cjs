import { describe, it, expect, beforeEach, vi } from "vitest";
import fs from "fs";
import path from "path";
// Mock the global objects that GitHub Actions provides
const mockCore = { debug: vi.fn(), info: vi.fn(), warning: vi.fn(), error: vi.fn(), setFailed: vi.fn(), setOutput: vi.fn() },
  mockGithub = { rest: { issues: { get: vi.fn(), unlock: vi.fn() } } },
  mockContext = { eventName: "issues", runId: 12345, repo: { owner: "testowner", repo: "testrepo" }, issue: { number: 42 }, payload: { issue: { number: 42 }, repository: { html_url: "https://github.com/testowner/testrepo" } } };
// Set up global mocks before importing the module
((global.core = mockCore),
  (global.github = mockGithub),
  (global.context = mockContext),
  describe("unlock-issue", () => {
    let unlockIssueScript;
    (beforeEach(() => {
      (vi.clearAllMocks(),
        // Reset context to default state
        (global.context.eventName = "issues"),
        (global.context.issue = { number: 42 }),
        (global.context.payload.issue = { number: 42 }));
      // Read the script content
      const scriptPath = path.join(process.cwd(), "unlock-issue.cjs");
      unlockIssueScript = fs.readFileSync(scriptPath, "utf8");
    }),
      it("should unlock issue successfully", async () => {
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        (mockGithub.rest.issues.get.mockResolvedValue({ data: { number: 42, locked: !0 } }),
          // Mock successful unlock
          mockGithub.rest.issues.unlock.mockResolvedValue({ status: 204 }),
          // Execute the script
          await eval(`(async () => { ${unlockIssueScript} })()`),
          expect(mockGithub.rest.issues.get).toHaveBeenCalledWith({ owner: "testowner", repo: "testrepo", issue_number: 42 }),
          expect(mockGithub.rest.issues.unlock).toHaveBeenCalledWith({ owner: "testowner", repo: "testrepo", issue_number: 42 }),
          expect(mockCore.info).toHaveBeenCalledWith("Checking if issue #42 is locked"),
          expect(mockCore.info).toHaveBeenCalledWith("Unlocking issue #42 after agent workflow execution"),
          expect(mockCore.info).toHaveBeenCalledWith("✅ Successfully unlocked issue #42"),
          expect(mockCore.setFailed).not.toHaveBeenCalled());
      }),
      it("should skip unlocking if issue is not locked", async () => {
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        (mockGithub.rest.issues.get.mockResolvedValue({ data: { number: 42, locked: !1 } }),
          // Execute the script
          await eval(`(async () => { ${unlockIssueScript} })()`),
          expect(mockGithub.rest.issues.get).toHaveBeenCalledWith({ owner: "testowner", repo: "testrepo", issue_number: 42 }),
          // Should not call unlock since issue is not locked
          expect(mockGithub.rest.issues.unlock).not.toHaveBeenCalled(),
          expect(mockCore.info).toHaveBeenCalledWith("Checking if issue #42 is locked"),
          expect(mockCore.info).toHaveBeenCalledWith("ℹ️ Issue #42 is not locked, skipping unlock operation"),
          expect(mockCore.setFailed).not.toHaveBeenCalled());
      }),
      it("should fail when issue number is not found in context", async () => {
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        // Remove issue number from context
        ((global.context.issue = {}),
          delete global.context.payload.issue,
          // Execute the script
          await eval(`(async () => { ${unlockIssueScript} })()`),
          expect(mockGithub.rest.issues.unlock).not.toHaveBeenCalled(),
          expect(mockCore.setFailed).toHaveBeenCalledWith("Issue number not found in context"));
      }),
      it("should handle API errors gracefully", async () => {
        // Mock issue get to return locked issue
        mockGithub.rest.issues.get.mockResolvedValue({ data: { number: 42, locked: !0 } });
        // Mock API error
        const apiError = new Error("Issue was not locked");
        (mockGithub.rest.issues.unlock.mockRejectedValue(apiError),
          // Execute the script
          await eval(`(async () => { ${unlockIssueScript} })()`),
          expect(mockGithub.rest.issues.unlock).toHaveBeenCalled(),
          expect(mockCore.error).toHaveBeenCalledWith("Failed to unlock issue: Issue was not locked"),
          expect(mockCore.setFailed).toHaveBeenCalledWith("Failed to unlock issue #42: Issue was not locked"));
      }),
      it("should handle non-Error exceptions", async () => {
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        // Mock issue get to return locked issue
        (mockGithub.rest.issues.get.mockResolvedValue({ data: { number: 42, locked: !0 } }),
          // Mock non-Error exception
          mockGithub.rest.issues.unlock.mockRejectedValue("String error"),
          // Execute the script
          await eval(`(async () => { ${unlockIssueScript} })()`),
          expect(mockCore.error).toHaveBeenCalledWith("Failed to unlock issue: String error"),
          expect(mockCore.setFailed).toHaveBeenCalledWith("Failed to unlock issue #42: String error"));
      }),
      it("should work with different issue numbers", async () => {
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        // Change issue number
        ((global.context.issue = { number: 200 }),
          (global.context.payload.issue = { number: 200 }),
          // Mock issue get to return locked issue
          mockGithub.rest.issues.get.mockResolvedValue({ data: { number: 200, locked: !0 } }),
          mockGithub.rest.issues.unlock.mockResolvedValue({ status: 204 }),
          // Execute the script
          await eval(`(async () => { ${unlockIssueScript} })()`),
          expect(mockGithub.rest.issues.unlock).toHaveBeenCalledWith({ owner: "testowner", repo: "testrepo", issue_number: 200 }),
          expect(mockCore.info).toHaveBeenCalledWith("Checking if issue #200 is locked"),
          expect(mockCore.info).toHaveBeenCalledWith("Unlocking issue #200 after agent workflow execution"),
          expect(mockCore.info).toHaveBeenCalledWith("✅ Successfully unlocked issue #200"));
      }),
      it("should handle permission errors", async () => {
        // Mock issue get to return locked issue
        mockGithub.rest.issues.get.mockResolvedValue({ data: { number: 42, locked: !0 } });
        // Mock permission error
        const permissionError = new Error("Resource not accessible by integration");
        (mockGithub.rest.issues.unlock.mockRejectedValue(permissionError),
          // Execute the script
          await eval(`(async () => { ${unlockIssueScript} })()`),
          expect(mockGithub.rest.issues.unlock).toHaveBeenCalled(),
          expect(mockCore.error).toHaveBeenCalledWith("Failed to unlock issue: Resource not accessible by integration"),
          expect(mockCore.setFailed).toHaveBeenCalledWith("Failed to unlock issue #42: Resource not accessible by integration"));
      }),
      it("should skip if issue is already unlocked (redundant test for completeness)", async () => {
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        // Mock issue get to return unlocked issue
        (mockGithub.rest.issues.get.mockResolvedValue({ data: { number: 42, locked: !1 } }),
          // Execute the script
          await eval(`(async () => { ${unlockIssueScript} })()`),
          // Should skip unlock since issue is not locked
          expect(mockGithub.rest.issues.unlock).not.toHaveBeenCalled(),
          expect(mockCore.info).toHaveBeenCalledWith("ℹ️ Issue #42 is not locked, skipping unlock operation"),
          expect(mockCore.setFailed).not.toHaveBeenCalled());
      }));
  }));
