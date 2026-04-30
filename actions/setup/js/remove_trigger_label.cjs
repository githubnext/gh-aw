// @ts-check
/// <reference types="@actions/github-script" />

const { ERR_API, ERR_CONFIG } = require("./error_codes.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Remove the label that triggered this workflow from the issue, pull request, or discussion.
 * This allows the same label to be applied again later to re-trigger the workflow.
 *
 * Supported events: issues (labeled), pull_request (labeled), discussion (labeled).
 * For workflow_dispatch, the step emits an empty label_name output and exits without error.
 */
async function main() {
  const labelNamesJSON = process.env.GH_AW_LABEL_NAMES;

  if (!labelNamesJSON) {
    core.setFailed(`${ERR_CONFIG}: Configuration error: GH_AW_LABEL_NAMES not specified.`);
    return;
  }

  let labelNames = [];
  try {
    labelNames = JSON.parse(labelNamesJSON);
    if (!Array.isArray(labelNames)) {
      core.setFailed(`${ERR_CONFIG}: Configuration error: GH_AW_LABEL_NAMES must be a JSON array.`);
      return;
    }
  } catch (error) {
    core.setFailed(`${ERR_CONFIG}: Configuration error: Failed to parse GH_AW_LABEL_NAMES: ${getErrorMessage(error)}`);
    return;
  }

  const eventName = context.eventName;

  // For workflow_dispatch and other non-labeled events, nothing to remove.
  if (eventName === "workflow_dispatch") {
    core.info("Event is workflow_dispatch – skipping label removal.");
    core.setOutput("label_name", "");
    return;
  }

  // Retrieve the label that was added from the event payload.
  const triggerLabel = context.payload?.label?.name;
  if (!triggerLabel) {
    core.info(`Event ${eventName} has no label payload – skipping label removal.`);
    core.setOutput("label_name", "");
    return;
  }

  // Confirm that this label is one of the configured command labels.
  if (!labelNames.includes(triggerLabel)) {
    core.info(`Trigger label '${triggerLabel}' is not in the configured label-command list [${labelNames.join(", ")}] – skipping removal.`);
    core.setOutput("label_name", triggerLabel);
    return;
  }

  core.info(`Removing trigger label '${triggerLabel}' (event: ${eventName})`);

  const owner = context.repo?.owner;
  const repo = context.repo?.repo;
  if (!owner || !repo) {
    core.setFailed(`${ERR_CONFIG}: Configuration error: Unable to determine repository owner/name from context.`);
    return;
  }

  try {
    if (eventName === "issues") {
      const issueNumber = context.payload?.issue?.number;
      if (!issueNumber) {
        core.warning("No issue number found in payload – skipping label removal.");
        core.setOutput("label_name", triggerLabel);
        return;
      }
      await github.rest.issues.removeLabel({
        owner,
        repo,
        issue_number: issueNumber,
        name: triggerLabel,
      });
      core.info(`✓ Removed label '${triggerLabel}' from issue #${issueNumber}`);
    } else if (eventName === "pull_request") {
      // Pull requests share the issues API for labels.
      const prNumber = context.payload?.pull_request?.number;
      if (!prNumber) {
        core.warning("No pull request number found in payload – skipping label removal.");
        core.setOutput("label_name", triggerLabel);
        return;
      }
      await github.rest.issues.removeLabel({
        owner,
        repo,
        issue_number: prNumber,
        name: triggerLabel,
      });
      core.info(`✓ Removed label '${triggerLabel}' from pull request #${prNumber}`);
    } else if (eventName === "discussion") {
      // Discussions require the GraphQL API for label management.
      const discussionNodeId = context.payload?.discussion?.node_id;
      const labelNodeId = context.payload?.label?.node_id;
      if (!discussionNodeId || !labelNodeId) {
        core.warning("No discussion or label node_id found in payload – skipping label removal.");
        core.setOutput("label_name", triggerLabel);
        return;
      }
      await github.graphql(
        `
        mutation RemoveLabelFromDiscussion($labelableId: ID!, $labelIds: [ID!]!) {
          removeLabelsFromLabelable(input: { labelableId: $labelableId, labelIds: $labelIds }) {
            clientMutationId
          }
        }
      `,
        {
          labelableId: discussionNodeId,
          labelIds: [labelNodeId],
        }
      );
      core.info(`✓ Removed label '${triggerLabel}' from discussion`);
    } else {
      core.info(`Event '${eventName}' does not support label removal – skipping.`);
    }
  } catch (/** @type {any} */ error) {
    // Non-fatal: log a warning but do not fail the step.
    // A 404 status means the label is no longer present on the item (e.g., another concurrent
    // workflow run already removed it), which is an expected outcome in multi-workflow setups.
    const status = error?.status;
    if (status === 404) {
      core.info(`Label '${triggerLabel}' is no longer present on the item – already removed by another run.`);
    } else {
      core.warning(`${ERR_API}: Failed to remove label '${triggerLabel}': ${getErrorMessage(error)}`);
    }
  }

  core.setOutput("label_name", triggerLabel);
}

module.exports = { main };
