import { describe, it, expect, beforeEach, vi } from "vitest";

// Mock the global objects that GitHub Actions provides
const mockCore = {
  debug: vi.fn(),
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(),
  },
};

const mockContext = {
  eventName: "issues",
  runId: 12345,
  repo: {
    owner: "testowner",
    repo: "testrepo",
  },
  payload: {
    issue: {
      number: 42,
    },
    pull_request: {
      number: 100,
    },
    repository: {
      html_url: "https://github.com/testowner/testrepo",
    },
  },
};

// Set up global mocks before importing the module
global.core = mockCore;
global.context = mockContext;

const {
  createEntityCallbacks,
  checkLabelFilter,
  checkTitlePrefixFilter,
  parseEntityConfig,
  resolveEntityNumber,
  escapeMarkdownTitle,
  ISSUE_CONFIG,
  PULL_REQUEST_CONFIG,
} = require("./close_entity_helpers.cjs");

describe("close_entity_helpers", () => {
  beforeEach(() => {
    vi.clearAllMocks();

    // Reset environment variables
    delete process.env.GH_AW_CLOSE_ISSUE_REQUIRED_LABELS;
    delete process.env.GH_AW_CLOSE_ISSUE_REQUIRED_TITLE_PREFIX;
    delete process.env.GH_AW_CLOSE_ISSUE_TARGET;
    delete process.env.GH_AW_CLOSE_PR_REQUIRED_LABELS;
    delete process.env.GH_AW_CLOSE_PR_REQUIRED_TITLE_PREFIX;
    delete process.env.GH_AW_CLOSE_PR_TARGET;

    // Reset context to default state
    global.context.eventName = "issues";
    global.context.payload.issue = { number: 42 };
    global.context.payload.pull_request = { number: 100 };
  });

  describe("checkLabelFilter", () => {
    it("should return true when no required labels specified", () => {
      const labels = [{ name: "bug" }];
      expect(checkLabelFilter(labels, [])).toBe(true);
    });

    it("should return true when entity has one of the required labels", () => {
      const labels = [{ name: "bug" }, { name: "enhancement" }];
      expect(checkLabelFilter(labels, ["bug", "wontfix"])).toBe(true);
    });

    it("should return false when entity has none of the required labels", () => {
      const labels = [{ name: "bug" }];
      expect(checkLabelFilter(labels, ["enhancement", "wontfix"])).toBe(false);
    });

    it("should return false when entity has no labels and required labels specified", () => {
      const labels = [];
      expect(checkLabelFilter(labels, ["bug"])).toBe(false);
    });
  });

  describe("checkTitlePrefixFilter", () => {
    it("should return true when no required prefix specified", () => {
      expect(checkTitlePrefixFilter("Some Title", "")).toBe(true);
    });

    it("should return true when title starts with required prefix", () => {
      expect(checkTitlePrefixFilter("[bug] Fix something", "[bug]")).toBe(true);
    });

    it("should return false when title does not start with required prefix", () => {
      expect(checkTitlePrefixFilter("Fix something", "[bug]")).toBe(false);
    });

    it("should be case-sensitive", () => {
      expect(checkTitlePrefixFilter("[BUG] Fix something", "[bug]")).toBe(false);
    });
  });

  describe("parseEntityConfig", () => {
    it("should return defaults when no environment variables set", () => {
      const config = parseEntityConfig("GH_AW_CLOSE_ISSUE");
      expect(config.requiredLabels).toEqual([]);
      expect(config.requiredTitlePrefix).toBe("");
      expect(config.target).toBe("triggering");
    });

    it("should parse required labels from environment", () => {
      process.env.GH_AW_CLOSE_ISSUE_REQUIRED_LABELS = "bug, enhancement, stale";
      const config = parseEntityConfig("GH_AW_CLOSE_ISSUE");
      expect(config.requiredLabels).toEqual(["bug", "enhancement", "stale"]);
    });

    it("should parse required title prefix from environment", () => {
      process.env.GH_AW_CLOSE_ISSUE_REQUIRED_TITLE_PREFIX = "[refactor]";
      const config = parseEntityConfig("GH_AW_CLOSE_ISSUE");
      expect(config.requiredTitlePrefix).toBe("[refactor]");
    });

    it("should parse target from environment", () => {
      process.env.GH_AW_CLOSE_ISSUE_TARGET = "*";
      const config = parseEntityConfig("GH_AW_CLOSE_ISSUE");
      expect(config.target).toBe("*");
    });

    it("should work with PR environment variable prefix", () => {
      process.env.GH_AW_CLOSE_PR_REQUIRED_LABELS = "ready-to-close";
      process.env.GH_AW_CLOSE_PR_TARGET = "123";
      const config = parseEntityConfig("GH_AW_CLOSE_PR");
      expect(config.requiredLabels).toEqual(["ready-to-close"]);
      expect(config.target).toBe("123");
    });
  });

  describe("resolveEntityNumber", () => {
    describe("with target '*'", () => {
      it("should resolve from item number field", () => {
        const result = resolveEntityNumber(ISSUE_CONFIG, "*", { issue_number: 50 }, true);
        expect(result.success).toBe(true);
        expect(result.number).toBe(50);
      });

      it("should handle string number field", () => {
        const result = resolveEntityNumber(ISSUE_CONFIG, "*", { issue_number: "75" }, true);
        expect(result.success).toBe(true);
        expect(result.number).toBe(75);
      });

      it("should fail when number field is missing", () => {
        const result = resolveEntityNumber(ISSUE_CONFIG, "*", {}, true);
        expect(result.success).toBe(false);
        expect(result.message).toContain("no issue_number specified");
      });

      it("should fail when number field is invalid", () => {
        const result = resolveEntityNumber(ISSUE_CONFIG, "*", { issue_number: "abc" }, true);
        expect(result.success).toBe(false);
        expect(result.message).toContain("Invalid issue number specified");
      });

      it("should fail when number is zero or negative", () => {
        const result = resolveEntityNumber(ISSUE_CONFIG, "*", { issue_number: -5 }, true);
        expect(result.success).toBe(false);
        expect(result.message).toContain("Invalid issue number specified");
      });

      it("should fail when number is zero (falsy)", () => {
        const result = resolveEntityNumber(ISSUE_CONFIG, "*", { issue_number: 0 }, true);
        expect(result.success).toBe(false);
        // 0 is falsy in JS, so it hits the "no number specified" branch
        expect(result.message).toContain("no issue_number specified");
      });
    });

    describe("with explicit target number", () => {
      it("should resolve from target configuration", () => {
        const result = resolveEntityNumber(ISSUE_CONFIG, "123", {}, true);
        expect(result.success).toBe(true);
        expect(result.number).toBe(123);
      });

      it("should fail when target is not a valid number", () => {
        const result = resolveEntityNumber(ISSUE_CONFIG, "invalid", {}, true);
        expect(result.success).toBe(false);
        expect(result.message).toContain("Invalid issue number in target configuration");
      });
    });

    describe("with target 'triggering'", () => {
      it("should resolve from context in issue event", () => {
        const result = resolveEntityNumber(ISSUE_CONFIG, "triggering", {}, true);
        expect(result.success).toBe(true);
        expect(result.number).toBe(42);
      });

      it("should fail when not in entity context", () => {
        const result = resolveEntityNumber(ISSUE_CONFIG, "triggering", {}, false);
        expect(result.success).toBe(false);
        expect(result.message).toContain("Not in issue context");
      });

      it("should fail when context payload has no number", () => {
        global.context.payload.issue = {};
        const result = resolveEntityNumber(ISSUE_CONFIG, "triggering", {}, true);
        expect(result.success).toBe(false);
        expect(result.message).toContain("no issue found in payload");
      });
    });

    describe("for pull requests", () => {
      beforeEach(() => {
        global.context.eventName = "pull_request";
      });

      it("should resolve PR number from item with target '*'", () => {
        const result = resolveEntityNumber(PULL_REQUEST_CONFIG, "*", { pull_request_number: 200 }, true);
        expect(result.success).toBe(true);
        expect(result.number).toBe(200);
      });

      it("should resolve PR number from triggering context", () => {
        const result = resolveEntityNumber(PULL_REQUEST_CONFIG, "triggering", {}, true);
        expect(result.success).toBe(true);
        expect(result.number).toBe(100);
      });
    });
  });

  describe("escapeMarkdownTitle", () => {
    it("should escape square brackets", () => {
      expect(escapeMarkdownTitle("[feature] Add new thing")).toBe("\\[feature\\] Add new thing");
    });

    it("should escape parentheses", () => {
      expect(escapeMarkdownTitle("Fix bug (urgent)")).toBe("Fix bug \\(urgent\\)");
    });

    it("should escape all markdown special characters", () => {
      expect(escapeMarkdownTitle("[test] (foo) [bar]")).toBe("\\[test\\] \\(foo\\) \\[bar\\]");
    });

    it("should not modify titles without special characters", () => {
      expect(escapeMarkdownTitle("Simple title")).toBe("Simple title");
    });
  });

  describe("ISSUE_CONFIG", () => {
    it("should have correct entity type", () => {
      expect(ISSUE_CONFIG.entityType).toBe("issue");
    });

    it("should have correct item type", () => {
      expect(ISSUE_CONFIG.itemType).toBe("close_issue");
    });

    it("should have correct item type display", () => {
      expect(ISSUE_CONFIG.itemTypeDisplay).toBe("close-issue");
    });

    it("should have correct context events", () => {
      expect(ISSUE_CONFIG.contextEvents).toContain("issues");
      expect(ISSUE_CONFIG.contextEvents).toContain("issue_comment");
    });

    it("should have correct URL path", () => {
      expect(ISSUE_CONFIG.urlPath).toBe("issues");
    });
  });

  describe("PULL_REQUEST_CONFIG", () => {
    it("should have correct entity type", () => {
      expect(PULL_REQUEST_CONFIG.entityType).toBe("pull_request");
    });

    it("should have correct item type", () => {
      expect(PULL_REQUEST_CONFIG.itemType).toBe("close_pull_request");
    });

    it("should have correct item type display", () => {
      expect(PULL_REQUEST_CONFIG.itemTypeDisplay).toBe("close-pull-request");
    });

    it("should have correct context events", () => {
      expect(PULL_REQUEST_CONFIG.contextEvents).toContain("pull_request");
      expect(PULL_REQUEST_CONFIG.contextEvents).toContain("pull_request_review_comment");
    });

    it("should have correct URL path", () => {
      expect(PULL_REQUEST_CONFIG.urlPath).toBe("pull");
    });
  });

  describe("createEntityCallbacks", () => {
    let mockGithub;

    beforeEach(() => {
      mockGithub = {
        rest: {
          issues: {
            get: vi.fn(),
            createComment: vi.fn(),
            update: vi.fn(),
          },
          pulls: {
            get: vi.fn(),
            update: vi.fn(),
          },
        },
      };
    });

    describe("for issues", () => {
      it("should create getDetails callback that fetches issue", async () => {
        const callbacks = createEntityCallbacks(ISSUE_CONFIG);

        mockGithub.rest.issues.get.mockResolvedValue({
          data: {
            number: 42,
            title: "Test Issue",
            labels: [{ name: "bug" }],
            html_url: "https://github.com/testowner/testrepo/issues/42",
            state: "open",
          },
        });

        const result = await callbacks.getDetails(mockGithub, "testowner", "testrepo", 42);

        expect(mockGithub.rest.issues.get).toHaveBeenCalledWith({
          owner: "testowner",
          repo: "testrepo",
          issue_number: 42,
        });
        expect(result.number).toBe(42);
        expect(result.title).toBe("Test Issue");
      });

      it("should create addComment callback that adds issue comment", async () => {
        const callbacks = createEntityCallbacks(ISSUE_CONFIG);

        mockGithub.rest.issues.createComment.mockResolvedValue({
          data: {
            id: 123,
            html_url: "https://github.com/testowner/testrepo/issues/42#issuecomment-123",
          },
        });

        const result = await callbacks.addComment(mockGithub, "testowner", "testrepo", 42, "Test comment");

        expect(mockGithub.rest.issues.createComment).toHaveBeenCalledWith({
          owner: "testowner",
          repo: "testrepo",
          issue_number: 42,
          body: "Test comment",
        });
        expect(result.id).toBe(123);
      });

      it("should create closeEntity callback that closes issue", async () => {
        const callbacks = createEntityCallbacks(ISSUE_CONFIG);

        mockGithub.rest.issues.update.mockResolvedValue({
          data: {
            number: 42,
            html_url: "https://github.com/testowner/testrepo/issues/42",
            title: "Test Issue",
          },
        });

        const result = await callbacks.closeEntity(mockGithub, "testowner", "testrepo", 42);

        expect(mockGithub.rest.issues.update).toHaveBeenCalledWith({
          owner: "testowner",
          repo: "testrepo",
          issue_number: 42,
          state: "closed",
        });
        expect(result.number).toBe(42);
      });

      it("should throw error when issue not found", async () => {
        const callbacks = createEntityCallbacks(ISSUE_CONFIG);

        mockGithub.rest.issues.get.mockResolvedValue({
          data: null,
        });

        await expect(callbacks.getDetails(mockGithub, "testowner", "testrepo", 42)).rejects.toThrow(
          "Issue #42 not found in testowner/testrepo"
        );
      });
    });

    describe("for pull requests", () => {
      it("should create getDetails callback that fetches PR", async () => {
        const callbacks = createEntityCallbacks(PULL_REQUEST_CONFIG);

        mockGithub.rest.pulls.get.mockResolvedValue({
          data: {
            number: 100,
            title: "Test PR",
            labels: [{ name: "enhancement" }],
            html_url: "https://github.com/testowner/testrepo/pull/100",
            state: "open",
          },
        });

        const result = await callbacks.getDetails(mockGithub, "testowner", "testrepo", 100);

        expect(mockGithub.rest.pulls.get).toHaveBeenCalledWith({
          owner: "testowner",
          repo: "testrepo",
          pull_number: 100,
        });
        expect(result.number).toBe(100);
        expect(result.title).toBe("Test PR");
      });

      it("should create addComment callback that adds PR comment", async () => {
        const callbacks = createEntityCallbacks(PULL_REQUEST_CONFIG);

        mockGithub.rest.issues.createComment.mockResolvedValue({
          data: {
            id: 456,
            html_url: "https://github.com/testowner/testrepo/issues/100#issuecomment-456",
          },
        });

        const result = await callbacks.addComment(mockGithub, "testowner", "testrepo", 100, "Test PR comment");

        expect(mockGithub.rest.issues.createComment).toHaveBeenCalledWith({
          owner: "testowner",
          repo: "testrepo",
          issue_number: 100,
          body: "Test PR comment",
        });
        expect(result.id).toBe(456);
      });

      it("should create closeEntity callback that closes PR", async () => {
        const callbacks = createEntityCallbacks(PULL_REQUEST_CONFIG);

        mockGithub.rest.pulls.update.mockResolvedValue({
          data: {
            number: 100,
            html_url: "https://github.com/testowner/testrepo/pull/100",
            title: "Test PR",
          },
        });

        const result = await callbacks.closeEntity(mockGithub, "testowner", "testrepo", 100);

        expect(mockGithub.rest.pulls.update).toHaveBeenCalledWith({
          owner: "testowner",
          repo: "testrepo",
          pull_number: 100,
          state: "closed",
        });
        expect(result.number).toBe(100);
      });

      it("should throw error when PR not found", async () => {
        const callbacks = createEntityCallbacks(PULL_REQUEST_CONFIG);

        mockGithub.rest.pulls.get.mockResolvedValue({
          data: null,
        });

        await expect(callbacks.getDetails(mockGithub, "testowner", "testrepo", 100)).rejects.toThrow(
          "Pull Request #100 not found in testowner/testrepo"
        );
      });
    });
  });
});
