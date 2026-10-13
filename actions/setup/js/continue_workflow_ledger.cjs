// @ts-check
/// <reference types="@actions/github-script" />
// @safe-outputs-exempt SEC-005: the continuation ledger is stored only in the current GitHub Actions repository.
"use strict";

/**
 * Continuation ledger for the continue_workflow safe output.
 *
 * Each continuation chain is stored as one JSON document (`chains/<chain_id>.json`) on a
 * dedicated branch. Every transition is a pure function over the chain document and every
 * write is a compare-and-swap: the new commit's only parent is the ledger head that was read,
 * and the ref is advanced with a non-forced update. Concurrent writers therefore observe a
 * 422 and re-run their transition against the fresh state, which makes duplicate deliveries
 * (re-run jobs, repeated maintenance ticks) idempotent instead of creating duplicate hops.
 */

const { getErrorMessage } = require("./error_helpers.cjs");
const { ERR_API, ERR_VALIDATION } = require("./error_codes.cjs");

const LEDGER_BRANCH = "continue-workflow-ledger";
const LEDGER_VERSION = 1;
const DEFAULT_MAX_HOPS = 3;
const MAX_HOPS_LIMIT = 10;
const MAX_DISPATCH_ATTEMPTS = 3;
const STALE_AFTER_MS = 24 * 60 * 60 * 1000;
const MAX_CAS_ATTEMPTS = 5;
const MAX_TEXT_LENGTH = 2000;
const DISPATCH_CURSOR_PATH = "dispatch-cursor.json";
const CHAIN_ID_PATTERN = /^[a-z0-9][a-z0-9._-]{0,127}$/;
const SHA_PATTERN = /^[0-9a-f]{40}$/;
const DISPATCH_ID_PATTERN = /^[A-Za-z0-9_-]{8,64}$/;
const WORKFLOW_FILE_PATTERN = /^[A-Za-z0-9][A-Za-z0-9._-]{0,99}\.ya?ml$/;
const ACTIVE = "active";
const TERMINAL_STATUSES = new Set(["completed", "exhausted", "failed"]);

class ContinueWorkflowError extends Error {
  /**
   * @param {string} code
   * @param {string} message
   */
  constructor(code, message) {
    super(`${ERR_VALIDATION}: ${message}`);
    this.code = code;
  }
}

/**
 * @param {string} code
 * @param {string} message
 * @returns {never}
 */
function fail(code, message) {
  throw new ContinueWorkflowError(code, message);
}

/** @param {unknown} value */
function clip(value) {
  if (typeof value !== "string") return "";
  const trimmed = value.trim();
  return trimmed.length > MAX_TEXT_LENGTH ? `${trimmed.slice(0, MAX_TEXT_LENGTH)}…` : trimmed;
}

/**
 * Normalizes a workflow identifier into a branch- and path-safe segment.
 * @param {string} workflowId
 */
function workflowSegment(workflowId) {
  const segment = String(workflowId || "")
    .toLowerCase()
    .replace(/[^a-z0-9._-]+/g, "-")
    .replace(/\.{2,}/g, ".")
    .replace(/^[-._]+|[-._]+$/g, "")
    .slice(0, 64)
    .replace(/[-._]+$/g, "");
  return segment || "workflow";
}

/**
 * @param {string} workflowId
 * @param {string|number} runId
 */
function buildChainId(workflowId, runId) {
  const id = `${workflowSegment(workflowId)}-${String(runId)}`;
  if (!isValidChainId(id)) fail("invalid_chain", `cannot derive a continuation chain id from workflow '${workflowId}' and run '${runId}'`);
  return id;
}

/**
 * @param {string} workflowId
 * @param {string|number} runId
 * @param {string} [branchPrefix]
 */
function buildWorkBranch(workflowId, runId, branchPrefix = "") {
  return `${branchPrefix || ""}continue-workflow/${workflowSegment(workflowId)}/${String(runId)}`;
}

