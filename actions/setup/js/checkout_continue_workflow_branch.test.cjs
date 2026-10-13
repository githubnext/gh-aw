import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { createRequire } from "module";
import fs from "fs";

const require = createRequire(import.meta.url);

global.core = { info: vi.fn(), warning: vi.fn(), error: vi.fn(), debug: vi.fn(), setFailed: vi.fn(), setOutput: vi.fn(), getInput: vi.fn(() => "checkout-token"), setSecret: vi.fn() };
global.context = { repo: { owner: "o", repo: "r" }, runId: 101, payload: {} };

const ledger = require("./continue_workflow_ledger.cjs");
const { createLedgerFake } = require("./continue_workflow.test_helpers.cjs");
const { main, CONTEXT_FILE } = require("./checkout_continue_workflow_branch.cjs");

const REPO = { owner: "o", repo: "r" };
const NOW = "2026-01-01T00:00:00.000Z";
const CHAIN_ID = "my-workflow-100";
const WORK_BRANCH = "continue-workflow/my-workflow/100";
const HEAD = "b".repeat(40);

function payload(hop = 1, dispatchId = "dispatch-0001") {
  return { inputs: { aw_context: JSON.stringify({ continuation: { chain_id: CHAIN_ID, hop, dispatch_id: dispatchId } }) } };
}

async function seed(github) {
  await ledger.mutateChain(
    github,
    REPO,
    CHAIN_ID,
    current => {
      let { chain } = ledger.startChain(current, {
        chainId: CHAIN_ID,
        runId: "100",
        workflowId: "my-workflow",
        workflowFile: "my-workflow.lock.yml",
        ref: "refs/heads/main",
        repository: "o/r",
        initiator: "octocat",
        baseBranch: "main",
        baseCommit: "a".repeat(40),
        branch: WORK_BRANCH,
        maxHops: 3,
        now: NOW,
      });
      chain = ledger.recordContinuation(chain, { hop: 0, runId: "100", headSha: HEAD, reason: "more", nextSteps: "tests", now: NOW }).chain;
      return ledger.claimDispatch(chain, { dispatchId: "dispatch-0001", now: NOW });
    },
    "Seed"
  );
}

function mockExec(fetchedSha = HEAD) {
  global.exec = {
    exec: vi.fn().mockResolvedValue(0),
    getExecOutput: vi.fn(async (_cmd, args) => (args[0] === "rev-parse" && args[1] === "--is-shallow-repository" ? { exitCode: 0, stdout: "false\n", stderr: "" } : { exitCode: 0, stdout: `${fetchedSha}\n`, stderr: "" })),
  };
}

