// @ts-check
import { afterEach, describe, expect, it } from "vitest";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { canonical } from "./work_queue_codec.cjs";
import { readStagedIntents } from "./work_queue_intents.cjs";
import { createWorkQueueFinishTool, createWorkQueueTools } from "./work_queue_mcp_server.cjs";
import { createWorkQueueContinuationTool } from "./work_queue_continuation.cjs";
import { authorizeWorkerClaim, finalizeWorkerResults, readFinishIntent, reconcileWorkerClaim } from "./finish_work_queue_claim.cjs";
import { queueFixture, noWriteClaimVerifier, REF, REPOSITORY, WORKFLOW } from "./work_queue_lifecycle.test_helpers.cjs";
import { planNext } from "./work_queue_replay.cjs";
import { dependencyStatus } from "./work_queue_graph.cjs";
import { workIssueStatus } from "./work_queue_issues.cjs";

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

describe("native keep-open Work continuation", () => {
  it("retains the same Work and open Issue status, gates a new Claim on verified delivery, and blocks successors", async () => {
    const test = setup();
    const { fixture } = test;
    const member = fixture.assignment.claims[0];
    const options = {
      assignment: fixture.assignment,
      githubClient: fixture.githubClient,
      context: fixture.workerContext,
      workflowRef: `${REPOSITORY}/${WORKFLOW}@${REF}`,
      finishIntentPath: test.finishIntentPath,
      readWorkQueueLog: fixture.readWorkQueueLog,
      publishWorkQueueRequest: fixture.publishWorkQueueRequest,
      now: fixture.at,
      sleepFn: async () => {},
    };
    const original = fixture.state.works.get(member.work_id);
    const successor = { ...original, work_id: "successor", depends_on: [{ kind: "work", work_id: member.work_id }] };
    createWorkQueueContinuationTool(test.snapshot, test).handler({});
    expect((await reconcileWorkerClaim(options)).claims.h1).toMatchObject({ state: "completed", authorized: true });
    expect(fixture.state.works.size).toBe(1);
    expect(fixture.state.works.get(member.work_id)).toMatchObject({ state: "available", barrier: "pending" });
    expect(planNext(fixture.state, "default", fixture.at).work_id).toBeUndefined();
    expect(workIssueStatus(fixture.state, fixture.state.works.get(member.work_id), fixture.at)).toBe("Verifying");
    expect((await finalizeWorkerResults({ ...options, verifyEffects: noWriteClaimVerifier(options) })).claims.h1.state).toBe("result");
    expect(fixture.state.works.get(member.work_id)).toMatchObject({ state: "available", barrier: "none" });
    expect(workIssueStatus(fixture.state, fixture.state.works.get(member.work_id), fixture.at)).toBe("Queued");
    expect(dependencyStatus(fixture.state, successor, fixture.at).reason).toBe("dependency_result_unavailable");
    fixture.append("dispatch_next", { pool: "default", max_claims: 1, max_dispatches: 1, max_bytes: 49152 }, fixture.dispatcher, "continuation-regrant");
    const retained = fixture.state.works.get(member.work_id);
    expect(retained.claim_id).not.toBe(member.claim_id);
    expect(retained.payload).toEqual(original.payload);
    expect(fixture.state.works.size).toBe(1);
    expect(await authorizeWorkerClaim({ ...options, claim_handle: member.handle })).toMatchObject({ authorized: false, suppressed: true, state: "result" });
  });

  it("stages the existing keep-open finish without submitting child Work or mutating the assignment", () => {
    const test = setup();
    const before = canonical(test.snapshot.worker);
    const tool = createWorkQueueContinuationTool(test.snapshot, test);
    const first = tool.handler({});
    expect(tool.handler({})).toEqual(first);
    const intents = readStagedIntents(test.finishIntentPath);
    expect(intents).toHaveLength(1);
    expect(intents[0]).toMatchObject({ version: 3, kind: "finish", claim_handle: "h1", parameters: { outcome: "completed", keep_open: true } });
    expect(fs.existsSync(test.intentPath)).toBe(false);
    expect(canonical(test.snapshot.worker)).toBe(before);
    const finish = readFinishIntent(test.finishIntentPath, test.snapshot.worker);
    expect(finish.errors).toEqual([]);
    expect(finish.intents.get("h1")).toMatchObject({ outcome: "completed", keep_open: true });
  });

  it("reuses an identical direct native finish and rejects conflicting terminal or cancelled finishes", () => {
    for (const outcome of ["completed", "cancelled"]) {
      const test = setup();
      const finish = createWorkQueueFinishTool({ ...test, snapshot: test.snapshot });
      finish.handler({ outcome });
      const before = readStagedIntents(test.finishIntentPath);
      expect(() => createWorkQueueContinuationTool(test.snapshot, test).handler({})).toThrow(/finish_conflict/);
      expect(readStagedIntents(test.finishIntentPath)).toEqual(before);
    }
    const test = setup();
    const finish = createWorkQueueFinishTool({ ...test, snapshot: test.snapshot });
    const first = finish.handler({ outcome: "completed", keep_open: true });
    expect(createWorkQueueContinuationTool(test.snapshot, test).handler({})).toEqual(first);
    expect(readStagedIntents(test.finishIntentPath)).toHaveLength(1);
    expect(() => finish.handler({ outcome: "completed" })).toThrow(/finish_conflict/);
  });

  it("requires an original multi-Claim selector and preserves independent sibling outcomes", () => {
    const test = setup({ count: 2 });
    const tool = createWorkQueueContinuationTool(test.snapshot, test);
    expect(() => tool.handler({})).toThrow(/claim_handle is required/);
    expect(() => tool.handler({ claim_handle: "foreign" })).toThrow(/foreign/);
    tool.handler({ claim_handle: "h2" });
    createWorkQueueFinishTool({ ...test, snapshot: test.snapshot }).handler({ claim_handle: "h1", outcome: "completed" });
    const finish = readFinishIntent(test.finishIntentPath, test.snapshot.worker);
    expect(finish.errors).toEqual([]);
    expect(finish.intents.get("h2")).toMatchObject({ outcome: "completed", keep_open: true });
    expect(finish.intents.get("h1")).toMatchObject({ outcome: "completed", keep_open: false });
  });

  it("rejects missing assignments and obsolete handoff or scheduling overrides before writing", () => {
    const test = setup();
    expect(() => createWorkQueueContinuationTool({ ...test.snapshot, worker: null }, test).handler({})).toThrow(/assignment_required/);
    for (const field of ["handoff", "pool", "priority", "fairness_key", "worker_profile", "max_hops", "outcome", "keep_open"]) {
      expect(() => createWorkQueueContinuationTool(test.snapshot, test).handler({ [field]: "forged" })).toThrow();
    }
    expect(fs.existsSync(test.finishIntentPath)).toBe(false);
    expect(fs.existsSync(test.intentPath)).toBe(false);
  });

  it("registers the helper only for authenticated worker-role snapshots", () => {
    const test = setup();
    expect(createWorkQueueTools(test.snapshot).map(tool => tool.name)).toContain("work_queue_continue");
    for (const role of ["dispatcher", "observer"]) {
      expect(createWorkQueueTools({ ...test.snapshot, worker: null, role }).map(tool => tool.name)).not.toContain("work_queue_continue");
    }
  });
});