/** @param {unknown} id */
function isValidChainId(id) {
  return typeof id === "string" && CHAIN_ID_PATTERN.test(id) && !id.includes("..");
}

/** @param {unknown} sha */
function isSha(sha) {
  return typeof sha === "string" && SHA_PATTERN.test(sha);
}

/** @param {unknown} value */
function parseMaxHops(value) {
  const parsed = typeof value === "number" ? value : typeof value === "string" && /^\d+$/.test(value.trim()) ? Number(value.trim()) : NaN;
  if (Number.isSafeInteger(parsed) && parsed >= 1 && parsed <= MAX_HOPS_LIMIT) return parsed;
  return DEFAULT_MAX_HOPS;
}

/**
 * Validates a ledger document read from storage.
 * @param {any} chain
 * @param {string} chainId
 */
function validateChain(chain, chainId) {
  if (!chain || typeof chain !== "object" || Array.isArray(chain)) fail("ledger_invalid", `continuation chain ${chainId} is not an object`);
  if (chain.version !== LEDGER_VERSION) fail("ledger_invalid", `continuation chain ${chainId} has unsupported version ${String(chain.version)}`);
  if (chain.chain_id !== chainId) fail("ledger_invalid", `continuation chain document does not match id ${chainId}`);
  if (typeof chain.branch !== "string" || !chain.branch) fail("ledger_invalid", `continuation chain ${chainId} has no work branch`);
  if (!Number.isSafeInteger(chain.max_hops) || chain.max_hops < 1) fail("ledger_invalid", `continuation chain ${chainId} has an invalid max_hops`);
  if (chain.status !== ACTIVE && !TERMINAL_STATUSES.has(chain.status)) fail("ledger_invalid", `continuation chain ${chainId} has an invalid status`);
  if (!Array.isArray(chain.hops)) fail("ledger_invalid", `continuation chain ${chainId} has no hop history`);
  if (chain.status === ACTIVE && (!chain.next || typeof chain.next !== "object")) fail("ledger_invalid", `active continuation chain ${chainId} has no next hop`);
  return chain;
}

/**
 * Parses the continuation request carried inside the workflow_dispatch aw_context input.
 * Returns null when the run is not a continuation.
 * @param {any} payload
 * @returns {{chain_id: string, hop: number, dispatch_id: string} | null}
 */
function parseContinuationRequest(payload) {
  const raw = payload?.inputs?.aw_context;
  if (typeof raw !== "string" || !raw.trim()) return null;
  let parsed;
  try {
    parsed = JSON.parse(raw);
  } catch {
    return null;
  }
  const continuation = parsed && typeof parsed === "object" && !Array.isArray(parsed) ? parsed.continuation : undefined;
  if (continuation === undefined || continuation === null) return null;
  if (typeof continuation !== "object" || Array.isArray(continuation)) fail("invalid_request", "aw_context.continuation must be an object");
  const { chain_id: chainId, hop, dispatch_id: dispatchId } = continuation;
  if (!isValidChainId(chainId)) fail("invalid_request", "aw_context.continuation.chain_id is invalid");
  if (!Number.isSafeInteger(hop) || hop < 1 || hop > MAX_HOPS_LIMIT) fail("invalid_request", "aw_context.continuation.hop is invalid");
  if (typeof dispatchId !== "string" || !DISPATCH_ID_PATTERN.test(dispatchId)) fail("invalid_request", "aw_context.continuation.dispatch_id is invalid");
  return { chain_id: chainId, hop, dispatch_id: dispatchId };
}

// ---------------------------------------------------------------------------
// Pure transitions. Each returns `{ chain, changed }` and never mutates its input.
// ---------------------------------------------------------------------------

/** @param {any} chain */
function clone(chain) {
  return chain ? JSON.parse(JSON.stringify(chain)) : chain;
}

