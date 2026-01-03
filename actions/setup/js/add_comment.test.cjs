// @ts-check
import { describe, it, expect, beforeEach, afterEach } from "vitest";
import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

describe("add_comment", () => {
  let mockCore;
  let mockGithub;
  let mockContext;
  let originalGlobals;

  beforeEach(() => {
    // Save original globals
    originalGlobals = {
      core: global.core,
      github: global.github,
      context: global.context,
    };

    // Setup mock core
    mockCore = {
      info: () => {},
      warning: () => {},
      error: () => {},
      setOutput: () => {},
      setFailed: () => {},
    };

    // Setup mock github API
    mockGithub = {
      rest: {
        issues: {
          createComment: async () => ({
            data: {
              id: 12345,
              html_url: "https://github.com/owner/repo/issues/42#issuecomment-12345",
            },
          }),
          listComments: async () => ({ data: [] }),
        },
      },
      graphql: async () => ({
        repository: {
          discussion: {
            id: "D_kwDOTest123",
            url: "https://github.com/owner/repo/discussions/10",
          },
        },
        addDiscussionComment: {
          comment: {
            id: "DC_kwDOTest456",
            url: "https://github.com/owner/repo/discussions/10#discussioncomment-456",
          },
        },
      }),
    };

    // Setup mock context
    mockContext = {
      eventName: "pull_request",
      runId: 12345,
      repo: {
        owner: "owner",
        repo: "repo",
      },
      payload: {
        pull_request: {
          number: 8535, // The correct PR that triggered the workflow
        },
      },
    };

    // Set globals
    global.core = mockCore;
    global.github = mockGithub;
    global.context = mockContext;
  });

  afterEach(() => {
    // Restore original globals
    global.core = originalGlobals.core;
    global.github = originalGlobals.github;
    global.context = originalGlobals.context;
  });

  describe("target configuration", () => {
    it("should use triggering PR context when target is 'triggering'", async () => {
      const addCommentScript = fs.readFileSync(path.join(__dirname, "add_comment.cjs"), "utf8");

      let capturedIssueNumber = null;
      mockGithub.rest.issues.createComment = async params => {
        capturedIssueNumber = params.issue_number;
        return {
          data: {
            id: 12345,
            html_url: `https://github.com/owner/repo/issues/${params.issue_number}#issuecomment-12345`,
          },
        };
      };

      // Execute the handler factory with target: "triggering"
      const handler = await eval(`(async () => { ${addCommentScript}; return await main({ target: 'triggering' }); })()`);

      const message = {
        type: "add_comment",
        body: "Test comment on triggering PR",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(true);
      expect(capturedIssueNumber).toBe(8535);
      expect(result.itemNumber).toBe(8535);
    });

    it("should use explicit PR number when target is a number", async () => {
      const addCommentScript = fs.readFileSync(path.join(__dirname, "add_comment.cjs"), "utf8");

      let capturedIssueNumber = null;
      mockGithub.rest.issues.createComment = async params => {
        capturedIssueNumber = params.issue_number;
        return {
          data: {
            id: 12345,
            html_url: `https://github.com/owner/repo/issues/${params.issue_number}#issuecomment-12345`,
          },
        };
      };

      // Execute the handler factory with target: 21 (explicit PR number)
      const handler = await eval(`(async () => { ${addCommentScript}; return await main({ target: '21' }); })()`);

      const message = {
        type: "add_comment",
        body: "Test comment on explicit PR",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(true);
      expect(capturedIssueNumber).toBe(21);
      expect(result.itemNumber).toBe(21);
    });

    it("should use item_number from message when target is '*'", async () => {
      const addCommentScript = fs.readFileSync(path.join(__dirname, "add_comment.cjs"), "utf8");

      let capturedIssueNumber = null;
      mockGithub.rest.issues.createComment = async params => {
        capturedIssueNumber = params.issue_number;
        return {
          data: {
            id: 12345,
            html_url: `https://github.com/owner/repo/issues/${params.issue_number}#issuecomment-12345`,
          },
        };
      };

      // Execute the handler factory with target: "*"
      const handler = await eval(`(async () => { ${addCommentScript}; return await main({ target: '*' }); })()`);

      const message = {
        type: "add_comment",
        item_number: 999,
        body: "Test comment on item_number PR",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(true);
      expect(capturedIssueNumber).toBe(999);
      expect(result.itemNumber).toBe(999);
    });

    it("should fail when target is '*' but no item_number provided", async () => {
      const addCommentScript = fs.readFileSync(path.join(__dirname, "add_comment.cjs"), "utf8");

      const handler = await eval(`(async () => { ${addCommentScript}; return await main({ target: '*' }); })()`);

      const message = {
        type: "add_comment",
        body: "Test comment without item_number",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(false);
      expect(result.error).toMatch(/no.*item_number/i);
    });

    it("should use issue context when triggered by an issue", async () => {
      const addCommentScript = fs.readFileSync(path.join(__dirname, "add_comment.cjs"), "utf8");

      // Change context to issue
      mockContext.eventName = "issues";
      mockContext.payload = {
        issue: {
          number: 42,
        },
      };

      let capturedIssueNumber = null;
      mockGithub.rest.issues.createComment = async params => {
        capturedIssueNumber = params.issue_number;
        return {
          data: {
            id: 12345,
            html_url: `https://github.com/owner/repo/issues/${params.issue_number}#issuecomment-12345`,
          },
        };
      };

      const handler = await eval(`(async () => { ${addCommentScript}; return await main({ target: 'triggering' }); })()`);

      const message = {
        type: "add_comment",
        body: "Test comment on issue",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(true);
      expect(capturedIssueNumber).toBe(42);
      expect(result.itemNumber).toBe(42);
      expect(result.isDiscussion).toBe(false);
    });
  });

  describe("discussion support", () => {
    it("should use discussion context when triggered by a discussion", async () => {
      const addCommentScript = fs.readFileSync(path.join(__dirname, "add_comment.cjs"), "utf8");

      // Change context to discussion
      mockContext.eventName = "discussion";
      mockContext.payload = {
        discussion: {
          number: 10,
        },
      };

      let capturedDiscussionNumber = null;
      let graphqlCallCount = 0;
      mockGithub.graphql = async (query, variables) => {
        graphqlCallCount++;
        if (query.includes("addDiscussionComment")) {
          return {
            addDiscussionComment: {
              comment: {
                id: "DC_kwDOTest456",
                url: "https://github.com/owner/repo/discussions/10#discussioncomment-456",
              },
            },
          };
        }
        // Query for discussion ID
        if (variables.number) {
          capturedDiscussionNumber = variables.number;
        }
        if (variables.num) {
          capturedDiscussionNumber = variables.num;
        }
        return {
          repository: {
            discussion: {
              id: "D_kwDOTest123",
              url: "https://github.com/owner/repo/discussions/10",
            },
          },
        };
      };

      const handler = await eval(`(async () => { ${addCommentScript}; return await main({ target: 'triggering' }); })()`);

      const message = {
        type: "add_comment",
        body: "Test comment on discussion",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(true);
      expect(capturedDiscussionNumber).toBe(10);
      expect(result.itemNumber).toBe(10);
      expect(result.isDiscussion).toBe(true);
    });
  });

  describe("regression test for wrong PR bug", () => {
    it("should NOT comment on a different PR when workflow runs on PR #8535", async () => {
      const addCommentScript = fs.readFileSync(path.join(__dirname, "add_comment.cjs"), "utf8");

      // Simulate the exact scenario from the bug:
      // - Workflow runs on PR #8535 (branch: copilot/enable-sandbox-mcp-gateway)
      // - Should comment on PR #8535, NOT PR #21
      mockContext.eventName = "pull_request";
      mockContext.payload = {
        pull_request: {
          number: 8535,
        },
      };

      let capturedIssueNumber = null;
      mockGithub.rest.issues.createComment = async params => {
        capturedIssueNumber = params.issue_number;
        return {
          data: {
            id: 12345,
            html_url: `https://github.com/owner/repo/issues/${params.issue_number}#issuecomment-12345`,
          },
        };
      };

      // Use default target configuration (should be "triggering")
      const handler = await eval(`(async () => { ${addCommentScript}; return await main({}); })()`);

      const message = {
        type: "add_comment",
        body: "## Smoke Test: Copilot Safe Inputs\n\n✅ Test passed",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(true);
      expect(capturedIssueNumber).toBe(8535);
      expect(result.itemNumber).toBe(8535);
      expect(capturedIssueNumber).not.toBe(21);
    });
  });
});
