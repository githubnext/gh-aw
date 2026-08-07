// @ts-check
// <reference types="@actions/github-script" />

const { executeExpiredEntityCleanup } = require("./expired_entity_main_flow.cjs");
const { addIssueThreadComment, closeIssue, createExpiredEntityHandler } = require("./expired_entity_handler_factory.cjs");
const { getWorkflowMetadata } = require("./workflow_metadata_helpers.cjs");
const { resolveExecutionOwnerRepo } = require("./repo_helpers.cjs");

async function main() {
  const { owner, repo } = resolveExecutionOwnerRepo();
  core.info(`Operating on repository: ${owner}/${repo}`);

  // Get workflow metadata for footer
  const { workflowName, workflowId, runUrl } = getWorkflowMetadata(owner, repo);

  await executeExpiredEntityCleanup(github, owner, repo, {
    entityType: "issues",
    graphqlField: "issues",
    resultKey: "issues",
    entityLabel: "Issue",
    summaryHeading: "Expired Issues Cleanup",
    processEntity: createExpiredEntityHandler({
      github,
      owner,
      repo,
      workflowName,
      workflowId,
      runUrl,
      entityNoun: "issue",
      entityLabel: "Issue",
      addComment: (issue, message) => addIssueThreadComment(github, owner, repo, issue.number, message),
      closeEntity: issue => closeIssue(github, owner, repo, issue.number),
    }),
  });
}

module.exports = { main };
