// @ts-check
/// <reference types="@actions/github-script" />
// @safe-outputs-exempt SEC-005: continuation work branches and the continuation ledger live only in the current GitHub Actions repository.
"use strict";

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

/**
 * continue_workflow: persist the agent's progress to one stable work branch and schedule
 * another run of the same workflow that resumes from that branch.
 *
 * Trust model:
 * - The agent only supplies commits (as a bundle), a reason and optional next steps.
 * - The work branch, the expected branch head, the hop number and the hop budget come from
 *   the continuation ledger, never from the agent or from files the agent can write.
 * - Every branch update is a compare-and-swap against the ledger's expected head, and every
 *   ledger transition is a compare-and-swap against the ledger head, so duplicate delivery
 *   can never create a duplicate hop or overwrite unrelated work.
 * - Exactly one pull request is created, by create_pull_request in the final hop, with the
 *   work branch as its head.
 */

const { getErrorMessage } = require("./error_helpers.cjs");
const { ERR_API, ERR_VALIDATION } = require("./error_codes.cjs");
const { isStagedMode } = require("./safe_output_helpers.cjs");
const { logStagedPreviewInfo } = require("./staged_preview.cjs");
const { createAuthenticatedGitHubClient } = require("./handler_auth.cjs");
const { normalizeBranchName } = require("./normalize_branch_name.cjs");
const { resolveTransportPaths } = require("./resolve_transport_paths.cjs");
const { pushSignedCommits } = require("./push_signed_commits.cjs");
const { checkFileProtection, checkFileProtectionPostApply } = require("./manifest_file_helpers.cjs");
const ledger = require("./continue_workflow_ledger.cjs");

/** @type {string} Safe output type handled by this module */
const HANDLER_TYPE = "continue_workflow";
const MAINTENANCE_WORKFLOW = "agentics-maintenance.yml";
const DISPATCH_OPERATION = "dispatch_continuations";
const DEFAULT_MAX_PATCH_KB = 4096;
const DEFAULT_MAX_PATCH_FILES = 100;

/**
 * Continuation bound to the current run (hop >= 1), resolved from aw_context and the ledger.
 * @type {any}
 */
let activeContinuation = null;
/** @type {string | null} */
let finalHeadSha = null;

function resetContinueWorkflowState() {
  activeContinuation = null;
  finalHeadSha = null;
}

/** @returns {any} */
function getActiveContinuation() {
  return activeContinuation;
}

/** @param {string} sha */
function noteFinalHead(sha) {
  finalHeadSha = sha;
}

function nowIso() {
  return new Date().toISOString();
}

function currentRunId() {
  return String(context.runId || process.env.GITHUB_RUN_ID || "");
}

function repoRef() {
  return { owner: context.repo.owner, repo: context.repo.repo };
}

/** @param {any} value */
function parsePositive(value, fallback) {
  const parsed = typeof value === "number" ? value : typeof value === "string" ? Number.parseInt(value, 10) : NaN;
  return Number.isSafeInteger(parsed) && parsed > 0 ? parsed : fallback;
}

/**
 * Resolves the continuation for this run from aw_context and binds the dispatched hop to it.
 * Must run before any safe output message is processed.
 * @param {any} config - continue_workflow handler configuration
 * @param {any} [githubClient]
 */
