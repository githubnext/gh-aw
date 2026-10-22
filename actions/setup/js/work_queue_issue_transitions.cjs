// @ts-check
"use strict";

const { canonical, queueError } = require("./work_queue_codec.cjs");
const { actorFromContext } = require("./work_queue_policy.cjs");

// This authority updates existing AW mirrors only. It cannot publish bindings,
// create resources, close Issues, or acquire another run's original Claims.
function transitionProjectionTargets(state, origin, requestIds) {
  if (
    state.policy.authorization !== "aw" ||
    origin.authenticated !== true ||
    origin.role !== "reconciler" ||
    origin.repository !== state.repository ||
    origin.workflow !== ".github/workflows/agentics-maintenance.yml" ||
    !["schedule", "workflow_dispatch"].includes(origin.event) ||
    !Number.isSafeInteger(origin.run_attempt) ||
    origin.run_attempt < 1 ||
    !/^(?:[a-f0-9]{40}|[a-f0-9]{64})$/.test(origin.ref)
  )
    throw queueError("projection_unauthorized", "terminal projection requires the authenticated original maintenance run");
  if (!Array.isArray(requestIds) || requestIds.length > 16 || new Set(requestIds).size !== requestIds.length) throw queueError("projection_limit", "terminal projection requires at most 16 distinct committed requests");
  const actor = actorFromContext(origin);
  const targets = new Map();
  for (const requestId of requestIds) {
    const receipt = state.requests.get(requestId);
    if (!receipt || canonical(receipt.actor) !== canonical(actor)) throw queueError("projection_unauthorized", "terminal projection request belongs to another publisher");
    for (const operation of receipt.operations) {
      const work = state.works.get(operation.work_id);
      if (!work || work.backing_issue) continue;
      const terminal =
        (operation.kind === "WorkCancellation" && work.state === "cancelled" && work.cancellation_commit_id === receipt.id) ||
        (["Result", "DeliveryFailure"].includes(operation.kind) &&
          work.state === "completed" &&
          work.completion_id === operation.completion_id &&
          work.claim_id === operation.claim_id &&
          canonical(state.terminalBarriers.get(work.work_id)) === canonical(operation));
      if (!terminal) continue;
      if (!work.issue_link || !work.issue_summary) throw queueError("projection_receipt_pending", "terminal synchronization requires existing Issue and canonical summary receipts; recreation is forbidden");
      if (work.issue_link.repository !== state.repository || !state.policy.pools[work.pool].allowed_repositories.includes(state.repository))
        throw queueError("projection_unauthorized", "terminal projection is outside the existing queue mirror repository");
      const claim = state.claims.get(operation.claim_id);
      targets.set(work.work_id, { work_id: work.work_id, claim_ids: claim?.issue_comment ? [claim.claim_id] : [], resource: work.issue_link, transition_request_id: requestId });
    }
  }
  if (targets.size > 256) throw queueError("projection_limit", "terminal projection requires at most 256 existing mirrors");
  return { actor, targets: [...targets.values()] };
}

module.exports = { transitionProjectionTargets };
