// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");

async function main() {
  const issueNumber = Number.parseInt(process.env.GH_AW_STEERING_ISSUE_NUMBER || "", 10);
  if (!Number.isFinite(issueNumber) || issueNumber <= 0) {
    core.info("No steering issue to complete");
    return;
  }

  const failureIssueNumber = Number.parseInt(process.env.GH_AW_FAILURE_ISSUE_NUMBER || "", 10);
  if (Number.isFinite(failureIssueNumber) && failureIssueNumber === issueNumber) {
    core.info(`Keeping steering issue #${issueNumber} open as the agent failure issue`);
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
  if (results.includes("failure") || results.includes("cancelled")) {
    core.info(`Keeping steering issue #${issueNumber} open because the workflow did not complete successfully`);
    return;
  }

  const pullRequestNumber = Number.parseInt(process.env.GH_AW_CREATED_PR_NUMBER || "", 10);
  const pullRequestUrl = process.env.GH_AW_CREATED_PR_URL || "";
  const body =
    Number.isFinite(pullRequestNumber) && pullRequestNumber > 0 && pullRequestUrl ? `The workflow completed successfully and created [pull request #${pullRequestNumber}](${pullRequestUrl}).` : "The workflow completed successfully.";
  try {
    await github.rest.issues.createComment({
      owner: context.repo.owner,
      repo: context.repo.repo,
      issue_number: issueNumber,
      body,
    });
    await github.rest.issues.update({
      owner: context.repo.owner,
      repo: context.repo.repo,
      issue_number: issueNumber,
      state: "closed",
      state_reason: "completed",
    });
    core.info(`Closed steering issue #${issueNumber}`);
  } catch (error) {
    core.warning(`Failed to complete steering issue #${issueNumber}: ${getErrorMessage(error)}`);
  }
}

module.exports = { main };
