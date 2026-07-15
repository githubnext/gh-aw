import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { createRequire } from "module";
import crypto from "crypto";

const req = createRequire(import.meta.url);
const { evaluateItem, normalizeOutcome } = req("./evaluate_outcomes.cjs");

function hashBody(body) {
  return crypto
    .createHash("sha256")
    .update(
      String(body || "")
        .replace(/\r\n/g, "\n")
        .replace(/[ \t]+\n/g, "\n")
        .trim(),
      "utf8"
    )
    .digest("hex");
}

/**
 * @param {Record<string, any>} apiResponses
 * @returns {(endpoint: string) => any}
 */
function mockAPI(apiResponses) {
  return endpoint => {
    if (!(endpoint in apiResponses)) {
      return null;
    }
    return apiResponses[endpoint];
  };
}

const createAPIStub = mockAPI;

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

    const commentApiError = createAPIStub({
      "repos/acme/repo/issues/comments/1001": null,
    });
    expect(evaluateItem(base, "acme/repo", { ghAPI: commentApiError }).result).toBe("unknown");

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
      number: 42,
      repo: "acme/repo",
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
    expect(evaluateItem({ ...item, labelsBefore: [] }, "acme/repo", { ghAPI: retained, nowMs: Date.parse("2026-05-27T00:00:00Z") }).detail).toBe("accepted:strong");
  });

  it("evaluates add_labels retention using production before_state.labels manifest shape", () => {
    // Production manifest from add_labels.cjs stores labels in before_state.labels, not labelsBefore.
    const item = {
      type: "add_labels",
      number: 42,
      repo: "acme/repo",
      timestamp: "2026-05-25T00:00:00Z",
      before_state: { labels: ["bug"] },
      labelsAdded: ["bug", "triage"],
    };

    const retained = createAPIStub({
      "repos/acme/repo/issues/42/labels": [{ name: "bug" }, { name: "triage" }],
    });
    expect(evaluateItem(item, "acme/repo", { ghAPI: retained, nowMs: Date.parse("2026-05-27T00:00:00Z") }).detail).toBe("accepted:strong");
  });

  it("returns unknown for add_labels when the item number is missing", () => {
    const missingNumber = evaluateItem(
      {
        type: "add_labels",
        repo: "acme/repo",
        timestamp: "2026-05-25T00:00:00Z",
        labelsBefore: ["bug"],
        labelsAdded: ["bug", "triage"],
      },
      "acme/repo",
      {
        ghAPI: createAPIStub({
          "repos/acme/repo/issues/42/labels": [{ name: "bug" }, { name: "triage" }],
        }),
        nowMs: Date.parse("2026-05-27T00:00:00Z"),
      }
    );
    expect(missingNumber.detail).toBe("unknown: issue number not found");
  });

  it("evaluates close_issue using persisted repo/number metadata", () => {
    const item = {
      type: "close_issue",
      number: 77,
      repo: "acme/repo",
      timestamp: "2026-05-26T00:00:00Z",
    };

    const closed = createAPIStub({
      "repos/acme/repo/issues/77": {
        state: "closed",
        comments: 1,
        reactions: { total_count: 2 },
        created_at: "2026-05-26T00:00:00Z",
        closed_at: "2026-05-26T00:10:00Z",
      },
    });
    expect(evaluateItem(item, "acme/repo", { ghAPI: closed })).toMatchObject({
      result: "accepted",
      outcome_status: "accepted",
      evidence_strength: "strong",
      signal: "closed",
      detail: "closed",
    });

    const open = createAPIStub({
      "repos/acme/repo/issues/77": { state: "open", comments: 1, reactions: { total_count: 0 } },
    });
    expect(evaluateItem(item, "acme/repo", { ghAPI: open })).toMatchObject({
      result: "rejected",
      outcome_status: "rejected",
      evidence_strength: "strong",
      signal: "not_closed",
      detail: "not_closed",
    });
  });

  it("returns unknown for close_issue when the item reference is missing", () => {
    const missingRef = evaluateItem({ type: "close_issue", timestamp: "2026-05-26T00:00:00Z" }, "acme/repo", { ghAPI: createAPIStub({}), nowMs: Date.parse("2026-05-27T00:00:00Z") });
    expect(missingRef.result).toBe("unknown");
    expect(missingRef.detail).toBe("missing issue reference");
  });

  it("returns unknown for close_issue on API error and sets pending_age_sec", () => {
    const nowMs = Date.parse("2026-05-27T00:00:00Z");
    const apiError = evaluateItem({ type: "close_issue", number: 77, repo: "acme/repo", timestamp: "2026-05-26T00:00:00Z" }, "acme/repo", { ghAPI: createAPIStub({}), nowMs });
    expect(apiError.result).toBe("unknown");
    expect(apiError.detail).toBe("api error");
    expect(typeof apiError.pending_age_sec).toBe("number");
  });

  it("evaluates close_pull_request using persisted repo/number metadata", () => {
    const item = {
      type: "close_pull_request",
      number: 55,
      repo: "acme/repo",
      timestamp: "2026-05-26T00:00:00Z",
    };

    const closed = createAPIStub({
      "repos/acme/repo/pulls/55": {
        state: "closed",
        merged: false,
        review_comments: 0,
        changed_files: 1,
        additions: 3,
        deletions: 1,
        comments: 0,
        created_at: "2026-05-26T00:00:00Z",
        closed_at: "2026-05-26T00:10:00Z",
      },
    });
    expect(evaluateItem(item, "acme/repo", { ghAPI: closed })).toMatchObject({
      result: "accepted",
      outcome_status: "accepted",
      evidence_strength: "strong",
      signal: "closed",
      detail: "closed",
    });

    const merged = createAPIStub({
      "repos/acme/repo/pulls/55": {
        state: "closed",
        merged: true,
        review_comments: 0,
        changed_files: 1,
        additions: 3,
        deletions: 1,
        comments: 0,
        created_at: "2026-05-26T00:00:00Z",
        merged_at: "2026-05-26T00:10:00Z",
      },
    });
    expect(evaluateItem(item, "acme/repo", { ghAPI: merged })).toMatchObject({
      result: "rejected",
      outcome_status: "rejected",
      evidence_strength: "strong",
      signal: "closed_by_merge",
      detail: "merged",
    });

    const open = createAPIStub({
      "repos/acme/repo/pulls/55": { state: "open", merged: false, review_comments: 0 },
    });
    expect(evaluateItem(item, "acme/repo", { ghAPI: open })).toMatchObject({
      result: "rejected",
      outcome_status: "rejected",
      evidence_strength: "strong",
      signal: "not_closed",
      detail: "not_closed",
    });
  });

  it("detects merged PR via merged_at when merged flag is absent", () => {
    const item = {
      type: "close_pull_request",
      number: 55,
      repo: "acme/repo",
      timestamp: "2026-05-26T00:00:00Z",
    };
    const mergedAtOnly = createAPIStub({
      "repos/acme/repo/pulls/55": {
        state: "closed",
        merged: false,
        merged_at: "2026-05-26T00:10:00Z",
        created_at: "2026-05-26T00:00:00Z",
      },
    });
    expect(evaluateItem(item, "acme/repo", { ghAPI: mergedAtOnly })).toMatchObject({
      result: "rejected",
      signal: "closed_by_merge",
      detail: "merged",
    });
  });

  it("returns unknown for close_pull_request when the item reference is missing", () => {
    const missingRef = evaluateItem({ type: "close_pull_request", timestamp: "2026-05-26T00:00:00Z" }, "acme/repo", { ghAPI: createAPIStub({}), nowMs: Date.parse("2026-05-27T00:00:00Z") });
    expect(missingRef.result).toBe("unknown");
    expect(missingRef.detail).toBe("missing pull request reference");
  });

  it("returns unknown for close_pull_request on API error and sets pending_age_sec", () => {
    const nowMs = Date.parse("2026-05-27T00:00:00Z");
    const apiError = evaluateItem({ type: "close_pull_request", number: 55, repo: "acme/repo", timestamp: "2026-05-26T00:00:00Z" }, "acme/repo", { ghAPI: createAPIStub({}), nowMs });
    expect(apiError.result).toBe("unknown");
    expect(apiError.detail).toBe("api error");
    expect(typeof apiError.pending_age_sec).toBe("number");
  });

  it("evaluates update_issue retained and reverted states from persisted execution metadata", () => {
    const retained = evaluateItem(
      {
        type: "update_issue",
        repo: "acme/repo",
        number: 12,
        before_state: {
          title: "Old title",
          body_hash: hashBody("Old body"),
          state: "open",
          labels: ["triage"],
          assignees: [],
        },
        after_state: {
          title: "New title",
          body_hash: hashBody("New body"),
          state: "open",
          labels: ["triage", "bug"],
          assignees: ["octo"],
        },
      },
      "acme/repo",
      {
        ghAPI: createAPIStub({
          "repos/acme/repo/issues/12": {
            title: "New title",
            body: "New body",
            state: "open",
            labels: [{ name: "triage" }, { name: "bug" }],
            assignees: [{ login: "octo" }],
          },
        }),
      }
    );
    expect(retained).toMatchObject({
      result: "accepted",
      outcome_status: "accepted",
      evidence_strength: "medium",
      signal: "state_retained",
      detail: "update retained",
    });

    const reverted = evaluateItem(
      {
        type: "update_issue",
        repo: "acme/repo",
        number: 12,
        before_state: {
          title: "Old title",
          body_hash: hashBody("Old body"),
          state: "open",
        },
        after_state: {
          title: "New title",
          body_hash: hashBody("New body"),
          state: "closed",
        },
      },
      "acme/repo",
      {
        ghAPI: createAPIStub({
          "repos/acme/repo/issues/12": {
            title: "Old title",
            body: "Old body",
            state: "open",
            labels: [],
            assignees: [],
          },
        }),
      }
    );
    expect(reverted).toMatchObject({
      result: "rejected",
      outcome_status: "rejected",
      evidence_strength: "strong",
      signal: "state_reverted",
      detail: "update reverted",
    });
  });

  it("evaluates update_pull_request retained-merged and replaced states from persisted execution metadata", () => {
    const retainedMerged = evaluateItem(
      {
        type: "update_pull_request",
        repo: "acme/repo",
        number: 99,
        before_state: {
          title: "Old title",
          body_hash: hashBody("Old body"),
          state: "open",
          base: "main",
          draft: true,
          head_sha: "abc123",
        },
        after_state: {
          title: "New title",
          body_hash: hashBody("New body"),
          state: "open",
          base: "release",
          draft: false,
          head_sha: "def456",
        },
      },
      "acme/repo",
      {
        ghAPI: createAPIStub({
          "repos/acme/repo/pulls/99": {
            title: "New title",
            body: "New body",
            state: "closed",
            merged: true,
            base: { ref: "release" },
            draft: false,
            head: { sha: "def456" },
          },
        }),
      }
    );
    expect(retainedMerged).toMatchObject({
      result: "accepted",
      outcome_status: "accepted",
      evidence_strength: "strong",
      signal: "state_retained_and_merged",
      detail: "update retained and merged",
    });

    const replaced = evaluateItem(
      {
        type: "update_pull_request",
        repo: "acme/repo",
        number: 99,
        before_state: {
          title: "Old title",
          body_hash: hashBody("Old body"),
          state: "open",
          base: "main",
          draft: true,
          head_sha: "abc123",
        },
        after_state: {
          title: "New title",
          body_hash: hashBody("New body"),
          state: "open",
          base: "release",
          draft: false,
          head_sha: "def456",
        },
      },
      "acme/repo",
      {
        ghAPI: createAPIStub({
          "repos/acme/repo/pulls/99": {
            title: "Maintainer rewrite",
            body: "New body with edits",
            state: "open",
            merged: false,
            base: { ref: "hotfix" },
            draft: false,
            head: { sha: "zzz999" },
          },
        }),
      }
    );
    expect(replaced).toMatchObject({
      result: "rejected",
      outcome_status: "rejected",
      evidence_strength: "strong",
      signal: "state_replaced",
      detail: "update replaced",
    });
  });
});

