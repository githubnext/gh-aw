// @ts-check
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

const { main } = require("./dispatch_repository.cjs");

describe("dispatch_repository", () => {
  /** @type {any} */
  let mockCore;
  /** @type {any} */
  let mockGithub;
  /** @type {any} */
  let mockContext;

  /** @type {any} */
  let dispatchEventCalls;

  /** @type {{ core: any, github: any, context: any }} */
  let savedGlobals;

  beforeEach(() => {
    savedGlobals = {
      core: global.core,
      github: global.github,
      context: global.context,
    };

    dispatchEventCalls = [];

    mockCore = {
      infos: /** @type {string[]} */ [],
      warnings: /** @type {string[]} */ [],
      errors: /** @type {string[]} */ [],
      info: /** @param {string} msg */ msg => mockCore.infos.push(msg),
      warning: /** @param {string} msg */ msg => mockCore.warnings.push(msg),
      error: /** @param {string} msg */ msg => mockCore.errors.push(msg),
    };

    mockGithub = {
      rest: {
        repos: {
          createDispatchEvent: /** @param {any} params */ async params => {
            dispatchEventCalls.push(params);
            return { data: {} };
          },
        },
      },
    };

    mockContext = {
      runId: 9999,
      repo: { owner: "test-owner", repo: "test-repo" },
      payload: {},
    };

    global.core = mockCore;
    global.github = mockGithub;
    global.context = mockContext;
  });

  afterEach(() => {
    global.core = savedGlobals.core;
    global.github = savedGlobals.github;
    global.context = savedGlobals.context;
    vi.restoreAllMocks();
  });

  describe("main factory", () => {
    it("should return a handler function", async () => {
      const handler = await main({ tools: {} });
      expect(typeof handler).toBe("function");
    });

    it("should return a handler function with no config", async () => {
      const handler = await main();
      expect(typeof handler).toBe("function");
    });

    it("should log initialization info", async () => {
      await main({ tools: { deploy: { event_type: "deploy", max: 2 } } });
      expect(mockCore.infos.some(/** @param {string} m */ m => m.includes("dispatch_repository handler initialized"))).toBe(true);
    });
  });

  describe("handleDispatchRepository", () => {
    it("should return error when tool_name is missing", async () => {
      const handler = await main({ tools: {} });
      const result = await handler({ tool_name: "" }, {});
      expect(result.success).toBe(false);
      expect(result.error).toMatch(/tool_name is required/);
    });

    it("should return error when tool_name is whitespace-only", async () => {
      const handler = await main({ tools: {} });
      const result = await handler({ tool_name: "   " }, {});
      expect(result.success).toBe(false);
      expect(result.error).toMatch(/tool_name is required/);
    });

    it("should return error when tool is not configured", async () => {
      const handler = await main({ tools: {} });
      const result = await handler({ tool_name: "unknown_tool" }, {});
      expect(result.success).toBe(false);
      expect(result.error).toMatch(/not configured/);
    });

    it("should return error when max count is reached", async () => {
      const handler = await main({
        tools: {
          deploy: { event_type: "deploy", repository: "test-owner/test-repo", max: 1 },
        },
      });

      // First dispatch succeeds
      const first = await handler({ tool_name: "deploy" }, {});
      expect(first.success).toBe(true);

      // Second dispatch hits the limit
      const second = await handler({ tool_name: "deploy" }, {});
      expect(second.success).toBe(false);
      expect(second.error).toMatch(/Max count/);
    });

    it("should return error when no target repository is configured", async () => {
      const handler = await main({
        tools: { deploy: { event_type: "deploy" } },
      });
      const result = await handler({ tool_name: "deploy" }, {});
      expect(result.success).toBe(false);
      expect(result.error).toMatch(/No target repository/);
    });

    it("should return error when repository slug is invalid", async () => {
      const handler = await main({
        tools: { deploy: { event_type: "deploy", repository: "not-valid-slug" } },
      });
      const result = await handler({ tool_name: "deploy" }, {});
      expect(result.success).toBe(false);
      expect(result.error).toMatch(/Invalid repository slug/);
    });

    it("should return error when event_type is not configured", async () => {
      const handler = await main({
        tools: { deploy: { repository: "test-owner/test-repo" } },
      });
      const result = await handler({ tool_name: "deploy" }, {});
      expect(result.success).toBe(false);
      expect(result.error).toMatch(/event_type is required/);
    });

    it("should dispatch successfully with valid config", async () => {
      const handler = await main({
        tools: {
          deploy: { event_type: "deploy", repository: "test-owner/other-repo", max: 2 },
        },
      });

      const result = await handler({ tool_name: "deploy" }, {});
      expect(result.success).toBe(true);
      expect(result.tool_name).toBe("deploy");
      expect(result.repository).toBe("test-owner/other-repo");
      expect(result.event_type).toBe("deploy");
      expect(dispatchEventCalls.length).toBe(1);
      expect(dispatchEventCalls[0].event_type).toBe("deploy");
    });

    it("should include message inputs in client_payload", async () => {
      const handler = await main({
        tools: {
          deploy: { event_type: "deploy", repository: "test-owner/test-repo", max: 5 },
        },
      });

      await handler({ tool_name: "deploy", inputs: { env: "production", version: "1.2.3" } }, {});

      expect(dispatchEventCalls.length).toBe(1);
      expect(dispatchEventCalls[0].client_payload.env).toBe("production");
      expect(dispatchEventCalls[0].client_payload.version).toBe("1.2.3");
    });

    it("should inject aw_context into client_payload", async () => {
      const handler = await main({
        tools: {
          deploy: { event_type: "deploy", repository: "test-owner/test-repo", max: 5 },
        },
      });

      await handler({ tool_name: "deploy" }, {});

      expect(dispatchEventCalls.length).toBe(1);
      expect(dispatchEventCalls[0].client_payload).toHaveProperty("aw_context");
    });

    it("should use message.repository over toolConfig.repository when both set", async () => {
      const handler = await main({
        tools: {
          deploy: { event_type: "deploy", repository: "test-owner/default-repo", max: 5 },
        },
      });

      const result = await handler({ tool_name: "deploy", repository: "test-owner/override-repo" }, {});

      expect(result.success).toBe(true);
      expect(result.repository).toBe("test-owner/override-repo");
      expect(dispatchEventCalls[0].owner).toBe("test-owner");
      expect(dispatchEventCalls[0].repo).toBe("override-repo");
    });

    it("should default to first allowed_repository when no target given", async () => {
      const handler = await main({
        tools: {
          deploy: { event_type: "deploy", allowed_repositories: ["test-owner/allowed-repo"], max: 5 },
        },
      });

      const result = await handler({ tool_name: "deploy" }, {});
      expect(result.success).toBe(true);
      expect(result.repository).toBe("test-owner/allowed-repo");
    });

    it("should return error on API failure", async () => {
      mockGithub.rest.repos.createDispatchEvent = async () => {
        throw new Error("API rate limit exceeded");
      };

      const handler = await main({
        tools: { deploy: { event_type: "deploy", repository: "test-owner/test-repo", max: 5 } },
      });

      const result = await handler({ tool_name: "deploy" }, {});
      expect(result.success).toBe(false);
      expect(result.error).toMatch(/Failed to dispatch/);
      expect(result.error).toMatch(/API rate limit exceeded/);
    });

    it("should return staged result when isStaged is true", async () => {
      const handler = await main({
        tools: { deploy: { event_type: "deploy", repository: "test-owner/test-repo", max: 5 } },
        staged: true,
      });

      const result = await handler({ tool_name: "deploy" }, {});
      expect(result.success).toBe(true);
      expect(result.staged).toBe(true);
      expect(dispatchEventCalls.length).toBe(0);
    });

    it("should parse numeric max from string config", async () => {
      const handler = await main({
        tools: { deploy: { event_type: "deploy", repository: "test-owner/test-repo", max: "3" } },
      });

      // Should allow up to 3 dispatches
      for (let i = 0; i < 3; i++) {
        const result = await handler({ tool_name: "deploy" }, {});
        expect(result.success).toBe(true);
      }

      const overflow = await handler({ tool_name: "deploy" }, {});
      expect(overflow.success).toBe(false);
      expect(overflow.error).toMatch(/Max count of 3/);
    });
  });
});
