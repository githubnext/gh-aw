// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Issue Management Helper Functions
 *
 * This module provides reusable functions for managing GitHub issues,
 * including finding/creating parent issues and linking sub-issues.
 * These functions are used by various workflow failure handlers.
 */

/**
 * Search for or create the parent issue for all agentic workflow failures
 * @returns {Promise<{number: number, node_id: string}>} Parent issue number and node ID
 */
async function ensureParentIssue() {
  const { owner, repo } = context.repo;
  const parentTitle = "[agentics] Agentic Workflow Issues";
  const parentLabel = "agentic-workflows";

  core.info(`Searching for parent issue: "${parentTitle}"`);

  // Search for existing parent issue
  const searchQuery = `repo:${owner}/${repo} is:issue is:open label:${parentLabel} in:title "${parentTitle}"`;

  try {
    const searchResult = await github.rest.search.issuesAndPullRequests({
      q: searchQuery,
      per_page: 1,
    });

    if (searchResult.data.total_count > 0) {
      const existingIssue = searchResult.data.items[0];
      core.info(`Found existing parent issue #${existingIssue.number}: ${existingIssue.html_url}`);
      return {
        number: existingIssue.number,
        node_id: existingIssue.node_id,
      };
    }
  } catch (error) {
    core.warning(`Error searching for parent issue: ${getErrorMessage(error)}`);
  }

  // Create parent issue if it doesn't exist
  core.info("No parent issue found, creating one");

  const parentBodyContent = `# Agentic Workflow Failures

This issue tracks all failures from agentic workflows in this repository. Each failed workflow run creates a sub-issue linked here for organization and easy filtering.

### Purpose

This parent issue helps you:
- View all workflow failures in one place by checking the sub-issues below
- Filter out failure issues from your main issue list using \`no:parent-issue\`
- Track the health of your agentic workflows over time

### Sub-Issues

All individual workflow failure issues are linked as sub-issues below. Click on any sub-issue to see details about a specific failure.

### Troubleshooting Failed Workflows

#### Using agentic-workflows Agent (Recommended)

**Agent:** \`agentic-workflows\`  
**Purpose:** Debug and fix workflow failures

**Instructions:**

1. Invoke the agent: Type \`/agent\` in GitHub Copilot Chat and select **agentic-workflows**
2. Provide context: Tell the agent to **debug** the workflow failure
3. Supply the workflow run URL for analysis
4. The agent will:
   - Analyze failure logs
   - Identify root causes
   - Propose specific fixes
   - Validate solutions

#### Using gh aw CLI

You can also debug failures using the \`gh aw\` CLI:

\`\`\`bash
# Download and analyze workflow logs
gh aw logs <workflow-run-url>

# Audit a specific workflow run
gh aw audit <run-id>
\`\`\`

#### Manual Investigation

1. Click on a sub-issue to see the failed workflow details
2. Follow the workflow run link in the issue
3. Review the agent job logs for error messages
4. Check the workflow configuration in your repository

### Resources

- [GitHub Agentic Workflows Documentation](https://github.com/githubnext/gh-aw)
- [Troubleshooting Guide](https://github.com/githubnext/gh-aw/blob/main/docs/troubleshooting.md)

---

> This issue is automatically managed by GitHub Agentic Workflows. Do not close this issue manually.`;

  // Add expiration marker (7 days from now)
  const expirationDate = new Date();
  expirationDate.setDate(expirationDate.getDate() + 7);
  const parentBody = `${parentBodyContent}\n\n<!-- gh-aw-expires: ${expirationDate.toISOString()} -->`;

  try {
    const newIssue = await github.rest.issues.create({
      owner,
      repo,
      title: parentTitle,
      body: parentBody,
      labels: [parentLabel],
    });

    core.info(`✓ Created parent issue #${newIssue.data.number}: ${newIssue.data.html_url}`);
    return {
      number: newIssue.data.number,
      node_id: newIssue.data.node_id,
    };
  } catch (error) {
    core.error(`Failed to create parent issue: ${getErrorMessage(error)}`);
    throw error;
  }
}