async function prepareContinueWorkflow(config, githubClient) {
  activeContinuation = null;
  finalHeadSha = null;
  const request = ledger.parseContinuationRequest(context.payload);
  if (!request) return null;
  if (context.eventName !== "workflow_dispatch") {
    throw new Error(`${ERR_VALIDATION}: continuation context is only accepted on workflow_dispatch runs`);
  }
  const client = githubClient || (await createAuthenticatedGitHubClient(config || {}));
  const runId = currentRunId();
  let chain;
  if (isStagedMode(config || {})) {
    chain = await ledger.readChain(client, repoRef(), request.chain_id);
    if (!chain) throw new Error(`${ERR_VALIDATION}: continuation chain ${request.chain_id} does not exist`);
  } else {
    ({ chain } = await ledger.mutateChain(client, repoRef(), request.chain_id, current => ledger.bindHop(current, { hop: request.hop, dispatchId: request.dispatch_id, runId, now: nowIso() }), "Bind continuation hop"));
  }
  const recorded = chain.hops.find(/** @param {any} h */ h => h.hop === request.hop && h.run_id === runId);
  const running = chain.status === "active" && chain.next?.hop === request.hop;
  activeContinuation = {
    chain_id: chain.chain_id,
    hop: request.hop,
    dispatch_id: request.dispatch_id,
    branch: chain.branch,
    base_branch: chain.base_branch,
    base_sha: running ? chain.next.base_sha : null,
    max_hops: chain.max_hops,
    protected_files: Array.isArray(chain.protected_files) ? chain.protected_files : [],
    protected_action: chain.protected_action || null,
    settled: Boolean(recorded) || !running,
    status: chain.status,
    pull_request: chain.pull_request || null,
  };
  core.info(`Continuation run: chain ${chain.chain_id}, hop ${request.hop}/${chain.max_hops}, branch ${chain.branch}${activeContinuation.settled ? " (already settled)" : ""}`);
  return activeContinuation;
}

/**
 * @param {string[]} args
 * @param {{ignoreReturnCode?: boolean}} [options]
 */
async function git(args, options = {}) {
  return exec.getExecOutput("git", args, { ignoreReturnCode: options.ignoreReturnCode === true, silent: true });
}

/** @param {string} sha */
async function ensureCommitAvailable(sha) {
  const present = await git(["cat-file", "-e", `${sha}^{commit}`], { ignoreReturnCode: true });
  if (present.exitCode === 0) return;
  const shallow = await git(["rev-parse", "--is-shallow-repository"], { ignoreReturnCode: true });
  const depthArgs = shallow.exitCode === 0 && shallow.stdout.trim() === "true" ? ["--depth=1"] : [];
  await exec.exec("git", ["fetch", "--no-tags", ...depthArgs, "origin", sha]);
}

/**
 * @param {any} githubClient
 * @param {string} branch
 * @returns {Promise<string|null>}
 */
async function readRemoteHead(githubClient, branch) {
  try {
    const { data } = await githubClient.rest.git.getRef({ ...repoRef(), ref: `heads/${branch}` });
    return data?.object?.sha || null;
  } catch (error) {
    if (error instanceof Error && "status" in error && error.status === 404) return null;
    throw error;
  }
}

/**
 * @param {any} githubClient
 * @param {string} expectedBase
 * @param {string} headSha
 * @param {string} remoteHead
 * @param {boolean} signedCommits
 */
async function isVerifiedContinuationReplay(githubClient, expectedBase, headSha, remoteHead, signedCommits) {
  if (!signedCommits) return remoteHead === headSha;
  const { stdout } = await git(["rev-list", "--reverse", "--parents", `${expectedBase}..${headSha}`]);
  const commits = stdout
    .trim()
    .split("\n")
    .map(line => line.split(" "));
  let remoteSha = remoteHead;
  for (let index = commits.length - 1; index >= 0; index--) {
    const [localSha, parent, ...otherParents] = commits[index];
    const expectedParent = index === 0 ? expectedBase : commits[index - 1][0];
    if (!ledger.isSha(localSha) || parent !== expectedParent || otherParents.length !== 0) return false;
    const { data } = await githubClient.rest.git.getCommit({ ...repoRef(), commit_sha: remoteSha });
    if (data?.sha !== remoteSha || data.parents?.length !== 1 || !ledger.isSha(data.parents[0].sha)) return false;
    if (data.verification?.verified !== true || data.verification.reason !== "valid") return false;
    const { stdout: tree } = await git(["rev-parse", `${localSha}^{tree}`]);
    if (data.tree?.sha !== tree.trim()) return false;
    const { stdout: message } = await git(["log", "-1", "--format=%B", localSha]);
    const [headline, ...bodyLines] = message.trim().split("\n");
    const body = bodyLines.join("\n").trim();
    const replayedMessage = headline + (body ? `\n\n${body}` : "");
    if (typeof data.message !== "string" || data.message.trim() !== replayedMessage) return false;
    remoteSha = data.parents[0].sha;
  }
  return remoteSha === expectedBase;
}