/**
 * Creates a chain for the initial run (hop 0). Re-delivery of the same run is idempotent.
 * @param {any} current
 * @param {{chainId: string, runId: string, workflowId: string, workflowFile: string, ref: string, repository: string, initiator: string, baseBranch: string, baseCommit: string, branch: string, maxHops: number, now: string}} input
 */
function startChain(current, input) {
  if (current) {
    const first = current.hops?.[0];
    if (first && first.hop === 0 && first.run_id === input.runId) return { chain: current, changed: false };
    if (current.next?.hop === 0 && current.next.run_id === input.runId) return { chain: current, changed: false };
    fail("chain_conflict", `continuation chain ${input.chainId} already exists for another run`);
  }
  if (!WORKFLOW_FILE_PATTERN.test(input.workflowFile)) fail("invalid_workflow", `cannot dispatch continuation for workflow file '${input.workflowFile}'`);
  if (!isSha(input.baseCommit)) fail("invalid_base", "initial continuation base commit is not a commit SHA");
  return {
    changed: true,
    chain: {
      version: LEDGER_VERSION,
      chain_id: input.chainId,
      repository: input.repository,
      workflow_id: input.workflowId,
      workflow_file: input.workflowFile,
      ref: input.ref,
      initiator: input.initiator,
      base_branch: input.baseBranch,
      base_commit: input.baseCommit,
      branch: input.branch,
      max_hops: input.maxHops,
      status: ACTIVE,
      created_at: input.now,
      updated_at: input.now,
      hops: [],
      next: { hop: 0, state: "running", run_id: input.runId, base_sha: input.baseCommit, dispatch_attempts: 0, updated_at: input.now },
      protected_files: [],
      pull_request: null,
      error: null,
    },
  };
}

/**
 * Binds a dispatched continuation hop to the run that received it.
 * @param {any} current
 * @param {{hop: number, dispatchId: string, runId: string, now: string}} input
 */
function bindHop(current, input) {
  if (!current) fail("chain_missing", "continuation chain does not exist");
  const chain = clone(current);
  if (chain.hops.some(/** @param {any} h */ h => h.hop === input.hop && h.run_id === input.runId)) return { chain: current, changed: false };
  if (chain.status !== ACTIVE) fail("chain_closed", `continuation chain ${chain.chain_id} is ${chain.status}`);
  const next = chain.next;
  if (next.hop !== input.hop) fail("hop_mismatch", `continuation chain ${chain.chain_id} expects hop ${next.hop}, not hop ${input.hop}`);
  if (next.dispatch_id !== input.dispatchId) fail("dispatch_mismatch", `continuation hop ${input.hop} was not dispatched with this dispatch id`);
  if (next.state === "running") {
    if (next.run_id === input.runId) return { chain: current, changed: false };
    fail("hop_claimed", `continuation hop ${input.hop} is already bound to run ${next.run_id}`);
  }
  if (next.state !== "dispatched") fail("hop_not_dispatched", `continuation hop ${input.hop} is ${next.state}, not dispatched`);
  chain.next = { ...next, state: "running", run_id: input.runId, updated_at: input.now };
  chain.updated_at = input.now;
  return { chain, changed: true };
}

/**
 * @param {any} chain
 * @param {{hop: number, runId: string}} input
 */
function requireRunningHop(chain, input) {
  if (chain.status !== ACTIVE) fail("chain_closed", `continuation chain ${chain.chain_id} is ${chain.status}`);
  if (chain.next.hop !== input.hop || chain.next.state !== "running" || chain.next.run_id !== input.runId) {
    fail("hop_not_bound", `continuation hop ${input.hop} is not bound to run ${input.runId}`);
  }
}

/**
 * @param {any} chain
 * @param {number} hop
 * @param {string} runId
 */
function findRecordedHop(chain, hop, runId) {
  return chain.hops.find(/** @param {any} h */ h => h.hop === hop && h.run_id === runId);
}

/** @param {string[]} existing @param {string[]} added */
function unionFiles(existing, added) {
  return [...new Set([...(Array.isArray(existing) ? existing : []), ...(Array.isArray(added) ? added : [])])].sort();
}