describe("evaluate_outcomes.cjs", () => {
  it("maps existence-only fallback to weak unknown evidence", () => {
    expect(normalizeOutcome("unknown", "object still exists")).toEqual({
      outcome_status: "unknown",
      evidence_strength: "weak",
      signal: "target_exists_only",
    });
  });

  it("maps dedicated review lifecycle details to typed signals", () => {
    expect(normalizeOutcome("accepted", "review approved")).toEqual({
      outcome_status: "accepted",
      evidence_strength: "strong",
      signal: "review_approved",
    });
    expect(normalizeOutcome("rejected", "review request removed")).toEqual({
      outcome_status: "rejected",
      evidence_strength: "strong",
      signal: "review_request_removed",
    });
    expect(normalizeOutcome("rejected", "review dismissed")).toEqual({
      outcome_status: "rejected",
      evidence_strength: "strong",
      signal: "review_dismissed",
    });
  });

  it("classifies add_reviewer approval as accepted", () => {
    const api = endpoint => {
      if (endpoint.endsWith("/reviews")) {
        return [{ state: "APPROVED", submitted_at: "2026-05-12T01:00:00Z", user: { login: "reviewer1" } }];
      }
      if (endpoint.endsWith("/requested_reviewers")) {
        return { users: [], teams: [] };
      }
      throw new Error(`unexpected endpoint: ${endpoint}`);
    };

    const result = evaluateItem(
      {
        type: "add_reviewer",
        repo: "owner/repo",
        number: 42,
        timestamp: "2026-05-12T00:00:00Z",
        metadata: {
          requested_reviewers: ["reviewer1"],
        },
      },
      "owner/repo",
      api
    );

    expect(normalizeOutcome(result.result, result.detail)).toMatchObject({
      outcome_status: "accepted",
      evidence_strength: "strong",
      signal: "review_approved",
    });
  });

  it("classifies add_reviewer removal without review as rejected", () => {
    const api = endpoint => {
      if (endpoint.endsWith("/reviews")) {
        return [];
      }
      if (endpoint.endsWith("/requested_reviewers")) {
        return { users: [], teams: [] };
      }
      throw new Error(`unexpected endpoint: ${endpoint}`);
    };

    const result = evaluateItem(
      {
        type: "add_reviewer",
        repo: "owner/repo",
        number: 42,
        timestamp: "2026-05-12T00:00:00Z",
        metadata: {
          requested_reviewers: ["reviewer1"],
        },
      },
      "owner/repo",
      api
    );

    expect(normalizeOutcome(result.result, result.detail)).toMatchObject({
      outcome_status: "rejected",
      evidence_strength: "strong",
      signal: "review_request_removed",
    });
  });

  it("classifies add_reviewer pending requests as pending", () => {
    const api = endpoint => {
      if (endpoint.endsWith("/reviews")) {
        return [];
      }
      if (endpoint.endsWith("/requested_reviewers")) {
        return { users: [{ login: "reviewer1" }], teams: [] };
      }
      throw new Error(`unexpected endpoint: ${endpoint}`);
    };

    const result = evaluateItem(
      {
        type: "add_reviewer",
        repo: "owner/repo",
        number: 42,
        timestamp: "2026-05-12T00:00:00Z",
        metadata: {
          requested_reviewers: ["reviewer1"],
        },
      },
      "owner/repo",
      api
    );

    expect(normalizeOutcome(result.result, result.detail)).toMatchObject({
      outcome_status: "pending",
      evidence_strength: "medium",
      signal: "awaiting_review",
    });
  });

  it("uses the latest requested-reviewer state when approval is superseded", () => {
    const api = endpoint => {
      if (endpoint.endsWith("/reviews")) {
        return [
          { state: "APPROVED", submitted_at: "2026-05-12T01:00:00Z", user: { login: "reviewer1" } },
          { state: "CHANGES_REQUESTED", submitted_at: "2026-05-12T02:00:00Z", user: { login: "reviewer1" } },
        ];
      }
      if (endpoint.endsWith("/requested_reviewers")) {
        return { users: [], teams: [] };
      }
      throw new Error(`unexpected endpoint: ${endpoint}`);
    };

    const result = evaluateItem(
      {
        type: "add_reviewer",
        repo: "owner/repo",
        number: 42,
        timestamp: "2026-05-12T00:00:00Z",
        metadata: {
          requested_reviewers: ["reviewer1"],
        },
      },
      "owner/repo",
      api
    );

    expect(normalizeOutcome(result.result, result.detail)).toMatchObject({
      outcome_status: "accepted",
      evidence_strength: "medium",
      signal: "review_submitted",
    });
  });

  it("ignores malformed submitted_at values for review-request acceptance", () => {
    const api = endpoint => {
      if (endpoint.endsWith("/reviews")) {
        return [{ state: "APPROVED", submitted_at: "not-a-timestamp", user: { login: "reviewer1" } }];
      }
      if (endpoint.endsWith("/requested_reviewers")) {
        return { users: [], teams: [] };
      }
      throw new Error(`unexpected endpoint: ${endpoint}`);
    };

    const result = evaluateItem(
      {
        type: "add_reviewer",
        repo: "owner/repo",
        number: 42,
        timestamp: "2026-05-12T00:00:00Z",
        metadata: {
          requested_reviewers: ["reviewer1"],
        },
      },
      "owner/repo",
      api
    );

    expect(normalizeOutcome(result.result, result.detail)).toMatchObject({
      outcome_status: "rejected",
      evidence_strength: "strong",
      signal: "review_request_removed",
    });
  });

  it("classifies dismissed submitted reviews as rejected", () => {
    const api = endpoint => {
      if (endpoint.endsWith("/pulls/42")) {
        return { state: "open", merged: false };
      }
      if (endpoint.endsWith("/reviews")) {
        return [{ id: 101, state: "DISMISSED", submitted_at: "2026-05-12T01:00:00Z" }];
      }
      throw new Error(`unexpected endpoint: ${endpoint}`);
    };

    const result = evaluateItem(
      {
        type: "submit_pull_request_review",
        repo: "owner/repo",
        number: 42,
        timestamp: "2026-05-12T01:00:00Z",
        metadata: { review_id: 101 },
      },
      "owner/repo",
      api
    );

    expect(normalizeOutcome(result.result, result.detail)).toMatchObject({
      outcome_status: "rejected",
      evidence_strength: "strong",
      signal: "review_dismissed",
    });
  });

  it("classifies changes requested, push, and merge as accepted", () => {
    const api = endpoint => {
      if (endpoint.endsWith("/pulls/42")) {
        return { state: "closed", merged: true, merged_at: "2026-05-12T05:00:00Z" };
      }
      if (endpoint.endsWith("/reviews")) {
        return [{ id: 101, state: "CHANGES_REQUESTED", submitted_at: "2026-05-12T02:00:00Z" }];
      }
      if (endpoint.endsWith("/commits")) {
        return [{ commit: { committer: { date: "2026-05-12T03:00:00Z" } } }];
      }
      throw new Error(`unexpected endpoint: ${endpoint}`);
    };

    const result = evaluateItem(
      {
        type: "submit_pull_request_review",
        repo: "owner/repo",
        number: 42,
        timestamp: "2026-05-12T02:00:00Z",
        metadata: { review_id: 101 },
      },
      "owner/repo",
      api
    );

    expect(normalizeOutcome(result.result, result.detail)).toMatchObject({
      outcome_status: "accepted",
      evidence_strength: "medium",
      signal: "changes_requested_addressed",
    });
  });

  it("returns unknown outcome when commit dates are missing", () => {
    const api = endpoint => {
      if (endpoint.endsWith("/pulls/42")) {
        return { state: "closed", merged: true, merged_at: "2026-05-12T05:00:00Z" };
      }
      if (endpoint.endsWith("/reviews")) {
        return [{ id: 101, state: "CHANGES_REQUESTED", submitted_at: "2026-05-12T02:00:00Z" }];
      }
      if (endpoint.endsWith("/commits")) {
        return [{ commit: { committer: { date: "" }, author: { date: "" } } }];
      }
      throw new Error(`unexpected endpoint: ${endpoint}`);
    };

    const result = evaluateItem(
      {
        type: "submit_pull_request_review",
        repo: "owner/repo",
        number: 42,
        timestamp: "2026-05-12T02:00:00Z",
        metadata: { review_id: 101 },
      },
      "owner/repo",
      api
    );

    expect(normalizeOutcome(result.result, result.detail)).toMatchObject({
      outcome_status: "unknown",
      evidence_strength: "weak",
      signal: "unknown",
    });
  });

  it("classifies latest review on open PR as pending", () => {
    const api = endpoint => {
      if (endpoint.endsWith("/pulls/42")) {
        return { state: "open", merged: false };
      }
      if (endpoint.endsWith("/reviews")) {
        return [
          { id: 100, state: "COMMENTED", submitted_at: "2026-05-12T00:30:00Z" },
          { id: 101, state: "COMMENTED", submitted_at: "2026-05-12T01:00:00Z" },
        ];
      }
      throw new Error(`unexpected endpoint: ${endpoint}`);
    };

    const result = evaluateItem(
      {
        type: "submit_pull_request_review",
        repo: "owner/repo",
        number: 42,
        timestamp: "2026-05-12T01:00:00Z",
        metadata: { review_id: 101 },
      },
      "owner/repo",
      api
    );

    expect(normalizeOutcome(result.result, result.detail)).toMatchObject({
      outcome_status: "pending",
      evidence_strength: "medium",
      signal: "latest_review_pending",
    });
  });

  it("does not reject merged CHANGES_REQUESTED reviews without a follow-up push", () => {
    const api = endpoint => {
      if (endpoint.endsWith("/pulls/42")) {
        return { state: "closed", merged: true, merged_at: "2026-05-12T05:00:00Z" };
      }
      if (endpoint.endsWith("/reviews")) {
        return [{ id: 101, state: "CHANGES_REQUESTED", submitted_at: "2026-05-12T02:00:00Z" }];
      }
      if (endpoint.endsWith("/commits")) {
        return [];
      }
      throw new Error(`unexpected endpoint: ${endpoint}`);
    };

    const result = evaluateItem(
      {
        type: "submit_pull_request_review",
        repo: "owner/repo",
        number: 42,
        timestamp: "2026-05-12T02:00:00Z",
        metadata: { review_id: 101 },
      },
      "owner/repo",
      api
    );

    expect(result.result).toBe("unknown");
  });

  it("ignores unsubmitted reviews when checking latest open review state", () => {
    const api = endpoint => {
      if (endpoint.endsWith("/pulls/42")) {
        return { state: "open", merged: false };
      }
      if (endpoint.endsWith("/reviews")) {
        return [
          { id: 100, state: "COMMENTED", submitted_at: "2026-05-12T00:30:00Z" },
          { id: 101, state: "PENDING" },
        ];
      }
      throw new Error(`unexpected endpoint: ${endpoint}`);
    };

    const result = evaluateItem(
      {
        type: "submit_pull_request_review",
        repo: "owner/repo",
        number: 42,
        timestamp: "2026-05-12T00:30:00Z",
        metadata: { review_id: 100 },
      },
      "owner/repo",
      api
    );

    expect(result.result).toBe("pending");
    expect(result.detail).toBe("latest review awaiting outcome");
  });
});

