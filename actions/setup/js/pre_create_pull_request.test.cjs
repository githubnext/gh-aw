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

    global.core = { setOutput: vi.fn() };
    global.exec = {
      getExecOutput: vi.fn().mockResolvedValue({ stdout: "checkout-head-sha\n" }),
    };
    global.context = {
      serverUrl: "https://github.com",
      workflow: "Fallback workflow",
      runId: 123,
      sha: "base-sha",
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
  });

  afterEach(() => {
    global.core = originalGlobals.core;
    global.github = originalGlobals.github;
    global.context = originalGlobals.context;
    global.exec = originalGlobals.exec;
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
        title: "[Test workflow] Work in progress",
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
});
