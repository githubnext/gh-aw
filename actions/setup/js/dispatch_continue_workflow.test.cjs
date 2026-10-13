import { describe, it, expect, beforeEach, vi } from "vitest";
import { createRequire } from "module";

const require = createRequire(import.meta.url);

const summary = { addRaw: vi.fn(), write: vi.fn() };
summary.addRaw.mockReturnValue(summary);
summary.write.mockResolvedValue(undefined);
global.core = { info: vi.fn(), warning: vi.fn(), error: vi.fn(), debug: vi.fn(), setFailed: vi.fn(), summary };
global.context = { repo: { owner: "o", repo: "r" } };

const ledger = require("./continue_workflow_ledger.cjs");
const { MAX_CHAINS_PER_RUN, main, buildAwContext } = require("./dispatch_continue_workflow.cjs");
const { createLedgerFake } = require("./continue_workflow.test_helpers.cjs");

const REPO = { owner: "o", repo: "r" };
const NOW = "2026-01-01T00:00:00.000Z";
const CHAIN_ID = "my-workflow-100";

async function seedQueuedChain(github, overrides = {}) {
  const chainId = overrides.chainId || CHAIN_ID;
  await ledger.mutateChain(
    github,
    REPO,
    chainId,
    current => {
      const started = ledger.startChain(current, {
        chainId,
        runId: "100",
        workflowId: "my-workflow",
        workflowFile: "my-workflow.lock.yml",
        ref: "refs/heads/main",
        repository: "o/r",
        initiator: "octocat",
        baseBranch: "main",
        baseCommit: "a".repeat(40),
        branch: "continue-workflow/my-workflow/100",
        maxHops: 3,
        now: NOW,
        ...overrides,
      });
      return ledger.recordContinuation(started.chain, { hop: 0, runId: "100", headSha: "b".repeat(40), reason: "more", now: NOW });
    },
    "Seed"
  );
}