describe("evaluate_outcomes create_pull_request evaluator", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-05-27T00:00:00Z"));
    delete process.env.GH_AW_OUTCOME_STALE_AFTER_SECONDS;
  });

  afterEach(() => {
    vi.useRealTimers();
    delete process.env.GH_AW_OUTCOME_STALE_AFTER_SECONDS;
  });

  it("classifies merged PR as accepted (strong)", () => {
    const ghAPI = mockAPI({
      "repos/owner/repo/pulls/12": {
        state: "closed",
        merged: true,
        created_at: "2026-05-20T00:00:00Z",
        merged_at: "2026-05-21T00:00:00Z",
        review_comments: 0,
        comments: 0,
      },
    });

    const result = evaluateItem({ type: "create_pull_request", repo: "owner/repo", url: "https://github.com/owner/repo/pull/12", timestamp: "2026-05-20T00:00:00Z" }, "owner/repo", { ghAPI });
    expect(result.result).toBe("accepted");
    expect(result.detail).toContain("strong");
    expect(normalizeOutcome(result.result, result.detail)).toEqual({
      outcome_status: "accepted",
      evidence_strength: "strong",
      signal: "merged",
    });
  });

  it("classifies approved open PR as accepted (medium)", () => {
    const ghAPI = mockAPI({
      "repos/owner/repo/pulls/13": { state: "open", merged: false },
      "repos/owner/repo/pulls/13/reviews": [{ state: "APPROVED" }],
    });

    const result = evaluateItem({ type: "create_pull_request", repo: "owner/repo", url: "https://github.com/owner/repo/pull/13", timestamp: "2026-05-20T00:00:00Z" }, "owner/repo", { ghAPI });
    expect(result.result).toBe("accepted");
    expect(result.detail).toContain("approved");
  });

  it("classifies closed-with-label PR as rejected (strong)", () => {
    const ghAPI = mockAPI({
      "repos/owner/repo/pulls/14": { state: "closed", merged: false, labels: [{ name: "not planned" }] },
      "repos/owner/repo/pulls/14/reviews": [],
    });

    const result = evaluateItem({ type: "create_pull_request", repo: "owner/repo", url: "https://github.com/owner/repo/pull/14", timestamp: "2026-05-20T00:00:00Z" }, "owner/repo", { ghAPI });
    expect(result.result).toBe("rejected");
    expect(result.detail).toContain("strong");
  });

  it("classifies closed-without-merge PR as rejected", () => {
    const ghAPI = mockAPI({
      "repos/owner/repo/pulls/15": { state: "closed", merged: false, labels: [] },
      "repos/owner/repo/pulls/15/reviews": [],
      "repos/owner/repo/issues/15/comments": [],
    });

    const result = evaluateItem({ type: "create_pull_request", repo: "owner/repo", url: "https://github.com/owner/repo/pull/15", timestamp: "2026-05-20T00:00:00Z" }, "owner/repo", { ghAPI });
    expect(result.result).toBe("rejected");
    expect(result.detail).toBe("closed without merge");
  });

  it("classifies open PR with no reviews as pending", () => {
    const ghAPI = mockAPI({
      "repos/owner/repo/pulls/16": { state: "open", merged: false },
      "repos/owner/repo/pulls/16/reviews": [],
    });

    const result = evaluateItem({ type: "create_pull_request", repo: "owner/repo", url: "https://github.com/owner/repo/pull/16", timestamp: "2026-05-26T23:50:00Z" }, "owner/repo", { ghAPI });
    expect(result.result).toBe("pending");
    expect(result.detail).toContain("no reviews");
  });

  it("classifies stale open PR with no reviews as ignored", () => {
    process.env.GH_AW_OUTCOME_STALE_AFTER_SECONDS = "60";
    const ghAPI = mockAPI({
      "repos/owner/repo/pulls/17": { state: "open", merged: false },
      "repos/owner/repo/pulls/17/reviews": [],
    });

    const result = evaluateItem({ type: "create_pull_request", repo: "owner/repo", url: "https://github.com/owner/repo/pull/17", timestamp: "2026-05-26T00:00:00Z" }, "owner/repo", { ghAPI });
    expect(result.result).toBe("ignored");
    expect(result.detail).toContain("stale");
  });

  it("classifies open PR with only CHANGES_REQUESTED as pending", () => {
    const ghAPI = mockAPI({
      "repos/owner/repo/pulls/18": { state: "open", merged: false },
      "repos/owner/repo/pulls/18/reviews": [{ state: "CHANGES_REQUESTED" }],
    });

    const result = evaluateItem({ type: "create_pull_request", repo: "owner/repo", url: "https://github.com/owner/repo/pull/18", timestamp: "2026-05-20T00:00:00Z" }, "owner/repo", { ghAPI });
    expect(result.result).toBe("pending");
    expect(result.detail).toContain("changes requested");
  });

  it("classifies open PR with mixed APPROVED and CHANGES_REQUESTED as unknown", () => {
    const ghAPI = mockAPI({
      "repos/owner/repo/pulls/19": { state: "open", merged: false },
      "repos/owner/repo/pulls/19/reviews": [{ state: "APPROVED" }, { state: "CHANGES_REQUESTED" }],
    });

    const result = evaluateItem({ type: "create_pull_request", repo: "owner/repo", url: "https://github.com/owner/repo/pull/19", timestamp: "2026-05-20T00:00:00Z" }, "owner/repo", { ghAPI });
    expect(result.result).toBe("unknown");
    expect(result.detail).toContain("mixed");
  });

  it("classifies open PR as unknown when reviews API fails", () => {
    const ghAPI = mockAPI({
      "repos/owner/repo/pulls/20": { state: "open", merged: false },
      "repos/owner/repo/pulls/20/reviews": null,
    });

    const result = evaluateItem({ type: "create_pull_request", repo: "owner/repo", url: "https://github.com/owner/repo/pull/20", timestamp: "2026-05-20T00:00:00Z" }, "owner/repo", { ghAPI });
    expect(result.result).toBe("unknown");
    expect(result.detail).toContain("reviews api error");
  });
});

