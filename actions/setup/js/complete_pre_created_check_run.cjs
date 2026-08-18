// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Closes the pre-created pull request and deletes its branch when the run produced no
 * changes, so runs that end without a `create_pull_request` output (or fail before it)
 * do not leave an empty placeholder pull request behind.
 * @returns {Promise<void>}
 */
async function discardUnusedPullRequest() {
  const pullNumber = Number.parseInt(process.env.GH_AW_PRE_CREATED_PULL_REQUEST_NUMBER || "", 10);
  const branch = process.env.GH_AW_PRE_CREATED_PULL_REQUEST_BRANCH || "";
  if (!Number.isFinite(pullNumber) || pullNumber <= 0 || !branch) {
    return;
  }

  try {
    const { data: pullRequest } = await github.rest.pulls.get({
      owner: context.repo.owner,
      repo: context.repo.repo,
      pull_number: pullNumber,
    });
    if (pullRequest.state !== "open" || pullRequest.head.ref !== branch || (pullRequest.changed_files ?? 0) > 0) {
      return;
    }

    core.info(`Closing pre-created pull request #${pullNumber} because the run produced no changes`);
    await github.rest.pulls.update({
      owner: context.repo.owner,
      repo: context.repo.repo,
      pull_number: pullNumber,
      state: "closed",
    });
    await github.rest.git.deleteRef({
      owner: context.repo.owner,
      repo: context.repo.repo,
      ref: `heads/${branch}`,
    });
  } catch (error) {
    core.warning(`Failed to discard unused pre-created pull request #${pullNumber}: ${getErrorMessage(error)}`);
  }
}

async function main() {
  const checkRunId = Number.parseInt(process.env.GH_AW_PRE_CREATED_CHECK_RUN_ID || "", 10);
  if (!Number.isFinite(checkRunId) || checkRunId <= 0) {
    core.info("No pre-created pull request check run to complete");
    return;
  }

  let needs;
  try {
    needs = JSON.parse(process.env.GH_AW_NEEDS || "{}");
  } catch (error) {
    core.warning(`Unable to parse downstream job results: ${error instanceof Error ? error.message : String(error)}`);
    needs = {};
  }
  const results = Object.values(needs)
    .map(value => value?.result)
    .filter(Boolean);
  const conclusion = results.includes("failure") ? "failure" : results.includes("cancelled") ? "cancelled" : "success";
  const runUrl = `${context.serverUrl}/${context.repo.owner}/${context.repo.repo}/actions/runs/${context.runId}`;

  await github.rest.checks.update({
    owner: context.repo.owner,
    repo: context.repo.repo,
    check_run_id: checkRunId,
    status: "completed",
    conclusion,
    completed_at: new Date().toISOString(),
    output: {
      title: `${context.workflow} ${conclusion}`,
      summary: `The [workflow run](${runUrl}) completed with conclusion: ${conclusion}.`,
    },
  });

  await discardUnusedPullRequest();
}

module.exports = { main };
