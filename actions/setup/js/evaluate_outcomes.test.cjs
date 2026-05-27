import { describe, it, expect } from "vitest";

const { evaluateItem } = require("./evaluate_outcomes.cjs");

function createAPIStub(fixtures) {
  return endpoint => fixtures[endpoint] ?? null;
}

describe("evaluate_outcomes type-specific evaluators", () => {
  it("evaluates create_issue outcomes using engagement signals", () => {
    const nowMs = Date.parse("2026-05-27T04:00:00Z");
    const item = {
      type: "create_issue",
      url: "https://github.com/acme/repo/issues/12",
      timestamp: "2026-05-25T04:00:00Z",
    };
    const ghAPI = createAPIStub({
      "repos/acme/repo/issues/12": { state: "open", comments: 0, reactions: { total_count: 0 }, user: { login: "author" } },
      "repos/acme/repo/issues/12/comments": [],
      "repos/acme/repo/issues/12/timeline": [{ event: "cross-referenced", source: { issue: { number: 88 } } }],
      "repos/acme/repo/pulls/88": { merged: true },
    });
    expect(evaluateItem(item, "acme/repo", { ghAPI, nowMs }).detail).toBe("accepted:strong");

    const noEngagement = createAPIStub({
      "repos/acme/repo/issues/12": { state: "open", comments: 0, reactions: { total_count: 0 }, user: { login: "author" } },
      "repos/acme/repo/issues/12/comments": [],
      "repos/acme/repo/issues/12/timeline": [],
    });
    expect(evaluateItem(item, "acme/repo", { ghAPI: noEngagement, nowMs }).result).toBe("pending");
  });

  it("handles missing create_issue reaction/comment fields without existence-only acceptance", () => {
    const item = {
      type: "create_issue",
      url: "https://github.com/acme/repo/issues/12",
      timestamp: "2026-05-26T04:00:00Z",
    };
    const ghAPI = createAPIStub({
      "repos/acme/repo/issues/12": { state: "open", user: { login: "author" }, comments: "unknown", reactions: null },
      "repos/acme/repo/issues/12/comments": [],
      "repos/acme/repo/issues/12/timeline": [],
    });
    expect(evaluateItem(item, "acme/repo", { ghAPI, nowMs: Date.parse("2026-05-27T04:00:00Z") }).result).toBe("pending");
  });

  it("classifies create_issue immediate close and close-without-activity as rejected", () => {
    const item = {
      type: "create_issue",
      url: "https://github.com/acme/repo/issues/7",
      timestamp: "2026-05-26T00:00:00Z",
    };
    const immediateClose = createAPIStub({
      "repos/acme/repo/issues/7": {
        state: "closed",
        comments: 0,
        reactions: { total_count: 0 },
        created_at: "2026-05-26T00:00:00Z",
        closed_at: "2026-05-26T00:05:00Z",
        user: { login: "author" },
      },
      "repos/acme/repo/issues/7/comments": [],
      "repos/acme/repo/issues/7/timeline": [{ event: "closed", actor: { login: "reviewer" } }],
    });
    expect(evaluateItem(item, "acme/repo", { ghAPI: immediateClose }).detail).toBe("rejected:strong");

    const closedNoActivity = createAPIStub({
      "repos/acme/repo/issues/7": {
        state: "closed",
        comments: 0,
        reactions: { total_count: 0 },
        created_at: "2026-05-26T00:00:00Z",
        closed_at: "2026-05-27T00:00:00Z",
        user: { login: "author" },
      },
      "repos/acme/repo/issues/7/comments": [],
      "repos/acme/repo/issues/7/timeline": [],
    });
    expect(evaluateItem(item, "acme/repo", { ghAPI: closedNoActivity }).detail).toBe("rejected:medium");
  });

  it("evaluates add_comment deletion, engagement, pending, and unknown", () => {
    const base = {
      type: "add_comment",
      url: "https://github.com/acme/repo/issues/21#issuecomment-1001",
      timestamp: "2026-05-26T00:00:00Z",
    };

    const deleted = createAPIStub({
      "repos/acme/repo/issues/comments/1001": null,
    });
    expect(evaluateItem(base, "acme/repo", { ghAPI: deleted }).detail).toContain("rejected:strong");

    const reacted = createAPIStub({
      "repos/acme/repo/issues/comments/1001": { id: 1001, created_at: "2026-05-26T00:00:00Z", user: { login: "copilot" }, reactions: { total_count: 2 } },
      "repos/acme/repo/issues/21/comments": [],
    });
    expect(evaluateItem(base, "acme/repo", { ghAPI: reacted }).detail).toBe("accepted:strong");

    const actedOnThread = createAPIStub({
      "repos/acme/repo/issues/comments/1001": { id: 1001, created_at: "2026-05-26T00:00:00Z", user: { login: "copilot" }, reactions: { total_count: 0 } },
      "repos/acme/repo/issues/21/comments": [{ created_at: "2026-05-26T00:01:00Z", user: { login: "copilot" }, body: "follow-up" }],
    });
    expect(evaluateItem(base, "acme/repo", { ghAPI: actedOnThread }).detail).toBe("accepted:medium");

    const pending = createAPIStub({
      "repos/acme/repo/issues/comments/1001": { id: 1001, created_at: "2026-05-26T00:00:00Z", user: { login: "copilot" }, reactions: { total_count: 0 } },
      "repos/acme/repo/issues/21/comments": [],
    });
    expect(evaluateItem(base, "acme/repo", { ghAPI: pending }).result).toBe("pending");

    expect(evaluateItem({ type: "add_comment", url: "https://github.com/acme/repo/issues/21" }, "acme/repo", { ghAPI: pending }).result).toBe("unknown");
  });

  it("evaluates add_labels retention using persisted before-state labels", () => {
    const item = {
      type: "add_labels",
      url: "https://github.com/acme/repo/issues/42",
      timestamp: "2026-05-25T00:00:00Z",
      labelsBefore: ["bug"],
      labelsAdded: ["bug", "triage"],
    };

    const retained = createAPIStub({
      "repos/acme/repo/issues/42/labels": [{ name: "bug" }, { name: "triage" }],
    });
    expect(evaluateItem(item, "acme/repo", { ghAPI: retained, nowMs: Date.parse("2026-05-27T00:00:00Z") }).detail).toBe("accepted:strong");

    const removedByNonAuthor = createAPIStub({
      "repos/acme/repo/issues/42/labels": [{ name: "bug" }],
      "repos/acme/repo/issues/42": { user: { login: "copilot" } },
      "repos/acme/repo/issues/42/events": [{ event: "unlabeled", actor: { login: "maintainer" }, label: { name: "triage" } }],
    });
    expect(evaluateItem(item, "acme/repo", { ghAPI: removedByNonAuthor, nowMs: Date.parse("2026-05-27T00:00:00Z") }).detail).toBe("rejected:strong");

    expect(evaluateItem(item, "acme/repo", { ghAPI: retained, nowMs: Date.parse("2026-05-25T00:01:00Z") }).result).toBe("pending");
    expect(evaluateItem({ ...item, labelsBefore: [] }, "acme/repo", { ghAPI: retained, nowMs: Date.parse("2026-05-27T00:00:00Z") }).result).toBe("unknown");
  });
});
