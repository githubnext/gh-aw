import { describe, it, expect, beforeEach, afterEach, afterAll, vi } from "vitest";
import { createRequire } from "module";
import crypto from "crypto";
import fs from "fs";
import path from "path";
import { execSync } from "child_process";

const require = createRequire(import.meta.url);

global.core = { debug: vi.fn(), info: vi.fn(), notice: vi.fn(), warning: vi.fn(), error: vi.fn(), setFailed: vi.fn(), setOutput: vi.fn() };
global.context = { repo: { owner: "owner", repo: "repo" }, eventName: "workflow_dispatch", payload: {} };

const { createHandlers } = require("./safe_outputs_handlers.cjs");
const { CONTEXT_FILE } = require("./checkout_continue_workflow_branch.cjs");

const scratch = path.resolve(`.safe-outputs-continue-workflow-${crypto.randomUUID()}`);
const BASE_SHA = "b".repeat(40);

function parse(result) {
  return { isError: result.isError === true, body: JSON.parse(result.content[0].text) };
}

describe("safe_outputs_handlers continue_workflow", () => {
  let appendSafeOutput;
  let handlers;
  let workspace;
  let continuationContext;

  beforeEach(() => {
    vi.clearAllMocks();
    workspace = path.join(scratch, crypto.randomUUID());
    fs.mkdirSync(workspace, { recursive: true });
    process.env.GIT_CEILING_DIRECTORIES = fs.realpathSync(scratch);
    process.env.GITHUB_WORKSPACE = workspace;
    process.env.GITHUB_REPOSITORY = "owner/repo";
    execSync("git init -q -b main && git config user.email t@example.com && git config user.name Test && git commit -q --allow-empty -m base", { cwd: workspace });

    continuationContext = null;
    const existsSync = fs.existsSync;
    const readFileSync = fs.readFileSync;
    vi.spyOn(fs, "existsSync").mockImplementation(file => (file === CONTEXT_FILE ? continuationContext !== null : existsSync(file)));
    vi.spyOn(fs, "readFileSync").mockImplementation((file, ...rest) => (file === CONTEXT_FILE ? continuationContext : readFileSync(file, ...rest)));

    appendSafeOutput = vi.fn();
    handlers = createHandlers({ debug: vi.fn() }, appendSafeOutput, { create_pull_request: { base_branch: "main" }, continue_workflow: { max: 1 } });
  });

  afterEach(() => vi.restoreAllMocks());
  afterAll(() => fs.rmSync(scratch, { recursive: true, force: true }));

  it("requires a reason", async () => {
    const result = parse(await handlers.continueWorkflowHandler({ branch: "work" }));
    expect(result.isError).toBe(true);
    expect(result.body.error).toContain("requires a 'reason'");
    expect(appendSafeOutput).not.toHaveBeenCalled();
  });

  it("refuses to continue once the hop budget is exhausted", async () => {
    continuationContext = JSON.stringify({ branch: "continue-workflow/wf/1", base_sha: BASE_SHA, base_branch: "main", hop: 3, max_hops: 3, remaining_hops: 0 });
    const result = parse(await handlers.continueWorkflowHandler({ reason: "more" }));
    expect(result.isError).toBe(true);
    expect(result.body.error).toContain("budget of 3 hops is exhausted");
    expect(result.body.error).toContain("create_pull_request");
  });

  it("requires continuation hops to commit on the continuation branch", async () => {
    continuationContext = JSON.stringify({ branch: "continue-workflow/wf/1", base_sha: BASE_SHA, base_branch: "main", hop: 1, max_hops: 3, remaining_hops: 2 });
    const result = parse(await handlers.continueWorkflowHandler({ reason: "more", branch: "elsewhere" }));
    expect(result.isError).toBe(true);
    expect(result.body.error).toContain("continue_workflow with branch 'continue-workflow/wf/1'");
  });

  it("rejects a corrupted continuation context", async () => {
    continuationContext = "{not json";
    const result = parse(await handlers.continueWorkflowHandler({ reason: "more" }));
    expect(result.isError).toBe(true);
    expect(result.body.error).toContain("not valid JSON");
  });

  it("requires a work branch other than the base branch", async () => {
    const result = parse(await handlers.continueWorkflowHandler({ reason: "more" }));
    expect(result.isError).toBe(true);
    expect(result.body.error).toContain("other than the base branch 'main'");
  });

  it("makes the final hop open its pull request from the continuation branch", async () => {
    continuationContext = JSON.stringify({ branch: "continue-workflow/wf/1", base_sha: BASE_SHA, base_branch: "main", hop: 1, max_hops: 3, remaining_hops: 2 });
    const result = parse(await handlers.createPullRequestHandler({ title: "Done", body: "Done", branch: "elsewhere" }));
    expect(result.isError).toBe(true);
    expect(result.body.error).toContain("create_pull_request with branch 'continue-workflow/wf/1'");
  });
});
