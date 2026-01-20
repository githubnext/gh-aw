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
  mockGithub = { rest: { pulls: { createReviewComment: vi.fn() } } },
  mockContext = {
    eventName: "pull_request",
    runId: 12345,
    repo: { owner: "testowner", repo: "testrepo" },
    payload: { pull_request: { number: 123, head: { sha: "abc123def456" } }, repository: { html_url: "https://github.com/testowner/testrepo" } },
  };
((global.core = mockCore),
  (global.github = mockGithub),
  (global.context = mockContext),
  describe("create_pr_review_comment.cjs", () => {
    let createPRReviewCommentScript, tempFilePath;
    const setAgentOutput = data => {
      tempFilePath = path.join("/tmp", `test_agent_output_${Date.now()}_${Math.random().toString(36).slice(2)}.json`);
      const content = "string" == typeof data ? data : JSON.stringify(data);
      (fs.writeFileSync(tempFilePath, content), (process.env.GH_AW_AGENT_OUTPUT = tempFilePath));
    };
    (beforeEach(() => {
      vi.clearAllMocks();
      const scriptPath = path.join(__dirname, "create_pr_review_comment.cjs");
      ((createPRReviewCommentScript = fs.readFileSync(scriptPath, "utf8")),
        delete process.env.GH_AW_AGENT_OUTPUT,
        delete process.env.GH_AW_PR_REVIEW_COMMENT_SIDE,
        delete process.env.GH_AW_PR_REVIEW_COMMENT_TARGET,
        (global.context = mockContext));
    }),
      afterEach(() => {
        tempFilePath && require("fs").existsSync(tempFilePath) && (require("fs").unlinkSync(tempFilePath), (tempFilePath = void 0));
      }),
      it("should create a single PR review comment with basic configuration", async () => {
        mockGithub.rest.pulls.createReviewComment.mockResolvedValue({ data: { id: 456, html_url: "https://github.com/testowner/testrepo/pull/123#discussion_r456" } });
        const message = { type: "create_pull_request_review_comment", path: "src/main.js", line: 10, body: "Consider using const instead of let here." };
        const handler = await eval(`(async () => { ${createPRReviewCommentScript}; return await main({}); })()`);
        const result = await handler(message, {});
        expect(result.success).toBe(true);
        expect(mockGithub.rest.pulls.createReviewComment).toHaveBeenCalledWith({
          owner: "testowner",
          repo: "testrepo",
          pull_number: 123,
          body: expect.stringContaining("Consider using const instead of let here."),
          path: "src/main.js",
          commit_id: "abc123def456",
          line: 10,
          side: "RIGHT",
        });
      }),
      it("should create a multi-line PR review comment", async () => {
        mockGithub.rest.pulls.createReviewComment.mockResolvedValue({ data: { id: 789, html_url: "https://github.com/testowner/testrepo/pull/123#discussion_r789" } });
        const message = { type: "create_pull_request_review_comment", path: "src/utils.js", line: 25, start_line: 20, side: "LEFT", body: "This entire function could be simplified using modern JS features." };
        const handler = await eval(`(async () => { ${createPRReviewCommentScript}; return await main({}); })()`);
        const result = await handler(message, {});
        expect(result.success).toBe(true);
        expect(mockGithub.rest.pulls.createReviewComment).toHaveBeenCalledWith({
          owner: "testowner",
          repo: "testrepo",
          pull_number: 123,
          body: expect.stringContaining("This entire function could be simplified using modern JS features."),
          path: "src/utils.js",
          commit_id: "abc123def456",
          line: 25,
          start_line: 20,
          side: "LEFT",
          start_side: "LEFT",
        });
      }),
      it("should handle multiple review comments", async () => {
        mockGithub.rest.pulls.createReviewComment
          .mockResolvedValueOnce({ data: { id: 111, html_url: "https://github.com/testowner/testrepo/pull/123#discussion_r111" } })
          .mockResolvedValueOnce({ data: { id: 222, html_url: "https://github.com/testowner/testrepo/pull/123#discussion_r222" } });
        const handler = await eval(`(async () => { ${createPRReviewCommentScript}; return await main({}); })()`);
        const message1 = { type: "create_pull_request_review_comment", path: "src/main.js", line: 10, body: "First comment" };
        const message2 = { type: "create_pull_request_review_comment", path: "src/utils.js", line: 25, body: "Second comment" };
        const result1 = await handler(message1, {});
        const result2 = await handler(message2, {});
        expect(result1.success).toBe(true);
        expect(result2.success).toBe(true);
        expect(mockGithub.rest.pulls.createReviewComment).toHaveBeenCalledTimes(2);
      }),
      it("should enforce max count limit", async () => {
        mockGithub.rest.pulls.createReviewComment.mockResolvedValue({ data: { id: 123, html_url: "https://github.com/testowner/testrepo/pull/123#discussion_r123" } });
        const handler = await eval(`(async () => { ${createPRReviewCommentScript}; return await main({ max: 2 }); })()`);
        const message1 = { type: "create_pull_request_review_comment", path: "src/main.js", line: 10, body: "First comment" };
        const message2 = { type: "create_pull_request_review_comment", path: "src/utils.js", line: 20, body: "Second comment" };
        const message3 = { type: "create_pull_request_review_comment", path: "src/test.js", line: 30, body: "Third comment" };

        const result1 = await handler(message1, {});
        const result2 = await handler(message2, {});
        const result3 = await handler(message3, {});

        expect(result1.success).toBe(true);
        expect(result2.success).toBe(true);
        expect(result3.success).toBe(false);
        expect(result3.error).toContain("Max count of 2 reached");
        expect(mockGithub.rest.pulls.createReviewComment).toHaveBeenCalledTimes(2);
      }),
      it("should use configured side from config", async () => {
        mockGithub.rest.pulls.createReviewComment.mockResolvedValue({ data: { id: 333, html_url: "https://github.com/testowner/testrepo/pull/123#discussion_r333" } });
        const handler = await eval(`(async () => { ${createPRReviewCommentScript}; return await main({ side: 'LEFT' }); })()`);
        const message = { type: "create_pull_request_review_comment", path: "src/main.js", line: 10, body: "Comment on left side" };
        const result = await handler(message, {});
        expect(result.success).toBe(true);
        expect(mockGithub.rest.pulls.createReviewComment).toHaveBeenCalledWith(expect.objectContaining({ side: "LEFT" }));
      }),
      it("should skip when not in pull request context", async () => {
        global.context = { ...mockContext, eventName: "issues", payload: { issue: { number: 123 }, repository: mockContext.payload.repository } };
        const handler = await eval(`(async () => { ${createPRReviewCommentScript}; return await main({}); })()`);
        const message = { type: "create_pull_request_review_comment", path: "src/main.js", line: 10, body: "This should not be created" };
        const result = await handler(message, {});
        expect(result.success).toBe(false);
        expect(result.error).toContain("Not in pull request context");
        expect(mockGithub.rest.pulls.createReviewComment).not.toHaveBeenCalled();
      }),
      it("should validate required fields and skip invalid items", async () => {
        const handler = await eval(`(async () => { ${createPRReviewCommentScript}; return await main({}); })()`);

        // Missing path
        const result1 = await handler({ type: "create_pull_request_review_comment", line: 10, body: "Missing path" }, {});
        expect(result1.success).toBe(false);
        expect(result1.error).toContain('Missing required field "path"');

        // Missing line
        const result2 = await handler({ type: "create_pull_request_review_comment", path: "src/main.js", body: "Missing line" }, {});
        expect(result2.success).toBe(false);
        expect(result2.error).toContain('Missing or invalid required field "line"');

        // Missing body
        const result3 = await handler({ type: "create_pull_request_review_comment", path: "src/main.js", line: 10 }, {});
        expect(result3.success).toBe(false);
        expect(result3.error).toContain('Missing or invalid required field "body"');

        expect(mockGithub.rest.pulls.createReviewComment).not.toHaveBeenCalled();
      }),
      it("should validate start_line is not greater than line", async () => {
        const handler = await eval(`(async () => { ${createPRReviewCommentScript}; return await main({}); })()`);
        const message = { type: "create_pull_request_review_comment", path: "src/main.js", line: 10, start_line: 15, body: "Invalid range" };
        const result = await handler(message, {});
        expect(result.success).toBe(false);
        expect(result.error).toContain("Invalid start_line");
        expect(mockGithub.rest.pulls.createReviewComment).not.toHaveBeenCalled();
      }),
      it("should validate side values", async () => {
        const handler = await eval(`(async () => { ${createPRReviewCommentScript}; return await main({}); })()`);
        const message = { type: "create_pull_request_review_comment", path: "src/main.js", line: 10, side: "INVALID_SIDE", body: "Invalid side value" };
        const result = await handler(message, {});
        expect(result.success).toBe(false);
        expect(result.error).toContain("Invalid side value");
        expect(mockGithub.rest.pulls.createReviewComment).not.toHaveBeenCalled();
      }),
      it("should include AI disclaimer in comment body", async () => {
        mockGithub.rest.pulls.createReviewComment.mockResolvedValue({ data: { id: 999, html_url: "https://github.com/testowner/testrepo/pull/123#discussion_r999" } });
        const handler = await eval(`(async () => { ${createPRReviewCommentScript}; return await main({}); })()`);
        const message = { type: "create_pull_request_review_comment", path: "src/main.js", line: 10, body: "Original comment" };
        const result = await handler(message, {});
        expect(result.success).toBe(true);
        expect(mockGithub.rest.pulls.createReviewComment).toHaveBeenCalledWith(expect.objectContaining({ body: expect.stringMatching(/Original comment[\s\S]*AI generated by/) }));
      }),
      it("should respect target configuration for specific PR number", async () => {
        mockGithub.rest.pulls.get = vi.fn().mockResolvedValue({ data: { number: 456, head: { sha: "def456abc789" } } });
        mockGithub.rest.pulls.createReviewComment = vi.fn().mockResolvedValue({ data: { id: 999, html_url: "https://github.com/testowner/testrepo/pull/456#discussion_r999" } });
        const handler = await eval(`(async () => { ${createPRReviewCommentScript}; return await main({ target: '456' }); })()`);
        const message = { type: "create_pull_request_review_comment", path: "src/main.js", line: 10, body: "Review comment on specific PR" };
        const result = await handler(message, {});
        expect(result.success).toBe(true);
        expect(mockGithub.rest.pulls.get).toHaveBeenCalledWith({ owner: "testowner", repo: "testrepo", pull_number: 456 });
        expect(mockGithub.rest.pulls.createReviewComment).toHaveBeenCalledWith(expect.objectContaining({ pull_number: 456, path: "src/main.js", line: 10, commit_id: "def456abc789" }));
      }),
      it('should respect target "*" configuration with pull_request_number in item', async () => {
        mockGithub.rest.pulls.get = vi.fn().mockResolvedValue({ data: { number: 789, head: { sha: "xyz789abc456" } } });
        mockGithub.rest.pulls.createReviewComment = vi.fn().mockResolvedValue({ data: { id: 888, html_url: "https://github.com/testowner/testrepo/pull/789#discussion_r888" } });
        const handler = await eval(`(async () => { ${createPRReviewCommentScript}; return await main({ target: '*' }); })()`);
        const message = { type: "create_pull_request_review_comment", pull_request_number: 789, path: "src/utils.js", line: 20, body: "Review comment on any PR" };
        const result = await handler(message, {});
        expect(result.success).toBe(true);
        expect(mockGithub.rest.pulls.get).toHaveBeenCalledWith({ owner: "testowner", repo: "testrepo", pull_number: 789 });
        expect(mockGithub.rest.pulls.createReviewComment).toHaveBeenCalledWith(expect.objectContaining({ pull_number: 789, path: "src/utils.js", line: 20, commit_id: "xyz789abc456" }));
      }),
      it('should skip item when target is "*" but no pull_request_number specified', async () => {
        const handler = await eval(`(async () => { ${createPRReviewCommentScript}; return await main({ target: '*' }); })()`);
        const message = { type: "create_pull_request_review_comment", path: "src/main.js", line: 10, body: "Review comment without PR number" };
        const result = await handler(message, {});
        expect(result.success).toBe(false);
        expect(result.error).toContain('Target is "*" but no pull_request_number specified');
        expect(mockGithub.rest.pulls.createReviewComment).not.toHaveBeenCalled();
      }),
      it("should skip comment creation when target is triggering but not in PR context", async () => {
        global.context = { eventName: "issues", runId: 12345, repo: { owner: "testowner", repo: "testrepo" }, payload: { issue: { number: 10 }, repository: { html_url: "https://github.com/testowner/testrepo" } } };
        const handler = await eval(`(async () => { ${createPRReviewCommentScript}; return await main({ target: 'triggering' }); })()`);
        const message = { type: "create_pull_request_review_comment", path: "src/main.js", line: 10, body: "This should not be created" };
        const result = await handler(message, {});
        expect(result.success).toBe(false);
        expect(result.error).toContain("Not in pull request context");
        expect(mockGithub.rest.pulls.createReviewComment).not.toHaveBeenCalled();
      }),
      it("should include workflow source in footer when GH_AW_WORKFLOW_SOURCE is provided", async () => {
        process.env.GH_AW_WORKFLOW_NAME = "Test Workflow";
        process.env.GH_AW_WORKFLOW_SOURCE = "githubnext/agentics/workflows/ci-doctor.md@v1.0.0";
        process.env.GH_AW_WORKFLOW_SOURCE_URL = "https://github.com/githubnext/agentics/tree/v1.0.0/workflows/ci-doctor.md";
        process.env.GH_AW_PROMPTS_DIR = path.join(__dirname, "..", "md");
        global.context = {
          eventName: "pull_request",
          runId: 12345,
          repo: { owner: "testowner", repo: "testrepo" },
          payload: { pull_request: { number: 10, head: { sha: "abc123" } }, repository: { html_url: "https://github.com/testowner/testrepo" } },
        };
        const mockComment = { id: 456, html_url: "https://github.com/testowner/testrepo/pull/10#discussion_r456" };
        mockGithub.rest.pulls.createReviewComment.mockResolvedValue({ data: mockComment });
        const handler = await eval(`(async () => { ${createPRReviewCommentScript}; return await main({}); })()`);
        const message = { type: "create_pull_request_review_comment", path: "src/main.js", line: 10, body: "Test review comment with source" };
        const result = await handler(message, {});
        expect(result.success).toBe(true);
        expect(mockGithub.rest.pulls.createReviewComment).toHaveBeenCalled();
        const callArgs = mockGithub.rest.pulls.createReviewComment.mock.calls[0][0];
        expect(callArgs.body).toContain("Test review comment with source");
        expect(callArgs.body).toContain("AI generated by [Test Workflow]");
        expect(callArgs.body).toContain("https://github.com/testowner/testrepo/actions/runs/12345");
        expect(callArgs.body).toContain("gh aw add githubnext/agentics/workflows/ci-doctor.md@v1.0.0");
        expect(callArgs.body).toContain("usage guide");
      }),
      it("should not include workflow source footer when GH_AW_WORKFLOW_SOURCE is not provided", async () => {
        process.env.GH_AW_WORKFLOW_NAME = "Test Workflow";
        delete process.env.GH_AW_WORKFLOW_SOURCE;
        global.context = {
          eventName: "pull_request",
          runId: 12345,
          repo: { owner: "testowner", repo: "testrepo" },
          payload: { pull_request: { number: 10, head: { sha: "abc123" } }, repository: { html_url: "https://github.com/testowner/testrepo" } },
        };
        const mockComment = { id: 457, html_url: "https://github.com/testowner/testrepo/pull/10#discussion_r457" };
        mockGithub.rest.pulls.createReviewComment.mockResolvedValue({ data: mockComment });
        const handler = await eval(`(async () => { ${createPRReviewCommentScript}; return await main({}); })()`);
        const message = { type: "create_pull_request_review_comment", path: "src/main.js", line: 10, body: "Test review comment without source" };
        const result = await handler(message, {});
        expect(result.success).toBe(true);
        expect(mockGithub.rest.pulls.createReviewComment).toHaveBeenCalled();
        const callArgs = mockGithub.rest.pulls.createReviewComment.mock.calls[0][0];
        expect(callArgs.body).toContain("Test review comment without source");
        expect(callArgs.body).toContain("AI generated by [Test Workflow]");
        expect(callArgs.body).not.toContain("gh aw add");
        expect(callArgs.body).not.toContain("usage guide");
      }),
      it("should include triggering PR number in footer when in PR context", async () => {
        process.env.GH_AW_WORKFLOW_NAME = "Test Workflow";
        global.context.eventName = "pull_request";
        global.context.payload.pull_request = { number: 123, head: { sha: "abc123" } };
        const mockComment = { id: 999, html_url: "https://github.com/testowner/testrepo/pull/123#discussion_r999" };
        mockGithub.rest.pulls.createReviewComment.mockResolvedValue({ data: mockComment });
        const handler = await eval(`(async () => { ${createPRReviewCommentScript}; return await main({}); })()`);
        const message = { type: "create_pull_request_review_comment", body: "Review comment from PR context", path: "test.js", line: 10 };
        const result = await handler(message, {});
        expect(result.success).toBe(true);
        const callArgs = mockGithub.rest.pulls.createReviewComment.mock.calls[0][0];
        expect(callArgs.body).toContain("Review comment from PR context");
        expect(callArgs.body).toContain("AI generated by [Test Workflow]");
        expect(callArgs.body).toContain("for #123");
      }));
  }));
