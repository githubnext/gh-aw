// @ts-check
/// <reference types="@actions/github-script" />

async function main() {
  const expectedBranch = process.env.GH_AW_EXPECTED_PRE_CREATED_PULL_REQUEST_BRANCH || "";
  const branch = process.env.GH_AW_PRE_CREATED_PULL_REQUEST_BRANCH || "";
  if (branch !== expectedBranch) {
    throw new Error(`Pre-created pull request branch did not match expected workflow branch: ${branch}`);
  }

  const pullNumber = Number.parseInt(process.env.GH_AW_PRE_CREATED_PULL_REQUEST_NUMBER || "", 10);
  if (!Number.isSafeInteger(pullNumber) || pullNumber <= 0) {
    throw new Error("Pre-created pull request number is invalid");
  }

  const { data: pullRequest } = await github.rest.pulls.get({
    owner: context.repo.owner,
    repo: context.repo.repo,
    pull_number: pullNumber,
  });
  const expectedRepo = `${context.repo.owner}/${context.repo.repo}`.toLowerCase();
  if (pullRequest.head.ref !== expectedBranch || pullRequest.head.repo?.full_name?.toLowerCase() !== expectedRepo || pullRequest.base.repo?.full_name?.toLowerCase() !== expectedRepo) {
    throw new Error("Pre-created pull request does not target the expected trusted repository branch");
  }

  core.setOutput("branch", expectedBranch);
}

module.exports = { main };
