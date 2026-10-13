import { describe, it, expect, beforeEach, vi } from "vitest";
import { createRequire } from "module";

const require = createRequire(import.meta.url);

global.core = { info: vi.fn(), warning: vi.fn(), error: vi.fn(), debug: vi.fn() };

const ledger = require("./continue_workflow_ledger.cjs");
const { createLedgerFake } = require("./continue_workflow.test_helpers.cjs");

const SHA_A = "a".repeat(40);
const SHA_B = "b".repeat(40);
const SHA_C = "c".repeat(40);
const NOW = "2026-01-01T00:00:00.000Z";
const REPO = { owner: "o", repo: "r" };

function startInput(overrides = {}) {
  return {
    chainId: "my-workflow-100",
    runId: "100",
    workflowId: "my-workflow",
    workflowFile: "my-workflow.lock.yml",
    ref: "refs/heads/main",
    repository: "o/r",
    initiator: "octocat",
    baseBranch: "main",
    baseCommit: SHA_A,
    branch: "continue-workflow/my-workflow/100",
    maxHops: 3,
    now: NOW,
    ...overrides,
  };
}

function recordInput(overrides = {}) {
  return { hop: 0, runId: "100", headSha: SHA_B, reason: "more work", nextSteps: "finish tests", now: NOW, ...overrides };
}

/** Builds an active chain whose hop `hop` is queued with base `base`. */
function queuedChain(maxHops = 3) {
  const { chain } = ledger.startChain(null, startInput({ maxHops }));
  return ledger.recordContinuation(chain, recordInput()).chain;
}

