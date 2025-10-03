/**
 * Creates a GitHub discussion to track agentic workflow run progress
 * This script runs in the activation job to create a discussion for tracking.
 */

// Get workflow information from environment
const workflowName = process.env.GITHUB_AW_WORKFLOW_NAME;
if (!workflowName) {
  throw new Error("GITHUB_AW_WORKFLOW_NAME environment variable is required");
}

const categoryName = process.env.GITHUB_AW_DISCUSSION_CATEGORY || "";
const runId = context.runId;
const runUrl = context.payload.repository
  ? `${context.payload.repository.html_url}/actions/runs/${runId}`
  : `https://github.com/actions/runs/${runId}`;

// Create discussion title and body
const title = `${workflowName} - Run ${runId}`;

// Build the body with context reference in Copilot Chat style
let bodyParts = [`**${workflowName}** [started work](${runUrl})`];

// Add context reference based on event type
const { owner, repo } = context.repo;
const eventName = context.eventName;

if (eventName === "issues" && context.payload.issue) {
  const issueNumber = context.payload.issue.number;
  bodyParts.push(` for issue #${issueNumber}`);
} else if (eventName === "pull_request" && context.payload.pull_request) {
  const prNumber = context.payload.pull_request.number;
  bodyParts.push(` for pull request #${prNumber}`);
} else if (eventName === "issue_comment" && context.payload.issue && context.payload.comment) {
  const issueNumber = context.payload.issue.number;
  bodyParts.push(` for comment on issue #${issueNumber}`);
} else if (eventName === "pull_request_review_comment" && context.payload.pull_request && context.payload.comment) {
  const prNumber = context.payload.pull_request.number;
  bodyParts.push(` for review comment on pull request #${prNumber}`);
} else if (eventName === "pull_request_review" && context.payload.pull_request && context.payload.review) {
  const prNumber = context.payload.pull_request.number;
  bodyParts.push(` for review on pull request #${prNumber}`);
} else if (eventName === "workflow_dispatch") {
  // For workflow_dispatch, try to find the associated PR to the branch
  try {
    const ref = context.ref; // e.g., refs/heads/my-branch
    const branch = ref.replace("refs/heads/", "");

    core.info(`Looking for PR associated with branch: ${branch}`);

    const prsQuery = `
      query($owner: String!, $repo: String!, $headRefName: String!) {
        repository(owner: $owner, name: $repo) {
          pullRequests(first: 1, headRefName: $headRefName, states: OPEN) {
            nodes {
              number
            }
          }
        }
      }
    `;

    const prsResult = await github.graphql(prsQuery, {
      owner,
      repo,
      headRefName: branch,
    });

    if (prsResult?.repository?.pullRequests?.nodes?.length > 0) {
      const prNumber = prsResult.repository.pullRequests.nodes[0].number;
      core.info(`Found associated PR #${prNumber} for branch ${branch}`);
      bodyParts.push(` for pull request #${prNumber}`);
    } else {
      core.info(`No open PR found for branch ${branch}`);
      bodyParts.push(` on 'workflow_dispatch' event`);
    }
  } catch (error) {
    core.warning(`Failed to lookup PR for branch: ${error.message}`);
    bodyParts.push(` on 'workflow_dispatch' event`);
  }
} else if (eventName) {
  bodyParts.push(` on '${eventName}' event`);
}

bodyParts.push(`.`);
const body = bodyParts.join("");

core.info(`Creating discussion to track workflow run: ${title}`);
core.info(`Category: ${categoryName || "(auto-resolve)"}`);

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

  // Resolve category: try categoryName, then workflow name, then "Agentic Workflows"
  let categoryId = null;
  let resolvedCategoryName = categoryName;

  // If categoryName is empty, try workflow name first, then default
  if (!resolvedCategoryName) {
    resolvedCategoryName = workflowName;
    core.info(`Category name not specified, trying workflow name: ${resolvedCategoryName}`);
  }

  // Try to find category by name or ID
  for (const category of discussionCategories) {
    if (category.name === resolvedCategoryName || category.id === resolvedCategoryName) {
      categoryId = category.id;
      break;
    }
  }

  // If not found and we were using workflow name, fall back to "Agentic Workflows"
  if (!categoryId && resolvedCategoryName === workflowName) {
    core.info(`Category "${resolvedCategoryName}" not found, trying default: "Agentic Workflows"`);
    resolvedCategoryName = "Agentic Workflows";
    for (const category of discussionCategories) {
      if (category.name === resolvedCategoryName || category.id === resolvedCategoryName) {
        categoryId = category.id;
        break;
      }
    }
  }

  // If category not found, log error and give up
  if (!categoryId) {
    core.error(`Category "${resolvedCategoryName}" not found. Cannot create discussion without a category.`);
    core.error(`Available categories: ${discussionCategories.map(c => c.name).join(", ")}`);
    return;
  }

  core.info(`Using category: ${resolvedCategoryName} (ID: ${categoryId})`);

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
