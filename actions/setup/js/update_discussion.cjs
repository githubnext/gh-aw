// @ts-check
/// <reference types="@actions/github-script" />

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

const { isDiscussionContext, getDiscussionNumber } = require("./update_context_helpers.cjs");
const { createUpdateHandlerFactory, createStandardFormatResult } = require("./update_handler_factory.cjs");
const { sanitizeTitle } = require("./sanitize_title.cjs");
const { validateLabels } = require("./safe_output_validator.cjs");
const { tryEnforceArrayLimit } = require("./limit_enforcement_helpers.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { ERR_NOT_FOUND } = require("./error_codes.cjs");
const { MAX_LABELS } = require("./constants.cjs");

/**
 * Fetches label node IDs for the given label names from a repository
 * @param {any} githubClient - GitHub API client
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {string[]} labelNames - Array of label names to resolve
 * @returns {Promise<Array<{name: string, id: string}>>} Array of matched label objects with name and ID
 */
async function fetchLabelIds(githubClient, owner, repo, labelNames) {
  if (!labelNames || labelNames.length === 0) {
    return [];
  }

  try {
    const labelsQuery = `
      query($owner: String!, $repo: String!) {
        repository(owner: $owner, name: $repo) {
          labels(first: 100) {
            nodes {
              id
              name
            }
          }
        }
      }
    `;

    const queryResult = await githubClient.graphql(labelsQuery, { owner, repo });
    const repoLabels = queryResult?.repository?.labels?.nodes || [];
    const labelMap = new Map(repoLabels.map(/** @param {any} l */ l => [l.name.toLowerCase(), l]));

    const matchedLabels = [];
    const unmatchedLabels = [];

    for (const requestedLabel of labelNames) {
      const matched = labelMap.get(requestedLabel.toLowerCase());
      if (matched) {
        matchedLabels.push({ name: matched.name, id: matched.id });
      } else {
        unmatchedLabels.push(requestedLabel);
      }
    }

    if (unmatchedLabels.length > 0) {
      core.warning(`Could not find label IDs for: ${unmatchedLabels.join(", ")}`);
      core.info(`These labels may not exist in the repository. Available: ${repoLabels.map(/** @param {any} l */ l => l.name).join(", ")}`);
    }

    return matchedLabels;
  } catch (error) {
    core.warning(`Failed to fetch label IDs: ${getErrorMessage(error)}`);
    return [];
  }
}

/**
 * Removes labels from a discussion using GraphQL
 * @param {any} githubClient - GitHub API client
 * @param {string} discussionId - Discussion node ID
 * @param {string[]} labelIds - Label node IDs to remove
 * @returns {Promise<void>}
 */
async function removeLabelsFromDiscussion(githubClient, discussionId, labelIds) {
  if (!labelIds || labelIds.length === 0) {
    return;
  }

  const mutation = `
    mutation($labelableId: ID!, $labelIds: [ID!]!) {
      removeLabelsFromLabelable(input: {
        labelableId: $labelableId,
        labelIds: $labelIds
      }) {
        labelable {
          ... on Discussion { id }
        }
      }
    }
  `;

  await githubClient.graphql(mutation, { labelableId: discussionId, labelIds });
}

/**
 * Adds labels to a discussion using GraphQL
 * @param {any} githubClient - GitHub API client
 * @param {string} discussionId - Discussion node ID
 * @param {string[]} labelIds - Label node IDs to add
 * @returns {Promise<void>}
 */
async function addLabelsToDiscussion(githubClient, discussionId, labelIds) {
  if (!labelIds || labelIds.length === 0) {
    return;
  }

  const mutation = `
    mutation($labelableId: ID!, $labelIds: [ID!]!) {
      addLabelsToLabelable(input: {
        labelableId: $labelableId,
        labelIds: $labelIds
      }) {
        labelable {
          ... on Discussion { id }
        }
      }
    }
  `;

  await githubClient.graphql(mutation, { labelableId: discussionId, labelIds });
}

/**
 * Execute the discussion update API call using GraphQL
 * @param {any} github - GitHub API client
 * @param {any} context - GitHub Actions context
 * @param {number} discussionNumber - Discussion number to update
 * @param {any} updateData - Data to update
 * @returns {Promise<any>} Updated discussion
 */
async function executeDiscussionUpdate(github, context, discussionNumber, updateData) {
  // Fetch the discussion node ID and current labels in one query
  const getDiscussionQuery = `
    query($owner: String!, $repo: String!, $number: Int!) {
      repository(owner: $owner, name: $repo) {
        discussion(number: $number) {
          id
          title
          body
          url
          labels(first: 100) {
            nodes {
              id
              name
            }
          }
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
    throw new Error(`${ERR_NOT_FOUND}: Discussion #${discussionNumber} not found`);
  }

  // Update title and/or body if provided
  if (updateData.title !== undefined || updateData.body !== undefined) {
    const mutation = `
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
    // Merge updated title/body back into discussion for return value
    const updated = mutationResult.updateDiscussion.discussion;
    discussion.title = updated.title;
    discussion.body = updated.body;
    discussion.url = updated.url;
  }

  // Handle label replacement if labels were provided
  if (updateData.labels !== undefined) {
    const currentLabels = discussion.labels?.nodes || [];
    const currentLabelIds = new Set(currentLabels.map(/** @param {any} l */ l => l.id));

    // Look up node IDs for the requested labels
    const requestedLabelData = await fetchLabelIds(github, context.repo.owner, context.repo.repo, updateData.labels);
    const requestedLabelIdSet = new Set(requestedLabelData.map(/** @param {any} l */ l => l.id));

    // Compute add/remove sets
    const labelsToAdd = requestedLabelData.filter(l => !currentLabelIds.has(l.id)).map(/** @param {any} l */ l => l.id);
    const labelsToRemove = currentLabels.filter(/** @param {any} l */ l => !requestedLabelIdSet.has(l.id)).map(/** @param {any} l */ l => l.id);

    if (labelsToAdd.length > 0) {
      core.info(`Adding ${labelsToAdd.length} label(s) to discussion #${discussionNumber}`);
      await addLabelsToDiscussion(github, discussion.id, labelsToAdd);
    }
    if (labelsToRemove.length > 0) {
      core.info(`Removing ${labelsToRemove.length} label(s) from discussion #${discussionNumber}`);
      await removeLabelsFromDiscussion(github, discussion.id, labelsToRemove);
    }
    if (labelsToAdd.length === 0 && labelsToRemove.length === 0) {
      core.info(`Labels unchanged for discussion #${discussionNumber}`);
    }
  }

  return discussion;
}

