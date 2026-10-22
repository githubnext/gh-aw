import { afterEach, beforeEach, describe, expect, it } from "vitest";
import path from "node:path";
import { fixture, mock } from "./work_queue_issues_checks.cjs";
import { main, workIssueStatus } from "./work_queue_issues.cjs";
import { transitionProjectionTargets } from "./work_queue_issue_transitions.cjs";
import { compactTransactions, newRequest } from "./work_queue_replay.cjs";
import { operationCommit, context } from "./work_queue_test_helpers.cjs";

const maintenance = context({ role: "reconciler", principal: "1001", repository: "owner/repo", workflow: ".github/workflows/agentics-maintenance.yml", run_id: "300", run_attempt: 1 }, { event: "schedule", ref: "0".repeat(40) });

async function setup() {
  const f = fixture({ count: 2, backing: false });
  const policy = f.log[0].operations[0].policy;
  policy.authorization = "aw";
  policy.producers = {};
  delete policy.projectors;
  const initial = f.log[0];
  initial.request = newRequest(initial.request.id, "policy", initial.actor, { operations: initial.operations });
  const m = mock(f);
  expect(await main(m.options)).toMatchObject({ pending: [] });
  const log = [...m.state().transactions];
  const workId = f.nodes[0].work_id;
  const actor = { role: "reconciler", principal: "1001", repository: "owner/repo", workflow: maintenance.workflow, run_id: maintenance.run_id, run_attempt: 1 };
  const receipt = operationCommit(log, "terminal-cancellation", "cancel_work", [{ kind: "WorkCancellation", work_id: workId, reason: "attempts_exhausted" }], actor, 200);
  log.push(receipt);
  m.setLog(log);
  return { f, m, log, workId, requestId: receipt.request.id };
}

