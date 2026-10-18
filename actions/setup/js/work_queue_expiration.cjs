// @ts-check
"use strict";

const { indexesFor, updateIndexes } = require("./work_queue_indexes.cjs");

function workExpired(work, at) {
  return work.expires_at !== undefined && at >= work.expires_at && work.state === "available" && !["pending", "failed"].includes(work.barrier);
}

// Deadlines remove unclaimed Work, never historical Claims or native capacity.
function advanceExpiration(state, at) {
  state.time_watermark = Math.max(state.time_watermark ?? 0, at);
  for (const id of indexesFor(state).expiring) {
    const work = state.works.get(id);
    if (!workExpired(work, state.time_watermark)) continue;
    work.state = "cancelled";
    work.cancellation_reason = "work_expired";
    updateIndexes(state, { work_id: id });
  }
}

function expirationView(state, at) {
  const time = Math.max(at, state.time_watermark ?? 0);
  const due = [...indexesFor(state).expiring].filter(id => workExpired(state.works.get(id), time));
  if (!due.length) return state;
  const view = { ...state, works: new Map(state.works) };
  for (const id of due) view.works.set(id, { ...state.works.get(id) });
  advanceExpiration(view, time);
  return view;
}

function expiringGrant(state, operations, now) {
  const works = operations
    .filter(operation => operation.kind === "Claim")
    .map(operation => state.works.get(operation.work_id))
    .filter(work => work?.expires_at !== undefined);
  if (!works.length) return false;
  const { integer } = require("./work_queue_codec.cjs");
  const at = Math.max(integer(now(), 0, Number.MAX_SAFE_INTEGER, "trusted publication time"), state.time_watermark ?? 0);
  return works.some(work => workExpired(work, at));
}

module.exports = { advanceExpiration, expirationView, expiringGrant, workExpired };
