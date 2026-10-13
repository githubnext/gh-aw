import { describe, it, expect, beforeEach, afterAll, vi } from "vitest";
import { createRequire } from "module";
import crypto from "crypto";
import fs from "fs";
import path from "path";
import { spawnSync } from "child_process";

const require = createRequire(import.meta.url);

const summary = { addRaw: vi.fn(), write: vi.fn() };
global.core = { info: vi.fn(), warning: vi.fn(), error: vi.fn(), debug: vi.fn(), notice: vi.fn(), setFailed: vi.fn(), setOutput: vi.fn(), summary };
global.context = { repo: { owner: "o", repo: "r" }, runId: 101, eventName: "workflow_dispatch", actor: "github-actions[bot]", payload: {} };

const ledger = require("./continue_workflow_ledger.cjs");
const continueWorkflow = require("./continue_workflow.cjs");
const { suppressPullRequestWhenContinuing } = require("./safe_output_handler_manager.cjs");
const { createLedgerFake } = require("./continue_workflow.test_helpers.cjs");

const REPO = { owner: "o", repo: "r" };
const NOW = "2026-01-01T00:00:00.000Z";
const CHAIN_ID = "my-workflow-100";
const WORK_BRANCH = "continue-workflow/my-workflow/100";
const SHA_A = "a".repeat(40);
const SHA_B = "b".repeat(40);

function dispatchPayload(hop = 1, dispatchId = "dispatch-0001") {
  return { inputs: { aw_context: JSON.stringify({ actor: "octocat", continuation: { chain_id: CHAIN_ID, hop, dispatch_id: dispatchId } }) } };
}

/** Seeds a chain whose hop 1 is dispatched with `dispatch-0001`. */
async function seedDispatchedChain(github, maxHops = 3) {
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
        baseCommit: SHA_A,
        branch: WORK_BRANCH,
        maxHops,
        now: NOW,
      });
      chain = ledger.recordContinuation(chain, { hop: 0, runId: "100", headSha: SHA_B, reason: "more", now: NOW }).chain;
      return ledger.claimDispatch(chain, { dispatchId: "dispatch-0001", now: NOW });
    },
    "Seed"
  );
}

