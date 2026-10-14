// @ts-check
import { describe, expect, it } from "vitest";
import fs from "node:fs";
import { performance } from "node:perf_hooks";
import { canonical } from "./work_queue_codec.cjs";
import { prepareCheckpoint, replayTransactionLog } from "./work_queue_replay.cjs";
import { readWorkQueueLog } from "./work_queue_store.cjs";
import { fakeGitHub } from "./work_queue_store_checks.cjs";
import { queueFixture } from "./work_queue_lifecycle.test_helpers.cjs";

const MAX_REPLAY_MS = 250;

function representativeLedger() {
  const queue = queueFixture({
    count: 15,
    batch: 1,
    workDefaults: {
      payload: {
        plan: "Analyze ESLint diagnostics within the original Claim and publish verified results. ".repeat(80),
        effect_contract: { kind: "none" },
      },
    },
  });
  for (let index = 0; index < 100; index++) {
    queue.append("control", { operations: [{ kind: "Control", control: "admission_paused", value: index % 2 === 0, reason: "performance_fixture" }] }, queue.administrator);
  }
  return { queue, contents: canonical(prepareCheckpoint(queue.transactions, "c".repeat(40), queue.administrator, queue.at)[0]) + "\n" };
}

function replayBudget(contents) {
  const samples = [];
  let expected;
  for (let iteration = 0; iteration < 5; iteration++) {
    const start = performance.now();
    const state = replayTransactionLog(contents);
    samples.push(performance.now() - start);
    const shape = { works: state.works.size, claims: state.claims.size, requests: state.requests.size };
    if (expected) expect(shape).toEqual(expected);
    else expected = shape;
    state.works.clear();
    state.requests.clear();
  }
  const median = samples.sort((left, right) => left - right)[2];
  console.info(JSON.stringify({ benchmark: "work-queue-replay", bytes: Buffer.byteLength(contents), ...expected, median_ms: Math.round(median) }));
  expect(median).toBeLessThan(MAX_REPLAY_MS);
}

describe("work queue live-sized performance budgets", () => {
  it("replays a checkpoint with the live ledger's Work, Claim and retained-request scale within budget", () => {
    const { contents } = representativeLedger();
    const state = replayTransactionLog(contents);
    expect(state.works.size).toBe(15);
    expect(state.claims.size).toBe(15);
    expect(state.requests.size).toBeGreaterThanOrEqual(100);
    expect(Buffer.byteLength(contents)).toBeGreaterThanOrEqual(200 * 1024);
    replayBudget(contents);
  });

  it.runIf(Boolean(process.env.GH_AW_WORK_QUEUE_PERF_LEDGER))("replays the captured live ledger within the same detached-state budget", () => {
    const filename = process.env.GH_AW_WORK_QUEUE_PERF_LEDGER;
    if (!filename) throw new Error("GH_AW_WORK_QUEUE_PERF_LEDGER must name a locally captured ledger");
    replayBudget(fs.readFileSync(filename, "utf8"));
  });

  it("bounds warm ledger reads to fresh repository/ref probes despite a long checkpoint ancestry", async () => {
    const { queue, contents } = representativeLedger();
    const fake = fakeGitHub(queue.transactions);
    const git = fake.githubClient.rest.git;
    const history = fake.refs.get("work-queue");
    const prior = "c".repeat(40);
    const getCommit = git.getCommit;
    git.getCommit = parameters => getCommit({ ...parameters, commit_sha: parameters.commit_sha === prior ? history : parameters.commit_sha });
    fake.refs.set("work-queue", prior);
    const blob = await git.createBlob({ content: contents, encoding: "utf-8" });
    const tree = await git.createTree({ tree: [{ path: "work-queue.jsonl", mode: "100644", type: "blob", sha: blob.data.sha }] });
    for (let index = 0; index < 32; index++) {
      const commit = await git.createCommit({ tree: tree.data.sha, parents: [fake.refs.get("work-queue")] });
      fake.refs.set("work-queue", commit.data.sha);
    }
    const calls = [];
    for (const [name, client] of Object.entries(fake.githubClient.rest)) {
      for (const [method, implementation] of Object.entries(client)) {
        client[method] = parameters => {
          calls.push(`${name}.${method}`);
          return implementation(parameters);
        };
      }
    }
    const options = { githubClient: fake.githubClient, owner: "owner", repo: "repo" };
    const cold = await readWorkQueueLog(options);
    expect(calls.length).toBeGreaterThan(100);
    const coldCalls = calls.length;
    cold.state.works.clear();
    cold.transactions.length = 0;
    for (let iteration = 0; iteration < 5; iteration++) {
      calls.length = 0;
      const warm = await readWorkQueueLog(options);
      expect(calls).toEqual(["repos.get", "git.getRef"]);
      expect(warm.state.works.size).toBe(15);
      expect(warm.state.claims.size).toBe(15);
      expect(warm.transactions).toHaveLength(1);
    }
    console.info(JSON.stringify({ benchmark: "work-queue-read", ancestry_commits: 32, cold_api_calls: coldCalls, warm_api_calls: calls.length }));
  }, 15000);
});
