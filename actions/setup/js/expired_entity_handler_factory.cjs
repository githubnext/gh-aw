// @ts-check
// <reference types="@actions/github-script" />

const { generateExpiredEntityFooter, getExpiredEntityCautionAlert } = require("./generate_footer.cjs");
const { formatDateInProjectTimeZone } = require("./project_timezone.cjs");
const { sanitizeContent } = require("./sanitize_content.cjs");

/**
 * @param {string} label
 * @returns {string}
 */
function sentenceCaseLabel(label) {
  return label.charAt(0).toUpperCase() + label.slice(1).toLowerCase();
}

/**
 * @param {{number: number, url: string, title: string}} entity
 * @returns {{number: number, url: string, title: string}}
 */
function createClosedRecord(entity) {
  return {
    number: entity.number,
    url: entity.url,
    title: entity.title,
  };
}

/**
 * @param {{
 *   entity: {expirationDate: Date},
 *   entityNoun: string,
 *   workflowName: string,
 *   workflowId: string,
 *   runUrl: string,
 *   footerSuffix?: string,
 * }} options
 * @returns {string}
 */
function createExpiredEntityClosingMessage({ entity, entityNoun, workflowName, workflowId, runUrl, footerSuffix = "" }) {
  const cautionAlert = getExpiredEntityCautionAlert(workflowName, runUrl);
  const expirationText = `This ${entityNoun} was automatically closed because it expired on ${formatDateInProjectTimeZone(entity.expirationDate) || entity.expirationDate.toISOString()}.`;

  return (cautionAlert ? cautionAlert + "\n\n" : "") + expirationText + generateExpiredEntityFooter(workflowName, runUrl, workflowId) + footerSuffix;
}

/**
 * Add comment to a GitHub Issue or Pull Request using REST API
 * @param {any} github - GitHub REST instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} issueNumber - Issue or Pull Request number
 * @param {string} message - Comment body
 * @returns {Promise<any>} Comment details
 */
async function addIssueThreadComment(github, owner, repo, issueNumber, message) {
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

/**
 * Close a GitHub Pull Request using REST API
 * @param {any} github - GitHub REST instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} prNumber - Pull request number
 * @returns {Promise<any>} Pull request details
 */
async function closePullRequest(github, owner, repo, prNumber) {
  const result = await github.rest.pulls.update({
    owner: owner,
    repo: repo,
    pull_number: prNumber,
    state: "closed",
  });

  return result.data;
}

/**
 * Add comment to a GitHub Discussion using GraphQL
 * @param {any} github - GitHub GraphQL instance
 * @param {string} discussionId - Discussion node ID
 * @param {string} message - Comment body
 * @returns {Promise<{id: string, url: string}>} Comment details
 */
async function addDiscussionComment(github, discussionId, message) {
  const result = await github.graphql(
    `
    mutation($dId: ID!, $body: String!) {
      addDiscussionComment(input: { discussionId: $dId, body: $body }) {
        comment {
          id
          url
        }
      }
    }`,
    { dId: discussionId, body: sanitizeContent(message) }
  );

  return result.addDiscussionComment.comment;
}

/**
 * Close a GitHub Discussion as OUTDATED using GraphQL
 * @param {any} github - GitHub GraphQL instance
 * @param {string} discussionId - Discussion node ID
 * @returns {Promise<{id: string, url: string}>} Discussion details
 */
async function closeDiscussionAsOutdated(github, discussionId) {
  const result = await github.graphql(
    `
    mutation($dId: ID!) {
      closeDiscussion(input: { discussionId: $dId, reason: OUTDATED }) {
        discussion {
          id
          url
        }
      }
    }`,
    { dId: discussionId }
  );

  return result.closeDiscussion.discussion;
}

/**
 * @param {{
 *   github: any,
 *   owner: string,
 *   repo: string,
 *   workflowName: string,
 *   workflowId: string,
 *   runUrl: string,
 *   entityNoun: string,
 *   entityLabel: string,
 *   footerSuffix?: string,
 *   beforeComment?: (entity: any) => Promise<{status: "closed" | "skipped", record: any} | undefined>,
 *   beforeCommentLog?: (entity: any) => string,
 *   beforeCloseLog?: (entity: any) => string,
 *   addComment: (entity: any, message: string) => Promise<any>,
 *   closeEntity: (entity: any) => Promise<any>,
 * }} options
 * @returns {(entity: any) => Promise<{status: "closed" | "skipped", record: any}>}
 */
function createExpiredEntityHandler(options) {
  return async entity => {
    const earlyResult = options.beforeComment ? await options.beforeComment(entity) : undefined;
    if (earlyResult) {
      return earlyResult;
    }

    const closingMessage = createExpiredEntityClosingMessage({
      entity,
      entityNoun: options.entityNoun,
      workflowName: options.workflowName,
      workflowId: options.workflowId,
      runUrl: options.runUrl,
      footerSuffix: options.footerSuffix,
    });

    if (options.beforeCommentLog) {
      core.info(options.beforeCommentLog(entity));
    }
    await options.addComment(entity, closingMessage);
    core.info(`  ✓ Comment added successfully`);

    if (options.beforeCloseLog) {
      core.info(options.beforeCloseLog(entity));
    }
    await options.closeEntity(entity);
    core.info(`  ✓ ${sentenceCaseLabel(options.entityLabel)} closed successfully`);

    return {
      status: "closed",
      record: createClosedRecord(entity),
    };
  };
}

module.exports = {
  addDiscussionComment,
  addIssueThreadComment,
  closeDiscussionAsOutdated,
  closeIssue,
  closePullRequest,
  createClosedRecord,
  createExpiredEntityClosingMessage,
  createExpiredEntityHandler,
};
