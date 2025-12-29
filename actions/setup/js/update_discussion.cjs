// @ts-check
/// <reference types="@actions/github-script" />

const { createUpdateHandler } = require("./update_runner.cjs");
const { isDiscussionContext, getDiscussionNumber } = require("./update_context_helpers.cjs");
const { generateFooterWithMessages } = require("./messages_footer.cjs");

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
  const { _operation, _rawBody, labels, ...fieldsToUpdate } = updateData;

  // Check if labels should be updated based on environment variable
  const shouldUpdateLabels = process.env.GH_AW_UPDATE_LABELS === "true" && labels !== undefined;

  // First, fetch the discussion node ID using its number
  const getDiscussionQuery = shouldUpdateLabels
    ? `
    query($owner: String!, $repo: String!, $number: Int!) {
      repository(owner: $owner, name: $repo) {
        discussion(number: $number) {
          id
          title
          body
          url
          labels(first: 100) {
            nodes {
              id
              name
            }
          }
        }
      }
    }
  `
    : `
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

  const discussion = queryResult.repository.discussion;
  const discussionId = discussion.id;
  const currentLabels = shouldUpdateLabels ? discussion.labels?.nodes || [] : [];

  // Ensure at least one field is being updated
  if (fieldsToUpdate.title === undefined && fieldsToUpdate.body === undefined && !shouldUpdateLabels) {
    throw new Error("At least one field (title, body, or labels) must be provided for update");
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

  // Update title and/or body if needed
  if (fieldsToUpdate.title !== undefined || fieldsToUpdate.body !== undefined) {
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
          ${mutationFields.join("\n          ")}
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
  }

  // Update labels if provided and enabled
  if (shouldUpdateLabels && Array.isArray(labels)) {
    // Get all repository labels using pagination
    const repoLabels = [];
    let hasNextPage = true;
    let cursor = null;

    while (hasNextPage) {
      const repoQuery = `
        query($owner: String!, $repo: String!, $cursor: String) {
          repository(owner: $owner, name: $repo) {
            id
            labels(first: 100, after: $cursor) {
              nodes {
                id
                name
              }
              pageInfo {
                hasNextPage
                endCursor
              }
            }
          }
        }
      `;

      const repoResult = await github.graphql(repoQuery, {
        owner: context.repo.owner,
        repo: context.repo.repo,
        cursor: cursor,
      });

      if (!repoResult?.repository) {
        throw new Error(`Repository ${context.repo.owner}/${context.repo.repo} not found`);
      }

      const labels = repoResult.repository.labels?.nodes || [];
      repoLabels.push(...labels);

      hasNextPage = repoResult.repository.labels?.pageInfo?.hasNextPage || false;
      cursor = repoResult.repository.labels?.pageInfo?.endCursor || null;
    }

    // Map label names to IDs
    const labelIds = labels.map(labelName => {
      const label = repoLabels.find(l => l.name === labelName);
      if (!label) {
        throw new Error(`Label "${labelName}" not found in repository`);
      }
      return label.id;
    });

    // Remove all current labels
    if (currentLabels.length > 0) {
      const removeLabelsMutation = `
        mutation($labelableId: ID!, $labelIds: [ID!]!) {
          removeLabelsFromLabelable(input: {
            labelableId: $labelableId
            labelIds: $labelIds
          }) {
            clientMutationId
          }
        }
      `;

      await github.graphql(removeLabelsMutation, {
        labelableId: discussionId,
        labelIds: currentLabels.map(l => l.id),
      });
    }

    // Add new labels
    if (labelIds.length > 0) {
      const addLabelsMutation = `
        mutation($labelableId: ID!, $labelIds: [ID!]!) {
          addLabelsToLabelable(input: {
            labelableId: $labelableId
            labelIds: $labelIds
          }) {
            clientMutationId
          }
        }
      `;

      await github.graphql(addLabelsMutation, {
        labelableId: discussionId,
        labelIds: labelIds,
      });
    }
  }

  // Fetch the updated discussion to return
  const finalQuery = shouldUpdateLabels
    ? `
    query($owner: String!, $repo: String!, $number: Int!) {
      repository(owner: $owner, name: $repo) {
        discussion(number: $number) {
          id
          title
          body
          url
          labels(first: 100) {
            nodes {
              id
              name
            }
          }
        }
      }
    }
  `
    : `
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

  const finalQueryResult = await github.graphql(finalQuery, {
    owner: context.repo.owner,
    repo: context.repo.repo,
    number: discussionNumber,
  });

  const updatedDiscussion = finalQueryResult.repository.discussion;

  // Return with html_url (which the GraphQL returns as 'url')
  return {
    ...updatedDiscussion,
    html_url: updatedDiscussion.url,
  };
}

// Create the handler using the factory
const main = createUpdateHandler({
  itemType: "update_discussion",
  displayName: "discussion",
  displayNamePlural: "discussions",
  numberField: "discussion_number",
  outputNumberKey: "discussion_number",
  outputUrlKey: "discussion_url",
  entityName: "Discussion",
  entityPrefix: "Discussion",
  targetLabel: "Target Discussion:",
  currentTargetText: "Current discussion",
  supportsStatus: false,
  supportsOperation: false,
  isValidContext: isDiscussionContext,
  getContextNumber: getDiscussionNumber,
  executeUpdate: executeDiscussionUpdate,
});

module.exports = { main };
