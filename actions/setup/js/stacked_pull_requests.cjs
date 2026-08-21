// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Stacked pull request support for the create_pull_request safe output.
 *
 * A stacked pull request targets the branch of another pull request instead of the default base
 * branch, so a chain of dependent changes can be reviewed and merged one at a time. Everything
 * related to stacks (configuration, run-scoped tracking, validation, GitHub API calls and body
 * metadata) lives in this file.
 *
 * @safe-outputs-exempt SEC-005: `targetRepo` in this file is a display-only "owner/repo" string
 * used solely for log/error message formatting (e.g. in `verifyStackBaseBranchExists`). It is not
 * an independent cross-repo configuration surface — the actual repo resolution and allowlist
 * enforcement happen in the caller, `create_pull_request.cjs`, via `resolveTargetRepoConfig()` /
 * `resolveAndValidateRepo()`, before the already-validated value is passed into this file.
 */

const { getErrorMessage } = require("./error_helpers.cjs");
const { normalizeBranchName } = require("./normalize_branch_name.cjs");

/**
 * @typedef {object} StackEntry
 * @property {string} branch - Effective branch name that was pushed
 * @property {number} number - Pull request number
 * @property {string} url - Pull request HTML url
 * @property {string} repo - "owner/repo" the pull request was created in
 */

/**
 * Determine whether stacked pull requests are enabled for this run.
 * Stacking is enabled by default and can be turned off with `stacked: false` in the workflow
 * front matter, for GitHub Enterprise Server and other instances without stacked pull request
 * support.
 * @param {any} config - Handler configuration
 * @returns {boolean}
 */
function isStackedEnabled(config) {
  return !config || config.stacked !== false;
}

/**
 * Normalize the optional stacked pull request metadata carried by a create_pull_request message.
 * Branch-like values are normalized with normalizeBranchName so that agent-provided content can
 * never break out of the HTML comment used to record the metadata in the pull request body.
 * @param {{stack_position?: unknown, stack_root?: unknown, dependencies?: unknown}} item
 * @returns {{position: number|null, root: string|null, dependencies: string[]}}
 */
function parseStackMetadata(item) {
  const rawPosition = item && item.stack_position !== undefined ? Number(item.stack_position) : NaN;
  const position = Number.isInteger(rawPosition) && rawPosition > 0 ? rawPosition : null;

  const rawRoot = item && typeof item.stack_root === "string" ? item.stack_root.trim() : "";
  const root = rawRoot ? normalizeBranchName(rawRoot) || null : null;

  /** @type {string[]} */
  const dependencies = [];
  if (item && Array.isArray(item.dependencies)) {
    for (const dependency of item.dependencies) {
      if (typeof dependency !== "string") continue;
      const normalized = normalizeBranchName(dependency.trim());
      if (normalized && !dependencies.includes(normalized)) {
        dependencies.push(normalized);
      }
    }
  }

  return { position, root, dependencies };
}

/**
 * Detect a circular stacked pull request dependency.
 * Walks the chain of already-created stack branches (branch -> base branch) starting from the
 * requested base branch and reports a cycle when one of the current pull request's branch names
 * is reached again.
 * @param {(string|null|undefined)[]} branchAliases - Branch names identifying the pull request being created
 * @param {string} baseBranch - Requested base branch
 * @param {Map<string, string>} stackParents - Map of branch name to the base branch it was created from
 * @returns {boolean}
 */
function hasCircularStackDependency(branchAliases, baseBranch, stackParents) {
  const aliases = new Set(branchAliases.filter(alias => typeof alias === "string" && alias !== ""));
  if (aliases.size === 0 || !baseBranch) {
    return false;
  }
  if (aliases.has(baseBranch)) {
    return true;
  }
  const visited = new Set();
  let current = baseBranch;
  while (current && stackParents.has(current) && !visited.has(current)) {
    visited.add(current);
    current = stackParents.get(current) || "";
    if (aliases.has(current)) {
      return true;
    }
  }
  return false;
}

/**
 * Build the stacked pull request metadata lines appended to the pull request body.
 * The visible "Depends on #N" references link the stack together, while the HTML comment keeps
 * the machine-readable stack structure. Metadata is recorded even when stacked pull requests are
 * disabled so the stack structure declared by the agent is not lost.
 * @param {{base?: string|null, position?: number|null, root?: string|null, dependencies?: string[], dependsOnPullRequests?: number[]}} options
 * @returns {string[]}
 */
function buildStackMetadataLines(options) {
  const base = options.base || null;
  const position = options.position ?? null;
  const root = options.root || null;
  const dependencies = Array.isArray(options.dependencies) ? options.dependencies : [];
  const dependsOnPullRequests = Array.isArray(options.dependsOnPullRequests) ? options.dependsOnPullRequests.filter(n => Number.isInteger(n) && n > 0) : [];

  if (!base && position === null && !root && dependencies.length === 0 && dependsOnPullRequests.length === 0) {
    return [];
  }

  /** @type {string[]} */
  const lines = [];
  for (const number of dependsOnPullRequests) {
    lines.push(`Depends on #${number}`);
  }
  /** @type {Record<string, unknown>} */
  const metadata = {};
  if (base) metadata.base = base;
  if (position !== null) metadata.position = position;
  if (root) metadata.root = root;
  if (dependencies.length > 0) metadata.dependencies = dependencies;
  if (dependsOnPullRequests.length > 0) metadata.depends_on = dependsOnPullRequests;
  lines.push(`<!-- gh-aw-stack: ${JSON.stringify(metadata)} -->`);
  return lines;
}

/**
 * Error message returned when a stacked pull request is requested but stacking is disabled.
 * @param {string} requestedBaseBranch
 * @param {string} defaultBaseBranch
 * @returns {string}
 */
