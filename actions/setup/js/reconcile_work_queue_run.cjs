// @ts-check
"use strict";

const { nativeId } = require("./work_queue_native.cjs");
const { loadQueue } = require("./work_queue_binding.cjs");
const { assignmentForDispatch } = require("./work_queue_replay.cjs");
const { reconcileDispatch } = require("./work_queue_reconciler.cjs");
const { isStagedMode } = require("./safe_output_helpers.cjs");

async function main(options = {}) {
  const context = options.context || global.context;
  const githubClient = options.githubClient || global.github;
  const coreApi = options.core || global.core;
  const run = context.payload?.workflow_run;
  if (context.eventName !== "workflow_run" || context.payload?.action !== "completed") return { status: "ignored" };
  if (context.payload.repository?.fork || run?.event !== "workflow_dispatch" || run.status !== "completed" || nativeId(run.run_attempt) !== "1") return { status: "ignored" };
  if (isStagedMode(options) || isStagedMode(options.config)) return { status: "staged_preview" };
  const runId = nativeId(run.id);
  const configured = { ...options, context, githubClient, core: coreApi };
  const latest = await loadQueue(configured);
  const candidates = [...latest.projection.dispatches.values()].filter(dispatch => dispatch.run?.run_id === runId && dispatch.run.run_attempt === 1);
  if (!candidates.length) return { status: "unbound" };
  if (candidates.length !== 1) throw new Error("work_queue_run_binding_ambiguous");
  const assignment = assignmentForDispatch(latest.projection, candidates[0].dispatch_id);
  // The event selects a stored binding; the reconciler independently authenticates
  // the publisher and original native attempt before any ledger mutation.
  const result = await reconcileDispatch({ ...configured, assignment, requestCancellation: false });
  coreApi?.info(`Work queue terminal reservation: ${result.state}; delivery remains independent`);
  return { dispatch_id: assignment.dispatch_id, ...result };
}

module.exports = { main };
