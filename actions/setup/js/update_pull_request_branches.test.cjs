// @ts-check
import { describe, it, expect, beforeEach, vi } from "vitest";

vi.mock("./github_rate_limit_logger.cjs", () => ({
  fetchAndLogRateLimit: vi.fn().mockResolvedValue(undefined),
}));

const moduleUnderTest = await import("./update_pull_request_branches.cjs");

describe("update_pull_request_branches", () => {
  /** @type {any} */
  let mockCore;
  /** @type {any} */
  let mockGithub;
  /** @type {any} */
  let mockContext;
  /** @type {any} */
  let fetchMock;

  beforeEach(() => {
    vi.clearAllMocks();

    mockCore = {
      info: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),
      debug: vi.fn(),
      notice: vi.fn(),
    };
    mockGithub = {
      paginate: vi.fn(),
      graphql: vi.fn(),
      rest: {
        pulls: {
          list: vi.fn(),
          get: vi.fn(),
          updateBranch: vi.fn(),
        },
      },
    };
    mockContext = {
      repo: {
        owner: "owner",
        repo: "repo",
      },
    };

    global.core = mockCore;
    global.github = mockGithub;
    global.context = mockContext;
    fetchMock = vi.fn();
    global.fetch = fetchMock;
    process.env.GH_TOKEN = "test-token";
  });

  it("updates only mergeable pull requests without active sessions", async () => {
    mockGithub.paginate.mockResolvedValue([{ number: 1 }, { number: 2 }, { number: 3 }]);
    mockGithub.rest.pulls.get.mockImplementation(async ({ pull_number }) => {
      if (pull_number === 1) return { data: { state: "open", mergeable: true, draft: false } };
      if (pull_number === 2) return { data: { state: "open", mergeable: false, draft: false } };
      return { data: { state: "open", mergeable: true, draft: false } };
    });
    mockGithub.graphql.mockResolvedValue({ viewer: { copilotEndpoints: { api: "https://api.copilot.test" } } });
    fetchMock.mockResolvedValue({
      ok: true,
      json: async () => ({
        sessions: [
          { resource_id: 3, state: "open", resource_type: "pull" },
          { resource_id: 10, state: "closed", resource_type: "pull" },
        ],
      }),
    });
    mockGithub.rest.pulls.updateBranch.mockResolvedValue({ data: {} });

    await moduleUnderTest.main();

    expect(mockGithub.rest.pulls.updateBranch).toHaveBeenCalledTimes(1);
    expect(mockGithub.rest.pulls.updateBranch).toHaveBeenCalledWith({
      owner: "owner",
      repo: "repo",
      pull_number: 1,
    });
  });

  it("continues on non-fatal updateBranch failures", async () => {
    mockGithub.paginate.mockResolvedValue([{ number: 7 }]);
    mockGithub.rest.pulls.get.mockResolvedValue({ data: { state: "open", mergeable: true, draft: false } });
    mockGithub.graphql.mockResolvedValue({ viewer: { copilotEndpoints: { api: "https://api.copilot.test" } } });
    fetchMock.mockResolvedValue({
      ok: true,
      json: async () => ({ sessions: [] }),
    });
    const err = new Error("Update branch failed");
    // @ts-ignore
    err.status = 422;
    mockGithub.rest.pulls.updateBranch.mockRejectedValue(err);

    await expect(moduleUnderTest.main()).resolves.not.toThrow();
    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Skipping PR #7"));
  });

  it("parses pull request numbers and active states correctly", () => {
    expect(moduleUnderTest.parsePullRequestNumber(12)).toBe(12);
    expect(moduleUnderTest.parsePullRequestNumber("34")).toBe(34);
    expect(moduleUnderTest.parsePullRequestNumber("0")).toBeNull();
    expect(moduleUnderTest.parsePullRequestNumber("not-a-number")).toBeNull();

    expect(moduleUnderTest.isActiveSessionState("OPEN")).toBe(true);
    expect(moduleUnderTest.isActiveSessionState("in_progress")).toBe(true);
    expect(moduleUnderTest.isActiveSessionState("closed")).toBe(false);
  });

  it("filters candidate pull requests to only those without active sessions", async () => {
    mockGithub.graphql.mockResolvedValue({ viewer: { copilotEndpoints: { api: "https://api.copilot.test" } } });
    fetchMock.mockResolvedValue({
      ok: true,
      json: async () => ({
        sessions: [
          { resource_id: 2, state: "OPEN", resource_type: "pull" },
          { resource_id: 9, state: "queued", resource_type: "pull" },
        ],
      }),
    });

    const result = await moduleUnderTest.filterPullRequestsWithoutActiveSessions([1, 2, 3]);

    expect(result).toEqual([1, 3]);
  });

  it("ignores draft pull requests when filtering mergeable pull requests", async () => {
    mockGithub.rest.pulls.get.mockImplementation(async ({ pull_number }) => {
      if (pull_number === 1) return { data: { state: "open", mergeable: true, draft: true } };
      if (pull_number === 2) return { data: { state: "open", mergeable: true, draft: false } };
      return { data: { state: "open", mergeable: false, draft: false } };
    });

    const result = await moduleUnderTest.filterMergeablePullRequests("owner", "repo", [1, 2, 3]);

    expect(result).toEqual([2]);
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Skipping PR #1"));
  });
});
