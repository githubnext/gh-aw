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

    it("should use explicit item_number even with triggering target", async () => {
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

      // Execute the handler factory with target: "triggering" (default)
      const handler = await eval(`(async () => { ${addCommentScript}; return await main({ target: 'triggering' }); })()`);

      const message = {
        type: "add_comment",
        item_number: 777,
        body: "Test comment with explicit item_number",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(true);
      expect(capturedIssueNumber).toBe(777);
      expect(result.itemNumber).toBe(777);
    });

    it("should resolve from context when item_number is not provided", async () => {
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

      // Execute the handler factory with target: "triggering" (default)
      const handler = await eval(`(async () => { ${addCommentScript}; return await main({ target: 'triggering' }); })()`);

      const message = {
        type: "add_comment",
        body: "Test comment without item_number, should use PR from context",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(true);
      expect(capturedIssueNumber).toBe(8535); // Should use PR number from mockContext
      expect(result.itemNumber).toBe(8535);
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

  describe("append-only-comments integration", () => {
    it("should not hide older comments when append-only-comments is enabled", async () => {
      const addCommentScript = fs.readFileSync(path.join(__dirname, "add_comment.cjs"), "utf8");

      // Set up environment variable for append-only-comments
      process.env.GH_AW_SAFE_OUTPUT_MESSAGES = JSON.stringify({
        appendOnlyComments: true,
      });
      process.env.GITHUB_WORKFLOW = "test-workflow";

      let hideCommentsWasCalled = false;
      let listCommentsCalls = 0;

      mockGithub.rest.issues.listComments = async () => {
        listCommentsCalls++;
        return {
          data: [
            {
              id: 999,
              node_id: "IC_kwDOTest999",
              body: "Old comment <!-- gh-aw-workflow-id: test-workflow -->",
            },
          ],
        };
      };

      mockGithub.graphql = async (query, variables) => {
        if (query.includes("minimizeComment")) {
          hideCommentsWasCalled = true;
        }
        return {
          minimizeComment: {
            minimizedComment: {
              isMinimized: true,
            },
          },
        };
      };

      let capturedComment = null;
      mockGithub.rest.issues.createComment = async params => {
        capturedComment = params;
        return {
          data: {
            id: 12345,
            html_url: `https://github.com/owner/repo/issues/${params.issue_number}#issuecomment-12345`,
          },
        };
      };

      // Execute with hide-older-comments enabled
      const handler = await eval(`(async () => { ${addCommentScript}; return await main({ hide_older_comments: true }); })()`);

      const message = {
        type: "add_comment",
        body: "New comment - should not hide old ones",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(true);
      expect(hideCommentsWasCalled).toBe(false);
      expect(listCommentsCalls).toBe(0);
      expect(capturedComment).toBeTruthy();
      expect(capturedComment.body).toContain("New comment - should not hide old ones");

      // Clean up
      delete process.env.GH_AW_SAFE_OUTPUT_MESSAGES;
      delete process.env.GITHUB_WORKFLOW;
    });

    it("should hide older comments when append-only-comments is not enabled", async () => {
      const addCommentScript = fs.readFileSync(path.join(__dirname, "add_comment.cjs"), "utf8");

      // Set up environment variable WITHOUT append-only-comments
      delete process.env.GH_AW_SAFE_OUTPUT_MESSAGES;
      process.env.GITHUB_WORKFLOW = "test-workflow";

      let hideCommentsWasCalled = false;
      let listCommentsCalls = 0;

      mockGithub.rest.issues.listComments = async () => {
        listCommentsCalls++;
        return {
          data: [
            {
              id: 999,
              node_id: "IC_kwDOTest999",
              body: "Old comment <!-- gh-aw-workflow-id: test-workflow -->",
            },
          ],
        };
      };

      mockGithub.graphql = async (query, variables) => {
        if (query.includes("minimizeComment")) {
          hideCommentsWasCalled = true;
        }
        return {
          minimizeComment: {
            minimizedComment: {
              isMinimized: true,
            },
          },
        };
      };

      let capturedComment = null;
      mockGithub.rest.issues.createComment = async params => {
        capturedComment = params;
        return {
          data: {
            id: 12345,
            html_url: `https://github.com/owner/repo/issues/${params.issue_number}#issuecomment-12345`,
          },
        };
      };

      // Execute with hide-older-comments enabled
      const handler = await eval(`(async () => { ${addCommentScript}; return await main({ hide_older_comments: true }); })()`);

      const message = {
        type: "add_comment",
        body: "New comment - should hide old ones",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(true);
      expect(hideCommentsWasCalled).toBe(true);
      expect(listCommentsCalls).toBeGreaterThan(0);
      expect(capturedComment).toBeTruthy();
      expect(capturedComment.body).toContain("New comment - should hide old ones");

      // Clean up
      delete process.env.GITHUB_WORKFLOW;
    });

    it("should include append-only marker in comment body when append-only-comments is enabled", async () => {
      const addCommentScript = fs.readFileSync(path.join(__dirname, "add_comment.cjs"), "utf8");

      // Set up environment variable for append-only-comments
      process.env.GH_AW_SAFE_OUTPUT_MESSAGES = JSON.stringify({
        appendOnlyComments: true,
      });
      process.env.GITHUB_WORKFLOW = "test-workflow";

      let capturedComment = null;
      mockGithub.rest.issues.createComment = async params => {
        capturedComment = params;
        return {
          data: {
            id: 12345,
            html_url: `https://github.com/owner/repo/issues/${params.issue_number}#issuecomment-12345`,
          },
        };
      };

      const handler = await eval(`(async () => { ${addCommentScript}; return await main({}); })()`);

      const message = {
        type: "add_comment",
        body: "Append-only comment",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(true);
      expect(capturedComment).toBeTruthy();
      expect(capturedComment.body).toContain("<!-- gh-aw-comment-type: append-only -->");

      // Clean up
      delete process.env.GH_AW_SAFE_OUTPUT_MESSAGES;
      delete process.env.GITHUB_WORKFLOW;
    });

    it("should not hide append-only comments from previous runs", async () => {
      const addCommentScript = fs.readFileSync(path.join(__dirname, "add_comment.cjs"), "utf8");

      // Set up environment WITHOUT append-only-comments (new run)
      delete process.env.GH_AW_SAFE_OUTPUT_MESSAGES;
      process.env.GITHUB_WORKFLOW = "test-workflow";

      let hideCommentsWasCalled = false;
      let hiddenCommentIds = [];

      // Simulate finding an old append-only comment and a regular comment
      mockGithub.rest.issues.listComments = async () => {
        return {
          data: [
            {
              id: 888,
              node_id: "IC_kwDOTest888",
              body: "Old append-only comment <!-- gh-aw-workflow-id: test-workflow --><!-- gh-aw-comment-type: append-only -->",
            },
            {
              id: 999,
              node_id: "IC_kwDOTest999",
              body: "Old regular comment <!-- gh-aw-workflow-id: test-workflow -->",
            },
          ],
        };
      };

      mockGithub.graphql = async (query, variables) => {
        if (query.includes("minimizeComment")) {
          hideCommentsWasCalled = true;
          hiddenCommentIds.push(variables.nodeId);
        }
        return {
          minimizeComment: {
            minimizedComment: {
              isMinimized: true,
            },
          },
        };
      };

      mockGithub.rest.issues.createComment = async params => {
        return {
          data: {
            id: 12345,
            html_url: `https://github.com/owner/repo/issues/${params.issue_number}#issuecomment-12345`,
          },
        };
      };

      // Execute with hide-older-comments enabled (but not append-only)
      const handler = await eval(`(async () => { ${addCommentScript}; return await main({ hide_older_comments: true }); })()`);

      const message = {
        type: "add_comment",
        body: "New comment",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(true);
      expect(hideCommentsWasCalled).toBe(true);
      // Should only hide the regular comment, not the append-only comment
      expect(hiddenCommentIds).toEqual(["IC_kwDOTest999"]);
      expect(hiddenCommentIds).not.toContain("IC_kwDOTest888");

      // Clean up
      delete process.env.GITHUB_WORKFLOW;
    });
  });
});