/**
 * Applies trusted policy to the exact commit range that will be persisted. The range is
 * computed here from the ledger's expected base, not from the agent-side patch, so the agent
 * cannot narrow it.
 * @param {string} baseSha
 * @param {string} headSha
 * @param {any} config
 */
async function checkContinuationPolicy(baseSha, headSha, config) {
  const { stdout: diff } = await git(["diff", "--binary", "--no-renames", baseSha, headSha]);
  const { enforcePullRequestLimits } = require("./create_pull_request.cjs");
  enforcePullRequestLimits(diff, parsePositive(config.max_patch_files, DEFAULT_MAX_PATCH_FILES));
  const sizeKb = Math.ceil(Buffer.byteLength(diff, "utf8") / 1024);
  const maxKb = parsePositive(config.max_patch_size, DEFAULT_MAX_PATCH_KB);
  if (sizeKb > maxKb) throw new Error(`${ERR_VALIDATION}: continuation changes (${sizeKb} KB) exceed the maximum allowed patch size (${maxKb} KB)`);
  const policyConfig = { ...config, protected_files_policy: config.protected_files_policy ?? "request_review" };
  const parsedProtection = checkFileProtection(diff, policyConfig);
  const { stdout: changedFileOutput } = await git(["diff", "--name-only", "-z", "--no-renames", baseSha, headSha]);
  const postApplyProtection = checkFileProtectionPostApply(changedFileOutput.split("\0").filter(Boolean), policyConfig);
  /** @type {ReturnType<typeof checkFileProtection> | ReturnType<typeof checkFileProtectionPostApply>} */
  let protection = parsedProtection;
  if (postApplyProtection.action === "deny" || parsedProtection.action === "allow") {
    protection = postApplyProtection.action === "allow" ? parsedProtection : postApplyProtection;
  } else if (postApplyProtection.action !== "allow") {
    protection = { ...parsedProtection, files: [...new Set([...parsedProtection.files, ...postApplyProtection.files])] };
  }
  if (protection.action === "deny") {
    const files = protection.files.join(", ");
    throw new Error(
      protection.source === "allowlist" || protection.source === "post-apply"
        ? `${ERR_VALIDATION}: continuation modifies files outside the allowed-files list (${files})`
        : `${ERR_VALIDATION}: continuation modifies protected files (${files})`
    );
  }
  if (protection.action === "fallback" || protection.action === "request_review") {
    return { protectedFiles: protection.files, protectedAction: protection.action };
  }
  return { protectedFiles: [], protectedAction: null };
}

/**
 * Persists one hop of commits to the stable work branch.
 *
 * The bundle is applied locally, the commits must descend from the expected base recorded in
 * the ledger, trusted policy is applied to base..head, and the branch is advanced only if it
 * still points at the expected base (signed commits: expectedHeadOid; unsigned: non-forced
 * fast-forward push). A retry requires the exact unsigned commit or a verified signed replay
 * of the same commits, trees, messages and parent chain rooted at the expected base.
 *
 * @param {{bundlePath: string, agentBranch: string, workBranch: string, baseBranch: string, expectedBase: string, isInitial: boolean, config: any, githubClient: any}} options
 * @returns {Promise<{headSha: string, protectedFiles: string[], protectedAction: string|null, alreadyPersisted: boolean}>}
 */
