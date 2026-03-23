// @ts-check
import { describe, it, expect, beforeEach, vi } from "vitest";
import { createRequire } from "module";

const require = createRequire(import.meta.url);

describe("update_discussion.cjs - label support", () => {
  let mockGithub;
  let mockCore;
  let mockContext;

  beforeEach(() => {
    vi.clearAllMocks();
    vi.resetModules();

    mockCore = {
      debug: vi.fn(),
      info: vi.fn(),
      notice: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),
      setFailed: vi.fn(),
      setOutput: vi.fn(),
      summary: {
        addRaw: vi.fn().mockReturnThis(),
        write: vi.fn().mockResolvedValue(undefined),
      },
    };

    mockGithub = {
      graphql: vi.fn(),
    };

    mockContext = {
      eventName: "discussion",
      repo: { owner: "testowner", repo: "testrepo" },
      serverUrl: "https://github.com",
      runId: 12345,
      payload: {
        discussion: { number: 42 },
      },
    };

    global.core = mockCore;
    global.github = mockGithub;
    global.context = mockContext;
  });

  /**
   * Builds the default graphql mock responses for a discussion query + mutation sequence.
   * @param {object} opts
   * @param {string[]} [opts.currentLabelIds] - IDs of labels currently on the discussion
   * @param {string[]} [opts.currentLabelNames] - Names of labels currently on the discussion
   * @param {Array<{id: string, name: string}>} [opts.repoLabels] - Labels available in the repo
   */
  function buildGraphqlMock({ currentLabelIds = [], currentLabelNames = [], repoLabels = [] } = {}) {
    const currentLabels = currentLabelIds.map((id, i) => ({ id, name: currentLabelNames[i] ?? id }));

    return vi.fn().mockImplementation(async query => {
      // Fetch discussion node ID + current labels
      if (query.includes("discussion(number:")) {
        return {
          repository: {
            discussion: {
              id: "D_disc123",
              title: "Test Discussion",
              body: "Test body",
              url: "https://github.com/testowner/testrepo/discussions/42",
              labels: { nodes: currentLabels },
            },
          },
        };
      }
      // Update discussion title/body mutation
      if (query.includes("updateDiscussion")) {
        return {
          updateDiscussion: {
            discussion: {
              id: "D_disc123",
              title: "Test Discussion",
              body: "Test body",
              url: "https://github.com/testowner/testrepo/discussions/42",
            },
          },
        };
      }
      // Fetch repository labels (for label ID lookup)
      if (query.includes("labels(first:")) {
        return {
          repository: {
            labels: { nodes: repoLabels },
          },
        };
      }
      // addLabelsToLabelable / removeLabelsFromLabelable mutations
      if (query.includes("addLabelsToLabelable") || query.includes("removeLabelsFromLabelable")) {
        return {};
      }
      return {};
    });
  }

  describe("label updates", () => {
    it("should add labels when discussion has no current labels", async () => {
      mockGithub.graphql = buildGraphqlMock({
        currentLabelIds: [],
        currentLabelNames: [],
        repoLabels: [
          { id: "LA_bug", name: "bug" },
          { id: "LA_idea", name: "idea" },
        ],
      });

      const { main } = require("./update_discussion.cjs");
      const handler = await main({ target: "triggering", max: 1, allow_labels: true });
      const result = await handler({ labels: ["bug"] }, {});

      expect(result.success).toBe(true);

      const addCall = mockGithub.graphql.mock.calls.find(c => c[0].includes("addLabelsToLabelable"));
      expect(addCall).toBeDefined();
      expect(addCall[1].labelIds).toEqual(["LA_bug"]);

      const removeCall = mockGithub.graphql.mock.calls.find(c => c[0].includes("removeLabelsFromLabelable"));
      expect(removeCall).toBeUndefined();
    });

    it("should remove labels no longer in the requested set", async () => {
      mockGithub.graphql = buildGraphqlMock({
        currentLabelIds: ["LA_bug", "LA_old"],
        currentLabelNames: ["bug", "old-label"],
        repoLabels: [
          { id: "LA_bug", name: "bug" },
          { id: "LA_idea", name: "idea" },
        ],
      });

      const { main } = require("./update_discussion.cjs");
      const handler = await main({ target: "triggering", max: 1, allow_labels: true });
      const result = await handler({ labels: ["bug"] }, {});

      expect(result.success).toBe(true);

      const removeCall = mockGithub.graphql.mock.calls.find(c => c[0].includes("removeLabelsFromLabelable"));
      expect(removeCall).toBeDefined();
      expect(removeCall[1].labelIds).toEqual(["LA_old"]);

      const addCall = mockGithub.graphql.mock.calls.find(c => c[0].includes("addLabelsToLabelable"));
      expect(addCall).toBeUndefined();
    });

    it("should replace all labels (add new, remove old)", async () => {
      mockGithub.graphql = buildGraphqlMock({
        currentLabelIds: ["LA_old"],
        currentLabelNames: ["old-label"],
        repoLabels: [
          { id: "LA_bug", name: "bug" },
          { id: "LA_idea", name: "idea" },
        ],
      });

      const { main } = require("./update_discussion.cjs");
      const handler = await main({ target: "triggering", max: 1, allow_labels: true });
      const result = await handler({ labels: ["bug", "idea"] }, {});

      expect(result.success).toBe(true);

      const addCall = mockGithub.graphql.mock.calls.find(c => c[0].includes("addLabelsToLabelable"));
      expect(addCall).toBeDefined();
      expect(addCall[1].labelIds).toEqual(expect.arrayContaining(["LA_bug", "LA_idea"]));

      const removeCall = mockGithub.graphql.mock.calls.find(c => c[0].includes("removeLabelsFromLabelable"));
      expect(removeCall).toBeDefined();
      expect(removeCall[1].labelIds).toEqual(["LA_old"]);
    });

    it("should do nothing when requested labels are unchanged", async () => {
      mockGithub.graphql = buildGraphqlMock({
        currentLabelIds: ["LA_bug"],
        currentLabelNames: ["bug"],
        repoLabels: [{ id: "LA_bug", name: "bug" }],
      });

      const { main } = require("./update_discussion.cjs");
      const handler = await main({ target: "triggering", max: 1, allow_labels: true });
      const result = await handler({ labels: ["bug"] }, {});

      expect(result.success).toBe(true);

      const addCall = mockGithub.graphql.mock.calls.find(c => c[0].includes("addLabelsToLabelable"));
      expect(addCall).toBeUndefined();

      const removeCall = mockGithub.graphql.mock.calls.find(c => c[0].includes("removeLabelsFromLabelable"));
      expect(removeCall).toBeUndefined();
    });

    it("should filter labels against allowed_labels list", async () => {
      mockGithub.graphql = buildGraphqlMock({
        currentLabelIds: [],
        currentLabelNames: [],
        repoLabels: [
          { id: "LA_bug", name: "bug" },
          { id: "LA_idea", name: "idea" },
          { id: "LA_other", name: "other" },
        ],
      });

      const { main } = require("./update_discussion.cjs");
      const handler = await main({
        target: "triggering",
        max: 1,
        allow_labels: true,
        allowed_labels: ["bug", "idea"],
      });
      // "other" is not in allowed_labels and should be filtered out
      const result = await handler({ labels: ["bug", "other"] }, {});

      expect(result.success).toBe(true);

      const addCall = mockGithub.graphql.mock.calls.find(c => c[0].includes("addLabelsToLabelable"));
      expect(addCall).toBeDefined();
      // Only "bug" should be added since "other" is not in allowed_labels
      expect(addCall[1].labelIds).toEqual(["LA_bug"]);
    });

    it("should not update labels when allow_labels is false", async () => {
      mockGithub.graphql = buildGraphqlMock({
        currentLabelIds: [],
        currentLabelNames: [],
        repoLabels: [{ id: "LA_bug", name: "bug" }],
      });

      const { main } = require("./update_discussion.cjs");
      // allow_labels not set (undefined) - labels update disabled; title update still works
      const handler = await main({ target: "triggering", max: 1, allow_title: true });
      const result = await handler({ title: "Updated Title", labels: ["bug"] }, {});

      expect(result.success).toBe(true);
      // Labels should not be updated (allow_labels not set)
      const addCall = mockGithub.graphql.mock.calls.find(c => c[0].includes("addLabelsToLabelable"));
      expect(addCall).toBeUndefined();
      // A warning should have been logged
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Label update not allowed"));
    });

    it("should update labels in combination with title", async () => {
      mockGithub.graphql = buildGraphqlMock({
        currentLabelIds: [],
        currentLabelNames: [],
        repoLabels: [{ id: "LA_bug", name: "bug" }],
      });

      const { main } = require("./update_discussion.cjs");
      const handler = await main({ target: "triggering", max: 1, allow_title: true, allow_labels: true });
      const result = await handler({ title: "New Title", labels: ["bug"] }, {});

      expect(result.success).toBe(true);

      const updateCall = mockGithub.graphql.mock.calls.find(c => c[0].includes("updateDiscussion"));
      expect(updateCall).toBeDefined();

      const addCall = mockGithub.graphql.mock.calls.find(c => c[0].includes("addLabelsToLabelable"));
      expect(addCall).toBeDefined();
      expect(addCall[1].labelIds).toEqual(["LA_bug"]);
    });
  });

  describe("label-only updates (no title/body)", () => {
    it("should update only labels without touching title or body", async () => {
      mockGithub.graphql = buildGraphqlMock({
        currentLabelIds: [],
        currentLabelNames: [],
        repoLabels: [{ id: "LA_idea", name: "idea" }],
      });

      const { main } = require("./update_discussion.cjs");
      const handler = await main({ target: "triggering", max: 1, allow_labels: true });
      const result = await handler({ labels: ["idea"] }, {});

      expect(result.success).toBe(true);

      // updateDiscussion mutation should NOT be called (no title/body update)
      const updateCall = mockGithub.graphql.mock.calls.find(c => c[0].includes("updateDiscussion"));
      expect(updateCall).toBeUndefined();

      // addLabelsToLabelable should be called
      const addCall = mockGithub.graphql.mock.calls.find(c => c[0].includes("addLabelsToLabelable"));
      expect(addCall).toBeDefined();
    });
  });
});
