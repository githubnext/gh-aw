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
        issues: {
          createComment: vi.fn(),
        },
        pulls: {
          list: vi.fn(),
          get: vi.fn(),
          updateBranch: vi.fn(),
        },
      },
    };
    mockContext = {
      runId: 123,
      serverUrl: "https://github.com",
      repo: {
        owner: "owner",
        repo: "repo",
      },
    };

    global.core = mockCore;
    global.github = mockGithub;
    global.context = mockContext;
  });

  it("updates only mergeable pull requests", async () => {
    mockGithub.paginate.mockResolvedValue([{ number: 1 }, { number: 2 }, { number: 3 }]);
    mockGithub.rest.pulls.get.mockImplementation(async ({ pull_number }) => {
      if (pull_number === 1) return { data: { state: "open", mergeable: true, draft: false, head: { repo: { full_name: "owner/repo" } } } };
      if (pull_number === 2) return { data: { state: "open", mergeable: false, draft: false, head: { repo: { full_name: "owner/repo" } } } };
      return { data: { state: "open", mergeable: true, draft: false, head: { repo: { full_name: "owner/repo" } } } };
    });
    mockGithub.rest.pulls.updateBranch.mockResolvedValue({ data: {} });

    vi.useFakeTimers();
    try {
      const mainPromise = moduleUnderTest.main();
      await vi.runAllTimersAsync();
      await mainPromise;
    } finally {
      vi.useRealTimers();
    }

    expect(mockGithub.rest.pulls.updateBranch).toHaveBeenCalledTimes(2);
    expect(mockGithub.rest.pulls.updateBranch).toHaveBeenNthCalledWith(1, {
      owner: "owner",
      repo: "repo",
      pull_number: 1,
    });
    expect(mockGithub.rest.pulls.updateBranch).toHaveBeenNthCalledWith(2, {
      owner: "owner",
      repo: "repo",
      pull_number: 3,
    });
    expect(mockGithub.rest.issues.createComment).toHaveBeenCalledTimes(2);
    expect(mockGithub.rest.issues.createComment).toHaveBeenNthCalledWith(1, {
      owner: "owner",
      repo: "repo",
      issue_number: 1,
      body: expect.stringContaining("[View workflow run](https://github.com/owner/repo/actions/runs/123)"),
    });
    expect(mockGithub.rest.issues.createComment).toHaveBeenNthCalledWith(2, {
      owner: "owner",
      repo: "repo",
      issue_number: 3,
      body: expect.stringContaining("[View workflow run](https://github.com/owner/repo/actions/runs/123)"),
    });
  });

  it("continues on non-fatal updateBranch failures", async () => {
    mockGithub.paginate.mockResolvedValue([{ number: 7 }]);
    mockGithub.rest.pulls.get.mockResolvedValue({ data: { state: "open", mergeable: true, draft: false, head: { repo: { full_name: "owner/repo" } } } });
    const err = new Error("Update branch failed");
    // @ts-ignore
    err.status = 422;
    mockGithub.rest.pulls.updateBranch.mockRejectedValue(err);

    await expect(moduleUnderTest.main()).resolves.not.toThrow();
    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Skipping PR #7"));
    expect(mockGithub.rest.issues.createComment).not.toHaveBeenCalled();
  });

  it("ignores draft pull requests when filtering mergeable pull requests", async () => {
    mockGithub.rest.pulls.get.mockImplementation(async ({ pull_number }) => {
      if (pull_number === 1) return { data: { state: "open", mergeable: true, draft: true, head: { repo: { full_name: "owner/repo" } } } };
      if (pull_number === 2) return { data: { state: "open", mergeable: true, draft: false, head: { repo: { full_name: "owner/repo" } } } };
      return { data: { state: "open", mergeable: false, draft: false, head: { repo: { full_name: "owner/repo" } } } };
    });

    const result = await moduleUnderTest.filterMergeablePullRequests("owner", "repo", [1, 2, 3]);

    expect(result).toEqual([2]);
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Skipping PR #1"));
  });

  it("ignores fork pull requests that cannot be updated by repository token", async () => {
    mockGithub.rest.pulls.get.mockImplementation(async ({ pull_number }) => {
      if (pull_number === 1) return { data: { state: "open", mergeable: true, draft: false, head: { repo: { full_name: "fork-owner/repo" } } } };
      return { data: { state: "open", mergeable: true, draft: false, head: { repo: { full_name: "owner/repo" } } } };
    });

    const result = await moduleUnderTest.filterMergeablePullRequests("owner", "repo", [1, 2]);

    expect(result).toEqual([2]);
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("reason=head_repository_mismatch"));
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("head_repo=fork-owner/repo"));
  });

  it("logs explicit reason when head repository is unavailable", async () => {
    mockGithub.rest.pulls.get.mockResolvedValue({
      data: { state: "open", mergeable: true, draft: false, head: { repo: null } },
    });

    const result = await moduleUnderTest.filterMergeablePullRequests("owner", "repo", [11]);

    expect(result).toEqual([]);
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("reason=head_repository_missing"));
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("head_repo=unknown"));
  });

  it("returns early when there are no open pull requests", async () => {
    mockGithub.paginate.mockResolvedValue([]);

    await moduleUnderTest.main();

    expect(mockGithub.rest.pulls.get).not.toHaveBeenCalled();
    expect(mockGithub.rest.pulls.updateBranch).not.toHaveBeenCalled();
  });

  it("returns early when no pull requests are mergeable", async () => {
    mockGithub.paginate.mockResolvedValue([{ number: 1 }]);
    mockGithub.rest.pulls.get.mockResolvedValue({
      data: { state: "open", mergeable: false, draft: false, head: { repo: { full_name: "owner/repo" } } },
    });

    await moduleUnderTest.main();

    expect(mockGithub.rest.pulls.updateBranch).not.toHaveBeenCalled();
  });

  it("counts fatal errors separately from non-fatal errors", async () => {
    mockGithub.paginate.mockResolvedValue([{ number: 10 }, { number: 11 }]);
    mockGithub.rest.pulls.get.mockResolvedValue({
      data: { state: "open", mergeable: true, draft: false, head: { repo: { full_name: "owner/repo" } } },
    });
    const fatalErr = new Error("Something unexpected");
    const nonFatalErr = Object.assign(new Error("update branch failed"), { status: 422 });
    mockGithub.rest.pulls.updateBranch.mockRejectedValueOnce(fatalErr).mockRejectedValueOnce(nonFatalErr);

    vi.useFakeTimers();
    try {
      const mainPromise = moduleUnderTest.main();
      await vi.runAllTimersAsync();
      await expect(mainPromise).resolves.not.toThrow();
    } finally {
      vi.useRealTimers();
    }
    expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Failed to update branch for PR #10"));
    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Skipping PR #11"));
    expect(mockCore.notice).toHaveBeenCalledWith(expect.stringContaining("updated=0, skipped=1, failed=1"));
  });

  it("identifies non-fatal error by status 422", () => {
    const err = Object.assign(new Error("Unprocessable"), { status: 422 });
    expect(moduleUnderTest.isNonFatalUpdateBranchError(err)).toBe(true);
  });

  it("identifies non-fatal error by message 'update branch failed'", () => {
    expect(moduleUnderTest.isNonFatalUpdateBranchError(new Error("Update branch failed"))).toBe(true);
  });

  it("identifies non-fatal error by message 'head branch is not behind'", () => {
    expect(moduleUnderTest.isNonFatalUpdateBranchError(new Error("Head branch is not behind base branch"))).toBe(true);
  });

  it("does not treat other errors as non-fatal", () => {
    expect(moduleUnderTest.isNonFatalUpdateBranchError(new Error("Something else went wrong"))).toBe(false);
  });

  it("skips closed pull requests when filtering mergeable pull requests", async () => {
    mockGithub.rest.pulls.get.mockImplementation(async ({ pull_number }) => {
      if (pull_number === 1) return { data: { state: "closed", mergeable: true, draft: false, head: { repo: { full_name: "owner/repo" } } } };
      return { data: { state: "open", mergeable: true, draft: false, head: { repo: { full_name: "owner/repo" } } } };
    });

    const result = await moduleUnderTest.filterMergeablePullRequests("owner", "repo", [1, 2]);

    expect(result).toEqual([2]);
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Skipping PR #1"));
  });

  it("skips pull requests where mergeable is null (GitHub pending computation)", async () => {
    mockGithub.rest.pulls.get.mockResolvedValue({
      data: { state: "open", mergeable: null, draft: false, head: { repo: { full_name: "owner/repo" } } },
    });

    const result = await moduleUnderTest.filterMergeablePullRequests("owner", "repo", [5]);

    expect(result).toEqual([]);
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("mergeable=null"));
  });

  it("does not treat non-object errors as non-fatal", () => {
    expect(moduleUnderTest.isNonFatalUpdateBranchError("some string error")).toBe(false);
    expect(moduleUnderTest.isNonFatalUpdateBranchError(null)).toBe(false);
    expect(moduleUnderTest.isNonFatalUpdateBranchError(42)).toBe(false);
  });

  it("does not treat 404 status errors as non-fatal", () => {
    const err = Object.assign(new Error("Not Found"), { status: 404 });
    expect(moduleUnderTest.isNonFatalUpdateBranchError(err)).toBe(false);
  });

  it("filters out non-integer pull request numbers", async () => {
    mockGithub.paginate.mockResolvedValue([{ number: 1 }, { number: "bad" }, { number: null }, { number: 2 }]);
    mockGithub.rest.pulls.get.mockResolvedValue({
      data: { state: "open", mergeable: true, draft: false, head: { repo: { full_name: "owner/repo" } } },
    });
    mockGithub.rest.pulls.updateBranch.mockResolvedValue({ data: {} });

    vi.useFakeTimers();
    try {
      const mainPromise = moduleUnderTest.main();
      await vi.runAllTimersAsync();
      await mainPromise;
    } finally {
      vi.useRealTimers();
    }

    expect(mockGithub.rest.pulls.updateBranch).toHaveBeenCalledTimes(2);
    expect(mockGithub.rest.pulls.updateBranch).toHaveBeenCalledWith(expect.objectContaining({ pull_number: 1 }));
    expect(mockGithub.rest.pulls.updateBranch).toHaveBeenCalledWith(expect.objectContaining({ pull_number: 2 }));
  });
});
