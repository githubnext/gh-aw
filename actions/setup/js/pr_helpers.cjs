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

module.exports = { detectForkPR, getPullRequestNumber };