/**
 * Records a persisted hop and queues the next one, or exhausts the chain when the hop budget is spent.
 * @param {any} current
 * @param {{hop: number, runId: string, headSha: string, reason: string, nextSteps?: string, protectedFiles?: string[], protectedAction?: string, now: string}} input
 */
function recordContinuation(current, input) {
  if (!current) fail("chain_missing", "continuation chain does not exist");
  if (findRecordedHop(current, input.hop, input.runId)) return { chain: current, changed: false };
  const chain = clone(current);
  requireRunningHop(chain, input);
  if (!isSha(input.headSha)) fail("invalid_head", "continuation head is not a commit SHA");
  const exhausted = input.hop + 1 > chain.max_hops;
  chain.hops.push({ hop: input.hop, run_id: input.runId, outcome: exhausted ? "exhausted" : "continued", head_sha: input.headSha, reason: clip(input.reason), next_steps: clip(input.nextSteps), at: input.now });
  chain.protected_files = unionFiles(chain.protected_files, input.protectedFiles || []);
  if (input.protectedAction) chain.protected_action = input.protectedAction;
  chain.updated_at = input.now;
  if (exhausted) {
    chain.status = "exhausted";
    chain.error = `continuation budget of ${chain.max_hops} hop(s) is exhausted; work branch ${chain.branch} was preserved`;
    chain.next = null;
  } else {
    chain.next = { hop: input.hop + 1, state: "queued", base_sha: input.headSha, dispatch_attempts: 0, last_error: null, updated_at: input.now };
  }
  return { chain, changed: true };
}

/**
 * Completes the chain with the single final pull request.
 * @param {any} current
 * @param {{hop: number, runId: string, headSha?: string, pullRequest: {number?: number, url?: string}, now: string}} input
 */
function completeChain(current, input) {
  if (!current) fail("chain_missing", "continuation chain does not exist");
  const recorded = findRecordedHop(current, input.hop, input.runId);
  if (recorded && recorded.outcome === "completed") return { chain: current, changed: false };
  const chain = clone(current);
  requireRunningHop(chain, input);
  chain.hops.push({ hop: input.hop, run_id: input.runId, outcome: "completed", head_sha: isSha(input.headSha) ? input.headSha : null, reason: "", next_steps: "", at: input.now });
  chain.status = "completed";
  chain.pull_request = { number: input.pullRequest.number ?? null, url: input.pullRequest.url ?? null };
  chain.next = null;
  chain.error = null;
  chain.updated_at = input.now;
  return { chain, changed: true };
}

/**
 * Fails the chain while preserving its work branch. A run may only fail its own bound hop.
 * @param {any} current
 * @param {{hop?: number, runId?: string, reason: string, now: string}} input
 */
function failChain(current, input) {
  if (!current) fail("chain_missing", "continuation chain does not exist");
  if (current.status !== ACTIVE) return { chain: current, changed: false };
  const chain = clone(current);
  if (input.runId !== undefined) {
    requireRunningHop(chain, { hop: Number(input.hop), runId: input.runId });
    chain.hops.push({ hop: input.hop, run_id: input.runId, outcome: "failed", head_sha: null, reason: clip(input.reason), next_steps: "", at: input.now });
  }
  chain.status = "failed";
  chain.error = clip(input.reason);
  chain.failed_next = chain.next;
  chain.next = null;
  chain.updated_at = input.now;
  return { chain, changed: true };
}

/**
 * Moves a queued hop to dispatched with a fresh dispatch id (one attempt).
 * @param {any} current
 * @param {{dispatchId: string, now: string}} input
 */
