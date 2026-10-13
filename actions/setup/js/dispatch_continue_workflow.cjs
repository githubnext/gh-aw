// @ts-check
/// <reference types="@actions/github-script" />
// @safe-outputs-exempt SEC-005: continuations are dispatched only to workflows in the current GitHub Actions repository.
"use strict";

/**
 * Agentic Maintenance step that dispatches queued continue_workflow hops.
 *
 * For every active chain whose next hop is queued, the hop is first moved to `dispatched`
 * with a fresh dispatch id (compare-and-swap), and only then is the workflow dispatched.
 * The dispatched run must present that dispatch id, so concurrent or repeated maintenance
 * runs can never start two runs for the same hop. Failed dispatches are re-queued a bounded
 * number of times; hops that never settle are failed so the chain is reported, not lost.
 */

const crypto = require("crypto");
const { getErrorMessage } = require("./error_helpers.cjs");
const { ERR_API } = require("./error_codes.cjs");
const ledger = require("./continue_workflow_ledger.cjs");

const MAX_CHAINS_PER_RUN = 50;

/** @param {any} chain */
function buildAwContext(chain, dispatchId) {
  return JSON.stringify({
    repo: chain.repository,
    actor: chain.initiator,
    origin_event: "continue_workflow",
    episode_id: chain.chain_id,
    root_run_id: chain.hops.find(/** @param {any} h */ h => h.hop === 0)?.run_id || "",
    continuation: { chain_id: chain.chain_id, hop: chain.next.hop, dispatch_id: dispatchId },
  });
}

/**
 * @param {any} githubClient
 * @param {any} chain
 */
async function resolveDispatchRef(githubClient, chain) {
  if (typeof chain.ref === "string" && /^refs\/(heads|tags)\/[A-Za-z0-9._/-]+$/.test(chain.ref)) return chain.ref;
  const { data } = await githubClient.rest.repos.get({ owner: context.repo.owner, repo: context.repo.repo });
  return `refs/heads/${data.default_branch}`;
}

/**
 * @param {{githubClient?: any, now?: () => string, newDispatchId?: () => string}} [options]
 */
async function main(options = {}) {
  const githubClient = options.githubClient || github;
  const now = options.now || (() => new Date().toISOString());
  const newDispatchId = options.newDispatchId || (() => crypto.randomBytes(12).toString("hex"));
  const repo = { owner: context.repo.owner, repo: context.repo.repo };
  /** @type {{dispatched: string[], requeued: string[], failed: string[], errors: string[]}} */
  const report = { dispatched: [], requeued: [], failed: [], errors: [] };

  const head = await ledger.readLedgerHead(githubClient, repo, ledger.LEDGER_BRANCH);
  const chainIds = head ? (await ledger.listChainIds(githubClient, repo, head)).sort() : [];
  if (!head || chainIds.length === 0) {
    core.info("No continuation chains recorded");
    return report;
  }

  const boundedScan = chainIds.length > MAX_CHAINS_PER_RUN;
  const cursor = boundedScan ? await ledger.readDispatchCursorAt(githubClient, repo, head.sha) : null;
  const ordered = cursor ? [...chainIds.filter(id => id > cursor), ...chainIds.filter(id => id <= cursor)] : chainIds;
  const selected = ordered.slice(0, MAX_CHAINS_PER_RUN);
  for (const chainId of selected) {
    try {
      // Settled chains are skipped from one snapshot read; only active chains are mutated.
      const snapshot = await ledger.readChainAt(githubClient, repo, head.sha, chainId);
      if (!snapshot || snapshot.status !== "active") continue;
      const expired = await ledger.mutateChain(githubClient, repo, chainId, current => ledger.expireStale(current, { now: now() }), "Expire stale continuation hop");
      if (expired.changed) {
        report.failed.push(`${chainId}: ${expired.chain.error}`);
        continue;
      }
      const dispatchId = newDispatchId();
      const claimed = await ledger.mutateChain(githubClient, repo, chainId, current => ledger.claimDispatch(current, { dispatchId, now: now() }), "Dispatch continuation hop");
      if (!claimed.changed) continue;
      const chain = claimed.chain;
      try {
        const ref = await resolveDispatchRef(githubClient, chain);
        await githubClient.rest.actions.createWorkflowDispatch({ ...repo, workflow_id: chain.workflow_file, ref, inputs: { aw_context: buildAwContext(chain, dispatchId) } });
        report.dispatched.push(`${chainId}: hop ${chain.next.hop}/${chain.max_hops} of ${chain.workflow_file} on ${chain.branch}`);
        core.info(`Dispatched continuation hop ${chain.next.hop} of chain ${chainId}`);
      } catch (dispatchError) {
        const errorMessage = getErrorMessage(dispatchError);
        core.warning(`Failed to dispatch continuation hop for chain ${chainId}: ${errorMessage}`);
        const result = await ledger.mutateChain(githubClient, repo, chainId, current => ledger.recordDispatchFailure(current, { dispatchId, error: errorMessage, now: now() }), "Record failed continuation dispatch");
        if (result.chain?.status === "failed") report.failed.push(`${chainId}: ${result.chain.error}`);
        else report.requeued.push(`${chainId}: ${errorMessage}`);
      }
    } catch (error) {
      report.errors.push(`${chainId}: ${getErrorMessage(error)}`);
    }
  }
  core.info(`Scanned ${selected.length} of ${chainIds.length} continuation chains`);

  if (boundedScan) {
    const last = selected[selected.length - 1];
    if (!last) throw new Error(`${ERR_API}: continuation dispatch scan selected no chains`);
    await ledger.advanceDispatchCursor(githubClient, repo, cursor, last);
    core.info(`Inspected ${selected.length} of ${chainIds.length} continuation chains; the next maintenance scan resumes after ${last}`);
  }

  const lines = ["## Continue-workflow dispatch", ""];
  if (selected.length < chainIds.length) lines.push(`Scanned ${selected.length} of ${chainIds.length} chains; the next maintenance run continues at the next bounded window.`, "");
  const section = (/** @type {string} */ title, /** @type {string[]} */ items) => {
    if (items.length === 0) return;
    lines.push(`### ${title}`, "", ...items.map(item => `- ${item}`), "");
  };
  section("Dispatched", report.dispatched);
  section("Re-queued after a failed dispatch", report.requeued);
  section("Failed (work branch preserved)", report.failed);
  section("Errors", report.errors);
  if (lines.length === 2) lines.push("No continuation hops were waiting to be dispatched.");
  await core.summary.addRaw(lines.join("\n")).write();
  for (const failure of report.failed) core.warning(`Continuation chain failed: ${failure}`);
  if (report.errors.length > 0) core.setFailed(`${ERR_API}: ${report.errors.length} continuation chain(s) could not be processed`);
  return report;
}

module.exports = { MAX_CHAINS_PER_RUN, main, buildAwContext };
