// @ts-check
"use strict";

const { createHash } = require("node:crypto");
const { canonical, closed, identity, integer } = require("./work_queue_codec.cjs");
const { normalizeClaimScope } = require("./work_queue_claim_scope.cjs");
const { readStagedIntents } = require("./work_queue_intents.cjs");

const MAX_CONTINUATION_HOPS = 3;
const MAX_HANDOFF_BYTES = 16 * 1024;

function continuationLineage(state, work) {
  const descendants = [];
  const visited = new Set();
  let current = work;
  while (Object.hasOwn(current.payload, "work_queue_continuation")) {
    if (visited.has(current.work_id) || descendants.length >= 10) throw new Error("work_queue_continuation_lineage_invalid");
    visited.add(current.work_id);
    const continuation = current.payload.work_queue_continuation;
    closed(continuation, ["root_work_id", "previous_work_id", "hop", "handoff"], [], "Work continuation");
    identity(continuation.root_work_id, "continuation root Work");
    identity(continuation.previous_work_id, "continuation predecessor Work");
    integer(continuation.hop, 1, 10, "continuation hop");
    validateHandoff(continuation.handoff);
    const creator = state.workCreators.get(current.work_id);
    const dispatch = creator?.role === "worker" ? state.dispatches.get(creator.dispatch_id) : undefined;
    const member = dispatch?.claims.find(claim => claim.handle === creator.claim_handle);
    const parent = member && state.works.get(member.work_id);
    if (!parent || parent.work_id !== continuation.previous_work_id || parent.graph_id !== current.graph_id || !current.depends_on.some(edge => edge.kind === "work" && edge.work_id === parent.work_id)) {
      throw new Error("work_queue_continuation_lineage_invalid");
    }
    descendants.push(continuation);
    current = parent;
  }
  for (const [index, continuation] of descendants.reverse().entries()) {
    if (continuation.root_work_id !== current.work_id || continuation.hop !== index + 1) throw new Error("work_queue_continuation_lineage_invalid");
  }
  return { root_work_id: current.work_id, hop: descendants.length };
}

function validateHandoff(handoff) {
  if (typeof handoff !== "string" || !handoff.trim() || Buffer.byteLength(handoff, "utf8") > MAX_HANDOFF_BYTES) throw new TypeError("continuation handoff must be nonempty and at most 16 KiB");
}

function prepareWorkQueueContinuation(snapshot, args, maxHops = MAX_CONTINUATION_HOPS) {
  closed(args, ["handoff"], ["claim_handle"], "work_queue_continue");
  validateHandoff(args.handoff);
  integer(maxHops, 1, 10, "maximum continuation hops");
  if (!snapshot.worker) throw new Error("work_queue_assignment_required");
  const scoped = normalizeClaimScope(args, snapshot.worker);
  const member = snapshot.worker.claims.find(claim => claim.handle === scoped.claim_handle);
  const work = member && snapshot.projection.works.get(member.work_id);
  if (!work || canonical(work.payload) !== canonical(member.work)) throw new Error("work_queue_continuation_assignment_invalid");
  const lineage = continuationLineage(snapshot.projection, work);
  if (lineage.hop >= maxHops) throw new Error("work_queue_continuation_hop_limit");
  const continuation = { ...lineage, previous_work_id: work.work_id, hop: lineage.hop + 1, handoff: args.handoff };
  const node = {
    graph_id: work.graph_id,
    node_key: `continue:${createHash("sha256").update(work.work_id).digest("hex")}`,
    worker_profile: work.worker_profile,
    ...(work.logical_contract === undefined ? {} : { logical_contract: work.logical_contract }),
    ...(work.execution_ref === undefined ? {} : { execution_ref: work.execution_ref }),
    ...(work.subject === undefined ? {} : { subject: work.subject }),
    payload: { ...work.payload, work_queue_continuation: continuation },
    depends_on: [{ kind: "work", work_id: work.work_id }],
  };
  return { claim_handle: scoped.claim_handle, nodes: [node] };
}

function createWorkQueueContinuationTool(snapshot, options = {}) {
  return {
    name: "work_queue_continue",
    description:
      "Experimental: stage one dependent child Work for an original Claim, preserving its task, effect contract, resource scope and worker profile. Supply a bounded handoff; publish checkpoint effects and finish the current Claim separately. The child waits for the predecessor's independently verified Result. This stages a work_queue_submit control, not a workflow dispatch.",
    inputSchema: {
      type: "object",
      properties: { claim_handle: { type: "string", minLength: 1, maxLength: 256 }, handoff: { type: "string", minLength: 1, maxLength: MAX_HANDOFF_BYTES } },
      required: ["handoff"],
      additionalProperties: false,
    },
    handler: args => {
      const parameters = prepareWorkQueueContinuation(snapshot, args, options.maxContinuationHops);
      const { createWorkQueueSubmitTool, DEFAULT_INTENT_PATH } = require("./work_queue_mcp_server.cjs");
      const intentId = `continuation:${createHash("sha256")
        .update(canonical([snapshot.worker.dispatch_id, parameters.claim_handle]))
        .digest("hex")}`;
      const intentPath = options.intentPath || process.env.GH_AW_WORK_QUEUE_INTENTS || DEFAULT_INTENT_PATH;
      const previous = readStagedIntents(intentPath).find(intent => intent.intent_id === intentId);
      if (previous) {
        const { claim_handle, ...body } = parameters;
        if (previous.kind !== "submit" || previous.claim_handle !== claim_handle || canonical(previous.parameters) !== canonical(body)) throw new Error("work_queue_continuation_conflict");
        return { content: [{ type: "text", text: JSON.stringify({ intent_id: intentId, status: "staged", claim_handle }) }] };
      }
      return createWorkQueueSubmitTool(snapshot, { ...options, createIntentId: () => intentId }).handler(parameters);
    },
  };
}

module.exports = { MAX_CONTINUATION_HOPS, MAX_HANDOFF_BYTES, prepareWorkQueueContinuation, createWorkQueueContinuationTool };