describe("evaluate_outcomes discussion evaluators", () => {
  it("classifies closed discussions as accepted", () => {
    const ghAPI = mockAPI({
      "repos/owner/repo/discussions/11": {
        state: "closed",
        closed: true,
        comments: 2,
        created_at: "2026-05-20T00:00:00Z",
        closed_at: "2026-05-21T00:00:00Z",
      },
    });

    const result = evaluateItem(
      {
        type: "close_discussion",
        repo: "owner/repo",
        url: "https://github.com/owner/repo/discussions/11",
        timestamp: "2026-05-20T00:00:00Z",
      },
      "owner/repo",
      { ghAPI }
    );

    expect(result.result).toBe("accepted");
    expect(result.detail).toBe("closed");
    expect(result.signal).toBe("closed");
  });

  it("classifies reopened discussions as rejected", () => {
    const ghAPI = mockAPI({
      "repos/owner/repo/discussions/12": {
        state: "open",
        closed: false,
        comments: 1,
      },
    });

    const result = evaluateItem(
      {
        type: "close_discussion",
        repo: "owner/repo",
        url: "https://github.com/owner/repo/discussions/12",
        timestamp: "2026-05-20T00:00:00Z",
      },
      "owner/repo",
      { ghAPI }
    );

    expect(result.result).toBe("rejected");
    expect(result.detail).toBe("not_closed");
    expect(result.signal).toBe("not_closed");
  });

  it("classifies answered discussions as accepted", () => {
    const ghAPI = mockAPI({
      "repos/owner/repo/discussions/13": {
        state: "open",
        comments: 0,
        answer_chosen_at: "2026-05-21T00:00:00Z",
      },
    });

    const result = evaluateItem(
      {
        type: "create_discussion",
        repo: "owner/repo",
        url: "https://github.com/owner/repo/discussions/13",
        timestamp: "2026-05-20T00:00:00Z",
      },
      "owner/repo",
      { ghAPI }
    );

    expect(result.result).toBe("accepted");
    expect(result.detail).toBe("answered");
    expect(result.signal).toBe("answered");
  });

  it("classifies discussions with replies as accepted", () => {
    const ghAPI = mockAPI({
      "repos/owner/repo/discussions/14": {
        state: "open",
        comments: 3,
      },
    });

    const result = evaluateItem(
      {
        type: "create_discussion",
        repo: "owner/repo",
        url: "https://github.com/owner/repo/discussions/14",
        timestamp: "2026-05-20T00:00:00Z",
      },
      "owner/repo",
      { ghAPI }
    );

    expect(result.result).toBe("accepted");
    expect(result.detail).toBe("has replies");
    expect(result.signal).toBe("engaged");
  });

  it("classifies locked discussions as rejected", () => {
    const ghAPI = mockAPI({
      "repos/owner/repo/discussions/15": {
        state: "open",
        comments: 0,
        locked: true,
      },
    });

    const result = evaluateItem(
      {
        type: "create_discussion",
        repo: "owner/repo",
        url: "https://github.com/owner/repo/discussions/15",
        timestamp: "2026-05-20T00:00:00Z",
      },
      "owner/repo",
      { ghAPI }
    );

    expect(result.result).toBe("rejected");
    expect(result.detail).toBe("locked");
    expect(result.signal).toBe("locked");
  });

  it("classifies unengaged discussions as ignored", () => {
    const ghAPI = mockAPI({
      "repos/owner/repo/discussions/16": {
        state: "open",
        comments: 0,
        locked: false,
      },
    });

    const result = evaluateItem(
      {
        type: "create_discussion",
        repo: "owner/repo",
        url: "https://github.com/owner/repo/discussions/16",
        timestamp: "2026-05-20T00:00:00Z",
      },
      "owner/repo",
      { ghAPI }
    );

    expect(result.result).toBe("ignored");
    expect(result.detail).toBe("no replies");
    expect(result.signal).toBe("no_engagement");
  });
});

