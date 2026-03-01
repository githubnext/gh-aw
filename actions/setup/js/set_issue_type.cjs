// @ts-check
/// <reference types="@actions/github-script" />

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

const { getErrorMessage } = require("./error_helpers.cjs");
const { logStagedPreviewInfo } = require("./staged_preview.cjs");
const { createAuthenticatedGitHubClient } = require("./handler_auth.cjs");

/** @type {string} Safe output type handled by this module */
const HANDLER_TYPE = "set_issue_type";

/**
 * Fetches the node ID of an issue for use in GraphQL mutations.
 * @param {Object} authClient - Authenticated GitHub client
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} issueNumber - Issue number
 * @returns {Promise<string>} Issue node ID
 */
async function getIssueNodeId(authClient, owner, repo, issueNumber) {
  const { data } = await authClient.rest.issues.get({
    owner,
    repo,
    issue_number: issueNumber,
  });
  return data.node_id;
}

/**
 * Fetches the available issue types for the given repository via GraphQL.
 * Returns an array of { id, name } objects, or an empty array if not supported.
 * @param {Object} authClient - Authenticated GitHub client
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @returns {Promise<Array<{id: string, name: string}>>} Available issue types
 */
async function fetchIssueTypes(authClient, owner, repo) {
  try {
    const result = await authClient.graphql(
      `query($owner: String!, $repo: String!) {
        repository(owner: $owner, name: $repo) {
          issueTypes(first: 100) {
            nodes {
              id
              name
            }
          }
        }
      }`,
      { owner, repo }
    );
    return result?.repository?.issueTypes?.nodes ?? [];
  } catch (error) {
    // Issue types may not be enabled for this repository/organization
    // Log at debug level to aid debugging without being noisy
    if (typeof core !== "undefined") {
      core.debug(`Could not fetch issue types (may not be enabled): ${error instanceof Error ? error.message : String(error)}`);
    }
    return [];
  }
}

/**
 * Sets the issue type via GraphQL mutation.
 * Passing null for typeId clears the type.
 * @param {Object} authClient - Authenticated GitHub client
 * @param {string} issueNodeId - GraphQL node ID of the issue
 * @param {string|null} typeId - GraphQL node ID of the issue type, or null to clear
 * @returns {Promise<void>}
 */
async function setIssueTypeById(authClient, issueNodeId, typeId) {
  await authClient.graphql(
    `mutation($issueId: ID!, $typeId: ID) {
      updateIssue(input: { id: $issueId, issueTypeId: $typeId }) {
        issue {
          id
        }
      }
    }`,
    { issueId: issueNodeId, typeId }
  );
}

/**
 * Main handler factory for set_issue_type
 * Returns a message handler function that processes individual set_issue_type messages
 * @type {HandlerFactoryFunction}
 */
async function main(config = {}) {
  // Extract configuration
  const allowedTypes = config.allowed || [];
  const maxCount = config.max || 5;
  const authClient = await createAuthenticatedGitHubClient(config);

  // Check if we're in staged mode
  const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true";

  core.info(`Set issue type configuration: max=${maxCount}`);
  if (allowedTypes.length > 0) {
    core.info(`Allowed issue types: ${allowedTypes.join(", ")}`);
  }

  // Track how many items we've processed for max limit
  let processedCount = 0;

  /**
   * Message handler function that processes a single set_issue_type message
   * @param {Object} message - The set_issue_type message to process
   * @param {Object} resolvedTemporaryIds - Map of temporary IDs to {repo, number}
   * @returns {Promise<Object>} Result with success/error status
   */
  return async function handleSetIssueType(message, resolvedTemporaryIds) {
    // Check if we've hit the max limit
    if (processedCount >= maxCount) {
      core.warning(`Skipping set_issue_type: max count of ${maxCount} reached`);
      return {
        success: false,
        error: `Max count of ${maxCount} reached`,
      };
    }

    processedCount++;

    const item = message;

    // Determine target issue number
    const issueNumber = item.issue_number !== undefined ? parseInt(String(item.issue_number), 10) : (context.payload?.issue?.number ?? null);

    if (!issueNumber || isNaN(issueNumber) || issueNumber <= 0) {
      const error = item.issue_number !== undefined ? `Invalid issue_number: ${item.issue_number}` : "No issue number available";
      core.warning(error);
      return { success: false, error };
    }

    const issueTypeName = item.issue_type ?? "";
    const isClear = issueTypeName === "";

    core.info(`Setting issue type on issue #${issueNumber}: ${isClear ? "(clear)" : JSON.stringify(issueTypeName)}`);

    // Validate against allowed list if configured (empty string always allowed to clear)
    if (allowedTypes.length > 0 && !isClear) {
      const normalizedAllowed = allowedTypes.map(t => t.toLowerCase());
      if (!normalizedAllowed.includes(issueTypeName.toLowerCase())) {
        const error = `Issue type ${JSON.stringify(issueTypeName)} is not in the allowed list: ${JSON.stringify(allowedTypes)}`;
        core.warning(error);
        return { success: false, error };
      }
    }

    // If in staged mode, preview without executing
    if (isStaged) {
      const description = isClear ? `Would clear issue type on issue #${issueNumber}` : `Would set issue type to ${JSON.stringify(issueTypeName)} on issue #${issueNumber}`;
      logStagedPreviewInfo(description);
      return {
        success: true,
        staged: true,
        previewInfo: {
          issue_number: issueNumber,
          issue_type: issueTypeName,
        },
      };
    }

    try {
      const owner = context.repo.owner;
      const repo = context.repo.repo;

      // Get the issue's node ID for GraphQL
      const issueNodeId = await getIssueNodeId(authClient, owner, repo, issueNumber);

      let typeId = null;
      if (!isClear) {
        // Fetch available issue types and find the matching one
        const issueTypes = await fetchIssueTypes(authClient, owner, repo);

        if (issueTypes.length === 0) {
          const error = "No issue types are available for this repository. Issue types must be configured in the repository or organization settings.";
          core.error(error);
          return { success: false, error };
        }

        const matchedType = issueTypes.find(t => t.name.toLowerCase() === issueTypeName.toLowerCase());
        if (!matchedType) {
          const availableNames = issueTypes.map(t => t.name).join(", ");
          const error = `Issue type ${JSON.stringify(issueTypeName)} not found. Available types: ${availableNames}`;
          core.error(error);
          return { success: false, error };
        }

        typeId = matchedType.id;
        core.info(`Resolved issue type ${JSON.stringify(issueTypeName)} to node ID: ${typeId}`);
      }

      await setIssueTypeById(authClient, issueNodeId, typeId);

      const successMsg = isClear ? `Successfully cleared issue type on issue #${issueNumber}` : `Successfully set issue type to ${JSON.stringify(issueTypeName)} on issue #${issueNumber}`;
      core.info(successMsg);

      return {
        success: true,
        issue_number: issueNumber,
        issue_type: issueTypeName,
      };
    } catch (error) {
      const errorMessage = getErrorMessage(error);
      core.error(`Failed to set issue type on issue #${issueNumber}: ${errorMessage}`);
      return { success: false, error: errorMessage };
    }
  };
}

module.exports = { main };
