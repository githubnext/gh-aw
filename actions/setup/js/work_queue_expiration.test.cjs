import { describe, it } from "vitest";
import assert from "node:assert/strict";
import { advanceExpiration, expiringGrant } from "./work_queue_expiration.cjs";
import { compactTransactions, generateRequestOperations, replayTransactions, serializeProjection, newRequest } from "./work_queue_replay.cjs";
import { defaultPolicy } from "./work_queue_policy.cjs";
import { planNext, reservationCounts } from "./work_queue_scheduler.cjs";
import { normalizeSubmitParameters } from "./work_queue_intents.cjs";
import { administrator, bind, finish, genesis, grant, operationCommit, submission } from "./work_queue_test_helpers.cjs";

function fixture() {
  const log = [genesis()];
  log.push(submission(log, ["old", "fresh"], { transform: (work, index) => (index === 0 ? { ...work, expires_at: 50 } : work) }));
  return log;
}

describe("unclaimed Work expiration", () => {
  it("uses an isolated expiration view before enforcing pending admission limits", () => {
    const policy = defaultPolicy({ repository: "owner/repo", principal: "1001" });
    policy.limits.pending_nodes = 1;
    const log = [genesis(policy)];
    log.push(submission(log, ["old"], { transform: work => ({ ...work, expires_at: 50 }) }));
    const state = replayTransactions(log);
    const candidate = submission(log, ["new"], { id: "new", at: 50 });
    assert.equal(generateRequestOperations(state, candidate.request, candidate.actor, 50, candidate.id).operations.length, 1);
    assert.equal([...state.works.values()][0].state, "available");
    assert.equal(replayTransactions([...log, candidate]).stats.available, 1);
  });
  it("expires at the exact boundary without changing historical replay or the caller's log", () => {
    const log = fixture();
    for (const at of [49, 50, 51]) {
      const state = replayTransactions(log, at);
      const [old, fresh] = [...state.works.values()];
      assert.equal(old.state, at < 50 ? "available" : "cancelled");
      assert.equal(planNext(state, "default", at).work_id, at < 50 ? old.work_id : fresh.work_id);
      assert.equal(state.stats.cancelled, at < 50 ? 0 : 1);
    }
    assert.equal([...replayTransactions(log).works.values()][0].state, "available");
    for (const at of [-1, null, 1.5, Number.MAX_SAFE_INTEGER + 1]) assert.throws(() => replayTransactions(log, at));
  });

  it("preserves the committed time watermark through rollback, checkpoints and suffix replay", () => {
    const log = fixture();
    log.push(operationCommit(log, "clock", "control", [{ kind: "Control", control: "grants_paused", value: false, reason: "time_observed" }], administrator, 60));
    log.push(submission(log, ["late"], { id: "late", at: 2, transform: work => ({ ...work, expires_at: 50 }) }));
    const state = replayTransactions(log);
    const checkpoint = compactTransactions(log, "a".repeat(40), administrator, 60);
    assert.deepEqual(serializeProjection(replayTransactions(checkpoint, 0)).works, serializeProjection(state).works);
    assert.throws(() => compactTransactions(log, "a".repeat(40), administrator, 2), /checkpoint_invalid/);
    for (const work of state.works.values()) if (work.node_key !== "fresh") assert.equal(work.cancellation_reason, "work_expired");
    const suffix = submission(checkpoint, ["suffix"], { id: "suffix", at: 3, transform: work => ({ ...work, expires_at: 50 }) });
    assert.equal([...replayTransactions([...checkpoint, suffix]).works.values()].at(-1).cancellation_reason, "work_expired");
    assert.deepEqual(serializeProjection(replayTransactions([...log].reverse())).works, serializeProjection(state).works);
  });

  it("leaves running Claims, completed delivery barriers, fairness charges and native capacity unchanged", () => {
    let log = fixture();
    const decision = grant(log);
    log.push(decision.commit);
    const id = decision.assignments[0].dispatch_id;
    log = bind(log, id);
    const before = replayTransactions(log);
    const active = replayTransactions(log, 100);
    assert.equal([...active.claims.values()][0].state, "open");
    assert.equal(reservationCounts(active, "default", "").native, 1);
    log.push(finish(log, id, "h1", "completed", { at: 100 }));
    const completed = replayTransactions(log, 200);
    assert.equal([...completed.claims.values()][0].state, "completed");
    assert.equal([...completed.works.values()][0].barrier, "pending");
    assert.deepEqual(serializeProjection(completed).clocks, serializeProjection(before).clocks);
    assert.equal(reservationCounts(completed, "default", "").native, 1);
  });

  it("expires Work returned by a cancelled Claim without closing a sibling or releasing its run", () => {
    let log = fixture();
    const decision = grant(log);
    log.push(decision.commit);
    const id = decision.assignments[0].dispatch_id;
    log = bind(log, id);
    log.push(finish(log, id, "h1", "cancelled", { at: 100 }));
    const state = replayTransactions(log);
    assert.equal([...state.claims.values()][0].state, "cancelled");
    assert.equal([...state.works.values()][0].cancellation_reason, "work_expired");
    assert.equal(reservationCounts(state, "default", "").native, 1);
  });

  it("rechecks selected deadlines immediately before publication, even without a CAS conflict", () => {
    const log = fixture();
    const state = replayTransactions(log);
    const decision = grant(log);
    assert.equal(
      expiringGrant(state, decision.operations, () => 49),
      false
    );
    assert.equal(
      expiringGrant(state, decision.operations, () => 50),
      true
    );
    const before = serializeProjection(state);
    advanceExpiration(state, 50);
    assert.equal([...state.works.values()][0].cancellation_reason, "work_expired");
    assert.deepEqual(serializeProjection(state).clocks, before.clocks);
  });

  it("normalizes a validated immutable absolute deadline and rejects invalid values or resubmission changes", () => {
    const log = fixture();
    const state = replayTransactions(log);
    const parameters = { nodes: [{ payload: { task: "new" }, expires_at: 100 }] };
    assert.equal(normalizeSubmitParameters(parameters, state.policy, 2, state).nodes[0].expires_at, 100);
    for (const expires_at of [null, -1, 1.5, Number.MAX_SAFE_INTEGER + 1, "100"]) assert.throws(() => normalizeSubmitParameters({ nodes: [{ payload: {}, expires_at }] }, state.policy, 2, state));
    const changed = { ...log[1].operations[0], expires_at: 100 };
    const request = newRequest("change-expiry", "submit", log[1].actor, { nodes: [changed] });
    assert.throws(() => replayTransactions([...log, { ...log[1], id: "changed", previous: log[1].id, request, operations: [changed] }]), /work_conflict/);
  });
});
