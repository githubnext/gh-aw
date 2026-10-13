// @ts-check
/// <reference types="@actions/github-script" />
"use strict";

/**
 * Agent-job step for continue_workflow runs.
 *
 * When this run was dispatched as a continuation hop (aw_context.continuation), check out the
 * chain's stable work branch at exactly the head recorded in the continuation ledger and
 * write a read-only context file for the agent and the safe-outputs MCP server. Any mismatch
 * between the dispatch and the ledger fails the run: a continuation never silently restarts
 * from scratch or from an unexpected branch state.
 */

const fs = require("fs");
const path = require("path");
const { getErrorMessage } = require("./error_helpers.cjs");
const { ERR_VALIDATION } = require("./error_codes.cjs");
const { buildGitAuthEnv } = require("./git_auth_env.cjs");
const { parseContinuationRequest, readChain } = require("./continue_workflow_ledger.cjs");

const CONTEXT_DIR = "/tmp/gh-aw/continue-workflow";
const CONTEXT_FILE = path.join(CONTEXT_DIR, "context.json");

/** @param {string} branch */
function isSafeBranchName(branch) {
  return typeof branch === "string" && /^[A-Za-z0-9._/-]+$/.test(branch) && !branch.includes("..") && !branch.startsWith("-") && !branch.endsWith("/");
}

async function isShallowRepository() {
  const result = await exec.getExecOutput("git", ["rev-parse", "--is-shallow-repository"], { ignoreReturnCode: true, silent: true });
  return result.exitCode === 0 && result.stdout.trim() === "true";
}

async function main() {
  let request;
  try {
    request = parseContinuationRequest(context.payload);
  } catch (error) {
    core.setFailed(getErrorMessage(error));
    return;
  }
  if (!request) {
    core.info("Not a continue_workflow hop; skipping continuation checkout");
    return;
  }

  try {
    const chain = await readChain(github, { owner: context.repo.owner, repo: context.repo.repo }, request.chain_id);
    if (!chain) throw new Error(`${ERR_VALIDATION}: continuation chain ${request.chain_id} does not exist`);
    if (chain.status !== "active") throw new Error(`${ERR_VALIDATION}: continuation chain ${request.chain_id} is ${chain.status}`);
    const next = chain.next;
    // A re-run of this same run may find the hop already bound to it by its safe-outputs job.
    const boundToThisRun = next?.state === "running" && next.run_id === String(context.runId);
    if (!next || next.hop !== request.hop || next.dispatch_id !== request.dispatch_id || (next.state !== "dispatched" && !boundToThisRun)) {
      throw new Error(`${ERR_VALIDATION}: this run is not the dispatched continuation hop ${request.hop} of chain ${request.chain_id}`);
    }
    if (!isSafeBranchName(chain.branch)) throw new Error(`${ERR_VALIDATION}: continuation chain ${request.chain_id} has an invalid work branch`);

    const branch = chain.branch;
    const baseSha = next.base_sha;
    const depthArgs = (await isShallowRepository()) ? ["--depth=50"] : [];
    const checkoutToken = core.getInput("github-token");
    if (!checkoutToken) throw new Error(`${ERR_VALIDATION}: continuation checkout requires the root checkout token`);
    const gitAuthEnv = buildGitAuthEnv(checkoutToken, core.setSecret);
    const gitOptions = { env: { ...process.env, ...gitAuthEnv }, silent: true };
    core.info(`Fetching continuation branch ${branch}`);
    await exec.exec("git", ["fetch", "--no-tags", ...depthArgs, "origin", `+refs/heads/${branch}:refs/remotes/origin/${branch}`], gitOptions);
    const { stdout } = await exec.getExecOutput("git", ["rev-parse", `refs/remotes/origin/${branch}^{commit}`], { silent: true });
    const fetched = stdout.trim();
    if (fetched !== baseSha) {
      throw new Error(`${ERR_VALIDATION}: continuation branch ${branch} is at ${fetched}, expected ${baseSha}; refusing to resume from an unexpected state`);
    }
    await exec.exec("git", ["checkout", "-B", branch, baseSha], gitOptions);

    const continuationContext = {
      chain_id: chain.chain_id,
      hop: request.hop,
      max_hops: chain.max_hops,
      remaining_hops: Math.max(0, chain.max_hops - request.hop),
      branch,
      base_sha: baseSha,
      base_branch: chain.base_branch,
      plan_file: "plan.md",
      history: chain.hops.map(/** @param {any} hop */ hop => ({ hop: hop.hop, outcome: hop.outcome, reason: hop.reason, next_steps: hop.next_steps })),
    };
    fs.mkdirSync(CONTEXT_DIR, { recursive: true });
    fs.writeFileSync(CONTEXT_FILE, `${JSON.stringify(continuationContext, null, 2)}\n`, "utf8");
    core.setOutput("branch", branch);
    core.setOutput("hop", String(request.hop));
    core.info(`Resumed continuation hop ${request.hop}/${chain.max_hops} on ${branch} at ${baseSha}`);
  } catch (error) {
    core.setFailed(`Continuation checkout failed: ${getErrorMessage(error)}`);
  }
}

module.exports = { main, CONTEXT_FILE };
