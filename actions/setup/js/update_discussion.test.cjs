// @ts-check
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
const { main } = require("./update_discussion.cjs");

/**
 * Tests for update_discussion.cjs
 *
 * These tests verify the safe-outputs behavior of discussion updates, particularly:
 * - Label-only updates do NOT modify title or body
 * - Title/body-only updates do NOT modify labels
 * - Config guards prevent unauthorized field updates
 * - All combinations of allowed fields work correctly
 */
describe("update_discussion", () => {
  let mockCore;
  let mockGithub;
  let mockContext;
  let originalGlobals;
  let originalEnv;

  // Track all graphql calls for assertion
  let graphqlCalls = /** @type {Array<{query: string, variables: any}>} */ [];

  // Default discussion returned by the "fetch discussion" query
  const defaultDiscussion = {
    id: "D_kwDOTest123",
    title: "Original Title",
    body: "Original body content",
    url: "https://github.com/testowner/testrepo/discussions/42",
    number: 42,
  };

  // Default repo labels for fetchLabelNodeIds
  const defaultLabels = [
    { id: "LA_kwDO1", name: "Label1" },
    { id: "LA_kwDO2", name: "Label2" },
    { id: "LA_kwDO3", name: "Label3" },
    { id: "LA_kwDO4", name: "Label4" },
    { id: "LA_kwDO_bug", name: "bug" },
    { id: "LA_kwDO_feature", name: "feature" },
  ];

  beforeEach(() => {
    originalGlobals = {
      core: global.core,
      github: global.github,
      context: global.context,
    };
    originalEnv = {
      staged: process.env.GH_AW_SAFE_OUTPUTS_STAGED,
      workflowName: process.env.GH_AW_WORKFLOW_NAME,
      workflowId: process.env.GH_AW_WORKFLOW_ID,
    };

    graphqlCalls = [];

    mockCore = {
      infos: /** @type {string[]} */ [],
      warnings: /** @type {string[]} */ [],
      errors: /** @type {string[]} */ [],
      info: /** @param {string} msg */ msg => mockCore.infos.push(msg),
      warning: /** @param {string} msg */ msg => mockCore.warnings.push(msg),
      error: /** @param {string} msg */ msg => mockCore.errors.push(msg),
      debug: vi.fn(),
      setOutput: vi.fn(),
      setFailed: vi.fn(),
    };

    mockGithub = {
      graphql: async (/** @type {string} */ query, /** @type {any} */ variables) => {
        graphqlCalls.push({ query, variables });

        // Fetch discussion by number (getDiscussionQuery)
        if (query.includes("discussion(number:")) {
          return {
            repository: {
              discussion: { ...defaultDiscussion },
            },
          };
        }

        // Fetch repo labels for fetchLabelNodeIds
        if (query.includes("labels(first: 100)") && query.includes("repository(owner:")) {
          return {
            repository: {
              labels: {
                nodes: defaultLabels,
              },
            },
          };
        }

        // Fetch discussion labels (for replaceDiscussionLabels removeQuery)
        if (query.includes("node(id:") && query.includes("on Discussion")) {
          return {
            node: {
              labels: {
                nodes: [],
              },
            },
          };
        }

        // updateDiscussion mutation
        if (query.includes("updateDiscussion")) {
          return {
            updateDiscussion: {
              discussion: {
                ...defaultDiscussion,
                title: variables.title || defaultDiscussion.title,
                body: variables.body || defaultDiscussion.body,
              },
            },
          };
        }

        // addLabelsToLabelable mutation
        if (query.includes("addLabelsToLabelable")) {
          return { addLabelsToLabelable: { clientMutationId: "test" } };
        }

        // removeLabelsFromLabelable mutation
        if (query.includes("removeLabelsFromLabelable")) {
          return { removeLabelsFromLabelable: { clientMutationId: "test" } };
        }

        throw new Error(`Unexpected GraphQL query: ${query.substring(0, 80)}`);
      },
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
    process.env.GH_AW_WORKFLOW_NAME = "Test Workflow";
    process.env.GH_AW_WORKFLOW_ID = "test-workflow";
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
    if (originalEnv.workflowName === undefined) {
      delete process.env.GH_AW_WORKFLOW_NAME;
    } else {
      process.env.GH_AW_WORKFLOW_NAME = originalEnv.workflowName;
    }
    if (originalEnv.workflowId === undefined) {
      delete process.env.GH_AW_WORKFLOW_ID;
    } else {
      process.env.GH_AW_WORKFLOW_ID = originalEnv.workflowId;
    }
  });

  // Helper to collect updateDiscussion mutation calls
  function getUpdateDiscussionMutations() {
    return graphqlCalls.filter(c => c.query.includes("updateDiscussion("));
  }

  // Helper to collect label mutation calls
  function getAddLabelsCalls() {
    return graphqlCalls.filter(c => c.query.includes("addLabelsToLabelable"));
  }

  function getRemoveLabelsCalls() {
    return graphqlCalls.filter(c => c.query.includes("removeLabelsFromLabelable"));
  }

  describe("labels-only update (the bug scenario)", () => {
    it("should update labels without calling updateDiscussion mutation when only labels are configured", async () => {
      // This is the exact scenario from the bug report:
      // safe-outputs: update-discussion: target: "*" allowed-labels: [Label1, Label2]
      const handler = await main({
        target: "*",
        allow_labels: true,
        allowed_labels: ["Label1", "Label2"],
      });

      const result = await handler({ type: "update_discussion", labels: ["Label1", "Label2"], discussion_number: 42 }, {});
      expect(result.success).toBe(true);

      // CRITICAL: The updateDiscussion mutation must NOT have been called
      // (body/title must not be modified when only labels need updating)
      const updateMutations = getUpdateDiscussionMutations();
      expect(updateMutations).toHaveLength(0);

      // Labels should have been updated
      const addCalls = getAddLabelsCalls();
      expect(addCalls).toHaveLength(1);
      expect(mockCore.infos.some(msg => msg.includes("Successfully replaced labels"))).toBe(true);
    });

    it("should reject body update when only labels are configured", async () => {
      // Bug scenario: AI passes body instead of labels because schema only showed 'body'
      // (before the fix, the schema didn't have a 'labels' field, so the AI used 'body')
      const handler = await main({
        target: "*",
        allow_labels: true,
        allowed_labels: ["Label1", "Label2"],
      });

      const result = await handler({ type: "update_discussion", body: '{"add_labels": ["Label3", "Label4"]}', discussion_number: 42 }, {});
      // Body update must be rejected since allow_body is not set
      expect(result.success).toBe(false);
      expect(result.error).toContain("Body updates are not allowed");

      // The discussion body must NOT have been changed
      const updateMutations = getUpdateDiscussionMutations();
      expect(updateMutations).toHaveLength(0);
    });

    it("should reject title update when only labels are configured", async () => {
      const handler = await main({
        target: "*",
        allow_labels: true,
        allowed_labels: ["Label1", "Label2"],
      });

      const result = await handler({ type: "update_discussion", title: "New Title", discussion_number: 42 }, {});
      expect(result.success).toBe(false);
      expect(result.error).toContain("Title updates are not allowed");

      const updateMutations = getUpdateDiscussionMutations();
      expect(updateMutations).toHaveLength(0);
    });

    it("should fail when all requested labels are not in the allowed list", async () => {
      const handler = await main({
        target: "*",
        allow_labels: true,
        allowed_labels: ["Label1", "Label2"],
      });

      // Agent attempts to set Label3 and Label4 which are NOT in the allowed list
      // When ALL labels are disallowed, validateLabels returns an error
      const result = await handler({ type: "update_discussion", labels: ["Label3", "Label4"], discussion_number: 42 }, {});
      // All labels were filtered out → error is returned
      expect(result.success).toBe(false);
      expect(result.error).toContain("No valid labels found after sanitization");

      // The updateDiscussion mutation must NOT have been called
      const updateMutations = getUpdateDiscussionMutations();
      expect(updateMutations).toHaveLength(0);

      // No label mutations should have been called either
      expect(getAddLabelsCalls()).toHaveLength(0);
    });

    it("should apply only the allowed subset of requested labels", async () => {
      const handler = await main({
        target: "*",
        allow_labels: true,
        allowed_labels: ["Label1", "Label2"],
      });

      // Agent requests Label1 (allowed) and Label3 (not allowed)
      const result = await handler({ type: "update_discussion", labels: ["Label1", "Label3"], discussion_number: 42 }, {});
      expect(result.success).toBe(true);

      // Must not call updateDiscussion mutation
      expect(getUpdateDiscussionMutations()).toHaveLength(0);

      // Labels were replaced - only Label1 should be added
      const addCalls = getAddLabelsCalls();
      expect(addCalls).toHaveLength(1);
      // Label1 has id "LA_kwDO1"
      expect(addCalls[0].variables.labelIds).toEqual(["LA_kwDO1"]);
    });
  });

  describe("title-only update", () => {
    it("should update title without calling label mutations when only title is configured", async () => {
      const handler = await main({
        target: "*",
        allow_title: true,
      });

      const result = await handler({ type: "update_discussion", title: "New Title", discussion_number: 42 }, {});
      expect(result.success).toBe(true);

      // The updateDiscussion mutation MUST have been called with the new title
      const updateMutations = getUpdateDiscussionMutations();
      expect(updateMutations).toHaveLength(1);
      expect(updateMutations[0].variables.title).toBe("New Title");
      // Body should preserve the existing value (not change it)
      expect(updateMutations[0].variables.body).toBe(defaultDiscussion.body);

      // No label mutations should have been called
      expect(getAddLabelsCalls()).toHaveLength(0);
      expect(getRemoveLabelsCalls()).toHaveLength(0);
    });

    it("should reject body update when only title is configured", async () => {
      const handler = await main({
        target: "*",
        allow_title: true,
      });

      const result = await handler({ type: "update_discussion", body: "New body", discussion_number: 42 }, {});
      expect(result.success).toBe(false);
      expect(result.error).toContain("Body updates are not allowed");

      expect(getUpdateDiscussionMutations()).toHaveLength(0);
    });

    it("should reject label update when only title is configured", async () => {
      const handler = await main({
        target: "*",
        allow_title: true,
      });

      const result = await handler({ type: "update_discussion", labels: ["bug"], discussion_number: 42 }, {});
      // labels update is silently skipped (allow_labels is not set, so the labels block is ignored)
      // The result is a no-op (no update fields)
      expect(result.success).toBe(true);
      expect(getUpdateDiscussionMutations()).toHaveLength(0);
      expect(getAddLabelsCalls()).toHaveLength(0);
    });
  });

  describe("body-only update", () => {
    it("should update body without modifying title or labels", async () => {
      const handler = await main({
        target: "*",
        allow_body: true,
      });

      const result = await handler({ type: "update_discussion", body: "Updated body content", discussion_number: 42 }, {});
      expect(result.success).toBe(true);

      // The updateDiscussion mutation MUST have been called with the new body
      const updateMutations = getUpdateDiscussionMutations();
      expect(updateMutations).toHaveLength(1);
      expect(updateMutations[0].variables.body).toContain("Updated body content");
      // Title should preserve the existing value (not change it)
      expect(updateMutations[0].variables.title).toBe(defaultDiscussion.title);

      // No label mutations should have been called
      expect(getAddLabelsCalls()).toHaveLength(0);
      expect(getRemoveLabelsCalls()).toHaveLength(0);
    });

    it("should reject title update when only body is configured", async () => {
      const handler = await main({
        target: "*",
        allow_body: true,
      });

      const result = await handler({ type: "update_discussion", title: "New Title", discussion_number: 42 }, {});
      expect(result.success).toBe(false);
      expect(result.error).toContain("Title updates are not allowed");

      expect(getUpdateDiscussionMutations()).toHaveLength(0);
    });
  });

  describe("title and body update (no labels)", () => {
    it("should update title and body without touching labels", async () => {
      const handler = await main({
        target: "*",
        allow_title: true,
        allow_body: true,
      });

      const result = await handler({ type: "update_discussion", title: "New Title", body: "New body content", discussion_number: 42 }, {});
      expect(result.success).toBe(true);

      const updateMutations = getUpdateDiscussionMutations();
      expect(updateMutations).toHaveLength(1);
      expect(updateMutations[0].variables.title).toBe("New Title");
      expect(updateMutations[0].variables.body).toContain("New body content");

      // No label mutations
      expect(getAddLabelsCalls()).toHaveLength(0);
    });

    it("should use existing title when only body is provided even with allow_title", async () => {
      const handler = await main({
        target: "*",
        allow_title: true,
        allow_body: true,
      });

      const result = await handler({ type: "update_discussion", body: "Body only update", discussion_number: 42 }, {});
      expect(result.success).toBe(true);

      const updateMutations = getUpdateDiscussionMutations();
      expect(updateMutations).toHaveLength(1);
      // Title should be preserved from the fetched discussion
      expect(updateMutations[0].variables.title).toBe(defaultDiscussion.title);
      expect(updateMutations[0].variables.body).toContain("Body only update");
    });
  });

  describe("title and body field isolation (independent updates)", () => {
    it("updating only title must not mutate the body", async () => {
      // Both title and body are allowed, but only title is provided.
      // The body sent to the API must equal the existing body exactly (not a new value).
      const handler = await main({
        target: "*",
        allow_title: true,
        allow_body: true,
      });

      const result = await handler({ type: "update_discussion", title: "Updated Title Only", discussion_number: 42 }, {});
      expect(result.success).toBe(true);

      const updateMutations = getUpdateDiscussionMutations();
      expect(updateMutations).toHaveLength(1);
      // Title was changed
      expect(updateMutations[0].variables.title).toBe("Updated Title Only");
      // Body was NOT changed — must be the original value fetched from the discussion
      expect(updateMutations[0].variables.body).toBe(defaultDiscussion.body);
    });

    it("updating only body must not mutate the title", async () => {
      // Both title and body are allowed, but only body is provided.
      // The title sent to the API must equal the existing title exactly (not a new value).
      const handler = await main({
        target: "*",
        allow_title: true,
        allow_body: true,
      });

      const result = await handler({ type: "update_discussion", body: "Updated body only", discussion_number: 42 }, {});
      expect(result.success).toBe(true);

      const updateMutations = getUpdateDiscussionMutations();
      expect(updateMutations).toHaveLength(1);
      // Title was NOT changed — must be the original value fetched from the discussion
      expect(updateMutations[0].variables.title).toBe(defaultDiscussion.title);
      // Body was changed
      expect(updateMutations[0].variables.body).toContain("Updated body only");
    });

    it("updating title independently does not affect a subsequent body-only update", async () => {
      // Simulate two separate handler calls (separate safe output messages).
      // First updates title; second updates body. Each must preserve the other.
      const handler = await main({
        target: "*",
        allow_title: true,
        allow_body: true,
      });

      // First call: update title only
      const result1 = await handler({ type: "update_discussion", title: "First Title Update", discussion_number: 42 }, {});
      expect(result1.success).toBe(true);

      let mutations = getUpdateDiscussionMutations();
      expect(mutations).toHaveLength(1);
      expect(mutations[0].variables.title).toBe("First Title Update");
      expect(mutations[0].variables.body).toBe(defaultDiscussion.body);

      // Clear tracked calls for the second assertion
      graphqlCalls.length = 0;

      // Second call: update body only (uses a fresh fetch of the discussion)
      const result2 = await handler({ type: "update_discussion", body: "Second Body Update", discussion_number: 42 }, {});
      expect(result2.success).toBe(true);

      mutations = getUpdateDiscussionMutations();
      expect(mutations).toHaveLength(1);
      // Title still equals the original (from fetch) — not the value written in call 1
      expect(mutations[0].variables.title).toBe(defaultDiscussion.title);
      expect(mutations[0].variables.body).toContain("Second Body Update");
    });
  });

  describe("all fields allowed", () => {
    it("should update title, body, and labels when all are configured", async () => {
      const handler = await main({
        target: "*",
        allow_title: true,
        allow_body: true,
        allow_labels: true,
        allowed_labels: ["Label1", "bug"],
      });

      const result = await handler({ type: "update_discussion", title: "New Title", body: "New body content", labels: ["Label1", "bug"], discussion_number: 42 }, {});
      expect(result.success).toBe(true);

      // updateDiscussion mutation should be called for title/body
      const updateMutations = getUpdateDiscussionMutations();
      expect(updateMutations).toHaveLength(1);
      expect(updateMutations[0].variables.title).toBe("New Title");
      expect(updateMutations[0].variables.body).toContain("New body content");

      // Label mutations should be called too
      const addCalls = getAddLabelsCalls();
      expect(addCalls).toHaveLength(1);
    });

    it("should update only labels when only labels field is provided even with all fields allowed", async () => {
      const handler = await main({
        target: "*",
        allow_title: true,
        allow_body: true,
        allow_labels: true,
      });

      const result = await handler({ type: "update_discussion", labels: ["bug"], discussion_number: 42 }, {});
      expect(result.success).toBe(true);

      // updateDiscussion mutation must NOT be called when only labels are provided
      // This is the key behavioral fix: label-only updates don't touch title/body
      const updateMutations = getUpdateDiscussionMutations();
      expect(updateMutations).toHaveLength(0);

      // Labels should be updated
      const addCalls = getAddLabelsCalls();
      expect(addCalls).toHaveLength(1);
    });

    it("should update only title when only title field is provided even with all fields allowed", async () => {
      const handler = await main({
        target: "*",
        allow_title: true,
        allow_body: true,
        allow_labels: true,
      });

      const result = await handler({ type: "update_discussion", title: "Title Only Update", discussion_number: 42 }, {});
      expect(result.success).toBe(true);

      // Only updateDiscussion mutation should be called, no label mutations
      const updateMutations = getUpdateDiscussionMutations();
      expect(updateMutations).toHaveLength(1);
      expect(updateMutations[0].variables.title).toBe("Title Only Update");
      expect(updateMutations[0].variables.body).toBe(defaultDiscussion.body);

      expect(getAddLabelsCalls()).toHaveLength(0);
      expect(getRemoveLabelsCalls()).toHaveLength(0);
    });
  });

  describe("no fields configured", () => {
    it("should reject body update when no fields are configured", async () => {
      const handler = await main({
        target: "*",
        // No allow_title, allow_body, or allow_labels
      });

      const result = await handler({ type: "update_discussion", body: "New body", discussion_number: 42 }, {});
      expect(result.success).toBe(false);
      expect(result.error).toContain("Body updates are not allowed");

      expect(getUpdateDiscussionMutations()).toHaveLength(0);
    });

    it("should reject title update when no fields are configured", async () => {
      const handler = await main({
        target: "*",
        // No allow_title, allow_body, or allow_labels
      });

      const result = await handler({ type: "update_discussion", title: "New title", discussion_number: 42 }, {});
      expect(result.success).toBe(false);
      expect(result.error).toContain("Title updates are not allowed");

      expect(getUpdateDiscussionMutations()).toHaveLength(0);
    });
  });

  describe("triggering context", () => {
    it("should use triggering discussion number when not explicitly provided", async () => {
      const handler = await main({
        target: "triggering",
        allow_labels: true,
      });

      const result = await handler({ type: "update_discussion", labels: ["bug"] }, {});
      expect(result.success).toBe(true);

      // Should have fetched discussion #42 from context
      const fetchCalls = graphqlCalls.filter(c => c.query.includes("discussion(number:"));
      expect(fetchCalls.length).toBeGreaterThan(0);
      expect(fetchCalls[0].variables.number).toBe(42);
    });

    it("should use explicit discussion_number when provided with target *", async () => {
      const handler = await main({
        target: "*",
        allow_labels: true,
      });

      const result = await handler({ type: "update_discussion", labels: ["bug"], discussion_number: 99 }, {});
      expect(result.success).toBe(true);

      const fetchCalls = graphqlCalls.filter(c => c.query.includes("discussion(number:"));
      expect(fetchCalls.length).toBeGreaterThan(0);
      expect(fetchCalls[0].variables.number).toBe(99);
    });
  });

  describe("temporary ID support", () => {
    it("should resolve a temporary ID for discussion_number", async () => {
      const handler = await main({
        target: "*",
        allow_labels: true,
      });

      const resolvedTemporaryIds = { aw_disc1: { repo: "testowner/testrepo", number: 99 } };
      const result = await handler({ type: "update_discussion", labels: ["bug"], discussion_number: "aw_disc1" }, resolvedTemporaryIds);
      expect(result.success).toBe(true);

      const fetchCalls = graphqlCalls.filter(c => c.query.includes("discussion(number:"));
      expect(fetchCalls.length).toBeGreaterThan(0);
      expect(fetchCalls[0].variables.number).toBe(99);
      expect(mockCore.infos.some(msg => msg.includes("aw_disc1") && msg.includes("99"))).toBe(true);
    });

    it("should resolve a temporary ID with # prefix for discussion_number", async () => {
      const handler = await main({
        target: "*",
        allow_labels: true,
      });

      const resolvedTemporaryIds = { aw_disc1: { repo: "testowner/testrepo", number: 99 } };
      const result = await handler({ type: "update_discussion", labels: ["bug"], discussion_number: "#aw_disc1" }, resolvedTemporaryIds);
      expect(result.success).toBe(true);

      const fetchCalls = graphqlCalls.filter(c => c.query.includes("discussion(number:"));
      expect(fetchCalls.length).toBeGreaterThan(0);
      expect(fetchCalls[0].variables.number).toBe(99);
    });

    it("should fail when temporary ID is not found in the map", async () => {
      const handler = await main({
        target: "*",
        allow_labels: true,
      });

      const result = await handler({ type: "update_discussion", labels: ["bug"], discussion_number: "aw_disc1" }, {});
      expect(result.success).toBe(false);
      expect(result.error).toContain("aw_disc1");
    });
  });

  describe("main factory", () => {
    it("should return a handler function", async () => {
      const handler = await main({ allow_labels: true });
      expect(typeof handler).toBe("function");
    });

    it("should return a handler function with empty config", async () => {
      const handler = await main();
      expect(typeof handler).toBe("function");
    });

    it("should return a handler function with labels-only config (bug scenario config)", async () => {
      // Corresponds to: safe-outputs: update-discussion: target: "*" allowed-labels: [Label1, Label2]
      const handler = await main({
        target: "*",
        allow_labels: true,
        allowed_labels: ["Label1", "Label2"],
      });
      expect(typeof handler).toBe("function");
    });
  });

  describe("GraphQL error logging", () => {
    it("should log detailed error info and re-throw when fetch query fails", async () => {
      const graphqlError = Object.assign(new Error("Request failed due to following response errors"), {
        errors: [{ type: "INSUFFICIENT_SCOPES", message: "Your token has not been granted the required scopes.", path: ["repository", "discussion"] }],
        status: 401,
      });
      mockGithub.graphql = vi.fn().mockRejectedValue(graphqlError);

      const handler = await main({ target: "*", allow_body: true });
      const result = await handler({ type: "update_discussion", body: "New body", discussion_number: 45 }, {});

      expect(result.success).toBe(false);

      // Verify logGraphQLError was called: it emits the operation name, message, and permission hint
      expect(mockCore.infos.some(msg => msg.includes("GraphQL error during:"))).toBe(true);
      expect(mockCore.infos.some(msg => msg.includes("fetch discussion #45"))).toBe(true);
      expect(mockCore.infos.some(msg => msg.includes("Request failed"))).toBe(true);
      expect(mockCore.infos.some(msg => msg.includes("discussions: write"))).toBe(true);
      expect(mockCore.infos.some(msg => msg.includes("INSUFFICIENT_SCOPES"))).toBe(true);
    });

    it("should log detailed error info and re-throw when updateDiscussion mutation fails", async () => {
      const mutationError = Object.assign(new Error("Request failed due to following response errors"), {
        errors: [{ type: "FORBIDDEN", message: "Resource not accessible by integration", path: ["updateDiscussion"] }],
        status: 403,
      });

      mockGithub.graphql = vi.fn().mockImplementation(async query => {
        if (query.includes("discussion(number:")) {
          return { repository: { discussion: { ...defaultDiscussion } } };
        }
        throw mutationError;
      });

      const handler = await main({ target: "*", allow_body: true });
      const result = await handler({ type: "update_discussion", body: "Updated body", discussion_number: 45 }, {});

      expect(result.success).toBe(false);

      // Verify logGraphQLError was called with the mutation operation name
      expect(mockCore.infos.some(msg => msg.includes("updateDiscussion mutation"))).toBe(true);
      expect(mockCore.infos.some(msg => msg.includes("discussion #45"))).toBe(true);
      expect(mockCore.infos.some(msg => msg.includes("HTTP status: 403"))).toBe(true);
      expect(mockCore.infos.some(msg => msg.includes("FORBIDDEN"))).toBe(true);
    });

    it("should log NOT_FOUND hint when discussion fetch returns NOT_FOUND error", async () => {
      const notFoundError = Object.assign(new Error("Could not resolve to a Discussion"), {
        errors: [{ type: "NOT_FOUND", message: "Could not resolve to a Discussion with the number of 45.", path: ["repository", "discussion"] }],
        status: 200,
      });
      mockGithub.graphql = vi.fn().mockRejectedValue(notFoundError);

      const handler = await main({ target: "*", allow_body: true });
      const result = await handler({ type: "update_discussion", body: "New body", discussion_number: 45 }, {});

      expect(result.success).toBe(false);
      expect(mockCore.infos.some(msg => msg.includes("NOT_FOUND"))).toBe(true);
      expect(mockCore.infos.some(msg => msg.includes("Check that the discussion number is correct"))).toBe(true);
    });
  });
});
