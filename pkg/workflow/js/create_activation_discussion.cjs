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
const body = `Agentic workflow \`${workflowName}\` started a run at ${new Date().toISOString()}.\n\n[View workflow run](${runUrl})`;

core.info(`Creating discussion to track workflow run: ${title}`);
core.info(`Category: ${categoryName}`);

try {
  // Get repository information
  const { owner, repo } = context.repo;

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

  // Find the category by name
  let categoryId = null;
  for (const category of discussionCategories) {
    if (category.name === categoryName) {
      categoryId = category.id;
      break;
    }
  }

  // If category not found, try to use the first available category
  if (!categoryId && discussionCategories.length > 0) {
    core.warning(`Category "${categoryName}" not found. Using first available category: ${discussionCategories[0].name}`);
    categoryId = discussionCategories[0].id;
  }

  if (!categoryId) {
    core.warning("No discussion categories available. Cannot create discussion.");
    return;
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
    core.setFailed("Failed to create discussion - no discussion returned");
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
  throw error;
}
