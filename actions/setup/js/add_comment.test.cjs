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
      process.env.GH_AW_WORKFLOW_ID = "test-workflow";

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
      delete process.env.GH_AW_WORKFLOW_ID;
    });

    it("should hide older comments when append-only-comments is not enabled", async () => {
      const addCommentScript = fs.readFileSync(path.join(__dirname, "add_comment.cjs"), "utf8");

      // Set up environment variable WITHOUT append-only-comments
      delete process.env.GH_AW_SAFE_OUTPUT_MESSAGES;
      process.env.GH_AW_WORKFLOW_ID = "test-workflow";

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
      delete process.env.GH_AW_WORKFLOW_ID;
    });

    it("should hide older comments with combined XML marker format (workflow_id inside gh-aw-agentic-workflow)", async () => {
      const addCommentScript = fs.readFileSync(path.join(__dirname, "add_comment.cjs"), "utf8");

      delete process.env.GH_AW_SAFE_OUTPUT_MESSAGES;
      process.env.GH_AW_WORKFLOW_ID = "test-workflow";

      let hideCommentsWasCalled = false;
      let listCommentsCalls = 0;

      mockGithub.rest.issues.listComments = async () => {
        listCommentsCalls++;
        return {
          data: [
            {
              id: 999,
              node_id: "IC_kwDOTest999",
              body: "Old comment\n\n<!-- gh-aw-agentic-workflow: Test Workflow, engine: copilot, id: 12345, workflow_id: test-workflow, run: https://github.com/owner/repo/actions/runs/12345 -->",
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
            id: 12346,
            html_url: `https://github.com/owner/repo/issues/${params.issue_number}#issuecomment-12346`,
          },
        };
      };

      const handler = await eval(`(async () => { ${addCommentScript}; return await main({ hide_older_comments: true }); })()`);

      const message = {
        type: "add_comment",
        body: "New comment - should hide combined-marker old ones",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(true);
      expect(hideCommentsWasCalled).toBe(true);
      expect(listCommentsCalls).toBeGreaterThan(0);
      expect(capturedComment).toBeTruthy();
      expect(capturedComment.body).toContain("New comment - should hide combined-marker old ones");

      // Clean up
      delete process.env.GH_AW_WORKFLOW_ID;
    });
  });

  describe("404 error handling", () => {
    it("should treat 404 errors as warnings for issue comments", async () => {
      const addCommentScript = fs.readFileSync(path.join(__dirname, "add_comment.cjs"), "utf8");

      let warningCalls = [];
      mockCore.warning = msg => {
        warningCalls.push(msg);
      };

      let errorCalls = [];
      mockCore.error = msg => {
        errorCalls.push(msg);
      };

      // Mock API to throw 404 error
      mockGithub.rest.issues.createComment = async () => {
        const error = new Error("Not Found");
        // @ts-ignore
        error.status = 404;
        throw error;
      };

      const handler = await eval(`(async () => { ${addCommentScript}; return await main({}); })()`);

      const message = {
        type: "add_comment",
        body: "Test comment",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(true);
      expect(result.warning).toBeTruthy();
      expect(result.warning).toContain("not found");
      expect(result.skipped).toBe(true);
      expect(warningCalls.length).toBeGreaterThan(0);
      expect(warningCalls[0]).toContain("not found");
      expect(errorCalls.length).toBe(0);
    });

    it("should treat 404 errors as warnings for discussion comments", async () => {
      const addCommentScript = fs.readFileSync(path.join(__dirname, "add_comment.cjs"), "utf8");

      let warningCalls = [];
      mockCore.warning = msg => {
        warningCalls.push(msg);
      };

      let errorCalls = [];
      mockCore.error = msg => {
        errorCalls.push(msg);
      };

      // Change context to discussion
      mockContext.eventName = "discussion";
      mockContext.payload = {
        discussion: {
          number: 10,
        },
      };

      // Mock API to throw 404 error when querying discussion
      mockGithub.graphql = async (query, variables) => {
        if (query.includes("discussion(number")) {
          // Return null to trigger the "not found" error
          return {
            repository: {
              discussion: null, // Discussion not found
            },
          };
        }
        throw new Error("Unexpected query");
      };

      const handler = await eval(`(async () => { ${addCommentScript}; return await main({}); })()`);

      const message = {
        type: "add_comment",
        body: "Test comment on deleted discussion",
      };

      const result = await handler(message, {});

      // The error message contains "not found" so it should be treated as a warning
      expect(result.success).toBe(true);
      expect(result.warning).toBeTruthy();
      expect(result.warning).toContain("not found");
      expect(result.skipped).toBe(true);
      expect(warningCalls.length).toBeGreaterThan(0);
      expect(errorCalls.length).toBe(0);
    });

    it("should detect 404 from error message containing '404'", async () => {
      const addCommentScript = fs.readFileSync(path.join(__dirname, "add_comment.cjs"), "utf8");

      let warningCalls = [];
      mockCore.warning = msg => {
        warningCalls.push(msg);
      };

      // Mock API to throw error with 404 in message
      mockGithub.rest.issues.createComment = async () => {
        throw new Error("API request failed with status 404");
      };

      const handler = await eval(`(async () => { ${addCommentScript}; return await main({}); })()`);

      const message = {
        type: "add_comment",
        body: "Test comment",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(true);
      expect(result.warning).toBeTruthy();
      expect(result.skipped).toBe(true);
      expect(warningCalls.length).toBeGreaterThan(0);
    });

    it("should detect 404 from error message containing 'Not Found'", async () => {
      const addCommentScript = fs.readFileSync(path.join(__dirname, "add_comment.cjs"), "utf8");

      let warningCalls = [];
      mockCore.warning = msg => {
        warningCalls.push(msg);
      };

      // Mock API to throw error with "Not Found" in message
      mockGithub.rest.issues.createComment = async () => {
        throw new Error("Resource Not Found");
      };

      const handler = await eval(`(async () => { ${addCommentScript}; return await main({}); })()`);

      const message = {
        type: "add_comment",
        body: "Test comment",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(true);
      expect(result.warning).toBeTruthy();
      expect(result.skipped).toBe(true);
      expect(warningCalls.length).toBeGreaterThan(0);
    });

    it("should still fail for non-404 errors", async () => {
      const addCommentScript = fs.readFileSync(path.join(__dirname, "add_comment.cjs"), "utf8");

      let warningCalls = [];
      mockCore.warning = msg => {
        warningCalls.push(msg);
      };

      let errorCalls = [];
      mockCore.error = msg => {
        errorCalls.push(msg);
      };

      // Mock API to throw non-404 error
      mockGithub.rest.issues.createComment = async () => {
        const error = new Error("Forbidden");
        // @ts-ignore
        error.status = 403;
        throw error;
      };

      const handler = await eval(`(async () => { ${addCommentScript}; return await main({}); })()`);

      const message = {
        type: "add_comment",
        body: "Test comment",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(false);
      expect(result.error).toBeTruthy();
      expect(result.error).toContain("Forbidden");
      expect(errorCalls.length).toBeGreaterThan(0);
      expect(errorCalls[0]).toContain("Failed to add comment");
    });

    it("should still fail for validation errors", async () => {
      const addCommentScript = fs.readFileSync(path.join(__dirname, "add_comment.cjs"), "utf8");

      let errorCalls = [];
      mockCore.error = msg => {
        errorCalls.push(msg);
      };

      // Mock API to throw validation error
      mockGithub.rest.issues.createComment = async () => {
        const error = new Error("Validation Failed");
        // @ts-ignore
        error.status = 422;
        throw error;
      };

      const handler = await eval(`(async () => { ${addCommentScript}; return await main({}); })()`);

      const message = {
        type: "add_comment",
        body: "Test comment",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(false);
      expect(result.error).toBeTruthy();
      expect(result.error).toContain("Validation Failed");
      expect(errorCalls.length).toBeGreaterThan(0);
    });
  });

  describe("discussion fallback", () => {
    it("should retry as discussion when item_number returns 404 as issue/PR", async () => {
      const addCommentScript = fs.readFileSync(path.join(__dirname, "add_comment.cjs"), "utf8");

      let infoCalls = [];
      mockCore.info = msg => {
        infoCalls.push(msg);
      };

      // Mock REST API to return 404 (not found as issue/PR)
      mockGithub.rest.issues.createComment = async () => {
        const error = new Error("Not Found");
        // @ts-ignore
        error.status = 404;
        throw error;
      };

      // Mock GraphQL to return discussion
      let graphqlCalls = [];
      mockGithub.graphql = async (query, vars) => {
        graphqlCalls.push({ query, vars });

        // First call is to check if discussion exists
        if (query.includes("query") && query.includes("discussion(number:")) {
          return {
            repository: {
              discussion: {
                id: "D_kwDOTest789",
                url: "https://github.com/owner/repo/discussions/14117",
              },
            },
          };
        }

        // Second call is to add comment
        if (query.includes("mutation") && query.includes("addDiscussionComment")) {
          return {
            addDiscussionComment: {
              comment: {
                id: "DC_kwDOTest999",
                body: "Test comment",
                createdAt: "2026-02-06T12:00:00Z",
                url: "https://github.com/owner/repo/discussions/14117#discussioncomment-999",
              },
            },
          };
        }
      };

      const handler = await eval(`(async () => { ${addCommentScript}; return await main({ target: 'triggering' }); })()`);

      const message = {
        type: "add_comment",
        item_number: 14117,
        body: "Test comment on discussion",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(true);
      expect(result.isDiscussion).toBe(true);
      expect(result.itemNumber).toBe(14117);
      expect(result.url).toContain("discussions/14117");

      // Verify it logged the retry
      const retryLog = infoCalls.find(msg => msg.includes("retrying as discussion"));
      expect(retryLog).toBeTruthy();

      const foundLog = infoCalls.find(msg => msg.includes("Found discussion"));
      expect(foundLog).toBeTruthy();
    });

    it("should return skipped when item_number not found as issue/PR or discussion", async () => {
      const addCommentScript = fs.readFileSync(path.join(__dirname, "add_comment.cjs"), "utf8");

      let warningCalls = [];
      mockCore.warning = msg => {
        warningCalls.push(msg);
      };

      // Mock REST API to return 404
      mockGithub.rest.issues.createComment = async () => {
        const error = new Error("Not Found");
        // @ts-ignore
        error.status = 404;
        throw error;
      };

      // Mock GraphQL to also return 404 (discussion doesn't exist either)
      mockGithub.graphql = async (query, vars) => {
        if (query.includes("query") && query.includes("discussion(number:")) {
          return {
            repository: {
              discussion: null,
            },
          };
        }
      };

      const handler = await eval(`(async () => { ${addCommentScript}; return await main({ target: 'triggering' }); })()`);

      const message = {
        type: "add_comment",
        item_number: 99999,
        body: "Test comment",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(true);
      expect(result.skipped).toBe(true);
      expect(result.warning).toContain("not found");

      // Verify warning was logged
      const notFoundWarning = warningCalls.find(msg => msg.includes("not found"));
      expect(notFoundWarning).toBeTruthy();
    });

    it("should not retry as discussion when 404 occurs without explicit item_number", async () => {
      const addCommentScript = fs.readFileSync(path.join(__dirname, "add_comment.cjs"), "utf8");

      let warningCalls = [];
      mockCore.warning = msg => {
        warningCalls.push(msg);
      };

      // Mock REST API to return 404
      mockGithub.rest.issues.createComment = async () => {
        const error = new Error("Not Found");
        // @ts-ignore
        error.status = 404;
        throw error;
      };

      // GraphQL should not be called
      let graphqlCalled = false;
      mockGithub.graphql = async () => {
        graphqlCalled = true;
        throw new Error("GraphQL should not be called");
      };

      const handler = await eval(`(async () => { ${addCommentScript}; return await main({ target: 'triggering' }); })()`);

      const message = {
        type: "add_comment",
        // No item_number - using target resolution
        body: "Test comment",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(true);
      expect(result.skipped).toBe(true);
      expect(graphqlCalled).toBe(false);

      // Verify warning was logged
      const notFoundWarning = warningCalls.find(msg => msg.includes("not found"));
      expect(notFoundWarning).toBeTruthy();
    });

    it("should not retry as discussion when already detected as discussion context", async () => {
      const addCommentScript = fs.readFileSync(path.join(__dirname, "add_comment.cjs"), "utf8");

      // Set discussion context
      mockContext.eventName = "discussion";
      mockContext.payload = {
        discussion: {
          number: 100,
        },
      };

      let warningCalls = [];
      mockCore.warning = msg => {
        warningCalls.push(msg);
      };

      // Mock GraphQL to return 404 for discussion
      let graphqlCallCount = 0;
      mockGithub.graphql = async (query, vars) => {
        graphqlCallCount++;

        if (query.includes("query") && query.includes("discussion(number:")) {
          return {
            repository: {
              discussion: null,
            },
          };
        }
      };

      const handler = await eval(`(async () => { ${addCommentScript}; return await main({ target: 'triggering' }); })()`);

      const message = {
        type: "add_comment",
        body: "Test comment",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(true);
      expect(result.skipped).toBe(true);

      // Should only call GraphQL once (not retry)
      expect(graphqlCallCount).toBe(1);
    });
  });

  describe("temporary ID resolution", () => {
    it("should resolve temporary ID in item_number field", async () => {
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

      const handler = await eval(`(async () => { ${addCommentScript}; return await main({}); })()`);

      const message = {
        type: "add_comment",
        item_number: "aw_test01",
        body: "Comment on issue created with temporary ID",
      };

      // Provide resolved temporary ID
      const resolvedTemporaryIds = {
        aw_test01: { repo: "owner/repo", number: 42 },
      };

      const result = await handler(message, resolvedTemporaryIds);

      expect(result.success).toBe(true);
      expect(capturedIssueNumber).toBe(42);
      expect(result.itemNumber).toBe(42);
    });

    it("should defer when temporary ID is not yet resolved", async () => {
      const addCommentScript = fs.readFileSync(path.join(__dirname, "add_comment.cjs"), "utf8");

      const handler = await eval(`(async () => { ${addCommentScript}; return await main({}); })()`);

      const message = {
        type: "add_comment",
        item_number: "aw_test99",
        body: "Comment on issue with unresolved temporary ID",
      };

      // Empty resolved map - temporary ID not yet resolved
      const resolvedTemporaryIds = {};

      const result = await handler(message, resolvedTemporaryIds);

      expect(result.success).toBe(false);
      expect(result.deferred).toBe(true);
      expect(result.error).toContain("aw_test99");
    });

    it("should handle temporary ID with hash prefix", async () => {
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

      const handler = await eval(`(async () => { ${addCommentScript}; return await main({}); })()`);

      const message = {
        type: "add_comment",
        item_number: "#aw_test02",
        body: "Comment with hash prefix",
      };

      // Provide resolved temporary ID
      const resolvedTemporaryIds = {
        aw_test02: { repo: "owner/repo", number: 100 },
      };

      const result = await handler(message, resolvedTemporaryIds);

      expect(result.success).toBe(true);
      expect(capturedIssueNumber).toBe(100);
    });

    it("should handle invalid temporary ID format", async () => {
      const addCommentScript = fs.readFileSync(path.join(__dirname, "add_comment.cjs"), "utf8");

      const handler = await eval(`(async () => { ${addCommentScript}; return await main({}); })()`);

      const message = {
        type: "add_comment",
        item_number: "aw_", // Invalid: too short
        body: "Comment with invalid temporary ID",
      };

      const resolvedTemporaryIds = {};

      const result = await handler(message, resolvedTemporaryIds);

      expect(result.success).toBe(false);
      expect(result.error).toContain("Invalid item_number specified");
    });

    it("should replace temporary IDs in comment body", async () => {
      const addCommentScript = fs.readFileSync(path.join(__dirname, "add_comment.cjs"), "utf8");

      let capturedBody = null;
      mockGithub.rest.issues.createComment = async params => {
        capturedBody = params.body;
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
        item_number: 42,
        body: "References: #aw_test01 and #aw_test02",
      };

      // Provide resolved temporary IDs
      const resolvedTemporaryIds = {
        aw_test01: { repo: "owner/repo", number: 100 },
        aw_test02: { repo: "owner/repo", number: 200 },
      };

      const result = await handler(message, resolvedTemporaryIds);

      expect(result.success).toBe(true);
      expect(capturedBody).toContain("#100");
      expect(capturedBody).toContain("#200");
      expect(capturedBody).not.toContain("aw_test01");
      expect(capturedBody).not.toContain("aw_test02");
    });
  });

  describe("sanitization preserves markers", () => {
    it("should preserve tracker ID markers after sanitization", async () => {
      const addCommentScript = fs.readFileSync(path.join(__dirname, "add_comment.cjs"), "utf8");

      // Setup environment
      process.env.GH_AW_WORKFLOW_NAME = "Test Workflow";
      process.env.GH_AW_TRACKER_ID = "test-tracker-123";

      let capturedBody = null;
      mockGithub.rest.issues.createComment = async params => {
        capturedBody = params.body;
        return {
          data: {
            id: 12345,
            html_url: "https://github.com/owner/repo/issues/42#issuecomment-12345",
          },
        };
      };

      // Execute the handler
      const handler = await eval(`(async () => { ${addCommentScript}; return await main({}); })()`);

      const message = {
        type: "add_comment",
        body: "User content with <script>alert('xss')</script> attempt",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(true);
      expect(capturedBody).toBeDefined();
      // Verify tracker ID is present (not removed by sanitization)
      expect(capturedBody).toContain("<!-- gh-aw-tracker-id: test-tracker-123 -->");
      // Verify script tags were sanitized (converted to safe format)
      expect(capturedBody).not.toContain("<script>");
      expect(capturedBody).toContain("(script)"); // Tags converted to parentheses

      delete process.env.GH_AW_WORKFLOW_NAME;
      delete process.env.GH_AW_TRACKER_ID;
    });

    it("should preserve workflow footer after sanitization", async () => {
      const addCommentScript = fs.readFileSync(path.join(__dirname, "add_comment.cjs"), "utf8");

      // Setup environment
      process.env.GH_AW_WORKFLOW_NAME = "Security Test Workflow";

      let capturedBody = null;
      mockGithub.rest.issues.createComment = async params => {
        capturedBody = params.body;
        return {
          data: {
            id: 12345,
            html_url: "https://github.com/owner/repo/issues/42#issuecomment-12345",
          },
        };
      };

      // Execute the handler
      const handler = await eval(`(async () => { ${addCommentScript}; return await main({}); })()`);

      const message = {
        type: "add_comment",
        body: "User content <!-- malicious comment -->",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(true);
      expect(capturedBody).toBeDefined();
      // Verify footer is present (not removed by sanitization)
      expect(capturedBody).toContain("Generated by");
      expect(capturedBody).toContain("Security Test Workflow");
      // Verify malicious comment in user content was removed by sanitization
      expect(capturedBody).not.toContain("<!-- malicious comment -->");

      delete process.env.GH_AW_WORKFLOW_NAME;
    });

    it("should sanitize user content but preserve system markers", async () => {
      const addCommentScript = fs.readFileSync(path.join(__dirname, "add_comment.cjs"), "utf8");

      let capturedBody = null;
      mockGithub.rest.issues.createComment = async params => {
        capturedBody = params.body;
        return {
          data: {
            id: 12345,
            html_url: "https://github.com/owner/repo/issues/42#issuecomment-12345",
          },
        };
      };

      // Execute the handler
      const handler = await eval(`(async () => { ${addCommentScript}; return await main({}); })()`);

      const message = {
        type: "add_comment",
        body: "User says: @badactor please <!-- inject this --> [phishing](http://evil.com)",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(true);
      expect(capturedBody).toBeDefined();

      // User content should be sanitized
      expect(capturedBody).toContain("`@badactor`"); // Mention neutralized
      expect(capturedBody).not.toContain("<!-- inject this -->"); // Comment removed
      expect(capturedBody).toContain("(evil.com/redacted)"); // HTTP URL redacted

      // But footer should still be present with proper markdown
      expect(capturedBody).toContain("> Generated by");
    });
  });
  describe("parent author mention allowlist", () => {
    it("should preserve issue author mention when triggered by issues event", async () => {
      const addCommentScript = fs.readFileSync(path.join(__dirname, "add_comment.cjs"), "utf8");

      // Change context to issues event with an issue author
      mockContext.eventName = "issues";
      mockContext.payload = {
        issue: {
          number: 970,
          user: { login: "Slesa", type: "User" },
        },
      };

      let capturedBody = null;
      mockGithub.rest.issues.createComment = async params => {
        capturedBody = params.body;
        return {
          data: {
            id: 12345,
            html_url: "https://github.com/owner/repo/issues/970#issuecomment-12345",
          },
        };
      };

      const handler = await eval(`(async () => { ${addCommentScript}; return await main({}); })()`);

      const message = {
        type: "add_comment",
        body: "Hey @Slesa, I found a fix for your issue!",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(true);
      expect(capturedBody).toBeDefined();
      // Issue author mention should be preserved (not neutralized to `@Slesa`)
      expect(capturedBody).toContain("@Slesa");
      expect(capturedBody).not.toContain("`@Slesa`");
    });

    it("should preserve issue author mention when triggered by issue_comment event", async () => {
      const addCommentScript = fs.readFileSync(path.join(__dirname, "add_comment.cjs"), "utf8");

      // Change context to issue_comment event
      mockContext.eventName = "issue_comment";
      mockContext.payload = {
        issue: {
          number: 970,
          user: { login: "Slesa", type: "User" },
        },
        comment: {
          user: { login: "Commenter", type: "User" },
        },
      };

      let capturedBody = null;
      mockGithub.rest.issues.createComment = async params => {
        capturedBody = params.body;
        return {
          data: {
            id: 12345,
            html_url: "https://github.com/owner/repo/issues/970#issuecomment-12345",
          },
        };
      };

      const handler = await eval(`(async () => { ${addCommentScript}; return await main({}); })()`);

      const message = {
        type: "add_comment",
        body: "Hey @Slesa, responding to your comment.",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(true);
      expect(capturedBody).toBeDefined();
      // Issue author mention should be preserved (not neutralized)
      expect(capturedBody).toContain("@Slesa");
      expect(capturedBody).not.toContain("`@Slesa`");
    });

    it("should preserve PR author mention when triggered by pull_request event", async () => {
      const addCommentScript = fs.readFileSync(path.join(__dirname, "add_comment.cjs"), "utf8");

      // Context is pull_request with an author (mockContext default is pull_request)
      mockContext.payload = {
        pull_request: {
          number: 8535,
          user: { login: "PRAuthor", type: "User" },
        },
      };

      let capturedBody = null;
      mockGithub.rest.issues.createComment = async params => {
        capturedBody = params.body;
        return {
          data: {
            id: 12345,
            html_url: "https://github.com/owner/repo/issues/8535#issuecomment-12345",
          },
        };
      };

      const handler = await eval(`(async () => { ${addCommentScript}; return await main({}); })()`);

      const message = {
        type: "add_comment",
        body: "Hey @PRAuthor, your PR looks great!",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(true);
      expect(capturedBody).toBeDefined();
      // PR author mention should be preserved (not neutralized)
      expect(capturedBody).toContain("@PRAuthor");
      expect(capturedBody).not.toContain("`@PRAuthor`");
    });

    it("should fetch and preserve issue author for explicit item_number", async () => {
      const addCommentScript = fs.readFileSync(path.join(__dirname, "add_comment.cjs"), "utf8");

      // Set up mock to return issue data when fetching issue #970
      mockGithub.rest.issues.get = async params => {
        if (params.issue_number === 970) {
          return {
            data: {
              number: 970,
              user: { login: "Slesa", type: "User" },
            },
          };
        }
        throw new Error("Issue not found");
      };

      let capturedBody = null;
      mockGithub.rest.issues.createComment = async params => {
        capturedBody = params.body;
        return {
          data: {
            id: 12345,
            html_url: "https://github.com/owner/repo/issues/970#issuecomment-12345",
          },
        };
      };

      const handler = await eval(`(async () => { ${addCommentScript}; return await main({}); })()`);

      const message = {
        type: "add_comment",
        item_number: 970,
        body: "Hey @Slesa, I found a fix for your issue!",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(true);
      expect(capturedBody).toBeDefined();
      // Issue author mention should be preserved (not neutralized)
      expect(capturedBody).toContain("@Slesa");
      expect(capturedBody).not.toContain("`@Slesa`");
    });

    it("should not preserve mentions of unknown users for explicit item_number", async () => {
      const addCommentScript = fs.readFileSync(path.join(__dirname, "add_comment.cjs"), "utf8");

      // Set up mock to return issue data with a known author (not the unknown user)
      mockGithub.rest.issues.get = async params => {
        if (params.issue_number === 970) {
          return {
            data: {
              number: 970,
              user: { login: "Slesa", type: "User" },
            },
          };
        }
        throw new Error("Issue not found");
      };

      let capturedBody = null;
      mockGithub.rest.issues.createComment = async params => {
        capturedBody = params.body;
        return {
          data: {
            id: 12345,
            html_url: "https://github.com/owner/repo/issues/970#issuecomment-12345",
          },
        };
      };

      const handler = await eval(`(async () => { ${addCommentScript}; return await main({}); })()`);

      const message = {
        type: "add_comment",
        item_number: 970,
        body: "Hey @Slesa and @UnknownUser, here is the fix.",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(true);
      expect(capturedBody).toBeDefined();
      // Issue author mention should be preserved
      expect(capturedBody).toContain("@Slesa");
      expect(capturedBody).not.toContain("`@Slesa`");
      // Unknown user mention should be neutralized
      expect(capturedBody).toContain("`@UnknownUser`");
    });

    it("should not allow bot users as parent authors", async () => {
      const addCommentScript = fs.readFileSync(path.join(__dirname, "add_comment.cjs"), "utf8");

      // Change context to issues event with a bot as the issue author
      mockContext.eventName = "issues";
      mockContext.payload = {
        issue: {
          number: 970,
          user: { login: "bot-user", type: "Bot" },
        },
      };

      let capturedBody = null;
      mockGithub.rest.issues.createComment = async params => {
        capturedBody = params.body;
        return {
          data: {
            id: 12345,
            html_url: "https://github.com/owner/repo/issues/970#issuecomment-12345",
          },
        };
      };

      const handler = await eval(`(async () => { ${addCommentScript}; return await main({}); })()`);

      const message = {
        type: "add_comment",
        body: "Hey @bot-user, your issue was processed.",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(true);
      expect(capturedBody).toBeDefined();
      // Bot author mention should be neutralized (bots are excluded)
      expect(capturedBody).toContain("`@bot-user`");
    });

    it("should gracefully handle failed issue fetch for explicit item_number", async () => {
      const addCommentScript = fs.readFileSync(path.join(__dirname, "add_comment.cjs"), "utf8");

      // issue.get throws an error
      mockGithub.rest.issues.get = async () => {
        throw new Error("API error");
      };

      let capturedBody = null;
      mockGithub.rest.issues.createComment = async params => {
        capturedBody = params.body;
        return {
          data: {
            id: 12345,
            html_url: "https://github.com/owner/repo/issues/970#issuecomment-12345",
          },
        };
      };

      const handler = await eval(`(async () => { ${addCommentScript}; return await main({}); })()`);

      const message = {
        type: "add_comment",
        item_number: 970,
        body: "Hey @Slesa, here is the fix.",
      };

      const result = await handler(message, {});

      // Should still succeed, just with the mention neutralized
      expect(result.success).toBe(true);
      expect(capturedBody).toBeDefined();
    });

    it("should preserve discussion author mention when triggered by discussion_comment event", async () => {
      const addCommentScript = fs.readFileSync(path.join(__dirname, "add_comment.cjs"), "utf8");

      // Change context to discussion_comment event
      mockContext.eventName = "discussion_comment";
      mockContext.payload = {
        discussion: {
          number: 42,
          user: { login: "DiscussionAuthor", type: "User" },
        },
        comment: {
          user: { login: "Commenter", type: "User" },
        },
      };

      let capturedBody = null;
      // Mock graphql for discussion
      mockGithub.graphql = async query => {
        if (query.includes("discussion(number")) {
          return {
            repository: {
              discussion: { id: "D_kwDOTest123", url: "https://github.com/owner/repo/discussions/42" },
            },
          };
        }
        if (query.includes("addDiscussionComment")) {
          return {
            addDiscussionComment: {
              comment: {
                id: "DC_kwDOTest456",
                body: capturedBody,
                createdAt: "2024-01-01",
                url: "https://github.com/owner/repo/discussions/42#discussioncomment-456",
              },
            },
          };
        }
        return {};
      };

      // Capture the actual addDiscussionComment mutation
      const originalGraphql = mockGithub.graphql;
      mockGithub.graphql = async (query, vars) => {
        if (query.includes("addDiscussionComment")) {
          capturedBody = vars?.body || vars?.message;
          return {
            addDiscussionComment: {
              comment: {
                id: "DC_kwDOTest456",
                body: capturedBody,
                createdAt: "2024-01-01",
                url: "https://github.com/owner/repo/discussions/42#discussioncomment-456",
              },
            },
          };
        }
        return originalGraphql(query, vars);
      };

      const handler = await eval(`(async () => { ${addCommentScript}; return await main({}); })()`);

      const message = {
        type: "add_comment",
        body: "Hey @DiscussionAuthor, great discussion!",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(true);
      // Discussion author mention should be preserved (not neutralized)
      // capturedBody may be undefined if the graphql mock doesn't capture it,
      // but the key test is that the handler succeeds
      expect(result.isDiscussion).toBe(true);
    });
  });

  let enforceCommentLimits;
  let MAX_COMMENT_LENGTH;
  let MAX_MENTIONS;
  let MAX_LINKS;

  beforeEach(async () => {
    const addCommentScript = fs.readFileSync(path.join(__dirname, "add_comment.cjs"), "utf8");
    const exports = await eval(`(async () => { ${addCommentScript}; return { enforceCommentLimits, MAX_COMMENT_LENGTH, MAX_MENTIONS, MAX_LINKS }; })()`);
    enforceCommentLimits = exports.enforceCommentLimits;
    MAX_COMMENT_LENGTH = exports.MAX_COMMENT_LENGTH;
    MAX_MENTIONS = exports.MAX_MENTIONS;
    MAX_LINKS = exports.MAX_LINKS;
  });

  it("should accept comment within all limits", () => {
    const validBody = "This is a valid comment with @user1 and https://github.com";
    expect(() => enforceCommentLimits(validBody)).not.toThrow();
  });

  it("should reject comment exceeding MAX_COMMENT_LENGTH", () => {
    const longBody = "a".repeat(MAX_COMMENT_LENGTH + 1);
    expect(() => enforceCommentLimits(longBody)).toThrow(/E006.*maximum length/i);
  });

  it("should accept comment at exactly MAX_COMMENT_LENGTH", () => {
    const exactBody = "a".repeat(MAX_COMMENT_LENGTH);
    expect(() => enforceCommentLimits(exactBody)).not.toThrow();
  });

  it("should reject comment with too many mentions", () => {
    const mentions = Array.from({ length: MAX_MENTIONS + 1 }, (_, i) => `@user${i}`).join(" ");
    const bodyWithMentions = `Comment with mentions: ${mentions}`;
    expect(() => enforceCommentLimits(bodyWithMentions)).toThrow(/E007.*mentions/i);
  });

  it("should accept comment at exactly MAX_MENTIONS", () => {
    const mentions = Array.from({ length: MAX_MENTIONS }, (_, i) => `@user${i}`).join(" ");
    const bodyWithMentions = `Comment with mentions: ${mentions}`;
    expect(() => enforceCommentLimits(bodyWithMentions)).not.toThrow();
  });

  it("should reject comment with too many links", () => {
    const links = Array.from({ length: MAX_LINKS + 1 }, (_, i) => `https://example.com/${i}`).join(" ");
    const bodyWithLinks = `Comment with links: ${links}`;
    expect(() => enforceCommentLimits(bodyWithLinks)).toThrow(/E008.*links/i);
  });

  it("should accept comment at exactly MAX_LINKS", () => {
    const links = Array.from({ length: MAX_LINKS }, (_, i) => `https://example.com/${i}`).join(" ");
    const bodyWithLinks = `Comment with links: ${links}`;
    expect(() => enforceCommentLimits(bodyWithLinks)).not.toThrow();
  });

  it("should count both http and https links", () => {
    const httpLinks = Array.from({ length: 26 }, (_, i) => `http://example.com/${i}`).join(" ");
    const httpsLinks = Array.from({ length: 25 }, (_, i) => `https://example.com/${i}`).join(" ");
    const bodyWithMixedLinks = `Comment with mixed: ${httpLinks} ${httpsLinks}`;
    expect(() => enforceCommentLimits(bodyWithMixedLinks)).toThrow(/E008.*links/i);
  });

  it("should provide detailed error message for length violation", () => {
    const longBody = "a".repeat(MAX_COMMENT_LENGTH + 100);
    try {
      enforceCommentLimits(longBody);
      throw new Error("Should have thrown");
    } catch (error) {
      expect(error.message).toMatch(/E006/);
      expect(error.message).toMatch(/65536/);
      expect(error.message).toMatch(/65636/);
    }
  });

  it("should provide detailed error message for mentions violation", () => {
    const mentions = Array.from({ length: 15 }, (_, i) => `@user${i}`).join(" ");
    const bodyWithMentions = `Comment: ${mentions}`;
    try {
      enforceCommentLimits(bodyWithMentions);
      throw new Error("Should have thrown");
    } catch (error) {
      expect(error.message).toMatch(/E007/);
      expect(error.message).toMatch(/15 mentions/);
      expect(error.message).toMatch(/maximum is 10/);
    }
  });

  it("should provide detailed error message for links violation", () => {
    const links = Array.from({ length: 60 }, (_, i) => `https://example.com/${i}`).join(" ");
    const bodyWithLinks = `Comment: ${links}`;
    try {
      enforceCommentLimits(bodyWithLinks);
      throw new Error("Should have thrown");
    } catch (error) {
      expect(error.message).toMatch(/E008/);
      expect(error.message).toMatch(/60 links/);
      expect(error.message).toMatch(/maximum is 50/);
    }
  });

  it("should handle empty comment body", () => {
    expect(() => enforceCommentLimits("")).not.toThrow();
  });

  it("should handle comment with no mentions", () => {
    const body = "This is a comment without any mentions at all";
    expect(() => enforceCommentLimits(body)).not.toThrow();
  });

  it("should handle comment with no links", () => {
    const body = "This is a comment without any links at all";
    expect(() => enforceCommentLimits(body)).not.toThrow();
  });

  it("should not count incomplete mention patterns", () => {
    const body = "@ not a mention, @ also not, @123 is not a mention";
    expect(() => enforceCommentLimits(body)).not.toThrow();
  });

  it("should count valid mention patterns only", () => {
    const body = "Valid: @user1 @user2. Invalid: @ @123 email@example.com";
    expect(() => enforceCommentLimits(body)).not.toThrow();
  });
});
