// @ts-check
/// <reference types="@actions/github-script" />

const { getCloseOlderDiscussionMessage } = require("./messages.cjs");

/**
 * Maximum number of older discussions to close
 */
const MAX_CLOSE_COUNT = 10;

/**
 * Search for open discussions with a matching title prefix
 * @param {any} github - GitHub GraphQL instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {string} titlePrefix - Title prefix to match
 * @param {string|undefined} categoryId - Optional category ID to filter by
 * @param {number} excludeNumber - Discussion number to exclude (the newly created one)
 * @returns {Promise<Array<{id: string, number: number, title: string, url: string}>>} Matching discussions
 */
async function searchOlderDiscussions(github, owner, repo, titlePrefix, categoryId, excludeNumber) {
  // GraphQL search query for open discussions with title prefix
  // Note: GitHub search doesn't support exact prefix matching, so we search for the prefix
  // and filter client-side
  const searchQuery = `repo:${owner}/${repo} is:open in:title "${titlePrefix}"`;

  const result = await github.graphql(
    `
    query($query: String!, $first: Int!) {
      search(query: $query, type: DISCUSSION, first: $first) {
        nodes {
          ... on Discussion {
            id
            number
            title
            url
            category {
              id
            }
            closed
          }
        }
      }
    }`,
    { query: searchQuery, first: 50 }
  );

  if (!result || !result.search || !result.search.nodes) {
    return [];
  }

  // Filter results:
  // 1. Must have title starting with the prefix
  // 2. Must not be the excluded discussion (newly created one)
  // 3. Must not be already closed
  // 4. If categoryId is specified, must match
  return result.search.nodes
    .filter(
      /** @param {any} d */ d =>
        d &&
        d.title &&
        d.title.startsWith(titlePrefix) &&
        d.number !== excludeNumber &&
        !d.closed &&
        (!categoryId || (d.category && d.category.id === categoryId))
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
    mutation($dId: ID!, $reason: DiscussionCloseReason!) {
      closeDiscussion(input: { discussionId: $dId, reason: $reason }) {
        discussion { 
          id
          url
        }
      }
    }`,
    { dId: discussionId, reason: "OUTDATED" }
  );

  return result.closeDiscussion.discussion;
}

/**
 * Close older discussions that match the title prefix
 * @param {any} github - GitHub GraphQL instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {string} titlePrefix - Title prefix to match
 * @param {string|undefined} categoryId - Optional category ID to filter by
 * @param {{number: number, url: string}} newDiscussion - The newly created discussion
 * @param {string} workflowName - Name of the workflow
 * @param {string} runUrl - URL of the workflow run
 * @returns {Promise<Array<{number: number, url: string}>>} List of closed discussions
 */
async function closeOlderDiscussions(github, owner, repo, titlePrefix, categoryId, newDiscussion, workflowName, runUrl) {
  core.info(`Searching for older discussions with title prefix: "${titlePrefix}"`);

  const olderDiscussions = await searchOlderDiscussions(github, owner, repo, titlePrefix, categoryId, newDiscussion.number);

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

  for (const discussion of discussionsToClose) {
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
      core.error(`✗ Failed to close discussion #${discussion.number}: ${error instanceof Error ? error.message : String(error)}`);
      // Continue with other discussions even if one fails
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
};
