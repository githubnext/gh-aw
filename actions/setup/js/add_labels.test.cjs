// @ts-check
const { describe, it, beforeEach } = require("node:test");
const assert = require("node:assert");
const { main } = require("./add_labels.cjs");

describe("add_labels", () => {
  let mockCore;
  let mockGithub;
  let mockContext;

  beforeEach(() => {
    // Reset mocks before each test
    mockCore = {
      info: () => {},
      warning: () => {},
      error: () => {},
      messages: [],
      infos: [],
      warnings: [],
      errors: [],
    };

    // Capture all logged messages
    mockCore.info = (msg) => {
      mockCore.infos.push(msg);
      mockCore.messages.push({ level: "info", message: msg });
    };
    mockCore.warning = (msg) => {
      mockCore.warnings.push(msg);
      mockCore.messages.push({ level: "warning", message: msg });
    };
    mockCore.error = (msg) => {
      mockCore.errors.push(msg);
      mockCore.messages.push({ level: "error", message: msg });
    };

    mockGithub = {
      rest: {
        issues: {
          addLabels: async () => ({}),
        },
      },
    };

    mockContext = {
      repo: {
        owner: "test-owner",
        repo: "test-repo",
      },
      payload: {
        issue: {
          number: 123,
        },
      },
    };

    // Set globals
    global.core = mockCore;
    global.github = mockGithub;
    global.context = mockContext;
  });

  describe("main factory", () => {
    it("should create a handler function with default configuration", async () => {
      const handler = await main();
      assert.strictEqual(typeof handler, "function");
    });

    it("should create a handler function with custom configuration", async () => {
      const handler = await main({
        allowed: ["bug", "enhancement"],
        max: 5,
      });
      assert.strictEqual(typeof handler, "function");
    });

    it("should log configuration on initialization", async () => {
      await main({ allowed: ["bug", "enhancement"], max: 3 });
      assert.ok(mockCore.infos.some((msg) => msg.includes("max=3")));
      assert.ok(mockCore.infos.some((msg) => msg.includes("bug, enhancement")));
    });
  });

  describe("handleAddLabels", () => {
    it("should add labels to an issue using explicit item_number", async () => {
      const handler = await main({ max: 10 });
      const addLabelsCalls = [];

      mockGithub.rest.issues.addLabels = async (params) => {
        addLabelsCalls.push(params);
        return {};
      };

      const result = await handler(
        {
          item_number: 456,
          labels: ["bug", "enhancement"],
        },
        {}
      );

      assert.strictEqual(result.success, true);
      assert.strictEqual(result.number, 456);
      assert.deepStrictEqual(result.labelsAdded, ["bug", "enhancement"]);
      assert.strictEqual(addLabelsCalls.length, 1);
      assert.strictEqual(addLabelsCalls[0].issue_number, 456);
      assert.deepStrictEqual(addLabelsCalls[0].labels, ["bug", "enhancement"]);
    });

    it("should add labels to an issue from context when item_number not provided", async () => {
      const handler = await main({ max: 10 });
      const addLabelsCalls = [];

      mockGithub.rest.issues.addLabels = async (params) => {
        addLabelsCalls.push(params);
        return {};
      };

      const result = await handler(
        {
          labels: ["documentation"],
        },
        {}
      );

      assert.strictEqual(result.success, true);
      assert.strictEqual(result.number, 123);
      assert.deepStrictEqual(result.labelsAdded, ["documentation"]);
      assert.strictEqual(result.contextType, "issue");
    });

    it("should add labels to a pull request from context", async () => {
      mockContext.payload = {
        pull_request: {
          number: 789,
        },
      };

      const handler = await main({ max: 10 });
      const addLabelsCalls = [];

      mockGithub.rest.issues.addLabels = async (params) => {
        addLabelsCalls.push(params);
        return {};
      };

      const result = await handler(
        {
          labels: ["needs-review"],
        },
        {}
      );

      assert.strictEqual(result.success, true);
      assert.strictEqual(result.number, 789);
      assert.strictEqual(result.contextType, "pull request");
    });

    it("should handle invalid item_number", async () => {
      const handler = await main({ max: 10 });

      const result = await handler(
        {
          item_number: "invalid",
          labels: ["bug"],
        },
        {}
      );

      assert.strictEqual(result.success, false);
      assert.ok(result.error.includes("Invalid item number"));
    });

    it("should handle missing item_number and no context", async () => {
      mockContext.payload = {};

      const handler = await main({ max: 10 });

      const result = await handler(
        {
          labels: ["bug"],
        },
        {}
      );

      assert.strictEqual(result.success, false);
      assert.ok(result.error.includes("No issue/PR number available"));
    });

    it("should respect max count limit", async () => {
      const handler = await main({ max: 2 });

      // First call succeeds
      const result1 = await handler(
        {
          item_number: 1,
          labels: ["bug"],
        },
        {}
      );
      assert.strictEqual(result1.success, true);

      // Second call succeeds
      const result2 = await handler(
        {
          item_number: 2,
          labels: ["enhancement"],
        },
        {}
      );
      assert.strictEqual(result2.success, true);

      // Third call should fail
      const result3 = await handler(
        {
          item_number: 3,
          labels: ["documentation"],
        },
        {}
      );
      assert.strictEqual(result3.success, false);
      assert.ok(result3.error.includes("Max count"));
    });

    it("should filter labels based on allowed list", async () => {
      const handler = await main({
        allowed: ["bug", "enhancement"],
        max: 10,
      });

      const addLabelsCalls = [];
      mockGithub.rest.issues.addLabels = async (params) => {
        addLabelsCalls.push(params);
        return {};
      };

      const result = await handler(
        {
          item_number: 100,
          labels: ["bug", "invalid-label", "enhancement"],
        },
        {}
      );

      assert.strictEqual(result.success, true);
      assert.deepStrictEqual(result.labelsAdded, ["bug", "enhancement"]);
    });

    it("should handle empty labels array", async () => {
      const handler = await main({ max: 10 });

      const result = await handler(
        {
          item_number: 100,
          labels: [],
        },
        {}
      );

      assert.strictEqual(result.success, false);
      assert.ok(result.error.includes("labels must be an array") || result.error.includes("No valid labels"));
    });

    it("should handle API errors gracefully", async () => {
      const handler = await main({ max: 10 });

      mockGithub.rest.issues.addLabels = async () => {
        throw new Error("API Error: Not found");
      };

      const result = await handler(
        {
          item_number: 100,
          labels: ["bug"],
        },
        {}
      );

      assert.strictEqual(result.success, false);
      assert.ok(result.error.includes("API Error"));
    });

    it("should deduplicate labels", async () => {
      const handler = await main({ max: 10 });
      const addLabelsCalls = [];

      mockGithub.rest.issues.addLabels = async (params) => {
        addLabelsCalls.push(params);
        return {};
      };

      const result = await handler(
        {
          item_number: 100,
          labels: ["bug", "bug", "enhancement", "bug"],
        },
        {}
      );

      assert.strictEqual(result.success, true);
      assert.deepStrictEqual(result.labelsAdded, ["bug", "enhancement"]);
    });

    it("should sanitize and trim label names", async () => {
      const handler = await main({ max: 10 });
      const addLabelsCalls = [];

      mockGithub.rest.issues.addLabels = async (params) => {
        addLabelsCalls.push(params);
        return {};
      };

      const result = await handler(
        {
          item_number: 100,
          labels: ["  bug  ", " enhancement ", "documentation"],
        },
        {}
      );

      assert.strictEqual(result.success, true);
      assert.ok(result.labelsAdded.length > 0);
    });

    it("should use spread operator for context.repo", async () => {
      const handler = await main({ max: 10 });
      const addLabelsCalls = [];

      mockGithub.rest.issues.addLabels = async (params) => {
        addLabelsCalls.push(params);
        return {};
      };

      await handler(
        {
          item_number: 100,
          labels: ["bug"],
        },
        {}
      );

      assert.strictEqual(addLabelsCalls[0].owner, "test-owner");
      assert.strictEqual(addLabelsCalls[0].repo, "test-repo");
    });
  });
});