async function persistContinuationHop(options) {
  const { bundlePath, agentBranch, workBranch, baseBranch, expectedBase, isInitial, config, githubClient } = options;
  if (!ledger.isSha(expectedBase)) throw new Error(`${ERR_VALIDATION}: continuation base is not a commit SHA`);
  const { applyBundleToBranch } = require("./create_pull_request.cjs");
  await ensureCommitAvailable(expectedBase);
  await applyBundleToBranch(bundlePath, workBranch, agentBranch, exec, baseBranch);
  const headSha = (await git(["rev-parse", "HEAD^{commit}"])).stdout.trim();
  if (headSha === expectedBase) throw new Error(`${ERR_VALIDATION}: continuation contains no new commits; commit the plan.md update before calling continue_workflow`);
  const ancestry = await git(["merge-base", "--is-ancestor", expectedBase, headSha], { ignoreReturnCode: true });
  if (ancestry.exitCode !== 0) throw new Error(`${ERR_VALIDATION}: continuation commits do not build on the work branch head ${expectedBase}`);
  const policy = await checkContinuationPolicy(expectedBase, headSha, config);

  const signedCommits = config.signed_commits !== false;
  const remoteHead = await readRemoteHead(githubClient, workBranch);
  const expectedRemote = isInitial ? null : expectedBase;
  if (remoteHead !== expectedRemote) {
    if (remoteHead && (await isVerifiedContinuationReplay(githubClient, expectedBase, headSha, remoteHead, signedCommits))) {
      core.info(`Work branch ${workBranch} already contains this verified hop (${remoteHead}); treating as already persisted`);
      return { headSha: remoteHead, ...policy, alreadyPersisted: true };
    }
    throw new Error(`${ERR_VALIDATION}: work branch ${workBranch} moved unexpectedly (expected ${expectedRemote || "no branch"}, found ${remoteHead || "no branch"}); the branch was preserved`);
  }

  const pushedHead = await pushSignedCommits({
    githubClient,
    ...repoRef(),
    branch: workBranch,
    baseRef: expectedBase,
    cwd: process.cwd(),
    signedCommits,
    allowGitPushFallback: false,
    requireBaseRefMatch: signedCommits,
    validationConfig: config,
  });
  const persistedHead = await readRemoteHead(githubClient, workBranch);
  if (!persistedHead || !ledger.isSha(persistedHead)) throw new Error(`${ERR_API}: work branch ${workBranch} could not be resolved after push`);
  if (persistedHead !== pushedHead) throw new Error(`${ERR_VALIDATION}: work branch ${workBranch} moved after push (expected ${pushedHead}, found ${persistedHead}); the branch was preserved`);
  core.info(`Persisted continuation commits to ${workBranch} at ${persistedHead}`);
  return { headSha: persistedHead, ...policy, alreadyPersisted: false };
}

/**
 * Verifies that the hop-0 base commit is reachable from the base branch so the work branch
 * never carries unrelated history into the final pull request.
 * @param {any} githubClient
 * @param {string} baseCommit
 * @param {string} baseBranch
 */
async function assertBaseOnBranch(githubClient, baseCommit, baseBranch) {
  const { data } = await githubClient.rest.repos.compareCommitsWithBasehead({ ...repoRef(), basehead: `${baseCommit}...${baseBranch}`, per_page: 1 });
  if (data?.status !== "ahead" && data?.status !== "identical") {
    throw new Error(`${ERR_VALIDATION}: continuation base commit ${baseCommit} is not part of ${baseBranch}`);
  }
}

/**
 * Base branch of a new chain: the configured base branch, else the branch this run was
 * triggered on, else the repository default branch.
 * @param {string} configuredBaseBranch
 */
function resolveInitialBaseBranch(configuredBaseBranch) {
  if (configuredBaseBranch) return configuredBaseBranch;
  const ref = String(process.env.GITHUB_REF || context.ref || "");
  if (ref.startsWith("refs/heads/")) return ref.slice("refs/heads/".length);
  return String(context.payload?.repository?.default_branch || "");
}

