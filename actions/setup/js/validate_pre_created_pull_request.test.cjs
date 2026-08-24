// @ts-check
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

describe("validate_pre_created_pull_request", () => {
  let originalGlobals;
  let originalEnv;

  beforeEach(() => {
    originalGlobals = {
      core: global.core,
      github: global.github,
      context: global.context,
    };
    originalEnv = { ...process.env };

    global.core = { setOutput: vi.fn(), info: vi.fn(), warning: vi.fn(), debug: vi.fn() };
    global.context = {
      repo: { owner: "owner", repo: "repo" },
    };
    global.github = {
      rest: {
        pulls: {
          get: vi.fn().mockResolvedValue({
            data: {
              head: {
                ref: "gh-aw/pre-created/123-2",
                repo: { full_name: "owner/repo" },
              },
              base: {
                repo: { full_name: "owner/repo" },
              },
            },
          }),
        },
      },
    };
    process.env.GH_AW_PRE_CREATED_PULL_REQUEST_NUMBER = "42";
    process.env.GH_AW_PRE_CREATED_PULL_REQUEST_BRANCH = "gh-aw/pre-created/123-2";
    process.env.GH_AW_EXPECTED_PRE_CREATED_PULL_REQUEST_BRANCH = "gh-aw/pre-created/123-2";
  });

  afterEach(() => {
    global.core = originalGlobals.core;
    global.github = originalGlobals.github;
    global.context = originalGlobals.context;
    vi.resetModules();
    process.env = originalEnv;
  });

  it("validates the branch and repository before emitting the trusted branch output", async () => {
    const { main } = await import("./validate_pre_created_pull_request.cjs");
    await main();

    expect(global.github.rest.pulls.get).toHaveBeenCalledWith({
      owner: "owner",
      repo: "repo",
      pull_number: 42,
    });
    expect(global.core.setOutput).toHaveBeenCalledWith("branch", "gh-aw/pre-created/123-2");
  });

  it("rejects unexpected branch output without fetching the pull request", async () => {
    process.env.GH_AW_PRE_CREATED_PULL_REQUEST_BRANCH = "attacker";
    const { main } = await import("./validate_pre_created_pull_request.cjs");

    await expect(main()).rejects.toThrow(/did not match expected workflow branch/);
    expect(global.github.rest.pulls.get).not.toHaveBeenCalled();
    expect(global.core.setOutput).not.toHaveBeenCalled();
  });

  it("rejects invalid pull request numbers", async () => {
    process.env.GH_AW_PRE_CREATED_PULL_REQUEST_NUMBER = "0";
    const { main } = await import("./validate_pre_created_pull_request.cjs");

    await expect(main()).rejects.toThrow(/pull request number is invalid/);
    expect(global.github.rest.pulls.get).not.toHaveBeenCalled();
  });

  it("rejects pull requests outside the expected repository branch", async () => {
    global.github.rest.pulls.get.mockResolvedValue({
      data: {
        head: {
          ref: "gh-aw/pre-created/123-2",
          repo: { full_name: "fork/repo" },
        },
        base: {
          repo: { full_name: "owner/repo" },
        },
      },
    });
    const { main } = await import("./validate_pre_created_pull_request.cjs");

    await expect(main()).rejects.toThrow(/does not target the expected trusted repository branch/);
    expect(global.core.setOutput).not.toHaveBeenCalled();
  });
});