describe("evaluate_outcomes push_to_pull_request_branch evaluator", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-05-27T00:00:00Z"));
  });

  afterEach(() => {
    vi.useRealTimers();
    delete process.env.GH_AW_OUTCOME_STALE_AFTER_SECONDS;
  });

  it("classifies merged PR with pushed commit included as accepted (strong)", () => {
    const ghAPI = mockAPI({
      "repos/owner/repo/pulls/21": { state: "closed", merged: true, head: { sha: "bbbbbbb" } },
      "repos/owner/repo/compare/aaaaaaa...bbbbbbb": { status: "ahead" },
    });

    const result = evaluateItem(
      {
        type: "push_to_pull_request_branch",
        repo: "owner/repo",
        pull_request_number: 21,
        commit_sha: "aaaaaaa",
        before_head_sha: "ccccccc",
        timestamp: "2026-05-20T00:00:00Z",
      },
      "owner/repo",
      { ghAPI }
    );
    expect(result.result).toBe("accepted");
    expect(result.detail).toContain("strong");
    expect(normalizeOutcome(result.result, result.detail)).toEqual({
      outcome_status: "accepted",
      evidence_strength: "strong",
      signal: "merged",
    });
  });

  it("classifies open PR with pushed commit still at HEAD as accepted (medium)", () => {
    const ghAPI = mockAPI({
      "repos/owner/repo/pulls/22": { state: "open", merged: false, head: { sha: "aaaaaaa" } },
    });

    const result = evaluateItem(
      {
        type: "push_to_pull_request_branch",
        repo: "owner/repo",
        pull_request_number: 22,
        commit_sha: "aaaaaaa",
        timestamp: "2026-05-20T00:00:00Z",
      },
      "owner/repo",
      { ghAPI }
    );
    expect(result.result).toBe("accepted");
    expect(result.detail).toContain("head");
  });

  it("matches short pushed SHA against full current HEAD SHA", () => {
    const ghAPI = mockAPI({
      "repos/owner/repo/pulls/30": { state: "open", merged: false, head: { sha: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" } },
    });

    const result = evaluateItem(
      {
        type: "push_to_pull_request_branch",
        repo: "owner/repo",
        pull_request_number: 30,
        commit_sha: "aaaaaaa",
        timestamp: "2026-05-26T23:50:00Z",
      },
      "owner/repo",
      { ghAPI }
    );
    expect(result.result).toBe("accepted");
    expect(result.detail).toContain("head");
  });

  it("classifies force-pushed-away commits as rejected (strong)", () => {
    const ghAPI = mockAPI({
      "repos/owner/repo/pulls/23": { state: "open", merged: false, head: { sha: "bbbbbbb" } },
      "repos/owner/repo/compare/aaaaaaa...bbbbbbb": { status: "behind" },
    });

    const result = evaluateItem(
      {
        type: "push_to_pull_request_branch",
        repo: "owner/repo",
        pull_request_number: 23,
        commit_sha: "aaaaaaa",
        before_head_sha: "ccccccc",
        timestamp: "2026-05-20T00:00:00Z",
      },
      "owner/repo",
      { ghAPI }
    );
    expect(result.result).toBe("rejected");
    expect(result.detail).toContain("force-pushed");
  });

  it("does not classify unknown retention as force-pushed-away", () => {
    const ghAPI = mockAPI({
      "repos/owner/repo/pulls/29": { state: "open", merged: false, head: { sha: "bbbbbbb" } },
      "repos/owner/repo/compare/aaaaaaa...bbbbbbb": null,
      "repos/owner/repo/pulls/29/reviews": [],
    });

    const result = evaluateItem(
      {
        type: "push_to_pull_request_branch",
        repo: "owner/repo",
        pull_request_number: 29,
        commit_sha: "aaaaaaa",
        before_head_sha: "ccccccc",
        timestamp: "2026-05-26T23:50:00Z",
      },
      "owner/repo",
      { ghAPI }
    );
    expect(result.result).toBe("pending");
    expect(result.detail).toContain("no review");
  });

  it("classifies closed-without-merge PR as rejected (medium)", () => {
    const ghAPI = mockAPI({
      "repos/owner/repo/pulls/24": { state: "closed", merged: false, head: { sha: "bbbbbbb" } },
      "repos/owner/repo/compare/aaaaaaa...bbbbbbb": { status: "ahead" },
    });

    const result = evaluateItem(
      {
        type: "push_to_pull_request_branch",
        repo: "owner/repo",
        pull_request_number: 24,
        commit_sha: "aaaaaaa",
        before_head_sha: "ccccccc",
        timestamp: "2026-05-20T00:00:00Z",
      },
      "owner/repo",
      { ghAPI }
    );
    expect(result.result).toBe("rejected");
    expect(result.detail).toContain("closed without merge");
  });

  it("classifies open PR with no reviews on pushed commits as pending", () => {
    const ghAPI = mockAPI({
      "repos/owner/repo/pulls/25": { state: "open", merged: false, head: { sha: "bbbbbbb" } },
      "repos/owner/repo/compare/aaaaaaa...bbbbbbb": { status: "ahead" },
      "repos/owner/repo/pulls/25/reviews": [],
    });

    const result = evaluateItem(
      {
        type: "push_to_pull_request_branch",
        repo: "owner/repo",
        pull_request_number: 25,
        commit_sha: "aaaaaaa",
        timestamp: "2026-05-26T23:50:00Z",
      },
      "owner/repo",
      { ghAPI }
    );
    expect(result.result).toBe("pending");
    expect(result.detail).toContain("no review");
  });

  it("classifies stale open PR with no reviews on pushed commits as ignored", () => {
    process.env.GH_AW_OUTCOME_STALE_AFTER_SECONDS = "60";
    const ghAPI = mockAPI({
      "repos/owner/repo/pulls/33": { state: "open", merged: false, head: { sha: "bbbbbbb" } },
      "repos/owner/repo/compare/aaaaaaa...bbbbbbb": { status: "ahead" },
      "repos/owner/repo/pulls/33/reviews": [],
    });

    const result = evaluateItem(
      {
        type: "push_to_pull_request_branch",
        repo: "owner/repo",
        pull_request_number: 33,
        commit_sha: "aaaaaaa",
        timestamp: "2026-05-20T00:00:00Z",
      },
      "owner/repo",
      { ghAPI }
    );
    expect(result.result).toBe("ignored");
    expect(result.detail).toContain("stale");
  });

  it("falls back to unknown when pushed commits are reviewed but not head/merged", () => {
    const ghAPI = mockAPI({
      "repos/owner/repo/pulls/26": { state: "open", merged: false, head: { sha: "bbbbbbb" } },
      "repos/owner/repo/compare/aaaaaaa...bbbbbbb": { status: "ahead" },
      "repos/owner/repo/pulls/26/reviews": [{ state: "APPROVED", commit_id: "aaaaaaa" }],
    });

    const result = evaluateItem(
      {
        type: "push_to_pull_request_branch",
        repo: "owner/repo",
        pull_request_number: 26,
        commit_sha: "aaaaaaa",
        timestamp: "2026-05-20T00:00:00Z",
      },
      "owner/repo",
      { ghAPI }
    );
    expect(result.result).toBe("unknown");
  });

  it("classifies as unknown when push evaluator reviews API fails", () => {
    const ghAPI = mockAPI({
      "repos/owner/repo/pulls/34": { state: "open", merged: false, head: { sha: "bbbbbbb" } },
      "repos/owner/repo/compare/aaaaaaa...bbbbbbb": { status: "ahead" },
      "repos/owner/repo/pulls/34/reviews": null,
    });

    const result = evaluateItem(
      {
        type: "push_to_pull_request_branch",
        repo: "owner/repo",
        pull_request_number: 34,
        commit_sha: "aaaaaaa",
        timestamp: "2026-05-20T00:00:00Z",
      },
      "owner/repo",
      { ghAPI }
    );
    expect(result.result).toBe("unknown");
    expect(result.detail).toContain("reviews api error");
  });

  it("falls back to pending when push item has no commit SHA", () => {
    const ghAPI = mockAPI({
      "repos/owner/repo/pulls/27": { state: "open", merged: false, head: { sha: "bbbbbbb" } },
      "repos/owner/repo/pulls/27/reviews": [],
    });

    const result = evaluateItem(
      {
        type: "push_to_pull_request_branch",
        repo: "owner/repo",
        pull_request_number: 27,
        timestamp: "2026-05-26T23:50:00Z",
      },
      "owner/repo",
      { ghAPI }
    );
    expect(result.result).toBe("pending");
  });

  it("does not treat item.head_sha as pushed commit evidence", () => {
    const ghAPI = mockAPI({
      "repos/owner/repo/pulls/35": { state: "open", merged: false, head: { sha: "aaaaaaa" } },
      "repos/owner/repo/pulls/35/reviews": [],
    });

    const result = evaluateItem(
      {
        type: "push_to_pull_request_branch",
        repo: "owner/repo",
        pull_request_number: 35,
        head_sha: "aaaaaaa",
        timestamp: "2026-05-26T23:50:00Z",
      },
      "owner/repo",
      { ghAPI }
    );
    expect(result.result).toBe("pending");
  });
});

describe("evaluate_outcomes pull request number aliases", () => {
  it("resolves PR number from pr alias", () => {
    const ghAPI = mockAPI({
      "repos/owner/repo/pulls/31": { state: "open", merged: false },
      "repos/owner/repo/pulls/31/reviews": [],
    });
    const result = evaluateItem({ type: "create_pull_request", repo: "owner/repo", pr: 31 }, "owner/repo", { ghAPI });
    expect(result.result).toBe("pending");
  });

  it("resolves PR number from pull_number alias", () => {
    const ghAPI = mockAPI({
      "repos/owner/repo/pulls/32": { state: "open", merged: false },
      "repos/owner/repo/pulls/32/reviews": [],
    });
    const result = evaluateItem({ type: "create_pull_request", repo: "owner/repo", pull_number: 32 }, "owner/repo", { ghAPI });
    expect(result.result).toBe("pending");
  });
});
