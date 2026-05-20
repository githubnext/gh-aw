// @ts-check
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

const globals = /** @type {any} */ global;
const { main, GITHUB_API_VERSION } = require("./route_slash_command.cjs");

describe("route_slash_command", () => {
  /** @type {{ core: any, github: any, context: any, exec: any, io: any, getOctokit: any }} */
  let savedGlobals;
  /** @type {any[]} */
  let dispatchCalls;
  /** @type {any[]} */
  let reactionCalls;

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
    reactionCalls = [];
    globals.core = {
      info: vi.fn(),
      warning: vi.fn(),
    };
    globals.github = {
      request: vi.fn(async (...args) => {
        reactionCalls.push(args);
        return { data: { id: 1 } };
      }),
      graphql: vi.fn(async () => ({ repository: { discussion: { id: "D_node" } }, addReaction: { reaction: { id: "R_1" } } })),
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
      payload: { issue: {}, comment: { id: 123456 } },
    };
    globals.exec = {};
    globals.io = {};
    globals.getOctokit = vi.fn();
    process.env.GH_AW_SLASH_ROUTING = JSON.stringify({
      archie: [{ workflow: "archie", events: ["issue_comment", "pull_request_comment"], ai_reaction: "eyes" }],
    });
    process.env.GH_AW_LABEL_ROUTING = JSON.stringify({});
    process.env.GH_AW_REVIEWER_ROUTING = JSON.stringify([]);
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
    delete process.env.GH_AW_LABEL_ROUTING;
    delete process.env.GH_AW_REVIEWER_ROUTING;
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
    expect(dispatchCalls[0].headers?.["X-GitHub-Api-Version"]).toBe(GITHUB_API_VERSION);
    expect(reactionCalls).toHaveLength(1);
    const awContext = JSON.parse(dispatchCalls[0].inputs.aw_context);
    expect(awContext.command_name).toBe("archie");
    expect(awContext.desired_ai_reaction).toBe("eyes");
  });

  it("treats issue_comment on pull requests as pull_request_comment", async () => {
    globals.context.payload.issue.pull_request = { url: "https://example.test/pr/1" };
    globals.context.payload.comment.body = "/archie please";
    await main();
    expect(dispatchCalls).toHaveLength(1);
  });

  it("does not add immediate reaction when no valid route reaction is configured", async () => {
    process.env.GH_AW_SLASH_ROUTING = JSON.stringify({
      archie: [{ workflow: "archie", events: ["issue_comment"], ai_reaction: "none" }],
    });
    globals.context.payload.comment.body = "/archie please";
    await main();
    expect(dispatchCalls).toHaveLength(1);
    expect(reactionCalls).toHaveLength(0);
    const awContext = JSON.parse(dispatchCalls[0].inputs.aw_context);
    expect(awContext.desired_ai_reaction).toBeUndefined();
  });

  it("adds immediate reaction for issues events using issue number", async () => {
    globals.context.eventName = "issues";
    globals.context.payload = { issue: { number: 42, body: "/archie please" } };
    process.env.GH_AW_SLASH_ROUTING = JSON.stringify({
      archie: [{ workflow: "archie", events: ["issues"], ai_reaction: "eyes" }],
    });
    await main();
    expect(dispatchCalls).toHaveLength(1);
    expect(reactionCalls).toHaveLength(1);
    expect(reactionCalls[0][0]).toBe("POST /repos/github/gh-aw/issues/42/reactions");
  });

  it("adds immediate reaction for pull_request events using PR number", async () => {
    globals.context.eventName = "pull_request";
    globals.context.payload = { pull_request: { number: 7, body: "/archie please" } };
    process.env.GH_AW_SLASH_ROUTING = JSON.stringify({
      archie: [{ workflow: "archie", events: ["pull_request"], ai_reaction: "eyes" }],
    });
    await main();
    expect(dispatchCalls).toHaveLength(1);
    expect(reactionCalls).toHaveLength(1);
    expect(reactionCalls[0][0]).toBe("POST /repos/github/gh-aw/issues/7/reactions");
  });

  it("adds immediate reaction for pull_request_review_comment events using comment id", async () => {
    globals.context.eventName = "pull_request_review_comment";
    globals.context.payload = { comment: { id: 99, body: "/archie please" } };
    process.env.GH_AW_SLASH_ROUTING = JSON.stringify({
      archie: [{ workflow: "archie", events: ["pull_request_review_comment"], ai_reaction: "eyes" }],
    });
    await main();
    expect(dispatchCalls).toHaveLength(1);
    expect(reactionCalls).toHaveLength(1);
    expect(reactionCalls[0][0]).toBe("POST /repos/github/gh-aw/pulls/comments/99/reactions");
  });

  it("adds immediate reaction for discussion_comment events using node_id", async () => {
    globals.context.eventName = "discussion_comment";
    globals.context.payload = { comment: { node_id: "DC_node", body: "/archie please" } };
    process.env.GH_AW_SLASH_ROUTING = JSON.stringify({
      archie: [{ workflow: "archie", events: ["discussion_comment"], ai_reaction: "eyes" }],
    });
    await main();
    expect(dispatchCalls).toHaveLength(1);
    expect(globals.github.graphql).toHaveBeenCalledOnce();
    expect(globals.github.graphql.mock.calls[0][1]).toEqual({ subjectId: "DC_node", content: "EYES" });
  });

  it("adds immediate reaction for discussion events by resolving discussion id", async () => {
    globals.context.eventName = "discussion";
    globals.context.payload = { discussion: { number: 3, body: "/archie please" } };
    process.env.GH_AW_SLASH_ROUTING = JSON.stringify({
      archie: [{ workflow: "archie", events: ["discussion"], ai_reaction: "eyes" }],
    });
    await main();
    expect(dispatchCalls).toHaveLength(1);
    expect(globals.github.graphql).toHaveBeenCalledTimes(2);
    expect(globals.github.graphql.mock.calls[0][1]).toEqual({ owner: "github", repo: "gh-aw", num: 3 });
    expect(globals.github.graphql.mock.calls[1][1]).toEqual({ subjectId: "D_node", content: "EYES" });
  });

  it("dispatches matching decentralized label routes for labeled events", async () => {
    globals.context.eventName = "pull_request";
    globals.context.payload = {
      action: "labeled",
      label: { name: "ci-doctor" },
      pull_request: { number: 23 },
    };
    process.env.GH_AW_LABEL_ROUTING = JSON.stringify({
      "ci-doctor": [{ workflow: "ci-doctor", events: ["pull_request"], ai_reaction: "eyes" }],
    });

    await main();

    expect(dispatchCalls).toHaveLength(1);
    expect(dispatchCalls[0].workflow_id).toBe("ci-doctor.lock.yml");
    expect(reactionCalls).toHaveLength(1);
    const awContext = JSON.parse(dispatchCalls[0].inputs.aw_context);
    expect(awContext.command_name).toBe("");
    expect(awContext.trigger_label).toBe("ci-doctor");
    expect(awContext.desired_ai_reaction).toBe("eyes");
  });

  it("skips labeled events when label name is missing", async () => {
    globals.context.eventName = "issues";
    globals.context.payload = { action: "labeled", issue: { number: 1 }, label: {} };
    process.env.GH_AW_LABEL_ROUTING = JSON.stringify({
      smoke: [{ workflow: "smoke-copilot", events: ["issues"] }],
    });

    await main();

    expect(dispatchCalls).toHaveLength(0);
    expect(globals.core.info).toHaveBeenCalledWith(expect.stringContaining("missing label name"));
  });

  it("dispatches all matching routes for a decentralized label", async () => {
    globals.context.eventName = "issues";
    globals.context.payload = { action: "labeled", issue: { number: 1 }, label: { name: "smoke" } };
    process.env.GH_AW_LABEL_ROUTING = JSON.stringify({
      smoke: [
        { workflow: "smoke-copilot", events: ["issues"] },
        { workflow: "ci-doctor", events: ["issues"] },
      ],
    });

    await main();

    expect(dispatchCalls).toHaveLength(2);
    expect(dispatchCalls[0].workflow_id).toBe("smoke-copilot.lock.yml");
    expect(dispatchCalls[1].workflow_id).toBe("ci-doctor.lock.yml");
  });

  it("dispatches reviewer lifecycle routes on pull_request ready_for_review", async () => {
    globals.context.eventName = "pull_request";
    globals.context.payload = { action: "ready_for_review", pull_request: { number: 12, state: "open" } };
    process.env.GH_AW_REVIEWER_ROUTING = JSON.stringify([{ workflow: "pr-reviewer", events: ["pull_request"] }]);

    await main();

    expect(dispatchCalls).toHaveLength(1);
    expect(dispatchCalls[0].workflow_id).toBe("pr-reviewer.lock.yml");
    const awContext = JSON.parse(dispatchCalls[0].inputs.aw_context);
    expect(awContext.command_name).toBe("");
    expect(awContext.reviewer_lifecycle_event).toBe("pull_request");
  });

  it("dispatches reviewer lifecycle routes on pull_request review_requested", async () => {
    globals.context.eventName = "pull_request";
    globals.context.payload = { action: "review_requested", pull_request: { number: 12, state: "open" } };
    process.env.GH_AW_REVIEWER_ROUTING = JSON.stringify([{ workflow: "pr-reviewer", events: ["pull_request"] }]);

    await main();

    expect(dispatchCalls).toHaveLength(1);
    expect(dispatchCalls[0].workflow_id).toBe("pr-reviewer.lock.yml");
    const awContext = JSON.parse(dispatchCalls[0].inputs.aw_context);
    expect(awContext.command_name).toBe("");
    expect(awContext.reviewer_lifecycle_event).toBe("pull_request");
  });

  it("skips centralized routing when PR is closed at workflow start", async () => {
    globals.context.eventName = "pull_request";
    globals.context.payload = { action: "ready_for_review", pull_request: { number: 12, state: "closed" } };
    process.env.GH_AW_REVIEWER_ROUTING = JSON.stringify([{ workflow: "pr-reviewer", events: ["pull_request"] }]);

    await main();

    expect(dispatchCalls).toHaveLength(0);
    expect(globals.core.info).toHaveBeenCalledWith(expect.stringContaining("Pull request is closed at workflow start"));
  });

  it("ignores pull_request actions outside reviewer lifecycle", async () => {
    process.env.GH_AW_REVIEWER_ROUTING = JSON.stringify([{ workflow: "pr-reviewer", events: ["pull_request"] }]);
    for (const action of ["opened", "synchronize"]) {
      dispatchCalls.length = 0;
      globals.context.eventName = "pull_request";
      globals.context.payload = {
        action,
        pull_request: { number: 9, state: "open" },
      };
      await main();
      expect(dispatchCalls).toHaveLength(0);
    }
  });
});
