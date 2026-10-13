// @ts-check
import { afterEach, describe, expect, it } from "vitest";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { canonical } from "./work_queue_codec.cjs";
import { readStagedIntents } from "./work_queue_intents.cjs";
import { createWorkQueueTools } from "./work_queue_mcp_server.cjs";
import { createWorkQueueContinuationTool, MAX_HANDOFF_BYTES, prepareWorkQueueContinuation } from "./work_queue_continuation.cjs";
import { queueFixture } from "./work_queue_lifecycle.test_helpers.cjs";
import { nodeId } from "./work_queue_graph.cjs";

const directories = [];
function setup(options = {}) {
  const fixture = queueFixture({ started: true, bound: true, count: 1, ...options });
  const snapshot = { worker: fixture.assignment, projection: fixture.state, role: "worker" };
  const directory = fs.mkdtempSync(path.join(fs.realpathSync(os.tmpdir()), "gh-aw-queue-continuation-"));
  directories.push(directory);
  return { fixture, snapshot, intentPath: path.join(directory, "intents.jsonl"), finishIntentPath: path.join(directory, "finish.jsonl") };
}
afterEach(() => {
  for (const directory of directories.splice(0)) fs.rmSync(directory, { recursive: true, force: true });
});

describe("native dependent Work continuation", () => {
  it("preserves the task, effect contract, resource scope, target and worker profile without selecting scheduling entitlement", () => {
    const payload = {
      task: "Finish the feature",
      effect_contract: { version: 1, outputs: [{ type: "work_queue_submit", min: 1, max: 1 }] },
      resource_scope: { version: 1, resources: [{ repository: "owner/repo", host: "github.com", repository_id: "7" }] },
      pull_request: { number: 42, head: "existing-pr" },
    };
    const test = setup({ workDefaults: { payload } });
    const before = canonical(test.snapshot.worker);
    const parameters = prepareWorkQueueContinuation(test.snapshot, { handoff: "Resume from the verified checkpoint." });
    const parent = test.snapshot.worker.claims[0];
    expect(parameters.claim_handle).toBe(parent.handle);
    const node = parameters.nodes[0];
    expect(node).toMatchObject({
      graph_id: "g1",
      worker_profile: "default",
      payload: { ...payload, work_queue_continuation: { root_work_id: parent.work_id, previous_work_id: parent.work_id, hop: 1, handoff: "Resume from the verified checkpoint." } },
      depends_on: [{ kind: "work", work_id: parent.work_id }],
    });
    for (const field of ["priority", "fairness_key", "pool", "work_id"]) expect(node).not.toHaveProperty(field);
    expect(canonical(test.snapshot.worker)).toBe(before);
  });

  it("stages only a native submit control, reuses exact requests and rejects conflicting handoffs", () => {
    const test = setup();
    const tool = createWorkQueueContinuationTool(test.snapshot, test);
    const first = tool.handler({ handoff: "Resume the task." });
    expect(tool.handler({ handoff: "Resume the task." })).toEqual(first);
    const intents = readStagedIntents(test.intentPath);
    expect(intents).toHaveLength(1);
    expect(intents[0]).toMatchObject({ version: 3, kind: "submit", claim_handle: "h1" });
    expect(fs.existsSync(test.finishIntentPath)).toBe(false);
    expect(() => tool.handler({ handoff: "Different checkpoint." })).toThrow(/continuation_conflict/);
    expect(readStagedIntents(test.intentPath)).toEqual(intents);
  });

  it("requires an original multi-Claim selector and rejects foreign handles or scheduling overrides", () => {
    const test = setup({ count: 2 });
    expect(() => prepareWorkQueueContinuation(test.snapshot, { handoff: "Resume." })).toThrow(/claim_handle is required/);
    expect(() => prepareWorkQueueContinuation(test.snapshot, { handoff: "Resume.", claim_handle: "foreign" })).toThrow(/foreign/);
    expect(prepareWorkQueueContinuation(test.snapshot, { handoff: "Resume.", claim_handle: "h2" }).nodes[0].depends_on).toEqual([{ kind: "work", work_id: test.snapshot.worker.claims[1].work_id }]);
    for (const field of ["pool", "priority", "fairness_key", "worker_profile", "max_hops", "root_work_id"]) {
      expect(() => prepareWorkQueueContinuation(test.snapshot, { handoff: "Resume.", claim_handle: "h1", [field]: "forged" })).toThrow();
    }
  });

  it("rejects missing assignments, altered payloads and forged continuation lineage", () => {
    const test = setup();
    expect(() => prepareWorkQueueContinuation({ ...test.snapshot, worker: null }, { handoff: "Resume." })).toThrow(/assignment_required/);
    const member = test.snapshot.worker.claims[0];
    const worker = { ...test.snapshot.worker, claims: [{ ...member, work: { ...member.work, task: "forged" } }] };
    expect(() => prepareWorkQueueContinuation({ ...test.snapshot, worker }, { handoff: "Resume." })).toThrow(/assignment_invalid/);
    const fake = setup({ workDefaults: { payload: { work_queue_continuation: { root_work_id: "root", previous_work_id: "parent", hop: 1, handoff: "forged" } } } });
    expect(() => prepareWorkQueueContinuation(fake.snapshot, { handoff: "Resume." })).toThrow(/lineage_invalid/);
  });

  it("enforces nonempty UTF-8 byte-bounded handoffs and trusted hop bounds", () => {
    const test = setup();
    for (const handoff of ["", " \n", "x".repeat(MAX_HANDOFF_BYTES + 1), "\u00e9".repeat(MAX_HANDOFF_BYTES / 2 + 1), null]) {
      expect(() => prepareWorkQueueContinuation(test.snapshot, { handoff })).toThrow(/handoff/);
    }
    expect(prepareWorkQueueContinuation(test.snapshot, { handoff: "x".repeat(MAX_HANDOFF_BYTES) }).nodes).toHaveLength(1);
    for (const limit of [0, 11, 1.5]) expect(() => prepareWorkQueueContinuation(test.snapshot, { handoff: "Resume." }, limit)).toThrow();
  });

  it("counts the entire projected lineage and refuses a fourth continuation", () => {
    const test = setup();
    const projection = {
      ...test.snapshot.projection,
      works: new Map(test.snapshot.projection.works),
      workCreators: new Map(test.snapshot.projection.workCreators),
      dispatches: new Map(test.snapshot.projection.dispatches),
    };
    let snapshot = { ...test.snapshot, projection };
    const root = snapshot.worker.claims[0].work_id;
    for (let hop = 1; hop <= 3; hop++) {
      const parameters = prepareWorkQueueContinuation(snapshot, { handoff: `Handoff ${hop}` });
      const node = parameters.nodes[0];
      expect(node.payload.work_queue_continuation).toMatchObject({ root_work_id: root, hop });
      const parent = projection.works.get(snapshot.worker.claims[0].work_id);
      const work_id = nodeId(node.graph_id, node.node_key);
      projection.works.set(work_id, { ...parent, ...node, work_id });
      projection.workCreators.set(work_id, { role: "worker", dispatch_id: snapshot.worker.dispatch_id, claim_handle: "h1" });
      projection.dispatches.set(snapshot.worker.dispatch_id, { claims: snapshot.worker.claims });
      snapshot = {
        ...snapshot,
        worker: {
          ...snapshot.worker,
          dispatch_id: `continuation-dispatch-${hop}`,
          claims: [{ ...snapshot.worker.claims[0], work_id, work: node.payload }],
        },
      };
    }
    expect(() => prepareWorkQueueContinuation(snapshot, { handoff: "Fourth hop" })).toThrow(/continuation_hop_limit/);
    const member = snapshot.worker.claims[0];
    const work = projection.works.get(member.work_id);
    work.payload = { ...work.payload, work_queue_continuation: { ...work.payload.work_queue_continuation, hop: 1 } };
    const forged = { ...snapshot, worker: { ...snapshot.worker, claims: [{ ...member, work: work.payload }] } };
    expect(() => prepareWorkQueueContinuation(forged, { handoff: "Reset budget" })).toThrow(/lineage_invalid/);
  });

  it("registers the helper only for authenticated worker-role snapshots", () => {
    const test = setup();
    expect(createWorkQueueTools(test.snapshot).map(tool => tool.name)).toContain("work_queue_continue");
    for (const role of ["dispatcher", "observer"]) {
      const snapshot = { ...test.snapshot, worker: null, role };
      expect(createWorkQueueTools(snapshot).map(tool => tool.name)).not.toContain("work_queue_continue");
    }
  });
});
