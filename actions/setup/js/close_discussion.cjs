// @ts-check
/// <reference types="@actions/github-script" />

const { loadAgentOutput } = require("./load_agent_output.cjs");
const { generateFooter } = require("./generate_footer.cjs");
const { getTrackerID } = require("./get_tracker_id.cjs");
const { getRepositoryUrl } = require("./get_repository_url.cjs");

/**
 * Get discussion details using GraphQL
 * @param {any} github - GitHub GraphQL instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} discussionNumber - Discussion number
 * @returns {Promise<{id: string, title: string, category: {name: string}, labels: {nodes: Array<{name: string}>}, url: string}>} Discussion details
 */
async function getDiscussionDetails(github, owner, repo, discussionNumber) {
  const { repository } = await github.graphql(
    `
    query($owner: String!, $repo: String!, $num: Int!) {
      repository(owner: $owner, name: $repo) {
        discussion(number: $num) { 
          id
          title
          category {
            name
          }
          labels(first: 100) {
            nodes {
              name
            }
          }
          url
        }
      }
    }`,
    { owner, repo, num: discussionNumber }
  );

  if (!repository || !repository.discussion) {
    throw new Error(`Discussion #${discussionNumber} not found in ${owner}/${repo}`);
  }

  return repository.discussion;
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

async function main() {
  // Check if we're in staged mode
  const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true";

  const result = loadAgentOutput();
  if (!result.success) {
    return;
  }

  // Find all close-discussion items
  const closeDiscussionItems = result.items.filter(/** @param {any} item */ item => item.type === "close_discussion");
  if (closeDiscussionItems.length === 0) {
    core.info("No close-discussion items found in agent output");
    return;
  }

  core.info(`Found ${closeDiscussionItems.length} close-discussion item(s)`);

  // Get configuration from environment
  const requiredLabels = process.env.GH_AW_CLOSE_DISCUSSION_REQUIRED_LABELS ? process.env.GH_AW_CLOSE_DISCUSSION_REQUIRED_LABELS.split(",").map(l => l.trim()) : [];
  const requiredTitlePrefix = process.env.GH_AW_CLOSE_DISCUSSION_REQUIRED_TITLE_PREFIX || "";
  const requiredCategory = process.env.GH_AW_CLOSE_DISCUSSION_REQUIRED_CATEGORY || "";
  const target = process.env.GH_AW_CLOSE_DISCUSSION_TARGET || "triggering";

  core.info(`Configuration: requiredLabels=${requiredLabels.join(",")}, requiredTitlePrefix=${requiredTitlePrefix}, requiredCategory=${requiredCategory}, target=${target}`);

  // Check if we're in a discussion context
  const isDiscussionContext = context.eventName === "discussion" || context.eventName === "discussion_comment";

  // If in staged mode, emit step summary instead of closing discussions
  if (isStaged) {
    let summaryContent = "## 🎭 Staged Mode: Close Discussions Preview\n\n";
    summaryContent += "The following discussions would be closed if staged mode was disabled:\n\n";

    for (let i = 0; i < closeDiscussionItems.length; i++) {
      const item = closeDiscussionItems[i];
      summaryContent += `### Discussion ${i + 1}\n`;

      const discussionNumber = item.discussion_number;
      if (discussionNumber) {
        const repoUrl = getRepositoryUrl();
        const discussionUrl = `${repoUrl}/discussions/${discussionNumber}`;
        summaryContent += `**Target Discussion:** [#${discussionNumber}](${discussionUrl})\n\n`;
      } else {
        summaryContent += `**Target:** Current discussion\n\n`;
      }

      if (item.reason) {
        summaryContent += `**Reason:** ${item.reason}\n\n`;
      }

      summaryContent += `**Comment:**\n${item.body || "No content provided"}\n\n`;

      if (requiredLabels.length > 0) {
        summaryContent += `**Required Labels:** ${requiredLabels.join(", ")}\n\n`;
      }
      if (requiredTitlePrefix) {
        summaryContent += `**Required Title Prefix:** ${requiredTitlePrefix}\n\n`;
      }
      if (requiredCategory) {
        summaryContent += `**Required Category:** ${requiredCategory}\n\n`;
      }

      summaryContent += "---\n\n";
    }

    // Write to step summary
    await core.summary.addRaw(summaryContent).write();
    core.info("📝 Discussion close preview written to step summary");
    return;
  }

  // Validate context based on target configuration
  if (target === "triggering" && !isDiscussionContext) {
    core.info('Target is "triggering" but not running in discussion context, skipping discussion close');
    return;
  }

  // Extract triggering context for footer generation
  const triggeringDiscussionNumber = context.payload?.discussion?.number;

  const closedDiscussions = [];

  // Process each close-discussion item
  for (let i = 0; i < closeDiscussionItems.length; i++) {
    const item = closeDiscussionItems[i];
    core.info(`Processing close-discussion item ${i + 1}/${closeDiscussionItems.length}: bodyLength=${item.body.length}`);

    // Determine the discussion number
    let discussionNumber;

    if (target === "*") {
      // For target "*", we need an explicit number from the item
      const targetNumber = item.discussion_number;
      if (targetNumber) {
        discussionNumber = parseInt(targetNumber, 10);
        if (isNaN(discussionNumber) || discussionNumber <= 0) {
          core.info(`Invalid discussion number specified: ${targetNumber}`);
          continue;
        }
      } else {
        core.info(`Target is "*" but no discussion_number specified in close-discussion item`);
        continue;
      }
    } else if (target && target !== "triggering") {
      // Explicit number specified in target configuration
      discussionNumber = parseInt(target, 10);
      if (isNaN(discussionNumber) || discussionNumber <= 0) {
        core.info(`Invalid discussion number in target configuration: ${target}`);
        continue;
      }
    } else {
      // Default behavior: use triggering discussion
      if (isDiscussionContext) {
        discussionNumber = context.payload.discussion?.number;
        if (!discussionNumber) {
          core.info("Discussion context detected but no discussion found in payload");
          continue;
        }
      } else {
        core.info("Not in discussion context and no explicit target specified");
        continue;
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
          continue;
        }
      }

      // Apply title prefix filter
      if (requiredTitlePrefix && !discussion.title.startsWith(requiredTitlePrefix)) {
        core.info(`Discussion #${discussionNumber} does not have required title prefix: ${requiredTitlePrefix}`);
        continue;
      }

      // Apply category filter
      if (requiredCategory && discussion.category.name !== requiredCategory) {
        core.info(`Discussion #${discussionNumber} is not in required category: ${requiredCategory}`);
        continue;
      }

      // Extract body from the JSON item
      let body = item.body.trim();

      // Add AI disclaimer with workflow name and run url
      const workflowName = process.env.GH_AW_WORKFLOW_NAME || "Workflow";
      const workflowSource = process.env.GH_AW_WORKFLOW_SOURCE || "";
      const workflowSourceURL = process.env.GH_AW_WORKFLOW_SOURCE_URL || "";
      const runId = context.runId;
      const githubServer = process.env.GITHUB_SERVER_URL || "https://github.com";
      const runUrl = context.payload.repository ? `${context.payload.repository.html_url}/actions/runs/${runId}` : `${githubServer}/${context.repo.owner}/${context.repo.repo}/actions/runs/${runId}`;

      // Add fingerprint comment if present
      body += getTrackerID("markdown");

      body += generateFooter(workflowName, runUrl, workflowSource, workflowSourceURL, undefined, undefined, triggeringDiscussionNumber);

      core.info(`Adding comment to discussion #${discussionNumber}`);
      core.info(`Comment content length: ${body.length}`);

      // Add comment first
      const comment = await addDiscussionComment(github, discussion.id, body);
      core.info("Added discussion comment: " + comment.url);

      // Then close the discussion
      core.info(`Closing discussion #${discussionNumber} with reason: ${item.reason || "none"}`);
      const closedDiscussion = await closeDiscussion(github, discussion.id, item.reason);
      core.info("Closed discussion: " + closedDiscussion.url);

      closedDiscussions.push({
        number: discussionNumber,
        url: discussion.url,
        comment_url: comment.url,
      });

      // Set output for the last closed discussion (for backward compatibility)
      if (i === closeDiscussionItems.length - 1) {
        core.setOutput("discussion_number", discussionNumber);
        core.setOutput("discussion_url", discussion.url);
        core.setOutput("comment_url", comment.url);
      }
    } catch (error) {
      core.error(`✗ Failed to close discussion #${discussionNumber}: ${error instanceof Error ? error.message : String(error)}`);
      throw error;
    }
  }

  // Write summary for all closed discussions
  if (closedDiscussions.length > 0) {
    let summaryContent = "\n\n## Closed Discussions\n";
    for (const discussion of closedDiscussions) {
      summaryContent += `- Discussion #${discussion.number}: [View Discussion](${discussion.url})\n`;
      summaryContent += `  - Comment: [View Comment](${discussion.comment_url})\n`;
    }
    await core.summary.addRaw(summaryContent).write();
  }

  core.info(`Successfully closed ${closedDiscussions.length} discussion(s)`);
  return closedDiscussions;
}
await main();
