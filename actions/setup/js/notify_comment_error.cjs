// @ts-check
/// <reference types="@actions/github-script" />

// This script updates an existing comment created by the activation job
// to notify about the workflow completion status (success or failure).
// It also processes noop messages and adds them to the activation comment.

const { loadAgentOutput } = require("./load_agent_output.cjs");
const { getRunSuccessMessage, getRunFailureMessage, getDetectionFailureMessage, getDetectionWarningMessage } = require("./messages_run_status.cjs");
const { getMessages } = require("./messages_core.cjs");
const { getErrorMessage, isLockedError } = require("./error_helpers.cjs");
const { sanitizeContent } = require("./sanitize_content.cjs");
const { ERR_VALIDATION } = require("./error_codes.cjs");
const { parseBoolTemplatable } = require("./templatable.cjs");
const { resolveTopLevelDiscussionCommentId } = require("./github_api_helpers.cjs");

/**
 * Collect generated asset URLs from safe output jobs
 * @returns {Array<string>} Array of generated asset URLs
 */
function collectGeneratedAssets() {
  const assets = [];

  // Get the safe output jobs mapping from environment
  const safeOutputJobsEnv = process.env.GH_AW_SAFE_OUTPUT_JOBS;
  if (!safeOutputJobsEnv) {
    return assets;
  }

  let jobOutputMapping;
  try {
    jobOutputMapping = JSON.parse(safeOutputJobsEnv);
  } catch (error) {
    core.warning(`Failed to parse GH_AW_SAFE_OUTPUT_JOBS: ${getErrorMessage(error)}`);
    return assets;
  }

  // Iterate through each job and collect its URL output
  for (const [jobName, urlKey] of Object.entries(jobOutputMapping)) {
    // Access the job output using the GitHub Actions context
    // The value will be set as an environment variable in the format GH_AW_OUTPUT_<JOB>_<KEY>
    const envVarName = `GH_AW_OUTPUT_${jobName.toUpperCase()}_${urlKey.toUpperCase()}`;
    const url = process.env[envVarName];

    if (url && url.trim() !== "") {
      assets.push(url);
      core.info(`Collected asset URL: ${url}`);
    }
  }

  return assets;
}

/**
 * Attempt to add a "needs-review" label to the triggering issue/PR.
 * This is a best-effort operation — failures are logged but do not fail the workflow.
 * @param {string|undefined} commentRepo - Repository in "owner/repo" format
 */
async function tryAddNeedsReviewLabel(commentRepo) {
  try {
    const repoOwner = commentRepo ? commentRepo.split("/")[0] : context.repo.owner;
    const repoName = commentRepo ? commentRepo.split("/")[1] : context.repo.repo;
    const issueNumber = context.payload?.issue?.number || context.payload?.pull_request?.number;

    if (!issueNumber) {
      core.info("No issue/PR number found — skipping needs-review label");
      return;
    }

    core.info(`Adding "needs-review" label to ${repoOwner}/${repoName}#${issueNumber}`);
    await github.request("POST /repos/{owner}/{repo}/issues/{issue_number}/labels", {
      owner: repoOwner,
      repo: repoName,
      issue_number: issueNumber,
      labels: ["needs-review"],
      headers: {
        Accept: "application/vnd.github+json",
      },
    });
    core.info('Successfully added "needs-review" label');
  } catch (/** @type {any} */ error) {
    // Best-effort: label might not exist in the repo, or permissions may be insufficient
    core.warning(`Failed to add "needs-review" label: ${getErrorMessage(error)}`);
  }
}