/**
 * Resolve discussion number from message and configuration
 * Discussions have special handling - they don't use the standard resolveTarget helper
 * @param {Object} item - The message item
 * @param {string} updateTarget - Target configuration
 * @param {Object} context - GitHub Actions context
 * @returns {{success: true, number: number} | {success: false, error: string}} Resolution result
 */
function resolveDiscussionNumber(item, updateTarget, context) {
  // Discussions are special - they have their own context type separate from issues/PRs
  // We need to handle them differently
  if (item.discussion_number !== undefined) {
    const discussionNumber = parseInt(String(item.discussion_number), 10);
    if (isNaN(discussionNumber)) {
      return {
        success: false,
        error: `Invalid discussion number: ${item.discussion_number}`,
      };
    }
    return { success: true, number: discussionNumber };
  } else if (updateTarget !== "triggering") {
    // Explicit number target
    const discussionNumber = parseInt(updateTarget, 10);
    if (isNaN(discussionNumber) || discussionNumber <= 0) {
      return {
        success: false,
        error: `Invalid discussion number in target: ${updateTarget}`,
      };
    }
    return { success: true, number: discussionNumber };
  } else {
    // Use triggering context (default)
    if (isDiscussionContext(context.eventName, context.payload)) {
      const discussionNumber = getDiscussionNumber(context.payload);
      if (!discussionNumber) {
        return {
          success: false,
          error: "No discussion number available",
        };
      }
      return { success: true, number: discussionNumber };
    } else {
      return {
        success: false,
        error: "Not in discussion context",
      };
    }
  }
}

/**
 * Build update data from message
 * @param {Object} item - The message item
 * @param {Object} config - Configuration object
 * @returns {{success: true, data: Object} | {success: false, error: string}} Update data result
 */
function buildDiscussionUpdateData(item, config) {
  const updateData = {};

  if (item.title !== undefined) {
    // Sanitize title for Unicode security (no prefix handling needed for updates)
    updateData.title = sanitizeTitle(item.title);
  }
  if (item.body !== undefined) {
    updateData.body = item.body;
  }

  // Handle labels - consistent with update_issue: labels are always processed when provided.
  // Optional allowed_labels config restricts which labels may be set.
  if (item.labels !== undefined) {
    const allowedLabels = config.allowed_labels || [];

    // Enforce max label count
    const labelsLimitResult = tryEnforceArrayLimit(item.labels, MAX_LABELS, "labels");
    if (!labelsLimitResult.success) {
      core.warning(`Discussion label update limit exceeded: ${labelsLimitResult.error}`);
      return { success: false, error: labelsLimitResult.error };
    }

    if (allowedLabels.length > 0) {
      // Filter to allowed labels only; if none remain treat as an empty set
      const labelsResult = validateLabels(item.labels, allowedLabels);
      if (!labelsResult.valid) {
        // All labels were filtered out (e.g. none in allowed list) - treat as empty set
        updateData.labels = [];
      } else {
        updateData.labels = labelsResult.value ?? [];
      }
    } else {
      updateData.labels = item.labels;
    }
  }

  return { success: true, data: updateData };
}

/**
 * Format success result for discussion update
 * Uses the standard format helper for consistency across update handlers
 */
const formatDiscussionSuccessResult = createStandardFormatResult({
  numberField: "number",
  urlField: "url",
  urlSource: "url",
});

/**
 * Main handler factory for update_discussion
 * Returns a message handler function that processes individual update_discussion messages
 * @type {HandlerFactoryFunction}
 */
const main = createUpdateHandlerFactory({
  itemType: "update_discussion",
  itemTypeName: "discussion",
  supportsPR: false,
  resolveItemNumber: resolveDiscussionNumber,
  buildUpdateData: buildDiscussionUpdateData,
  executeUpdate: executeDiscussionUpdate,
  formatSuccessResult: formatDiscussionSuccessResult,
});

module.exports = { main, buildDiscussionUpdateData };
