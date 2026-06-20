// @ts-check
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

const globals = /** @type {any} */ global;
const { main, parseSlashCommand, GITHUB_API_VERSION } = require("./route_slash_command.cjs");

describe("parseSlashCommand", () => {
  it("extracts a simple command name", () => {
    expect(parseSlashCommand("/archie")).toBe("archie");
  });

  it("extracts a command name with dashes", () => {
    expect(parseSlashCommand("/smoke-copilot-sdk")).toBe("smoke-copilot-sdk");
  });

  it("extracts only the command name from text with arguments", () => {
    expect(parseSlashCommand("/archie please do this")).toBe("archie");
  });

  it("extracts a command with dashes from text with arguments", () => {
    expect(parseSlashCommand("/smoke-copilot-sdk run tests")).toBe("smoke-copilot-sdk");
  });

  it("returns empty string when text does not start with a slash command", () => {
    expect(parseSlashCommand("hello /archie")).toBe("");
  });

  it("returns empty string for text starting with just a slash", () => {
    expect(parseSlashCommand("/")).toBe("");
  });

  it("returns empty string for empty string", () => {
    expect(parseSlashCommand("")).toBe("");
  });

  it("trims leading whitespace before matching", () => {
    expect(parseSlashCommand("  /smoke-copilot-sdk")).toBe("smoke-copilot-sdk");
  });

  it("does not match when command is followed by punctuation", () => {
    expect(parseSlashCommand("/smoke-copilot-sdk!")).toBe("");
  });

  it("does not match a slash command in the middle of text", () => {
    expect(parseSlashCommand("some text /archie")).toBe("");
  });

  it("extracts a command name with underscores", () => {
    expect(parseSlashCommand("/code_review")).toBe("code_review");
  });

  it("extracts a command name with dots", () => {
    expect(parseSlashCommand("/cmd.add")).toBe("cmd.add");
  });

  it("does not match a command starting with a dash", () => {
    expect(parseSlashCommand("/-command")).toBe("");
  });

  it("does not match command followed by a colon", () => {
    expect(parseSlashCommand("/archie:more")).toBe("");
  });
});