function claimDispatch(current, input) {
  if (!current || current.status !== ACTIVE || current.next?.state !== "queued") return { chain: current, changed: false };
  if (!DISPATCH_ID_PATTERN.test(input.dispatchId)) fail("invalid_dispatch", "dispatch id is invalid");
  const chain = clone(current);
  chain.next = { ...chain.next, state: "dispatched", dispatch_id: input.dispatchId, dispatch_attempts: (chain.next.dispatch_attempts || 0) + 1, dispatched_at: input.now, updated_at: input.now };
  chain.updated_at = input.now;
  return { chain, changed: true };
}

/**
 * Records a failed dispatch attempt: re-queue while attempts remain, otherwise fail the chain.
 * @param {any} current
 * @param {{dispatchId: string, error: string, now: string, maxAttempts?: number}} input
 */
function recordDispatchFailure(current, input) {
  if (!current || current.status !== ACTIVE || current.next?.state !== "dispatched" || current.next.dispatch_id !== input.dispatchId) return { chain: current, changed: false };
  const chain = clone(current);
  const maxAttempts = input.maxAttempts || MAX_DISPATCH_ATTEMPTS;
  if ((chain.next.dispatch_attempts || 0) >= maxAttempts) {
    return failChain(current, { reason: `continuation hop ${chain.next.hop} could not be dispatched after ${chain.next.dispatch_attempts} attempt(s): ${clip(input.error)}`, now: input.now });
  }
  chain.next = { ...chain.next, state: "queued", dispatch_id: undefined, last_error: clip(input.error), updated_at: input.now };
  delete chain.next.dispatch_id;
  chain.updated_at = input.now;
  return { chain, changed: true };
}

/**
 * Fails chains whose dispatched or running hop never settled (agent failure, detection block,
 * cancelled run), so they are reported instead of silently hanging.
 * @param {any} current
 * @param {{now: string, staleAfterMs?: number}} input
 */
function expireStale(current, input) {
  if (!current || current.status !== ACTIVE) return { chain: current, changed: false };
  const state = current.next?.state;
  if (state !== "dispatched" && state !== "running") return { chain: current, changed: false };
  const since = Date.parse(current.next.updated_at || current.updated_at || "");
  const now = Date.parse(input.now);
  if (!Number.isFinite(since) || !Number.isFinite(now) || now - since <= (input.staleAfterMs ?? STALE_AFTER_MS)) return { chain: current, changed: false };
  const runNote = current.next.run_id ? ` (run ${current.next.run_id})` : "";
  return failChain(current, { reason: `continuation hop ${current.next.hop}${runNote} did not settle; work branch ${current.branch} was preserved`, now: input.now });
}

// ---------------------------------------------------------------------------
// Storage
// ---------------------------------------------------------------------------

/** @param {any} error */
function httpStatus(error) {
  return error?.status ?? error?.response?.status;
}

/** @param {string} chainId */
function chainPath(chainId) {
  if (!isValidChainId(chainId)) fail("invalid_chain", `invalid continuation chain id '${chainId}'`);
  return `chains/${chainId}.json`;
}

/**
 * @param {any} githubClient
 * @param {{owner: string, repo: string}} repoRef
 * @param {string} branch
 */
async function readLedgerHead(githubClient, { owner, repo }, branch) {
  try {
    const { data } = await githubClient.rest.git.getRef({ owner, repo, ref: `heads/${branch}` });
    const sha = data?.object?.sha;
    if (!isSha(sha)) throw new Error(`${ERR_API}: continuation ledger ref returned no commit`);
    const commit = await githubClient.rest.git.getCommit({ owner, repo, commit_sha: sha });
    return { sha, treeSha: commit.data.tree.sha };
  } catch (error) {
    if (httpStatus(error) === 404) return null;
    throw error;
  }
}

/**
 * @param {any} githubClient
 * @param {{owner: string, repo: string}} repoRef
 * @param {string} commitSha
 * @param {string} path
 */
