import { describe, it, expect, beforeEach, vi } from "vitest";
import fs from "fs";
import path from "path";
const mockCore = {
    debug: vi.fn(),
    info: vi.fn(),
    notice: vi.fn(),
    warning: vi.fn(),
    error: vi.fn(),
    setFailed: vi.fn(),
    setOutput: vi.fn(),
    exportVariable: vi.fn(),
    setSecret: vi.fn(),
    getInput: vi.fn(),
    getBooleanInput: vi.fn(),
    getMultilineInput: vi.fn(),
    getState: vi.fn(),
    saveState: vi.fn(),
    startGroup: vi.fn(),
    endGroup: vi.fn(),
    group: vi.fn(),
    addPath: vi.fn(),
    setCommandEcho: vi.fn(),
    isDebug: vi.fn().mockReturnValue(!1),
    getIDToken: vi.fn(),
    toPlatformPath: vi.fn(),
    toPosixPath: vi.fn(),
    toWin32Path: vi.fn(),
    summary: { addRaw: vi.fn().mockReturnThis(), write: vi.fn().mockResolvedValue() },
  },
  mockGithub = { rest: { issues: { createComment: vi.fn() } } },
  mockContext = { eventName: "issues", runId: 12345, repo: { owner: "testowner", repo: "testrepo" }, payload: { issue: { number: 123 }, repository: { html_url: "https://github.com/testowner/testrepo" } } };
