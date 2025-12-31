// @ts-check
/// <reference types="@actions/github-script" />

const { loadAgentOutput } = require("./load_agent_output.cjs");
const { generateFooterWithMessages } = require("./messages_footer.cjs");
const { getRepositoryUrl } = require("./get_repository_url.cjs");
const { replaceTemporaryIdReferences, loadTemporaryIdMap } = require("./temporary_id.cjs");
const { getTrackerID } = require("./get_tracker_id.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Hide/minimize a comment using the GraphQL API
 * @param {any} github - GitHub GraphQL instance
 * @param {string} nodeId - Comment node ID
 * @param {string} reason - Reason for hiding (default: outdated)
 * @returns {Promise<{id: string, isMinimized: boolean}>}
 */
async function minimizeComment(github, nodeId, reason = "outdated") {
  const query = /* GraphQL */ `
    mutation ($nodeId: ID!, $classifier: ReportedContentClassifiers!) {
      minimizeComment(input: { subjectId: $nodeId, classifier: $classifier }) {
        minimizedComment {
          isMinimized
        }
      }
    }
  `;

  const result = await github.graphql(query, { nodeId, classifier: reason });

  return {
    id: nodeId,
    isMinimized: result.minimizeComment.minimizedComment.isMinimized,
  };
}

/**
 * Find comments on an issue/PR with a specific tracker-id
 * @param {any} github - GitHub REST API instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} issueNumber - Issue/PR number
 * @param {string} workflowId - Workflow ID to search for
 * @returns {Promise<Array<{id: number, node_id: string, body: string}>>}
 */
async function findCommentsWithTrackerId(github, owner, repo, issueNumber, workflowId) {
  const comments = [];
  let page = 1;
  const perPage = 100;

  // Paginate through all comments
  while (true) {
    const { data } = await github.rest.issues.listComments({
      owner,
      repo,
      issue_number: issueNumber,
      per_page: perPage,
      page,
    });

    if (data.length === 0) {
      break;
    }

    // Filter comments that contain the workflow-id and are NOT reaction comments
    const filteredComments = data.filter(comment => comment.body?.includes(`<!-- workflow-id: ${workflowId} -->`) && !comment.body.includes(`<!-- comment-type: reaction -->`)).map(({ id, node_id, body }) => ({ id, node_id, body }));

    comments.push(...filteredComments);

    if (data.length < perPage) {
      break;
    }

    page++;
  }

  return comments;
}

/**
 * Find comments on a discussion with a specific workflow ID
 * @param {any} github - GitHub GraphQL instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} discussionNumber - Discussion number
 * @param {string} workflowId - Workflow ID to search for
 * @returns {Promise<Array<{id: string, body: string}>>}
 */
async function findDiscussionCommentsWithTrackerId(github, owner, repo, discussionNumber, workflowId) {
  const query = /* GraphQL */ `
    query ($owner: String!, $repo: String!, $num: Int!, $cursor: String) {
      repository(owner: $owner, name: $repo) {
        discussion(number: $num) {
          comments(first: 100, after: $cursor) {
            nodes {
              id
              body
            }
            pageInfo {
              hasNextPage
              endCursor
            }
          }
        }
      }
    }
  `;

  const comments = [];
  let cursor = null;

  while (true) {
    const result = await github.graphql(query, { owner, repo, num: discussionNumber, cursor });

    if (!result.repository?.discussion?.comments?.nodes) {
      break;
    }

    const filteredComments = result.repository.discussion.comments.nodes
      .filter(comment => comment.body?.includes(`<!-- workflow-id: ${workflowId} -->`) && !comment.body.includes(`<!-- comment-type: reaction -->`))
      .map(({ id, body }) => ({ id, body }));

    comments.push(...filteredComments);

    if (!result.repository.discussion.comments.pageInfo.hasNextPage) {
      break;
    }

    cursor = result.repository.discussion.comments.pageInfo.endCursor;
  }

  return comments;
}

/**
 * Hide all previous comments from the same workflow
 * @param {any} github - GitHub API instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} itemNumber - Issue/PR/Discussion number
 * @param {string} workflowId - Workflow ID to match
 * @param {boolean} isDiscussion - Whether this is a discussion
 * @param {string} reason - Reason for hiding (default: outdated)
 * @param {string[] | null} allowedReasons - List of allowed reasons (default: null for all)
 * @returns {Promise<number>} Number of comments hidden
 */
async function hideOlderComments(github, owner, repo, itemNumber, workflowId, isDiscussion, reason = "outdated", allowedReasons = null) {
  if (!workflowId) {
    core.info("No workflow ID available, skipping hide-older-comments");
    return 0;
  }

  // Normalize reason to uppercase for GitHub API
  const normalizedReason = reason.toUpperCase();

  // Validate reason against allowed reasons if specified (case-insensitive)
  if (allowedReasons && allowedReasons.length > 0) {
    const normalizedAllowedReasons = allowedReasons.map(r => r.toUpperCase());
    if (!normalizedAllowedReasons.includes(normalizedReason)) {
      core.warning(`Reason "${reason}" is not in allowed-reasons list [${allowedReasons.join(", ")}]. Skipping hide-older-comments.`);
      return 0;
    }
  }

  core.info(`Searching for previous comments with workflow ID: ${workflowId}`);

  let comments;
  if (isDiscussion) {
    comments = await findDiscussionCommentsWithTrackerId(github, owner, repo, itemNumber, workflowId);
  } else {
    comments = await findCommentsWithTrackerId(github, owner, repo, itemNumber, workflowId);
  }

  if (comments.length === 0) {
    core.info("No previous comments found with matching workflow ID");
    return 0;
  }

  core.info(`Found ${comments.length} previous comment(s) to hide with reason: ${normalizedReason}`);

  let hiddenCount = 0;
  for (const comment of comments) {
    // TypeScript can't narrow the union type here, but we know it's safe due to isDiscussion check
    // @ts-expect-error - comment has node_id when not a discussion
    const nodeId = isDiscussion ? String(comment.id) : comment.node_id;
    core.info(`Hiding comment: ${nodeId}`);

    const result = await minimizeComment(github, nodeId, normalizedReason);
    hiddenCount++;
    core.info(`✓ Hidden comment: ${nodeId}`);
  }

  core.info(`Successfully hidden ${hiddenCount} comment(s)`);
  return hiddenCount;
}

/**
 * Comment on a GitHub Discussion using GraphQL
 * @param {any} github - GitHub REST API instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} discussionNumber - Discussion number
 * @param {string} message - Comment body
 * @param {string|undefined} replyToId - Optional comment node ID to reply to (for threaded comments)
 * @returns {Promise<{id: string, html_url: string, discussion_url: string}>} Comment details
 */
async function commentOnDiscussion(github, owner, repo, discussionNumber, message, replyToId) {
  // 1. Retrieve discussion node ID
  const { repository } = await github.graphql(
    `
    query($owner: String!, $repo: String!, $num: Int!) {
      repository(owner: $owner, name: $repo) {
        discussion(number: $num) { 
          id 
          url
        }
      }
    }`,
    { owner, repo, num: discussionNumber }
  );

  if (!repository || !repository.discussion) {
    throw new Error(`Discussion #${discussionNumber} not found in ${owner}/${repo}`);
  }

  const discussionId = repository.discussion.id;
  const discussionUrl = repository.discussion.url;

  // 2. Add comment (with optional replyToId for threading)
  const mutation = replyToId
    ? `mutation($dId: ID!, $body: String!, $replyToId: ID!) {
        addDiscussionComment(input: { discussionId: $dId, body: $body, replyToId: $replyToId }) {
          comment { 
            id 
            body 
            createdAt 
            url
          }
        }
      }`
    : `mutation($dId: ID!, $body: String!) {
        addDiscussionComment(input: { discussionId: $dId, body: $body }) {
          comment { 
            id 
            body 
            createdAt 
            url
          }
        }
      }`;

  const variables = replyToId ? { dId: discussionId, body: message, replyToId } : { dId: discussionId, body: message };

  const result = await github.graphql(mutation, variables);

  const comment = result.addDiscussionComment.comment;

  return {
    id: comment.id,
    html_url: comment.url,
    discussion_url: discussionUrl,
  };
}

async function main(config = {}) {
  // Check if we're in staged mode
  const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true";
  const isDiscussionExplicit = process.env.GITHUB_AW_COMMENT_DISCUSSION === "true";
  const hideOlderCommentsEnabled = config.hide_older_comments === true;

  // Load the temporary ID map from create_issue job
  const temporaryIdMap = loadTemporaryIdMap();
  if (temporaryIdMap.size > 0) {
    core.info(`Loaded temporary ID map with ${temporaryIdMap.size} entries`);
  }

  const result = loadAgentOutput();
  if (!result.success) {
    return;
  }

  // Find all add-comment items
  const commentItems = result.items.filter(/** @param {any} item */ item => item.type === "add_comment");
  if (commentItems.length === 0) {
    core.info("No add-comment items found in agent output");
    return;
  }

  core.info(`Found ${commentItems.length} add-comment item(s)`);

  // Helper function to get the target number (issue, discussion, or pull request)
  function getTargetNumber(item) {
    return item.item_number;
  }

  // Get the target configuration from config object
  const commentTarget = config.target || "triggering";
  core.info(`Comment target configuration: ${commentTarget}`);

  // Check if we're in an issue, pull request, or discussion context
  const isIssueContext = context.eventName === "issues" || context.eventName === "issue_comment";
  const isPRContext = context.eventName === "pull_request" || context.eventName === "pull_request_review" || context.eventName === "pull_request_review_comment";
  const isDiscussionContext = context.eventName === "discussion" || context.eventName === "discussion_comment";
  const isDiscussion = isDiscussionContext || isDiscussionExplicit;

  // Get workflow ID for hiding older comments
  // Use GITHUB_WORKFLOW environment variable which is automatically set by GitHub Actions
  const workflowId = process.env.GITHUB_WORKFLOW || "";

  // Parse allowed reasons from environment variable
  const allowedReasons = process.env.GH_AW_ALLOWED_REASONS
    ? (() => {
        try {
          const parsed = JSON.parse(process.env.GH_AW_ALLOWED_REASONS);
          core.info(`Allowed reasons for hiding: [${parsed.join(", ")}]`);
          return parsed;
        } catch (error) {
          core.warning(`Failed to parse GH_AW_ALLOWED_REASONS: ${getErrorMessage(error)}`);
          return null;
        }
      })()
    : null;

  if (hideOlderCommentsEnabled) {
    core.info(`Hide-older-comments is enabled with workflow ID: ${workflowId || "(none)"}`);
  }

  // If in staged mode, emit step summary instead of creating comments
  if (isStaged) {
    let summaryContent = "## 🎭 Staged Mode: Add Comments Preview\n\n";
    summaryContent += "The following comments would be added if staged mode was disabled:\n\n";

    // Show created items references if available
    const createdIssueUrl = process.env.GH_AW_CREATED_ISSUE_URL;
    const createdIssueNumber = process.env.GH_AW_CREATED_ISSUE_NUMBER;
    const createdDiscussionUrl = process.env.GH_AW_CREATED_DISCUSSION_URL;
    const createdDiscussionNumber = process.env.GH_AW_CREATED_DISCUSSION_NUMBER;
    const createdPullRequestUrl = process.env.GH_AW_CREATED_PULL_REQUEST_URL;
    const createdPullRequestNumber = process.env.GH_AW_CREATED_PULL_REQUEST_NUMBER;

    if (createdIssueUrl || createdDiscussionUrl || createdPullRequestUrl) {
      summaryContent += "#### Related Items\n\n";
      if (createdIssueUrl && createdIssueNumber) {
        summaryContent += `- Issue: [#${createdIssueNumber}](${createdIssueUrl})\n`;
      }
      if (createdDiscussionUrl && createdDiscussionNumber) {
        summaryContent += `- Discussion: [#${createdDiscussionNumber}](${createdDiscussionUrl})\n`;
      }
      if (createdPullRequestUrl && createdPullRequestNumber) {
        summaryContent += `- Pull Request: [#${createdPullRequestNumber}](${createdPullRequestUrl})\n`;
      }
      summaryContent += "\n";
    }

    for (let i = 0; i < commentItems.length; i++) {
      const item = commentItems[i];
      summaryContent += `### Comment ${i + 1}\n`;
      const targetNumber = getTargetNumber(item);
      if (targetNumber) {
        const repoUrl = getRepositoryUrl();
        if (isDiscussion) {
          const discussionUrl = `${repoUrl}/discussions/${targetNumber}`;
          summaryContent += `**Target Discussion:** [#${targetNumber}](${discussionUrl})\n\n`;
        } else {
          const issueUrl = `${repoUrl}/issues/${targetNumber}`;
          summaryContent += `**Target Issue:** [#${targetNumber}](${issueUrl})\n\n`;
        }
      } else {
        if (isDiscussion) {
          summaryContent += `**Target:** Current discussion\n\n`;
        } else {
          summaryContent += `**Target:** Current issue/PR\n\n`;
        }
      }
      summaryContent += `**Body:**\n${item.body || "No content provided"}\n\n`;
      summaryContent += "---\n\n";
    }

    // Write to step summary
    await core.summary.addRaw(summaryContent).write();
    core.info("📝 Comment creation preview written to step summary");
    return;
  }

  // Validate context based on target configuration
  if (commentTarget === "triggering" && !isIssueContext && !isPRContext && !isDiscussionContext) {
    core.info('Target is "triggering" but not running in issue, pull request, or discussion context, skipping comment creation');
    return;
  }

  // Extract triggering context for footer generation
  const triggeringIssueNumber = context.payload?.issue?.number && !context.payload?.issue?.pull_request ? context.payload.issue.number : undefined;
  const triggeringPRNumber = context.payload?.pull_request?.number || (context.payload?.issue?.pull_request ? context.payload.issue.number : undefined);
  const triggeringDiscussionNumber = context.payload?.discussion?.number;

  const createdComments = [];

  // Process each comment item
  for (let i = 0; i < commentItems.length; i++) {
    const commentItem = commentItems[i];
    core.info(`Processing add-comment item ${i + 1}/${commentItems.length}: bodyLength=${commentItem.body.length}`);

    // Determine the issue/PR number and comment endpoint for this comment
    let itemNumber;
    let commentEndpoint;

    if (commentTarget === "*") {
      // For target "*", we need an explicit number from the comment item
      const targetNumber = getTargetNumber(commentItem);
      if (targetNumber) {
        itemNumber = parseInt(targetNumber, 10);
        if (isNaN(itemNumber) || itemNumber <= 0) {
          core.info(`Invalid target number specified: ${targetNumber}`);
          continue;
        }
        commentEndpoint = isDiscussion ? "discussions" : "issues";
      } else {
        core.info(`Target is "*" but no number specified in comment item`);
        continue;
      }
    } else if (commentTarget && commentTarget !== "triggering") {
      // Explicit number specified in target configuration
      itemNumber = parseInt(commentTarget, 10);
      if (isNaN(itemNumber) || itemNumber <= 0) {
        core.info(`Invalid target number in target configuration: ${commentTarget}`);
        continue;
      }
      commentEndpoint = isDiscussion ? "discussions" : "issues";
    } else {
      // Default behavior: use triggering issue/PR/discussion
      if (isIssueContext) {
        itemNumber = context.payload.issue?.number || context.payload.pull_request?.number || context.payload.discussion?.number;
        if (context.payload.issue) {
          commentEndpoint = "issues";
        } else {
          core.info("Issue context detected but no issue found in payload");
          continue;
        }
      } else if (isPRContext) {
        itemNumber = context.payload.pull_request?.number || context.payload.issue?.number || context.payload.discussion?.number;
        if (context.payload.pull_request) {
          commentEndpoint = "issues"; // PR comments use the issues API endpoint
        } else {
          core.info("Pull request context detected but no pull request found in payload");
          continue;
        }
      } else if (isDiscussionContext) {
        itemNumber = context.payload.discussion?.number || context.payload.issue?.number || context.payload.pull_request?.number;
        if (context.payload.discussion) {
          commentEndpoint = "discussions"; // Discussion comments use GraphQL via commentOnDiscussion
        } else {
          core.info("Discussion context detected but no discussion found in payload");
          continue;
        }
      }
    }

    if (!itemNumber) {
      core.info("Could not determine issue, pull request, or discussion number");
      continue;
    }

    // Extract body from the JSON item and replace temporary ID references
    let body = replaceTemporaryIdReferences(commentItem.body.trim(), temporaryIdMap);

    // Append references to created issues, discussions, and pull requests if they exist
    const createdIssueUrl = process.env.GH_AW_CREATED_ISSUE_URL;
    const createdIssueNumber = process.env.GH_AW_CREATED_ISSUE_NUMBER;
    const createdDiscussionUrl = process.env.GH_AW_CREATED_DISCUSSION_URL;
    const createdDiscussionNumber = process.env.GH_AW_CREATED_DISCUSSION_NUMBER;
    const createdPullRequestUrl = process.env.GH_AW_CREATED_PULL_REQUEST_URL;
    const createdPullRequestNumber = process.env.GH_AW_CREATED_PULL_REQUEST_NUMBER;

    // Add references section if any URLs are available
    const references = [
      createdIssueUrl && createdIssueNumber && `- Issue: [#${createdIssueNumber}](${createdIssueUrl})`,
      createdDiscussionUrl && createdDiscussionNumber && `- Discussion: [#${createdDiscussionNumber}](${createdDiscussionUrl})`,
      createdPullRequestUrl && createdPullRequestNumber && `- Pull Request: [#${createdPullRequestNumber}](${createdPullRequestUrl})`,
    ].filter(Boolean);

    if (references.length > 0) {
      body += `\n\n#### Related Items\n\n${references.join("\n")}\n`;
    }

    // Add AI disclaimer with workflow name and run url
    const workflowName = process.env.GH_AW_WORKFLOW_NAME || "Workflow";
    const workflowSource = process.env.GH_AW_WORKFLOW_SOURCE || "";
    const workflowSourceURL = process.env.GH_AW_WORKFLOW_SOURCE_URL || "";
    const runId = context.runId;
    const githubServer = process.env.GITHUB_SERVER_URL || "https://github.com";
    const runUrl = context.payload.repository ? `${context.payload.repository.html_url}/actions/runs/${runId}` : `${githubServer}/${context.repo.owner}/${context.repo.repo}/actions/runs/${runId}`;

    // Add workflow ID comment marker if present
    if (workflowId) {
      body += `\n\n<!-- workflow-id: ${workflowId} -->`;
    }

    // Add tracker-id comment if present
    const trackerIDComment = getTrackerID("markdown");
    if (trackerIDComment) {
      body += trackerIDComment;
    }

    // Add comment type marker to identify this as an add-comment
    body += `\n\n<!-- comment-type: add-comment -->`;

    body += generateFooterWithMessages(workflowName, runUrl, workflowSource, workflowSourceURL, triggeringIssueNumber, triggeringPRNumber, triggeringDiscussionNumber);

    // Hide older comments from the same workflow if enabled
    if (hideOlderCommentsEnabled && workflowId) {
      core.info("Hide-older-comments is enabled, searching for previous comments to hide");
      await hideOlderComments(github, context.repo.owner, context.repo.repo, itemNumber, workflowId, commentEndpoint === "discussions", "outdated", allowedReasons);
    }

    let comment;

    // Use GraphQL API for discussions, REST API for issues/PRs
    if (commentEndpoint === "discussions") {
      core.info(`Creating comment on discussion #${itemNumber}`);
      core.info(`Comment content length: ${body.length}`);

      // For discussion_comment events, extract the comment node_id to create a threaded reply
      const replyToId = context.eventName === "discussion_comment" && context.payload?.comment?.node_id ? context.payload.comment.node_id : undefined;

      if (replyToId) {
        core.info(`Creating threaded reply to comment ${replyToId}`);
      }

      // Create discussion comment using GraphQL
      comment = await commentOnDiscussion(github, context.repo.owner, context.repo.repo, itemNumber, body, replyToId);
      core.info("Created discussion comment #" + comment.id + ": " + comment.html_url);

      // Add discussion_url to the comment object for consistency
      comment.discussion_url = comment.discussion_url;
    } else {
      core.info(`Creating comment on ${commentEndpoint} #${itemNumber}`);
      core.info(`Comment content length: ${body.length}`);

      // Create regular issue/PR comment using REST API
      const { data: restComment } = await github.rest.issues.createComment({
        owner: context.repo.owner,
        repo: context.repo.repo,
        issue_number: itemNumber,
        body: body,
      });

      comment = restComment;
      core.info("Created comment #" + comment.id + ": " + comment.html_url);
    }

    createdComments.push(comment);

    // Set output for the last created comment (for backward compatibility)
    if (i === commentItems.length - 1) {
      core.setOutput("comment_id", comment.id);
      core.setOutput("comment_url", comment.html_url);
    }

    // Add metadata for tracking (includes comment ID, item number, and repo info)
    // This is used by the handler manager to track comments with unresolved temp IDs
    try {
      // @ts-ignore - Adding tracking metadata dynamically
      comment._tracking = {
        commentId: comment.id,
        itemNumber: itemNumber,
        repo: `${context.repo.owner}/${context.repo.repo}`,
        isDiscussion: commentEndpoint === "discussions",
      };
    } catch (error) {
      // Silently ignore tracking errors to not break existing functionality
      core.debug(`Failed to add tracking metadata: ${getErrorMessage(error)}`);
    }
  }

  // Write summary for all created comments
  if (createdComments.length > 0) {
    const summaryContent = "\n\n## GitHub Comments\n" + createdComments.map(c => `- Comment #${c.id}: [View Comment](${c.html_url})`).join("\n");
    await core.summary.addRaw(summaryContent).write();
  }

  core.info(`Successfully created ${createdComments.length} comment(s)`);
  return createdComments;
}

module.exports = { main };
