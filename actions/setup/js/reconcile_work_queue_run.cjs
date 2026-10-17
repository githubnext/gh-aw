// @ts-check
"use strict";

const log = require("./work_queue_logging.cjs").createWorkQueueLogger("run");
const { fetchNativeRunAttempt, hasDispatchToken, nativeId, validateNativeRun } = require("./work_queue_native.cjs");
const { expectedWorkerRun, loadQueue } = require("./work_queue_binding.cjs");
const { assignmentForDispatch } = require("./work_queue_replay.cjs");
const { reconcileDispatch } = require("./work_queue_reconciler.cjs");
const { isStagedMode } = require("./safe_output_helpers.cjs");

async function main(options = {}) {
  log.debug("run.reconcile.start");
  try {
    const context = options.context || global.context;
    const githubClient = options.githubClient || global.github;
    const coreApi = options.core || global.core;
    const run = context.payload?.workflow_run;
    if (context.eventName !== "workflow_run" || context.payload?.action !== "completed") {
      log.debug("run.reconcile.ignored", { unsupported_event: true });
      return { status: "ignored" };
    }
    if (context.payload.repository?.fork || run?.event !== "workflow_dispatch" || run.status !== "completed" || nativeId(run.run_attempt) !== "1") {
      log.debug("run.reconcile.ignored", { unsupported_run: true });
      return { status: "ignored" };
    }
    if (isStagedMode(options) || isStagedMode(options.config)) {
      log.debug("run.reconcile.staged");
      return { status: "staged_preview" };
    }
    const runId = nativeId(run.id);
    const configured = { ...options, context, githubClient, core: coreApi };
    const latest = await loadQueue(configured);
    const candidates = [...latest.projection.dispatches.values()].filter(dispatch => dispatch.run?.run_id === runId && dispatch.run.run_attempt === 1);
    log.debug("run.binding.checked", { candidates: candidates.length });
    if (!candidates.length) {
      const unbound = [...latest.projection.dispatches.values()].filter(dispatch => !dispatch.released && !dispatch.run && ["started", "uncertain", "unresolved"].includes(dispatch.state));
      if (unbound.length === 0) {
        log.debug("run.reconcile.unbound");
        return { status: "unbound" };
      }
      const repository = `${context.repo.owner}/${context.repo.repo}`;
      const nativeRun = await fetchNativeRunAttempt(githubClient, repository, runId, 1);
      const recoverable = unbound.filter(dispatch => hasDispatchToken(nativeRun.display_title, dispatch.dispatch_id));
      log.debug("run.recovery.checked", { candidates: recoverable.length });
      if (recoverable.length === 0) {
        log.debug("run.reconcile.unbound");
        return { status: "unbound" };
      }
      if (recoverable.length !== 1) {
        log.debug("run.binding.ambiguous");
        throw new Error("work_queue_run_binding_ambiguous");
      }
      const assignment = assignmentForDispatch(latest.projection, recoverable[0].dispatch_id);
      const expected = expectedWorkerRun(assignment, recoverable[0].profile, context, repository, recoverable[0].profile.principal ?? recoverable[0].credential_principal);
      const proof = validateNativeRun(nativeRun, { ...expected, run_id: runId });
      if (!proof.terminal) {
        log.debug("run.recovery.nonterminal");
        return { status: "nonterminal" };
      }
      // The authenticated run title only selects a candidate; the reconciler
      // independently verifies the exact assignment before it writes a binding.
      const result = await reconcileDispatch({ ...configured, assignment, returnedRunId: runId, requestCancellation: false });
      log.debug("run.reconcile.complete", { released: result.released, claims: assignment.claims.length });
      coreApi?.info(`Work queue terminal reservation: ${result.state}; delivery remains independent`);
      return { dispatch_id: assignment.dispatch_id, ...result };
    }
    if (candidates.length !== 1) {
      log.debug("run.binding.ambiguous");
      throw new Error("work_queue_run_binding_ambiguous");
    }
    const assignment = assignmentForDispatch(latest.projection, candidates[0].dispatch_id);
    // The event selects a stored binding; the reconciler independently authenticates
    // the publisher and original native attempt before any ledger mutation.
    const result = await reconcileDispatch({ ...configured, assignment, requestCancellation: false });
    log.debug("run.reconcile.complete", { released: result.released, claims: assignment.claims.length });
    coreApi?.info(`Work queue terminal reservation: ${result.state}; delivery remains independent`);
    return { dispatch_id: assignment.dispatch_id, ...result };
  } catch (error) {
    log.failure("run.reconcile.failed", error);
    throw error;
  }
}

module.exports = { main };