describe("exact committed terminal Issue projection", () => {
  beforeEach(() => {
    process.env.GH_AW_PROMPTS_DIR = path.resolve(import.meta.dirname, "../md");
  });
  afterEach(() => {
    delete process.env.GH_AW_PROMPTS_DIR;
  });

  it("updates only the original maintenance transition's existing mirror and summary without creating or closing anything", async () => {
    const { m, workId, requestId } = await setup();
    const before = m.state();
    const calls = m.calls.length;
    const result = await main({ ...m.options, trustedContext: maintenance, transitionRequestIds: [requestId], now: 250 });
    expect(result.pending).toEqual([]);
    const work = m.state().works.get(workId);
    const issue = m.issues.get(work.issue_link.number);
    expect(issue.state).toBe("OPEN");
    expect(issue.labels.nodes.map(label => label.name)).toEqual(expect.arrayContaining(["work", "work:cancelled"]));
    expect(issue.labels.nodes.some(label => label.name === "work:queued")).toBe(false);
    expect(m.comments.get(work.issue_summary).body).toContain("Status: **Cancelled**");
    expect(m.comments.get(work.issue_summary).body).toContain("attempts_exhausted");
    expect(m.issues.size).toBe(2);
    expect(m.comments.size).toBe(2);
    expect(m.state().tip).toBe(before.tip);
    const native = m.calls.slice(calls).filter(call => call[0] === "mutation");
    expect(native.some(call => /createIssue|addComment|closeIssue/.test(call[1]))).toBe(false);
    expect(m.locks.size).toBe(0);
  });

  it("preserves exact transition authority through compaction without new ledger fields or operations", async () => {
    const { m, log, requestId } = await setup();
    const expected = transitionProjectionTargets(m.state(), maintenance, [requestId]);
    m.setLog(compactTransactions(log, "a".repeat(40), { role: "administrator", principal: "1001", repository: "owner/repo" }, 250));
    expect(transitionProjectionTargets(m.state(), maintenance, [requestId])).toEqual(expected);
    expect(await main({ ...m.options, trustedContext: maintenance, transitionRequestIds: [requestId] })).toMatchObject({ pending: [] });
  });

  it.each([{ run_id: "301" }, { run_attempt: 2 }, { principal: "2002" }, { workflow: ".github/workflows/other.yml" }, { role: "projector" }, { event: "issues" }, { repository: "foreign/repo" }, { authenticated: false }])(
    "rejects changed authority %j before native writes",
    async change => {
      const { m, requestId } = await setup();
      const count = m.calls.length;
      const result = await main({ ...m.options, trustedContext: { ...maintenance, ...change }, transitionRequestIds: [requestId] });
      expect(result.pending.length).toBeGreaterThan(0);
      expect(m.calls.slice(count).some(call => call[0] === "mutation" || call[0] === "lock")).toBe(false);
    }
  );

  it("never recreates a deleted canonical summary or bypasses a retained coordination fence", async () => {
    const { m, workId, requestId } = await setup();
    const work = m.state().works.get(workId);
    m.comments.delete(work.issue_summary);
    const result = await main({ ...m.options, trustedContext: maintenance, transitionRequestIds: [requestId] });
    expect(result.pending.length).toBeGreaterThan(0);
    expect(m.comments.size).toBe(1);
    expect(m.issues.size).toBe(2);
    expect(m.warnings.join("\n")).toMatch(/deleted|transferred|canonical/);
  });

  it("does not broaden original admission/Claim authority to the maintenance publisher", async () => {
    const { m } = await setup();
    const count = m.calls.length;
    expect(await main({ ...m.options, trustedContext: maintenance })).toMatchObject({ pending: [] });
    expect(m.calls.slice(count).some(call => call[0] === "mutation" || call[0] === "lock")).toBe(false);
  });

  it("keeps missing immutable receipts pending instead of creating another mirror", async () => {
    const { m, workId, requestId } = await setup();
    const state = m.state();
    delete state.works.get(workId).issue_summary;
    expect(() => transitionProjectionTargets(state, maintenance, [requestId])).toThrow(/existing Issue and canonical summary receipts/);
    expect(() => transitionProjectionTargets(m.state(), maintenance, Array(17).fill(requestId))).toThrow(/16 distinct/);
  });

  it("maps completed Work with failed delivery to Needs attention rather than Done", () => {
    expect(workIssueStatus({}, { state: "completed", barrier: "failed" }, 100)).toBe("Needs attention");
  });

  it.each(["WorkQueueTrackingLabel", "WorkQueueLabelColor"])("retains shared fences after an unknown %s write without retrying", async operation => {
    const { m, requestId } = await setup();
    if (operation === "WorkQueueLabelColor") m.discovery.label.color = "000000";
    const original = m.options.githubClient.graphql;
    let writes = 0;
    m.options.githubClient.graphql = async (query, variables) => {
      const response = await original(query, variables);
      if (query.includes(operation)) {
        writes++;
        throw new Error("secret-sentinel lost acknowledgement");
      }
      return response;
    };
    const result = await main({ ...m.options, trustedContext: maintenance, transitionRequestIds: [requestId] });
    expect(result.pending.length).toBeGreaterThan(0);
    expect(writes).toBe(1);
    expect(m.locks.size).toBe(2);
    expect(m.warnings.join("\n")).not.toContain("secret-sentinel");
    expect(m.issues.size).toBe(2);
  });

  it("retains shared fences for unverified label-edit acknowledgements", async () => {
    const { m, requestId } = await setup();
    const original = m.options.githubClient.graphql;
    m.options.githubClient.graphql = async (query, variables) => {
      const response = await original(query, variables);
      if (query.includes("WorkQueueIssueProjection")) {
        for (const value of Object.values(response)) if (value.labelable) value.labelable.labels.nodes = [];
      }
      return response;
    };
    const result = await main({ ...m.options, trustedContext: maintenance, transitionRequestIds: [requestId] });
    expect(result.pending.length).toBeGreaterThan(0);
    expect(m.locks.size).toBe(2);
  });
});