describe("continue_workflow_ledger", () => {
  beforeEach(() => vi.clearAllMocks());

  describe("helpers", () => {
    it("defaults max hops to 3 and rejects out-of-range values", () => {
      expect(ledger.DEFAULT_MAX_HOPS).toBe(3);
      expect(ledger.parseMaxHops(undefined)).toBe(3);
      expect(ledger.parseMaxHops(0)).toBe(3);
      expect(ledger.parseMaxHops(11)).toBe(3);
      expect(ledger.parseMaxHops("abc")).toBe(3);
      expect(ledger.parseMaxHops(5)).toBe(5);
      expect(ledger.parseMaxHops("10")).toBe(10);
    });

    it("derives stable, path-safe chain ids and work branches", () => {
      expect(ledger.buildChainId("My Workflow!", 42)).toBe("my-workflow-42");
      expect(ledger.buildWorkBranch("My Workflow!", 42, "agent/")).toBe("agent/continue-workflow/my-workflow/42");
      expect(ledger.isValidChainId("../evil")).toBe(false);
      expect(ledger.isValidChainId("ok-1")).toBe(true);
    });

    it("parses continuation requests from aw_context", () => {
      const awContext = { continuation: { chain_id: "wf-1", hop: 2, dispatch_id: "abcdef0123456789" } };
      expect(ledger.parseContinuationRequest({ inputs: { aw_context: JSON.stringify(awContext) } })).toEqual({ chain_id: "wf-1", hop: 2, dispatch_id: "abcdef0123456789" });
      expect(ledger.parseContinuationRequest({ inputs: { aw_context: JSON.stringify({ actor: "x" }) } })).toBeNull();
      expect(ledger.parseContinuationRequest({})).toBeNull();
      expect(() => ledger.parseContinuationRequest({ inputs: { aw_context: JSON.stringify({ continuation: { chain_id: "wf-1", hop: 0, dispatch_id: "abcdef0123456789" } }) } })).toThrow(/hop is invalid/);
      expect(() => ledger.parseContinuationRequest({ inputs: { aw_context: JSON.stringify({ continuation: { chain_id: "../x", hop: 1, dispatch_id: "abcdef0123456789" } }) } })).toThrow(/chain_id is invalid/);
      expect(() => ledger.parseContinuationRequest({ inputs: { aw_context: JSON.stringify({ continuation: { chain_id: "wf-1", hop: 1, dispatch_id: "short" } }) } })).toThrow(/dispatch_id is invalid/);
    });
  });

  describe("transitions", () => {
    it("starts a chain once per run and rejects a different run", () => {
      const first = ledger.startChain(null, startInput());
      expect(first.changed).toBe(true);
      expect(first.chain.next).toMatchObject({ hop: 0, state: "running", run_id: "100", base_sha: SHA_A });
      const withHop = ledger.recordContinuation(first.chain, recordInput()).chain;
      expect(ledger.startChain(withHop, startInput()).changed).toBe(false);
      expect(() => ledger.startChain(withHop, startInput({ runId: "200" }))).toThrow(/already exists/);
      expect(() => ledger.startChain(null, startInput({ workflowFile: "../x.yml" }))).toThrow(/cannot dispatch/);
      expect(() => ledger.startChain(null, startInput({ baseCommit: "main" }))).toThrow(/not a commit SHA/);
    });

    it("records a continuation and queues the next hop exactly once", () => {
      const { chain } = ledger.startChain(null, startInput());
      const recorded = ledger.recordContinuation(chain, recordInput());
      expect(recorded.changed).toBe(true);
      expect(recorded.chain.status).toBe("active");
      expect(recorded.chain.next).toMatchObject({ hop: 1, state: "queued", base_sha: SHA_B });
      expect(recorded.chain.hops).toHaveLength(1);
      // Duplicate delivery of the same hop is a no-op.
      const again = ledger.recordContinuation(recorded.chain, recordInput());
      expect(again.changed).toBe(false);
      expect(again.chain.hops).toHaveLength(1);
      // Recording from a run that does not own the hop is rejected.
      expect(() => ledger.recordContinuation(recorded.chain, recordInput({ hop: 1, runId: "999" }))).toThrow(/not bound/);
    });

    it("exhausts the chain when the hop budget is spent and preserves the branch", () => {
      let chain = ledger.startChain(null, startInput({ maxHops: 1 })).chain;
      chain = ledger.recordContinuation(chain, recordInput()).chain;
      chain = ledger.claimDispatch(chain, { dispatchId: "dispatch-0001", now: NOW }).chain;
      chain = ledger.bindHop(chain, { hop: 1, dispatchId: "dispatch-0001", runId: "101", now: NOW }).chain;
      const result = ledger.recordContinuation(chain, recordInput({ hop: 1, runId: "101", headSha: SHA_C }));
      expect(result.chain.status).toBe("exhausted");
      expect(result.chain.next).toBeNull();
      expect(result.chain.error).toContain("continue-workflow/my-workflow/100");
    });

    it("binds a dispatched hop only for the matching dispatch id and run", () => {
      const queued = queuedChain();
      expect(() => ledger.bindHop(queued, { hop: 1, dispatchId: "dispatch-0001", runId: "101", now: NOW })).toThrow(/dispatch id|not dispatched/);
      const dispatched = ledger.claimDispatch(queued, { dispatchId: "dispatch-0001", now: NOW }).chain;
      expect(() => ledger.bindHop(dispatched, { hop: 2, dispatchId: "dispatch-0001", runId: "101", now: NOW })).toThrow(/expects hop 1/);
      expect(() => ledger.bindHop(dispatched, { hop: 1, dispatchId: "dispatch-9999", runId: "101", now: NOW })).toThrow(/not dispatched with this dispatch id/);
      const bound = ledger.bindHop(dispatched, { hop: 1, dispatchId: "dispatch-0001", runId: "101", now: NOW });
      expect(bound.chain.next).toMatchObject({ state: "running", run_id: "101" });
      // Same run re-binding is idempotent; another run cannot steal the hop.
      expect(ledger.bindHop(bound.chain, { hop: 1, dispatchId: "dispatch-0001", runId: "101", now: NOW }).changed).toBe(false);
      expect(() => ledger.bindHop(bound.chain, { hop: 1, dispatchId: "dispatch-0001", runId: "102", now: NOW })).toThrow(/already bound/);
    });

    it("completes the chain with the single final pull request", () => {
      let chain = ledger.claimDispatch(queuedChain(), { dispatchId: "dispatch-0001", now: NOW }).chain;
      chain = ledger.bindHop(chain, { hop: 1, dispatchId: "dispatch-0001", runId: "101", now: NOW }).chain;
      const completed = ledger.completeChain(chain, { hop: 1, runId: "101", headSha: SHA_C, pullRequest: { number: 7, url: "https://github.com/o/r/pull/7" }, now: NOW });
      expect(completed.chain.status).toBe("completed");
      expect(completed.chain.pull_request).toEqual({ number: 7, url: "https://github.com/o/r/pull/7" });
      expect(ledger.completeChain(completed.chain, { hop: 1, runId: "101", pullRequest: {}, now: NOW }).changed).toBe(false);
    });

    it("fails the chain only for the bound run and is a no-op once settled", () => {
      let chain = ledger.claimDispatch(queuedChain(), { dispatchId: "dispatch-0001", now: NOW }).chain;
      chain = ledger.bindHop(chain, { hop: 1, dispatchId: "dispatch-0001", runId: "101", now: NOW }).chain;
      expect(() => ledger.failChain(chain, { hop: 1, runId: "102", reason: "x", now: NOW })).toThrow(/not bound/);
      const failed = ledger.failChain(chain, { hop: 1, runId: "101", reason: "boom", now: NOW });
      expect(failed.chain.status).toBe("failed");
      expect(failed.chain.error).toBe("boom");
      expect(failed.chain.branch).toBe("continue-workflow/my-workflow/100");
      expect(ledger.failChain(failed.chain, { reason: "again", now: NOW }).changed).toBe(false);
    });

    it("re-queues failed dispatches and fails the chain after the attempt limit", () => {
      let chain = queuedChain();
      for (let attempt = 1; attempt < ledger.MAX_DISPATCH_ATTEMPTS; attempt++) {
        const id = `dispatch-000${attempt}`;
        chain = ledger.claimDispatch(chain, { dispatchId: id, now: NOW }).chain;
        // A stale dispatch id cannot record a failure for a newer attempt.
        expect(ledger.recordDispatchFailure(chain, { dispatchId: "dispatch-other", error: "x", now: NOW }).changed).toBe(false);
        chain = ledger.recordDispatchFailure(chain, { dispatchId: id, error: "rate limited", now: NOW }).chain;
        expect(chain.next).toMatchObject({ state: "queued", dispatch_attempts: attempt, last_error: "rate limited" });
        expect(chain.next.dispatch_id).toBeUndefined();
      }
      chain = ledger.claimDispatch(chain, { dispatchId: "dispatch-last", now: NOW }).chain;
      chain = ledger.recordDispatchFailure(chain, { dispatchId: "dispatch-last", error: "still failing", now: NOW }).chain;
      expect(chain.status).toBe("failed");
      expect(chain.error).toContain("could not be dispatched");
    });

    it("only claims queued hops", () => {
      const dispatched = ledger.claimDispatch(queuedChain(), { dispatchId: "dispatch-0001", now: NOW }).chain;
      expect(ledger.claimDispatch(dispatched, { dispatchId: "dispatch-0002", now: NOW }).changed).toBe(false);
      expect(() => ledger.claimDispatch(queuedChain(), { dispatchId: "bad id!", now: NOW })).toThrow(/dispatch id is invalid/);
    });

    it("expires hops that never settle", () => {
      const dispatched = ledger.claimDispatch(queuedChain(), { dispatchId: "dispatch-0001", now: NOW }).chain;
      expect(ledger.expireStale(dispatched, { now: "2026-01-01T12:00:00.000Z" }).changed).toBe(false);
      const expired = ledger.expireStale(dispatched, { now: "2026-01-02T00:00:01.000Z" });
      expect(expired.chain.status).toBe("failed");
      expect(expired.chain.error).toContain("did not settle");
      // Queued hops are waiting for dispatch, not stale.
      expect(ledger.expireStale(queuedChain(), { now: "2027-01-01T00:00:00.000Z" }).changed).toBe(false);
    });
  });

  describe("ledger validation", () => {
    it.each([
      [{ branch: "main" }, /invalid work branch/],
      [{ branch: "continue-workflow/../main" }, /invalid work branch/],
      [{ branch: "continue-workflow//workflow/100" }, /invalid work branch/],
      [{ base_branch: "main.." }, /invalid base branch/],
      [{ base_branch: "main/.hidden" }, /invalid base branch/],
      [{ workflow_file: "../workflow.yml" }, /invalid workflow file/],
      [{ workflow_file: "workflow.yaml.exe" }, /invalid workflow file/],
      [{ max_hops: 0 }, /invalid max_hops/],
      [{ max_hops: 11 }, /invalid max_hops/],
      [{ max_hops: 1.5 }, /invalid max_hops/],
    ])("rejects unsafe ledger fields %#", (overrides, error) => {
      const chain = queuedChain();
      expect(() => ledger.validateChain({ ...chain, ...overrides }, chain.chain_id)).toThrow(error);
    });

    it("accepts a namespaced branch with a configured prefix", () => {
      const chain = queuedChain();
      const prefixedChain = { ...chain, branch: "agent/continue-workflow/my-workflow/100" };
      expect(ledger.validateChain(prefixedChain, chain.chain_id)).toBe(prefixedChain);
    });
  });

  describe("storage", () => {
    it("creates the ledger branch and reads chains back", async () => {
      const { github, store } = createLedgerFake();
      expect(await ledger.readChain(github, REPO, "my-workflow-100")).toBeNull();
      expect(await ledger.listChainIds(github, REPO)).toEqual([]);
      await ledger.mutateChain(github, REPO, "my-workflow-100", current => ledger.startChain(current, startInput()), "Start");
      expect(store.calls.createRef).toBe(1);
      const chain = await ledger.readChain(github, REPO, "my-workflow-100");
      expect(chain.status).toBe("active");
      expect(await ledger.listChainIds(github, REPO)).toEqual(["my-workflow-100"]);
    });

    it("does not write when a transition is a no-op", async () => {
      const { github, store } = createLedgerFake();
      await ledger.mutateChain(github, REPO, "my-workflow-100", current => ledger.startChain(current, startInput()), "Start");
      const before = store.refs.get(ledger.LEDGER_BRANCH);
      const result = await ledger.mutateChain(github, REPO, "my-workflow-100", current => ledger.startChain(current, startInput()), "Start again");
      expect(result.changed).toBe(false);
      expect(store.refs.get(ledger.LEDGER_BRANCH)).toBe(before);
    });

    it("retries on a concurrent ledger write and keeps both chains", async () => {
      let injected = false;
      const fake = createLedgerFake({
        onBeforeUpdateRef: async () => {
          if (injected) return;
          injected = true;
          // Another workflow records its own chain between our read and our ref update.
          await ledger.mutateChain(fake.github, REPO, "other-200", current => ledger.startChain(current, startInput({ chainId: "other-200", runId: "200", branch: "continue-workflow/other/200" })), "Concurrent start");
        },
      });
      const { github, store } = fake;
      await ledger.mutateChain(github, REPO, "my-workflow-100", current => ledger.startChain(current, startInput()), "Start");
      const result = await ledger.mutateChain(github, REPO, "my-workflow-100", current => ledger.recordContinuation(current, recordInput()), "Record");
      expect(result.changed).toBe(true);
      expect(store.calls.updateRef).toBeGreaterThanOrEqual(3);
      expect((await ledger.listChainIds(github, REPO)).sort()).toEqual(["my-workflow-100", "other-200"]);
      expect((await ledger.readChain(github, REPO, "my-workflow-100")).next).toMatchObject({ hop: 1, state: "queued" });
      expect((await ledger.readChain(github, REPO, "other-200")).status).toBe("active");
    });

    it("loses the race cleanly when a concurrent writer wins between read and update", async () => {
      const fake = createLedgerFake();
      const { github, store } = fake;
      await ledger.mutateChain(github, REPO, "my-workflow-100", current => ledger.startChain(current, startInput()), "Start");
      await ledger.mutateChain(github, REPO, "my-workflow-100", current => ledger.recordContinuation(current, recordInput()), "Record");
      const originalUpdate = github.rest.git.updateRef;
      let raced = false;
      github.rest.git.updateRef = async params => {
        if (!raced) {
          raced = true;
          github.rest.git.updateRef = originalUpdate;
          await ledger.mutateChain(github, REPO, "my-workflow-100", current => ledger.claimDispatch(current, { dispatchId: "dispatch-winner", now: NOW }), "Winner");
        }
        return originalUpdate(params);
      };
      const loser = await ledger.mutateChain(github, REPO, "my-workflow-100", current => ledger.claimDispatch(current, { dispatchId: "dispatch-loser", now: NOW }), "Loser");
      expect(loser.changed).toBe(false);
      expect(store.readFile(ledger.LEDGER_BRANCH, "chains/my-workflow-100.json").next.dispatch_id).toBe("dispatch-winner");
    });

    it("gives up after the CAS attempt limit", async () => {
      const { github } = createLedgerFake();
      await ledger.mutateChain(github, REPO, "my-workflow-100", current => ledger.startChain(current, startInput()), "Start");
      github.rest.git.updateRef = async () => {
        throw Object.assign(new Error("Update is not a fast forward"), { status: 422 });
      };
      await expect(ledger.mutateChain(github, REPO, "my-workflow-100", current => ledger.recordContinuation(current, recordInput()), "Record")).rejects.toThrow(/did not converge/);
    });

    it("rejects corrupted chain documents", async () => {
      const { github } = createLedgerFake();
      await ledger.mutateChain(github, REPO, "my-workflow-100", current => ledger.startChain(current, startInput()), "Start");
      github.rest.repos.getContent = async () => ({ data: { content: Buffer.from("{not json").toString("base64"), encoding: "base64" } });
      await expect(ledger.readChain(github, REPO, "my-workflow-100")).rejects.toThrow(/not valid JSON/);
      github.rest.repos.getContent = async () => ({ data: { content: Buffer.from(JSON.stringify({ version: 1, chain_id: "other" })).toString("base64"), encoding: "base64" } });
      await expect(ledger.readChain(github, REPO, "my-workflow-100")).rejects.toThrow(/does not match/);
      github.rest.repos.getContent = async () => ({ data: { content: Buffer.from("null").toString("base64"), encoding: "base64" } });
      await expect(ledger.readChain(github, REPO, "my-workflow-100")).rejects.toThrow(/not an object/);
    });

    it("persists the dispatch cursor without overwriting a concurrent scan", async () => {
      const { github, store } = createLedgerFake();
      await ledger.mutateChain(github, REPO, "my-workflow-100", current => ledger.startChain(current, startInput()), "Start");
      const initial = await ledger.readLedgerHead(github, REPO, ledger.LEDGER_BRANCH);
      expect(await ledger.readDispatchCursorAt(github, REPO, initial.sha)).toBeNull();
      await ledger.advanceDispatchCursor(github, REPO, null, "my-workflow-100");
      expect((await ledger.advanceDispatchCursor(github, REPO, null, "other-200")).changed).toBe(false);
      expect(store.readFile(ledger.LEDGER_BRANCH, "dispatch-cursor.json").after_chain_id).toBe("my-workflow-100");
      expect((await ledger.advanceDispatchCursor(github, REPO, "my-workflow-100", "other-200")).changed).toBe(true);
      expect((await ledger.readChain(github, REPO, "my-workflow-100")).status).toBe("active");
    });

    it("does not rewind the dispatch cursor after a compare-and-swap conflict", async () => {
      let raced = false;
      const fake = createLedgerFake({
        onBeforeUpdateRef: async () => {
          if (raced) return;
          raced = true;
          await ledger.advanceDispatchCursor(fake.github, REPO, null, "winner-200");
        },
      });
      await ledger.mutateChain(fake.github, REPO, "my-workflow-100", current => ledger.startChain(current, startInput()), "Start");
      expect((await ledger.advanceDispatchCursor(fake.github, REPO, null, "loser-100")).changed).toBe(false);
      expect(fake.store.readFile(ledger.LEDGER_BRANCH, "dispatch-cursor.json").after_chain_id).toBe("winner-200");
    });

    it("fails explicitly when GitHub truncates the chain inventory", async () => {
      const { github } = createLedgerFake();
      await ledger.mutateChain(github, REPO, "my-workflow-100", current => ledger.startChain(current, startInput()), "Start");
      github.rest.git.getTree = async () => ({ data: { tree: [], truncated: true } });
      await expect(ledger.listChainIds(github, REPO)).rejects.toThrow(/ledger tree is truncated/);
    });
  });
});
