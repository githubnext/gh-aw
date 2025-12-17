// @ts-check
/// <reference types="@actions/github-script" />

const { loadAgentOutput } = require("./load_agent_output.cjs");
const { generateFooterWithMessages } = require("./messages_footer.cjs");

/**
 * Parse a GitHub URL, short path, or number to extract owner, repo, and number
 * Supports issue and discussion URLs in multiple formats
 * @param {string} input - GitHub URL (https://github.com/owner/repo/issues/123), short path (owner/repo/issues/123), or just a number (123 or #123)
 * @param {string} defaultType - Default type when only number is provided ('issue' or 'discussion')
 * @returns {{owner: string, repo: string, number: number, type: 'issue'|'discussion'}|null}
 */
function parseGitHubUrl(input, defaultType = "issue") {
  // Match full issue URL: https://github.com/owner/repo/issues/123
  const issueMatch = input.match(/github\.com\/([^/]+)\/([^/]+)\/issues\/(\d+)/);
  if (issueMatch) {
    return {
      owner: issueMatch[1],
      repo: issueMatch[2],
      number: parseInt(issueMatch[3], 10),
      type: "issue",
    };
  }

  // Match full discussion URL: https://github.com/owner/repo/discussions/123
  const discussionMatch = input.match(/github\.com\/([^/]+)\/([^/]+)\/discussions\/(\d+)/);
  if (discussionMatch) {
    return {
      owner: discussionMatch[1],
      repo: discussionMatch[2],
      number: parseInt(discussionMatch[3], 10),
      type: "discussion",
    };
  }

  // Match short issue path: owner/repo/issues/123
  const shortIssueMatch = input.match(/^([^/]+)\/([^/]+)\/issues\/(\d+)$/);
  if (shortIssueMatch) {
    return {
      owner: shortIssueMatch[1],
      repo: shortIssueMatch[2],
      number: parseInt(shortIssueMatch[3], 10),
      type: "issue",
    };
  }

  // Match short discussion path: owner/repo/discussions/123
  const shortDiscussionMatch = input.match(/^([^/]+)\/([^/]+)\/discussions\/(\d+)$/);
  if (shortDiscussionMatch) {
    return {
      owner: shortDiscussionMatch[1],
      repo: shortDiscussionMatch[2],
      number: parseInt(shortDiscussionMatch[3], 10),
      type: "discussion",
    };
  }

  // Match just a number (with or without #): 123 or #123
  const numberMatch = input.match(/^#?(\d+)$/);
  if (numberMatch) {
    return {
      owner: context.repo.owner,
      repo: context.repo.repo,
      number: parseInt(numberMatch[1], 10),
      type: defaultType,
    };
  }

  return null;
}

/**
 * Comment on a GitHub Issue using REST API
 * @param {any} github - GitHub REST API instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} issueNumber - Issue number
 * @param {string} message - Comment body
 * @returns {Promise<{id: number, html_url: string}>} Comment details
 */
async function commentOnIssue(github, owner, repo, issueNumber, message) {
  const { data } = await github.rest.issues.createComment({
    owner,
    repo,
    issue_number: issueNumber,
    body: message,
  });

  return {
    id: data.id,
    html_url: data.html_url,
  };
}

/**
 * Comment on a GitHub Discussion using GraphQL
 * @param {any} github - GitHub GraphQL instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} discussionNumber - Discussion number
 * @param {string} message - Comment body
 * @returns {Promise<{id: string, html_url: string}>} Comment details
 */
async function commentOnDiscussion(github, owner, repo, discussionNumber, message) {
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

  // 2. Add comment
  const result = await github.graphql(
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

  const comment = result.addDiscussionComment.comment;

  return {
    id: comment.id,
    html_url: comment.url,
  };
}

/**
 * Main function to handle noop safe output
 * No-op is a fallback output type that logs messages for transparency
 * without taking any GitHub API actions (unless post-as-comment is configured)
 */
async function main() {
  // Check if we're in staged mode
  const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true";

  // Check if post-as-comment URL is provided
  const postAsCommentUrl = process.env.GH_AW_NOOP_POST_AS_COMMENT;

  const result = loadAgentOutput();
  if (!result.success) {
    return;
  }

  // Find all noop items
  const noopItems = result.items.filter(/** @param {any} item */ item => item.type === "noop");
  if (noopItems.length === 0) {
    core.info("No noop items found in agent output");
    return;
  }

  core.info(`Found ${noopItems.length} noop item(s)`);

  // If in staged mode, emit step summary instead of logging
  if (isStaged) {
    let summaryContent = "## 🎭 Staged Mode: No-Op Messages Preview\n\n";
    summaryContent += "The following messages would be logged if staged mode was disabled:\n\n";

    for (let i = 0; i < noopItems.length; i++) {
      const item = noopItems[i];
      summaryContent += `### Message ${i + 1}\n`;
      summaryContent += `${item.message}\n\n`;
      summaryContent += "---\n\n";
    }

    await core.summary.addRaw(summaryContent).write();
    core.info("📝 No-op message preview written to step summary");
    return;
  }

  // If post-as-comment URL is provided, post messages as comments
  if (postAsCommentUrl) {
    core.info(`Post-as-comment URL configured: ${postAsCommentUrl}`);

    // Determine default type based on event context
    const isDiscussionContext = context.eventName === "discussion" || context.eventName === "discussion_comment";
    const defaultType = isDiscussionContext ? "discussion" : "issue";

    const parsedUrl = parseGitHubUrl(postAsCommentUrl, defaultType);
    if (!parsedUrl) {
      core.warning(`Invalid GitHub URL format: ${postAsCommentUrl}. Expected issue or discussion URL.`);
      core.warning("Falling back to logging only.");
    } else {
      core.info(`Parsed URL: ${parsedUrl.type} #${parsedUrl.number} in ${parsedUrl.owner}/${parsedUrl.repo}`);

      // Get workflow metadata for footer
      const workflowName = process.env.GH_AW_WORKFLOW_NAME || "Workflow";
      const workflowSource = process.env.GH_AW_WORKFLOW_SOURCE || "";
      const workflowSourceURL = process.env.GH_AW_WORKFLOW_SOURCE_URL || "";

      // Get run URL
      const runUrl = context.payload.repository
        ? `${context.payload.repository.html_url}/actions/runs/${context.runId}`
        : `https://github.com/${context.repo.owner}/${context.repo.repo}/actions/runs/${context.runId}`;

      // Get triggering numbers from context
      const triggeringIssueNumber = context.payload?.issue?.number;
      const triggeringPRNumber = context.payload?.pull_request?.number;
      const triggeringDiscussionNumber = context.payload?.discussion?.number;

      // Post each noop message as a comment
      for (let i = 0; i < noopItems.length; i++) {
        const item = noopItems[i];
        try {
          // Build the comment body with the noop message and footer
          let commentBody = item.message;
          commentBody += generateFooterWithMessages(
            workflowName,
            runUrl,
            workflowSource,
            workflowSourceURL,
            triggeringIssueNumber,
            triggeringPRNumber,
            triggeringDiscussionNumber
          );

          let commentResult;
          if (parsedUrl.type === "issue") {
            core.info(`Posting message ${i + 1} as comment on issue #${parsedUrl.number}...`);
            commentResult = await commentOnIssue(github, parsedUrl.owner, parsedUrl.repo, parsedUrl.number, commentBody);
          } else {
            // discussion
            core.info(`Posting message ${i + 1} as comment on discussion #${parsedUrl.number}...`);
            commentResult = await commentOnDiscussion(github, parsedUrl.owner, parsedUrl.repo, parsedUrl.number, commentBody);
          }
          core.info(`✓ Comment posted: ${commentResult.html_url}`);
        } catch (error) {
          core.warning(`Failed to post comment for message ${i + 1}: ${error instanceof Error ? error.message : String(error)}`);
        }
      }

      core.info(`Successfully posted ${noopItems.length} noop message(s) as comment(s)`);
      return;
    }
  }

  // Default behavior: Process each noop item - just log the messages for transparency
  let summaryContent = "\n\n## No-Op Messages\n\n";
  summaryContent += "The following messages were logged for transparency:\n\n";

  for (let i = 0; i < noopItems.length; i++) {
    const item = noopItems[i];
    core.info(`No-op message ${i + 1}: ${item.message}`);
    summaryContent += `- ${item.message}\n`;
  }

  // Write summary for all noop messages
  await core.summary.addRaw(summaryContent).write();

  // Export the first noop message for use in add-comment default reporting
  if (noopItems.length > 0) {
    core.setOutput("noop_message", noopItems[0].message);
    core.exportVariable("GH_AW_NOOP_MESSAGE", noopItems[0].message);
  }

  core.info(`Successfully processed ${noopItems.length} noop message(s)`);
}

await main();
