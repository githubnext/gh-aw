import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import fs from "fs";
import path from "path";
const mockCore = { debug: vi.fn(), info: vi.fn(), warning: vi.fn(), error: vi.fn(), setFailed: vi.fn(), setOutput: vi.fn(), summary: { addRaw: vi.fn().mockReturnThis(), write: vi.fn().mockResolvedValue() } },
  mockGithub = { rest: {}, graphql: vi.fn() },
  mockContext = { eventName: "issue_comment", runId: 12345, repo: { owner: "testowner", repo: "testrepo" }, payload: { issue: { number: 42 }, repository: { html_url: "https://github.com/testowner/testrepo" } } };
((global.core = mockCore),
  (global.github = mockGithub),
  (global.context = mockContext),
  describe("hide_comment", () => {
    let hideCommentScript, tempFilePath;
    const setAgentOutput = data => {
      tempFilePath = path.join("/tmp", `test_agent_output_${Date.now()}_${Math.random().toString(36).slice(2)}.json`);
      const content = "string" == typeof data ? data : JSON.stringify(data);
      (fs.writeFileSync(tempFilePath, content), (process.env.GH_AW_AGENT_OUTPUT = tempFilePath));
    };
    (beforeEach(() => {
      (vi.clearAllMocks(),
        delete process.env.GH_AW_SAFE_OUTPUTS_STAGED,
        delete process.env.GH_AW_AGENT_OUTPUT,
        delete process.env.GITHUB_SERVER_URL,
        (global.context.eventName = "issue_comment"),
        (global.context.payload.issue = { number: 42 }));
      const scriptPath = path.join(process.cwd(), "hide_comment.cjs");
      hideCommentScript = fs.readFileSync(scriptPath, "utf8");
    }),
      afterEach(() => {
        tempFilePath && fs.existsSync(tempFilePath) && (fs.unlinkSync(tempFilePath), (tempFilePath = void 0));
      }),
      it("should handle empty agent output", async () => {
        (setAgentOutput({ items: [], errors: [] }), await eval(`(async () => { ${hideCommentScript}; await main({}); })()`), expect(mockCore.info).toHaveBeenCalledWith("No hide-comment items found in agent output"));
      }),
      it("should handle missing agent output", async () => {
        (await eval(`(async () => { ${hideCommentScript}; await main({}); })()`), expect(mockCore.info).toHaveBeenCalledWith("No GH_AW_AGENT_OUTPUT environment variable found"));
      }),
      it("should hide a comment successfully", async () => {
        const commentNodeId = "IC_kwDOABCD123456";
        (setAgentOutput({ items: [{ type: "hide_comment", comment_id: commentNodeId }], errors: [] }),
          mockGithub.graphql.mockResolvedValueOnce({ minimizeComment: { minimizedComment: { isMinimized: !0 } } }),
          await eval(`(async () => { ${hideCommentScript}; await main({}); })()`),
          expect(mockCore.info).toHaveBeenCalledWith("Found 1 hide-comment item(s)"),
          expect(mockCore.info).toHaveBeenCalledWith(`Hiding comment: ${commentNodeId} (reason: SPAM)`),
          expect(mockCore.info).toHaveBeenCalledWith(`Successfully hidden comment: ${commentNodeId}`),
          expect(mockGithub.graphql).toHaveBeenCalledWith(expect.stringContaining("minimizeComment"), expect.objectContaining({ nodeId: commentNodeId })),
          expect(mockCore.setOutput).toHaveBeenCalledWith("comment_id", commentNodeId),
          expect(mockCore.setOutput).toHaveBeenCalledWith("is_hidden", "true"));
      }),
      it("should handle GraphQL errors", async () => {
        const commentNodeId = "IC_kwDOABCD123456";
        setAgentOutput({ items: [{ type: "hide_comment", comment_id: commentNodeId }], errors: [] });
        const errorMessage = "Comment not found";
        (mockGithub.graphql.mockRejectedValueOnce(new Error(errorMessage)),
          await eval(`(async () => { ${hideCommentScript}; await main({}); })()`),
          expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining(errorMessage)),
          expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining(errorMessage)));
      }),
      it("should preview hiding in staged mode", async () => {
        process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true";
        const commentNodeId = "IC_kwDOABCD123456";
        (setAgentOutput({ items: [{ type: "hide_comment", comment_id: commentNodeId }], errors: [] }),
          await eval(`(async () => { ${hideCommentScript}; await main({}); })()`),
          expect(mockCore.info).toHaveBeenCalledWith("Found 1 hide-comment item(s)"),
          expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("Staged Mode: Hide Comments Preview")),
          expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining(commentNodeId)),
          expect(mockCore.summary.write).toHaveBeenCalled(),
          expect(mockGithub.graphql).not.toHaveBeenCalled());
      }),
      it("should handle multiple hide-comment items", async () => {
        const commentNodeId1 = "IC_kwDOABCD111111",
          commentNodeId2 = "IC_kwDOABCD222222";
        (setAgentOutput({
          items: [
            { type: "hide_comment", comment_id: commentNodeId1 },
            { type: "hide_comment", comment_id: commentNodeId2 },
          ],
          errors: [],
        }),
          mockGithub.graphql.mockResolvedValueOnce({ minimizeComment: { minimizedComment: { isMinimized: !0 } } }).mockResolvedValueOnce({ minimizeComment: { minimizedComment: { isMinimized: !0 } } }),
          await eval(`(async () => { ${hideCommentScript}; await main({}); })()`),
          expect(mockCore.info).toHaveBeenCalledWith("Found 2 hide-comment item(s)"),
          expect(mockGithub.graphql).toHaveBeenCalledTimes(2),
          expect(mockCore.info).toHaveBeenCalledWith(`Successfully hidden comment: ${commentNodeId1}`),
          expect(mockCore.info).toHaveBeenCalledWith(`Successfully hidden comment: ${commentNodeId2}`));
      }),
      it("should fail when comment_id is missing", async () => {
        (setAgentOutput({ items: [{ type: "hide_comment" }], errors: [] }),
          await eval(`(async () => { ${hideCommentScript}; await main({}); })()`),
          expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("comment_id is required")),
          expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("comment_id is required")));
      }),
      it("should fail when hiding returns false", async () => {
        const commentNodeId = "IC_kwDOABCD123456";
        (setAgentOutput({ items: [{ type: "hide_comment", comment_id: commentNodeId }], errors: [] }),
          mockGithub.graphql.mockResolvedValueOnce({ minimizeComment: { minimizedComment: { isMinimized: !1 } } }),
          await eval(`(async () => { ${hideCommentScript}; await main({}); })()`),
          expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Failed to hide comment")),
          expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Failed to hide comment")));
      }),
      it("should respect allowed-reasons when hiding comments", async () => {
        (setAgentOutput({ items: [{ type: "hide_comment", comment_id: "IC_kwDOABCD123456", reason: "SPAM" }] }),
          mockGithub.graphql.mockResolvedValueOnce({ minimizeComment: { minimizedComment: { isMinimized: !0 } } }),
          await eval(`(async () => { ${hideCommentScript}; await main({ allowed_reasons: ["SPAM", "ABUSE"] }); })()`),
          expect(mockGithub.graphql).toHaveBeenCalledWith(expect.stringContaining("minimizeComment"), expect.objectContaining({ nodeId: "IC_kwDOABCD123456", classifier: "SPAM" })),
          expect(mockCore.info).toHaveBeenCalledWith("Allowed reasons for hiding: [SPAM, ABUSE]"),
          expect(mockCore.info).toHaveBeenCalledWith("Successfully hidden comment: IC_kwDOABCD123456"));
      }),
      it("should skip hiding when reason is not in allowed-reasons", async () => {
        (setAgentOutput({ items: [{ type: "hide_comment", comment_id: "IC_kwDOABCD123456", reason: "OUTDATED" }] }),
          await eval(`(async () => { ${hideCommentScript}; await main({ allowed_reasons: ["SPAM"] }); })()`),
          expect(mockGithub.graphql).not.toHaveBeenCalled(),
          expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining('Reason "OUTDATED" is not in allowed-reasons list')));
      }),
      it("should allow all reasons when allowed-reasons is not specified", async () => {
        (setAgentOutput({ items: [{ type: "hide_comment", comment_id: "IC_kwDOABCD123456", reason: "RESOLVED" }] }),
          mockGithub.graphql.mockResolvedValueOnce({ minimizeComment: { minimizedComment: { isMinimized: !0 } } }),
          await eval(`(async () => { ${hideCommentScript}; await main({}); })()`),
          expect(mockGithub.graphql).toHaveBeenCalledWith(expect.stringContaining("minimizeComment"), expect.objectContaining({ nodeId: "IC_kwDOABCD123456", classifier: "RESOLVED" })),
          expect(mockCore.info).toHaveBeenCalledWith("Successfully hidden comment: IC_kwDOABCD123456"));
      }),
      it("should support lowercase reasons and normalize to uppercase", async () => {
        (setAgentOutput({ items: [{ type: "hide_comment", comment_id: "IC_kwDOABCD123456", reason: "spam" }] }),
          mockGithub.graphql.mockResolvedValueOnce({ minimizeComment: { minimizedComment: { isMinimized: !0 } } }),
          await eval(`(async () => { ${hideCommentScript}; await main({ allowed_reasons: ["spam", "abuse"] }); })()`),
          expect(mockGithub.graphql).toHaveBeenCalledWith(expect.stringContaining("minimizeComment"), expect.objectContaining({ nodeId: "IC_kwDOABCD123456", classifier: "SPAM" })),
          expect(mockCore.info).toHaveBeenCalledWith("Successfully hidden comment: IC_kwDOABCD123456"));
      }));
  }));
