"use strict";

const { deflateRawSync, inflateRawSync } = require("node:zlib");
const { canonical, digest } = require("./work_queue_codec.cjs");
const { compactTransactions, newRequest, replayTransactions, serializeProjection } = require("./work_queue_replay.cjs");

function rewriteCheckpoint(checkpoint, change) {
  const result = structuredClone(checkpoint);
  const root = result[0];
  const operation = root.operations[0];
  const { requests_compressed, ...fields } = operation.state;
  const snapshot = { ...fields, requests: JSON.parse(inflateRawSync(Buffer.from(requests_compressed, "base64")).toString("utf8")) };
  change(snapshot, root, operation);
  operation.state_sha256 = digest(snapshot);
  const { requests, ...projection } = snapshot;
  operation.state = { ...projection, requests_compressed: deflateRawSync(Buffer.from(canonical(requests))).toString("base64") };
  root.request = newRequest(root.request.id, "checkpoint", root.actor, {
    prior_git_sha: operation.prior_git_sha,
    prior_tip: operation.prior_tip,
    history_sha256: operation.history_sha256,
    state_sha256: operation.state_sha256,
  });
  return result;
}

function legacyCheckpoint(transactions, priorGitSHA, actor, at) {
  const state = replayTransactions(transactions);
  const checkpoint = compactTransactions(transactions, priorGitSHA, actor, Math.max(at, state.time_watermark));
  // Before expiration, last_at and the minimum envelope time used the final
  // causal commit, not the maximum timestamp in the retained history.
  return rewriteCheckpoint(checkpoint, (snapshot, root) => {
    snapshot.last_at = state.transactions.at(-1).at;
    root.at = at;
  });
}

async function main() {
  const input = JSON.parse(require("node:fs").readFileSync(0, "utf8"));
  if (process.argv.includes("--git-read")) {
    const { createGitLedgerClient, registerQueueLedgerClient } = require("./work_queue_git_transport.cjs");
    const { readWorkQueueLog } = require("./work_queue_store.cjs");
    const githubClient = { auth: async () => ({ type: "token", token: "local-checkpoint-regression" }) };
    const ledger = createGitLedgerClient({ githubClient, owner: "owner", repo: "repo", branch: "work-queue", remote: input.remote });
    registerQueueLedgerClient(githubClient, ledger);
    try {
      const result = await readWorkQueueLog({ githubClient, owner: "owner", repo: "repo" });
      process.stdout.write(canonical({ state: serializeProjection(result.state), time_watermark: result.state.time_watermark, transactions: result.transactions }));
    } catch (error) {
      process.stdout.write(canonical({ code: error.code }));
    } finally {
      ledger.close();
    }
  } else {
    const checkpoint = input.legacy ? legacyCheckpoint(input.transactions, input.prior_git_sha, input.actor, input.at) : compactTransactions(input.transactions, input.prior_git_sha, input.actor, input.at);
    process.stdout.write(canonical({ checkpoint }));
  }
}

if (require.main === module)
  main().catch(error => {
    console.error(error);
    process.exitCode = 1;
  });
module.exports = { legacyCheckpoint, rewriteCheckpoint };