((global.core = mockCore),
  (global.github = mockGithub),
  (global.context = mockContext),
  describe("add_comment.cjs", () => {
    let createCommentScript, tempFilePath;
    const setAgentOutput = data => {
      tempFilePath = path.join("/tmp", `test_agent_output_${Date.now()}_${Math.random().toString(36).slice(2)}.json`);
      const content = "string" == typeof data ? data : JSON.stringify(data);
      (fs.writeFileSync(tempFilePath, content), (process.env.GH_AW_AGENT_OUTPUT = tempFilePath));
    };
    (beforeEach(() => {
      (vi.clearAllMocks(), delete process.env.GH_AW_AGENT_OUTPUT, delete process.env.GITHUB_WORKFLOW, (global.context.eventName = "issues"), (global.context.payload.issue = { number: 123 }));
      const scriptPath = path.join(process.cwd(), "add_comment.cjs");
      createCommentScript = fs.readFileSync(scriptPath, "utf8");
    }),
      afterEach(() => {
        tempFilePath && require("fs").existsSync(tempFilePath) && (require("fs").unlinkSync(tempFilePath), (tempFilePath = void 0));
      }),
      it("should skip when no agent output is provided", async () => {
        (delete process.env.GH_AW_AGENT_OUTPUT,
          await eval(`(async () => { ${createCommentScript}; await main(); })()`),
          expect(mockCore.info).toHaveBeenCalledWith("No GH_AW_AGENT_OUTPUT environment variable found"),
          expect(mockGithub.rest.issues.createComment).not.toHaveBeenCalled());
      }),
      it("should skip when agent output is empty", async () => {
        (setAgentOutput(""), await eval(`(async () => { ${createCommentScript}; await main(); })()`), expect(mockCore.info).toHaveBeenCalledWith("Agent output content is empty"), expect(mockGithub.rest.issues.createComment).not.toHaveBeenCalled());
      }),
      it("should skip when not in issue or PR context", async () => {
        (setAgentOutput({ items: [{ type: "add_comment", body: "Test comment content" }] }),
          (global.context.eventName = "push"),
          await eval(`(async () => { ${createCommentScript}; await main(); })()`),
          expect(mockCore.info).toHaveBeenCalledWith('Target is "triggering" but not running in issue, pull request, or discussion context, skipping comment creation'),
          expect(mockGithub.rest.issues.createComment).not.toHaveBeenCalled());
      }),
      it("should create comment on issue successfully", async () => {
        (setAgentOutput({ items: [{ type: "add_comment", body: "Test comment content" }] }), (global.context.eventName = "issues"));
        const mockComment = { id: 456, html_url: "https://github.com/testowner/testrepo/issues/123#issuecomment-456" };
        (mockGithub.rest.issues.createComment.mockResolvedValue({ data: mockComment }),
          await eval(`(async () => { ${createCommentScript}; await main(); })()`),
          expect(mockGithub.rest.issues.createComment).toHaveBeenCalledWith({ owner: "testowner", repo: "testrepo", issue_number: 123, body: expect.stringContaining("Test comment content") }),
          expect(mockCore.setOutput).toHaveBeenCalledWith("comment_id", 456),
          expect(mockCore.setOutput).toHaveBeenCalledWith("comment_url", mockComment.html_url),
          expect(mockCore.summary.addRaw).toHaveBeenCalled(),
          expect(mockCore.summary.write).toHaveBeenCalled());
      }),
      it("should create comment on pull request successfully", async () => {
        (setAgentOutput({ items: [{ type: "add_comment", body: "Test PR comment content" }] }), (global.context.eventName = "pull_request"), (global.context.payload.pull_request = { number: 789 }), delete global.context.payload.issue);
        const mockComment = { id: 789, html_url: "https://github.com/testowner/testrepo/issues/789#issuecomment-789" };
        (mockGithub.rest.issues.createComment.mockResolvedValue({ data: mockComment }),
          await eval(`(async () => { ${createCommentScript}; await main(); })()`),
          expect(mockGithub.rest.issues.createComment).toHaveBeenCalledWith({ owner: "testowner", repo: "testrepo", issue_number: 789, body: expect.stringContaining("Test PR comment content") }));
      }),
      it("should include run information in comment body", async () => {
        (setAgentOutput({ items: [{ type: "add_comment", body: "Test content" }] }), (global.context.eventName = "issues"), (global.context.payload.issue = { number: 123 }));
        const mockComment = { id: 456, html_url: "https://github.com/testowner/testrepo/issues/123#issuecomment-456" };
        (mockGithub.rest.issues.createComment.mockResolvedValue({ data: mockComment }),
          await eval(`(async () => { ${createCommentScript}; await main(); })()`),
          expect(mockGithub.rest.issues.createComment).toHaveBeenCalled(),
          expect(mockGithub.rest.issues.createComment.mock.calls).toHaveLength(1));
        const callArgs = mockGithub.rest.issues.createComment.mock.calls[0][0];
        (expect(callArgs.body).toContain("Test content"), expect(callArgs.body).toContain("This treasure was crafted by"), expect(callArgs.body).toContain("https://github.com/testowner/testrepo/actions/runs/12345"));
      }),
      it("should include workflow source in footer when GH_AW_WORKFLOW_SOURCE is provided", async () => {
        (setAgentOutput({ items: [{ type: "add_comment", body: "Test content with source" }] }),
          (process.env.GH_AW_WORKFLOW_NAME = "Test Workflow"),
          (process.env.GH_AW_WORKFLOW_SOURCE = "githubnext/agentics/workflows/ci-doctor.md@v1.0.0"),
          (process.env.GH_AW_WORKFLOW_SOURCE_URL = "https://github.com/githubnext/agentics/tree/v1.0.0/workflows/ci-doctor.md"),
          (global.context.eventName = "issues"),
          (global.context.payload.issue = { number: 123 }));
        const mockComment = { id: 456, html_url: "https://github.com/testowner/testrepo/issues/123#issuecomment-456" };
        (mockGithub.rest.issues.createComment.mockResolvedValue({ data: mockComment }), await eval(`(async () => { ${createCommentScript}; await main(); })()`), expect(mockGithub.rest.issues.createComment).toHaveBeenCalled());
        const callArgs = mockGithub.rest.issues.createComment.mock.calls[0][0];
        (expect(callArgs.body).toContain("Test content with source"),
          expect(callArgs.body).toContain("[🏴‍☠️ Test Workflow]"),
          expect(callArgs.body).toContain("https://github.com/testowner/testrepo/actions/runs/12345"),
          expect(callArgs.body).toContain("gh aw add githubnext/agentics/workflows/ci-doctor.md@v1.0.0"));
      }),
      it("should not include workflow source footer when GH_AW_WORKFLOW_SOURCE is not provided", async () => {
        (setAgentOutput({ items: [{ type: "add_comment", body: "Test content without source" }] }),
          (process.env.GH_AW_WORKFLOW_NAME = "Test Workflow"),
          delete process.env.GH_AW_WORKFLOW_SOURCE,
          (global.context.eventName = "issues"),
          (global.context.payload.issue = { number: 123 }));
        const mockComment = { id: 456, html_url: "https://github.com/testowner/testrepo/issues/123#issuecomment-456" };
        (mockGithub.rest.issues.createComment.mockResolvedValue({ data: mockComment }), await eval(`(async () => { ${createCommentScript}; await main(); })()`), expect(mockGithub.rest.issues.createComment).toHaveBeenCalled());
        const callArgs = mockGithub.rest.issues.createComment.mock.calls[0][0];
        (expect(callArgs.body).toContain("Test content without source"), expect(callArgs.body).toContain("[🏴‍☠️ Test Workflow]"), expect(callArgs.body).not.toContain("gh aw add"));
      }),
      it("should use GITHUB_SERVER_URL when repository context is not available", async () => {
        (setAgentOutput({ items: [{ type: "add_comment", body: "Test content with custom server" }] }),
          (process.env.GITHUB_SERVER_URL = "https://github.enterprise.com"),
          (global.context.eventName = "issues"),
          (global.context.payload.issue = { number: 123 }),
          delete global.context.payload.repository);
        const mockComment = { id: 456, html_url: "https://github.enterprise.com/testowner/testrepo/issues/123#issuecomment-456" };
        (mockGithub.rest.issues.createComment.mockResolvedValue({ data: mockComment }), await eval(`(async () => { ${createCommentScript}; await main(); })()`), expect(mockGithub.rest.issues.createComment).toHaveBeenCalled());
        const callArgs = mockGithub.rest.issues.createComment.mock.calls[0][0];
        (expect(callArgs.body).toContain("Test content with custom server"),
          expect(callArgs.body).toContain("https://github.enterprise.com/testowner/testrepo/actions/runs/12345"),
          expect(callArgs.body).not.toContain("https://github.com/testowner/testrepo/actions/runs/12345"),
          delete process.env.GITHUB_SERVER_URL);
      }),
      it("should fallback to https://github.com when GITHUB_SERVER_URL is not set and repository context is missing", async () => {
        (setAgentOutput({ items: [{ type: "add_comment", body: "Test content with fallback" }] }),
          delete process.env.GITHUB_SERVER_URL,
          (global.context.eventName = "issues"),
          (global.context.payload.issue = { number: 123 }),
          delete global.context.payload.repository);
        const mockComment = { id: 456, html_url: "https://github.com/testowner/testrepo/issues/123#issuecomment-456" };
        (mockGithub.rest.issues.createComment.mockResolvedValue({ data: mockComment }), await eval(`(async () => { ${createCommentScript}; await main(); })()`), expect(mockGithub.rest.issues.createComment).toHaveBeenCalled());
        const callArgs = mockGithub.rest.issues.createComment.mock.calls[0][0];
        (expect(callArgs.body).toContain("Test content with fallback"), expect(callArgs.body).toContain("https://github.com/testowner/testrepo/actions/runs/12345"));
      }),
      it("should include triggering issue number in footer when in issue context", async () => {
        (setAgentOutput({ items: [{ type: "add_comment", body: "Comment from issue context" }] }), (process.env.GH_AW_WORKFLOW_NAME = "Test Workflow"), (global.context.eventName = "issues"), (global.context.payload.issue = { number: 42 }));
        const mockComment = { id: 789, html_url: "https://github.com/testowner/testrepo/issues/42#issuecomment-789" };
        (mockGithub.rest.issues.createComment.mockResolvedValue({ data: mockComment }), await eval(`(async () => { ${createCommentScript}; await main(); })()`));
        const callArgs = mockGithub.rest.issues.createComment.mock.calls[0][0];
        (expect(callArgs.body).toContain("Comment from issue context"), expect(callArgs.body).toContain("[🏴‍☠️ Test Workflow]"), expect(callArgs.body).toContain("#42"));
      }),
      it("should include triggering PR number in footer when in PR context", async () => {
        (setAgentOutput({ items: [{ type: "add_comment", body: "Comment from PR context" }] }),
          (process.env.GH_AW_WORKFLOW_NAME = "Test Workflow"),
          (global.context.eventName = "pull_request"),
          delete global.context.payload.issue,
          (global.context.payload.pull_request = { number: 123 }));
        const mockComment = { id: 890, html_url: "https://github.com/testowner/testrepo/pull/123#issuecomment-890" };
        (mockGithub.rest.issues.createComment.mockResolvedValue({ data: mockComment }), await eval(`(async () => { ${createCommentScript}; await main(); })()`));
        const callArgs = mockGithub.rest.issues.createComment.mock.calls[0][0];
        (expect(callArgs.body).toContain("Comment from PR context"), expect(callArgs.body).toContain("[🏴‍☠️ Test Workflow]"), expect(callArgs.body).toContain("#123"), delete global.context.payload.pull_request);
      }),
      it("should use header level 4 for related items in comments", async () => {
        (setAgentOutput({ items: [{ type: "add_comment", body: "Test comment with related items" }] }),
          (global.context.eventName = "issues"),
          (global.context.payload.issue = { number: 123 }),
          (process.env.GH_AW_CREATED_ISSUE_URL = "https://github.com/testowner/testrepo/issues/456"),
          (process.env.GH_AW_CREATED_ISSUE_NUMBER = "456"),
          (process.env.GH_AW_CREATED_DISCUSSION_URL = "https://github.com/testowner/testrepo/discussions/789"),
          (process.env.GH_AW_CREATED_DISCUSSION_NUMBER = "789"),
          (process.env.GH_AW_CREATED_PULL_REQUEST_URL = "https://github.com/testowner/testrepo/pull/101"),
          (process.env.GH_AW_CREATED_PULL_REQUEST_NUMBER = "101"));
        const mockComment = { id: 890, html_url: "https://github.com/testowner/testrepo/issues/123#issuecomment-890" };
        (mockGithub.rest.issues.createComment.mockResolvedValue({ data: mockComment }), await eval(`(async () => { ${createCommentScript}; await main(); })()`));
        const callArgs = mockGithub.rest.issues.createComment.mock.calls[0][0];
        (expect(callArgs.body).toContain("#### Related Items"),
          expect(callArgs.body).toMatch(/####\s+Related Items/),
          expect(callArgs.body).not.toMatch(/^##\s+Related Items/m),
          expect(callArgs.body).not.toMatch(/\*\*Related Items:\*\*/),
          expect(callArgs.body).toContain("- Issue: [#456](https://github.com/testowner/testrepo/issues/456)"),
          expect(callArgs.body).toContain("- Discussion: [#789](https://github.com/testowner/testrepo/discussions/789)"),
          expect(callArgs.body).toContain("- Pull Request: [#101](https://github.com/testowner/testrepo/pull/101)"),
          delete process.env.GH_AW_CREATED_ISSUE_URL,
          delete process.env.GH_AW_CREATED_ISSUE_NUMBER,
          delete process.env.GH_AW_CREATED_DISCUSSION_URL,
          delete process.env.GH_AW_CREATED_DISCUSSION_NUMBER,
          delete process.env.GH_AW_CREATED_PULL_REQUEST_URL,
          delete process.env.GH_AW_CREATED_PULL_REQUEST_NUMBER);
      }),
      it("should use header level 4 for related items in staged mode preview", async () => {
        (setAgentOutput({ items: [{ type: "add_comment", body: "Test comment in staged mode" }] }),
          (global.context.eventName = "issues"),
          (global.context.payload.issue = { number: 123 }),
          (process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true"),
          (process.env.GH_AW_CREATED_ISSUE_URL = "https://github.com/testowner/testrepo/issues/456"),
          (process.env.GH_AW_CREATED_ISSUE_NUMBER = "456"),
          (process.env.GH_AW_CREATED_DISCUSSION_URL = "https://github.com/testowner/testrepo/discussions/789"),
          (process.env.GH_AW_CREATED_DISCUSSION_NUMBER = "789"),
          (process.env.GH_AW_CREATED_PULL_REQUEST_URL = "https://github.com/testowner/testrepo/pull/101"),
          (process.env.GH_AW_CREATED_PULL_REQUEST_NUMBER = "101"),
          await eval(`(async () => { ${createCommentScript}; await main(); })()`),
          expect(mockCore.summary.addRaw).toHaveBeenCalled());
        const summaryContent = mockCore.summary.addRaw.mock.calls[0][0];
        (expect(summaryContent).toContain("#### Related Items"),
          expect(summaryContent).toMatch(/####\s+Related Items/),
          expect(summaryContent).not.toMatch(/^##\s+Related Items/m),
          expect(summaryContent).not.toMatch(/\*\*Related Items:\*\*/),
          expect(summaryContent).toContain("- Issue: [#456](https://github.com/testowner/testrepo/issues/456)"),
          expect(summaryContent).toContain("- Discussion: [#789](https://github.com/testowner/testrepo/discussions/789)"),
          expect(summaryContent).toContain("- Pull Request: [#101](https://github.com/testowner/testrepo/pull/101)"),
          delete process.env.GH_AW_SAFE_OUTPUTS_STAGED,
          delete process.env.GH_AW_CREATED_ISSUE_URL,
          delete process.env.GH_AW_CREATED_ISSUE_NUMBER,
          delete process.env.GH_AW_CREATED_DISCUSSION_URL,
          delete process.env.GH_AW_CREATED_DISCUSSION_NUMBER,
          delete process.env.GH_AW_CREATED_PULL_REQUEST_URL,
          delete process.env.GH_AW_CREATED_PULL_REQUEST_NUMBER);
      }),
      it("should create comment on discussion using GraphQL when in discussion_comment context", async () => {
        (setAgentOutput({ items: [{ type: "add_comment", body: "Test discussion comment" }] }),
          (global.context.eventName = "discussion_comment"),
          (global.context.payload.discussion = { number: 1993 }),
          (global.context.payload.comment = { id: 12345, node_id: "DC_kwDOABcD1M4AaBbC" }),
          delete global.context.payload.issue,
          delete global.context.payload.pull_request);
        const mockGraphqlResponse = vi.fn();
        (mockGraphqlResponse.mockResolvedValueOnce({ repository: { discussion: { id: "D_kwDOPc1QR84BpqRs", url: "https://github.com/testowner/testrepo/discussions/1993" } } }).mockResolvedValueOnce({
          addDiscussionComment: { comment: { id: "DC_kwDOPc1QR84BpqRt", body: "Test discussion comment", createdAt: "2025-10-19T22:00:00Z", url: "https://github.com/testowner/testrepo/discussions/1993#discussioncomment-123" } },
        }),
          (global.github.graphql = mockGraphqlResponse),
          await eval(`(async () => { ${createCommentScript}; await main(); })()`),
          expect(mockGraphqlResponse).toHaveBeenCalledTimes(2),
          expect(mockGraphqlResponse.mock.calls[0][0]).toContain("query"),
          expect(mockGraphqlResponse.mock.calls[0][0]).toContain("discussion(number: $num)"),
          expect(mockGraphqlResponse.mock.calls[0][1]).toEqual({ owner: "testowner", repo: "testrepo", num: 1993 }),
          expect(mockGraphqlResponse.mock.calls[1][0]).toContain("mutation"),
          expect(mockGraphqlResponse.mock.calls[1][0]).toContain("addDiscussionComment"),
          expect(mockGraphqlResponse.mock.calls[1][0]).toContain("replyToId"),
          expect(mockGraphqlResponse.mock.calls[1][1].body).toContain("Test discussion comment"),
          expect(mockGraphqlResponse.mock.calls[1][1].replyToId).toBe("DC_kwDOABcD1M4AaBbC"),
          expect(mockGithub.rest.issues.createComment).not.toHaveBeenCalled(),
          expect(mockCore.setOutput).toHaveBeenCalledWith("comment_id", "DC_kwDOPc1QR84BpqRt"),
          expect(mockCore.setOutput).toHaveBeenCalledWith("comment_url", "https://github.com/testowner/testrepo/discussions/1993#discussioncomment-123"),
          delete global.github.graphql,
          delete global.context.payload.discussion,
          delete global.context.payload.comment);
      }),
      it("should create comment on discussion using GraphQL when GITHUB_AW_COMMENT_DISCUSSION is true (explicit discussion mode)", async () => {
        (setAgentOutput({ items: [{ type: "add_comment", body: "Test explicit discussion comment", item_number: 2001 }] }),
          (process.env.GH_AW_COMMENT_TARGET = "*"),
          (process.env.GITHUB_AW_COMMENT_DISCUSSION = "true"),
          (global.context.eventName = "issues"),
          (global.context.payload.issue = { number: 123 }),
          delete global.context.payload.discussion,
          delete global.context.payload.pull_request);
        const mockGraphqlResponse = vi.fn();
        (mockGraphqlResponse.mockResolvedValueOnce({ repository: { discussion: { id: "D_kwDOPc1QR84BpqRu", url: "https://github.com/testowner/testrepo/discussions/2001" } } }).mockResolvedValueOnce({
          addDiscussionComment: { comment: { id: "DC_kwDOPc1QR84BpqRv", body: "Test explicit discussion comment", createdAt: "2025-10-22T12:00:00Z", url: "https://github.com/testowner/testrepo/discussions/2001#discussioncomment-456" } },
        }),
          (global.github.graphql = mockGraphqlResponse),
          await eval(`(async () => { ${createCommentScript}; await main(); })()`),
          expect(mockGraphqlResponse).toHaveBeenCalledTimes(2),
          expect(mockGraphqlResponse.mock.calls[0][0]).toContain("query"),
          expect(mockGraphqlResponse.mock.calls[0][0]).toContain("discussion(number: $num)"),
          expect(mockGraphqlResponse.mock.calls[0][1]).toEqual({ owner: "testowner", repo: "testrepo", num: 2001 }),
          expect(mockGraphqlResponse.mock.calls[1][0]).toContain("mutation"),
          expect(mockGraphqlResponse.mock.calls[1][0]).toContain("addDiscussionComment"),
          expect(mockGraphqlResponse.mock.calls[1][1].body).toContain("Test explicit discussion comment"),
          expect(mockGraphqlResponse.mock.calls[1][1].replyToId).toBeUndefined(),
          expect(mockGithub.rest.issues.createComment).not.toHaveBeenCalled(),
          expect(mockCore.setOutput).toHaveBeenCalledWith("comment_id", "DC_kwDOPc1QR84BpqRv"),
          expect(mockCore.setOutput).toHaveBeenCalledWith("comment_url", "https://github.com/testowner/testrepo/discussions/2001#discussioncomment-456"),
          expect(mockCore.info).toHaveBeenCalledWith("Creating comment on discussion #2001"),
          delete process.env.GH_AW_COMMENT_TARGET,
          delete process.env.GITHUB_AW_COMMENT_DISCUSSION,
          delete global.github.graphql);
      }),
      it("should replace temporary ID references in comment body using the temporary ID map", async () => {
        (setAgentOutput({ items: [{ type: "add_comment", body: "This comment references issue #aw_aabbccdd1122 which was created earlier." }] }),
          (process.env.GH_AW_TEMPORARY_ID_MAP = JSON.stringify({ aw_aabbccdd1122: 456 })),
          mockGithub.rest.issues.createComment.mockResolvedValue({ data: { id: 99999, html_url: "https://github.com/testowner/testrepo/issues/123#issuecomment-99999" } }),
          await eval(`(async () => { ${createCommentScript}; await main(); })()`),
          expect(mockGithub.rest.issues.createComment).toHaveBeenCalledWith(expect.objectContaining({ body: expect.stringContaining("#456") })),
          expect(mockGithub.rest.issues.createComment).toHaveBeenCalledWith(expect.objectContaining({ body: expect.not.stringContaining("#aw_aabbccdd1122") })),
          delete process.env.GH_AW_TEMPORARY_ID_MAP);
      }),
      it("should load temporary ID map and log the count", async () => {
        (setAgentOutput({ items: [{ type: "add_comment", body: "Test comment" }] }),
          (process.env.GH_AW_TEMPORARY_ID_MAP = JSON.stringify({ aw_abc123: 100, aw_def456: 200 })),
          mockGithub.rest.issues.createComment.mockResolvedValue({ data: { id: 99999, html_url: "https://github.com/testowner/testrepo/issues/123#issuecomment-99999" } }),
          await eval(`(async () => { ${createCommentScript}; await main(); })()`),
          expect(mockCore.info).toHaveBeenCalledWith("Loaded temporary ID map with 2 entries"),
          delete process.env.GH_AW_TEMPORARY_ID_MAP);
      }),
      it("should handle empty temporary ID map gracefully", async () => {
        (setAgentOutput({ items: [{ type: "add_comment", body: "Comment with #aw_000000000000 that won't be resolved" }] }),
          (process.env.GH_AW_TEMPORARY_ID_MAP = "{}"),
          mockGithub.rest.issues.createComment.mockResolvedValue({ data: { id: 99999, html_url: "https://github.com/testowner/testrepo/issues/123#issuecomment-99999" } }),
          await eval(`(async () => { ${createCommentScript}; await main(); })()`),
          expect(mockGithub.rest.issues.createComment).toHaveBeenCalledWith(expect.objectContaining({ body: expect.stringContaining("#aw_000000000000") })),
          delete process.env.GH_AW_TEMPORARY_ID_MAP);
      }),
      it("should use custom footer message when GH_AW_SAFE_OUTPUT_MESSAGES is configured", async () => {
        (setAgentOutput({ items: [{ type: "add_comment", body: "Test comment with custom footer" }] }),
          (process.env.GH_AW_WORKFLOW_NAME = "Custom Workflow"),
          (process.env.GH_AW_SAFE_OUTPUT_MESSAGES = JSON.stringify({ footer: "> Custom AI footer by [{workflow_name}]({run_url})", footerInstall: "> Custom install: `gh aw add {workflow_source}`" })),
          (global.context.eventName = "issues"),
          (global.context.payload.issue = { number: 456 }));
        const mockComment = { id: 999, html_url: "https://github.com/testowner/testrepo/issues/456#issuecomment-999" };
        (mockGithub.rest.issues.createComment.mockResolvedValue({ data: mockComment }), await eval(`(async () => { ${createCommentScript}; await main(); })()`));
        const callArgs = mockGithub.rest.issues.createComment.mock.calls[0][0];
        (expect(callArgs.body).toContain("Test comment with custom footer"),
          expect(callArgs.body).toContain("Custom AI footer by [Custom Workflow]"),
          expect(callArgs.body).toContain("https://github.com/testowner/testrepo/actions/runs/12345"),
          expect(callArgs.body).not.toContain("Ahoy!"),
          expect(callArgs.body).not.toContain("treasure was crafted"),
          delete process.env.GH_AW_SAFE_OUTPUT_MESSAGES);
      }),
      it("should use custom footer with install instructions when workflow source is provided", async () => {
        (setAgentOutput({ items: [{ type: "add_comment", body: "Test comment with custom footer and install" }] }),
          (process.env.GH_AW_WORKFLOW_NAME = "Custom Workflow"),
          (process.env.GH_AW_WORKFLOW_SOURCE = "owner/repo/workflow.md@main"),
          (process.env.GH_AW_WORKFLOW_SOURCE_URL = "https://github.com/owner/repo"),
          (process.env.GH_AW_SAFE_OUTPUT_MESSAGES = JSON.stringify({ footer: "> Generated by [{workflow_name}]({run_url})", footerInstall: "> Install: `gh aw add {workflow_source}`" })),
          (global.context.eventName = "issues"),
          (global.context.payload.issue = { number: 789 }));
        const mockComment = { id: 1001, html_url: "https://github.com/testowner/testrepo/issues/789#issuecomment-1001" };
        (mockGithub.rest.issues.createComment.mockResolvedValue({ data: mockComment }), await eval(`(async () => { ${createCommentScript}; await main(); })()`));
        const callArgs = mockGithub.rest.issues.createComment.mock.calls[0][0];
        (expect(callArgs.body).toContain("Test comment with custom footer and install"),
          expect(callArgs.body).toContain("Generated by [Custom Workflow]"),
          expect(callArgs.body).toContain("Install: `gh aw add owner/repo/workflow.md@main`"),
          expect(callArgs.body).not.toContain("Ahoy!"),
          expect(callArgs.body).not.toContain("plunder this workflow"),
          delete process.env.GH_AW_SAFE_OUTPUT_MESSAGES,
          delete process.env.GH_AW_WORKFLOW_SOURCE,
          delete process.env.GH_AW_WORKFLOW_SOURCE_URL);
      }),
      it("should hide older comments when hide-older-comments is enabled", async () => {
        (setAgentOutput({ items: [{ type: "add_comment", body: "New comment from workflow" }] }),
          (process.env.GITHUB_WORKFLOW = "test-workflow-123"),
          (process.env.GH_AW_HIDE_OLDER_COMMENTS = "true"),
          (global.context.eventName = "issues"),
          (global.context.payload.issue = { number: 100 }),
          (mockGithub.rest.issues.listComments = vi.fn().mockResolvedValue({
            data: [
              { id: 1, node_id: "IC_oldcomment1", body: "Old comment 1\n\n\x3c!-- workflow-id: test-workflow-123 --\x3e" },
              { id: 2, node_id: "IC_oldcomment2", body: "Old comment 2\n\n\x3c!-- workflow-id: test-workflow-123 --\x3e" },
              { id: 3, node_id: "IC_othercomment", body: "Comment from different workflow" },
            ],
          })),
          (mockGithub.graphql = vi.fn().mockResolvedValue({ minimizeComment: { minimizedComment: { isMinimized: !0 } } })));
        const mockNewComment = { id: 4, html_url: "https://github.com/testowner/testrepo/issues/100#issuecomment-4" };
        (mockGithub.rest.issues.createComment.mockResolvedValue({ data: mockNewComment }),
          await eval(`(async () => { ${createCommentScript}; await main(); })()`),
          expect(mockGithub.rest.issues.listComments).toHaveBeenCalledWith({ owner: "testowner", repo: "testrepo", issue_number: 100, per_page: 100, page: 1 }),
          expect(mockGithub.graphql).toHaveBeenCalledTimes(2),
          expect(mockGithub.graphql).toHaveBeenCalledWith(expect.stringContaining("minimizeComment"), expect.objectContaining({ nodeId: "IC_oldcomment1", classifier: "OUTDATED" })),
          expect(mockGithub.graphql).toHaveBeenCalledWith(expect.stringContaining("minimizeComment"), expect.objectContaining({ nodeId: "IC_oldcomment2", classifier: "OUTDATED" })),
          expect(mockGithub.rest.issues.createComment).toHaveBeenCalledWith(expect.objectContaining({ owner: "testowner", repo: "testrepo", issue_number: 100, body: expect.stringContaining("New comment from workflow") })),
          delete process.env.GITHUB_WORKFLOW,
          delete process.env.GH_AW_HIDE_OLDER_COMMENTS);
      }),
      it("should not hide comments when hide-older-comments is not enabled", async () => {
        (setAgentOutput({ items: [{ type: "add_comment", body: "New comment without hiding" }] }),
          (process.env.GITHUB_WORKFLOW = "test-workflow-456"),
          (global.context.eventName = "issues"),
          (global.context.payload.issue = { number: 200 }),
          (mockGithub.rest.issues.listComments = vi.fn()),
          (mockGithub.graphql = vi.fn()));
        const mockNewComment = { id: 5, html_url: "https://github.com/testowner/testrepo/issues/200#issuecomment-5" };
        (mockGithub.rest.issues.createComment.mockResolvedValue({ data: mockNewComment }),
          await eval(`(async () => { ${createCommentScript}; await main(); })()`),
          expect(mockGithub.rest.issues.listComments).not.toHaveBeenCalled(),
          expect(mockGithub.graphql).not.toHaveBeenCalled(),
          expect(mockGithub.rest.issues.createComment).toHaveBeenCalled(),
          delete process.env.GITHUB_WORKFLOW);
      }),
      it("should skip hiding when workflow-id is not available", async () => {
        (setAgentOutput({ items: [{ type: "add_comment", body: "Comment without workflow-id" }] }),
          (process.env.GH_AW_HIDE_OLDER_COMMENTS = "true"),
          (global.context.eventName = "issues"),
          (global.context.payload.issue = { number: 300 }),
          (mockGithub.rest.issues.listComments = vi.fn()),
          (mockGithub.graphql = vi.fn()));
        const mockNewComment = { id: 6, html_url: "https://github.com/testowner/testrepo/issues/300#issuecomment-6" };
        (mockGithub.rest.issues.createComment.mockResolvedValue({ data: mockNewComment }),
          await eval(`(async () => { ${createCommentScript}; await main(); })()`),
          expect(mockGithub.rest.issues.listComments).not.toHaveBeenCalled(),
          expect(mockGithub.graphql).not.toHaveBeenCalled(),
          expect(mockGithub.rest.issues.createComment).toHaveBeenCalled(),
          delete process.env.GH_AW_HIDE_OLDER_COMMENTS);
      }),
      it("should respect allowed-reasons when hiding comments", async () => {
        (setAgentOutput({ items: [{ type: "add_comment", body: "New comment with allowed reasons" }] }),
          (process.env.GITHUB_WORKFLOW = "test-workflow-789"),
          (process.env.GH_AW_HIDE_OLDER_COMMENTS = "true"),
          (process.env.GH_AW_ALLOWED_REASONS = JSON.stringify(["OUTDATED", "RESOLVED"])),
          (global.context.eventName = "issues"),
          (global.context.payload.issue = { number: 400 }),
          (mockGithub.rest.issues.listComments = vi.fn().mockResolvedValue({ data: [{ id: 1, node_id: "IC_oldcomment1", body: "Old comment\n\n\x3c!-- workflow-id: test-workflow-789 --\x3e" }] })),
          (mockGithub.graphql = vi.fn().mockResolvedValue({ minimizeComment: { minimizedComment: { isMinimized: !0 } } })));
        const mockNewComment = { id: 2, html_url: "https://github.com/testowner/testrepo/issues/400#issuecomment-2" };
        (mockGithub.rest.issues.createComment.mockResolvedValue({ data: mockNewComment }),
          await eval(`(async () => { ${createCommentScript}; await main(); })()`),
          expect(mockGithub.graphql).toHaveBeenCalledWith(expect.stringContaining("minimizeComment"), expect.objectContaining({ nodeId: "IC_oldcomment1", classifier: "OUTDATED" })),
          delete process.env.GITHUB_WORKFLOW,
          delete process.env.GH_AW_HIDE_OLDER_COMMENTS,
          delete process.env.GH_AW_ALLOWED_REASONS);
      }),
      it("should skip hiding when reason is not in allowed-reasons", async () => {
        (setAgentOutput({ items: [{ type: "add_comment", body: "New comment with restricted reasons" }] }),
          (process.env.GITHUB_WORKFLOW = "test-workflow-999"),
          (process.env.GH_AW_HIDE_OLDER_COMMENTS = "true"),
          (process.env.GH_AW_ALLOWED_REASONS = JSON.stringify(["SPAM"])),
          (global.context.eventName = "issues"),
          (global.context.payload.issue = { number: 500 }),
          (mockGithub.rest.issues.listComments = vi.fn()),
          (mockGithub.graphql = vi.fn()));
        const mockNewComment = { id: 3, html_url: "https://github.com/testowner/testrepo/issues/500#issuecomment-3" };
        (mockGithub.rest.issues.createComment.mockResolvedValue({ data: mockNewComment }),
          await eval(`(async () => { ${createCommentScript}; await main(); })()`),
          expect(mockGithub.rest.issues.listComments).not.toHaveBeenCalled(),
          expect(mockGithub.graphql).not.toHaveBeenCalled(),
          expect(mockGithub.rest.issues.createComment).toHaveBeenCalled(),
          delete process.env.GITHUB_WORKFLOW,
          delete process.env.GH_AW_HIDE_OLDER_COMMENTS,
          delete process.env.GH_AW_ALLOWED_REASONS);
      }),
      it("should support lowercase allowed-reasons", async () => {
        (setAgentOutput({ items: [{ type: "add_comment", body: "New comment with lowercase reasons" }] }),
          (process.env.GITHUB_WORKFLOW = "test-workflow-lowercase"),
          (process.env.GH_AW_HIDE_OLDER_COMMENTS = "true"),
          (process.env.GH_AW_ALLOWED_REASONS = JSON.stringify(["outdated", "resolved"])),
          (global.context.eventName = "issues"),
          (global.context.payload.issue = { number: 600 }),
          (mockGithub.rest.issues.listComments = vi.fn().mockResolvedValue({ data: [{ id: 1, node_id: "IC_oldcomment1", body: "Old comment\n\n\x3c!-- workflow-id: test-workflow-lowercase --\x3e" }] })),
          (mockGithub.graphql = vi.fn().mockResolvedValue({ minimizeComment: { minimizedComment: { isMinimized: !0 } } })));
        const mockNewComment = { id: 4, html_url: "https://github.com/testowner/testrepo/issues/600#issuecomment-4" };
        (mockGithub.rest.issues.createComment.mockResolvedValue({ data: mockNewComment }),
          await eval(`(async () => { ${createCommentScript}; await main(); })()`),
          expect(mockGithub.graphql).toHaveBeenCalledWith(expect.stringContaining("minimizeComment"), expect.objectContaining({ nodeId: "IC_oldcomment1", classifier: "OUTDATED" })),
          delete process.env.GITHUB_WORKFLOW,
          delete process.env.GH_AW_HIDE_OLDER_COMMENTS,
          delete process.env.GH_AW_ALLOWED_REASONS);
      }));
  }));
