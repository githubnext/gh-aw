// @ts-check
/// <reference types="@actions/github-script" />

const { processCloseEntityItems, ISSUE_CONFIG } = require("./close_entity_helpers.cjs");
const { generateFooter } = require("./generate_footer.cjs");
const { getTrackerID } = require("./get_tracker_id.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Get issue details using REST API
 * @param {any} github - GitHub REST API instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} issueNumber - Issue number
 * @returns {Promise<{number: number, title: string, labels: Array<{name: string}>, html_url: string, state: string}>} Issue details
 */
async function getIssueDetails(github, owner, repo, issueNumber) {
  const { data: issue } = await github.rest.issues.get({
    owner,
    repo,
    issue_number: issueNumber,
  });

  if (!issue) {
    throw new Error(`Issue #${issueNumber} not found in ${owner}/${repo}`);
  }

  return issue;
}

/**
 * Add comment to a GitHub Issue using REST API
 * @param {any} github - GitHub REST API instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} issueNumber - Issue number
 * @param {string} message - Comment body
 * @returns {Promise<{id: number, html_url: string}>} Comment details
 */
async function addIssueComment(github, owner, repo, issueNumber, message) {
  const { data: comment } = await github.rest.issues.createComment({
    owner,
    repo,
    issue_number: issueNumber,
    body: message,
  });

  return comment;
}

/**
 * Close a GitHub Issue using REST API
 * @param {any} github - GitHub REST API instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} issueNumber - Issue number
 * @returns {Promise<{number: number, html_url: string, title: string}>} Issue details
 */
async function closeIssue(github, owner, repo, issueNumber) {
  const { data: issue } = await github.rest.issues.update({
    owner,
    repo,
    issue_number: issueNumber,
    state: "closed",
  });

  return issue;
}

/**
 * Factory function for creating close issue handler
 * @param {Object} [config] - Handler configuration
 * @param {string[]} [config.requiredLabels] - Required labels (any match)
 * @param {string} [config.requiredTitlePrefix] - Required title prefix
 * @param {string} [config.target] - Target configuration ("triggering", "*", or explicit number)
 * @param {string} [config.workflowName] - Workflow name for footer
 * @param {string} [config.workflowSource] - Workflow source for footer
 * @param {string} [config.workflowSourceURL] - Workflow source URL for footer
 * @param {boolean} [config._legacyMode] - Internal flag for backward compatibility
 * @returns {Promise<Function|void|any>} Handler function that processes individual messages, or result in legacy mode
 */
async function main(config = {}) {
  // Legacy mode: maintain backward compatibility with old calling pattern
  if (!config || Object.keys(config).length === 0 || config._legacyMode === true) {
    return processCloseEntityItems(ISSUE_CONFIG, {
      getDetails: getIssueDetails,
      addComment: addIssueComment,
      closeEntity: closeIssue,
    });
  }

  const {
    requiredLabels = [],
    requiredTitlePrefix = "",
    target = "triggering",
    workflowName = "Workflow",
    workflowSource = "",
    workflowSourceURL = "",
  } = config;

  /**
   * Process a single close_issue message
   * @param {Object} outputItem - The safe output item
   * @param {Object} resolvedTemporaryIds - Map of temporary IDs to actual IDs
   * @returns {Promise<{repo: string, number: number}|undefined>} Result with repo and number, or undefined if skipped
   */
  return async function (outputItem, resolvedTemporaryIds) {
    // Determine the issue number
    let issueNumber;

    if (target === "*") {
      // Use explicit number from the item
      const targetNumber = outputItem.issue_number;
      if (targetNumber) {
        issueNumber = parseInt(targetNumber, 10);
        if (isNaN(issueNumber) || issueNumber <= 0) {
          core.info(`Invalid issue number specified: ${targetNumber}`);
          return;
        }
      } else {
        core.info(`Target is "*" but no issue_number specified in close-issue item`);
        return;
      }
    } else if (target !== "triggering") {
      // Explicit number specified in target configuration
      issueNumber = parseInt(target, 10);
      if (isNaN(issueNumber) || issueNumber <= 0) {
        core.info(`Invalid issue number in target configuration: ${target}`);
        return;
      }
    } else {
      // Default behavior: use triggering issue
      const isIssueContext = context.eventName === "issues" || context.eventName === "issue_comment";
      if (isIssueContext) {
        issueNumber = context.payload.issue?.number;
        if (!issueNumber) {
          core.info("Issue context detected but no issue found in payload");
          return;
        }
      } else {
        core.info("Not in issue context and no explicit target specified");
        return;
      }
    }

    try {
      // Fetch issue details to check filters
      const issue = await getIssueDetails(github, context.repo.owner, context.repo.repo, issueNumber);

      // Apply label filter
      if (requiredLabels.length > 0) {
        const issueLabels = issue.labels.map(l => l.name);
        const hasRequiredLabel = requiredLabels.some(required => issueLabels.includes(required));
        if (!hasRequiredLabel) {
          core.info(`Issue #${issueNumber} does not have required labels: ${requiredLabels.join(", ")}`);
          return;
        }
      }

      // Apply title prefix filter
      if (requiredTitlePrefix && !issue.title.startsWith(requiredTitlePrefix)) {
        core.info(`Issue #${issueNumber} does not have required title prefix: ${requiredTitlePrefix}`);
        return;
      }

      // Check if already closed
      if (issue.state === "closed") {
        core.info(`Issue #${issueNumber} is already closed, skipping`);
        return;
      }

      // Build comment body
      const runId = context.runId;
      const githubServer = "https://github.com";
      const runUrl = context.payload.repository
        ? `${context.payload.repository.html_url}/actions/runs/${runId}`
        : `${githubServer}/${context.repo.owner}/${context.repo.repo}/actions/runs/${runId}`;

      const triggeringIssueNumber = context.payload?.issue?.number;
      const triggeringPRNumber = context.payload?.pull_request?.number;

      let commentBody = outputItem.body.trim();
      commentBody += getTrackerID("markdown");
      commentBody += generateFooter(
        workflowName,
        runUrl,
        workflowSource,
        workflowSourceURL,
        triggeringIssueNumber,
        triggeringPRNumber,
        undefined
      );

      // Add comment before closing
      const comment = await addIssueComment(github, context.repo.owner, context.repo.repo, issueNumber, commentBody);
      core.info(`✓ Added comment to issue #${issueNumber}: ${comment.html_url}`);

      // Close the issue
      const closedIssue = await closeIssue(github, context.repo.owner, context.repo.repo, issueNumber);
      core.info(`✓ Closed issue #${issueNumber}: ${closedIssue.html_url}`);

      return {
        repo: `${context.repo.owner}/${context.repo.repo}`,
        number: issueNumber,
      };
    } catch (error) {
      core.error(`✗ Failed to close issue #${issueNumber}: ${getErrorMessage(error)}`);
      throw error;
    }
  };
}

module.exports = { main };
