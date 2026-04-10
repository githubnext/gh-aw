// @ts-check
/// <reference types="@actions/github-script" />

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

const { resolveTargetRepoConfig, resolveAndValidateRepo } = require("./repo_helpers.cjs");
const { createAuthenticatedGitHubClient } = require("./handler_auth.cjs");
const { ERR_NOT_FOUND } = require("./error_codes.cjs");
const { createCloseEntityHandler, ISSUE_CONFIG } = require("./close_entity_helpers.cjs");

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
    throw new Error(`${ERR_NOT_FOUND}: Issue #${issueNumber} not found in ${owner}/${repo}`);
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
 * @param {string} [stateReason] - The reason for closing: "COMPLETED", "NOT_PLANNED", or "DUPLICATE"
 * @returns {Promise<{number: number, html_url: string, title: string}>} Issue details
 */
async function closeIssue(github, owner, repo, issueNumber, stateReason) {
  const { data: issue } = await github.rest.issues.update({
    owner,
    repo,
    issue_number: issueNumber,
    state: "closed",
    state_reason: (stateReason || "COMPLETED").toLowerCase(),
  });

  return issue;
}

/**
 * Main handler factory for close_issue
 * Returns a message handler function that processes individual close_issue messages
 * @type {HandlerFactoryFunction}
 */
async function main(config = {}) {
  const configStateReason = config.state_reason || "COMPLETED";
  const requiredLabels = config.required_labels || [];
  const requiredTitlePrefix = config.required_title_prefix || "";
  const { defaultTargetRepo, allowedRepos } = resolveTargetRepoConfig(config);
  const githubClient = await createAuthenticatedGitHubClient(config);

  core.info(`Close issue configuration: max=${config.max || 10}, state_reason=${configStateReason}`);
  if (requiredLabels.length > 0) {
    core.info(`Required labels: ${requiredLabels.join(", ")}`);
  }
  if (requiredTitlePrefix) {
    core.info(`Required title prefix: ${requiredTitlePrefix}`);
  }
  core.info(`Default target repo: ${defaultTargetRepo}`);
  if (allowedRepos.size > 0) {
    core.info(`Allowed repos: ${Array.from(allowedRepos).join(", ")}`);
  }

  return createCloseEntityHandler(
    config,
    ISSUE_CONFIG,
    {
      resolveTarget(item) {
        // Resolve and validate target repository
        const repoResult = resolveAndValidateRepo(item, defaultTargetRepo, allowedRepos, "issue");
        if (!repoResult.success) {
          return { success: false, error: repoResult.error };
        }
        const { repo: entityRepo, repoParts } = repoResult;

        // Determine issue number
        let issueNumber;
        if (item.issue_number !== undefined) {
          issueNumber = parseInt(String(item.issue_number), 10);
          if (isNaN(issueNumber)) {
            return { success: false, error: `Invalid issue number: ${item.issue_number}` };
          }
        } else {
          const contextIssue = context.payload?.issue?.number;
          if (!contextIssue) {
            return { success: false, error: "No issue number available" };
          }
          issueNumber = contextIssue;
        }

        return { success: true, entityNumber: issueNumber, owner: repoParts.owner, repo: repoParts.repo, entityRepo };
      },

      getDetails: getIssueDetails,

      validateLabels(entity, entityNumber, requiredLabels) {
        if (requiredLabels.length > 0) {
          const issueLabels = entity.labels.map(/** @param {any} l */ l => (typeof l === "string" ? l : l.name || ""));
          const missingLabels = requiredLabels.filter(required => !issueLabels.includes(required));
          if (missingLabels.length > 0) {
            return {
              valid: false,
              warning: `Issue #${entityNumber} missing required labels: ${missingLabels.join(", ")}`,
              error: `Missing required labels: ${missingLabels.join(", ")}`,
            };
          }
        }
        return { valid: true };
      },

      buildCommentBody(sanitizedBody) {
        // Issues post the sanitized body directly without a workflow footer
        return sanitizedBody;
      },

      addComment: addIssueComment,

      closeEntity(github, owner, repo, entityNumber, item) {
        // Support item-level state_reason override, falling back to config-level default
        const stateReason = item.state_reason || configStateReason;
        core.info(`Closing issue #${entityNumber} with state_reason=${stateReason}`);
        return closeIssue(github, owner, repo, entityNumber, stateReason);
      },

      continueOnCommentError: false,

      buildSuccessResult(closedEntity, commentResult, wasAlreadyClosed) {
        return {
          success: true,
          number: closedEntity.number,
          url: closedEntity.html_url,
          title: closedEntity.title,
          alreadyClosed: wasAlreadyClosed,
        };
      },
    },
    githubClient
  );
}

module.exports = { main };
