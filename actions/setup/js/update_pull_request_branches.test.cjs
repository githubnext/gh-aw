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
  let mockExec;
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
      rest: {
        pulls: {
          list: vi.fn(),
          get: vi.fn(),
          updateBranch: vi.fn(),
        },
      },
    };
    mockExec = {
      getExecOutput: vi.fn(),
    };
    mockContext = {
      repo: {
        owner: "owner",
        repo: "repo",
      },
    };

    global.core = mockCore;
    global.github = mockGithub;
    global.exec = mockExec;
    global.context = mockContext;
  });

  it("updates only mergeable pull requests without active sessions", async () => {
    mockGithub.paginate.mockResolvedValue([{ number: 1 }, { number: 2 }, { number: 3 }]);
    mockGithub.rest.pulls.get.mockImplementation(async ({ pull_number }) => {
      if (pull_number === 1) return { data: { state: "open", mergeable: true, draft: false } };
      if (pull_number === 2) return { data: { state: "open", mergeable: false, draft: false } };
      return { data: { state: "open", mergeable: true, draft: false } };
    });
    mockExec.getExecOutput.mockResolvedValue({
      stdout: JSON.stringify([
        { pullRequestNumber: 3, state: "open" },
        { pullRequestNumber: 10, state: "closed" },
      ]),
      stderr: "",
      exitCode: 0,
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
    mockExec.getExecOutput.mockResolvedValue({
      stdout: JSON.stringify([]),
      stderr: "",
      exitCode: 0,
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
});
