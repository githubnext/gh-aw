import { describe, expect, it } from "vitest";
const queue = require("./work_queue_replay.cjs");
const { defaultPolicy } = require("./work_queue_policy.cjs");
const { dependencyStatus } = require("./work_queue_graph.cjs");
const { administrator, bind, context, evidence, finish, genesis, grant, operationCommit, submission, workerActor } = require("./work_queue_test_helpers.cjs");
const { workIssueStatus } = require("./work_queue_issues.cjs");
const { canonical, digest } = require("./work_queue_codec.cjs");
const { deflateRawSync, inflateRawSync } = require("node:zlib");

function fixture() {
  const policy = defaultPolicy({ repository: "owner/repo", principal: "1001" });
  policy.pools.default.profiles.default.max_claims = 3;
  let log = [genesis(policy)];
  log.push(submission(log, ["a", "b", "c"]));
  const granted = grant(log, { max_claims: 3 });
  log.push(granted.commit);
  const assignment = granted.assignments[0];
  log = bind(log, assignment.dispatch_id);
  return { log, assignment };
}

function settle(log, assignment, member, { failed = false, id = "result" } = {}) {
  const state = queue.replayTransactions(log);
  const work = state.works.get(member.work_id);
  const dispatch = state.dispatches.get(assignment.dispatch_id);
  const operation = failed
    ? {
        kind: "DeliveryFailure",
        work_id: member.work_id,
        claim_id: member.claim_id,
        completion_id: work.completion_id,
        reason: "verification_exhausted",
        disposition: "unknown",
        evidence: evidence(state, dispatch, "terminal_run", 50, { run_id: "200", run_attempt: 1, status: "completed", conclusion: "failure", attempts: state.policy.pools.default.reconciliation.max_attempts, effects: "unknown" }),
      }
    : {
        kind: "Result",
        work_id: member.work_id,
        claim_id: member.claim_id,
        completion_id: work.completion_id,
        descriptor: { ok: true },
        evidence: { ...evidence(state, dispatch, "delivery", 50, { run_id: "200", run_attempt: 1, receipt: "verified-result", effects: "none" }), source: "verified_receipts" },
      };
  return [...log, operationCommit(log, id, failed ? "delivery_failure" : "result", [operation], undefined, 50)];
}

