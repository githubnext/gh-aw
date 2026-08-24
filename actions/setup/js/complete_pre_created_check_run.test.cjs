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
        issues: {
          createComment: vi.fn().mockResolvedValue({ data: {} }),
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

  it("closes the pre-created pull request and deletes its branch when create-pull-request did not consume it", async () => {
    process.env.GH_AW_NEEDS = JSON.stringify({ agent: { result: "success" } });
    process.env.GH_AW_PRE_CREATED_PULL_REQUEST_NUMBER = "42";
    process.env.GH_AW_PRE_CREATED_PULL_REQUEST_BRANCH = "gh-aw/pre-created/1-1";
    const { main } = await import("./complete_pre_created_check_run.cjs");
    await main();

    expect(global.github.rest.pulls.update).toHaveBeenCalledWith(expect.objectContaining({ pull_number: 42, state: "closed" }));
    expect(global.github.rest.git.deleteRef).toHaveBeenCalledWith(expect.objectContaining({ ref: "heads/gh-aw/pre-created/1-1" }));
  });

  it("posts the no-op message before closing the pre-created pull request", async () => {
    process.env.GH_AW_NEEDS = JSON.stringify({ agent: { result: "success" } });
    process.env.GH_AW_PRE_CREATED_PULL_REQUEST_NUMBER = "42";
    process.env.GH_AW_PRE_CREATED_PULL_REQUEST_BRANCH = "gh-aw/pre-created/1-1";
    process.env.GH_AW_NOOP_COMMENT_BODY = "### Test workflow\n\nnothing to do";
    const { main } = await import("./complete_pre_created_check_run.cjs");
    await main();

    expect(global.github.rest.issues.createComment).toHaveBeenCalledWith(expect.objectContaining({ issue_number: 42, body: "### Test workflow\n\nnothing to do" }));
    expect(global.github.rest.pulls.update).toHaveBeenCalledWith(expect.objectContaining({ pull_number: 42, state: "closed" }));
  });

  it("still closes the pre-created pull request when posting the no-op comment fails", async () => {
    process.env.GH_AW_NEEDS = JSON.stringify({ agent: { result: "success" } });
    process.env.GH_AW_PRE_CREATED_PULL_REQUEST_NUMBER = "42";
    process.env.GH_AW_PRE_CREATED_PULL_REQUEST_BRANCH = "gh-aw/pre-created/1-1";
    process.env.GH_AW_NOOP_COMMENT_BODY = "nothing to do";
    global.github.rest.issues.createComment.mockRejectedValue(new Error("boom"));
    const { main } = await import("./complete_pre_created_check_run.cjs");
    await main();

    expect(global.core.warning).toHaveBeenCalledWith(expect.stringContaining("Failed to comment on pre-created pull request #42"));
    expect(global.github.rest.pulls.update).toHaveBeenCalledWith(expect.objectContaining({ pull_number: 42, state: "closed" }));
  });

  it("does not comment when the pre-created pull request is kept", async () => {
    process.env.GH_AW_NEEDS = JSON.stringify({ agent: { result: "success" } });
    process.env.GH_AW_PRE_CREATED_PULL_REQUEST_NUMBER = "42";
    process.env.GH_AW_PRE_CREATED_PULL_REQUEST_BRANCH = "gh-aw/pre-created/1-1";
    process.env.GH_AW_SAFE_OUTPUT_CREATED_PR_NUMBER = "42";
    process.env.GH_AW_NOOP_COMMENT_BODY = "nothing to do";
    const { main } = await import("./complete_pre_created_check_run.cjs");
    await main();

    expect(global.github.rest.issues.createComment).not.toHaveBeenCalled();
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

  it("keeps the pre-created pull request when create-pull-request consumed it", async () => {
    process.env.GH_AW_NEEDS = JSON.stringify({ agent: { result: "success" } });
    process.env.GH_AW_PRE_CREATED_PULL_REQUEST_NUMBER = "42";
    process.env.GH_AW_PRE_CREATED_PULL_REQUEST_BRANCH = "gh-aw/pre-created/1-1";
    process.env.GH_AW_SAFE_OUTPUT_CREATED_PR_NUMBER = "42";
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
  it("links the failure issue on the pre-created pull request", async () => {
    process.env.GH_AW_NEEDS = JSON.stringify({ agent: { result: "failure" } });
    process.env.GH_AW_PRE_CREATED_PULL_REQUEST_NUMBER = "42";
    process.env.GH_AW_FAILURE_ISSUE_NUMBER = "99";
    process.env.GH_AW_FAILURE_ISSUE_URL = "https://github.com/owner/repo/issues/99";
    const { main } = await import("./complete_pre_created_check_run.cjs");
    await main();

    expect(global.github.rest.issues.createComment).toHaveBeenCalledWith({
      owner: "owner",
      repo: "repo",
      issue_number: 42,
      body: "The agent workflow failed. See [failure issue #99](https://github.com/owner/repo/issues/99).",
    });
  });

  it("does not link a failure issue when none was created", async () => {
    process.env.GH_AW_NEEDS = JSON.stringify({ agent: { result: "failure" } });
    process.env.GH_AW_PRE_CREATED_PULL_REQUEST_NUMBER = "42";
    delete process.env.GH_AW_FAILURE_ISSUE_NUMBER;
    delete process.env.GH_AW_FAILURE_ISSUE_URL;
    const { main } = await import("./complete_pre_created_check_run.cjs");
    await main();

    expect(global.github.rest.issues.createComment).not.toHaveBeenCalled();
  });

  it("does not link a failure issue when no pre-created pull request exists", async () => {
    process.env.GH_AW_NEEDS = JSON.stringify({ agent: { result: "failure" } });
    delete process.env.GH_AW_PRE_CREATED_PULL_REQUEST_NUMBER;
    process.env.GH_AW_FAILURE_ISSUE_NUMBER = "99";
    process.env.GH_AW_FAILURE_ISSUE_URL = "https://github.com/owner/repo/issues/99";
    const { main } = await import("./complete_pre_created_check_run.cjs");
    await main();

    expect(global.github.rest.issues.createComment).not.toHaveBeenCalled();
  });

  it("warns instead of failing when linking the failure issue fails", async () => {
    process.env.GH_AW_NEEDS = JSON.stringify({ agent: { result: "failure" } });
    process.env.GH_AW_PRE_CREATED_PULL_REQUEST_NUMBER = "42";
    process.env.GH_AW_PRE_CREATED_PULL_REQUEST_BRANCH = "gh-aw/pre-created/1-1";
    process.env.GH_AW_FAILURE_ISSUE_NUMBER = "99";
    process.env.GH_AW_FAILURE_ISSUE_URL = "https://github.com/owner/repo/issues/99";
    global.github.rest.issues.createComment.mockRejectedValue(new Error("boom"));
    const { main } = await import("./complete_pre_created_check_run.cjs");
    await main();

    expect(global.core.warning).toHaveBeenCalledWith(expect.stringContaining("Failed to add failure issue link to pre-created pull request #42: boom"));
    expect(global.github.rest.checks.update).toHaveBeenCalled();
  });
});
