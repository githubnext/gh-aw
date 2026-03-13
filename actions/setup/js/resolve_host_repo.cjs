// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Resolves the target repository for the activation job checkout.
 *
 * Uses GITHUB_WORKFLOW_REF to determine the platform (host) repository regardless
 * of the triggering event. This fixes cross-repo activation for event-driven relays
 * (e.g. on: issue_comment, on: push) where github.event_name is NOT 'workflow_call',
 * so the expression introduced in #20301 incorrectly fell back to github.repository
 * (the caller's repo) instead of the platform repo.
 *
 * GITHUB_WORKFLOW_REF always reflects the currently executing workflow file, not the
 * triggering event. Its format is:
 *   owner/repo/.github/workflows/file.yml@refs/heads/main
 *
 * When the platform workflow runs cross-repo (called via uses:), GITHUB_WORKFLOW_REF
 * starts with the platform repo slug, while GITHUB_REPOSITORY is the caller repo.
 * Comparing the two lets us detect cross-repo invocations without relying on event_name.
 *
 * SEC-005: The targetRepo value is resolved solely from trusted system environment
 * variables (GITHUB_WORKFLOW_REF, GITHUB_REPOSITORY) set by the GitHub Actions
 * runtime. It is not derived from user-supplied input, so no validateTargetRepo
 * allowlist check is required in this handler.
 */

/**
 * @returns {Promise<void>}
 */
async function main() {
  const workflowRef = process.env.GITHUB_WORKFLOW_REF || "";
  const currentRepo = process.env.GITHUB_REPOSITORY || "";

  // GITHUB_WORKFLOW_REF format: owner/repo/.github/workflows/file.yml@ref
  // The regex captures everything before the third slash segment (i.e., the owner/repo prefix).
  const match = workflowRef.match(/^([^/]+\/[^/]+)\//);
  const workflowRepo = match ? match[1] : "";

  // Fall back to currentRepo when GITHUB_WORKFLOW_REF cannot be parsed
  const targetRepo = workflowRepo || currentRepo;

  core.info(`GITHUB_WORKFLOW_REF: ${workflowRef}`);
  core.info(`GITHUB_REPOSITORY: ${currentRepo}`);
  core.info(`Resolved host repo for activation checkout: ${targetRepo}`);

  if (targetRepo !== currentRepo && targetRepo !== "") {
    core.info(`Cross-repo invocation detected: platform repo is "${targetRepo}", caller is "${currentRepo}"`);
    await core.summary.addRaw(`**Activation Checkout**: Checking out platform repo \`${targetRepo}\` (caller: \`${currentRepo}\`)`).write();
  } else {
    core.info(`Same-repo invocation: checking out ${targetRepo}`);
  }

  core.setOutput("target_repo", targetRepo);
}

module.exports = { main };