describe("keep-open Claim completion", () => {
  it("rejects checkpoints that rewrite the retention choice even with recomputed projection hashes", () => {
    const { log, assignment } = fixture();
    const member = assignment.claims[0];
    log.push(finish(log, assignment.dispatch_id, member.handle, "completed", { keep_open: true }));
    const checkpoint = queue.prepareCheckpoint(log, "a".repeat(40), administrator, 40);
    const operation = checkpoint[0].operations[0];
    const wire = operation.state;
    const requests = JSON.parse(inflateRawSync(Buffer.from(wire.requests_compressed, "base64")).toString("utf8"));
    delete wire.claims[member.claim_id].keep_open;
    for (const receipt of requests) for (const event of receipt.events) if (event.kind === "Completion" && event.claim_id === member.claim_id) delete event.keep_open;
    wire.requests_compressed = deflateRawSync(Buffer.from(canonical(requests))).toString("base64");
    const { requests_compressed, ...state } = wire;
    operation.state_sha256 = digest({ ...state, requests });
    checkpoint[0].request = queue.newRequest(checkpoint[0].request.id, "checkpoint", administrator, { ...checkpoint[0].request.parameters, state_sha256: operation.state_sha256 });
    expect(() => queue.replayTransactions(checkpoint)).toThrow(/checkpoint_invalid/);
  });
  it("retains Work, gates reclaims on delivery, and preserves historical once-only authority", () => {
    let { log, assignment } = fixture();
    const member = assignment.claims[0];
    const child = { ...stateWork(log, member.work_id), work_id: "child", node_key: "child", depends_on: [{ kind: "work", work_id: member.work_id }] };
    log.push(finish(log, assignment.dispatch_id, member.handle, "completed", { keep_open: true }));
    let state = queue.replayTransactions(log);
    const actor = workerActor(state, assignment.dispatch_id, member.handle);
    expect(state.works.get(member.work_id).state).toBe("available");
    expect(state.claims.get(member.claim_id)).toMatchObject({ state: "completed", keep_open: true, barrier: "pending" });
    const trustedContext = context(actor, { ref: state.dispatches.get(assignment.dispatch_id).run.ref, event: "workflow_dispatch" });
    expect(queue.validateClaimAuthority(state, member.claim_id, trustedContext, { requireCompletion: true })).toBeTruthy();
    expect(queue.planNext(state, "default", 40).work_id).toBeUndefined();
    expect(workIssueStatus(state, state.works.get(member.work_id), 40)).toBe("Verifying");
    expect(queue.finishOperations(state, queue.newRequest("retry", "finish", actor, { dispatch_id: assignment.dispatch_id, claim_handle: member.handle, outcome: "completed", keep_open: true }), actor, 40)).toEqual([]);
    expect(() => queue.finishOperations(state, queue.newRequest("conflict", "finish", actor, { dispatch_id: assignment.dispatch_id, claim_handle: member.handle, outcome: "completed" }), actor, 40)).toThrow(/completion_conflict/);
    const pendingCheckpoint = queue.prepareCheckpoint(log, "a".repeat(40), administrator, 40);
    expect(queue.replayTransactions(pendingCheckpoint).claims.get(member.claim_id)).toEqual(state.claims.get(member.claim_id));
    log = settle(log, assignment, member);
    state = queue.replayTransactions(log);
    expect(state.works.get(member.work_id)).toMatchObject({ state: "available", barrier: "none", attempts: 0 });
    expect(state.works.get(member.work_id).claim_id).toBeUndefined();
    expect(state.claims.get(member.claim_id).barrier).toBe("verified");
    expect(dependencyStatus(state, child, 50).reason).toBe("dependency_result_unavailable");
    expect(workIssueStatus(state, state.works.get(member.work_id), 50)).toBe("Queued");
    expect(() => queue.validateClaimAuthority(state, member.claim_id, trustedContext, { requireCompletion: true })).toThrow();
    const granted = grant(log, { id: "reclaim", at: 60 });
    expect(granted.assignments[0].claims[0].work_id).toBe(member.work_id);
    log.push(granted.commit);
    state = queue.replayTransactions(log);
    expect(state.claims.get(member.claim_id).barrier).toBe("verified");
    const checkpoint = queue.prepareCheckpoint(log, "b".repeat(40), administrator, 60);
    expect(queue.replayTransactions(checkpoint).claims.get(member.claim_id)).toEqual(state.claims.get(member.claim_id));
  });

  it("blocks failed retained delivery and permits explicit disposition without reopening effects", () => {
    let { log, assignment } = fixture();
    const member = assignment.claims[0];
    log.push(finish(log, assignment.dispatch_id, member.handle, "completed", { keep_open: true }));
    log = settle(log, assignment, member, { failed: true });
    let state = queue.replayTransactions(log);
    expect(queue.planNext(state, "default", 50).work_id).toBeUndefined();
    expect(state.works.get(member.work_id)).toMatchObject({ state: "available", barrier: "failed" });
    expect(workIssueStatus(state, state.works.get(member.work_id), 50)).toBe("Needs attention");
    log.push(operationCommit(log, "dispose", "cancel_work", [{ kind: "WorkCancellation", work_id: member.work_id, reason: "inspected" }], administrator, 60));
    state = queue.replayTransactions(log);
    expect(state.works.get(member.work_id).state).toBe("cancelled");
    expect(state.claims.get(member.claim_id)).toMatchObject({ state: "completed", barrier: "failed" });
  });

  it("rejects invalid booleans and cancelled keep-open and normalizes false/absent", () => {
    const { log, assignment } = fixture();
    const actor = workerActor(queue.replayTransactions(log), assignment.dispatch_id, "h1");
    const parameters = { dispatch_id: assignment.dispatch_id, claim_handle: "h1", outcome: "completed" };
    for (const keep_open of [null, "true", 1]) expect(() => queue.newRequest("invalid", "finish", actor, { ...parameters, keep_open })).toThrow();
    expect(() => queue.newRequest("cancel", "finish", actor, { ...parameters, outcome: "cancelled", keep_open: true })).toThrow();
    expect(queue.newRequest("same", "finish", actor, parameters).fingerprint).toBe(queue.newRequest("same", "finish", actor, { ...parameters, keep_open: false }).fingerprint);
  });
});

function stateWork(log, id) {
  return queue.replayTransactions(log).works.get(id);
}
