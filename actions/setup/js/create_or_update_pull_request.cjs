// @ts-check

const { withRetry, RATE_LIMIT_RETRY_CONFIG } = require("./error_recovery.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");

/** @type {string} GraphQL mutation to mark a pull request as ready for review */
const MARK_PR_READY_MUTATION = /* GraphQL */ `
  mutation ($pullRequestId: ID!) {
    markPullRequestReadyForReview(input: { pullRequestId: $pullRequestId }) {
      pullRequest {
        isDraft
      }
    }
  }
`;

/** @type {string} GraphQL mutation to convert a pull request back to a draft */
const CONVERT_PR_TO_DRAFT_MUTATION = /* GraphQL */ `
  mutation ($pullRequestId: ID!) {
    convertPullRequestToDraft(input: { pullRequestId: $pullRequestId }) {
      pullRequest {
        isDraft
      }
    }
  }
`;

/**
 * Align the draft state of a pre-created pull request with the configured policy.
 * The REST `pulls.update` endpoint ignores the `draft` field, so the GraphQL
 * mutations must be used to transition between draft and ready-for-review.
 * @param {any} githubClient - Authenticated Octokit client
 * @param {string} pullRequestNodeId - The PR's GraphQL node ID
 * @param {boolean} isDraft - Current draft state of the pull request
 * @param {boolean} draft - Desired draft state
 * @param {number} pullRequestNumber - Pull request number (for logging)
 * @returns {Promise<void>}
 */
async function alignPullRequestDraftState(githubClient, pullRequestNodeId, isDraft, draft, pullRequestNumber) {
  const wantsDraft = draft !== false;
  if (wantsDraft === isDraft || !pullRequestNodeId) {
    return;
  }
  const mutation = wantsDraft ? CONVERT_PR_TO_DRAFT_MUTATION : MARK_PR_READY_MUTATION;
  try {
    await githubClient.graphql(mutation, { pullRequestId: pullRequestNodeId });
    core.info(`${wantsDraft ? "Converted" : "Marked"} pull request #${pullRequestNumber} ${wantsDraft ? "back to draft" : "as ready for review"}`);
  } catch (error) {
    core.warning(`Failed to update draft state of pull request #${pullRequestNumber}: ${getErrorMessage(error)}`);
  }
}

async function createOrUpdatePullRequest(options) {
  const { githubClient, repoParts, title, body, branchName, baseBranch, draft, preCreatedPullRequestNumber, preCreatedBranch } = options;
  if (!(preCreatedPullRequestNumber > 0)) {
    return withRetry(
      () =>
        githubClient.rest.pulls.create({
          owner: repoParts.owner,
          repo: repoParts.repo,
          title,
          body,
          head: branchName,
          base: baseBranch,
          draft,
        }),
      RATE_LIMIT_RETRY_CONFIG,
      `create pull request in ${repoParts.owner}/${repoParts.repo}`
    );
  }

  // The state check is intentionally performed outside the retry wrapper: a closed or
  // relocated pull request is not a transient failure, so it must fail immediately
  // instead of consuming the retry budget.
  const { data: existingPullRequest } = await withRetry(
    () =>
      githubClient.rest.pulls.get({
        owner: repoParts.owner,
        repo: repoParts.repo,
        pull_number: preCreatedPullRequestNumber,
      }),
    RATE_LIMIT_RETRY_CONFIG,
    `read pre-created pull request #${preCreatedPullRequestNumber}`
  );
  if (existingPullRequest.state !== "open" || existingPullRequest.head.ref !== preCreatedBranch) {
    throw new Error(`Pre-created pull request #${preCreatedPullRequestNumber} is not open on branch ${preCreatedBranch}`);
  }

  const updated = await withRetry(
    () =>
      githubClient.rest.pulls.update({
        owner: repoParts.owner,
        repo: repoParts.repo,
        pull_number: preCreatedPullRequestNumber,
        title,
        body,
        base: baseBranch,
      }),
    RATE_LIMIT_RETRY_CONFIG,
    `update pre-created pull request #${preCreatedPullRequestNumber}`
  );

  // Pre-created pull requests always start as drafts, so the configured draft policy is
  // applied after the content update.
  await alignPullRequestDraftState(githubClient, existingPullRequest.node_id, existingPullRequest.draft === true, draft, preCreatedPullRequestNumber);

  return updated;
}

module.exports = { createOrUpdatePullRequest };
