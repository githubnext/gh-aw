import { describe, it, expect, beforeEach, vi } from "vitest";

const mockCore = {
  debug: vi.fn(),
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(undefined),
  },
};

global.core = mockCore;

const mockGetReview = vi.fn();
const mockDismissReview = vi.fn();
const mockGetPullRequest = vi.fn();
const mockListReviews = vi.fn();

const mockGithub = {
  rest: {
    pulls: {
      getReview: mockGetReview,
      dismissReview: mockDismissReview,
      get: mockGetPullRequest,
      listReviews: mockListReviews,
    },
  },
};

global.github = mockGithub;
global.context = {
  actor: "github-actions[bot]",
  eventName: "pull_request",
  repo: { owner: "test-owner", repo: "test-repo" },
  payload: { pull_request: { number: 42 } },
};

describe("dismiss_pull_request_review", () => {
  let handler;

  beforeEach(async () => {
    vi.resetModules();
    vi.clearAllMocks();
    process.env.GITHUB_ACTOR = "github-actions[bot]";

    mockGetReview.mockResolvedValue({
      data: {
        html_url: "https://github.com/test-owner/test-repo/pull/42#pullrequestreview-123",
        user: { login: "github-actions[bot]" },
      },
    });
    mockDismissReview.mockResolvedValue({
      data: {
        html_url: "https://github.com/test-owner/test-repo/pull/42#pullrequestreview-123",
      },
    });
    mockGetPullRequest.mockResolvedValue({
      data: {
        mergeable_state: "clean",
        requested_reviewers: [],
        requested_teams: [],
      },
    });
    mockListReviews.mockResolvedValue({
      data: [
        {
          id: 123,
          state: "CHANGES_REQUESTED",
          submitted_at: "2026-07-06T00:00:00Z",
          user: { login: "github-actions[bot]" },
        },
      ],
    });

    const { main } = require("./dismiss_pull_request_review.cjs");
    handler = await main({ max: 10 });
  });

  it("dismisses a review when author matches current actor", async () => {
    const result = await handler({
      type: "dismiss_pull_request_review",
      review_id: 123,
      justification: "This stale review no longer reflects the updated implementation.",
    });

    expect(result.success).toBe(true);
    expect(mockGetReview).toHaveBeenCalledWith(
      expect.objectContaining({
        pull_number: 42,
        review_id: 123,
      })
    );
    expect(mockDismissReview).toHaveBeenCalledWith(
      expect.objectContaining({
        pull_number: 42,
        review_id: 123,
      })
    );
  });

  it("rejects when provided author differs from current actor", async () => {
    const result = await handler({
      type: "dismiss_pull_request_review",
      review_id: 123,
      author: "octocat",
      justification: "This stale review no longer reflects the updated implementation.",
    });

    expect(result.success).toBe(false);
    expect(result.error).toContain("author must match the current workflow actor");
    expect(mockDismissReview).not.toHaveBeenCalled();
  });

  it("rejects when fetched review author differs from current actor", async () => {
    mockGetReview.mockResolvedValueOnce({
      data: {
        user: { login: "octocat" },
      },
    });

    const result = await handler({
      type: "dismiss_pull_request_review",
      review_id: 123,
      justification: "This stale review no longer reflects the updated implementation.",
    });

    expect(result.success).toBe(false);
    expect(result.error).toContain("review author");
    expect(mockDismissReview).not.toHaveBeenCalled();
  });

  it("resolves review_id=auto to latest dismissible review by current actor", async () => {
    mockGetReview.mockResolvedValueOnce({
      data: {
        html_url: "https://github.com/test-owner/test-repo/pull/42#pullrequestreview-456",
        user: { login: "github-actions[bot]" },
      },
    });
    mockListReviews.mockResolvedValueOnce({
      data: [
        {
          id: 111,
          state: "APPROVED",
          submitted_at: "2026-07-01T00:00:00Z",
          user: { login: "github-actions[bot]" },
        },
        {
          id: 456,
          state: "CHANGES_REQUESTED",
          submitted_at: "2026-07-02T00:00:00Z",
          user: { login: "github-actions[bot]" },
        },
      ],
    });

    const result = await handler({
      type: "dismiss_pull_request_review",
      review_id: "auto",
      justification: "This stale review no longer reflects the updated implementation.",
    });

    expect(result.success).toBe(true);
    expect(mockDismissReview).toHaveBeenCalledWith(
      expect.objectContaining({
        pull_number: 42,
        review_id: 456,
      })
    );
  });

  it("detects degenerate blocked state with no requested reviewers when review_id=auto has no candidate", async () => {
    mockGetPullRequest.mockResolvedValueOnce({
      data: {
        mergeable_state: "blocked",
        requested_reviewers: [],
        requested_teams: [],
      },
    });
    mockListReviews.mockResolvedValueOnce({ data: [] });

    const result = await handler({
      type: "dismiss_pull_request_review",
      review_id: "auto",
      justification: "This stale review no longer reflects the updated implementation.",
    });

    expect(result.success).toBe(false);
    expect(result.error).toContain("degenerate review-required state");
    expect(mockDismissReview).not.toHaveBeenCalled();
  });
});
