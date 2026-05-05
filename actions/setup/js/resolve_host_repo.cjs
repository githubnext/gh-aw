// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Resolves the target repository and refs for the activation job.
 *
 * Uses the job.workflow_* context fields to determine the platform (host)
 * repository and produce two distinct ref outputs for different consumers:
 *
 * - target_checkout_ref: the immutable commit SHA from job.workflow_sha, used
 *   by actions/checkout to pin the activation checkout to the exact executing
 *   revision rather than a moving branch/tag ref.
 *
 * - target_ref: the branch or tag ref parsed from job.workflow_ref (the
 *   substring after "@", e.g. "refs/heads/main" from
 *   "owner/repo/.github/workflows/file.yml@refs/heads/main"), used by
 *   dispatch_workflow.cjs as the `ref` argument to createWorkflowDispatch.
 *   The GitHub workflow dispatch API only accepts branch/tag refs; passing a
 *   commit SHA causes "No ref found for: <sha>" errors.
 *
 * These fields are passed via environment variables (JOB_WORKFLOW_REPOSITORY,
 * JOB_WORKFLOW_SHA, JOB_WORKFLOW_REF, etc.) to avoid shell injection — the
 * ${{ }} expressions are evaluated in the env: block, not interpolated into
 * script source.
 *
 * job.workflow_repository provides the owner/repo of the currently executing
 * workflow file, correctly identifying the platform repo in all relay patterns:
 * cross-repo workflow_call, event-driven relays (on: issue_comment, on: push),
 * and cross-org scenarios.
 *
 * @safe-outputs-exempt SEC-005: values sourced from trusted GitHub Actions runner context via env vars only
 */

/**
 * Parses the dispatch-compatible branch/tag ref from job.workflow_ref.
 * job.workflow_ref has the form "owner/repo/.github/workflows/file.yml@refs/heads/main".
 * Returns the substring after "@", or an empty string if missing or malformed.
 *
 * @param {string} workflowRef
 * @returns {string}
 */
function parseDispatchRef(workflowRef) {
  if (!workflowRef) {
    return "";
  }
  const atIndex = workflowRef.indexOf("@");
  if (atIndex === -1) {
    return "";
  }
  return workflowRef.slice(atIndex + 1);
}

/**
 * @returns {Promise<void>}
 */
async function main() {
  const targetRepo = process.env.JOB_WORKFLOW_REPOSITORY || "";
  const targetCheckoutRef = process.env.JOB_WORKFLOW_SHA || "";
  const workflowRef = process.env.JOB_WORKFLOW_REF || "";
  const targetDispatchRef = parseDispatchRef(workflowRef);
  const targetRepoName = targetRepo.split("/").pop() || "";
  const currentRepo = process.env.GITHUB_REPOSITORY || "";

  core.info("Resolving host repo via job.workflow_* context");
  core.info(`job.workflow_repository = ${targetRepo}`);
  core.info(`job.workflow_sha        = ${targetCheckoutRef}`);
  core.info(`job.workflow_ref        = ${workflowRef}`);
  core.info(`job.workflow_file_path  = ${process.env.JOB_WORKFLOW_FILE_PATH || ""}`);
  core.info(`github.repository       = ${currentRepo}`);
  core.info("");
  core.info(`Resolved target_repo         = ${targetRepo}`);
  core.info(`Resolved target_repo_name    = ${targetRepoName}`);
  core.info(`Resolved target_checkout_ref = ${targetCheckoutRef}`);
  core.info(`Resolved target_ref          = ${targetDispatchRef}`);

  if (!targetDispatchRef) {
    core.warning(
      `Could not parse a branch/tag ref from JOB_WORKFLOW_REF="${workflowRef}". ` +
        "dispatch_workflow safe outputs may fail if they rely on target_ref. " +
        "Falling back to empty string — do not use target_checkout_ref (SHA) as the dispatch ref."
    );
  }

  if (targetRepo && targetRepo !== currentRepo) {
    core.info(`Cross-repo invocation detected: platform repo "${targetRepo}" differs from caller "${currentRepo}"`);
  } else {
    core.info(`Same-repo invocation: platform and caller are both "${targetRepo}"`);
  }

  core.setOutput("target_repo", targetRepo);
  core.setOutput("target_repo_name", targetRepoName);
  // target_checkout_ref: immutable SHA used by actions/checkout for exact-revision pinning.
  core.setOutput("target_checkout_ref", targetCheckoutRef);
  // target_ref: dispatch-compatible branch/tag ref used by dispatch_workflow.cjs.
  // The GitHub workflow dispatch API requires a branch or tag, not a commit SHA.
  core.setOutput("target_ref", targetDispatchRef);
}

module.exports = { main, parseDispatchRef };
