// @ts-check
const { describe, it, beforeEach } = require("node:test");
const assert = require("node:assert");

describe("add_comment", () => {
  let originalEnv;
  let github;
  let core;
  let context;

  beforeEach(() => {
    // Save original environment
    originalEnv = { ...process.env };

    // Mock GitHub API
    github = {
      rest: {
        issues: {
          listComments: async ({ page = 1, per_page = 100 }) => {
            if (page > 1) {
              return { data: [] };
            }
            return {
              data: [
                {
                  id: 1,
                  node_id: "MDEyOklzc3VlQ29tbWVudDE=",
                  body: "Regular comment",
                },
                {
                  id: 2,
                  node_id: "MDEyOklzc3VlQ29tbWVudDI=",
                  body: "<!-- workflow-id: test-workflow -->Previous comment",
                },
                {
                  id: 3,
                  node_id: "MDEyOklzc3VlQ29tbWVudDM=",
                  body: "<!-- workflow-id: test-workflow --><!-- comment-type: reaction -->Reaction",
                },
              ],
            };
          },
          createComment: async ({ body }) => ({
            data: {
              id: 100,
              html_url: "https://github.com/owner/repo/issues/1#issuecomment-100",
              body,
            },
          }),
        },
      },
      graphql: async (query, variables) => {
        if (query.includes("minimizeComment")) {
          return {
            minimizeComment: {
              minimizedComment: { isMinimized: true },
            },
          };
        }

        if (query.includes("discussion(number:")) {
          if (query.includes("mutation")) {
            return {
              addDiscussionComment: {
                comment: {
                  id: "DC_kwDOABCDEF4",
                  body: variables.body,
                  createdAt: "2025-01-01T00:00:00Z",
                  url: "https://github.com/owner/repo/discussions/1#discussioncomment-1",
                },
              },
            };
          }
          return {
            repository: {
              discussion: {
                id: "D_kwDOABCDEF4",
                url: "https://github.com/owner/repo/discussions/1",
                comments: {
                  nodes: [
                    {
                      id: "DC_kwDOABCDEF1",
                      body: "Regular discussion comment",
                    },
                    {
                      id: "DC_kwDOABCDEF2",
                      body: "<!-- workflow-id: test-workflow -->Previous discussion comment",
                    },
                  ],
                  pageInfo: {
                    hasNextPage: false,
                    endCursor: null,
                  },
                },
              },
            },
          };
        }

        return { repository: null };
      },
    };

    // Mock core API
    core = {
      info: () => {},
      warning: () => {},
      error: () => {},
      setFailed: () => {},
      setOutput: () => {},
      summary: {
        addRaw: () => ({ write: async () => {} }),
      },
    };

    // Mock context
    context = {
      repo: { owner: "owner", repo: "repo" },
      runId: 123456789,
      eventName: "issues",
      payload: {
        issue: { number: 1 },
      },
    };

    // Set globals for github-script context
    global.github = github;
    global.core = core;
    global.context = context;

    // Set environment variables
    process.env.GITHUB_WORKFLOW = "test-workflow";
    process.env.GH_AW_WORKFLOW_NAME = "Test Workflow";
  });

  // Clean up after each test
  const afterEach = () => {
    process.env = originalEnv;
    delete global.github;
    delete global.core;
    delete global.context;
  };

  it("should create a handler function", async () => {
    const { main } = require("./add_comment.cjs");
    const handler = await main({});

    assert.strictEqual(typeof handler, "function");
    afterEach();
  });

  it("should add comment to issue", async () => {
    const { main } = require("./add_comment.cjs");
    const handler = await main({});

    const result = await handler(
      { body: "Test comment" },
      {}
    );

    assert.strictEqual(result.success, true);
    assert.strictEqual(result.itemNumber, 1);
    assert.strictEqual(result.isDiscussion, false);
    assert.strictEqual(typeof result.commentId, "number");
    assert.ok(result.url);
    afterEach();
  });

  it("should add comment with temporary ID replacement", async () => {
    const { main } = require("./add_comment.cjs");
    const handler = await main({});

    const result = await handler(
      { body: "Related to TEMP-ID-123" },
      { "TEMP-ID-123": { type: "issue", number: 42 } }
    );

    assert.strictEqual(result.success, true);
    afterEach();
  });

  it("should respect max count limit", async () => {
    const { main } = require("./add_comment.cjs");
    const handler = await main({ max: 1 });

    // First comment should succeed
    const result1 = await handler({ body: "First comment" }, {});
    assert.strictEqual(result1.success, true);

    // Second comment should fail due to max limit
    const result2 = await handler({ body: "Second comment" }, {});
    assert.strictEqual(result2.success, false);
    assert.ok(result2.error.includes("Max count"));
    afterEach();
  });

  it("should handle invalid item_number", async () => {
    const { main } = require("./add_comment.cjs");
    const handler = await main({});

    const result = await handler(
      { item_number: "invalid", body: "Test" },
      {}
    );

    assert.strictEqual(result.success, false);
    assert.ok(result.error.includes("Invalid item number"));
    afterEach();
  });

  it("should handle missing context when no item_number provided", async () => {
    const { main } = require("./add_comment.cjs");
    context.payload = {}; // Remove issue context

    const handler = await main({});
    const result = await handler({ body: "Test" }, {});

    assert.strictEqual(result.success, false);
    assert.ok(result.error.includes("No target number"));
    afterEach();
  });

  it("should use explicit item_number when provided", async () => {
    const { main } = require("./add_comment.cjs");
    const handler = await main({});

    const result = await handler(
      { item_number: 5, body: "Test comment" },
      {}
    );

    assert.strictEqual(result.success, true);
    assert.strictEqual(result.itemNumber, 5);
    afterEach();
  });

  it("should handle discussion comments", async () => {
    const { main } = require("./add_comment.cjs");
    context.eventName = "discussion";
    context.payload = { discussion: { number: 1 } };

    const handler = await main({});
    const result = await handler({ body: "Discussion comment" }, {});

    assert.strictEqual(result.success, true);
    assert.strictEqual(result.isDiscussion, true);
    afterEach();
  });

  it("should handle API errors gracefully", async () => {
    const { main } = require("./add_comment.cjs");
    
    // Mock API error
    github.rest.issues.createComment = async () => {
      throw new Error("API Error: Rate limit exceeded");
    };

    const handler = await main({});
    const result = await handler({ body: "Test" }, {});

    assert.strictEqual(result.success, false);
    assert.ok(result.error.includes("Rate limit"));
    afterEach();
  });

  it("should add tracker ID and footer to comment body", async () => {
    const { main } = require("./add_comment.cjs");
    let capturedBody = "";

    github.rest.issues.createComment = async ({ body }) => {
      capturedBody = body;
      return {
        data: {
          id: 100,
          html_url: "https://github.com/owner/repo/issues/1#issuecomment-100",
          body,
        },
      };
    };

    const handler = await main({});
    await handler({ body: "Test comment" }, {});

    assert.ok(capturedBody.includes("Test comment"));
    assert.ok(capturedBody.includes("AI generated by"));
    assert.ok(capturedBody.includes("Test Workflow"));
    afterEach();
  });

  it("should handle hide_older_comments configuration", async () => {
    const { main } = require("./add_comment.cjs");
    let minimizeCalled = false;

    github.graphql = async (query) => {
      if (query.includes("minimizeComment")) {
        minimizeCalled = true;
        return {
          minimizeComment: {
            minimizedComment: { isMinimized: true },
          },
        };
      }
      return { repository: { discussion: null } };
    };

    const handler = await main({ hide_older_comments: true });
    await handler({ body: "New comment" }, {});

    assert.strictEqual(minimizeCalled, true);
    afterEach();
  });
});
