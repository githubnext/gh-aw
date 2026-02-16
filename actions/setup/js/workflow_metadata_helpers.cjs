// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Get workflow metadata from environment variables and context
 * This is used by expired entity cleanup scripts to build workflow run URLs
 * for closing comment footers.
 *
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @returns {{workflowName: string, workflowId: string, runId: number, runUrl: string}} Workflow metadata
 */
function getWorkflowMetadata(owner, repo) {
  const workflowName = process.env.GH_AW_WORKFLOW_NAME || "Workflow";
  const workflowId = process.env.GH_AW_WORKFLOW_ID || "";
  const runId = context.runId || 0;
  const githubServer = process.env.GITHUB_SERVER_URL || "https://github.com";
  const runUrl = context.payload?.repository ? `${context.payload.repository.html_url}/actions/runs/${runId}` : `${githubServer}/${owner}/${repo}/actions/runs/${runId}`;

  return {
    workflowName,
    workflowId,
    runId,
    runUrl,
  };
}

module.exports = {
  getWorkflowMetadata,
};
