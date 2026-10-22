import { describe, it } from "vitest";
import assert from "node:assert/strict";
import { createRequire } from "node:module";

const require = createRequire(import.meta.url);
const { canonical } = require("./work_queue_codec.cjs");
const { administrator, genesis, operationCommit, submission } = require("./work_queue_test_helpers.cjs");
const { compactTransactions, prepareCheckpoint, replayTransactions, serializeProjection } = require("./work_queue_replay.cjs");
const { legacyCheckpoint } = require("./work_queue_checkpoint_regression_helpers.cjs");

function expiringLog() {
  const log = [genesis()];
  log.push(submission(log, ["old", "fresh"], { transform: (work, index) => (index === 0 ? { ...work, expires_at: 50 } : work) }));
  return log;
}

describe("checkpoint and expiration internal-state regressions", () => {
  it("refreshes every public view after an isolated as-of expiration pass", () => {
    const log = expiringLog();
    const original = canonical(log);
    const historical = replayTransactions(log);
    const before = canonical(serializeProjection(historical));
    const viewed = replayTransactions(log, 50);
    const [old, fresh] = [...viewed.works.values()];
    assert.equal(old.state, "cancelled");
    assert.deepEqual(viewed.work, { [old.work_id]: "cancelled", [fresh.work_id]: "available" });
    assert.deepEqual(viewed.available, [fresh.work_id]);
    assert.deepEqual(viewed.winner, historical.winner);
    assert.deepEqual(viewed.claim, historical.claim);
    assert.deepEqual(viewed.stats, { ...historical.stats, available: 1, cancelled: 1 });
    assert.equal(canonical(serializeProjection(historical)), before);
    assert.equal(canonical(log), original);
    assert.deepEqual(replayTransactions(log).available, [old.work_id, fresh.work_id]);
  });

  it("retains accepted explicit expiration cancellations in the checkpoint hash", () => {
    const log = expiringLog();
    const work = log[1].operations[0];
    const operation = { kind: "WorkCancellation", work_id: work.work_id, reason: "work_expired" };
    log.push(operationCommit(log, "cancel-expired", "cancel_work", [operation], administrator, 60));
    const state = replayTransactions(log);
    assert.deepEqual(state.cancellations.get(work.work_id), operation);
    const checkpoint = compactTransactions(log, "a".repeat(40), administrator, 60);
    assert.deepEqual(checkpoint[0].operations[0].state.cancellations[work.work_id], operation);
    assert.deepEqual(replayTransactions(checkpoint).cancellations.get(work.work_id), operation);
  });

  it("defaults both prepareCheckpoint input forms to the committed watermark", () => {
    const log = [genesis()];
    log.push(operationCommit(log, "tick", "control", [{ kind: "Control", control: "grants_paused", value: false, reason: "clock" }], administrator, 60));
    log.push(submission(log, ["backdated"], { at: 2 }));
    for (const source of [log, replayTransactions(log)]) {
      const checkpoint = prepareCheckpoint(source, "a".repeat(40));
      assert.equal(checkpoint[0].at, 60);
      assert.equal(checkpoint[0].operations[0].state.last_at, 60);
    }
    const legacy = legacyCheckpoint(log, "a".repeat(40), administrator, 2);
    for (const source of [legacy, replayTransactions(legacy)]) {
      const checkpoint = prepareCheckpoint(source, "b".repeat(40));
      assert.equal(checkpoint[0].at, 60);
      assert.equal(checkpoint[0].operations[0].state.last_at, 60);
    }
    const future = structuredClone(log);
    future[1].at = Number.MAX_SAFE_INTEGER;
    const checkpoint = compactTransactions(future, "b".repeat(40), administrator);
    assert.equal(checkpoint[0].at, Number.MAX_SAFE_INTEGER);
    assert.throws(() => compactTransactions(log, "a".repeat(40), administrator, null), /checkpoint_invalid/);
    assert.throws(() => prepareCheckpoint(log, "a".repeat(40), administrator, 2), /checkpoint_invalid/);
  });
});
