// @ts-check
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

describe("complete_pre_created_check_run", () => {
  let originalGlobals;
  let originalEnv;

  beforeEach(() => {
    originalGlobals = {
      core: global.core,
      github: global.github,
      context: global.context,
    };
    originalEnv = { ...process.env };
    global.core = { info: vi.fn(), warning: vi.fn() };
    global.context = {
      serverUrl: "https://github.com",
      workflow: "Test workflow",
      runId: 123,
      repo: { owner: "owner", repo: "repo" },
    };
    global.github = {
      rest: {
        checks: {
          update: vi.fn().mockResolvedValue({ data: {} }),
        },
        pulls: {
          get: vi.fn().mockResolvedValue({ data: { state: "open", head: { ref: "gh-aw/pre-created/1-1" }, changed_files: 0 } }),
          update: vi.fn().mockResolvedValue({ data: {} }),
        },
        git: {
          deleteRef: vi.fn().mockResolvedValue({ data: {} }),
        },
      },
    };
    process.env.GH_AW_PRE_CREATED_CHECK_RUN_ID = "99";
  });

  afterEach(() => {
    global.core = originalGlobals.core;
    global.github = originalGlobals.github;
    global.context = originalGlobals.context;
    process.env = originalEnv;
  });

  it("completes the check with the workflow result", async () => {
    process.env.GH_AW_NEEDS = JSON.stringify({
      agent: { result: "success" },
      safe_outputs: { result: "failure" },
    });
    const { main } = await import("./complete_pre_created_check_run.cjs");
    await main();

    expect(global.github.rest.checks.update).toHaveBeenCalledWith(
      expect.objectContaining({
        check_run_id: 99,
        status: "completed",
        conclusion: "failure",
      })
    );
  });

  it("completes the check when downstream job results are malformed", async () => {
    process.env.GH_AW_NEEDS = "{";
    const { main } = await import("./complete_pre_created_check_run.cjs");
    await main();

    expect(global.core.warning).toHaveBeenCalledOnce();
    expect(global.github.rest.checks.update).toHaveBeenCalledWith(expect.objectContaining({ check_run_id: 99, conclusion: "success" }));
  });

  it("completes the check with 'cancelled' when a job was cancelled", async () => {
    process.env.GH_AW_NEEDS = JSON.stringify({
      agent: { result: "cancelled" },
      safe_outputs: { result: "success" },
    });
    const { main } = await import("./complete_pre_created_check_run.cjs");
    await main();

    expect(global.github.rest.checks.update).toHaveBeenCalledWith(expect.objectContaining({ check_run_id: 99, conclusion: "cancelled" }));
  });

  it("does nothing when no check run was pre-created", async () => {
    delete process.env.GH_AW_PRE_CREATED_CHECK_RUN_ID;
    const { main } = await import("./complete_pre_created_check_run.cjs");
    await main();

    expect(global.github.rest.checks.update).not.toHaveBeenCalled();
    expect(global.core.info).toHaveBeenCalledWith("No pre-created pull request check run to complete");
  });

  it("closes the pre-created pull request and deletes its branch when no changes were produced", async () => {
    process.env.GH_AW_NEEDS = JSON.stringify({ agent: { result: "success" } });
    process.env.GH_AW_PRE_CREATED_PULL_REQUEST_NUMBER = "42";
    process.env.GH_AW_PRE_CREATED_PULL_REQUEST_BRANCH = "gh-aw/pre-created/1-1";
    const { main } = await import("./complete_pre_created_check_run.cjs");
    await main();

    expect(global.github.rest.pulls.update).toHaveBeenCalledWith(expect.objectContaining({ pull_number: 42, state: "closed" }));
    expect(global.github.rest.git.deleteRef).toHaveBeenCalledWith(expect.objectContaining({ ref: "heads/gh-aw/pre-created/1-1" }));
  });

  it("keeps the pre-created pull request when the agent contributed changes", async () => {
    process.env.GH_AW_NEEDS = JSON.stringify({ agent: { result: "success" } });
    process.env.GH_AW_PRE_CREATED_PULL_REQUEST_NUMBER = "42";
    process.env.GH_AW_PRE_CREATED_PULL_REQUEST_BRANCH = "gh-aw/pre-created/1-1";
    global.github.rest.pulls.get.mockResolvedValue({
      data: { state: "open", head: { ref: "gh-aw/pre-created/1-1" }, changed_files: 3 },
    });
    const { main } = await import("./complete_pre_created_check_run.cjs");
    await main();

    expect(global.github.rest.pulls.update).not.toHaveBeenCalled();
    expect(global.github.rest.git.deleteRef).not.toHaveBeenCalled();
  });

  it("warns instead of failing when discarding the pull request fails", async () => {
    process.env.GH_AW_NEEDS = JSON.stringify({ agent: { result: "success" } });
    process.env.GH_AW_PRE_CREATED_PULL_REQUEST_NUMBER = "42";
    process.env.GH_AW_PRE_CREATED_PULL_REQUEST_BRANCH = "gh-aw/pre-created/1-1";
    global.github.rest.pulls.update.mockRejectedValue(new Error("boom"));
    const { main } = await import("./complete_pre_created_check_run.cjs");
    await main();

    expect(global.core.warning).toHaveBeenCalledWith(expect.stringContaining("Failed to discard unused pre-created pull request #42"));
  });

  it("does not touch pull requests when no pre-created pull request is configured", async () => {
    process.env.GH_AW_NEEDS = JSON.stringify({ agent: { result: "success" } });
    delete process.env.GH_AW_PRE_CREATED_PULL_REQUEST_NUMBER;
    delete process.env.GH_AW_PRE_CREATED_PULL_REQUEST_BRANCH;
    const { main } = await import("./complete_pre_created_check_run.cjs");
    await main();

    expect(global.github.rest.pulls.get).not.toHaveBeenCalled();
  });
});