async function readLedgerDocumentAt(githubClient, { owner, repo }, commitSha, path) {
  try {
    const { data } = await githubClient.rest.repos.getContent({ owner, repo, path, ref: commitSha });
    if (!data || Array.isArray(data) || typeof data.content !== "string") fail("ledger_invalid", `continuation ledger document ${path} is not a file`);
    const text = Buffer.from(data.content, data.encoding === "base64" ? "base64" : "utf8").toString("utf8");
    const document = JSON.parse(text);
    if (!document || typeof document !== "object" || Array.isArray(document)) fail("ledger_invalid", `continuation ledger document ${path} is not an object`);
    return document;
  } catch (error) {
    if (httpStatus(error) === 404) return null;
    if (error instanceof SyntaxError) fail("ledger_invalid", `continuation ledger document ${path} is not valid JSON`);
    throw error;
  }
}

/**
 * @param {any} githubClient
 * @param {{owner: string, repo: string}} repoRef
 * @param {string} commitSha
 * @param {string} chainId
 */
async function readChainAt(githubClient, repoRef, commitSha, chainId) {
  const document = await readLedgerDocumentAt(githubClient, repoRef, commitSha, chainPath(chainId));
  return document === null ? null : validateChain(document, chainId);
}

/**
 * Reads one chain from the ledger head.
 * @param {any} githubClient
 * @param {{owner: string, repo: string, branch?: string}} options
 * @param {string} chainId
 */
async function readChain(githubClient, options, chainId) {
  const head = await readLedgerHead(githubClient, options, options.branch || LEDGER_BRANCH);
  if (!head) return null;
  return readChainAt(githubClient, options, head.sha, chainId);
}

/**
 * Applies a pure transition to one chain with compare-and-swap semantics.
 * @param {any} githubClient
 * @param {{owner: string, repo: string, branch?: string, maxAttempts?: number}} options
 * @param {string} chainId
 * @param {(current: any) => {chain: any, changed: boolean}} transition
 * @param {string} message
 * @returns {Promise<{chain: any, changed: boolean}>}
 */
async function mutateChain(githubClient, options, chainId, transition, message) {
  const result = await mutateLedgerDocument(
    githubClient,
    options,
    chainPath(chainId),
    current => {
      const next = transition(current === null ? null : validateChain(current, chainId));
      if (next.changed) validateChain(next.chain, chainId);
      return { document: next.chain, changed: next.changed };
    },
    `${message} (${chainId})`
  );
  return { chain: result.document, changed: result.changed };
}

/**
 * @param {any} githubClient
 * @param {{owner: string, repo: string, branch?: string, maxAttempts?: number}} options
 * @param {string} path
 * @param {(current: any) => {document: any, changed: boolean}} transition
 * @param {string} message
 */
async function mutateLedgerDocument(githubClient, options, path, transition, message) {
  const { owner, repo } = options;
  const branch = options.branch || LEDGER_BRANCH;
  const maxAttempts = options.maxAttempts || MAX_CAS_ATTEMPTS;
  let lastError;
  for (let attempt = 1; attempt <= maxAttempts; attempt++) {
    const head = await readLedgerHead(githubClient, { owner, repo }, branch);
    const current = head ? await readLedgerDocumentAt(githubClient, { owner, repo }, head.sha, path) : null;
    const result = transition(current);
    if (!result.changed) return result;
    try {
      const content = `${JSON.stringify(result.document, null, 2)}\n`;
      const blob = await githubClient.rest.git.createBlob({ owner, repo, content, encoding: "utf-8" });
      const tree = await githubClient.rest.git.createTree({ owner, repo, ...(head ? { base_tree: head.treeSha } : {}), tree: [{ path, mode: "100644", type: "blob", sha: blob.data.sha }] });
      const commit = await githubClient.rest.git.createCommit({ owner, repo, message, tree: tree.data.sha, parents: head ? [head.sha] : [] });
      if (head) await githubClient.rest.git.updateRef({ owner, repo, ref: `heads/${branch}`, sha: commit.data.sha, force: false });
      else await githubClient.rest.git.createRef({ owner, repo, ref: `refs/heads/${branch}`, sha: commit.data.sha });
      core.info(`Continuation ledger updated: ${message}`);
      return result;
    } catch (error) {
      lastError = error;
      const status = httpStatus(error);
      if (status !== 409 && status !== 422) throw error;
      core.info(`Continuation ledger write conflicted (attempt ${attempt}/${maxAttempts}); re-reading ledger`);
    }
  }
  throw new Error(`${ERR_API}: continuation ledger write did not converge after ${maxAttempts} attempt(s): ${getErrorMessage(lastError)}`);
}