/** Resolves the workflow file name and dispatch ref of the current run. */
function resolveDispatchTarget() {
  const workflowRef = process.env.GITHUB_WORKFLOW_REF || "";
  const [workflowPath, ref] = workflowRef.split("@");
  const workflowFile = (workflowPath || "").split("/").pop() || "";
  if (!ledger.WORKFLOW_FILE_PATTERN.test(workflowFile)) throw new Error(`${ERR_VALIDATION}: cannot resolve the workflow file for continuation from GITHUB_WORKFLOW_REF`);
  const dispatchRef = ref && /^refs\/(heads|tags)\/[A-Za-z0-9._/-]+$/.test(ref) ? ref : "";
  return { workflowFile, ref: dispatchRef };
}

/** Initiator of the chain, used by pre-activation to authorize maintenance dispatches. */
function resolveInitiator() {
  const raw = context.payload?.inputs?.aw_context;
  if (context.eventName === "workflow_dispatch" && context.actor === "github-actions[bot]" && typeof raw === "string") {
    try {
      const actor = JSON.parse(raw)?.actor;
      if (typeof actor === "string" && actor.trim()) return actor.trim();
    } catch {
      // fall through to the run actor
    }
  }
  return String(context.actor || "");
}

/**
 * Asks Agentic Maintenance to dispatch queued continuations now. The schedule is the
 * fallback, so failures here are warnings only.
 * @param {any} githubClient
 */
async function requestContinuationDispatch(githubClient) {
  try {
    const defaultBranch = context.payload?.repository?.default_branch;
    let ref = defaultBranch ? `refs/heads/${defaultBranch}` : "";
    if (!ref) {
      const { data } = await githubClient.rest.repos.get(repoRef());
      ref = `refs/heads/${data.default_branch}`;
    }
    await githubClient.rest.actions.createWorkflowDispatch({ ...repoRef(), workflow_id: MAINTENANCE_WORKFLOW, ref, inputs: { operation: DISPATCH_OPERATION } });
    core.info(`Requested Agentic Maintenance to dispatch the next continuation hop`);
    return true;
  } catch (error) {
    core.warning(`Could not request immediate continuation dispatch (${getErrorMessage(error)}); scheduled Agentic Maintenance will dispatch it`);
    return false;
  }
}

