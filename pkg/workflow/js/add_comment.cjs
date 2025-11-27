// @ts-check
/// <reference types="@actions/github-script" />

const { runSafeOutput } = require("./safe_output_runner.cjs");
const { generateFooter } = require("./generate_footer.cjs");
const { getTrackerID } = require("./get_tracker_id.cjs");
const { getRepositoryUrl } = require("./get_repository_url.cjs");
const { replaceTemporaryIdReferences, loadTemporaryIdMap } = require("./temporary_id.cjs");

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
  let result;
  if (replyToId) {
    // Create a threaded reply to an existing comment
    result = await github.graphql(
      `
      mutation($dId: ID!, $body: String!, $replyToId: ID!) {
        addDiscussionComment(input: { discussionId: $dId, body: $body, replyToId: $replyToId }) {
          comment { 
            id 
            body 
            createdAt 
            url
          }
        }
      }`,
      { dId: discussionId, body: message, replyToId }
    );
  } else {
    // Create a top-level comment on the discussion
    result = await github.graphql(
      `
      mutation($dId: ID!, $body: String!) {
        addDiscussionComment(input: { discussionId: $dId, body: $body }) {
          comment { 
            id 
            body 
            createdAt 
            url
          }
        }
      }`,
      { dId: discussionId, body: message }
    );
  }

  const comment = result.addDiscussionComment.comment;

  return {
    id: comment.id,
    html_url: comment.url,
    discussion_url: discussionUrl,
  };
}

/**
 * Render function for staged preview
 * @param {any} item - The add_comment item
 * @param {number} index - Index of the item
 * @returns {string} Markdown content for the preview
 */
function renderAddCommentPreview(item, index) {
  const isDiscussionExplicit = process.env.GITHUB_AW_COMMENT_DISCUSSION === "true";
  const isDiscussionContext = context.eventName === "discussion" || context.eventName === "discussion_comment";
  const isDiscussion = isDiscussionContext || isDiscussionExplicit;

  let content = `### Comment ${index + 1}\n`;
  const targetNumber = item.item_number;
  if (targetNumber) {
    const repoUrl = getRepositoryUrl();
    if (isDiscussion) {
      const discussionUrl = `${repoUrl}/discussions/${targetNumber}`;
      content += `**Target Discussion:** [#${targetNumber}](${discussionUrl})\n\n`;
    } else {
      const issueUrl = `${repoUrl}/issues/${targetNumber}`;
      content += `**Target Issue:** [#${targetNumber}](${issueUrl})\n\n`;
    }
  } else {
    if (isDiscussion) {
      content += `**Target:** Current discussion\n\n`;
    } else {
      content += `**Target:** Current issue/PR\n\n`;
    }
  }
  content += `**Body:**\n${item.body || "No content provided"}\n\n`;
  return content;
}

/**
 * Generate staged summary header with related items
 * @returns {string} Markdown header for staged preview
 */
function getRelatedItemsHeader() {
  const createdIssueUrl = process.env.GH_AW_CREATED_ISSUE_URL;
  const createdIssueNumber = process.env.GH_AW_CREATED_ISSUE_NUMBER;
  const createdDiscussionUrl = process.env.GH_AW_CREATED_DISCUSSION_URL;
  const createdDiscussionNumber = process.env.GH_AW_CREATED_DISCUSSION_NUMBER;
  const createdPullRequestUrl = process.env.GH_AW_CREATED_PULL_REQUEST_URL;
  const createdPullRequestNumber = process.env.GH_AW_CREATED_PULL_REQUEST_NUMBER;

  let header = "";
  if (createdIssueUrl || createdDiscussionUrl || createdPullRequestUrl) {
    header += "#### Related Items\n\n";
    if (createdIssueUrl && createdIssueNumber) {
      header += `- Issue: [#${createdIssueNumber}](${createdIssueUrl})\n`;
    }
    if (createdDiscussionUrl && createdDiscussionNumber) {
      header += `- Discussion: [#${createdDiscussionNumber}](${createdDiscussionUrl})\n`;
    }
    if (createdPullRequestUrl && createdPullRequestNumber) {
      header += `- Pull Request: [#${createdPullRequestNumber}](${createdPullRequestUrl})\n`;
    }
    header += "\n";
  }
  return header;
}

/**
 * Process add_comment items
 * @param {any[]} commentItems - The add_comment items to process
 */
