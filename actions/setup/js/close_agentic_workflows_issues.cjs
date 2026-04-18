// @ts-check
/// <reference types="@actions/github-script" />

const { resolveExecutionOwnerRepo } = require("./repo_helpers.cjs");

const TARGET_LABEL = "agentic-workflows";
const NO_REPRO_MESSAGE = `Closing as no repro.

If this is still reproducible, please open a new issue with clear reproduction steps.`;

/**
 * Close all open issues with the "agentic-workflows" label.
 * @returns {Promise<void>}
 */
async function main() {
  const { owner, repo } = resolveExecutionOwnerRepo();
  core.info(`Operating on repository: ${owner}/${repo}`);
  core.info(`Searching for open issues labeled "${TARGET_LABEL}"`);

  /** @type {Array<any>} */
  const issues = await github.paginate(github.rest.issues.listForRepo, {
    owner,
    repo,
    labels: TARGET_LABEL,
    state: "open",
    per_page: 100,
  });

  const targetIssues = issues.filter(issue => !issue.pull_request);
  core.info(`Found ${targetIssues.length} issue(s) to close`);

  if (targetIssues.length === 0) {
    return;
  }

  for (const issue of targetIssues) {
    core.info(`Closing issue #${issue.number}: ${issue.title}`);

    await github.rest.issues.createComment({
      owner,
      repo,
      issue_number: issue.number,
      body: NO_REPRO_MESSAGE,
    });

    await github.rest.issues.update({
      owner,
      repo,
      issue_number: issue.number,
      state: "closed",
      state_reason: "not_planned",
    });
  }
}

module.exports = { main };
