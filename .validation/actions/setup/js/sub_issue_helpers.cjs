// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");

// Maximum number of sub-issues per parent issue
const MAX_SUB_ISSUES = 64;

/**
 * Gets the sub-issue count for a parent issue using GraphQL
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} issueNumber - Issue number
 * @returns {Promise<number|null>} - Sub-issue count or null if query failed
 */
async function getSubIssueCount(owner, repo, issueNumber) {
  try {
    const subIssueQuery = `
      query($owner: String!, $repo: String!, $issueNumber: Int!) {
        repository(owner: $owner, name: $repo) {
          issue(number: $issueNumber) {
            subIssues(first: ${MAX_SUB_ISSUES + 1}) {
              totalCount
            }
          }
        }
      }
    `;

    const result = await github.graphql(subIssueQuery, {
      owner,
      repo,
      issueNumber,
    });

    return result?.repository?.issue?.subIssues?.totalCount || 0;
  } catch (error) {
    core.warning(`Could not check sub-issue count for #${issueNumber}: ${getErrorMessage(error)}`);
    return null;
  }
}

module.exports = {
  MAX_SUB_ISSUES,
  getSubIssueCount,
};
