// @ts-check
import { describe, it, expect, beforeEach, vi } from "vitest";
const { main, deterministicLabelColor } = require("./replace_label.cjs");

describe("replace_label", () => {
  let mockCore;
  let mockGithub;
  let mockContext;

  beforeEach(() => {
    mockCore = {
      info: () => {},
      warning: () => {},
      error: () => {},
      debug: () => {},
      messages: [],
      infos: [],
      warnings: [],
      errors: [],
    };

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
          get: async () => ({
            data: {
              title: "Test issue title",
              labels: [
                { name: "in-progress", node_id: "LA_in_progress_123" },
                { name: "bug", node_id: "LA_bug_456" },
              ],
              node_id: "I_issue_789",
            },
          }),
          getLabel: async ({ name }) => {
            const labels = {
              done: { name: "done", node_id: "LA_done_789", color: "0075ca" },
              "in-progress": { name: "in-progress", node_id: "LA_in_progress_123", color: "e4e669" },
            };
            if (labels[name]) {
              return { data: labels[name] };
            }
            const err = new Error("Not Found");
            err.status = 404;
            throw err;
          },
          createLabel: async ({ name, color }) => ({
            data: { name, node_id: `LA_${name}_new`, color },
          }),
        },
      },
      graphql: async (mutation, variables) => {
        if (mutation.includes("ReplaceLabelMutation")) {
          return {
            removeLabels: { clientMutationId: null },
            addLabels: {
              labelable: {
                labels: {
                  nodes: [{ name: "done" }],
                },
              },
            },
          };
        }
        throw new Error("Unknown query");
      },
    };

    mockContext = {
      repo: {
        owner: "test-owner",
        repo: "test-repo",
      },
      payload: {
        issue: {
          number: 42,
        },
      },
    };

    global.core = mockCore;
    global.github = mockGithub;
    global.context = mockContext;
  });

  it("should replace label when both labels are valid", async () => {
    const handler = await main({ allowed_add: [], allowed_remove: [], blocked: [] });
    const result = await handler({ label_to_remove: "in-progress", label_to_add: "done" }, {});

    expect(result.success).toBe(true);
    expect(result.labelRemoved).toBe("in-progress");
    expect(result.labelAdded).toBe("done");
  });

  it("should create label_to_add if it does not exist in the repo", async () => {
    let createLabelCalled = false;
    mockGithub.rest.issues.getLabel = async ({ name }) => {
      const err = new Error("Not Found");
      err.status = 404;
      throw err; // All labels "not found"
    };
    mockGithub.rest.issues.createLabel = async ({ name, color }) => {
      createLabelCalled = true;
      return { data: { name, node_id: `LA_${name}_created`, color } };
    };

    const handler = await main({});
    const result = await handler({ label_to_remove: "in-progress", label_to_add: "needs-review" }, {});

    expect(result.success).toBe(true);
    expect(createLabelCalled).toBe(true);
  });

  it("should succeed even when label_to_remove is not present on the issue", async () => {
    mockGithub.rest.issues.get = async () => ({
      data: {
        title: "Test issue",
        labels: [{ name: "bug", node_id: "LA_bug_456" }], // "in-progress" not present
        node_id: "I_issue_789",
      },
    });

    const handler = await main({});
    const result = await handler({ label_to_remove: "in-progress", label_to_add: "done" }, {});

    expect(result.success).toBe(true);
    expect(result.labelRemoved).toBeNull();
    expect(result.labelAdded).toBe("done");
  });

  it("should return error when label_to_remove is missing", async () => {
    const handler = await main({});
    // @ts-ignore - testing missing field
    const result = await handler({ label_to_add: "done" }, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("label_to_remove");
  });

  it("should return error when label_to_add is missing", async () => {
    const handler = await main({});
    // @ts-ignore - testing missing field
    const result = await handler({ label_to_remove: "in-progress" }, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("label_to_add");
  });

  it("should reject label_to_add that is not in allowed-add list", async () => {
    const handler = await main({ allowed_add: ["approved", "done"] });
    const result = await handler({ label_to_remove: "in-progress", label_to_add: "wontfix" }, {});

    expect(result.success).toBe(false);
  });

  it("should reject label_to_remove that is not in allowed-remove list", async () => {
    const handler = await main({ allowed_remove: ["in-progress", "review-needed"] });
    const result = await handler({ label_to_remove: "bug", label_to_add: "done" }, {});

    expect(result.success).toBe(false);
  });

  it("should reject labels matching blocked patterns", async () => {
    const handler = await main({ blocked: ["~*"] });
    const result = await handler({ label_to_remove: "in-progress", label_to_add: "~internal" }, {});

    expect(result.success).toBe(false);
  });

  it("should skip when required-labels filter does not match", async () => {
    const handler = await main({ required_labels: ["approved"] });
    // Issue has "in-progress" and "bug" but not "approved"
    const result = await handler({ label_to_remove: "in-progress", label_to_add: "done" }, {});

    expect(result.success).toBe(false);
    expect(result.skipped).toBe(true);
  });

  it("should skip when required-title-prefix does not match", async () => {
    const handler = await main({ required_title_prefix: "[BUG]" });
    // Issue title is "Test issue title", does not start with "[BUG]"
    const result = await handler({ label_to_remove: "in-progress", label_to_add: "done" }, {});

    expect(result.success).toBe(false);
    expect(result.skipped).toBe(true);
  });

  it("should return staged result when in staged mode", async () => {
    const handler = await main({ staged: true });
    const result = await handler({ label_to_remove: "in-progress", label_to_add: "done" }, {});

    expect(result.success).toBe(true);
    expect(result.staged).toBe(true);
    expect(result.previewInfo?.labelToRemove).toBe("in-progress");
    expect(result.previewInfo?.labelToAdd).toBe("done");
  });

  it("should return error when no item number is available", async () => {
    global.context = {
      repo: { owner: "test-owner", repo: "test-repo" },
      payload: {}, // no issue or pull_request in payload
    };

    const handler = await main({});
    const result = await handler({ label_to_remove: "in-progress", label_to_add: "done" }, {});

    expect(result.success).toBe(false);
  });
});

describe("deterministicLabelColor", () => {
  it("should return a 6-char hex string", () => {
    const color = deterministicLabelColor("done");
    expect(color).toMatch(/^[0-9a-f]{6}$/);
  });

  it("should return different colors for different labels", () => {
    const c1 = deterministicLabelColor("done");
    const c2 = deterministicLabelColor("in-progress");
    expect(c1).not.toBe(c2);
  });

  it("should return the same color for the same label", () => {
    const c1 = deterministicLabelColor("needs-review");
    const c2 = deterministicLabelColor("needs-review");
    expect(c1).toBe(c2);
  });

  it("should return pastel colors (128-191 per channel)", () => {
    for (const name of ["done", "in-progress", "approved", "needs-review", "blocked"]) {
      const hex = deterministicLabelColor(name);
      const r = parseInt(hex.slice(0, 2), 16);
      const g = parseInt(hex.slice(2, 4), 16);
      const b = parseInt(hex.slice(4, 6), 16);
      expect(r).toBeGreaterThanOrEqual(128);
      expect(r).toBeLessThanOrEqual(191);
      expect(g).toBeGreaterThanOrEqual(128);
      expect(g).toBeLessThanOrEqual(191);
      expect(b).toBeGreaterThanOrEqual(128);
      expect(b).toBeLessThanOrEqual(191);
    }
  });
});
