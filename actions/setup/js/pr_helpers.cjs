// @ts-check

/**
 * Detect if a pull request is from a fork repository.
 *
 * Uses multiple signals for robust detection:
 * 1. Check if head.repo.fork is explicitly true (GitHub's fork flag)
 * 2. Compare repository full names if both repos exist
 * 3. Handle deleted fork case (head.repo is null)
 *
 * @param {object} pullRequest - The pull request object from GitHub context
 * @returns {{isFork: boolean, reason: string}} Fork detection result with reason
 */
function detectForkPR(pullRequest) {
  let isFork = false;
  let reason = "same repository";

  if (!pullRequest.head?.repo) {
    // Head repo is null - likely a deleted fork
    isFork = true;
    reason = "head repository deleted (was likely a fork)";
  } else if (pullRequest.head.repo.fork === true) {
    // GitHub's explicit fork flag
    isFork = true;
    reason = "head.repo.fork flag is true";
  } else if (pullRequest.head.repo.full_name !== pullRequest.base?.repo?.full_name) {
    // Different repository names
    isFork = true;
    reason = "different repository names";
  }

  return { isFork, reason };
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
 * @param {string|undefined} configuredBaseBranch - explicitly configured base branch (may be undefined)
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