describe("continue_workflow", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    summary.addRaw.mockReturnValue(summary);
    summary.write.mockResolvedValue(undefined);
    global.context.eventName = "workflow_dispatch";
    global.context.runId = 101;
    global.context.payload = {};
    continueWorkflow.resetContinueWorkflowState();
  });

  describe("prepareContinueWorkflow", () => {
    it("is a no-op for runs that are not continuation hops", async () => {
      const { github } = createLedgerFake();
      expect(await continueWorkflow.prepareContinueWorkflow({}, github)).toBeNull();
      expect(continueWorkflow.getActiveContinuation()).toBeNull();
    });

    it("rejects continuation context outside workflow_dispatch", async () => {
      const { github } = createLedgerFake();
      global.context.eventName = "issues";
      global.context.payload = dispatchPayload();
      await expect(continueWorkflow.prepareContinueWorkflow({}, github)).rejects.toThrow(/only accepted on workflow_dispatch/);
    });

    it("binds the dispatched hop to this run and exposes trusted ledger values", async () => {
      const { github } = createLedgerFake();
      await seedDispatchedChain(github);
      global.context.payload = dispatchPayload();
      const active = await continueWorkflow.prepareContinueWorkflow({}, github);
      expect(active).toMatchObject({ chain_id: CHAIN_ID, hop: 1, branch: WORK_BRANCH, base_branch: "main", base_sha: SHA_B, max_hops: 3, settled: false });
      const chain = await ledger.readChain(github, REPO, CHAIN_ID);
      expect(chain.next).toMatchObject({ state: "running", run_id: "101" });
      // A job re-run of the same run binds idempotently.
      await expect(continueWorkflow.prepareContinueWorkflow({}, github)).resolves.toMatchObject({ settled: false });
    });

    it("rejects a forged dispatch id", async () => {
      const { github } = createLedgerFake();
      await seedDispatchedChain(github);
      global.context.payload = dispatchPayload(1, "dispatch-forged");
      await expect(continueWorkflow.prepareContinueWorkflow({}, github)).rejects.toThrow(/dispatch id/);
    });

    it("rejects a second run for an already bound hop", async () => {
      const { github } = createLedgerFake();
      await seedDispatchedChain(github);
      global.context.payload = dispatchPayload();
      await continueWorkflow.prepareContinueWorkflow({}, github);
      global.context.runId = 102;
      await expect(continueWorkflow.prepareContinueWorkflow({}, github)).rejects.toThrow(/already bound to run 101/);
    });

    it("does not write the ledger in staged mode", async () => {
      const { github, store } = createLedgerFake();
      await seedDispatchedChain(github);
      const head = store.refs.get(ledger.LEDGER_BRANCH);
      global.context.payload = dispatchPayload();
      const active = await continueWorkflow.prepareContinueWorkflow({ staged: true }, github);
      expect(active.chain_id).toBe(CHAIN_ID);
      expect(store.refs.get(ledger.LEDGER_BRANCH)).toBe(head);
    });
  });

  describe("finalizeContinueWorkflow", () => {
    async function bound() {
      const fake = createLedgerFake();
      await seedDispatchedChain(fake.github);
      global.context.payload = dispatchPayload();
      await continueWorkflow.prepareContinueWorkflow({}, fake.github);
      return fake;
    }

    it("completes the chain with the single final pull request", async () => {
      const { github } = await bound();
      continueWorkflow.noteFinalHead("c".repeat(40));
      await continueWorkflow.finalizeContinueWorkflow({}, [{ type: "create_pull_request", success: true, result: { number: 7, url: "https://github.com/o/r/pull/7" } }], github);
      const chain = await ledger.readChain(github, REPO, CHAIN_ID);
      expect(chain.status).toBe("completed");
      expect(chain.pull_request).toEqual({ number: 7, url: "https://github.com/o/r/pull/7" });
      expect(chain.hops.at(-1)).toMatchObject({ hop: 1, outcome: "completed", head_sha: "c".repeat(40) });
      expect(global.core.setFailed).not.toHaveBeenCalled();
    });

    it("leaves a successful continue_workflow hop as recorded", async () => {
      const { github, store } = await bound();
      const head = store.refs.get(ledger.LEDGER_BRANCH);
      await continueWorkflow.finalizeContinueWorkflow({}, [{ type: "continue_workflow", success: true }], github);
      expect(store.refs.get(ledger.LEDGER_BRANCH)).toBe(head);
    });

    it("fails the chain and preserves the branch when continue_workflow fails", async () => {
      const { github } = await bound();
      await continueWorkflow.finalizeContinueWorkflow({}, [{ type: "continue_workflow", success: false, error: "patch too large" }], github);
      const chain = await ledger.readChain(github, REPO, CHAIN_ID);
      expect(chain.status).toBe("failed");
      expect(chain.error).toContain("patch too large");
      expect(chain.branch).toBe(WORK_BRANCH);
      expect(global.core.setFailed).toHaveBeenCalledWith(expect.stringContaining(`Work branch ${WORK_BRANCH} was preserved`));
    });

    it("fails the chain when the hop produced neither output", async () => {
      const { github } = await bound();
      await continueWorkflow.finalizeContinueWorkflow({}, [], github);
      expect((await ledger.readChain(github, REPO, CHAIN_ID)).error).toContain("ended without continue_workflow or a created pull request");
      expect(global.core.setFailed).toHaveBeenCalled();
    });

    it("reports a fallback issue instead of completing the chain", async () => {
      const { github } = await bound();
      await continueWorkflow.finalizeContinueWorkflow({}, [{ type: "create_pull_request", success: true, result: { fallback_used: true, issue_url: "https://github.com/o/r/issues/9" } }], github);
      expect((await ledger.readChain(github, REPO, CHAIN_ID)).status).toBe("failed");
      expect(global.core.warning).toHaveBeenCalledWith(expect.stringContaining("fallback issue"));
      expect(global.core.setFailed).not.toHaveBeenCalled();
    });
  });

  describe("handler", () => {
    beforeEach(() => {
      global.github = createLedgerFake().github;
      process.env.GH_AW_WORKFLOW_ID = "my-workflow";
    });

    it("requires a reason and a branch", async () => {
      const handler = await continueWorkflow.main({});
      expect(await handler({ branch: "x" })).toMatchObject({ success: false, error: expect.stringContaining("requires a reason") });
      const second = await continueWorkflow.main({});
      expect(await second({ reason: "more" })).toMatchObject({ success: false, error: expect.stringContaining("requires the branch") });
    });

    it("processes at most one continuation per run", async () => {
      const handler = await continueWorkflow.main({});
      await handler({ reason: "x" });
      expect(await handler({ reason: "x", branch: "b" })).toMatchObject({ success: false, error: expect.stringContaining("Max count") });
    });

    it("requires the bundle produced by the safe-outputs server", async () => {
      process.env.GITHUB_SHA = SHA_A;
      process.env.GITHUB_REF = "refs/heads/main";
      const handler = await continueWorkflow.main({});
      expect(await handler({ reason: "more", branch: `missing-bundle-${crypto.randomUUID()}` })).toMatchObject({ success: false, error: expect.stringContaining("requires a git bundle") });
    });

    it("rejects progress committed to a branch other than the continuation branch", async () => {
      const { github } = createLedgerFake();
      await seedDispatchedChain(github);
      global.context.payload = dispatchPayload();
      await continueWorkflow.prepareContinueWorkflow({}, github);
      const handler = await continueWorkflow.main({});
      expect(await handler({ reason: "more", branch: "other" })).toMatchObject({ success: false, error: expect.stringContaining(`continuation branch ${WORK_BRANCH}`) });
    });

    it("skips an already settled hop", async () => {
      const { github } = createLedgerFake();
      await seedDispatchedChain(github);
      global.context.payload = dispatchPayload();
      await continueWorkflow.prepareContinueWorkflow({}, github);
      await ledger.mutateChain(github, REPO, CHAIN_ID, current => ledger.recordContinuation(current, { hop: 1, runId: "101", headSha: "c".repeat(40), reason: "x", now: NOW }), "Record");
      await continueWorkflow.prepareContinueWorkflow({}, github);
      const handler = await continueWorkflow.main({});
      expect(await handler({ reason: "more", branch: WORK_BRANCH })).toMatchObject({ success: true, skipped: true });
    });
  });

  describe("suppressPullRequestWhenContinuing", () => {
    it("skips create_pull_request when the run continues the workflow", async () => {
      const cpr = vi.fn();
      const handlers = new Map([
        ["continue_workflow", vi.fn()],
        ["create_pull_request", cpr],
      ]);
      suppressPullRequestWhenContinuing(handlers, [{ type: "create_pull_request" }, { type: "continue_workflow" }]);
      const result = await handlers.get("create_pull_request")({});
      expect(result).toMatchObject({ success: true, skipped: true });
      expect(cpr).not.toHaveBeenCalled();
    });

    it("keeps create_pull_request for the final hop", () => {
      const cpr = vi.fn();
      const handlers = new Map([
        ["continue_workflow", vi.fn()],
        ["create_pull_request", cpr],
      ]);
      suppressPullRequestWhenContinuing(handlers, [{ type: "create_pull_request" }]);
      expect(handlers.get("create_pull_request")).toBe(cpr);
    });
  });
});

