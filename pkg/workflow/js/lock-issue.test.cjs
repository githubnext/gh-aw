import { describe, it, expect, beforeEach, vi } from "vitest";
import fs from "fs";
import path from "path";
// Mock the global objects that GitHub Actions provides
const mockCore = { debug: vi.fn(), info: vi.fn(), warning: vi.fn(), error: vi.fn(), setFailed: vi.fn(), setOutput: vi.fn() },
  mockGithub = { rest: { issues: { get: vi.fn(), lock: vi.fn() } } },
  mockContext = { eventName: "issues", runId: 12345, repo: { owner: "testowner", repo: "testrepo" }, issue: { number: 42 }, payload: { issue: { number: 42 }, repository: { html_url: "https://github.com/testowner/testrepo" } } };
// Set up global mocks before importing the module
((global.core = mockCore),
  (global.github = mockGithub),
  (global.context = mockContext),
  describe("lock-issue", () => {
    let lockIssueScript;
    (beforeEach(() => {
      (vi.clearAllMocks(),
        // Reset context to default state
        (global.context.eventName = "issues"),
        (global.context.issue = { number: 42 }),
        (global.context.payload.issue = { number: 42 }));
      // Read the script content
      const scriptPath = path.join(process.cwd(), "lock-issue.cjs");
      lockIssueScript = fs.readFileSync(scriptPath, "utf8");
    }),
      it("should lock issue successfully", async () => {
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
          // Mock successful lock
          mockGithub.rest.issues.lock.mockResolvedValue({ status: 204 }),
          // Execute the script
          await eval(`(async () => { ${lockIssueScript} })()`),
          expect(mockGithub.rest.issues.get).toHaveBeenCalledWith({ owner: "testowner", repo: "testrepo", issue_number: 42 }),
          expect(mockGithub.rest.issues.lock).toHaveBeenCalledWith({ owner: "testowner", repo: "testrepo", issue_number: 42 }),
          expect(mockCore.info).toHaveBeenCalledWith("Checking if issue #42 is already locked"),
          expect(mockCore.info).toHaveBeenCalledWith("Locking issue #42 for agent workflow execution"),
          expect(mockCore.info).toHaveBeenCalledWith("✅ Successfully locked issue #42"),
          expect(mockCore.setOutput).toHaveBeenCalledWith("locked", "true"),
          expect(mockCore.setFailed).not.toHaveBeenCalled());
      }),
      it("should skip locking if issue is already locked", async () => {
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
          // Execute the script
          await eval(`(async () => { ${lockIssueScript} })()`),
          expect(mockGithub.rest.issues.get).toHaveBeenCalledWith({ owner: "testowner", repo: "testrepo", issue_number: 42 }),
          // Should not call lock since issue is already locked
          expect(mockGithub.rest.issues.lock).not.toHaveBeenCalled(),
          expect(mockCore.info).toHaveBeenCalledWith("Checking if issue #42 is already locked"),
          expect(mockCore.info).toHaveBeenCalledWith("ℹ️ Issue #42 is already locked, skipping lock operation"),
          expect(mockCore.setOutput).toHaveBeenCalledWith("locked", "false"),
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
          await eval(`(async () => { ${lockIssueScript} })()`),
          expect(mockGithub.rest.issues.lock).not.toHaveBeenCalled(),
          expect(mockCore.setFailed).toHaveBeenCalledWith("Issue number not found in context"));
      }),
      it("should handle API errors gracefully", async () => {
        // Mock issue get to return unlocked issue
        mockGithub.rest.issues.get.mockResolvedValue({ data: { number: 42, locked: !1 } });
        // Mock API error
        const apiError = new Error("API rate limit exceeded");
        (mockGithub.rest.issues.lock.mockRejectedValue(apiError),
          // Execute the script
          await eval(`(async () => { ${lockIssueScript} })()`),
          expect(mockGithub.rest.issues.lock).toHaveBeenCalled(),
          expect(mockCore.error).toHaveBeenCalledWith("Failed to lock issue: API rate limit exceeded"),
          expect(mockCore.setFailed).toHaveBeenCalledWith("Failed to lock issue #42: API rate limit exceeded"),
          expect(mockCore.setOutput).toHaveBeenCalledWith("locked", "false"));
      }),
      it("should handle non-Error exceptions", async () => {
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
          // Mock non-Error exception
          mockGithub.rest.issues.lock.mockRejectedValue("String error"),
          // Execute the script
          await eval(`(async () => { ${lockIssueScript} })()`),
          expect(mockCore.error).toHaveBeenCalledWith("Failed to lock issue: String error"),
          expect(mockCore.setFailed).toHaveBeenCalledWith("Failed to lock issue #42: String error"),
          expect(mockCore.setOutput).toHaveBeenCalledWith("locked", "false"));
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
        ((global.context.issue = { number: 100 }),
          (global.context.payload.issue = { number: 100 }),
          // Mock issue get to return unlocked issue
          mockGithub.rest.issues.get.mockResolvedValue({ data: { number: 100, locked: !1 } }),
          mockGithub.rest.issues.lock.mockResolvedValue({ status: 204 }),
          // Execute the script
          await eval(`(async () => { ${lockIssueScript} })()`),
          expect(mockGithub.rest.issues.lock).toHaveBeenCalledWith({ owner: "testowner", repo: "testrepo", issue_number: 100 }),
          expect(mockCore.info).toHaveBeenCalledWith("Checking if issue #100 is already locked"),
          expect(mockCore.info).toHaveBeenCalledWith("Locking issue #100 for agent workflow execution"),
          expect(mockCore.info).toHaveBeenCalledWith("✅ Successfully locked issue #100"));
      }),
      it("should not provide a lock reason", async () => {
        (mockGithub.rest.issues.get.mockResolvedValue({ data: { number: 42, locked: !1 } }),
          mockGithub.rest.issues.lock.mockResolvedValue({ status: 204 }),
          // Execute the script
          await eval(`(async () => { ${lockIssueScript} })()`));
        const lockCall = mockGithub.rest.issues.lock.mock.calls[0][0];
        // Verify no lock_reason is provided
        (expect(lockCall).not.toHaveProperty("lock_reason"), expect(lockCall).toEqual({ owner: "testowner", repo: "testrepo", issue_number: 42 }));
      }));
  }));