async function processCommentItems(commentItems) {
  const isDiscussionExplicit = process.env.GITHUB_AW_COMMENT_DISCUSSION === "true";

  // Load the temporary ID map from create_issue job
  const temporaryIdMap = loadTemporaryIdMap();
  if (temporaryIdMap.size > 0) {
    core.info(`Loaded temporary ID map with ${temporaryIdMap.size} entries`);
  }

  // Helper function to get the target number (issue, discussion, or pull request)
  function getTargetNumber(item) {
    return item.item_number;
  }

  // Get the target configuration from environment variable
  const commentTarget = process.env.GH_AW_COMMENT_TARGET || "triggering";
  core.info(`Comment target configuration: ${commentTarget}`);

  // Check if we're in an issue, pull request, or discussion context
  const isIssueContext = context.eventName === "issues" || context.eventName === "issue_comment";
  const isPRContext =
    context.eventName === "pull_request" ||
    context.eventName === "pull_request_review" ||
    context.eventName === "pull_request_review_comment";
  const isDiscussionContext = context.eventName === "discussion" || context.eventName === "discussion_comment";
  const isDiscussion = isDiscussionContext || isDiscussionExplicit;

  // Validate context based on target configuration
  if (commentTarget === "triggering" && !isIssueContext && !isPRContext && !isDiscussionContext) {
    core.info('Target is "triggering" but not running in issue, pull request, or discussion context, skipping comment creation');
    return;
  }

  // Extract triggering context for footer generation
  const triggeringIssueNumber =
    context.payload?.issue?.number && !context.payload?.issue?.pull_request ? context.payload.issue.number : undefined;
  const triggeringPRNumber =
    context.payload?.pull_request?.number || (context.payload?.issue?.pull_request ? context.payload.issue.number : undefined);
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
    let hasReferences = false;
    let referencesSection = "\n\n#### Related Items\n\n";

    if (createdIssueUrl && createdIssueNumber) {
      referencesSection += `- Issue: [#${createdIssueNumber}](${createdIssueUrl})\n`;
      hasReferences = true;
    }
    if (createdDiscussionUrl && createdDiscussionNumber) {
      referencesSection += `- Discussion: [#${createdDiscussionNumber}](${createdDiscussionUrl})\n`;
      hasReferences = true;
    }
    if (createdPullRequestUrl && createdPullRequestNumber) {
      referencesSection += `- Pull Request: [#${createdPullRequestNumber}](${createdPullRequestUrl})\n`;
      hasReferences = true;
    }

    if (hasReferences) {
      body += referencesSection;
    }

    // Add AI disclaimer with workflow name and run url
    const workflowName = process.env.GH_AW_WORKFLOW_NAME || "Workflow";
    const workflowSource = process.env.GH_AW_WORKFLOW_SOURCE || "";
    const workflowSourceURL = process.env.GH_AW_WORKFLOW_SOURCE_URL || "";
    const runId = context.runId;
    const githubServer = process.env.GITHUB_SERVER_URL || "https://github.com";
    const runUrl = context.payload.repository
      ? `${context.payload.repository.html_url}/actions/runs/${runId}`
      : `${githubServer}/${context.repo.owner}/${context.repo.repo}/actions/runs/${runId}`;

    // Add fingerprint comment if present
    body += getTrackerID("markdown");

    body += generateFooter(
      workflowName,
      runUrl,
      workflowSource,
      workflowSourceURL,
      triggeringIssueNumber,
      triggeringPRNumber,
      triggeringDiscussionNumber
    );

    try {
      let comment;

      // Use GraphQL API for discussions, REST API for issues/PRs
      if (commentEndpoint === "discussions") {
        core.info(`Creating comment on discussion #${itemNumber}`);
        core.info(`Comment content length: ${body.length}`);

        // For discussion_comment events, extract the comment node_id to create a threaded reply
        let replyToId;
        if (context.eventName === "discussion_comment" && context.payload?.comment?.node_id) {
          replyToId = context.payload.comment.node_id;
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
    } catch (error) {
      core.error(`✗ Failed to create comment: ${error instanceof Error ? error.message : String(error)}`);
      throw error;
    }
  }

  // Write summary for all created comments
  if (createdComments.length > 0) {
    let summaryContent = "\n\n## GitHub Comments\n";
    for (const comment of createdComments) {
      summaryContent += `- Comment #${comment.id}: [View Comment](${comment.html_url})\n`;
    }
    await core.summary.addRaw(summaryContent).write();
  }

  core.info(`Successfully created ${createdComments.length} comment(s)`);
  return createdComments;
}

async function main() {
  await runSafeOutput({
    itemType: "add_comment",
    itemTypePlural: "add-comment",
    stagedTitle: "Add Comments",
    stagedDescription: getRelatedItemsHeader() + "The following comments would be added if staged mode was disabled:",
    renderStagedItem: renderAddCommentPreview,
    processItems: processCommentItems,
  });
}

await main();
