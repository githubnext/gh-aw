import { describe, expect, it } from "vitest";
import { createRequire } from "module";

const req = createRequire(import.meta.url);
const { evaluateItem, normalizeOutcome } = req("./evaluate_outcomes.cjs");

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
});
