// @ts-check
/// <reference types="@actions/github-script" />

const { getCloseOlderDiscussionMessage } = require("./messages_close_discussion.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Maximum number of older discussions to close
 */
const MAX_CLOSE_COUNT = 10;

/**
 * Delay between GraphQL API calls in milliseconds to avoid rate limiting
 */
const GRAPHQL_DELAY_MS = 500;

/**
 * Delay execution for a specified number of milliseconds
 * @param {number} ms - Milliseconds to delay
 * @returns {Promise<void>}
 */
function delay(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

/**
 * Search for open discussions with a matching title prefix and/or labels
 * @param {any} github - GitHub GraphQL instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {string} titlePrefix - Title prefix to match (empty string to skip prefix matching)
 * @param {string[]} labels - Labels to match (empty array to skip label matching)
 * @param {string|undefined} categoryId - Optional category ID to filter by
 * @param {number} excludeNumber - Discussion number to exclude (the newly created one)
 * @returns {Promise<Array<{id: string, number: number, title: string, url: string}>>} Matching discussions
 */
async function searchOlderDiscussions(github, owner, repo, titlePrefix, labels, categoryId, excludeNumber) {
  // Build GraphQL search query
  // Search for open discussions, optionally with title prefix or labels
  let searchQuery = `repo:${owner}/${repo} is:open`;

  if (titlePrefix) {
    // Escape quotes in title prefix to prevent query injection
    const escapedPrefix = titlePrefix.replace(/"/g, '\\"');
    searchQuery += ` in:title "${escapedPrefix}"`;
  }

  // Add label filters to the search query
  // Note: GitHub search uses AND logic for multiple labels, so discussions must have ALL labels.
  // We add each label as a separate filter and also validate client-side for extra safety.
  if (labels && labels.length > 0) {
    for (const label of labels) {
      // Escape quotes in label names to prevent query injection
      const escapedLabel = label.replace(/"/g, '\\"');
      searchQuery += ` label:"${escapedLabel}"`;
    }
  }

  const result = await github.graphql(
    `
    query($searchTerms: String!, $first: Int!) {
      search(query: $searchTerms, type: DISCUSSION, first: $first) {
        nodes {
          ... on Discussion {
            id
            number
            title
            url
            category {
              id
            }
            labels(first: 100) {
              nodes {
                name
              }
            }
            closed
          }
        }
      }
    }`,
    { searchTerms: searchQuery, first: 50 }
  );

  if (!result || !result.search || !result.search.nodes) {
    return [];
  }

  // Filter results:
  // 1. Must not be the excluded discussion (newly created one)
  // 2. Must not be already closed
  // 3. If titlePrefix is specified, must have title starting with the prefix
  // 4. If labels are specified, must have ALL specified labels (AND logic, not OR)
  // 5. If categoryId is specified, must match
  return result.search.nodes
    .filter(
      /** @param {any} d */ d => {
        if (!d || d.number === excludeNumber || d.closed) {
          return false;
        }

        // Check title prefix if specified
        if (titlePrefix && d.title && !d.title.startsWith(titlePrefix)) {
          return false;
        }

        // Check labels if specified - requires ALL labels to match (AND logic)
        // This is intentional: we only want to close discussions that have ALL the specified labels
        if (labels && labels.length > 0) {
          const discussionLabels = d.labels?.nodes?.map((/** @type {{name: string}} */ l) => l.name) || [];
          const hasAllLabels = labels.every(label => discussionLabels.includes(label));
          if (!hasAllLabels) {
            return false;
          }
        }

        // Check category if specified
        if (categoryId && (!d.category || d.category.id !== categoryId)) {
          return false;
        }

        return true;
      }
    )
    .map(
      /** @param {any} d */ d => ({
        id: d.id,
        number: d.number,
        title: d.title,
        url: d.url,
      })
    );
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
    { dId: discussionId, body: message }
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
 * Close older discussions that match the title prefix and/or labels
 * @param {any} github - GitHub GraphQL instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {string} titlePrefix - Title prefix to match (empty string to skip)
 * @param {string[]} labels - Labels to match (empty array to skip)
 * @param {string|undefined} categoryId - Optional category ID to filter by
 * @param {{number: number, url: string}} newDiscussion - The newly created discussion
 * @param {string} workflowName - Name of the workflow
 * @param {string} runUrl - URL of the workflow run
 * @returns {Promise<Array<{number: number, url: string}>>} List of closed discussions
 */
async function closeOlderDiscussions(github, owner, repo, titlePrefix, labels, categoryId, newDiscussion, workflowName, runUrl) {
  // Build search criteria description for logging
  const searchCriteria = [];
  if (titlePrefix) searchCriteria.push(`title prefix: "${titlePrefix}"`);
  if (labels && labels.length > 0) searchCriteria.push(`labels: [${labels.join(", ")}]`);
  core.info(`Searching for older discussions with ${searchCriteria.join(" and ")}`);

  const olderDiscussions = await searchOlderDiscussions(github, owner, repo, titlePrefix, labels, categoryId, newDiscussion.number);

  if (olderDiscussions.length === 0) {
    core.info("No older discussions found to close");
    return [];
  }

  core.info(`Found ${olderDiscussions.length} older discussion(s) to close`);

  // Limit to MAX_CLOSE_COUNT discussions
  const discussionsToClose = olderDiscussions.slice(0, MAX_CLOSE_COUNT);

  if (olderDiscussions.length > MAX_CLOSE_COUNT) {
    core.warning(`Found ${olderDiscussions.length} older discussions, but only closing the first ${MAX_CLOSE_COUNT}`);
  }

  const closedDiscussions = [];

  for (let i = 0; i < discussionsToClose.length; i++) {
    const discussion = discussionsToClose[i];
    try {
      // Generate closing message using the messages module
      const closingMessage = getCloseOlderDiscussionMessage({
        newDiscussionUrl: newDiscussion.url,
        newDiscussionNumber: newDiscussion.number,
        workflowName,
        runUrl,
      });

      // Add comment first
      core.info(`Adding closing comment to discussion #${discussion.number}`);
      await addDiscussionComment(github, discussion.id, closingMessage);

      // Then close the discussion as outdated
      core.info(`Closing discussion #${discussion.number} as outdated`);
      await closeDiscussionAsOutdated(github, discussion.id);

      closedDiscussions.push({
        number: discussion.number,
        url: discussion.url,
      });

      core.info(`✓ Closed discussion #${discussion.number}: ${discussion.url}`);
    } catch (error) {
      core.error(`✗ Failed to close discussion #${discussion.number}: ${getErrorMessage(error)}`);
      // Continue with other discussions even if one fails
    }

    // Add delay between GraphQL operations to avoid rate limiting (except for the last item)
    if (i < discussionsToClose.length - 1) {
      await delay(GRAPHQL_DELAY_MS);
    }
  }

  return closedDiscussions;
}

module.exports = {
  closeOlderDiscussions,
  searchOlderDiscussions,
  addDiscussionComment,
  closeDiscussionAsOutdated,
  MAX_CLOSE_COUNT,
  GRAPHQL_DELAY_MS,
};
