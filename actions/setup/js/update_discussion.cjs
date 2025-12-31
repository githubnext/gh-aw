// @ts-check
/// <reference types="@actions/github-script" />

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

const { isDiscussionContext, getDiscussionNumber } = require("./update_context_helpers.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Execute the discussion update API call using GraphQL
 * @param {any} github - GitHub API client
 * @param {any} context - GitHub Actions context
 * @param {number} discussionNumber - Discussion number to update
 * @param {any} updateData - Data to update
 * @returns {Promise<any>} Updated discussion
 */
async function executeDiscussionUpdate(github, context, discussionNumber, updateData) {
  // First, fetch the discussion node ID
  const getDiscussionQuery = `
    query($owner: String!, $repo: String!, $number: Int!) {
      repository(owner: $owner, name: $repo) {
        discussion(number: $number) {
          id
          title
          body
          url
        }
      }
    }
  `;

  const queryResult = await github.graphql(getDiscussionQuery, {
    owner: context.repo.owner,
    repo: context.repo.repo,
    number: discussionNumber,
  });

  const discussion = queryResult?.repository?.discussion;
  if (!discussion) {
    throw new Error(`Discussion #${discussionNumber} not found`);
  }

  // Build mutation for updating discussion
  let mutation = `
    mutation($discussionId: ID!, $title: String, $body: String) {
      updateDiscussion(input: { discussionId: $discussionId, title: $title, body: $body }) {
        discussion {
          id
          title
          body
          url
        }
      }
    }
  `;

  const variables = {
    discussionId: discussion.id,
    title: updateData.title || discussion.title,
    body: updateData.body || discussion.body,
  };

  const mutationResult = await github.graphql(mutation, variables);
  return mutationResult.updateDiscussion.discussion;
}

/**
 * Main handler factory for update_discussion
 * Returns a message handler function that processes individual update_discussion messages
 * @type {HandlerFactoryFunction}
 */
async function main(config = {}) {
  // Extract configuration
  const updateTarget = config.target || "triggering";
  const maxCount = config.max || 10;

  core.info(`Update discussion configuration: max=${maxCount}, target=${updateTarget}`);

  // Track state
  let processedCount = 0;

  /**
   * Message handler function
   * @param {Object} message - The update_discussion message
   * @param {Object} resolvedTemporaryIds - Resolved temporary IDs
   * @returns {Promise<Object>} Result
   */
  return async function handleUpdateDiscussion(message, resolvedTemporaryIds) {
    // Check max limit
    if (processedCount >= maxCount) {
      core.warning(`Skipping update_discussion: max count of ${maxCount} reached`);
      return {
        success: false,
        error: `Max count of ${maxCount} reached`,
      };
    }

    processedCount++;

    const item = message;

    // Determine target discussion number
    let discussionNumber;
    if (item.discussion_number !== undefined) {
      discussionNumber = parseInt(String(item.discussion_number), 10);
      if (isNaN(discussionNumber)) {
        core.warning(`Invalid discussion number: ${item.discussion_number}`);
        return {
          success: false,
          error: `Invalid discussion number: ${item.discussion_number}`,
        };
      }
    } else {
      // Use triggering context
      if (updateTarget === "triggering" && isDiscussionContext(context.eventName, context.payload)) {
        discussionNumber = getDiscussionNumber(context.payload);
        if (!discussionNumber) {
          core.warning("No discussion number in triggering context");
          return {
            success: false,
            error: "No discussion number available",
          };
        }
      } else {
        core.warning("No discussion_number provided");
        return {
          success: false,
          error: "No discussion number provided",
        };
      }
    }

    // Build update data
    const updateData = {};
    if (item.title !== undefined) {
      updateData.title = item.title;
    }
    if (item.body !== undefined) {
      updateData.body = item.body;
    }

    if (Object.keys(updateData).length === 0) {
      core.warning("No update fields provided");
      return {
        success: false,
        error: "No update fields provided",
      };
    }

    core.info(`Updating discussion #${discussionNumber} with: ${JSON.stringify(Object.keys(updateData))}`);

    try {
      const updatedDiscussion = await executeDiscussionUpdate(github, context, discussionNumber, updateData);
      core.info(`Successfully updated discussion #${discussionNumber}: ${updatedDiscussion.url}`);

      return {
        success: true,
        number: discussionNumber,
        url: updatedDiscussion.url,
        title: updatedDiscussion.title,
      };
    } catch (error) {
      const errorMessage = getErrorMessage(error);
      core.error(`Failed to update discussion #${discussionNumber}: ${errorMessage}`);
      return {
        success: false,
        error: errorMessage,
      };
    }
  };
}

module.exports = { main };