describe("dispatch_continue_workflow", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    summary.addRaw.mockReturnValue(summary);
  });

  it("does nothing when no ledger exists", async () => {
    const { github, store } = createLedgerFake();
    const report = await main({ githubClient: github });
    expect(report.dispatched).toEqual([]);
    expect(store.calls.createWorkflowDispatch).toHaveLength(0);
  });

  it("claims the queued hop before dispatching it with a bound aw_context", async () => {
    const { github, store } = createLedgerFake();
    await seedQueuedChain(github);
    const report = await main({ githubClient: github, now: () => NOW, newDispatchId: () => "dispatch-0001" });

    expect(report.dispatched).toHaveLength(1);
    expect(store.calls.createWorkflowDispatch).toHaveLength(1);
    const [dispatch] = store.calls.createWorkflowDispatch;
    expect(dispatch).toMatchObject({ owner: "o", repo: "r", workflow_id: "my-workflow.lock.yml", ref: "refs/heads/main" });
    const awContext = JSON.parse(dispatch.inputs.aw_context);
    expect(awContext).toMatchObject({ actor: "octocat", origin_event: "continue_workflow", root_run_id: "100", continuation: { chain_id: CHAIN_ID, hop: 1, dispatch_id: "dispatch-0001" } });
    // The dispatched input round-trips through the agent-side parser.
    expect(ledger.parseContinuationRequest({ inputs: { aw_context: dispatch.inputs.aw_context } })).toEqual({ chain_id: CHAIN_ID, hop: 1, dispatch_id: "dispatch-0001" });

    const chain = await ledger.readChain(github, REPO, CHAIN_ID);
    expect(chain.next).toMatchObject({ hop: 1, state: "dispatched", dispatch_id: "dispatch-0001", dispatch_attempts: 1 });
  });

  it("never dispatches the same hop twice across maintenance runs", async () => {
    const { github, store } = createLedgerFake();
    await seedQueuedChain(github);
    await main({ githubClient: github, now: () => NOW, newDispatchId: () => "dispatch-0001" });
    await main({ githubClient: github, now: () => NOW, newDispatchId: () => "dispatch-0002" });
    expect(store.calls.createWorkflowDispatch).toHaveLength(1);
  });

  it("falls back to the default branch when the chain has no dispatch ref", async () => {
    const { github, store } = createLedgerFake();
    await seedQueuedChain(github, { ref: "" });
    await main({ githubClient: github, now: () => NOW, newDispatchId: () => "dispatch-0001" });
    expect(store.calls.createWorkflowDispatch[0].ref).toBe("refs/heads/main");
  });

  it("re-queues a failed dispatch so a later run retries it", async () => {
    const { github, store } = createLedgerFake();
    await seedQueuedChain(github);
    store.failNextDispatch(Object.assign(new Error("Server Error"), { status: 500 }));
    const failedRun = await main({ githubClient: github, now: () => NOW, newDispatchId: () => "dispatch-0001" });
    expect(failedRun.requeued).toHaveLength(1);
    const requeued = await ledger.readChain(github, REPO, CHAIN_ID);
    expect(requeued.next).toMatchObject({ state: "queued", dispatch_attempts: 1, last_error: "Server Error" });

    const retry = await main({ githubClient: github, now: () => NOW, newDispatchId: () => "dispatch-0002" });
    expect(retry.dispatched).toHaveLength(1);
    expect(JSON.parse(store.calls.createWorkflowDispatch[0].inputs.aw_context).continuation.dispatch_id).toBe("dispatch-0002");
  });

  it("fails a hop that never settled and reports the preserved branch", async () => {
    const { github, store } = createLedgerFake();
    await seedQueuedChain(github);
    await main({ githubClient: github, now: () => NOW, newDispatchId: () => "dispatch-0001" });
    const report = await main({ githubClient: github, now: () => "2026-01-03T00:00:00.000Z", newDispatchId: () => "dispatch-0002" });
    expect(report.failed).toHaveLength(1);
    expect(report.failed[0]).toContain("continue-workflow/my-workflow/100");
    expect((await ledger.readChain(github, REPO, CHAIN_ID)).status).toBe("failed");
    expect(store.calls.createWorkflowDispatch).toHaveLength(1);
    expect(global.core.warning).toHaveBeenCalledWith(expect.stringContaining("Continuation chain failed"));
  });

  it("skips settled chains without writing to the ledger", async () => {
    const { github, store } = createLedgerFake();
    await seedQueuedChain(github);
    await ledger.mutateChain(github, REPO, CHAIN_ID, current => ledger.failChain(current, { reason: "stopped", now: NOW }), "Fail");
    const head = store.refs.get(ledger.LEDGER_BRANCH);
    const report = await main({ githubClient: github, now: () => NOW, newDispatchId: () => "dispatch-0001" });
    expect(report.dispatched).toEqual([]);
    expect(store.refs.get(ledger.LEDGER_BRANCH)).toBe(head);
  });

  it("bounds historical reads and resumes later pages without starving queued chains", async () => {
    const { github, store } = createLedgerFake();
    for (let index = 0; index < 120; index++) {
      const chainId = `history-${String(index).padStart(3, "0")}`;
      await seedQueuedChain(github, { chainId });
      await ledger.mutateChain(github, REPO, chainId, current => ledger.failChain(current, { reason: "settled", now: NOW }), "Settle");
    }
    await seedQueuedChain(github, { chainId: "z-queued" });
    const snapshots = vi.spyOn(ledger, "readChainAt");
    const contents = vi.spyOn(github.rest.repos, "getContent");
    for (let pass = 0; pass < 3; pass++) {
      snapshots.mockClear();
      contents.mockClear();
      await main({ githubClient: github, now: () => NOW, newDispatchId: () => "dispatch-0001" });
      expect(snapshots).toHaveBeenCalledTimes(MAX_CHAINS_PER_RUN);
      expect(contents.mock.calls.filter(([params]) => params.path.startsWith("chains/")).length).toBeLessThanOrEqual(52);
    }
    expect(store.calls.createWorkflowDispatch).toHaveLength(1);
    expect(JSON.parse(store.calls.createWorkflowDispatch[0].inputs.aw_context).continuation.chain_id).toBe("z-queued");
    expect(store.readFile(ledger.LEDGER_BRANCH, "dispatch-cursor.json").after_chain_id).toBe("history-028");
    snapshots.mockRestore();
    contents.mockRestore();
  });

  it("counts running chains against the scan budget, even when no dispatch is claimed", async () => {
    const { github } = createLedgerFake();
    for (let index = 0; index < 55; index++) {
      const chainId = `running-${String(index).padStart(3, "0")}`;
      await seedQueuedChain(github, { chainId });
      await ledger.mutateChain(github, REPO, chainId, current => ledger.claimDispatch(current, { dispatchId: "dispatch-0001", now: NOW }), "Claim");
    }
    const snapshots = vi.spyOn(ledger, "readChainAt");
    const report = await main({ githubClient: github, now: () => NOW });
    expect(snapshots).toHaveBeenCalledTimes(50);
    expect(report.dispatched).toEqual([]);
    const head = await ledger.readLedgerHead(github, REPO, ledger.LEDGER_BRANCH);
    expect(await ledger.readDispatchCursorAt(github, REPO, head.sha)).toBe("running-049");
    snapshots.mockRestore();
  });

  it("reports chains that cannot be processed", async () => {
    const { github } = createLedgerFake();
    await seedQueuedChain(github);
    const getContent = github.rest.repos.getContent;
    github.rest.repos.getContent = async params => {
      const response = await getContent(params);
      return { data: { ...response.data, content: Buffer.from("{bad").toString("base64") } };
    };
    const report = await main({ githubClient: github, now: () => NOW, newDispatchId: () => "dispatch-0001" });
    expect(report.errors).toHaveLength(1);
    expect(global.core.setFailed).toHaveBeenCalled();
  });

  it("builds aw_context from the chain initiator", () => {
    const awContext = JSON.parse(buildAwContext({ repository: "o/r", initiator: "alice", chain_id: "c-1", hops: [{ hop: 0, run_id: "9" }], next: { hop: 1 } }, "dispatch-0001"));
    expect(awContext).toEqual({ repo: "o/r", actor: "alice", origin_event: "continue_workflow", episode_id: "c-1", root_run_id: "9", continuation: { chain_id: "c-1", hop: 1, dispatch_id: "dispatch-0001" } });
  });
});
