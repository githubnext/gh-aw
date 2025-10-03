/**
 * Creates a GitHub discussion to track agentic workflow run progress
 * This script runs in the activation job to create a discussion for tracking.
 */

// Get workflow information from environment
const workflowName = process.env.GITHUB_AW_WORKFLOW_NAME || "Workflow";
const categoryName = process.env.GITHUB_AW_DISCUSSION_CATEGORY || "Agentic Workflows";
const runId = context.runId;
const runUrl = context.payload.repository
  ? `${context.payload.repository.html_url}/actions/runs/${runId}`
  : `https://github.com/actions/runs/${runId}`;

// Create discussion title and body
const title = `${workflowName} - Run ${runId}`;

// Build the body with context reference
let bodyParts = [`Agentic workflow \`${workflowName}\` started a run at ${new Date().toISOString()}.`];

// Add context reference based on event type
const { owner, repo } = context.repo;
const eventName = context.eventName;

if (eventName === "issues" && context.payload.issue) {
  const issueNumber = context.payload.issue.number;
  bodyParts.push(`\nTriggered by issue #${issueNumber}`);
} else if (eventName === "pull_request" && context.payload.pull_request) {
  const prNumber = context.payload.pull_request.number;
  bodyParts.push(`\nTriggered by pull request #${prNumber}`);
} else if (eventName === "issue_comment" && context.payload.issue && context.payload.comment) {
  const issueNumber = context.payload.issue.number;
  const commentId = context.payload.comment.id;
  bodyParts.push(`\nTriggered by comment on issue #${issueNumber}`);
} else if (eventName === "pull_request_review_comment" && context.payload.pull_request && context.payload.comment) {
  const prNumber = context.payload.pull_request.number;
  bodyParts.push(`\nTriggered by review comment on pull request #${prNumber}`);
} else if (eventName === "pull_request_review" && context.payload.pull_request && context.payload.review) {
  const prNumber = context.payload.pull_request.number;
  bodyParts.push(`\nTriggered by review on pull request #${prNumber}`);
} else if (eventName) {
  bodyParts.push(`\nTriggered by event: \`${eventName}\``);
}

bodyParts.push(`\n[View workflow run](${runUrl})`);
const body = bodyParts.join("");

core.info(`Creating discussion to track workflow run: ${title}`);
core.info(`Category: ${categoryName}`);

try {
  // First, get the repository ID and discussion categories
  const repoQuery = `
    query($owner: String!, $repo: String!) {
      repository(owner: $owner, name: $repo) {
        id
        discussionCategories(first: 100) {
          nodes {
            id
            name
            slug
          }
        }
      }
    }
  `;

  const repoResult = await github.graphql(repoQuery, {
    owner,
    repo,
  });

  if (!repoResult || !repoResult.repository) {
    core.warning("Could not fetch repository information. Discussions may not be enabled.");
    return;
  }

  const repositoryId = repoResult.repository.id;
  const discussionCategories = repoResult.repository.discussionCategories.nodes;

  // Find the category by name or ID
  let categoryId = null;
  for (const category of discussionCategories) {
    if (category.name === categoryName || category.id === categoryName) {
      categoryId = category.id;
      break;
    }
  }

  // If category not found, create it
  if (!categoryId) {
    core.info(`Category "${categoryName}" not found. Creating new discussion category...`);

    const createCategoryMutation = `
      mutation($repositoryId: ID!, $name: String!, $emoji: String!) {
        createDiscussionCategory(input: {
          repositoryId: $repositoryId
          name: $name
          emoji: $emoji
          isAnswerable: false
        }) {
          discussionCategory {
            id
            name
          }
        }
      }
    `;

    try {
      const createCategoryResult = await github.graphql(createCategoryMutation, {
        repositoryId,
        name: categoryName,
        emoji: "🤖",
      });

      if (
        createCategoryResult &&
        createCategoryResult.createDiscussionCategory &&
        createCategoryResult.createDiscussionCategory.discussionCategory
      ) {
        categoryId = createCategoryResult.createDiscussionCategory.discussionCategory.id;
        core.info(`✓ Created category "${categoryName}" with ID: ${categoryId}`);
      } else {
        core.warning("Failed to create discussion category. Cannot create discussion.");
        return;
      }
    } catch (categoryError) {
      const categoryErrorMessage = categoryError instanceof Error ? categoryError.message : String(categoryError);
      core.warning(`Failed to create discussion category: ${categoryErrorMessage}`);
      core.warning("Cannot create discussion without a category.");
      return;
    }
  }

  // Create the discussion using GraphQL API
  const createDiscussionMutation = `
    mutation($repositoryId: ID!, $categoryId: ID!, $title: String!, $body: String!) {
      createDiscussion(input: {
        repositoryId: $repositoryId,
        categoryId: $categoryId,
        title: $title,
        body: $body
      }) {
        discussion {
          id
          number
          title
          url
        }
      }
    }
  `;

  const createResult = await github.graphql(createDiscussionMutation, {
    repositoryId,
    categoryId,
    title,
    body,
  });

  if (!createResult || !createResult.createDiscussion || !createResult.createDiscussion.discussion) {
    core.error("Failed to create discussion - no discussion returned");
    return;
  }

  const discussion = createResult.createDiscussion.discussion;

  core.info(`✓ Created discussion #${discussion.number}: ${discussion.title}`);
  core.info(`  URL: ${discussion.url}`);

  // Set outputs for downstream jobs
  core.setOutput("discussion-id", discussion.id);
  core.setOutput("discussion-number", discussion.number);
  core.setOutput("discussion-url", discussion.url);
} catch (error) {
  const errorMessage = error instanceof Error ? error.message : String(error);

  // Check if the error is due to discussions not being enabled
  if (
    errorMessage.includes("Not Found") ||
    errorMessage.includes("not found") ||
    errorMessage.includes("Could not resolve to a Repository") ||
    errorMessage.includes("Discussions are disabled")
  ) {
    core.info("⚠ Cannot create discussion: Discussions are not enabled for this repository");
    core.info("Consider enabling discussions in repository settings if you want to track workflow runs");
    return;
  }

  core.error(`Failed to create discussion: ${errorMessage}`);
}