describe("route_slash_command", () => {
  /** @type {{ core: any, github: any, context: any, exec: any, io: any, getOctokit: any }} */
  let savedGlobals;
  /** @type {any[]} */
  let dispatchCalls;
  /** @type {any[]} */
  let reactionCalls;
  /** @type {any} */
  let summaryMock;

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
    summaryMock = {};
    summaryMock.addHeading = vi.fn(() => summaryMock);
    summaryMock.addRaw = vi.fn(() => summaryMock);
    summaryMock.addEOL = vi.fn(() => summaryMock);
    summaryMock.write = vi.fn(async () => undefined);
    globals.core = {
      info: vi.fn(),
      warning: vi.fn(),
      summary: summaryMock,
    };
    globals.github = {
      request: vi.fn(async (...args) => {
        reactionCalls.push(args);
        return { data: { id: 1 } };
      }),
      graphql: vi.fn(async () => ({ repository: { discussion: { id: "D_node" } }, addReaction: { reaction: { id: "R_1" } } })),
      rest: {
        actions: {
          listRepoWorkflows: vi.fn(async () => ({
            data: {
              workflows: [
                { path: ".github/workflows/archie.lock.yml", state: "active" },
                { path: ".github/workflows/ci-doctor.lock.yml", state: "active" },
                { path: ".github/workflows/smoke-copilot.lock.yml", state: "active" },
              ],
            },
          })),
          createWorkflowDispatch: vi.fn(async params => {
            dispatchCalls.push(params);
          }),
        },
        pulls: {
          get: vi.fn(async ({ pull_number }) => ({
            data: {
              number: pull_number,
              head: { ref: "feature/pr-branch" },
            },
          })),
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
    expect(summaryMock.addRaw).toHaveBeenCalledWith("- Selected command: `/archie`", true);
    expect(summaryMock.addRaw).toHaveBeenCalledWith("- Configured commands: 1", true);
    expect(summaryMock.addRaw).toHaveBeenCalledWith("<details><summary>Configured commands</summary>\n\n- `/archie`\n\n</details>", true);
    expect(summaryMock.write).toHaveBeenCalledWith({ overwrite: false });
  });

  it("logs empty selected command in summary when no slash command is present", async () => {
    globals.context.payload.comment.body = "hello there";

    await main();

    expect(summaryMock.addRaw).toHaveBeenCalledWith("- Selected command: `<none>`", true);
    expect(summaryMock.addRaw).toHaveBeenCalledWith("- Configured commands: 1", true);
    expect(summaryMock.addRaw).toHaveBeenCalledWith("<details><summary>Configured commands</summary>\n\n- `/archie`\n\n</details>", true);
    expect(summaryMock.write).toHaveBeenCalledWith({ overwrite: false });
  });

  it("treats issue_comment on pull requests as pull_request_comment", async () => {
    globals.context.payload.issue.pull_request = { url: "https://example.test/pr/1" };
    globals.context.payload.issue.number = 1;
    globals.context.payload.comment.body = "/archie please";
    await main();
    expect(dispatchCalls).toHaveLength(1);
  });

  it("dispatches slash commands from issue comments on PRs to the PR head branch", async () => {
    globals.context.payload.issue.pull_request = { url: "https://example.test/pr/1" };
    globals.context.payload.issue.number = 1;
    globals.context.payload.comment.body = "/archie please";

    await main();

    expect(globals.github.rest.pulls.get).toHaveBeenCalledWith({
      owner: "github",
      repo: "gh-aw",
      pull_number: 1,
      headers: {
        "X-GitHub-Api-Version": GITHUB_API_VERSION,
      },
    });
    expect(dispatchCalls).toHaveLength(1);
    expect(dispatchCalls[0].ref).toBe("refs/heads/feature/pr-branch");
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

  it("dispatches decentralized label routes on issue-backed PR labels to the PR head branch", async () => {
    globals.context.eventName = "issues";
    globals.context.payload = {
      action: "labeled",
      label: { name: "ci-doctor" },
      issue: {
        number: 23,
        pull_request: { url: "https://example.test/pr/23" },
      },
    };
    process.env.GH_AW_LABEL_ROUTING = JSON.stringify({
      "ci-doctor": [{ workflow: "ci-doctor", events: ["issues"], ai_reaction: "eyes" }],
    });

    await main();

    expect(globals.github.rest.pulls.get).toHaveBeenCalledWith({
      owner: "github",
      repo: "gh-aw",
      pull_number: 23,
      headers: {
        "X-GitHub-Api-Version": GITHUB_API_VERSION,
      },
    });
    expect(dispatchCalls).toHaveLength(1);
    expect(dispatchCalls[0].ref).toBe("refs/heads/feature/pr-branch");
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

  it("skips slash routes when target workflow is disabled", async () => {
    globals.github.rest.actions.createWorkflowDispatch = vi.fn(async () => {
      throw Object.assign(new Error("Workflow was disabled"), {
        status: 422,
        response: { status: 422, data: { message: "Workflow was disabled" } },
      });
    });
    globals.context.payload.comment.body = "/archie please";

    await main();

    expect(dispatchCalls).toHaveLength(0);
    expect(globals.github.rest.actions.listRepoWorkflows).not.toHaveBeenCalled();
    expect(globals.core.info).toHaveBeenCalledWith(expect.stringContaining("Skipping workflow 'archie.lock.yml' because it is disabled."));
  });

  it("skips label routes when target workflow is disabled", async () => {
    globals.github.rest.actions.createWorkflowDispatch = vi.fn(async () => {
      throw Object.assign(new Error("Workflow is disabled"), {
        status: 422,
        response: { status: 422, data: { message: "Workflow is disabled" } },
      });
    });
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

    expect(dispatchCalls).toHaveLength(0);
    expect(globals.github.rest.actions.listRepoWorkflows).not.toHaveBeenCalled();
    expect(globals.core.info).toHaveBeenCalledWith(expect.stringContaining("Skipping workflow 'ci-doctor.lock.yml' because it is disabled."));
  });

  it("ignores disabled workflow_dispatch failures for disabled label routes", async () => {
    globals.github.rest.actions.createWorkflowDispatch = vi.fn(async params => {
      if (params.workflow_id === "smoke-otel-backends.lock.yml") {
        throw Object.assign(new Error("Cannot trigger a 'workflow_dispatch' on a disabled workflow"), {
          status: 422,
          response: { status: 422, data: { message: "Cannot trigger a 'workflow_dispatch' on a disabled workflow" } },
        });
      }
      dispatchCalls.push(params);
    });
    globals.context.eventName = "pull_request";
    globals.context.payload = {
      action: "labeled",
      label: { name: "smoke" },
      pull_request: { number: 23 },
    };
    process.env.GH_AW_LABEL_ROUTING = JSON.stringify({
      smoke: [
        { workflow: "smoke-copilot", events: ["pull_request"] },
        { workflow: "smoke-otel-backends", events: ["pull_request"] },
      ],
    });

    await main();

    expect(dispatchCalls).toHaveLength(1);
    expect(dispatchCalls[0].workflow_id).toBe("smoke-copilot.lock.yml");
    expect(globals.core.info).toHaveBeenCalledWith(expect.stringContaining("Skipping workflow 'smoke-otel-backends.lock.yml' because it is disabled."));
    expect(globals.core.info).toHaveBeenCalledWith(expect.stringContaining("Completed decentralized label routing for 'smoke'."));
  });

  it("skips centralized routing when PR is closed at workflow start", async () => {
    globals.context.eventName = "pull_request";
    globals.context.payload = { action: "ready_for_review", pull_request: { number: 12, state: "closed" } };

    await main();

    expect(dispatchCalls).toHaveLength(0);
    expect(globals.core.info).toHaveBeenCalledWith(expect.stringContaining("Pull request is closed at workflow start"));
  });

  it("dispatches only the exact matching command when command name contains dashes", async () => {
    process.env.GH_AW_SLASH_ROUTING = JSON.stringify({
      smoke: [{ workflow: "smoke", events: ["issue_comment"] }],
      "smoke-copilot": [{ workflow: "smoke-copilot", events: ["issue_comment"] }],
      "smoke-copilot-sdk": [{ workflow: "smoke-copilot-sdk", events: ["issue_comment"] }],
    });
    globals.context.payload.comment.body = "/smoke-copilot-sdk";

    await main();

    expect(dispatchCalls).toHaveLength(1);
    expect(dispatchCalls[0].workflow_id).toBe("smoke-copilot-sdk.lock.yml");
    const awContext = JSON.parse(dispatchCalls[0].inputs.aw_context);
    expect(awContext.command_name).toBe("smoke-copilot-sdk");
  });

  it("dispatches wildcard slash routes using the actual matched command name", async () => {
    process.env.GH_AW_SLASH_ROUTING = JSON.stringify({
      "smoke*": [{ workflow: "smoke-family", events: ["issue_comment"] }],
    });
    globals.context.payload.comment.body = "/smoke-copilot-sdk";

    await main();

    expect(dispatchCalls).toHaveLength(1);
    expect(dispatchCalls[0].workflow_id).toBe("smoke-family.lock.yml");
    const awContext = JSON.parse(dispatchCalls[0].inputs.aw_context);
    expect(awContext.command_name).toBe("smoke-copilot-sdk");
  });

  it("dispatches catch-all slash routes using the actual matched command name", async () => {
    process.env.GH_AW_SLASH_ROUTING = JSON.stringify({
      "*": [{ workflow: "skillet", events: ["issue_comment"] }],
    });
    globals.context.payload.comment.body = "/developer review auth changes";

    await main();

    expect(dispatchCalls).toHaveLength(1);
    expect(dispatchCalls[0].workflow_id).toBe("skillet.lock.yml");
    const awContext = JSON.parse(dispatchCalls[0].inputs.aw_context);
    expect(awContext.command_name).toBe("developer");
  });

  it("does not dispatch smoke-copilot-sdk when command is smoke-copilot", async () => {
    process.env.GH_AW_SLASH_ROUTING = JSON.stringify({
      "smoke-copilot": [{ workflow: "smoke-copilot", events: ["issue_comment"] }],
      "smoke-copilot-sdk": [{ workflow: "smoke-copilot-sdk", events: ["issue_comment"] }],
    });
    globals.context.payload.comment.body = "/smoke-copilot";

    await main();

    expect(dispatchCalls).toHaveLength(1);
    expect(dispatchCalls[0].workflow_id).toBe("smoke-copilot.lock.yml");
  });
});
