// @ts-check

/**
 * Detect if a pull request is from a fork repository.
 *
 * A "fork PR" means the head and base are in *different* repositories
 * (cross-repo PR). Detection uses two signals:
 * 1. Handle deleted fork case (head.repo is null)
 * 2. Compare repository full names — different names mean cross-repo
 *
 * NOTE: We intentionally do NOT check head.repo.fork. That flag indicates
 * whether the repository *itself* is a fork of another repo, not whether
 * the PR is cross-repo. A same-repo PR in a forked repository (common in
 * OSS) would have fork=true but is NOT a cross-repo fork PR. Using that
 * flag caused false positives that forced `gh pr checkout` instead of fast
 * `git fetch`, which then failed due to stale GH_HOST values. See #24208.
 *
 * @param {object} pullRequest - The pull request object from GitHub context
 * @returns {{isFork: boolean, reason: string}} Fork detection result with reason
 */
function detectForkPR(pullRequest) {
  if (!pullRequest.head?.repo) {
    // Head repo is null - likely a deleted fork
    return { isFork: true, reason: "head repository deleted (was likely a fork)" };
  }

  if (pullRequest.head.repo.full_name !== pullRequest.base?.repo?.full_name) {
    // Different repository names — this is a cross-repo (fork) PR
    return { isFork: true, reason: "different repository names" };
  }

  return { isFork: false, reason: "same repository" };
}

/**
 * Extract and validate pull request number from a message or GitHub context.
 *
 * Tries to get PR number from:
 * 1. The message's pull_request_number field (if provided)
 * 2. The GitHub context payload (if in a PR context)
 *
 * @param {object|undefined} messageItem - The message object that might contain pull_request_number
 * @param {object} context - The GitHub context object with payload information
 * @returns {{prNumber: number|null, error: string|null}} Result with PR number or error message
 */
function getPullRequestNumber(messageItem, context) {
  // Try to get from message first
  if (messageItem?.pull_request_number !== undefined) {
    const prNumber = parseInt(String(messageItem.pull_request_number), 10);
    if (isNaN(prNumber)) {
      return {
        prNumber: null,
        error: `Invalid pull_request_number: ${messageItem.pull_request_number}`,
      };
    }
    return { prNumber, error: null };
  }

  // Fall back to context
  const contextPR = context.payload?.pull_request?.number;
  if (!contextPR) {
    return {
      prNumber: null,
      error: "No pull_request_number provided and not in pull request context",
    };
  }

  return { prNumber: contextPR, error: null };
}

/**
 * Resolves the pull request repository ID and effective base branch.
 * Fetches `id` and `defaultBranchRef.name` from the GitHub API.
 * The effective base branch is the explicitly configured branch (if any),
 * falling back to the repository's actual default branch.
 *
 * @param {import("@actions/github-script").AsyncFunctionArguments["github"]} github
 * @param {string} owner
 * @param {string} repo
 * @param {string|null|undefined} configuredBaseBranch - explicitly configured base branch (may be null or undefined)
 * @returns {Promise<{repoId: string, effectiveBaseBranch: string|null, resolvedDefaultBranch: string|null}>}
 */
async function resolvePullRequestRepo(github, owner, repo, configuredBaseBranch) {
  const query = `
    query($owner: String!, $name: String!) {
      repository(owner: $owner, name: $name) {
        id
        defaultBranchRef { name }
      }
    }
  `;
  const response = await github.graphql(query, { owner, name: repo });
  const repoId = response.repository.id;
  const resolvedDefaultBranch = response.repository.defaultBranchRef?.name ?? null;
  const effectiveBaseBranch = configuredBaseBranch || resolvedDefaultBranch;
  return { repoId, effectiveBaseBranch, resolvedDefaultBranch };
}

/**
 * Builds a branch instruction string to prepend to custom instructions.
 * Tells the agent which branch to create its work branch from, with an
 * optional NOT clause when the effective branch differs from the repo default.
 *
 * @param {string} effectiveBaseBranch - the branch the agent should branch from
 * @param {string|null} resolvedDefaultBranch - the repo's actual default branch (used in NOT clause)
 * @returns {string}
 */
function buildBranchInstruction(effectiveBaseBranch, resolvedDefaultBranch) {
  const notClause = resolvedDefaultBranch && resolvedDefaultBranch !== effectiveBaseBranch ? `, NOT from '${resolvedDefaultBranch}'` : "";
  return `IMPORTANT: Create your branch from the '${effectiveBaseBranch}' branch${notClause}.`;
}

module.exports = { detectForkPR, getPullRequestNumber, resolvePullRequestRepo, buildBranchInstruction };
