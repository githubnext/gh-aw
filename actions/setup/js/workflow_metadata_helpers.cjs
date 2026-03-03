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
  const runUrl = buildWorkflowRunUrl({ runId, serverUrl: context.serverUrl }, { owner, repo });

  return {
    workflowName,
    workflowId,
    runId,
    runUrl,
  };
}

/**
 * Build a workflow run URL from the provided context and workflow repository.
 *
 * This is the canonical helper for constructing attribution run URLs in safe-output
 * handlers. It must always receive the **original** workflow repo (not an overridden
 * cross-repo effectiveContext) so that footer links point back to the actual workflow
 * run regardless of which repository the output action targets.
 *
 * @param {any} ctx - GitHub Actions context (provides serverUrl and runId)
 * @param {{ owner: string, repo: string }} workflowRepo - The repository that owns the workflow run
 * @returns {string} The full workflow run URL
 */
function buildWorkflowRunUrl(ctx, workflowRepo) {
  const server = ctx.serverUrl || process.env.GITHUB_SERVER_URL || "https://github.com";
  return `${server}/${workflowRepo.owner}/${workflowRepo.repo}/actions/runs/${ctx.runId}`;
}

module.exports = {
  getWorkflowMetadata,
  buildWorkflowRunUrl,
};
