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
});
