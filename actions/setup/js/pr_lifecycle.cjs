// @ts-check

/**
 * Only work triggered while a PR was open becomes obsolete on closure.
 * Closed-event workflows (including merge follow-ups) must keep running.
 *
 * @param {any} githubClient
 * @param {any} runtimeContext
 * @returns {Promise<string | null>}
 */
async function getStalePullRequestReason(githubClient, runtimeContext) {
  if (process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true") return null;
  const payload = runtimeContext?.payload;
  const pr = payload?.pull_request || (payload?.issue?.pull_request ? payload.issue : null);
  if (!pr || pr.state !== "open" || !Number.isSafeInteger(pr.number) || pr.number <= 0) return null;

  const { owner, repo } = runtimeContext.repo;
  const { data } = await githubClient.rest.pulls.get({ owner, repo, pull_number: pr.number });
  if (data.state === "open") return null;
  if (data.state !== "closed") throw new Error(`Cannot determine current state of pull request #${pr.number}`);

  return `Pull request #${pr.number} closed after this workflow was triggered; obsolete safe outputs were skipped`;
}

class StalePullRequestError extends Error {}

module.exports = { getStalePullRequestReason, StalePullRequestError };
