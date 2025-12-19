// @ts-check
/// <reference types="@actions/github-script" />

const { runUpdateWorkflow, createRenderStagedItem, createGetSummaryLine } = require("./update_runner.cjs");
const { isDiscussionContext, getDiscussionNumber } = require("./update_context_helpers.cjs");
const { generateFooterWithMessages } = require("./messages_footer.cjs");

// Use shared helper for staged preview rendering
const renderStagedItem = createRenderStagedItem({
  entityName: "Discussion",
  numberField: "discussion_number",
  targetLabel: "Target Discussion:",
  currentTargetText: "Current discussion",
  includeOperation: false,
});

/**
 * Execute the discussion update API call using GraphQL
 * @param {any} github - GitHub API client
 * @param {any} context - GitHub Actions context
 * @param {number} discussionNumber - Discussion number to update
 * @param {any} updateData - Data to update
 * @returns {Promise<any>} Updated discussion
 */
async function executeDiscussionUpdate(github, context, discussionNumber, updateData) {
  // Remove internal fields used for operation handling
  const { _operation, _rawBody, ...fieldsToUpdate } = updateData;

  // First, fetch the discussion node ID using its number
  const getDiscussionQuery = `
    query($owner: String!, $repo: String!, $number: Int!) {
      repository(owner: $owner, name: $repo) {
        discussion(number: $number) {
          id
          title
          body
          url
        }
      }
    }
  `;

  const queryResult = await github.graphql(getDiscussionQuery, {
    owner: context.repo.owner,
    repo: context.repo.repo,
    number: discussionNumber,
  });

  if (!queryResult?.repository?.discussion) {
    throw new Error(`Discussion #${discussionNumber} not found`);
  }

  const discussionId = queryResult.repository.discussion.id;

  // Ensure at least one field is being updated
  if (fieldsToUpdate.title === undefined && fieldsToUpdate.body === undefined) {
    throw new Error("At least one field (title or body) must be provided for update");
  }

  // Add footer to body if body is being updated
  if (fieldsToUpdate.body !== undefined) {
    const workflowName = process.env.GH_AW_WORKFLOW_NAME || "Workflow";
    const workflowSource = process.env.GH_AW_WORKFLOW_SOURCE || "";
    const workflowSourceURL = process.env.GH_AW_WORKFLOW_SOURCE_URL || "";
    const runId = context.runId;
    const githubServer = process.env.GITHUB_SERVER_URL || "https://github.com";
    const runUrl = context.payload.repository ? `${context.payload.repository.html_url}/actions/runs/${runId}` : `${githubServer}/${context.repo.owner}/${context.repo.repo}/actions/runs/${runId}`;

    // Get triggering context numbers
    const triggeringIssueNumber = context.payload.issue?.number;
    const triggeringPRNumber = context.payload.pull_request?.number;
    const triggeringDiscussionNumber = context.payload.discussion?.number;

    // Append footer to the body
    const footer = generateFooterWithMessages(workflowName, runUrl, workflowSource, workflowSourceURL, triggeringIssueNumber, triggeringPRNumber, triggeringDiscussionNumber);
    fieldsToUpdate.body = fieldsToUpdate.body + footer;
  }

  // Build the update mutation dynamically based on which fields are being updated
  const mutationFields = [];
  if (fieldsToUpdate.title !== undefined) {
    mutationFields.push("title: $title");
  }
  if (fieldsToUpdate.body !== undefined) {
    mutationFields.push("body: $body");
  }

  const updateDiscussionMutation = `
    mutation($discussionId: ID!${fieldsToUpdate.title !== undefined ? ", $title: String!" : ""}${fieldsToUpdate.body !== undefined ? ", $body: String!" : ""}) {
      updateDiscussion(input: {
        discussionId: $discussionId
        ${mutationFields.join("\n        ")}
      }) {
        discussion {
          id
          number
          title
          body
          url
        }
      }
    }
  `;

  const variables = {
    discussionId: discussionId,
  };

  if (fieldsToUpdate.title !== undefined) {
    variables.title = fieldsToUpdate.title;
  }

  if (fieldsToUpdate.body !== undefined) {
    variables.body = fieldsToUpdate.body;
  }

  const mutationResult = await github.graphql(updateDiscussionMutation, variables);

  if (!mutationResult?.updateDiscussion?.discussion) {
    throw new Error("Failed to update discussion");
  }

  const discussion = mutationResult.updateDiscussion.discussion;

  // Return with html_url (which the GraphQL returns as 'url')
  return {
    ...discussion,
    html_url: discussion.url,
  };
}

// Use shared helper for summary line generation
const getSummaryLine = createGetSummaryLine({
  entityPrefix: "Discussion",
});

async function main() {
  return await runUpdateWorkflow({
    itemType: "update_discussion",
    displayName: "discussion",
    displayNamePlural: "discussions",
    numberField: "discussion_number",
    outputNumberKey: "discussion_number",
    outputUrlKey: "discussion_url",
    isValidContext: isDiscussionContext,
    getContextNumber: getDiscussionNumber,
    supportsStatus: false,
    supportsOperation: false,
    renderStagedItem,
    executeUpdate: executeDiscussionUpdate,
    getSummaryLine,
  });
}

await main();