/** @param {any} document @returns {string|null} */
function dispatchCursor(document) {
  if (document === null) return null;
  if (!document || document.version !== LEDGER_VERSION || !isValidChainId(document.after_chain_id)) {
    fail("ledger_invalid", "continuation dispatch cursor is invalid");
  }
  return document.after_chain_id;
}

/**
 * @param {any} githubClient
 * @param {{owner: string, repo: string}} repoRef
 * @param {string} commitSha
 */
async function readDispatchCursorAt(githubClient, repoRef, commitSha) {
  return dispatchCursor(await readLedgerDocumentAt(githubClient, repoRef, commitSha, DISPATCH_CURSOR_PATH));
}

/**
 * Advances only the cursor this maintenance run read; an overlapping run cannot rewind it.
 * @param {any} githubClient
 * @param {{owner: string, repo: string}} repoRef
 * @param {string|null} expected
 * @param {string} next
 */
async function advanceDispatchCursor(githubClient, repoRef, expected, next) {
  if (!isValidChainId(next) || (expected !== null && !isValidChainId(expected))) fail("invalid_cursor", "continuation dispatch cursor must name a valid chain");
  return mutateLedgerDocument(
    githubClient,
    repoRef,
    DISPATCH_CURSOR_PATH,
    current => {
      const previous = dispatchCursor(current);
      if (previous !== expected || previous === next) return { document: current, changed: false };
      return { document: { version: LEDGER_VERSION, after_chain_id: next }, changed: true };
    },
    "Advance continuation dispatch cursor"
  );
}

/**
 * Lists all chain ids recorded on the ledger branch.
 * @param {any} githubClient
 * @param {{owner: string, repo: string, branch?: string}} options
 * @param {{sha: string, treeSha: string}} [snapshot]
 * @returns {Promise<string[]>}
 */
async function listChainIds(githubClient, options, snapshot) {
  const head = snapshot || (await readLedgerHead(githubClient, options, options.branch || LEDGER_BRANCH));
  if (!head) return [];
  const { data } = await githubClient.rest.git.getTree({ owner: options.owner, repo: options.repo, tree_sha: head.treeSha, recursive: "true" });
  if (data.truncated) throw new Error(`${ERR_API}: continuation ledger tree is truncated; refusing to skip undiscovered chains`);
  return (data.tree || [])
    .map(/** @param {any} entry */ entry => (entry.type === "blob" && typeof entry.path === "string" ? /^chains\/([^/]+)\.json$/.exec(entry.path)?.[1] : undefined))
    .filter(/** @param {any} id */ id => isValidChainId(id));
}

module.exports = {
  LEDGER_BRANCH,
  LEDGER_VERSION,
  DEFAULT_MAX_HOPS,
  MAX_HOPS_LIMIT,
  MAX_DISPATCH_ATTEMPTS,
  STALE_AFTER_MS,
  WORKFLOW_FILE_PATTERN,
  ContinueWorkflowError,
  workflowSegment,
  buildChainId,
  buildWorkBranch,
  isValidChainId,
  isSha,
  parseMaxHops,
  parseContinuationRequest,
  validateChain,
  readLedgerHead,
  readChainAt,
  startChain,
  bindHop,
  recordContinuation,
  completeChain,
  failChain,
  claimDispatch,
  recordDispatchFailure,
  expireStale,
  readChain,
  mutateChain,
  listChainIds,
  readDispatchCursorAt,
  advanceDispatchCursor,
};
