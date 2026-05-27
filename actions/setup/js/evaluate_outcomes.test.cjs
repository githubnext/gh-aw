import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { createRequire } from "module";

const req = createRequire(import.meta.url);
const { evaluateItem, __setGHAPIForTest } = req("./evaluate_outcomes.cjs");

/**
 * @param {Record<string, any>} apiResponses
 */
function mockAPI(apiResponses) {
  __setGHAPIForTest(endpoint => {
    if (!(endpoint in apiResponses)) {
      return null;
    }
    return apiResponses[endpoint];
  });
}

describe("evaluate_outcomes create_pull_request evaluator", () => {
  beforeEach(() => {
    __setGHAPIForTest(null);
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-05-27T00:00:00Z"));
    delete process.env.GH_AW_OUTCOME_STALE_AFTER_SECONDS;
  });

  afterEach(() => {
    vi.useRealTimers();
    delete process.env.GH_AW_OUTCOME_STALE_AFTER_SECONDS;
  });

  it("classifies merged PR as accepted (strong)", () => {
    mockAPI({
      "repos/owner/repo/pulls/12": {
        state: "closed",
        merged: true,
        created_at: "2026-05-20T00:00:00Z",
        merged_at: "2026-05-21T00:00:00Z",
        review_comments: 0,
        comments: 0,
      },
    });

    const result = evaluateItem({ type: "create_pull_request", repo: "owner/repo", url: "https://github.com/owner/repo/pull/12", timestamp: "2026-05-20T00:00:00Z" }, "owner/repo");
    expect(result.result).toBe("accepted");
    expect(result.detail).toContain("strong");
  });

  it("classifies approved open PR as accepted (medium)", () => {
    mockAPI({
      "repos/owner/repo/pulls/13": { state: "open", merged: false },
      "repos/owner/repo/pulls/13/reviews": [{ state: "APPROVED" }],
    });

    const result = evaluateItem({ type: "create_pull_request", repo: "owner/repo", url: "https://github.com/owner/repo/pull/13", timestamp: "2026-05-20T00:00:00Z" }, "owner/repo");
    expect(result.result).toBe("accepted");
    expect(result.detail).toContain("approved");
  });

  it("classifies closed-with-label PR as rejected (strong)", () => {
    mockAPI({
      "repos/owner/repo/pulls/14": { state: "closed", merged: false, labels: [{ name: "not planned" }] },
      "repos/owner/repo/pulls/14/reviews": [],
    });

    const result = evaluateItem({ type: "create_pull_request", repo: "owner/repo", url: "https://github.com/owner/repo/pull/14", timestamp: "2026-05-20T00:00:00Z" }, "owner/repo");
    expect(result.result).toBe("rejected");
    expect(result.detail).toContain("strong");
  });

  it("classifies closed-without-merge PR as rejected", () => {
    mockAPI({
      "repos/owner/repo/pulls/15": { state: "closed", merged: false, labels: [] },
      "repos/owner/repo/pulls/15/reviews": [],
      "repos/owner/repo/issues/15/comments": [],
    });

    const result = evaluateItem({ type: "create_pull_request", repo: "owner/repo", url: "https://github.com/owner/repo/pull/15", timestamp: "2026-05-20T00:00:00Z" }, "owner/repo");
    expect(result.result).toBe("rejected");
    expect(result.detail).toBe("closed without merge");
  });

  it("classifies open PR with no reviews as pending", () => {
    mockAPI({
      "repos/owner/repo/pulls/16": { state: "open", merged: false },
      "repos/owner/repo/pulls/16/reviews": [],
    });

    const result = evaluateItem({ type: "create_pull_request", repo: "owner/repo", url: "https://github.com/owner/repo/pull/16", timestamp: "2026-05-26T23:50:00Z" }, "owner/repo");
    expect(result.result).toBe("pending");
    expect(result.detail).toContain("no reviews");
  });

  it("classifies stale open PR with no reviews as ignored", () => {
    process.env.GH_AW_OUTCOME_STALE_AFTER_SECONDS = "60";
    mockAPI({
      "repos/owner/repo/pulls/17": { state: "open", merged: false },
      "repos/owner/repo/pulls/17/reviews": [],
    });

    const result = evaluateItem({ type: "create_pull_request", repo: "owner/repo", url: "https://github.com/owner/repo/pull/17", timestamp: "2026-05-26T00:00:00Z" }, "owner/repo");
    expect(result.result).toBe("ignored");
    expect(result.detail).toContain("stale");
  });

  it("classifies open PR with mixed review state as unknown", () => {
    mockAPI({
      "repos/owner/repo/pulls/18": { state: "open", merged: false },
      "repos/owner/repo/pulls/18/reviews": [{ state: "CHANGES_REQUESTED" }],
    });

    const result = evaluateItem({ type: "create_pull_request", repo: "owner/repo", url: "https://github.com/owner/repo/pull/18", timestamp: "2026-05-20T00:00:00Z" }, "owner/repo");
    expect(result.result).toBe("unknown");
  });
});

describe("evaluate_outcomes push_to_pull_request_branch evaluator", () => {
  beforeEach(() => {
    __setGHAPIForTest(null);
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-05-27T00:00:00Z"));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("classifies merged PR with pushed commit included as accepted (strong)", () => {
    mockAPI({
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
      "owner/repo"
    );
    expect(result.result).toBe("accepted");
    expect(result.detail).toContain("strong");
  });

  it("classifies open PR with pushed commit still at HEAD as accepted (medium)", () => {
    mockAPI({
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
      "owner/repo"
    );
    expect(result.result).toBe("accepted");
    expect(result.detail).toContain("head");
  });

  it("classifies force-pushed-away commits as rejected (strong)", () => {
    mockAPI({
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
      "owner/repo"
    );
    expect(result.result).toBe("rejected");
    expect(result.detail).toContain("force-pushed");
  });

  it("classifies closed-without-merge PR as rejected (medium)", () => {
    mockAPI({
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
      "owner/repo"
    );
    expect(result.result).toBe("rejected");
    expect(result.detail).toContain("closed without merge");
  });

  it("classifies open PR with no reviews on pushed commits as pending", () => {
    mockAPI({
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
        timestamp: "2026-05-20T00:00:00Z",
      },
      "owner/repo"
    );
    expect(result.result).toBe("pending");
    expect(result.detail).toContain("no review");
  });

  it("falls back to unknown when pushed commits are reviewed but not head/merged", () => {
    mockAPI({
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
      "owner/repo"
    );
    expect(result.result).toBe("unknown");
  });
});
