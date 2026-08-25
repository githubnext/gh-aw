// @ts-check
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import path from "path";
import { fileURLToPath } from "url";

describe("create_steering_issue", () => {
  let originalGlobals;
  let originalEnv;

  beforeEach(() => {
    originalGlobals = {
      core: global.core,
      github: global.github,
      context: global.context,
    };
    originalEnv = { ...process.env };
    global.core = { setOutput: vi.fn(), info: vi.fn() };
    global.context = {
      serverUrl: "https://github.com",
      workflow: "Fallback workflow",
      runId: 123,
      repo: { owner: "owner", repo: "repo" },
    };
    global.github = {
      rest: {
        issues: {
          create: vi.fn().mockResolvedValue({
            data: { number: 42, html_url: "https://github.com/owner/repo/issues/42" },
          }),
        },
      },
    };
    process.env.GH_AW_WORKFLOW_NAME = "Test workflow";
    process.env.GH_AW_PROMPTS_DIR = path.join(path.dirname(fileURLToPath(import.meta.url)), "../md");
  });

  afterEach(() => {
    global.core = originalGlobals.core;
    global.github = originalGlobals.github;
    global.context = originalGlobals.context;
    process.env = originalEnv;
    vi.resetModules();
  });

  it("creates a steering issue and exports its identity", async () => {
    const { main } = await import("./create_steering_issue.cjs");
    await main();

    expect(global.github.rest.issues.create).toHaveBeenCalledWith({
      owner: "owner",
      repo: "repo",
      title: "[WIP] Test workflow: work in progress",
      body: expect.stringContaining("Add a comment containing the keyword `steer`"),
    });
    expect(global.core.setOutput).toHaveBeenCalledWith("issue_number", 42);
    expect(global.core.setOutput).toHaveBeenCalledWith("issue_url", "https://github.com/owner/repo/issues/42");
  });

  it("sanitizes and truncates the title", async () => {
    process.env.GH_AW_WORKFLOW_NAME = `@team ${"A".repeat(300)}`;
    const { main } = await import("./create_steering_issue.cjs");
    await main();

    const title = global.github.rest.issues.create.mock.calls[0][0].title;
    expect(title).toHaveLength(256);
    expect(title).toMatch(/^\[WIP\] `@team` A+/);
  });
});
