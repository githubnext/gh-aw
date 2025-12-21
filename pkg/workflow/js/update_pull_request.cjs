// @ts-check
/// <reference types="@actions/github-script" />

const { createUpdateHandler } = require("./update_runner.cjs");
const { updatePRBody } = require("./update_pr_description_helpers.cjs");
const { isPRContext, getPRNumber } = require("./update_context_helpers.cjs");

/**
 * Execute the pull request update API call
 * @param {any} github - GitHub API client
 * @param {any} context - GitHub Actions context
 * @param {number} prNumber - PR number to update
 * @param {any} updateData - Data to update
 * @returns {Promise<any>} Updated pull request
 */
async function executePRUpdate(github, context, prNumber, updateData) {
  // Handle body operation (append/prepend/replace/replace-island)
  const operation = updateData._operation || "replace";
  const rawBody = updateData._rawBody;

  // Remove internal fields
  const { _operation, _rawBody, ...apiData } = updateData;

  // If we have a body with operation, handle it
  if (rawBody !== undefined && operation !== "replace") {
    // Fetch current PR body for operations that need it
    const { data: currentPR } = await github.rest.pulls.get({
      owner: context.repo.owner,
      repo: context.repo.repo,
      pull_number: prNumber,
    });
    const currentBody = currentPR.body || "";

    // Get workflow run URL for AI attribution
    const workflowName = process.env.GH_AW_WORKFLOW_NAME || "GitHub Agentic Workflow";
    const runUrl = `${context.serverUrl}/${context.repo.owner}/${context.repo.repo}/actions/runs/${context.runId}`;

    // Use helper to update body
    apiData.body = updatePRBody({
      currentBody,
      newContent: rawBody,
      operation,
      workflowName,
      runUrl,
      runId: context.runId,
    });

    core.info(`Will update body (length: ${apiData.body.length})`);
  } else if (rawBody !== undefined) {
    // Replace: just use the new content as-is (already in apiData.body)
    core.info("Operation: replace (full body replacement)");
  }

  const { data: pr } = await github.rest.pulls.update({
    owner: context.repo.owner,
    repo: context.repo.repo,
    pull_number: prNumber,
    ...apiData,
  });

  return pr;
}

// Create the handler using the factory
const main = createUpdateHandler({
  itemType: "update_pull_request",
  displayName: "pull request",
  displayNamePlural: "pull requests",
  numberField: "pull_request_number",
  outputNumberKey: "pull_request_number",
  outputUrlKey: "pull_request_url",
  entityName: "Pull Request",
  entityPrefix: "PR",
  targetLabel: "Target PR:",
  currentTargetText: "Current pull request",
  supportsStatus: false,
  supportsOperation: true,
  isValidContext: isPRContext,
  getContextNumber: getPRNumber,
  executeUpdate: executePRUpdate,
});

await main();
