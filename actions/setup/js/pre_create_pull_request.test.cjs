// @ts-check
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

describe("pre_create_pull_request", () => {
  let originalGlobals;
  let originalEnv;

  beforeEach(() => {
    originalGlobals = {
      core: global.core,
      github: global.github,
      context: global.context,
      exec: global.exec,
    };
    originalEnv = { ...process.env };

    global.core = { setOutput: vi.fn(), info: vi.fn(), warning: vi.fn(), debug: vi.fn() };
    global.context = {
      serverUrl: "https://github.com",
      workflow: "Fallback workflow",
      runId: 123,
      sha: "base-sha",
      eventName: "workflow_dispatch",
      payload: { repository: { default_branch: "main" } },
      repo: { owner: "owner", repo: "repo" },
    };
    global.github = {
      rest: {
        repos: {
          get: vi.fn().mockResolvedValue({ data: { default_branch: "main" } }),
          getCommit: vi.fn().mockResolvedValue({
            data: { sha: "base-sha", commit: { tree: { sha: "tree-sha" } } },
          }),
        },
        git: {
          createCommit: vi.fn().mockResolvedValue({ data: { sha: "pre-created-sha" } }),
          createRef: vi.fn().mockResolvedValue({ data: {} }),
        },
        pulls: {
          create: vi.fn().mockResolvedValue({
            data: { number: 42, html_url: "https://github.com/owner/repo/pull/42" },
          }),
        },
        checks: {
          create: vi.fn().mockResolvedValue({ data: { id: 99 } }),
        },
      },
    };
    process.env.GH_AW_WORKFLOW_NAME = "Test workflow";
    process.env.GITHUB_RUN_ATTEMPT = "2";
    delete process.env.GH_AW_CUSTOM_BASE_BRANCH;
    delete process.env.GH_AW_PR_TITLE_PREFIX;
    delete process.env.GITHUB_BASE_REF;
  });

  afterEach(() => {
    global.core = originalGlobals.core;
    global.github = originalGlobals.github;
    global.context = originalGlobals.context;
    global.exec = originalGlobals.exec;
    vi.resetModules();
    process.env = originalEnv;
  });

  it("creates a draft PR and an in-progress check on a unique empty commit", async () => {
    const { main } = await import("./pre_create_pull_request.cjs");
    await main();

    expect(global.github.rest.git.createCommit).toHaveBeenCalledWith(
      expect.objectContaining({
        tree: "tree-sha",
        parents: ["base-sha"],
      })
    );
    expect(global.github.rest.git.createRef).toHaveBeenCalledWith(
      expect.objectContaining({
        ref: "refs/heads/gh-aw/pre-created/123-2",
        sha: "pre-created-sha",
      })
    );
    expect(global.github.rest.pulls.create).toHaveBeenCalledWith(
      expect.objectContaining({
        title: "[WIP] Test workflow: work in progress",
        body: expect.stringContaining("/actions/runs/123"),
        base: "main",
        draft: true,
      })
    );
    expect(global.github.rest.checks.create).toHaveBeenCalledWith(
      expect.objectContaining({
        name: "Test workflow",
        head_sha: "pre-created-sha",
        status: "in_progress",
      })
    );
    expect(global.core.setOutput).toHaveBeenCalledWith("pull_request_number", 42);
    expect(global.core.setOutput).toHaveBeenCalledWith("branch", "gh-aw/pre-created/123-2");
  });

  it("applies the configured title prefix after the WIP marker", async () => {
    process.env.GH_AW_PR_TITLE_PREFIX = "[bot] ";
    const { main } = await import("./pre_create_pull_request.cjs");
    await main();

    expect(global.github.rest.pulls.create).toHaveBeenCalledWith(
      expect.objectContaining({
        title: "[WIP] [bot] Test workflow: work in progress",
      })
    );
  });

  it("explains that the pull request is in progress and that steering is unsupported", async () => {
    const { main } = await import("./pre_create_pull_request.cjs");
    await main();

    const body = global.github.rest.pulls.create.mock.calls[0][0].body;
    expect(body).toContain("work in progress");
    expect(body).toContain("Steering is not supported yet");
    expect(body).toContain("Test workflow");
  });

  it("always creates the allocated pull request as a draft, even when the draft policy is disabled", async () => {
    // The configured draft policy is only applied later, in the safe output phase, so the
    // allocation step must not consume it.
    process.env.GH_AW_SAFE_OUTPUTS_CONFIG = JSON.stringify({ "create-pull-request": { draft: false } });
    const { main } = await import("./pre_create_pull_request.cjs");
    await main();

    expect(global.github.rest.pulls.create).toHaveBeenCalledWith(expect.objectContaining({ draft: true }));
  });

  it("forks the pre-created branch from the configured base branch", async () => {
    process.env.GH_AW_CUSTOM_BASE_BRANCH = "release/v1";
    const { main } = await import("./pre_create_pull_request.cjs");
    await main();

    expect(global.github.rest.repos.getCommit).toHaveBeenCalledWith(
      expect.objectContaining({
        ref: "heads/release/v1",
      })
    );
    expect(global.github.rest.pulls.create).toHaveBeenCalledWith(
      expect.objectContaining({
        base: "release/v1",
      })
    );
  });

  it("rejects base branch names that require normalization", async () => {
    process.env.GH_AW_CUSTOM_BASE_BRANCH = "bad branch;name";
    const { main } = await import("./pre_create_pull_request.cjs");

    await expect(main()).rejects.toThrow(/Invalid base branch/);
    expect(global.github.rest.git.createRef).not.toHaveBeenCalled();
  });

  it("deletes the allocated branch when the pull request cannot be created", async () => {
    global.github.rest.git.deleteRef = vi.fn().mockResolvedValue({ data: {} });
    global.github.rest.pulls.create = vi.fn().mockRejectedValue(new Error("boom"));
    const { main } = await import("./pre_create_pull_request.cjs");

    await expect(main()).rejects.toThrow(/boom/);
    expect(global.github.rest.git.deleteRef).toHaveBeenCalledWith(
      expect.objectContaining({
        ref: "heads/gh-aw/pre-created/123-2",
      })
    );
    expect(global.core.setOutput).not.toHaveBeenCalled();
  });
});
