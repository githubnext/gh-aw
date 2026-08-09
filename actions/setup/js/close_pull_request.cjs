// @ts-check
/// <reference types="@actions/github-script" />

const { createAuthenticatedGitHubClient } = require("./handler_auth.cjs");
const { getPullRequestDetails, addIssueThreadComment, closePullRequest } = require("./close_rest_helpers.cjs");
const { createCloseEntityHandler, checkLabelFilter, buildCommentBody, PULL_REQUEST_CONFIG } = require("./close_entity_helpers.cjs");
const { resolveTargetRepoConfig, resolveAndValidateRepo } = require("./repo_helpers.cjs");

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

/**
 * Handler factory for close-pull-request safe outputs
 * @type {HandlerFactoryFunction}
 */
async function main(config = {}) {
  const requiredLabels = config.required_labels || [];
  const requiredTitlePrefix = config.required_title_prefix || "";
  const { defaultTargetRepo, allowedRepos } = resolveTargetRepoConfig(config);
  const githubClient = await createAuthenticatedGitHubClient(config);
  const configuredTargetRepo = config["target-repo"] || "";

  core.info(`Close pull request configuration: max=${config.max || 10}`);
  core.info(`Configured target repo: ${configuredTargetRepo || "(unset)"}`);
  core.info(`Default target repo: ${defaultTargetRepo}`);
  if (allowedRepos.size > 0) {
    core.info(`Allowed repos: ${Array.from(allowedRepos).join(", ")}`);
  }
  if (requiredLabels.length > 0) {
    core.info(`Required labels: ${requiredLabels.join(", ")}`);
  }
  if (requiredTitlePrefix) {
    core.info(`Required title prefix: ${requiredTitlePrefix}`);
  }

  return createCloseEntityHandler(
    config,
    PULL_REQUEST_CONFIG,
    {
      resolveTarget(item) {
        // Resolve and validate target repository
        const repoResult = resolveAndValidateRepo(item, defaultTargetRepo, allowedRepos, "pull request");
        if (!repoResult.success) {
          return { success: false, error: repoResult.error };
        }
        const { repo: entityRepo, repoParts } = repoResult;

        let prNumber;
        if (item.pull_request_number !== undefined) {
          prNumber = parseInt(String(item.pull_request_number), 10);
          if (Number.isNaN(prNumber)) {
            return { success: false, error: `Invalid pull request number: ${item.pull_request_number}` };
          }
        } else {
          const contextPR = context.payload?.pull_request?.number;
          if (!contextPR) {
            return { success: false, error: "No pull_request_number provided and not in pull request context" };
          }
          prNumber = contextPR;
        }
        return { success: true, entityNumber: prNumber, owner: repoParts.owner, repo: repoParts.repo, entityRepo };
      },

      getDetails: getPullRequestDetails,

      validateLabels(entity, entityNumber, requiredLabels) {
        if (!checkLabelFilter(entity.labels, requiredLabels)) {
          return {
            valid: false,
            warning: `Skipping PR #${entityNumber}: does not match label filter (required: ${requiredLabels.join(", ")})`,
            error: "PR does not match required labels",
          };
        }
        return { valid: true };
      },

      buildCommentBody(sanitizedBody) {
        const triggeringPRNumber = context.payload?.pull_request?.number;
        const triggeringIssueNumber = context.payload?.issue?.number;
        return buildCommentBody(sanitizedBody, triggeringIssueNumber, triggeringPRNumber);
      },

      addComment: addIssueThreadComment,

      closeEntity(github, owner, repo, prNumber) {
        core.info(`Closing PR #${prNumber} in ${owner}/${repo}`);
        return closePullRequest(github, owner, repo, prNumber);
      },

      continueOnCommentError: true,

      buildSuccessResult(closedEntity, commentResult, wasAlreadyClosed, commentPosted) {
        return {
          success: true,
          pull_request_number: closedEntity.number,
          pull_request_url: closedEntity.html_url,
          alreadyClosed: wasAlreadyClosed,
          commentPosted,
        };
      },
    },
    githubClient
  );
}

module.exports = { main };
