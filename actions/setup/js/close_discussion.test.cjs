// @ts-check
import { describe, it, expect, beforeEach, afterEach } from "vitest";
const { main } = require("./close_discussion.cjs");

describe("close_discussion", () => {
  let mockCore;
  let mockGithub;
  let mockContext;
  let originalGlobals;
  let originalEnv;

  beforeEach(() => {
    originalGlobals = {
      core: global.core,
      github: global.github,
      context: global.context,
    };
    originalEnv = {
      staged: process.env.GH_AW_SAFE_OUTPUTS_STAGED,
    };

    mockCore = {
      infos: /** @type {string[]} */ [],
      warnings: /** @type {string[]} */ [],
      errors: /** @type {string[]} */ [],
      info: /** @param {string} msg */ msg => mockCore.infos.push(msg),
      warning: /** @param {string} msg */ msg => mockCore.warnings.push(msg),
      error: /** @param {string} msg */ msg => mockCore.errors.push(msg),
      setOutput: () => {},
      setFailed: () => {},
    };

    mockGithub = {
      graphql: async (/** @type {string} */ query, /** @type {any} */ variables) => {
        // Handle different query types
        if (query.includes("closeDiscussion")) {
          return {
            closeDiscussion: {
              discussion: {
                id: "D_kwDOTest123",
                url: "https://github.com/owner/repo/discussions/42",
              },
            },
          };
        }
        if (query.includes("addDiscussionComment")) {
          return {
            addDiscussionComment: {
              comment: {
                id: "DC_kwDOTest456",
                url: "https://github.com/owner/repo/discussions/42#discussioncomment-456",
              },
            },
          };
        }
        // Default: return discussion details query response
        return {
          repository: {
            discussion: {
              id: "D_kwDOTest123",
              title: "Test Discussion",
              closed: false,
              category: { name: "General" },
              url: "https://github.com/owner/repo/discussions/42",
              labels: {
                nodes: [{ name: "help-wanted" }],
                pageInfo: { hasNextPage: false, endCursor: null },
              },
            },
          },
        };
      },
    };

    mockContext = {
      repo: { owner: "test-owner", repo: "test-repo" },
      payload: {
        discussion: { number: 42 },
      },
    };

    global.core = mockCore;
    global.github = mockGithub;
    global.context = mockContext;
    delete process.env.GH_AW_SAFE_OUTPUTS_STAGED;
  });

  afterEach(() => {
    global.core = originalGlobals.core;
    global.github = originalGlobals.github;
    global.context = originalGlobals.context;
    if (originalEnv.staged === undefined) {
      delete process.env.GH_AW_SAFE_OUTPUTS_STAGED;
    } else {
      process.env.GH_AW_SAFE_OUTPUTS_STAGED = originalEnv.staged;
    }
  });

  describe("main factory", () => {
    it("should return a handler function with default config", async () => {
      const handler = await main();
      expect(typeof handler).toBe("function");
    });

    it("should return a handler function with custom config", async () => {
      const handler = await main({ required_labels: ["bug"], max: 5 });
      expect(typeof handler).toBe("function");
    });

    it("should log configuration on initialization", async () => {
      await main({ required_labels: ["bug", "automated"], required_title_prefix: "[bot]", max: 3 });
      expect(mockCore.infos.some(msg => msg.includes("max=3"))).toBe(true);
      expect(mockCore.infos.some(msg => msg.includes("bug, automated"))).toBe(true);
      expect(mockCore.infos.some(msg => msg.includes("[bot]"))).toBe(true);
    });
  });

  describe("handleCloseDiscussion", () => {
    it("should close a discussion using explicit discussion_number", async () => {
      const handler = await main({ max: 10 });
      const closeCalls = /** @type {any[]} */ [];

      const originalGraphql = mockGithub.graphql;
      mockGithub.graphql = async (query, variables) => {
        if (query.includes("closeDiscussion")) {
          closeCalls.push(variables);
          return {
            closeDiscussion: {
              discussion: {
                id: "D_kwDOTest123",
                url: "https://github.com/owner/repo/discussions/99",
              },
            },
          };
        }
        return originalGraphql(query, variables);
      };

      const result = await handler({ discussion_number: 99 }, {});

      expect(result.success).toBe(true);
      expect(result.number).toBe(99);
      expect(closeCalls.length).toBe(1);
    });

    it("should close a discussion from context when discussion_number not provided", async () => {
      const handler = await main({ max: 10 });
      const result = await handler({}, {});

      expect(result.success).toBe(true);
      expect(result.number).toBe(42);
    });

    it("should add a comment when body is provided", async () => {
      const handler = await main({ max: 10 });
      const commentCalls = /** @type {any[]} */ [];

      const originalGraphql = mockGithub.graphql;
      mockGithub.graphql = async (query, variables) => {
        if (query.includes("addDiscussionComment")) {
          commentCalls.push(variables);
        }
        return originalGraphql(query, variables);
      };

      const result = await handler({ body: "Closing this discussion." }, {});

      expect(result.success).toBe(true);
      expect(result.commentUrl).toBeDefined();
      expect(commentCalls.length).toBe(1);
    });

    it("should not add a comment when body is not provided", async () => {
      const handler = await main({ max: 10 });
      const commentCalls = /** @type {any[]} */ [];

      const originalGraphql = mockGithub.graphql;
      mockGithub.graphql = async (query, variables) => {
        if (query.includes("addDiscussionComment")) {
          commentCalls.push(variables);
        }
        return originalGraphql(query, variables);
      };

      const result = await handler({}, {});

      expect(result.success).toBe(true);
      expect(result.commentUrl).toBeUndefined();
      expect(commentCalls.length).toBe(0);
    });

    it("should close with reason when reason is provided", async () => {
      const handler = await main({ max: 10 });
      const closeCalls = /** @type {any[]} */ [];

      const originalGraphql = mockGithub.graphql;
      mockGithub.graphql = async (query, variables) => {
        if (query.includes("closeDiscussion")) {
          closeCalls.push({ query, variables });
          return {
            closeDiscussion: {
              discussion: {
                id: "D_kwDOTest123",
                url: "https://github.com/owner/repo/discussions/42",
              },
            },
          };
        }
        return originalGraphql(query, variables);
      };

      const result = await handler({ reason: "RESOLVED" }, {});

      expect(result.success).toBe(true);
      expect(closeCalls.length).toBe(1);
      expect(closeCalls[0].variables.reason).toBe("RESOLVED");
    });

    it("should return error when no discussion number is available", async () => {
      mockContext.payload = {};
      const handler = await main({ max: 10 });
      const result = await handler({}, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain("No discussion number available");
    });

    it("should return error when discussion_number is invalid", async () => {
      const handler = await main({ max: 10 });
      const result = await handler({ discussion_number: "not-a-number" }, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain("Invalid discussion number");
    });

    it("should enforce max count limit", async () => {
      const handler = await main({ max: 2 });

      await handler({}, {});
      await handler({}, {});
      const result = await handler({}, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain("Max count of 2 reached");
      expect(mockCore.warnings.some(msg => msg.includes("max count of 2 reached"))).toBe(true);
    });

    it("should validate required labels", async () => {
      const handler = await main({ required_labels: ["approved"], max: 10 });
      const result = await handler({}, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain("Missing required labels: approved");
    });

    it("should pass validation when required labels are present", async () => {
      mockGithub.graphql = async (query, variables) => {
        if (query.includes("closeDiscussion") || query.includes("addDiscussionComment")) {
          return {
            closeDiscussion: {
              discussion: { id: "D_kwDOTest123", url: "https://github.com/owner/repo/discussions/42" },
            },
            addDiscussionComment: {
              comment: { id: "DC_1", url: "https://github.com/owner/repo/discussions/42#comment-1" },
            },
          };
        }
        return {
          repository: {
            discussion: {
              id: "D_kwDOTest123",
              title: "Test Discussion",
              closed: false,
              category: { name: "General" },
              url: "https://github.com/owner/repo/discussions/42",
              labels: {
                nodes: [{ name: "approved" }, { name: "help-wanted" }],
                pageInfo: { hasNextPage: false, endCursor: null },
              },
            },
          },
        };
      };

      const handler = await main({ required_labels: ["approved"], max: 10 });
      const result = await handler({}, {});

      expect(result.success).toBe(true);
    });

    it("should validate required title prefix", async () => {
      const handler = await main({ required_title_prefix: "[RFC]", max: 10 });
      const result = await handler({}, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain('Title doesn\'t start with "[RFC]"');
    });

    it("should pass validation when required title prefix matches", async () => {
      mockGithub.graphql = async (query, variables) => {
        if (query.includes("closeDiscussion")) {
          return {
            closeDiscussion: {
              discussion: { id: "D_kwDOTest123", url: "https://github.com/owner/repo/discussions/42" },
            },
          };
        }
        return {
          repository: {
            discussion: {
              id: "D_kwDOTest123",
              title: "[RFC] Test Discussion",
              closed: false,
              category: { name: "General" },
              url: "https://github.com/owner/repo/discussions/42",
              labels: { nodes: [], pageInfo: { hasNextPage: false, endCursor: null } },
            },
          },
        };
      };

      const handler = await main({ required_title_prefix: "[RFC]", max: 10 });
      const result = await handler({}, {});

      expect(result.success).toBe(true);
    });

    it("should handle already-closed discussion gracefully", async () => {
      mockGithub.graphql = async (query, variables) => {
        if (query.includes("closeDiscussion")) {
          throw new Error("Should not be called for already closed discussion");
        }
        if (query.includes("addDiscussionComment")) {
          return {
            addDiscussionComment: {
              comment: { id: "DC_1", url: "https://github.com/owner/repo/discussions/42#comment-1" },
            },
          };
        }
        return {
          repository: {
            discussion: {
              id: "D_kwDOTest123",
              title: "Test Discussion",
              closed: true,
              category: { name: "General" },
              url: "https://github.com/owner/repo/discussions/42",
              labels: { nodes: [], pageInfo: { hasNextPage: false, endCursor: null } },
            },
          },
        };
      };

      const handler = await main({ max: 10 });
      const result = await handler({}, {});

      expect(result.success).toBe(true);
      expect(result.alreadyClosed).toBe(true);
      expect(mockCore.infos.some(msg => msg.includes("already closed"))).toBe(true);
    });

    it("should return staged result when in staged mode", async () => {
      process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true";
      const handler = await main({ max: 10 });
      const result = await handler({ body: "Some comment" }, {});

      expect(result.success).toBe(true);
      expect(result.staged).toBe(true);
      expect(result.previewInfo?.hasComment).toBe(true);
    });

    it("should return error when GraphQL throws an unexpected error", async () => {
      mockGithub.graphql = async () => {
        throw new Error("API rate limit exceeded");
      };

      const handler = await main({ max: 10 });
      const result = await handler({}, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain("rate limit");
      expect(mockCore.errors.some(msg => msg.includes("Failed to close discussion"))).toBe(true);
    });

    it("should handle discussion not found error", async () => {
      mockGithub.graphql = async () => ({
        repository: { discussion: null },
      });

      const handler = await main({ max: 10 });
      const result = await handler({}, {});

      expect(result.success).toBe(false);
    });

    it("should include discussion URL in success result", async () => {
      const handler = await main({ max: 10 });
      const result = await handler({}, {});

      expect(result.success).toBe(true);
      expect(result.url).toBe("https://github.com/owner/repo/discussions/42");
    });
  });
});
