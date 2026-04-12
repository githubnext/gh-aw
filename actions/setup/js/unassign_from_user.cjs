// @ts-check
/// <reference types="@actions/github-script" />

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

const { processItems } = require("./safe_output_processor.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { resolveTargetRepoConfig, resolveAndValidateRepo } = require("./repo_helpers.cjs");
const { resolveIssueNumber, extractAssignees } = require("./safe_output_helpers.cjs");
const { logStagedPreviewInfo } = require("./staged_preview.cjs");
const { createAuthenticatedGitHubClient } = require("./handler_auth.cjs");
const { createCountGatedHandler } = require("./handler_scaffold.cjs");

/** @type {string} Safe output type handled by this module */
const HANDLER_TYPE = "unassign_from_user";

/**
 * Main handler factory for unassign_from_user
 * Uses shared count-gated scaffold for max-limit enforcement.
 * @type {HandlerFactoryFunction}
 */
const main = createCountGatedHandler({
  handlerType: HANDLER_TYPE,
  setup: async (config, maxCount, isStaged) => {
    // Extract configuration
    const allowedAssignees = config.allowed || [];
    const blockedAssignees = config.blocked || [];

    // Resolve target repository configuration
    const { defaultTargetRepo, allowedRepos } = resolveTargetRepoConfig(config);
    const githubClient = await createAuthenticatedGitHubClient(config);

    core.info(`Unassign from user configuration: max=${maxCount}`);
    if (allowedAssignees.length > 0) {
      core.info(`Allowed assignees to unassign: ${allowedAssignees.join(", ")}`);
    }
    if (blockedAssignees.length > 0) {
      core.info(`Blocked assignees to unassign: ${blockedAssignees.join(", ")}`);
    }
    core.info(`Default target repository: ${defaultTargetRepo}`);
    if (allowedRepos.size > 0) {
      core.info(`Additional allowed repositories: ${Array.from(allowedRepos).join(", ")}`);
    }

    /**
     * Message handler function that processes a single unassign_from_user message
     * @param {Object} message - The unassign_from_user message to process
     * @param {Object} resolvedTemporaryIds - Map of temporary IDs to {repo, number}
     * @returns {Promise<Object>} Result with success/error status
     */
    return async function handleUnassignFromUser(message, resolvedTemporaryIds) {
      const unassignItem = message;

      // Determine issue number using shared helper
      const issueResult = resolveIssueNumber(unassignItem);
      if (!issueResult.success) {
        core.warning(`Skipping unassign_from_user: ${issueResult.error}`);
        return {
          success: false,
          error: issueResult.error,
        };
      }
      const issueNumber = issueResult.issueNumber;

      // Extract assignees using shared helper
      const requestedAssignees = extractAssignees(unassignItem);

      core.info(`Requested assignees to unassign: ${JSON.stringify(requestedAssignees)}`);

      // Use shared helper to filter, sanitize, dedupe, and limit
      const uniqueAssignees = processItems(requestedAssignees, allowedAssignees, maxCount, blockedAssignees);

      if (uniqueAssignees.length === 0) {
        core.info("No assignees to remove");
        return {
          success: true,
          issueNumber: issueNumber,
          assigneesRemoved: [],
          message: "No valid assignees found",
        };
      }

      // Resolve and validate target repository
      const repoResult = resolveAndValidateRepo(unassignItem, defaultTargetRepo, allowedRepos, "issue");

      if (!repoResult.success) {
        core.warning(`Repository validation failed: ${repoResult.error}`);
        return {
          success: false,
          error: repoResult.error,
        };
      }

      const repoParts = repoResult.repoParts;
      const targetRepo = repoResult.repo;

      core.info(`Unassigning ${uniqueAssignees.length} users from issue #${issueNumber} in ${targetRepo}: ${JSON.stringify(uniqueAssignees)}`);

      // If in staged mode, preview without executing
      if (isStaged) {
        logStagedPreviewInfo(`Would unassign users from issue #${issueNumber} in ${targetRepo}`);
        return {
          success: true,
          staged: true,
          previewInfo: {
            issueNumber,
            repo: targetRepo,
            assignees: uniqueAssignees,
          },
        };
      }

      try {
        // Remove assignees from the issue
        await githubClient.rest.issues.removeAssignees({
          owner: repoParts.owner,
          repo: repoParts.repo,
          issue_number: issueNumber,
          assignees: uniqueAssignees,
        });

        core.info(`Successfully unassigned ${uniqueAssignees.length} user(s) from issue #${issueNumber} in ${targetRepo}`);

        return {
          success: true,
          issueNumber: issueNumber,
          repo: targetRepo,
          assigneesRemoved: uniqueAssignees,
        };
      } catch (error) {
        const errorMessage = getErrorMessage(error);
        core.error(`Failed to unassign users: ${errorMessage}`);
        return {
          success: false,
          error: errorMessage,
        };
      }
    };
  },
});

module.exports = { main };
