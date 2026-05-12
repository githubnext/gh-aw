// @ts-check
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

const globals = /** @type {any} */ global;
const { main } = require("./route_slash_command.cjs");

describe("route_slash_command", () => {
  /** @type {{ core: any, github: any, context: any, exec: any, io: any, getOctokit: any }} */
  let savedGlobals;
  /** @type {any[]} */
  let dispatchCalls;

  beforeEach(() => {
    savedGlobals = {
      core: globals.core,
      github: globals.github,
      context: globals.context,
      exec: globals.exec,
      io: globals.io,
      getOctokit: globals.getOctokit,
    };
    dispatchCalls = [];
    globals.core = {
      info: vi.fn(),
    };
    globals.github = {
      rest: {
        actions: {
          createWorkflowDispatch: vi.fn(async params => {
            dispatchCalls.push(params);
          }),
        },
      },
    };
    globals.context = {
      eventName: "issue_comment",
      ref: "refs/heads/main",
      repo: { owner: "github", repo: "gh-aw" },
      payload: { issue: {}, comment: {} },
    };
    globals.exec = {};
    globals.io = {};
    globals.getOctokit = vi.fn();
    process.env.GH_AW_SLASH_ROUTING = JSON.stringify({
      archie: [{ workflow: "archie", events: ["issue_comment", "pull_request_comment"] }],
    });
    process.env.GITHUB_WORKSPACE = `${process.cwd()}`;
  });

  afterEach(() => {
    globals.core = savedGlobals.core;
    globals.github = savedGlobals.github;
    globals.context = savedGlobals.context;
    globals.exec = savedGlobals.exec;
    globals.io = savedGlobals.io;
    globals.getOctokit = savedGlobals.getOctokit;
    delete process.env.GH_AW_SLASH_ROUTING;
    delete process.env.GITHUB_WORKSPACE;
    delete process.env.GITHUB_REF;
    delete process.env.GITHUB_HEAD_REF;
    vi.restoreAllMocks();
  });

  it("skips dispatch when text does not start with slash command", async () => {
    globals.context.payload.comment.body = "hello /archie";
    await main();
    expect(dispatchCalls).toHaveLength(0);
    expect(globals.core.info).toHaveBeenCalledWith(expect.stringContaining("No slash command found"));
  });

  it("dispatches only matching command and event routes", async () => {
    globals.context.payload.comment.body = "/archie please";
    await main();
    expect(dispatchCalls).toHaveLength(1);
    expect(dispatchCalls[0].workflow_id).toBe("archie.lock.yml");
  });

  it("treats issue_comment on pull requests as pull_request_comment", async () => {
    globals.context.payload.issue.pull_request = { url: "https://example.test/pr/1" };
    globals.context.payload.comment.body = "/archie please";
    await main();
    expect(dispatchCalls).toHaveLength(1);
  });
});
