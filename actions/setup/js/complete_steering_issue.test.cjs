// @ts-check
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

describe("complete_steering_issue", () => {
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
    global.context = { repo: { owner: "owner", repo: "repo" } };
    global.github = {
      rest: {
        issues: {
          createComment: vi.fn().mockResolvedValue({ data: {} }),
          update: vi.fn().mockResolvedValue({ data: {} }),
        },
      },
    };
    process.env.GH_AW_STEERING_ISSUE_NUMBER = "42";
    process.env.GH_AW_NEEDS = JSON.stringify({
      agent: { result: "success" },
      safe_outputs: { result: "success" },
    });
  });

  afterEach(() => {
    global.core = originalGlobals.core;
    global.github = originalGlobals.github;
    global.context = originalGlobals.context;
    process.env = originalEnv;
    vi.resetModules();
  });

  it("closes the steering issue and links the created pull request", async () => {
    process.env.GH_AW_CREATED_PR_NUMBER = "7";
    process.env.GH_AW_CREATED_PR_URL = "https://github.com/owner/repo/pull/7";
    const { main } = await import("./complete_steering_issue.cjs");
    await main();

    expect(global.github.rest.issues.createComment).toHaveBeenCalledWith(
      expect.objectContaining({
        issue_number: 42,
        body: "The workflow completed successfully and created [pull request #7](https://github.com/owner/repo/pull/7).",
      })
    );
    expect(global.github.rest.issues.update).toHaveBeenCalledWith(expect.objectContaining({ issue_number: 42, state: "closed", state_reason: "completed" }));
  });

  it("keeps the issue open when it was reused for failure reporting", async () => {
    process.env.GH_AW_FAILURE_ISSUE_NUMBER = "42";
    const { main } = await import("./complete_steering_issue.cjs");
    await main();

    expect(global.github.rest.issues.update).not.toHaveBeenCalled();
    expect(global.core.info).toHaveBeenCalledWith("Keeping steering issue #42 open as the agent failure issue");
  });

  it("keeps the issue open when a job failed and failure reporting did not complete", async () => {
    process.env.GH_AW_NEEDS = JSON.stringify({ agent: { result: "failure" } });
    const { main } = await import("./complete_steering_issue.cjs");
    await main();

    expect(global.github.rest.issues.update).not.toHaveBeenCalled();
    expect(global.core.info).toHaveBeenCalledWith("Keeping steering issue #42 open because the workflow did not complete successfully");
  });

  it("does nothing when no steering issue was created", async () => {
    delete process.env.GH_AW_STEERING_ISSUE_NUMBER;
    const { main } = await import("./complete_steering_issue.cjs");
    await main();

    expect(global.github.rest.issues.update).not.toHaveBeenCalled();
    expect(global.core.info).toHaveBeenCalledWith("No steering issue to complete");
  });
});
