// @ts-check
/// <reference types="@actions/github-script" />

const { generateFooter } = require("./generate_footer.cjs");
const { getTrackerID } = require("./get_tracker_id.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Get discussion details using GraphQL with pagination for labels
 * @param {any} github - GitHub GraphQL instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} discussionNumber - Discussion number
 * @returns {Promise<{id: string, title: string, category: {name: string}, labels: {nodes: Array<{name: string}>}, url: string}>} Discussion details
 */
async function getDiscussionDetails(github, owner, repo, discussionNumber) {
  // Fetch all labels with pagination
  const allLabels = [];
  let hasNextPage = true;
  let cursor = null;
  let discussion = null;

  while (hasNextPage) {
    const query = await github.graphql(
      `
      query($owner: String!, $repo: String!, $num: Int!, $cursor: String) {
        repository(owner: $owner, name: $repo) {
          discussion(number: $num) {
            id
            title
            category {
              name
            }
            url
            labels(first: 100, after: $cursor) {
              nodes {
                name
              }
              pageInfo {
                hasNextPage
                endCursor
              }
            }
          }
        }
      }`,
      { owner, repo, num: discussionNumber, cursor }
    );

    if (!query?.repository?.discussion) {
      throw new Error(`Discussion #${discussionNumber} not found in ${owner}/${repo}`);
    }

    // Store the discussion metadata from the first query
    if (!discussion) {
      discussion = {
        id: query.repository.discussion.id,
        title: query.repository.discussion.title,
        category: query.repository.discussion.category,
        url: query.repository.discussion.url,
      };
    }

    const labels = query.repository.discussion.labels?.nodes || [];
    allLabels.push(...labels);

    hasNextPage = query.repository.discussion.labels?.pageInfo?.hasNextPage || false;
    cursor = query.repository.discussion.labels?.pageInfo?.endCursor || null;
  }

  // discussion is guaranteed to be set because we always enter the while loop at least once
  // and throw an error if the discussion is not found
  if (!discussion) {
    throw new Error(`Failed to fetch discussion #${discussionNumber}`);
  }

  return {
    id: discussion.id,
    title: discussion.title,
    category: discussion.category,
    url: discussion.url,
    labels: {
      nodes: allLabels,
    },
  };
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
 * Close a GitHub Discussion using GraphQL
 * @param {any} github - GitHub GraphQL instance
 * @param {string} discussionId - Discussion node ID
 * @param {string|undefined} reason - Optional close reason (RESOLVED, DUPLICATE, OUTDATED, or ANSWERED)
 * @returns {Promise<{id: string, url: string}>} Discussion details
 */
async function closeDiscussion(github, discussionId, reason) {
  const mutation = reason
    ? `
      mutation($dId: ID!, $reason: DiscussionCloseReason!) {
        closeDiscussion(input: { discussionId: $dId, reason: $reason }) {
          discussion { 
            id
            url
          }
        }
      }`
    : `
      mutation($dId: ID!) {
        closeDiscussion(input: { discussionId: $dId }) {
          discussion { 
            id
            url
          }
        }
      }`;

  const variables = reason ? { dId: discussionId, reason } : { dId: discussionId };
  const result = await github.graphql(mutation, variables);

  return result.closeDiscussion.discussion;
}

/**
 * Factory function for creating close discussion handler
 * @param {Object} config - Handler configuration
 * @param {string[]} [config.requiredLabels] - Required labels (any match)
 * @param {string} [config.requiredTitlePrefix] - Required title prefix
 * @param {string} [config.requiredCategory] - Required category
 * @param {string} [config.target] - Target configuration ("triggering", "*", or explicit number)
 * @returns {Function} Handler function that processes individual messages
 */
async function main(config = {}) {
  const { requiredLabels = [], requiredTitlePrefix = "", requiredCategory = "", target = "triggering" } = config;

  /**
   * Process a single close_discussion message
   * @param {Object} outputItem - The safe output item
   * @param {Object} resolvedTemporaryIds - Map of temporary IDs to actual IDs
   * @returns {Promise<{repo: string, number: number}|undefined>} Result with repo and number, or undefined if skipped
   */
  return async function (outputItem, resolvedTemporaryIds) {
    // Determine the discussion number
    let discussionNumber;

    if (target === "*") {
      // Use explicit number from the item
      const targetNumber = outputItem.discussion_number;
      if (targetNumber) {
        discussionNumber = parseInt(targetNumber, 10);
        if (isNaN(discussionNumber) || discussionNumber <= 0) {
          core.info(`Invalid discussion number specified: ${targetNumber}`);
          return;
        }
      } else {
        core.info(`Target is "*" but no discussion_number specified in close-discussion item`);
        return;
      }
    } else if (target !== "triggering") {
      // Explicit number specified in target configuration
      discussionNumber = parseInt(target, 10);
      if (isNaN(discussionNumber) || discussionNumber <= 0) {
        core.info(`Invalid discussion number in target configuration: ${target}`);
        return;
      }
    } else {
      // Default behavior: use triggering discussion
      const isDiscussionContext = context.eventName === "discussion" || context.eventName === "discussion_comment";
      if (isDiscussionContext) {
        discussionNumber = context.payload.discussion?.number;
        if (!discussionNumber) {
          core.info("Discussion context detected but no discussion found in payload");
          return;
        }
      } else {
        core.info("Not in discussion context and no explicit target specified");
        return;
      }
    }

    try {
      // Fetch discussion details to check filters
      const discussion = await getDiscussionDetails(github, context.repo.owner, context.repo.repo, discussionNumber);

      // Apply label filter
      if (requiredLabels.length > 0) {
        const discussionLabels = discussion.labels.nodes.map(l => l.name);
        const hasRequiredLabel = requiredLabels.some(required => discussionLabels.includes(required));
        if (!hasRequiredLabel) {
          core.info(`Discussion #${discussionNumber} does not have required labels: ${requiredLabels.join(", ")}`);
          return;
        }
      }

      // Apply title prefix filter
      if (requiredTitlePrefix && !discussion.title.startsWith(requiredTitlePrefix)) {
        core.info(`Discussion #${discussionNumber} does not have required title prefix: ${requiredTitlePrefix}`);
        return;
      }

      // Apply category filter
      if (requiredCategory && discussion.category.name !== requiredCategory) {
        core.info(`Discussion #${discussionNumber} is not in required category: ${requiredCategory}`);
        return;
      }

      // Build comment body
      const workflowName = process.env.GH_AW_WORKFLOW_NAME || "Workflow";
      const workflowSource = process.env.GH_AW_WORKFLOW_SOURCE || "";
      const workflowSourceURL = process.env.GH_AW_WORKFLOW_SOURCE_URL || "";
      const runId = context.runId;
      const githubServer = process.env.GITHUB_SERVER_URL || "https://github.com";
      const runUrl = context.payload.repository
        ? `${context.payload.repository.html_url}/actions/runs/${runId}`
        : `${githubServer}/${context.repo.owner}/${context.repo.repo}/actions/runs/${runId}`;

      const triggeringDiscussionNumber = context.payload?.discussion?.number;

      let body = outputItem.body.trim();
      body += getTrackerID("markdown");
      body += generateFooter(
        workflowName,
        runUrl,
        workflowSource,
        workflowSourceURL,
        undefined,
        undefined,
        triggeringDiscussionNumber
      );

      core.info(`Adding comment to discussion #${discussionNumber}`);
      core.info(`Comment content length: ${body.length}`);

      // Add comment first
      const comment = await addDiscussionComment(github, discussion.id, body);
      core.info("Added discussion comment: " + comment.url);

      // Then close the discussion
      core.info(`Closing discussion #${discussionNumber} with reason: ${outputItem.reason || "none"}`);
      const closedDiscussion = await closeDiscussion(github, discussion.id, outputItem.reason);
      core.info("Closed discussion: " + closedDiscussion.url);

      return {
        repo: `${context.repo.owner}/${context.repo.repo}`,
        number: discussionNumber,
      };
    } catch (error) {
      core.error(`✗ Failed to close discussion #${discussionNumber}: ${getErrorMessage(error)}`);
      throw error;
    }
  };
}

module.exports = { main };