async function main() {
  const commentId = process.env.GH_AW_COMMENT_ID;
  const commentRepo = process.env.GH_AW_COMMENT_REPO;
  const runUrl = process.env.GH_AW_RUN_URL;
  const workflowName = process.env.GH_AW_WORKFLOW_NAME || "Workflow";
  const agentConclusion = process.env.GH_AW_AGENT_CONCLUSION || "failure";
  const detectionConclusion = process.env.GH_AW_DETECTION_CONCLUSION;
  const detectionReason = process.env.GH_AW_DETECTION_REASON || "";
  const assignToAgentErrorCount = parseInt(process.env.GH_AW_ASSIGNMENT_ERROR_COUNT || "0", 10);

  const messagesConfig = getMessages();
  const appendOnlyComments = messagesConfig?.appendOnlyComments === true;

  // If activation comments are disabled entirely, skip all comment updates
  if (!parseBoolTemplatable(messagesConfig?.activationComments, true)) {
    core.info("activation-comments is disabled: skipping completion comment update");
    return;
  }

  core.info(`Comment ID: ${commentId}`);
  core.info(`Comment Repo: ${commentRepo}`);
  core.info(`Run URL: ${runUrl}`);
  core.info(`Workflow Name: ${workflowName}`);
  core.info(`Agent Conclusion: ${agentConclusion}`);
  if (detectionConclusion) {
    core.info(`Detection Conclusion: ${detectionConclusion}`);
  }
  if (assignToAgentErrorCount > 0) {
    core.info(`Assignment Error Count: ${assignToAgentErrorCount}`);
  }

  // Load agent output to check for noop messages
  let noopMessages = [];
  const agentOutputResult = loadAgentOutput();
  if (agentOutputResult.success) {
    const noopItems = agentOutputResult.items.filter(item => item.type === "noop");
    if (noopItems.length > 0) {
      core.info(`Found ${noopItems.length} noop message(s)`);
      noopMessages = noopItems.map(item => item.message);
    }
  }

  // If append-only is enabled, we do NOT require an activation comment ID.
  // If it's disabled, and there's no comment to update but we have noop messages, write to step summary.
  if (!appendOnlyComments && !commentId && noopMessages.length > 0) {
    core.info("No comment ID found, writing noop messages to step summary");

    let summaryContent = "## No-Op Messages\n\n";
    summaryContent += "The following messages were logged for transparency:\n\n";

    if (noopMessages.length === 1) {
      summaryContent += noopMessages[0];
    } else {
      summaryContent += noopMessages.map((msg, idx) => `${idx + 1}. ${msg}`).join("\n");
    }

    await core.summary.addRaw(summaryContent).write();
    core.info(`Successfully wrote ${noopMessages.length} noop message(s) to step summary`);
    return;
  }

  if (!appendOnlyComments && !commentId) {
    core.info("No comment ID found and no noop messages to process, skipping comment update");
    return;
  }

  // At this point, we have a comment to update
  if (!runUrl) {
    core.setFailed(`${ERR_VALIDATION}: Run URL is required`);
    return;
  }

  // Parse comment repo (format: "owner/repo")
  const repoOwner = commentRepo ? commentRepo.split("/")[0] : context.repo.owner;
  const repoName = commentRepo ? commentRepo.split("/")[1] : context.repo.repo;

  core.info(`Updating comment in ${repoOwner}/${repoName}`);

  // Determine the message based on agent conclusion using custom messages if configured
  let message;
  let detectionWarningMessage = "";

  // Check if detection job failed (if detection job exists)
  if (detectionConclusion && detectionConclusion === "failure") {
    // Detection job failed in error mode - report this prominently
    message = getDetectionFailureMessage({
      workflowName,
      runUrl,
    });
  } else if (detectionConclusion && detectionConclusion === "warning") {
    // Detection job produced a warning (continue-on-error mode)
    // Show success message but append caution section with progressive disclosure
    if (agentConclusion === "success" && assignToAgentErrorCount === 0) {
      message = getRunSuccessMessage({
        workflowName,
        runUrl,
      });
    } else {
      let statusText;
      if (agentConclusion === "success" && assignToAgentErrorCount > 0) {
        statusText = "failed to assign the coding agent";
      } else if (agentConclusion === "cancelled") {
        statusText = "was cancelled";
      } else if (agentConclusion === "skipped") {
        statusText = "was skipped";
      } else if (agentConclusion === "timed_out") {
        statusText = "timed out";
      } else {
        statusText = "failed";
      }
      message = getRunFailureMessage({
        workflowName,
        runUrl,
        status: statusText,
      });
    }
    // Build the caution section for detection warning
    detectionWarningMessage = getDetectionWarningMessage({
      workflowName,
      runUrl,
      reason: detectionReason,
    });
  } else if (agentConclusion === "success" && assignToAgentErrorCount === 0) {
    message = getRunSuccessMessage({
      workflowName,
      runUrl,
    });
  } else {
    // Determine status text based on conclusion type
    let statusText;
    if (agentConclusion === "success" && assignToAgentErrorCount > 0) {
      // Agent itself succeeded but one or more agent assignments failed
      statusText = "failed to assign the coding agent";
    } else if (agentConclusion === "cancelled") {
      statusText = "was cancelled";
    } else if (agentConclusion === "skipped") {
      statusText = "was skipped";
    } else if (agentConclusion === "timed_out") {
      statusText = "timed out";
    } else {
      statusText = "failed";
    }

    message = getRunFailureMessage({
      workflowName,
      runUrl,
      status: statusText,
    });
  }

  // Append detection warning caution section if present
  if (detectionWarningMessage) {
    message += "\n\n" + detectionWarningMessage;
  }

  // Add noop messages to the comment if any
  if (noopMessages.length > 0) {
    message += "\n\n";
    if (noopMessages.length === 1) {
      message += noopMessages[0];
    } else {
      message += noopMessages.map((msg, idx) => `${idx + 1}. ${msg}`).join("\n");
    }
  }

  // Collect generated asset URLs from safe output jobs
  const generatedAssets = collectGeneratedAssets();
  if (generatedAssets.length > 0) {
    message += "\n\n";
    generatedAssets.forEach(url => {
      message += `${url}\n`;
    });
  }

  // Add "needs-review" label when detection produced a warning
  if (detectionConclusion === "warning") {
    await tryAddNeedsReviewLabel(commentRepo);
  }

  // Append-only mode: create a new comment instead of updating the activation comment.
  if (appendOnlyComments) {
    try {
      const eventName = context.eventName;

      // Discussions: create a new discussion comment (threaded reply for discussion_comment)
      if (eventName === "discussion" || eventName === "discussion_comment") {
        const discussionNumber = context.payload?.discussion?.number;
        if (!discussionNumber) {
          core.warning("Unable to determine discussion number for append-only comment; skipping");
          return;
        }

        const { repository } = await github.graphql(
          `
          query($owner: String!, $repo: String!, $num: Int!) {
            repository(owner: $owner, name: $repo) {
              discussion(number: $num) {
                id
              }
            }
          }`,
          { owner: repoOwner, repo: repoName, num: discussionNumber }
        );

        const discussionId = repository?.discussion?.id;
        if (!discussionId) {
          core.warning("Unable to resolve discussion id for append-only comment; skipping");
          return;
        }

        // GitHub Discussions only supports two nesting levels, so if the triggering comment is
        // itself a reply, we resolve the top-level parent's node ID.
        const replyToId = eventName === "discussion_comment" ? await resolveTopLevelDiscussionCommentId(github, context.payload?.comment?.node_id) : null;
        const mutation = replyToId
          ? `mutation($dId: ID!, $body: String!, $replyToId: ID!) {
              addDiscussionComment(input: { discussionId: $dId, body: $body, replyToId: $replyToId }) {
                comment { id url }
              }
            }`
          : `mutation($dId: ID!, $body: String!) {
              addDiscussionComment(input: { discussionId: $dId, body: $body }) {
                comment { id url }
              }
            }`;

        const sanitizedMessage = sanitizeContent(message);
        const variables = replyToId ? { dId: discussionId, body: sanitizedMessage, replyToId } : { dId: discussionId, body: sanitizedMessage };
        const result = await github.graphql(mutation, variables);
        const created = result?.addDiscussionComment?.comment;
        core.info("Successfully created append-only discussion comment");
        if (created?.id) core.info(`Comment ID: ${created.id}`);
        if (created?.url) core.info(`Comment URL: ${created.url}`);
        return;
      }

      // Issues/PRs: determine issue number from event payload and create a new issue comment
      const issueNumber = context.payload?.issue?.number || context.payload?.pull_request?.number;
      if (!issueNumber) {
        core.warning("Unable to determine issue/PR number for append-only comment; skipping");
        return;
      }

      const sanitizedMessage = sanitizeContent(message);
      const response = await github.request("POST /repos/{owner}/{repo}/issues/{issue_number}/comments", {
        owner: repoOwner,
        repo: repoName,
        issue_number: issueNumber,
        body: sanitizedMessage,
        headers: {
          Accept: "application/vnd.github+json",
        },
      });

      core.info("Successfully created append-only comment");
      if (response?.data?.id) core.info(`Comment ID: ${response.data.id}`);
      if (response?.data?.html_url) core.info(`Comment URL: ${response.data.html_url}`);
      return;
    } catch (error) {
      // Check if the error is due to a locked issue/PR/discussion
      if (isLockedError(error)) {
        // Silently ignore locked resource errors - just log for debugging
        core.info(`Cannot create append-only comment: resource is locked (this is expected and not an error)`);
        return;
      }

      // Don't fail the workflow if we can't create the comment (for other errors)
      core.warning(`Failed to create append-only comment: ${getErrorMessage(error)}`);
      return;
    }
  }

  // At this point, we must have a comment ID (verified by earlier checks)
  if (!commentId) {
    core.setFailed(`${ERR_VALIDATION}: Comment ID is required for updating existing comment`);
    return;
  }

  // Check if this is a discussion comment (GraphQL node ID format)
  const isDiscussionComment = commentId.startsWith("DC_");

  const sanitizedMessage = sanitizeContent(message);

  try {
    if (isDiscussionComment) {
      // Update discussion comment using GraphQL
      const result = await github.graphql(
        `
        mutation($commentId: ID!, $body: String!) {
          updateDiscussionComment(input: { commentId: $commentId, body: $body }) {
            comment {
              id
              url
            }
          }
        }`,
        { commentId: commentId, body: sanitizedMessage }
      );

      const comment = result.updateDiscussionComment.comment;
      core.info(`Successfully updated discussion comment`);
      core.info(`Comment ID: ${comment.id}`);
      core.info(`Comment URL: ${comment.url}`);
    } else {
      // Update issue/PR comment using REST API
      const response = await github.request("PATCH /repos/{owner}/{repo}/issues/comments/{comment_id}", {
        owner: repoOwner,
        repo: repoName,
        comment_id: parseInt(commentId, 10),
        body: sanitizedMessage,
        headers: {
          Accept: "application/vnd.github+json",
        },
      });

      core.info(`Successfully updated comment`);
      core.info(`Comment ID: ${response.data.id}`);
      core.info(`Comment URL: ${response.data.html_url}`);
    }
  } catch (error) {
    // Check if the error is due to a locked issue/PR/discussion
    if (isLockedError(error)) {
      // Silently ignore locked resource errors - just log for debugging
      core.info(`Cannot update comment: resource is locked (this is expected and not an error)`);
      return;
    }

    // Don't fail the workflow if we can't update the comment (for other errors)
    core.warning(`Failed to update comment: ${getErrorMessage(error)}`);
  }
}

module.exports = { main };