function stackedDisabledError(requestedBaseBranch, defaultBaseBranch) {
  return (
    `Stacked pull requests are disabled for this workflow, so base branch '${requestedBaseBranch}' cannot be used. ` +
    `Target the default base branch '${defaultBaseBranch}' instead, or remove safe-outputs.create-pull-request.stacked: false ` +
    `if this GitHub instance supports stacked pull requests. GitHub Enterprise Server may not support them.`
  );
}

/**
 * Error message returned when the base branch of a stacked pull request does not exist.
 * @param {string} baseBranch
 * @param {string} targetRepo
 * @param {string} defaultBaseBranch
 * @returns {string}
 */
function missingStackBaseError(baseBranch, targetRepo, defaultBaseBranch) {
  return (
    `Base branch '${baseBranch}' does not exist in ${targetRepo}, so the stacked pull request cannot be created. ` +
    `Create the pull request that owns '${baseBranch}' first (stacked pull requests must be emitted in dependency order), ` +
    `or target the default base branch '${defaultBaseBranch}'.`
  );
}

/**
 * Error message returned when a stacked pull request would depend on itself.
 * @param {string} branchName
 * @param {string} baseBranch
 * @returns {string}
 */
function circularStackError(branchName, baseBranch) {
  return `Circular stacked pull request dependency detected: branch '${branchName}' cannot depend on base branch '${baseBranch}'. Review the stack order and the dependencies field.`;
}

/**
 * Verify that the base branch of a stacked pull request already exists in the target repository.
 * A missing branch is reported with actionable guidance instead of an opaque API error; other API
 * failures (and clients without the repos.getBranch API) degrade to a warning so the pull request
 * creation is still attempted.
 * CONTRACT: `repoParts` and `targetRepo` MUST already be allowlist-validated by the caller
 * (create_pull_request.cjs's `resolveTargetRepoConfig()` / `resolveAndValidateRepo()`) before this
 * function is invoked. `targetRepo` is used here only for log/error message formatting, never for
 * an independent authorization decision — do not wire in unvalidated input from a future refactor.
 * @param {any} githubClient - Authenticated Octokit client
 * @param {{owner: string, repo: string}} repoParts - Must already be allowlist-validated by the caller
 * @param {string} baseBranch
 * @param {string} targetRepo - "owner/repo" used in log and error messages only; must already be allowlist-validated by the caller
 * @param {string} defaultBaseBranch
 * @returns {Promise<{success: boolean, error?: string}>}
 */
async function verifyStackBaseBranchExists(githubClient, repoParts, baseBranch, targetRepo, defaultBaseBranch) {
  const getBranchApi = githubClient?.rest?.repos?.getBranch;
  if (typeof getBranchApi !== "function") {
    core.warning(`Skipping base branch existence check for ${baseBranch}: repos.getBranch API is not available`);
    return { success: true };
  }
  try {
    await githubClient.rest.repos.getBranch({ owner: repoParts.owner, repo: repoParts.repo, branch: baseBranch });
    core.info(`Base branch ${baseBranch} exists in ${targetRepo}`);
  } catch (baseBranchError) {
    const status = /** @type {{ status?: number }} */ (baseBranchError || {}).status;
    if (status === 404) {
      const error = missingStackBaseError(baseBranch, targetRepo, defaultBaseBranch);
      core.warning(error);
      return { success: false, error };
    }
    core.warning(`Could not verify base branch ${baseBranch} in ${targetRepo}: ${getErrorMessage(baseBranchError)}`);
  }
  return { success: true };
}

/**
 * Run-scoped tracker for the pull requests created so far, used to resolve stack base branches and
 * to detect circular dependencies. Branches are keyed by both the agent-provided branch name and
 * the effective (prefixed/salted) branch name that was actually pushed.
 * @returns {{record: (entry: StackEntry, options: {agentBranch?: string|null, baseBranch: string}) => void, get: (branch: string, repo: string) => StackEntry|null, parents: Map<string, string>, resolveDependencies: (branches: string[], repo: string) => number[]}}
 */
function createStackTracker() {
  /** @type {Map<string, StackEntry>} */
  const branches = new Map();
  /** @type {Map<string, string>} Effective branch name -> base branch it was created from */
  const parents = new Map();

  return {
    parents,
    /**
     * Record a created pull request so later messages in the same run can stack on top of it.
     * @param {StackEntry} entry
     * @param {{agentBranch?: string|null, baseBranch: string}} options
     */
    record(entry, options) {
      branches.set(entry.branch, entry);
      if (options.agentBranch) {
        branches.set(options.agentBranch, entry);
      }
      parents.set(entry.branch, options.baseBranch);
    },
    /**
     * Look up a pull request created earlier in this run by branch name, scoped to a repository.
     * @param {string} branch
     * @param {string} repo
     * @returns {StackEntry|null}
     */
    get(branch, repo) {
      const entry = branches.get(branch);
      return entry && entry.repo === repo ? entry : null;
    },
    /**
     * Resolve dependency branch names to pull request numbers created earlier in this run.
     * @param {string[]} dependencyBranches
     * @param {string} repo
     * @returns {number[]}
     */
    resolveDependencies(dependencyBranches, repo) {
      /** @type {number[]} */
      const numbers = [];
      for (const branch of dependencyBranches) {
        const entry = branches.get(branch);
        if (entry && entry.repo === repo && !numbers.includes(entry.number)) {
          numbers.push(entry.number);
        }
      }
      return numbers;
    },
  };
}

module.exports = {
  isStackedEnabled,
  parseStackMetadata,
  hasCircularStackDependency,
  buildStackMetadataLines,
  stackedDisabledError,
  missingStackBaseError,
  circularStackError,
  verifyStackBaseBranchExists,
  createStackTracker,
};
