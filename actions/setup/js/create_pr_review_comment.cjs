// @ts-check
/// <reference types="@actions/github-script" />

const { loadAgentOutput } = require("./load_agent_output.cjs");
const { generateStagedPreview } = require("./staged_preview.cjs");
const { generateFooter } = require("./generate_footer.cjs");
const { getRepositoryUrl } = require("./get_repository_url.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");

async function main() {
  // Check if we're in staged mode
  const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true";

  const result = loadAgentOutput();
  if (!result.success) {
    return;
  }

  // Find all create-pr-review-comment items
  const reviewCommentItems = result.items.filter(/** @param {any} item */ item => item.type === "create_pull_request_review_comment");
  if (reviewCommentItems.length === 0) {
    core.info("No create-pull-request-review-comment items found in agent output");
    return;
  }

  core.info(`Found ${reviewCommentItems.length} create-pull-request-review-comment item(s)`);

  // If in staged mode, emit step summary instead of creating review comments
  if (isStaged) {
    await generateStagedPreview({
      title: "Create PR Review Comments",
      description: "The following review comments would be created if staged mode was disabled:",
      items: reviewCommentItems,
      renderItem: (item, index) => {
        let content = `#### Review Comment ${index + 1}\n`;
        if (item.pull_request_number) {
          const repoUrl = getRepositoryUrl();
          const pullUrl = `${repoUrl}/pull/${item.pull_request_number}`;
          content += `**Target PR:** [#${item.pull_request_number}](${pullUrl})\n\n`;
        } else {
          content += `**Target:** Current PR\n\n`;
        }
        content += `**File:** ${item.path || "No path provided"}\n\n`;
        content += `**Line:** ${item.line || "No line provided"}\n\n`;
        if (item.start_line) {
          content += `**Start Line:** ${item.start_line}\n\n`;
        }
        content += `**Side:** ${item.side || "RIGHT"}\n\n`;
        content += `**Body:**\n${item.body || "No content provided"}\n\n`;
        return content;
      },
    });
    return;
  }

  // Get the side configuration from environment variable
  const defaultSide = process.env.GH_AW_PR_REVIEW_COMMENT_SIDE || "RIGHT";
  core.info(`Default comment side configuration: ${defaultSide}`);

  // Get the target configuration from environment variable
  const commentTarget = process.env.GH_AW_PR_REVIEW_COMMENT_TARGET || "triggering";
  core.info(`PR review comment target configuration: ${commentTarget}`);

  // Check if we're in a pull request context, or an issue comment context on a PR
  const isPRContext =
    context.eventName === "pull_request" ||
    context.eventName === "pull_request_review" ||
    context.eventName === "pull_request_review_comment" ||
    (context.eventName === "issue_comment" && context.payload.issue && context.payload.issue.pull_request);

  // Validate context based on target configuration
  if (commentTarget === "triggering" && !isPRContext) {
    core.info('Target is "triggering" but not running in pull request context, skipping review comment creation');
    return;
  }

  // Extract triggering context for footer generation
  const triggeringIssueNumber = context.payload?.issue?.number && !context.payload?.issue?.pull_request ? context.payload.issue.number : undefined;
  const triggeringPRNumber = context.payload?.pull_request?.number || (context.payload?.issue?.pull_request ? context.payload.issue.number : undefined);
  const triggeringDiscussionNumber = context.payload?.discussion?.number;

  const createdComments = [];

  // Process each review comment item
  for (let i = 0; i < reviewCommentItems.length; i++) {
    const commentItem = reviewCommentItems[i];
    core.info(
      `Processing create-pull-request-review-comment item ${i + 1}/${reviewCommentItems.length}: bodyLength=${commentItem.body ? commentItem.body.length : "undefined"}, path=${commentItem.path}, line=${commentItem.line}, startLine=${commentItem.start_line}`
    );

    // Validate required fields
    if (!commentItem.path) {
      core.info('Missing required field "path" in review comment item');
      continue;
    }

    if (!commentItem.line || (typeof commentItem.line !== "number" && typeof commentItem.line !== "string")) {
      core.info('Missing or invalid required field "line" in review comment item');
      continue;
    }

    if (!commentItem.body || typeof commentItem.body !== "string") {
      core.info('Missing or invalid required field "body" in review comment item');
      continue;
    }

    // Determine the PR number for this review comment
    let pullRequestNumber;
    let pullRequest;

    if (commentTarget === "*") {
      // For target "*", we need an explicit PR number from the comment item
      if (commentItem.pull_request_number) {
        pullRequestNumber = parseInt(commentItem.pull_request_number, 10);
        if (isNaN(pullRequestNumber) || pullRequestNumber <= 0) {
          core.info(`Invalid pull request number specified: ${commentItem.pull_request_number}`);
          continue;
        }
      } else {
        core.info('Target is "*" but no pull_request_number specified in comment item');
        continue;
      }
    } else if (commentTarget && commentTarget !== "triggering") {
      // Explicit PR number specified in target
      pullRequestNumber = parseInt(commentTarget, 10);
      if (isNaN(pullRequestNumber) || pullRequestNumber <= 0) {
        core.info(`Invalid pull request number in target configuration: ${commentTarget}`);
        continue;
      }
    } else {
      // Default behavior: use triggering PR
      if (context.payload.pull_request) {
        pullRequestNumber = context.payload.pull_request.number;
        pullRequest = context.payload.pull_request;
      } else if (context.payload.issue && context.payload.issue.pull_request) {
        pullRequestNumber = context.payload.issue.number;
      } else {
        core.info("Pull request context detected but no pull request found in payload");
        continue;
      }
    }

    if (!pullRequestNumber) {
      core.info("Could not determine pull request number");
      continue;
    }

    // If we don't have the full PR details yet, fetch them
    if (!pullRequest || !pullRequest.head || !pullRequest.head.sha) {
      try {
        const { data: fullPR } = await github.rest.pulls.get({
          owner: context.repo.owner,
          repo: context.repo.repo,
          pull_number: pullRequestNumber,
        });
        pullRequest = fullPR;
        core.info(`Fetched full pull request details for PR #${pullRequestNumber}`);
      } catch (error) {
        core.info(`Failed to fetch pull request details for PR #${pullRequestNumber}: ${getErrorMessage(error)}`);
        continue;
      }
    }

    // Check if we have the commit SHA needed for creating review comments
    if (!pullRequest || !pullRequest.head || !pullRequest.head.sha) {
      core.info(`Pull request head commit SHA not found for PR #${pullRequestNumber} - cannot create review comment`);
      continue;
    }

    core.info(`Creating review comment on PR #${pullRequestNumber}`);

    // Parse line numbers
    const line = parseInt(commentItem.line, 10);
    if (isNaN(line) || line <= 0) {
      core.info(`Invalid line number: ${commentItem.line}`);
      continue;
    }

    let startLine = undefined;
    if (commentItem.start_line) {
      startLine = parseInt(commentItem.start_line, 10);
      if (isNaN(startLine) || startLine <= 0 || startLine > line) {
        core.info(`Invalid start_line number: ${commentItem.start_line} (must be <= line: ${line})`);
        continue;
      }
    }

    // Determine side (LEFT or RIGHT)
    const side = commentItem.side || defaultSide;
    if (side !== "LEFT" && side !== "RIGHT") {
      core.info(`Invalid side value: ${side} (must be LEFT or RIGHT)`);
      continue;
    }

    // Extract body from the JSON item
    let body = commentItem.body.trim();

    // Add AI disclaimer with workflow name and run url
    const workflowName = process.env.GH_AW_WORKFLOW_NAME || "Workflow";
    const workflowSource = process.env.GH_AW_WORKFLOW_SOURCE || "";
    const workflowSourceURL = process.env.GH_AW_WORKFLOW_SOURCE_URL || "";
    const runId = context.runId;
    const githubServer = process.env.GITHUB_SERVER_URL || "https://github.com";
    const runUrl = context.payload.repository ? `${context.payload.repository.html_url}/actions/runs/${runId}` : `${githubServer}/${context.repo.owner}/${context.repo.repo}/actions/runs/${runId}`;
    body += generateFooter(workflowName, runUrl, workflowSource, workflowSourceURL, triggeringIssueNumber, triggeringPRNumber, triggeringDiscussionNumber);

    core.info(`Creating review comment on PR #${pullRequestNumber} at ${commentItem.path}:${line}${startLine ? ` (lines ${startLine}-${line})` : ""} [${side}]`);
    core.info(`Comment content length: ${body.length}`);

    try {
      // Prepare the request parameters
      /** @type {any} */
      const requestParams = {
        owner: context.repo.owner,
        repo: context.repo.repo,
        pull_number: pullRequestNumber,
        body: body,
        path: commentItem.path,
        commit_id: pullRequest && pullRequest.head ? pullRequest.head.sha : "", // Required for creating review comments
        line: line,
        side: side,
      };

      // Add start_line for multi-line comments
      if (startLine !== undefined) {
        requestParams.start_line = startLine;
        requestParams.start_side = side; // start_side should match side for consistency
      }

      // Create the review comment using GitHub API
      const { data: comment } = await github.rest.pulls.createReviewComment(requestParams);

      core.info("Created review comment #" + comment.id + ": " + comment.html_url);
      createdComments.push(comment);

      // Set output for the last created comment (for backward compatibility)
      if (i === reviewCommentItems.length - 1) {
        core.setOutput("review_comment_id", comment.id);
        core.setOutput("review_comment_url", comment.html_url);
      }
    } catch (error) {
      core.error(`✗ Failed to create review comment: ${getErrorMessage(error)}`);
      throw error;
    }
  }

  // Write summary for all created comments
  if (createdComments.length > 0) {
    let summaryContent = "\n\n## GitHub PR Review Comments\n";
    for (const comment of createdComments) {
      summaryContent += `- Review Comment #${comment.id}: [View Comment](${comment.html_url})\n`;
    }
    await core.summary.addRaw(summaryContent).write();
  }

  core.info(`Successfully created ${createdComments.length} review comment(s)`);
  return createdComments;
}

module.exports = { main };
