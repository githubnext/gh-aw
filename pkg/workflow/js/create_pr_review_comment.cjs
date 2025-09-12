async function main() {
  // Read the validated output content from environment variable
  const outputContent = process.env.GITHUB_AW_AGENT_OUTPUT;
  if (!outputContent) {
    core.info("No GITHUB_AW_AGENT_OUTPUT environment variable found");
    return;
  }

  if (outputContent.trim() === "") {
    core.info("Agent output content is empty");
    return;
  }

  core.info(`Agent output content length:: ${outputContent.length}`);

  // Parse the validated output JSON
  let validatedOutput;
  try {
    validatedOutput = JSON.parse(outputContent);
  } catch (error) {
    core.info(
      `Error parsing agent output JSON: ${error instanceof Error ? error.message : String(error)}`
    );
    return;
  }

  if (!validatedOutput.items || !Array.isArray(validatedOutput.items)) {
    core.info("No valid items found in agent output");
    return;
  }

  // Find all create-pull-request-review-comment items
  const reviewCommentItems = validatedOutput.items.filter(
    /** @param {any} item */ item =>
      item.type === "create-pull-request-review-comment"
  );
  if (reviewCommentItems.length === 0) {
    core.info(
      "No create-pull-request-review-comment items found in agent output"
    );
    return;
  }

  core.info(
    `Found ${reviewCommentItems.length} create-pull-request-review-comment item(s)`
  );

  // If in staged mode, emit step summary instead of creating review comments
  if (process.env.GITHUB_AW_SAFE_OUTPUTS_STAGED === "true") {
    let summaryContent =
      "## 🎭 Staged Mode: Create PR Review Comments Preview\n\n";
    summaryContent +=
      "The following review comments would be created if staged mode was disabled:\n\n";

    for (let i = 0; i < reviewCommentItems.length; i++) {
      const item = reviewCommentItems[i];
      summaryContent += `### Review Comment ${i + 1}\n`;
      summaryContent += `**File:** ${item.path || "No path provided"}\n\n`;
      summaryContent += `**Line:** ${item.line || "No line provided"}\n\n`;
      if (item.start_line) {
        summaryContent += `**Start Line:** ${item.start_line}\n\n`;
      }
      summaryContent += `**Side:** ${item.side || "RIGHT"}\n\n`;
      summaryContent += `**Body:**\n${item.body || "No content provided"}\n\n`;
      summaryContent += "---\n\n";
    }

    // Write to step summary
    await core.summary.addRaw(summaryContent).write();
    core.info("📝 PR review comment creation preview written to step summary");
    return;
  }

  // Get the side configuration from environment variable
  const defaultSide = process.env.GITHUB_AW_PR_REVIEW_COMMENT_SIDE || "RIGHT";
  core.info(`Default comment side configuration: ${defaultSide}`);

  // Check if we're in a pull request context, or an issue comment context on a PR
  const isPRContext =
    context.eventName === "pull_request" ||
    context.eventName === "pull_request_review" ||
    context.eventName === "pull_request_review_comment" ||
    (context.eventName === "issue_comment" &&
      context.payload.issue &&
      context.payload.issue.pull_request);

  if (!isPRContext) {
    core.info(
      "Not running in pull request context, skipping review comment creation"
    );
    return;
  }
  let pullRequest = context.payload.pull_request;

  if (!pullRequest) {
    //No, github.event.issue.pull_request does not contain the full pull request data like head.sha. It only includes a minimal object with a url pointing to the pull request API resource.
    //To get full PR details (like head.sha, base.ref, etc.), you need to make an API call using that URL.

    if (context.payload.issue && context.payload.issue.pull_request) {
      // Fetch full pull request details using the GitHub API
      const prUrl = context.payload.issue.pull_request.url;
      try {
        const { data: fullPR } = await github.request(`GET ${prUrl}`, {
          headers: {
            Accept: "application/vnd.github+json",
          },
        });
        pullRequest = fullPR;
        core.info("Fetched full pull request details from API");
      } catch (error) {
        core.info(
          `Failed to fetch full pull request details: ${error instanceof Error ? error.message : String(error)}`
        );
        return;
      }
    } else {
      core.info(
        "Pull request data not found in payload - cannot create review comments"
      );
      return;
    }
  }

  // Check if we have the commit SHA needed for creating review comments
  if (!pullRequest || !pullRequest.head || !pullRequest.head.sha) {
    core.info(
      "Pull request head commit SHA not found in payload - cannot create review comments"
    );
    return;
  }

  const pullRequestNumber = pullRequest.number;
  core.info(`Creating review comments on PR #${pullRequestNumber}`);

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

    if (
      !commentItem.line ||
      (typeof commentItem.line !== "number" &&
        typeof commentItem.line !== "string")
    ) {
      core.info(
        'Missing or invalid required field "line" in review comment item'
      );
      continue;
    }

    if (!commentItem.body || typeof commentItem.body !== "string") {
      core.info(
        'Missing or invalid required field "body" in review comment item'
      );
      continue;
    }

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
        core.info(
          `Invalid start_line number: ${commentItem.start_line} (must be <= line: ${line})`
        );
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

    // Add AI disclaimer with run id, run htmlurl
    const runId = context.runId;
    const runUrl = context.payload.repository
      ? `${context.payload.repository.html_url}/actions/runs/${runId}`
      : `https://github.com/actions/runs/${runId}`;
    body += `\n\n> Generated by Agentic Workflow [Run](${runUrl})\n`;

    core.info(
      `Creating review comment on PR #${pullRequestNumber} at ${commentItem.path}:${line}${startLine ? ` (lines ${startLine}-${line})` : ""} [${side}]`
    );
    core.info(`Comment content length:: ${body.length}`);

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
      const { data: comment } =
        await github.rest.pulls.createReviewComment(requestParams);

      core.info(
        "Created review comment #" + comment.id + ": " + comment.html_url
      );
      createdComments.push(comment);

      // Set output for the last created comment (for backward compatibility)
      if (i === reviewCommentItems.length - 1) {
        core.setOutput("review_comment_id", comment.id);
        core.setOutput("review_comment_url", comment.html_url);
      }
    } catch (error) {
      core.error(
        `✗ Failed to create review comment: ${error instanceof Error ? error.message : String(error)}`
      );
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
await main();
