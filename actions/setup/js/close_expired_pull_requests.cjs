// @ts-check
// <reference types="@actions/github-script" />

const { executeExpiredEntityCleanup } = require("./expired_entity_main_flow.cjs");
const { addIssueThreadComment, closePullRequest, createExpiredEntityHandler } = require("./expired_entity_handler_factory.cjs");
const { getWorkflowMetadata } = require("./workflow_metadata_helpers.cjs");
const { resolveExecutionOwnerRepo } = require("./repo_helpers.cjs");

async function main() {
  const { owner, repo } = resolveExecutionOwnerRepo();
  core.info(`Operating on repository: ${owner}/${repo}`);

  // Get workflow metadata for footer
  const { workflowName, workflowId, runUrl } = getWorkflowMetadata(owner, repo);

  await executeExpiredEntityCleanup(github, owner, repo, {
    entityType: "pull requests",
    graphqlField: "pullRequests",
    resultKey: "pullRequests",
    entityLabel: "Pull Request",
    summaryHeading: "Expired Pull Requests Cleanup",
    processEntity: createExpiredEntityHandler({
      github,
      owner,
      repo,
      workflowName,
      workflowId,
      runUrl,
      entityNoun: "pull request",
      entityLabel: "Pull Request",
      addComment: (pr, message) => addIssueThreadComment(github, owner, repo, pr.number, message),
      closeEntity: pr => closePullRequest(github, owner, repo, pr.number),
    }),
  });
}

module.exports = { main };
