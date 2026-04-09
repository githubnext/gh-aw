// @ts-check
import { describe, it, expect, beforeEach } from "vitest";
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
    mockCore.info = msg => {
      mockCore.infos.push(msg);
      mockCore.messages.push({ level: "info", message: msg });
    };
    mockCore.warning = msg => {
      mockCore.warnings.push(msg);
      mockCore.messages.push({ level: "warning", message: msg });
    };
    mockCore.error = msg => {
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
      expect(typeof handler).toBe("function");
    });

    it("should create a handler function with custom configuration", async () => {
      const handler = await main({
        allowed: ["bug", "enhancement"],
        max: 5,
      });
      expect(typeof handler).toBe("function");
    });

    it("should log configuration on initialization", async () => {
      await main({ allowed: ["bug", "enhancement"], max: 3 });
      expect(mockCore.infos.some(msg => msg.includes("max=3"))).toBe(true);
      expect(mockCore.infos.some(msg => msg.includes("bug, enhancement"))).toBe(true);
    });
  });

  describe("handleAddLabels", () => {
    it("should add labels to an issue using explicit item_number", async () => {
      const handler = await main({ max: 10 });
      const addLabelsCalls = [];

      mockGithub.rest.issues.addLabels = async params => {
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

      expect(result.success).toBe(true);
      expect(result.number).toBe(456);
      expect(result.labelsAdded).toEqual(["bug", "enhancement"]);
      expect(addLabelsCalls.length).toBe(1);
      expect(addLabelsCalls[0].issue_number).toBe(456);
      expect(addLabelsCalls[0].labels).toEqual(["bug", "enhancement"]);
    });

    it("should add labels to an issue from context when item_number not provided", async () => {
      const handler = await main({ max: 10 });
      const addLabelsCalls = [];

      mockGithub.rest.issues.addLabels = async params => {
        addLabelsCalls.push(params);
        return {};
      };

      const result = await handler(
        {
          labels: ["documentation"],
        },
        {}
      );

      expect(result.success).toBe(true);
      expect(result.number).toBe(123);
      expect(result.labelsAdded).toEqual(["documentation"]);
      expect(result.contextType).toBe("issue");
    });

    it("should add labels to a pull request from context", async () => {
      mockContext.payload = {
        pull_request: {
          number: 789,
        },
      };

      const handler = await main({ max: 10 });
      const addLabelsCalls = [];

      mockGithub.rest.issues.addLabels = async params => {
        addLabelsCalls.push(params);
        return {};
      };

      const result = await handler(
        {
          labels: ["needs-review"],
        },
        {}
      );

      expect(result.success).toBe(true);
      expect(result.number).toBe(789);
      expect(result.contextType).toBe("pull request");
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

      expect(result.success).toBe(false);
      expect(result.error.includes("Invalid item number")).toBe(true);
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

      expect(result.success).toBe(false);
      expect(result.error.includes("No issue/PR number available")).toBe(true);
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
      expect(result1.success).toBe(true);

      // Second call succeeds
      const result2 = await handler(
        {
          item_number: 2,
          labels: ["enhancement"],
        },
        {}
      );
      expect(result2.success).toBe(true);

      // Third call should fail
      const result3 = await handler(
        {
          item_number: 3,
          labels: ["documentation"],
        },
        {}
      );
      expect(result3.success).toBe(false);
      expect(result3.error.includes("Max count")).toBe(true);
    });

    it("should filter labels based on allowed list", async () => {
      const handler = await main({
        allowed: ["bug", "enhancement"],
        max: 10,
      });

      const addLabelsCalls = [];
      mockGithub.rest.issues.addLabels = async params => {
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

      expect(result.success).toBe(true);
      expect(result.labelsAdded).toEqual(["bug", "enhancement"]);
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

      expect(result.success).toBe(false);
      expect(result.error).toContain("No labels provided");
      expect(result.error).toContain("repository's available labels");
    });

    it("should handle missing labels field", async () => {
      const handler = await main({ max: 10 });

      const result = await handler(
        {
          item_number: 100,
        },
        {}
      );

      expect(result.success).toBe(false);
      expect(result.error).toContain("No labels provided");
      expect(result.error).toContain("repository's available labels");
    });

    it("should return allowed labels list when labels missing and allowed list configured", async () => {
      const handler = await main({
        allowed: ["bug", "enhancement", "documentation"],
        max: 10,
      });

      const result = await handler(
        {
          item_number: 100,
          labels: [],
        },
        {}
      );

      expect(result.success).toBe(false);
      expect(result.error).toContain("No labels provided");
      expect(result.error).toContain("allowed list");
      expect(result.error).toContain("bug");
      expect(result.error).toContain("enhancement");
      expect(result.error).toContain("documentation");
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

      expect(result.success).toBe(false);
      expect(result.error.includes("API Error")).toBe(true);
    });

    it("should deduplicate labels", async () => {
      const handler = await main({ max: 10 });
      const addLabelsCalls = [];

      mockGithub.rest.issues.addLabels = async params => {
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

      expect(result.success).toBe(true);
      expect(result.labelsAdded).toEqual(["bug", "enhancement"]);
    });

    it("should sanitize and trim label names", async () => {
      const handler = await main({ max: 10 });
      const addLabelsCalls = [];

      mockGithub.rest.issues.addLabels = async params => {
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

      expect(result.success).toBe(true);
      expect(result.labelsAdded.length).toBeGreaterThan(0);
    });

    it("should use spread operator for context.repo", async () => {
      const handler = await main({ max: 10 });
      const addLabelsCalls = [];

      mockGithub.rest.issues.addLabels = async params => {
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

      expect(addLabelsCalls[0].owner).toBe("test-owner");
      expect(addLabelsCalls[0].repo).toBe("test-repo");
    });

    it("should support target-repo from config", async () => {
      const handler = await main({
        max: 10,
        "target-repo": "external-org/external-repo",
      });
      const addLabelsCalls = [];

      mockGithub.rest.issues.addLabels = async params => {
        addLabelsCalls.push(params);
        return {};
      };

      const result = await handler(
        {
          item_number: 100,
          labels: ["bug"],
        },
        {}
      );

      expect(result.success).toBe(true);
      expect(addLabelsCalls[0].owner).toBe("external-org");
      expect(addLabelsCalls[0].repo).toBe("external-repo");
    });

    it("should support repo field in message for cross-repository operations", async () => {
      const handler = await main({
        max: 10,
        "target-repo": "default-org/default-repo",
        allowed_repos: ["cross-org/cross-repo"],
      });
      const addLabelsCalls = [];

      mockGithub.rest.issues.addLabels = async params => {
        addLabelsCalls.push(params);
        return {};
      };

      const result = await handler(
        {
          item_number: 456,
          labels: ["enhancement"],
          repo: "cross-org/cross-repo",
        },
        {}
      );

      expect(result.success).toBe(true);
      expect(addLabelsCalls[0].owner).toBe("cross-org");
      expect(addLabelsCalls[0].repo).toBe("cross-repo");
    });

    it("should reject repo not in allowed-repos list", async () => {
      const handler = await main({
        max: 10,
        "target-repo": "default-org/default-repo",
        allowed_repos: ["allowed-org/allowed-repo"],
      });

      const result = await handler(
        {
          item_number: 100,
          labels: ["bug"],
          repo: "unauthorized-org/unauthorized-repo",
        },
        {}
      );

      expect(result.success).toBe(false);
      expect(result.error).toContain("not in the allowed-repos list");
    });

    it("should qualify bare repo name with default repo org", async () => {
      const handler = await main({
        max: 10,
        "target-repo": "github/default-repo",
        allowed_repos: ["github/gh-aw"],
      });
      const addLabelsCalls = [];

      mockGithub.rest.issues.addLabels = async params => {
        addLabelsCalls.push(params);
        return {};
      };

      const result = await handler(
        {
          item_number: 100,
          labels: ["bug"],
          repo: "gh-aw", // Bare name without org
        },
        {}
      );

      expect(result.success).toBe(true);
      expect(addLabelsCalls[0].owner).toBe("github");
      expect(addLabelsCalls[0].repo).toBe("gh-aw");
    });

    it("should enforce max limit on labels per operation", async () => {
      const handler = await main({ max: 10 });

      // Try to add more than MAX_LABELS (10)
      const result = await handler(
        {
          item_number: 100,
          labels: [
            "label1",
            "label2",
            "label3",
            "label4",
            "label5",
            "label6",
            "label7",
            "label8",
            "label9",
            "label10",
            "label11", // 11th label exceeds limit
          ],
        },
        {}
      );

      expect(result.success).toBe(false);
      expect(result.error).toContain("E003");
      expect(result.error).toContain("Cannot add more than 10 labels");
      expect(result.error).toContain("received 11");
    });

    it("should resolve temporary ID in item_number to real issue number", async () => {
      const handler = await main({ max: 10 });
      const addLabelsCalls = [];

      mockGithub.rest.issues.addLabels = async params => {
        addLabelsCalls.push(params);
        return {};
      };

      const result = await handler(
        {
          item_number: "aw_report1",
          labels: ["bug"],
        },
        { aw_report1: { repo: "test-owner/test-repo", number: 42 } }
      );

      expect(result.success).toBe(true);
      expect(result.number).toBe(42);
      expect(addLabelsCalls.length).toBe(1);
      expect(addLabelsCalls[0].issue_number).toBe(42);
    });

    it("should defer when item_number is an unresolved temporary ID", async () => {
      const handler = await main({ max: 10 });

      const result = await handler(
        {
          item_number: "aw_report1",
          labels: ["bug"],
        },
        {}
      );

      expect(result.success).toBe(false);
      expect(result.deferred).toBe(true);
      expect(result.error).toContain("aw_report1");
    });

    it("should resolve temporary ID with hash prefix in item_number", async () => {
      const handler = await main({ max: 10 });
      const addLabelsCalls = [];

      mockGithub.rest.issues.addLabels = async params => {
        addLabelsCalls.push(params);
        return {};
      };

      const result = await handler(
        {
          item_number: "#aw_report1",
          labels: ["enhancement"],
        },
        { aw_report1: { repo: "test-owner/test-repo", number: 99 } }
      );

      expect(result.success).toBe(true);
      expect(result.number).toBe(99);
      expect(addLabelsCalls[0].issue_number).toBe(99);
    });

    it("should preview labels in staged mode without calling API", async () => {
      const handler = await main({ max: 10, staged: true });
      const addLabelsCalls = [];

      mockGithub.rest.issues.addLabels = async params => {
        addLabelsCalls.push(params);
        return {};
      };

      const result = await handler(
        {
          item_number: 100,
          labels: ["bug", "enhancement"],
        },
        {}
      );

      expect(result.success).toBe(true);
      expect(result.staged).toBe(true);
      expect(result.previewInfo).toBeDefined();
      expect(result.previewInfo.number).toBe(100);
      expect(result.previewInfo.labels).toEqual(["bug", "enhancement"]);
      expect(addLabelsCalls.length).toBe(0);
    });

    it("should count staged calls toward processedCount", async () => {
      const handler = await main({ max: 1, staged: true });

      const result1 = await handler({ item_number: 1, labels: ["bug"] }, {});
      expect(result1.success).toBe(true);
      expect(result1.staged).toBe(true);

      const result2 = await handler({ item_number: 2, labels: ["enhancement"] }, {});
      expect(result2.success).toBe(false);
      expect(result2.error).toContain("Max count");
    });

    it("should filter out labels matching blocked patterns", async () => {
      const handler = await main({
        max: 10,
        blocked: ["internal-*", "~*"],
      });
      const addLabelsCalls = [];

      mockGithub.rest.issues.addLabels = async params => {
        addLabelsCalls.push(params);
        return {};
      };

      const result = await handler(
        {
          item_number: 100,
          labels: ["bug", "internal-only", "~secret", "enhancement"],
        },
        {}
      );

      expect(result.success).toBe(true);
      expect(result.labelsAdded).toEqual(["bug", "enhancement"]);
      expect(addLabelsCalls[0].labels).toEqual(["bug", "enhancement"]);
    });

    it("should succeed with empty labelsAdded when all labels filtered by allowed list", async () => {
      const handler = await main({
        max: 10,
        allowed: ["bug", "enhancement"],
      });

      const result = await handler(
        {
          item_number: 100,
          labels: ["documentation", "invalid-label"],
        },
        {}
      );

      expect(result.success).toBe(true);
      expect(result.labelsAdded).toEqual([]);
      expect(result.message).toContain("No valid labels");
    });

    it("should use default max=10 when config.max is not provided", async () => {
      // No max provided - defaults to 10 via ?? operator
      const handler = await main({});
      const result = await handler({ item_number: 1, labels: ["bug"] }, {});
      expect(result.success).toBe(true);
    });

    it("should handle labels array containing only whitespace strings gracefully", async () => {
      const handler = await main({ max: 10 });

      const result = await handler(
        {
          item_number: 100,
          labels: ["   ", "\t"],
        },
        {}
      );

      // Whitespace-only labels are sanitized away, resulting in no labels added
      expect(result.success).toBe(true);
      expect(result.labelsAdded).toEqual([]);
    });

    it("should log initialization info without allowed/blocked when not configured", async () => {
      await main({ max: 5 });
      // Should not log allowed/blocked info when not configured
      expect(mockCore.infos.some(msg => msg.includes("Allowed labels:"))).toBe(false);
      expect(mockCore.infos.some(msg => msg.includes("Blocked patterns:"))).toBe(false);
      expect(mockCore.infos.some(msg => msg.includes("max=5"))).toBe(true);
    });
  });
});
