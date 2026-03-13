// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Resolves the target repository and ref for the activation job checkout.
 *
 * Uses GITHUB_WORKFLOW_REF to determine the platform (host) repository and branch/ref
 * regardless of the triggering event. This fixes cross-repo activation for event-driven
 * relays (e.g. on: issue_comment, on: push) where github.event_name is NOT 'workflow_call',
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
 * In a caller-hosted relay pinned to a feature branch (e.g. uses: platform/.github/workflows/
 * gateway.lock.yml@feature-branch), the @feature-branch portion is encoded in
 * GITHUB_WORKFLOW_REF. Emitting it as target_ref allows the activation checkout to use
 * the correct branch rather than the platform repo's default branch.
 *
 * SEC-005: The targetRepo and targetRef values are resolved solely from trusted system
 * environment variables (GITHUB_WORKFLOW_REF, GITHUB_REPOSITORY, GITHUB_REF) set by the
 * GitHub Actions runtime. They are not derived from user-supplied input, so no allowlist
 * check is required in this handler.
 */

/**
 * @returns {Promise<void>}
 */
async function main() {
  const workflowRef = process.env.GITHUB_WORKFLOW_REF || "";
  const currentRepo = process.env.GITHUB_REPOSITORY || "";

  // GITHUB_WORKFLOW_REF format: owner/repo/.github/workflows/file.yml@ref
  // The regex captures everything before the third slash segment (i.e., the owner/repo prefix).
  const repoMatch = workflowRef.match(/^([^/]+\/[^/]+)\//);
  const workflowRepo = repoMatch ? repoMatch[1] : "";

  // Fall back to currentRepo when GITHUB_WORKFLOW_REF cannot be parsed
  const targetRepo = workflowRepo || currentRepo;

  // Extract the ref portion after '@' from GITHUB_WORKFLOW_REF.
  // GITHUB_WORKFLOW_REF format: owner/repo/.github/workflows/file.yml@ref
  // The ref may be a full ref like "refs/heads/feature-branch", a short name like "main",
  // a tag like "refs/tags/v1.0.0", or a commit SHA like "abc123def".
  //
  // When GITHUB_WORKFLOW_REF has no '@' segment (e.g., env var not set or malformed),
  // fall back to an empty string so that actions/checkout uses the repository's default
  // branch. We intentionally do NOT fall back to GITHUB_REF here because in cross-repo
  // scenarios GITHUB_REF is the *caller* repo's ref, not the callee's, and using it
  // would check out the wrong branch.
  const refMatch = workflowRef.match(/@(.+)$/);
  const targetRef = refMatch ? refMatch[1] : "";

  core.info(`GITHUB_WORKFLOW_REF: ${workflowRef}`);
  core.info(`GITHUB_REPOSITORY: ${currentRepo}`);
  core.info(`Resolved host repo for activation checkout: ${targetRepo}`);
  core.info(`Resolved host ref for activation checkout: ${targetRef}`);

  if (targetRepo !== currentRepo && targetRepo !== "") {
    core.info(`Cross-repo invocation detected: platform repo is "${targetRepo}", caller is "${currentRepo}"`);
    await core.summary.addRaw(`**Activation Checkout**: Checking out platform repo \`${targetRepo}\` @ \`${targetRef}\` (caller: \`${currentRepo}\`)`).write();
  } else {
    core.info(`Same-repo invocation: checking out ${targetRepo} @ ${targetRef}`);
  }

  // Compute the repository name (without owner prefix) for use cases that require
  // only the repo name, such as actions/create-github-app-token which expects
  // `repositories` to contain repo names only when `owner` is also provided.
  const targetRepoName = targetRepo.includes("/") ? targetRepo.substring(targetRepo.indexOf("/") + 1) : targetRepo;

  core.setOutput("target_repo", targetRepo);
  core.setOutput("target_repo_name", targetRepoName);
  core.setOutput("target_ref", targetRef);
}

module.exports = { main };
