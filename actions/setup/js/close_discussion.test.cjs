import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import fs from "fs";
import path from "path";
const mockCore = { debug: vi.fn(), info: vi.fn(), warning: vi.fn(), error: vi.fn(), setFailed: vi.fn(), setOutput: vi.fn(), summary: { addRaw: vi.fn().mockReturnThis(), write: vi.fn().mockResolvedValue() } },
  mockGithub = { rest: {}, graphql: vi.fn() },
  mockContext = { eventName: "discussion", runId: 12345, repo: { owner: "testowner", repo: "testrepo" }, payload: { discussion: { number: 42 }, repository: { html_url: "https://github.com/testowner/testrepo" } } };
((global.core = mockCore),
  (global.github = mockGithub),
  (global.context = mockContext),
  describe("close_discussion", () => {
    let closeDiscussionScript, tempFilePath;
    const setAgentOutput = data => {
      tempFilePath = path.join("/tmp", `test_agent_output_${Date.now()}_${Math.random().toString(36).slice(2)}.json`);
      const content = "string" == typeof data ? data : JSON.stringify(data);
      (fs.writeFileSync(tempFilePath, content), (process.env.GH_AW_AGENT_OUTPUT = tempFilePath));
    };
    (beforeEach(() => {
      (vi.clearAllMocks(),
        delete process.env.GH_AW_SAFE_OUTPUTS_STAGED,
        delete process.env.GH_AW_AGENT_OUTPUT,
        delete process.env.GH_AW_CLOSE_DISCUSSION_REQUIRED_LABELS,
        delete process.env.GH_AW_CLOSE_DISCUSSION_REQUIRED_TITLE_PREFIX,
        delete process.env.GH_AW_CLOSE_DISCUSSION_REQUIRED_CATEGORY,
        delete process.env.GH_AW_CLOSE_DISCUSSION_TARGET,
        delete process.env.GH_AW_WORKFLOW_NAME,
        delete process.env.GITHUB_SERVER_URL,
        (global.context.eventName = "discussion"),
        (global.context.payload.discussion = { number: 42 }));
      const scriptPath = path.join(process.cwd(), "close_discussion.cjs");
      closeDiscussionScript = fs.readFileSync(scriptPath, "utf8");
    }),
      afterEach(() => {
        tempFilePath && fs.existsSync(tempFilePath) && (fs.unlinkSync(tempFilePath), (tempFilePath = void 0));
      }),
      it("should handle empty agent output", async () => {
        (setAgentOutput({ items: [], errors: [] }), await eval(`(async () => { ${closeDiscussionScript}; await main(); })()`), expect(mockCore.info).toHaveBeenCalledWith("No close-discussion items found in agent output"));
      }),
      it("should handle missing agent output", async () => {
        (await eval(`(async () => { ${closeDiscussionScript}; await main(); })()`), expect(mockCore.info).toHaveBeenCalledWith("No GH_AW_AGENT_OUTPUT environment variable found"));
      }),
      it("should close discussion with comment in non-staged mode", async () => {
        const validatedOutput = { items: [{ type: "close_discussion", body: "This discussion is resolved.", reason: "RESOLVED" }], errors: [] };
        (setAgentOutput(validatedOutput),
          (process.env.GH_AW_WORKFLOW_NAME = "Test Workflow"),
          (process.env.GITHUB_SERVER_URL = "https://github.com"),
          mockGithub.graphql
            .mockResolvedValueOnce({ repository: { discussion: { id: "D_kwDOABCDEF01", title: "Test Discussion", category: { name: "General" }, labels: { nodes: [] }, url: "https://github.com/testowner/testrepo/discussions/42" } } })
            .mockResolvedValueOnce({ addDiscussionComment: { comment: { id: "DC_kwDOABCDEF02", url: "https://github.com/testowner/testrepo/discussions/42#discussioncomment-123" } } })
            .mockResolvedValueOnce({ closeDiscussion: { discussion: { id: "D_kwDOABCDEF01", url: "https://github.com/testowner/testrepo/discussions/42" } } }),
          await eval(`(async () => { ${closeDiscussionScript}; await main(); })()`),
          expect(mockCore.info).toHaveBeenCalledWith("Found 1 close-discussion item(s)"),
          expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Processing close-discussion item 1/1")),
          expect(mockCore.info).toHaveBeenCalledWith("Adding comment to discussion #42"),
          expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Closing discussion #42 with reason: RESOLVED")),
          expect(mockCore.setOutput).toHaveBeenCalledWith("discussion_number", 42),
          expect(mockCore.setOutput).toHaveBeenCalledWith("discussion_url", expect.any(String)),
          expect(mockCore.setOutput).toHaveBeenCalledWith("comment_url", expect.any(String)));
      }),
      it("should show preview in staged mode", async () => {
        const validatedOutput = { items: [{ type: "close_discussion", body: "This discussion is resolved.", reason: "RESOLVED" }], errors: [] };
        (setAgentOutput(validatedOutput),
          (process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true"),
          await eval(`(async () => { ${closeDiscussionScript}; await main(); })()`),
          expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("🎭 Staged Mode: Close Discussions Preview")),
          expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("**Target:** Current discussion")),
          expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("**Reason:** RESOLVED")),
          expect(mockCore.summary.write).toHaveBeenCalled(),
          expect(mockCore.info).toHaveBeenCalledWith("📝 Discussion close preview written to step summary"));
      }),
      it("should filter by required labels", async () => {
        const validatedOutput = { items: [{ type: "close_discussion", body: "Closing this discussion." }], errors: [] };
        (setAgentOutput(validatedOutput),
          (process.env.GH_AW_CLOSE_DISCUSSION_REQUIRED_LABELS = "resolved,completed"),
          mockGithub.graphql.mockResolvedValueOnce({
            repository: { discussion: { id: "D_kwDOABCDEF01", title: "Test Discussion", category: { name: "General" }, labels: { nodes: [{ name: "question" }] }, url: "https://github.com/testowner/testrepo/discussions/42" } },
          }),
          await eval(`(async () => { ${closeDiscussionScript}; await main(); })()`),
          expect(mockCore.info).toHaveBeenCalledWith("Discussion #42 does not have required labels: resolved, completed"),
          expect(mockCore.setOutput).not.toHaveBeenCalled());
      }),
      it("should filter by title prefix", async () => {
        const validatedOutput = { items: [{ type: "close_discussion", body: "Closing this discussion." }], errors: [] };
        (setAgentOutput(validatedOutput),
          (process.env.GH_AW_CLOSE_DISCUSSION_REQUIRED_TITLE_PREFIX = "[task]"),
          mockGithub.graphql.mockResolvedValueOnce({
            repository: { discussion: { id: "D_kwDOABCDEF01", title: "Test Discussion", category: { name: "General" }, labels: { nodes: [] }, url: "https://github.com/testowner/testrepo/discussions/42" } },
          }),
          await eval(`(async () => { ${closeDiscussionScript}; await main(); })()`),
          expect(mockCore.info).toHaveBeenCalledWith("Discussion #42 does not have required title prefix: [task]"),
          expect(mockCore.setOutput).not.toHaveBeenCalled());
      }),
      it("should filter by category", async () => {
        const validatedOutput = { items: [{ type: "close_discussion", body: "Closing this discussion." }], errors: [] };
        (setAgentOutput(validatedOutput),
          (process.env.GH_AW_CLOSE_DISCUSSION_REQUIRED_CATEGORY = "Announcements"),
          mockGithub.graphql.mockResolvedValueOnce({
            repository: { discussion: { id: "D_kwDOABCDEF01", title: "Test Discussion", category: { name: "General" }, labels: { nodes: [] }, url: "https://github.com/testowner/testrepo/discussions/42" } },
          }),
          await eval(`(async () => { ${closeDiscussionScript}; await main(); })()`),
          expect(mockCore.info).toHaveBeenCalledWith("Discussion #42 is not in required category: Announcements"),
          expect(mockCore.setOutput).not.toHaveBeenCalled());
      }),
      it("should handle explicit discussion_number", async () => {
        const validatedOutput = { items: [{ type: "close_discussion", body: "Closing this discussion.", discussion_number: 99 }], errors: [] };
        (setAgentOutput(validatedOutput),
          (process.env.GH_AW_CLOSE_DISCUSSION_TARGET = "*"),
          (process.env.GH_AW_WORKFLOW_NAME = "Test Workflow"),
          mockGithub.graphql
            .mockResolvedValueOnce({ repository: { discussion: { id: "D_kwDOABCDEF01", title: "Test Discussion", category: { name: "General" }, labels: { nodes: [] }, url: "https://github.com/testowner/testrepo/discussions/99" } } })
            .mockResolvedValueOnce({ addDiscussionComment: { comment: { id: "DC_kwDOABCDEF02", url: "https://github.com/testowner/testrepo/discussions/99#discussioncomment-123" } } })
            .mockResolvedValueOnce({ closeDiscussion: { discussion: { id: "D_kwDOABCDEF01", url: "https://github.com/testowner/testrepo/discussions/99" } } }),
          await eval(`(async () => { ${closeDiscussionScript}; await main(); })()`),
          expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Closing discussion #99")),
          expect(mockCore.setOutput).toHaveBeenCalledWith("discussion_number", 99));
      }),
      it("should skip if not in discussion context with triggering target", async () => {
        const validatedOutput = { items: [{ type: "close_discussion", body: "Closing this discussion." }], errors: [] };
        (setAgentOutput(validatedOutput),
          (mockContext.eventName = "issues"),
          await eval(`(async () => { ${closeDiscussionScript}; await main(); })()`),
          expect(mockCore.info).toHaveBeenCalledWith('Target is "triggering" but not running in discussion context, skipping discussion close'),
          expect(mockCore.setOutput).not.toHaveBeenCalled());
      }),
      it("should handle GraphQL errors gracefully", async () => {
        const validatedOutput = { items: [{ type: "close_discussion", body: "This discussion is resolved." }], errors: [] };
        (setAgentOutput(validatedOutput),
          mockGithub.graphql.mockRejectedValueOnce(new Error("GraphQL error: Discussion not found")),
          await expect(async () => {
            await eval(`(async () => { ${closeDiscussionScript}; await main(); })()`);
          }).rejects.toThrow(),
          expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Failed to close discussion #42")));
      }));
  }));
