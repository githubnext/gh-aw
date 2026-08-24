// @ts-check
/// <reference types="@actions/github-script" />
// @safe-outputs-exempt SEC-004

const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Closes the pre-created pull request and deletes its branch when the safe-outputs
 * job did not consume it via create_pull_request, so runs that end without using the
 * pre-created PR do not leave an empty placeholder pull request behind. When the run
 * produced a no-op message, the same comment posted on the no-op runs issue is added to
 * the pull request before closing it.
 * @returns {Promise<void>}
 */
async function discardUnusedPullRequest() {
  const pullNumber = Number.parseInt(process.env.GH_AW_PRE_CREATED_PULL_REQUEST_NUMBER || "", 10);
  const branch = process.env.GH_AW_PRE_CREATED_PULL_REQUEST_BRANCH || "";
  const consumedPullNumber = Number.parseInt(process.env.GH_AW_SAFE_OUTPUT_CREATED_PR_NUMBER || "", 10);
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
    if (Number.isFinite(consumedPullNumber) && consumedPullNumber > 0 && consumedPullNumber === pullNumber) {
      core.info(`Keeping pre-created pull request #${pullNumber} because create-pull-request consumed it`);
      return;
    }

    core.info(`Closing pre-created pull request #${pullNumber} because create-pull-request did not consume it`);
    const noopComment = process.env.GH_AW_NOOP_COMMENT_BODY || "";
    if (noopComment.trim()) {
      try {
        await github.rest.issues.createComment({
          owner: context.repo.owner,
          repo: context.repo.repo,
          issue_number: pullNumber,
          body: noopComment,
        });
      } catch (error) {
        core.warning(`Failed to comment on pre-created pull request #${pullNumber}: ${getErrorMessage(error)}`);
      }
    }
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
    core.warning(`Unable to parse downstream job results: ${getErrorMessage(error)}`);
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
