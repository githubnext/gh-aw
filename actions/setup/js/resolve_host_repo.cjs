// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Resolves the target repository and ref for the activation job checkout.
 *
 * Uses the job.workflow_* context fields to determine the platform (host)
 * repository and pin the checkout to the exact executing commit SHA.
 *
 * These fields are passed via environment variables (JOB_WORKFLOW_REPOSITORY,
 * JOB_WORKFLOW_SHA, etc.) to avoid shell injection — the ${{ }} expressions
 * are evaluated in the env: block, not interpolated into script source.
 *
 * job.workflow_repository provides the owner/repo of the currently executing
 * workflow file, correctly identifying the platform repo in all relay patterns:
 * cross-repo workflow_call, event-driven relays (on: issue_comment, on: push),
 * and cross-org scenarios.
 *
 * job.workflow_sha provides the immutable commit SHA of the workflow being
 * executed, ensuring the activation checkout pins to the exact revision rather
 * than a moving branch/tag ref.
 *
 * @safe-outputs-exempt SEC-005: values sourced from trusted GitHub Actions runner context via env vars only
 */

/**
 * @returns {Promise<void>}
 */
async function main() {
  const targetRepo = process.env.JOB_WORKFLOW_REPOSITORY || "";
  const targetRef = process.env.JOB_WORKFLOW_SHA || "";
  const targetRepoName = targetRepo.split("/").pop() || "";
  const currentRepo = process.env.GITHUB_REPOSITORY || "";

  core.info("Resolving host repo via job.workflow_* context");
  core.info(`job.workflow_repository = ${targetRepo}`);
  core.info(`job.workflow_sha        = ${targetRef}`);
  core.info(`job.workflow_ref        = ${process.env.JOB_WORKFLOW_REF || ""}`);
  core.info(`job.workflow_file_path  = ${process.env.JOB_WORKFLOW_FILE_PATH || ""}`);
  core.info(`github.repository       = ${currentRepo}`);
  core.info("");
  core.info(`Resolved target_repo      = ${targetRepo}`);
  core.info(`Resolved target_repo_name = ${targetRepoName}`);
  core.info(`Resolved target_ref       = ${targetRef}`);

  if (targetRepo && targetRepo !== currentRepo) {
    core.info(`Cross-repo invocation detected: platform repo "${targetRepo}" differs from caller "${currentRepo}"`);
  } else {
    core.info(`Same-repo invocation: platform and caller are both "${targetRepo}"`);
  }

  core.setOutput("target_repo", targetRepo);
  core.setOutput("target_repo_name", targetRepoName);
  core.setOutput("target_ref", targetRef);
}

module.exports = { main };
