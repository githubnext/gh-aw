// @ts-check
// <reference types="@actions/github-script" />

const { executeExpiredEntityCleanup } = require("./expired_entity_main_flow.cjs");
const { generateExpiredEntityFooter } = require("./generate_footer.cjs");
const { sanitizeContent } = require("./sanitize_content.cjs");
const { getWorkflowMetadata } = require("./workflow_metadata_helpers.cjs");

/**
 * Add comment to a GitHub Issue using REST API
 * @param {any} github - GitHub REST instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} issueNumber - Issue number
 * @param {string} message - Comment body
 * @returns {Promise<any>} Comment details
 */
async function addIssueComment(github, owner, repo, issueNumber, message) {
  const result = await github.rest.issues.createComment({
    owner: owner,
    repo: repo,
    issue_number: issueNumber,
    body: sanitizeContent(message),
  });

  return result.data;
}

/**
 * Close a GitHub Issue using REST API
 * @param {any} github - GitHub REST instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} issueNumber - Issue number
 * @returns {Promise<any>} Issue details
 */
async function closeIssue(github, owner, repo, issueNumber) {
  const result = await github.rest.issues.update({
    owner: owner,
    repo: repo,
    issue_number: issueNumber,
    state: "closed",
    state_reason: "not_planned",
  });

  return result.data;
}

async function main() {
  const owner = context.repo.owner;
  const repo = context.repo.repo;

  // Get workflow metadata for footer
  const { workflowName, workflowId, runUrl } = getWorkflowMetadata(owner, repo);

  await executeExpiredEntityCleanup(github, owner, repo, {
    entityType: "issues",
    graphqlField: "issues",
    resultKey: "issues",
    entityLabel: "Issue",
    summaryHeading: "Expired Issues Cleanup",
    processEntity: async issue => {
      const closingMessage = `This issue was automatically closed because it expired on ${issue.expirationDate.toISOString()}.` + generateExpiredEntityFooter(workflowName, runUrl, workflowId);

      await addIssueComment(github, owner, repo, issue.number, closingMessage);
      core.info(`  ✓ Comment added successfully`);

      await closeIssue(github, owner, repo, issue.number);
      core.info(`  ✓ Issue closed successfully`);

      return {
        status: "closed",
        record: {
          number: issue.number,
          url: issue.url,
          title: issue.title,
        },
      };
    },
  });
}

module.exports = { main };