/** @type {HandlerFactoryFunction} */
async function main(config = {}) {
  const maxCount = 1;
  const isStaged = isStagedMode(config);
  const maxHops = ledger.parseMaxHops(config.max_hops);
  const branchPrefix = normalizeBranchName(config.branch_prefix || "") || "";
  const configuredBaseBranch = typeof config.base_branch === "string" ? config.base_branch.trim() : "";
  const githubClient = await createAuthenticatedGitHubClient(config);
  let processedCount = 0;

  return async function handleContinueWorkflow(message) {
    if (processedCount >= maxCount) {
      core.warning(`Skipping ${HANDLER_TYPE}: max count of ${maxCount} reached`);
      return { success: false, error: `Max count of ${maxCount} reached` };
    }
    processedCount++;

    const reason = typeof message?.reason === "string" ? message.reason.trim() : "";
    const nextSteps = typeof message?.next_steps === "string" ? message.next_steps.trim() : "";
    const agentBranch = typeof message?.branch === "string" ? message.branch.trim() : "";
    if (!reason) return { success: false, error: "continue_workflow requires a reason" };
    if (!agentBranch) return { success: false, error: "continue_workflow requires the branch that holds the committed progress" };

    const active = activeContinuation;
    if (active?.settled) {
      return { success: true, skipped: true, reason: `continuation hop ${active.hop} of chain ${active.chain_id} was already settled` };
    }

    const runId = currentRunId();
    const workflowId = process.env.GH_AW_WORKFLOW_ID || "";
    if (!workflowId) return { success: false, error: "GH_AW_WORKFLOW_ID is required for continue_workflow" };
    const hop = active ? active.hop : 0;
    const isInitial = !active;
    const chainId = active ? active.chain_id : ledger.buildChainId(workflowId, runId);
    const workBranch = active ? active.branch : ledger.buildWorkBranch(workflowId, runId, branchPrefix);
    // Hop 0 starts from the commit this run checked out; later hops from the ledger's recorded head.
    const baseBranch = active ? active.base_branch : resolveInitialBaseBranch(configuredBaseBranch);
    const expectedBase = active ? active.base_sha : String(process.env.GITHUB_SHA || context.sha || "").trim();

    if (active && agentBranch !== active.branch) {
      return { success: false, error: `continue_workflow must be called on the continuation branch ${active.branch}, not ${agentBranch}` };
    }
    if (!baseBranch || normalizeBranchName(baseBranch) !== baseBranch) return { success: false, error: "continue_workflow could not resolve a valid base branch" };
    if (!ledger.isSha(expectedBase)) return { success: false, error: "continue_workflow could not resolve the base commit of the work branch" };

    const { bundlePath } = resolveTransportPaths(message, `${context.repo.owner}/${context.repo.repo}`);
    if (!bundlePath) return { success: false, error: "continue_workflow requires a git bundle with the committed progress" };

    if (isStaged) {
      logStagedPreviewInfo(`Would persist hop ${hop} to ${workBranch} and schedule hop ${hop + 1} of at most ${active ? active.max_hops : maxHops}`);
      return { success: true, staged: true, branch: workBranch, hop };
    }

    const failHop = async (/** @type {string} */ error) => {
      if (active) {
        try {
          await ledger.mutateChain(githubClient, repoRef(), chainId, current => ledger.failChain(current, { hop, runId, reason: error, now: nowIso() }), "Fail continuation hop");
        } catch (ledgerError) {
          core.warning(`Could not record continuation failure in the ledger: ${getErrorMessage(ledgerError)}`);
        }
      }
      return { success: false, error: `${error}. Work branch ${workBranch} was preserved.` };
    };

    try {
      if (isInitial) {
        const existing = await ledger.readChain(githubClient, repoRef(), chainId);
        if (existing && existing.hops.some(/** @param {any} h */ h => h.hop === 0 && h.run_id === runId)) {
          await requestContinuationDispatch(githubClient);
          return { success: true, skipped: true, reason: `continuation chain ${chainId} already recorded hop 0` };
        }
        await assertBaseOnBranch(githubClient, expectedBase, baseBranch);
      }

      const persisted = await persistContinuationHop({ bundlePath, agentBranch, workBranch, baseBranch, expectedBase, isInitial, config, githubClient });
      const now = nowIso();
      const recordInput = { hop, runId, headSha: persisted.headSha, reason, nextSteps, protectedFiles: persisted.protectedFiles, protectedAction: persisted.protectedAction || undefined, now };
      let initialChain;
      if (isInitial) {
        const target = resolveDispatchTarget();
        initialChain = {
          chainId,
          runId,
          workflowId,
          workflowFile: target.workflowFile,
          ref: target.ref,
          repository: `${context.repo.owner}/${context.repo.repo}`,
          initiator: resolveInitiator(),
          baseBranch,
          baseCommit: expectedBase,
          branch: workBranch,
          maxHops,
          now,
        };
      }
      const { chain } = await ledger.mutateChain(
        githubClient,
        repoRef(),
        chainId,
        current => {
          const started = initialChain ? ledger.startChain(current, initialChain) : { chain: current, changed: false };
          const recorded = ledger.recordContinuation(started.chain, recordInput);
          return { chain: recorded.chain, changed: started.changed || recorded.changed };
        },
        `Record continuation hop ${hop}`
      );

      if (chain.status === "exhausted") {
        core.warning(chain.error);
        await core.summary.addRaw(`\n## Continuation budget exhausted\n\n- **Chain**: \`${chainId}\`\n- **Work branch**: \`${workBranch}\` (preserved)\n- **Hops used**: ${chain.max_hops}\n`).write();
        return { success: false, error: chain.error };
      }

      await requestContinuationDispatch(githubClient);
      await core.summary.addRaw(`\n## Workflow continuation scheduled\n\n- **Chain**: \`${chainId}\`\n- **Work branch**: \`${workBranch}\` at \`${persisted.headSha}\`\n- **Next hop**: ${chain.next.hop} of ${chain.max_hops}\n`).write();
      return { success: true, chain_id: chainId, branch: workBranch, head_sha: persisted.headSha, hop, next_hop: chain.next.hop, max_hops: chain.max_hops };
    } catch (error) {
      const errorMessage = getErrorMessage(error);
      core.error(`continue_workflow failed: ${errorMessage}`);
      return failHop(errorMessage);
    }
  };
}

