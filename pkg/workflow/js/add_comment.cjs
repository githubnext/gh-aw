async function main() {
  // Check if we're in staged mode
  const isStaged = process.env.GITHUB_AW_SAFE_OUTPUTS_STAGED === "true";

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

  core.info(`Agent output content length: ${outputContent.length}`);

  // Parse the validated output JSON
  let validatedOutput;
  try {
    validatedOutput = JSON.parse(outputContent);
  } catch (error) {
    core.setFailed(`Error parsing agent output JSON: ${error instanceof Error ? error.message : String(error)}`);
    return;
  }

  if (!validatedOutput.items || !Array.isArray(validatedOutput.items)) {
    core.info("No valid items found in agent output");
    return;
  }

  // Find all add-comment items
  const commentItems = validatedOutput.items.filter(/** @param {any} item */ item => item.type === "add-comment");
  if (commentItems.length === 0) {
    core.info("No add-comment items found in agent output");
    return;
  }

  core.info(`Found ${commentItems.length} add-comment item(s)`);

  // Helper function to get the repository URL for different purposes
  function getRepositoryUrl() {
    // For trial mode, use target repository for issue/PR URLs but execution context for action runs
    const targetRepo = process.env.GITHUB_AW_TARGET_REPO;

    if (targetRepo) {
      // Use target repository for issue/PR URLs in trial mode
      return `https://github.com/${targetRepo}`;
    } else if (context.payload.repository) {
      // Use execution context repository (default behavior)
      return context.payload.repository.html_url;
    } else {
      // Final fallback for action runs when context repo is not available
      return `https://github.com/${context.repo.owner}/${context.repo.repo}`;
    }
  }

  // If in staged mode, emit step summary instead of creating comments
  if (isStaged) {
    let summaryContent = "## 🎭 Staged Mode: Add Comments Preview\n\n";
    summaryContent += "The following comments would be added if staged mode was disabled:\n\n";

    for (let i = 0; i < commentItems.length; i++) {
      const item = commentItems[i];
      summaryContent += `### Comment ${i + 1}\n`;
      if (item.issue_number) {
        const repoUrl = getRepositoryUrl();
        const issueUrl = `${repoUrl}/issues/${item.issue_number}`;
        summaryContent += `**Target Issue:** [#${item.issue_number}](${issueUrl})\n\n`;
      } else {
        summaryContent += `**Target:** Current issue/PR\n\n`;
      }
      summaryContent += `**Body:**\n${item.body || "No content provided"}\n\n`;
      summaryContent += "---\n\n";
    }

    // Write to step summary
    await core.summary.addRaw(summaryContent).write();
    core.info("📝 Comment creation preview written to step summary");
    return;
  }

  // Get the target configuration from environment variable
  const commentTarget = process.env.GITHUB_AW_COMMENT_TARGET || "triggering";
  core.info(`Comment target configuration: ${commentTarget}`);

  // Check if we're in an issue, pull request, or discussion context
  const isIssueContext = context.eventName === "issues" || context.eventName === "issue_comment";
  const isPRContext =
    context.eventName === "pull_request" ||
    context.eventName === "pull_request_review" ||
    context.eventName === "pull_request_review_comment";
  const isDiscussionContext = context.eventName === "discussion" || context.eventName === "discussion_comment";

  // Validate context based on target configuration
  if (commentTarget === "triggering" && !isIssueContext && !isPRContext && !isDiscussionContext) {
    core.info('Target is "triggering" but not running in issue, pull request, or discussion context, skipping comment creation');
    return;
  }

  const createdComments = [];

  // Process each comment item
  for (let i = 0; i < commentItems.length; i++) {
    const commentItem = commentItems[i];
    core.info(`Processing add-comment item ${i + 1}/${commentItems.length}: bodyLength=${commentItem.body.length}`);

    // Determine the issue/PR/discussion number and comment endpoint for this comment
    let issueNumber;
    let discussionNumber;
    let commentEndpoint;
    let isDiscussionComment = false;

    if (commentTarget === "*") {
      // For target "*", we need an explicit issue number from the comment item
      if (commentItem.issue_number) {
        issueNumber = parseInt(commentItem.issue_number, 10);
        if (isNaN(issueNumber) || issueNumber <= 0) {
          core.info(`Invalid issue number specified: ${commentItem.issue_number}`);
          continue;
        }
        commentEndpoint = "issues";
      } else {
        core.info('Target is "*" but no issue_number specified in comment item');
        continue;
      }
    } else if (commentTarget && commentTarget !== "triggering") {
      // Explicit issue number specified in target
      issueNumber = parseInt(commentTarget, 10);
      if (isNaN(issueNumber) || issueNumber <= 0) {
        core.info(`Invalid issue number in target configuration: ${commentTarget}`);
        continue;
      }
      commentEndpoint = "issues";
    } else {
      // Default behavior: use triggering issue/PR/discussion
      if (isIssueContext) {
        if (context.payload.issue) {
          issueNumber = context.payload.issue.number;
          commentEndpoint = "issues";
        } else {
          core.info("Issue context detected but no issue found in payload");
          continue;
        }
      } else if (isPRContext) {
        if (context.payload.pull_request) {
          issueNumber = context.payload.pull_request.number;
          commentEndpoint = "issues"; // PR comments use the issues API endpoint
        } else {
          core.info("Pull request context detected but no pull request found in payload");
          continue;
        }
      } else if (isDiscussionContext) {
        if (context.payload.discussion) {
          discussionNumber = context.payload.discussion.number;
          isDiscussionComment = true;
        } else {
          core.info("Discussion context detected but no discussion found in payload");
          continue;
        }
      }
    }

    if (!issueNumber && !discussionNumber) {
      core.info("Could not determine issue, pull request, or discussion number");
      continue;
    }

    // Extract body from the JSON item
    let body = commentItem.body.trim();
    // Add AI disclaimer with workflow name and run url
    const workflowName = process.env.GITHUB_AW_WORKFLOW_NAME || "Workflow";
    const runId = context.runId;
    const runUrl = context.payload.repository
      ? `${context.payload.repository.html_url}/actions/runs/${runId}`
      : `https://github.com/${context.repo.owner}/${context.repo.repo}/actions/runs/${runId}`;
    body += `\n\n> AI generated by [${workflowName}](${runUrl})\n`;

    try {
      let comment;

      if (isDiscussionComment) {
        // For discussions, we need to use GraphQL API
        core.info(`Creating comment on discussion #${discussionNumber}`);
        core.info(`Comment content length: ${body.length}`);

        // First, get the discussion ID (GraphQL node ID)
        const discussionQuery = `
          query($owner: String!, $repo: String!, $number: Int!) {
            repository(owner: $owner, name: $repo) {
              discussion(number: $number) {
                id
              }
            }
          }
        `;

        const discussionResult = await github.graphql(discussionQuery, {
          owner: context.repo.owner,
          repo: context.repo.repo,
          number: discussionNumber,
        });

        if (!discussionResult?.repository?.discussion?.id) {
          throw new Error(`Could not find discussion #${discussionNumber}`);
        }

        const discussionId = discussionResult.repository.discussion.id;

        // Create the comment using GraphQL mutation
        const addCommentMutation = `
          mutation($discussionId: ID!, $body: String!) {
            addDiscussionComment(input: {discussionId: $discussionId, body: $body}) {
              comment {
                id
                url
                databaseId
              }
            }
          }
        `;

        const mutationResult = await github.graphql(addCommentMutation, {
          discussionId: discussionId,
          body: body,
        });

        if (!mutationResult?.addDiscussionComment?.comment) {
          throw new Error("Failed to create discussion comment");
        }

        comment = {
          id: mutationResult.addDiscussionComment.comment.databaseId,
          html_url: mutationResult.addDiscussionComment.comment.url,
        };

        core.info("Created discussion comment #" + comment.id + ": " + comment.html_url);
      } else {
        // For issues and PRs, use REST API
        core.info(`Creating comment on ${commentEndpoint} #${issueNumber}`);
        core.info(`Comment content length: ${body.length}`);

        const { data: restComment } = await github.rest.issues.createComment({
          owner: context.repo.owner,
          repo: context.repo.repo,
          issue_number: issueNumber,
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
await main();