describe("checkout_continue_workflow_branch", () => {
  let writeSpy;
  beforeEach(() => {
    vi.clearAllMocks();
    global.context.runId = 101;
    global.context.payload = {};
    global.core.getInput.mockReturnValue("checkout-token");
    writeSpy = vi.spyOn(fs, "writeFileSync").mockImplementation(() => {});
    vi.spyOn(fs, "mkdirSync").mockImplementation(() => undefined);
  });
  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllEnvs();
  });

  it("skips runs that are not continuation hops", async () => {
    mockExec();
    global.github = createLedgerFake().github;
    await main();
    expect(global.exec.exec).not.toHaveBeenCalled();
    expect(global.core.setFailed).not.toHaveBeenCalled();
  });

  it("checks out the work branch at the recorded head and writes the context", async () => {
    const { github } = createLedgerFake();
    await seed(github);
    global.github = github;
    global.context.payload = payload();
    mockExec();
    await main();
    expect(global.core.setFailed).not.toHaveBeenCalled();
    const auth = {
      env: expect.objectContaining({
        GIT_CONFIG_COUNT: "1",
        GIT_CONFIG_KEY_0: `http.${(process.env.GITHUB_SERVER_URL || "https://github.com").replace(/\/$/, "")}/.extraheader`,
        GIT_CONFIG_VALUE_0: `Authorization: basic ${Buffer.from("x-access-token:checkout-token").toString("base64")}`,
      }),
      silent: true,
    };
    expect(global.exec.exec).toHaveBeenCalledWith("git", ["fetch", "--no-tags", "origin", `+refs/heads/${WORK_BRANCH}:refs/remotes/origin/${WORK_BRANCH}`], auth);
    expect(global.exec.exec).toHaveBeenCalledWith("git", ["checkout", "-B", WORK_BRANCH, HEAD], auth);
    expect(global.core.setSecret).toHaveBeenCalledWith(Buffer.from("x-access-token:checkout-token").toString("base64"));
    expect(global.exec.exec).not.toHaveBeenCalledWith("git", expect.arrayContaining(["config"]), expect.anything());
    const [file, content] = writeSpy.mock.calls[0];
    expect(file).toBe(CONTEXT_FILE);
    expect(JSON.parse(content)).toMatchObject({
      chain_id: CHAIN_ID,
      hop: 1,
      max_hops: 3,
      remaining_hops: 2,
      branch: WORK_BRANCH,
      base_sha: HEAD,
      base_branch: "main",
      plan_file: "plan.md",
      history: [{ hop: 0, outcome: "continued", next_steps: "tests" }],
    });
  });

  it("fails explicitly rather than attempting an unauthenticated fetch", async () => {
    const { github } = createLedgerFake();
    await seed(github);
    global.github = github;
    global.context.payload = payload();
    global.core.getInput.mockReturnValue("");
    mockExec();
    await main();
    expect(global.core.setFailed).toHaveBeenCalledWith(expect.stringContaining("requires the root checkout token"));
    expect(global.exec.exec).not.toHaveBeenCalled();
  });

  it("refuses to resume when the work branch moved", async () => {
    const { github } = createLedgerFake();
    await seed(github);
    global.github = github;
    global.context.payload = payload();
    mockExec("c".repeat(40));
    await main();
    expect(global.core.setFailed).toHaveBeenCalledWith(expect.stringContaining("refusing to resume"));
    expect(global.exec.exec).not.toHaveBeenCalledWith("git", expect.arrayContaining(["checkout"]));
  });

  it("rejects a dispatch that does not match the ledger", async () => {
    const { github } = createLedgerFake();
    await seed(github);
    global.github = github;
    mockExec();
    global.context.payload = payload(1, "dispatch-forged");
    await main();
    expect(global.core.setFailed).toHaveBeenCalledWith(expect.stringContaining("not the dispatched continuation hop"));
    global.context.payload = payload(2);
    await main();
    expect(global.core.setFailed).toHaveBeenCalledTimes(2);
  });

  it("accepts a re-run of the run already bound to the hop, but not another run", async () => {
    const { github } = createLedgerFake();
    await seed(github);
    await ledger.mutateChain(github, REPO, CHAIN_ID, current => ledger.bindHop(current, { hop: 1, dispatchId: "dispatch-0001", runId: "101", now: NOW }), "Bind");
    global.github = github;
    global.context.payload = payload();
    mockExec();
    await main();
    expect(global.core.setFailed).not.toHaveBeenCalled();
    global.context.runId = 102;
    await main();
    expect(global.core.setFailed).toHaveBeenCalledWith(expect.stringContaining("not the dispatched continuation hop"));
  });

  it("fails for settled or missing chains", async () => {
    const { github } = createLedgerFake();
    global.github = github;
    global.context.payload = payload();
    mockExec();
    await main();
    expect(global.core.setFailed).toHaveBeenCalledWith(expect.stringContaining("does not exist"));
    await seed(github);
    await ledger.mutateChain(github, REPO, CHAIN_ID, current => ledger.failChain(current, { reason: "stop", now: NOW }), "Fail");
    await main();
    expect(global.core.setFailed).toHaveBeenLastCalledWith(expect.stringContaining("is failed"));
  });
});