/**
 * Settles a continuation run after all safe outputs were processed:
 * - continue_workflow already recorded its own outcome;
 * - a successful create_pull_request completes the chain with the single final pull request;
 * - anything else fails the chain and preserves the work branch.
 * @param {any} config
 * @param {Array<any>} results - processMessages results
 * @param {any} [githubClient]
 */
async function finalizeContinueWorkflow(config, results, githubClient) {
  const active = activeContinuation;
  if (!active || active.settled || isStagedMode(config || {})) return;
  const list = Array.isArray(results) ? results : [];
  // A successful continue_workflow already recorded the hop; a failed one may have left the
  // hop running (for example on input validation), so it is settled as failed below.
  const continuation = list.find(r => r?.type === HANDLER_TYPE);
  if (continuation?.success === true) return;
  const client = githubClient || (await createAuthenticatedGitHubClient(config || {}));
  const runId = currentRunId();
  const prResult = list.find(r => r?.type === "create_pull_request" && r.success === true && !r.skipped);
  const pr = prResult?.result;
  try {
    if (pr && pr.number && pr.url && !pr.fallback_used) {
      await ledger.mutateChain(
        client,
        repoRef(),
        active.chain_id,
        current => ledger.completeChain(current, { hop: active.hop, runId, headSha: finalHeadSha || undefined, pullRequest: { number: pr.number, url: pr.url }, now: nowIso() }),
        "Complete continuation chain"
      );
      core.info(`Continuation chain ${active.chain_id} completed with pull request #${pr.number}`);
      return;
    }
    const reason = continuation
      ? `continue_workflow failed in continuation hop ${active.hop}: ${continuation.error || continuation.result?.error || "unknown error"}`
      : pr?.fallback_used
        ? `final hop produced fallback issue ${pr.issue_url || ""} instead of a pull request`.trim()
        : `continuation hop ${active.hop} ended without continue_workflow or a created pull request`;
    await ledger.mutateChain(client, repoRef(), active.chain_id, current => ledger.failChain(current, { hop: active.hop, runId, reason, now: nowIso() }), "Fail continuation chain");
    const message = `Continuation chain ${active.chain_id} stopped: ${reason}. Work branch ${active.branch} was preserved.`;
    await core.summary.addRaw(`\n## Workflow continuation stopped\n\n${message}\n`).write();
    if (pr?.fallback_used) core.warning(message);
    else core.setFailed(`${ERR_VALIDATION}: ${message}`);
  } catch (error) {
    core.setFailed(`${ERR_API}: failed to settle continuation chain ${active.chain_id}: ${getErrorMessage(error)}`);
  }
}

module.exports = {
  main,
  HANDLER_TYPE,
  DISPATCH_OPERATION,
  MAINTENANCE_WORKFLOW,
  prepareContinueWorkflow,
  finalizeContinueWorkflow,
  getActiveContinuation,
  noteFinalHead,
  persistContinuationHop,
  resetContinueWorkflowState,
};