/**
 * Link an issue as a sub-issue to a parent issue
 * @param {string} parentNodeId - GraphQL node ID of the parent issue
 * @param {string} subIssueNodeId - GraphQL node ID of the sub-issue
 * @param {number} parentNumber - Parent issue number (for logging)
 * @param {number} subIssueNumber - Sub-issue number (for logging)
 */
async function linkSubIssue(parentNodeId, subIssueNodeId, parentNumber, subIssueNumber) {
  core.info(`Linking issue #${subIssueNumber} as sub-issue of #${parentNumber}`);

  try {
    // Use GraphQL to link the sub-issue
    await github.graphql(
      `mutation($parentId: ID!, $subIssueId: ID!) {
        addSubIssue(input: {issueId: $parentId, subIssueId: $subIssueId}) {
          issue {
            id
            number
          }
          subIssue {
            id
            number
          }
        }
      }`,
      {
        parentId: parentNodeId,
        subIssueId: subIssueNodeId,
      }
    );

    core.info(`✓ Successfully linked #${subIssueNumber} as sub-issue of #${parentNumber}`);
  } catch (error) {
    const errorMessage = getErrorMessage(error);
    if (errorMessage.includes("Field 'addSubIssue' doesn't exist") || errorMessage.includes("not yet available")) {
      core.warning(`Sub-issue API not available. Issue #${subIssueNumber} created but not linked to parent.`);
    } else {
      core.warning(`Failed to link sub-issue: ${errorMessage}`);
    }
  }
}

/**
 * Search for an existing issue by title and label
 * @param {string} issueTitle - Title of the issue to search for
 * @param {string} label - Label to filter by (e.g., "agentic-workflows")
 * @returns {Promise<{number: number, html_url: string, node_id: string} | null>} Issue info or null if not found
 */
async function findExistingIssue(issueTitle, label = "agentic-workflows") {
  const { owner, repo } = context.repo;
  const searchQuery = `repo:${owner}/${repo} is:issue is:open label:${label} in:title "${issueTitle}"`;

  try {
    const searchResult = await github.rest.search.issuesAndPullRequests({
      q: searchQuery,
      per_page: 1,
    });

    if (searchResult.data.total_count > 0) {
      const existingIssue = searchResult.data.items[0];
      core.info(`Found existing issue #${existingIssue.number}: ${existingIssue.html_url}`);
      return {
        number: existingIssue.number,
        html_url: existingIssue.html_url,
        node_id: existingIssue.node_id,
      };
    }

    core.info(`No existing issue found with title: "${issueTitle}"`);
    return null;
  } catch (error) {
    core.warning(`Error searching for issue: ${getErrorMessage(error)}`);
    return null;
  }
}

/**
 * Create a comment on an existing issue
 * @param {number} issueNumber - Issue number to comment on
 * @param {string} commentBody - Comment body (markdown)
 * @returns {Promise<void>}
 */
async function addIssueComment(issueNumber, commentBody) {
  const { owner, repo } = context.repo;

  try {
    await github.rest.issues.createComment({
      owner,
      repo,
      issue_number: issueNumber,
      body: commentBody,
    });

    core.info(`✓ Added comment to issue #${issueNumber}`);
  } catch (error) {
    core.warning(`Failed to add comment to issue #${issueNumber}: ${getErrorMessage(error)}`);
    throw error;
  }
}

/**
 * Create a new issue
 * @param {string} title - Issue title
 * @param {string} body - Issue body (markdown)
 * @param {Array<string>} labels - Issue labels
 * @returns {Promise<{number: number, html_url: string, node_id: string}>} Created issue info
 */
async function createIssue(title, body, labels = ["agentic-workflows"]) {
  const { owner, repo } = context.repo;

  try {
    const newIssue = await github.rest.issues.create({
      owner,
      repo,
      title,
      body,
      labels,
    });

    core.info(`✓ Created new issue #${newIssue.data.number}: ${newIssue.data.html_url}`);
    return {
      number: newIssue.data.number,
      html_url: newIssue.data.html_url,
      node_id: newIssue.data.node_id,
    };
  } catch (error) {
    core.error(`Failed to create issue: ${getErrorMessage(error)}`);
    throw error;
  }
}

module.exports = {
  ensureParentIssue,
  linkSubIssue,
  findExistingIssue,
  addIssueComment,
  createIssue,
};
