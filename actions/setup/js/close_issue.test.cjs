import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import fs from "fs";
import path from "path";
const mockCore = { debug: vi.fn(), info: vi.fn(), warning: vi.fn(), error: vi.fn(), setFailed: vi.fn(), setOutput: vi.fn(), summary: { addRaw: vi.fn().mockReturnThis(), write: vi.fn().mockResolvedValue() } },
  mockGithub = { rest: { issues: { get: vi.fn(), createComment: vi.fn(), update: vi.fn() } } },
  mockContext = { eventName: "issues", runId: 12345, repo: { owner: "testowner", repo: "testrepo" }, payload: { issue: { number: 42 }, repository: { html_url: "https://github.com/testowner/testrepo" } } };
((global.core = mockCore),
  (global.github = mockGithub),
  (global.context = mockContext),
  describe("close_issue", () => {
    let closeIssueScript, tempFilePath;
    const setAgentOutput = data => {
      tempFilePath = path.join("/tmp", `test_agent_output_${Date.now()}_${Math.random().toString(36).slice(2)}.json`);
      const content = "string" == typeof data ? data : JSON.stringify(data);
      (fs.writeFileSync(tempFilePath, content), (process.env.GH_AW_AGENT_OUTPUT = tempFilePath));
    };
    (beforeEach(() => {
      (vi.clearAllMocks(),
        delete process.env.GH_AW_SAFE_OUTPUTS_STAGED,
        delete process.env.GH_AW_AGENT_OUTPUT,
        delete process.env.GH_AW_CLOSE_ISSUE_REQUIRED_LABELS,
        delete process.env.GH_AW_CLOSE_ISSUE_REQUIRED_TITLE_PREFIX,
        delete process.env.GH_AW_CLOSE_ISSUE_TARGET,
        delete process.env.GH_AW_WORKFLOW_NAME,
        delete process.env.GITHUB_SERVER_URL,
        (global.context.eventName = "issues"),
        (global.context.payload.issue = { number: 42 }));
      const scriptPath = path.join(process.cwd(), "close_issue.cjs");
      closeIssueScript = fs.readFileSync(scriptPath, "utf8");
    }),
      afterEach(() => {
        tempFilePath && fs.existsSync(tempFilePath) && (fs.unlinkSync(tempFilePath), (tempFilePath = void 0));
      }),
      it("should handle empty agent output", async () => {
        (setAgentOutput({ items: [], errors: [] }), await eval(`(async () => { ${closeIssueScript}; await main(); })()`), expect(mockCore.info).toHaveBeenCalledWith("No close-issue items found in agent output"));
      }),
      it("should handle missing agent output", async () => {
        (await eval(`(async () => { ${closeIssueScript}; await main(); })()`), expect(mockCore.info).toHaveBeenCalledWith("No GH_AW_AGENT_OUTPUT environment variable found"));
      }),
      it("should close issue with comment in triggering context", async () => {
        (setAgentOutput({ items: [{ type: "close_issue", body: "Closing this issue due to completion." }] }),
          mockGithub.rest.issues.get.mockResolvedValue({ data: { number: 42, title: "[test] Test Issue", labels: [{ name: "bug" }], state: "open", html_url: "https://github.com/testowner/testrepo/issues/42" } }),
          mockGithub.rest.issues.createComment.mockResolvedValue({ data: { id: 123, html_url: "https://github.com/testowner/testrepo/issues/42#issuecomment-123" } }),
          mockGithub.rest.issues.update.mockResolvedValue({ data: { number: 42, html_url: "https://github.com/testowner/testrepo/issues/42", title: "[test] Test Issue" } }),
          await eval(`(async () => { ${closeIssueScript}; await main(); })()`),
          expect(mockGithub.rest.issues.get).toHaveBeenCalledWith({ owner: "testowner", repo: "testrepo", issue_number: 42 }),
          expect(mockGithub.rest.issues.createComment).toHaveBeenCalled(),
          expect(mockGithub.rest.issues.update).toHaveBeenCalledWith({ owner: "testowner", repo: "testrepo", issue_number: 42, state: "closed" }),
          expect(mockCore.setOutput).toHaveBeenCalledWith("issue_number", 42),
          expect(mockCore.setOutput).toHaveBeenCalledWith("issue_url", "https://github.com/testowner/testrepo/issues/42"));
      }),
      it("should close specific issue when target is *", async () => {
        (setAgentOutput({ items: [{ type: "close_issue", issue_number: 100, body: "Closing this issue." }] }),
          (process.env.GH_AW_CLOSE_ISSUE_TARGET = "*"),
          mockGithub.rest.issues.get.mockResolvedValue({ data: { number: 100, title: "[refactor] Refactor Test", labels: [{ name: "refactoring" }], state: "open", html_url: "https://github.com/testowner/testrepo/issues/100" } }),
          mockGithub.rest.issues.createComment.mockResolvedValue({ data: { id: 456, html_url: "https://github.com/testowner/testrepo/issues/100#issuecomment-456" } }),
          mockGithub.rest.issues.update.mockResolvedValue({ data: { number: 100, html_url: "https://github.com/testowner/testrepo/issues/100", title: "[refactor] Refactor Test" } }),
          await eval(`(async () => { ${closeIssueScript}; await main(); })()`),
          expect(mockGithub.rest.issues.get).toHaveBeenCalledWith({ owner: "testowner", repo: "testrepo", issue_number: 100 }),
          expect(mockCore.setOutput).toHaveBeenCalledWith("issue_number", 100));
      }),
      it("should filter by required title prefix", async () => {
        (setAgentOutput({ items: [{ type: "close_issue", issue_number: 50, body: "Closing this issue." }] }),
          (process.env.GH_AW_CLOSE_ISSUE_TARGET = "*"),
          (process.env.GH_AW_CLOSE_ISSUE_REQUIRED_TITLE_PREFIX = "[refactor] "),
          mockGithub.rest.issues.get.mockResolvedValue({ data: { number: 50, title: "[bug] Bug Fix", labels: [], state: "open", html_url: "https://github.com/testowner/testrepo/issues/50" } }),
          await eval(`(async () => { ${closeIssueScript}; await main(); })()`),
          expect(mockGithub.rest.issues.get).toHaveBeenCalled(),
          expect(mockGithub.rest.issues.update).not.toHaveBeenCalled(),
          expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("does not have required title prefix")));
      }),
      it("should filter by required labels", async () => {
        (setAgentOutput({ items: [{ type: "close_issue", issue_number: 60, body: "Closing this issue." }] }),
          (process.env.GH_AW_CLOSE_ISSUE_TARGET = "*"),
          (process.env.GH_AW_CLOSE_ISSUE_REQUIRED_LABELS = "automated,stale"),
          mockGithub.rest.issues.get.mockResolvedValue({ data: { number: 60, title: "Test Issue", labels: [{ name: "bug" }], state: "open", html_url: "https://github.com/testowner/testrepo/issues/60" } }),
          await eval(`(async () => { ${closeIssueScript}; await main(); })()`),
          expect(mockGithub.rest.issues.get).toHaveBeenCalled(),
          expect(mockGithub.rest.issues.update).not.toHaveBeenCalled(),
          expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("does not have required labels")));
      }),
      it("should skip already closed issues", async () => {
        (setAgentOutput({ items: [{ type: "close_issue", issue_number: 70, body: "Closing this issue." }] }),
          (process.env.GH_AW_CLOSE_ISSUE_TARGET = "*"),
          mockGithub.rest.issues.get.mockResolvedValue({ data: { number: 70, title: "Already Closed", labels: [], state: "closed", html_url: "https://github.com/testowner/testrepo/issues/70" } }),
          await eval(`(async () => { ${closeIssueScript}; await main(); })()`),
          expect(mockGithub.rest.issues.get).toHaveBeenCalled(),
          expect(mockGithub.rest.issues.update).not.toHaveBeenCalled(),
          expect(mockCore.info).toHaveBeenCalledWith("Issue #70 is already closed, skipping"));
      }),
      it("should work in staged mode", async () => {
        (setAgentOutput({ items: [{ type: "close_issue", issue_number: 80, body: "This is a test close." }] }),
          (process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true"),
          await eval(`(async () => { ${closeIssueScript}; await main(); })()`),
          expect(mockGithub.rest.issues.get).not.toHaveBeenCalled(),
          expect(mockGithub.rest.issues.update).not.toHaveBeenCalled(),
          expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("🎭 Staged Mode: Close Issues Preview")),
          expect(mockCore.info).toHaveBeenCalledWith("📝 Issue close preview written to step summary"));
      }),
      it("should handle multiple issues in batch", async () => {
        (setAgentOutput({
          items: [
            { type: "close_issue", issue_number: 91, body: "Closing issue 91." },
            { type: "close_issue", issue_number: 92, body: "Closing issue 92." },
          ],
        }),
          (process.env.GH_AW_CLOSE_ISSUE_TARGET = "*"),
          mockGithub.rest.issues.get
            .mockResolvedValueOnce({ data: { number: 91, title: "Issue 91", labels: [], state: "open", html_url: "https://github.com/testowner/testrepo/issues/91" } })
            .mockResolvedValueOnce({ data: { number: 92, title: "Issue 92", labels: [], state: "open", html_url: "https://github.com/testowner/testrepo/issues/92" } }),
          mockGithub.rest.issues.createComment.mockResolvedValue({ data: { id: 999, html_url: "https://github.com/testowner/testrepo/issues/91#issuecomment-999" } }),
          mockGithub.rest.issues.update.mockResolvedValue({ data: { number: 91, html_url: "https://github.com/testowner/testrepo/issues/91", title: "Issue 91" } }),
          await eval(`(async () => { ${closeIssueScript}; await main(); })()`),
          expect(mockGithub.rest.issues.get).toHaveBeenCalledTimes(2),
          expect(mockGithub.rest.issues.update).toHaveBeenCalledTimes(2),
          expect(mockCore.info).toHaveBeenCalledWith("Successfully closed 2 issue(s)"));
      }),
      it("should skip when not in issue context and target is triggering", async () => {
        (setAgentOutput({ items: [{ type: "close_issue", body: "Closing issue." }] }),
          (global.context.eventName = "push"),
          delete global.context.payload.issue,
          await eval(`(async () => { ${closeIssueScript}; await main(); })()`),
          expect(mockGithub.rest.issues.update).not.toHaveBeenCalled(),
          expect(mockCore.info).toHaveBeenCalledWith('Target is "triggering" but not running in issue context, skipping issue close'));
      }));
  }));