describe("persistContinuationHop (real git)", () => {
  const scratch = path.resolve(`.continue-workflow-test-${crypto.randomUUID()}`);
  const originDir = path.join(scratch, "origin.git");
  const agentDir = path.join(scratch, "agent");
  const jobDir = path.join(scratch, "job");
  const gitEnv = { ...process.env, GIT_CONFIG_NOSYSTEM: "1", GIT_CONFIG_GLOBAL: path.join(scratch, "gitconfig"), GIT_CEILING_DIRECTORIES: scratch };

  function git(cwd, args, allowFailure = false) {
    // The bare origin is addressed explicitly (safe.bareRepository may be "explicit").
    const fullArgs = cwd === originDir ? ["--git-dir", originDir, ...args] : args;
    const result = spawnSync("git", fullArgs, { cwd: cwd === originDir ? scratch : cwd, encoding: "utf8", env: gitEnv });
    if (result.status !== 0 && !allowFailure) throw new Error(`git ${args.join(" ")} failed: ${result.stderr}`);
    return result;
  }

  function commitFile(cwd, file, content, message) {
    fs.writeFileSync(path.join(cwd, file), content);
    git(cwd, ["add", file]);
    git(cwd, ["commit", "-q", "-m", message]);
    return git(cwd, ["rev-parse", "HEAD"]).stdout.trim();
  }

  function bundle(name, ref, exclude) {
    const file = path.join(scratch, `${name}.bundle`);
    git(agentDir, ["bundle", "create", file, ref, `^${exclude}`]);
    return file;
  }

  // Remote branch reads go to the bare origin; everything else to the ledger fake.
  function githubFor(verified = false) {
    const { github } = createLedgerFake();
    github.rest.git.getRef = async ({ ref }) => {
      const result = git(originDir, ["rev-parse", "--verify", "-q", `refs/${ref}^{commit}`], true);
      if (result.status !== 0) throw Object.assign(new Error("Not Found"), { status: 404 });
      return { data: { object: { sha: result.stdout.trim() } } };
    };
    github.rest.git.getCommit = async ({ commit_sha }) => ({
      data: {
        sha: commit_sha,
        tree: { sha: git(originDir, ["rev-parse", `${commit_sha}^{tree}`]).stdout.trim() },
        parents: git(originDir, ["rev-list", "--parents", "-1", commit_sha])
          .stdout.trim()
          .split(" ")
          .slice(1)
          .map(sha => ({ sha })),
        message: git(originDir, ["log", "-1", "--format=%B", commit_sha]).stdout.trim(),
        verification: { verified, reason: verified ? "valid" : "unsigned" },
      },
    });
    return github;
  }

  function replayCommit(localSha, parent, message = git(agentDir, ["log", "-1", "--format=%B", localSha]).stdout.trim()) {
    return git(agentDir, ["-c", "user.name=Replay", "-c", "user.email=replay@example.com", "commit-tree", `${localSha}^{tree}`, ...(parent ? ["-p", parent] : []), "-m", message]).stdout.trim();
  }

  function pushReplay(sha) {
    git(agentDir, ["push", "-q", "--force", "origin", `${sha}:refs/heads/${WORK_BRANCH}`]);
  }

  function execIn(cwd) {
    const run = (args, opts = {}) => spawnSync("git", args, { cwd, encoding: "utf8", env: { ...gitEnv, ...(opts.env || {}) } });
    return {
      getExecOutput: async (program, args, opts = {}) => {
        if (program !== "git") throw new Error(`unexpected command ${program}`);
        const result = run(args, opts);
        if (result.status !== 0 && !opts.ignoreReturnCode) throw new Error(result.stderr);
        return { exitCode: result.status ?? 0, stdout: result.stdout ?? "", stderr: result.stderr ?? "" };
      },
      exec: async (program, args, opts = {}) => {
        if (program !== "git") throw new Error(`unexpected command ${program}`);
        const result = run(args, opts);
        if (result.status !== 0 && !opts.ignoreReturnCode) throw new Error(result.stderr);
        return result.status ?? 0;
      },
    };
  }

  let baseSha;
  const config = { signed_commits: false };

  function persist(options) {
    return continueWorkflow.persistContinuationHop({ agentBranch: "agent", workBranch: WORK_BRANCH, baseBranch: "main", config, githubClient: githubFor(), ...options });
  }

  beforeEach(() => {
    fs.rmSync(scratch, { recursive: true, force: true });
    fs.mkdirSync(scratch, { recursive: true });
    fs.writeFileSync(path.join(scratch, "gitconfig"), "[user]\n\tname = Test\n\temail = test@example.com\n[init]\n\tdefaultBranch = main\n");
    git(scratch, ["init", "-q", "--bare", originDir]);
    git(scratch, ["init", "-q", agentDir]);
    baseSha = commitFile(agentDir, "README.md", "base\n", "base");
    git(agentDir, ["remote", "add", "origin", originDir]);
    git(agentDir, ["push", "-q", "origin", "main"]);
    git(scratch, ["clone", "-q", originDir, jobDir]);
    global.exec = execIn(jobDir);
  });

  afterAll(() => fs.rmSync(scratch, { recursive: true, force: true }));

  it("creates the work branch, then advances it hop by hop with compare-and-swap", async () => {
    git(agentDir, ["checkout", "-q", "-b", "agent"]);
    const hop0 = commitFile(agentDir, "plan.md", "# Plan\n- [x] step 1\n", "hop 0");
    const first = await persist({ bundlePath: bundle("hop0", "agent", baseSha), expectedBase: baseSha, isInitial: true });
    expect(first).toMatchObject({ headSha: hop0, alreadyPersisted: false });
    expect(git(originDir, ["rev-parse", WORK_BRANCH]).stdout.trim()).toBe(hop0);

    // Duplicate delivery of the same hop is recognized instead of pushed twice.
    const duplicate = await persist({ bundlePath: bundle("hop0-again", "agent", baseSha), expectedBase: baseSha, isInitial: true });
    expect(duplicate).toMatchObject({ headSha: hop0, alreadyPersisted: true });

    const hop1 = commitFile(agentDir, "plan.md", "# Plan\n- [x] step 1\n- [x] step 2\n", "hop 1");
    const second = await persist({ bundlePath: bundle("hop1", "agent", hop0), expectedBase: hop0, isInitial: false });
    expect(second).toMatchObject({ headSha: hop1, alreadyPersisted: false });
    expect(git(originDir, ["rev-parse", WORK_BRANCH]).stdout.trim()).toBe(hop1);
  });

  it("refuses to overwrite a work branch that moved unexpectedly", async () => {
    git(agentDir, ["checkout", "-q", "-b", "agent"]);
    const hop0 = commitFile(agentDir, "plan.md", "# Plan\n", "hop 0");
    await persist({ bundlePath: bundle("hop0", "agent", baseSha), expectedBase: baseSha, isInitial: true });
    // Someone else pushes to the work branch.
    git(agentDir, ["checkout", "-q", "-b", "intruder", hop0]);
    commitFile(agentDir, "other.txt", "x\n", "intruder");
    git(agentDir, ["push", "-q", "origin", `intruder:${WORK_BRANCH}`]);
    git(agentDir, ["checkout", "-q", "agent"]);
    commitFile(agentDir, "plan.md", "# Plan\n- step 2\n", "hop 1");
    await expect(persist({ bundlePath: bundle("hop1", "agent", hop0), expectedBase: hop0, isInitial: false })).rejects.toThrow(/moved unexpectedly/);
  });

  it("rejects an unsigned same-tree commit that is not the original delivery", async () => {
    git(agentDir, ["checkout", "-q", "-b", "agent"]);
    const hop = commitFile(agentDir, "plan.md", "# Plan\n", "hop 0");
    const replacement = replayCommit(hop, baseSha);
    pushReplay(replacement);
    await expect(persist({ bundlePath: bundle("retry", "agent", baseSha), expectedBase: baseSha, isInitial: true })).rejects.toThrow(/moved unexpectedly/);
    expect(git(originDir, ["rev-parse", WORK_BRANCH]).stdout.trim()).toBe(replacement);
  });

  it("recovers a verified signed replay of every commit rooted at the expected base", async () => {
    git(agentDir, ["checkout", "-q", "-b", "agent"]);
    const first = commitFile(agentDir, "plan.md", "# Plan\n", "hop 0\n\nDetailed handoff");
    const second = commitFile(agentDir, "plan.md", "# Plan\n- step 2\n", "hop 1");
    const signedFirst = replayCommit(first, baseSha);
    const signedSecond = replayCommit(second, signedFirst);
    pushReplay(signedSecond);
    const result = await persist({ bundlePath: bundle("signed-retry", "agent", baseSha), expectedBase: baseSha, isInitial: true, config: { signed_commits: true }, githubClient: githubFor(true) });
    expect(result).toMatchObject({ headSha: signedSecond, alreadyPersisted: true });
  });

  it("rejects a signed same-tree replay with unrelated parentage", async () => {
    git(agentDir, ["checkout", "-q", "-b", "agent"]);
    const hop = commitFile(agentDir, "plan.md", "# Plan\n", "hop 0");
    const foreignBase = replayCommit(baseSha, null);
    const foreignHead = replayCommit(hop, foreignBase);
    pushReplay(foreignHead);
    await expect(persist({ bundlePath: bundle("foreign-retry", "agent", baseSha), expectedBase: baseSha, isInitial: true, config: { signed_commits: true }, githubClient: githubFor(true) })).rejects.toThrow(/moved unexpectedly/);
    expect(git(originDir, ["rev-parse", WORK_BRANCH]).stdout.trim()).toBe(foreignHead);
  });

  it("rejects an unsigned intermediate commit even when the replay head is verified", async () => {
    git(agentDir, ["checkout", "-q", "-b", "agent"]);
    const first = commitFile(agentDir, "plan.md", "# Plan\n", "hop 0");
    const second = commitFile(agentDir, "plan.md", "# Plan\n- step 2\n", "hop 1");
    const unsignedFirst = replayCommit(first, baseSha);
    const signedSecond = replayCommit(second, unsignedFirst);
    pushReplay(signedSecond);
    const github = githubFor(true);
    const getCommit = github.rest.git.getCommit;
    github.rest.git.getCommit = async params => {
      const response = await getCommit(params);
      if (params.commit_sha === unsignedFirst) response.data.verification = { verified: false, reason: "unsigned" };
      return response;
    };
    await expect(persist({ bundlePath: bundle("unsigned-retry", "agent", baseSha), expectedBase: baseSha, isInitial: true, config: { signed_commits: true }, githubClient: github })).rejects.toThrow(/moved unexpectedly/);
  });

  it("rejects a signed same-tree commit with a different message", async () => {
    git(agentDir, ["checkout", "-q", "-b", "agent"]);
    const hop = commitFile(agentDir, "plan.md", "# Plan\n", "hop 0");
    pushReplay(replayCommit(hop, baseSha, "unrelated commit"));
    await expect(persist({ bundlePath: bundle("message-retry", "agent", baseSha), expectedBase: baseSha, isInitial: true, config: { signed_commits: true }, githubClient: githubFor(true) })).rejects.toThrow(/moved unexpectedly/);
  });

  it("does not record a branch that moved between push and readback", async () => {
    git(agentDir, ["checkout", "-q", "-b", "agent"]);
    const hop = commitFile(agentDir, "plan.md", "# Plan\n", "hop 0");
    const github = githubFor();
    const getRef = github.rest.git.getRef;
    let reads = 0;
    github.rest.git.getRef = async params => {
      if (++reads === 2) pushReplay(replayCommit(hop, baseSha, "changed after push"));
      return getRef(params);
    };
    await expect(persist({ bundlePath: bundle("raced-push", "agent", baseSha), expectedBase: baseSha, isInitial: true, githubClient: github })).rejects.toThrow(/moved after push/);
  });

  it("rejects commits that do not build on the recorded head", async () => {
    git(agentDir, ["checkout", "-q", "-b", "agent"]);
    const hop0 = commitFile(agentDir, "plan.md", "# Plan\n", "hop 0");
    await persist({ bundlePath: bundle("hop0", "agent", baseSha), expectedBase: baseSha, isInitial: true });
    // The agent restarts from main instead of resuming the work branch.
    git(agentDir, ["checkout", "-q", "-B", "agent", baseSha]);
    commitFile(agentDir, "plan.md", "# Restarted\n", "restart");
    await expect(persist({ bundlePath: bundle("restart", "agent", baseSha), expectedBase: hop0, isInitial: false })).rejects.toThrow(/do not build on the work branch head/);
  });

  it("rejects a hop without new commits", async () => {
    git(agentDir, ["checkout", "-q", "-b", "agent"]);
    const hop0 = commitFile(agentDir, "plan.md", "# Plan\n", "hop 0");
    await persist({ bundlePath: bundle("hop0", "agent", baseSha), expectedBase: baseSha, isInitial: true });
    git(jobDir, ["fetch", "-q", "origin", WORK_BRANCH]);
    git(jobDir, ["checkout", "-q", "main"]);
    git(jobDir, ["branch", "-D", WORK_BRANCH], true);
    await expect(persist({ bundlePath: bundle("hop0-again", "agent", baseSha), expectedBase: hop0, isInitial: false })).rejects.toThrow(/no new commits/);
  });
});
