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
 * GITHUB_WORKFLOW_REF reflects the currently executing workflow file for most triggers, but
 * in cross-org workflow_call scenarios it resolves to the TOP-LEVEL CALLER's workflow ref,
 * not the reusable (callee) workflow being executed. Its format is:
 *   owner/repo/.github/workflows/file.yml@refs/heads/main
 *
 * When the platform workflow runs cross-repo (called via uses: from the same org),
 * GITHUB_WORKFLOW_REF starts with the platform repo slug, while GITHUB_REPOSITORY is the
 * caller repo. Comparing the two lets us detect cross-repo invocations without relying on
 * event_name.
 *
 * For cross-org workflow_call, GITHUB_WORKFLOW_REF and GITHUB_REPOSITORY both resolve to
 * the caller's repo. In that case we fall back to the referenced_workflows API lookup to
 * find the actual callee (platform) repo and ref.
 *
 * In a caller-hosted relay pinned to a feature branch (e.g. uses: platform/.github/workflows/
 * gateway.lock.yml@feature-branch), the @feature-branch portion is encoded in
 * GITHUB_WORKFLOW_REF. Emitting it as target_ref allows the activation checkout to use
 * the correct branch rather than the platform repo's default branch.
 *
 * SEC-005: The targetRepo and targetRef values are resolved solely from trusted system
 * environment variables (GITHUB_WORKFLOW_REF, GITHUB_REPOSITORY, GITHUB_REF) and the
 * GitHub Actions API (referenced_workflows), set/provided by the GitHub Actions runtime.
 * They are not derived from user-supplied input, so no allowlist check is required here.
 *
 * @safe-outputs-exempt SEC-005: values sourced from trusted GitHub Actions runtime env vars and referenced_workflows API only
 */

// Matches the "owner/repo" prefix from a GitHub workflow path of the form "owner/repo/...".
const REPO_PREFIX_RE = /^([^/]+\/[^/]+)\//;

/**
 * Attempts to resolve the callee repository and ref from the referenced_workflows API.
 *
 * This is used as a fallback when GITHUB_WORKFLOW_REF points to the same repo as
 * GITHUB_REPOSITORY (cross-org workflow_call scenario), because in that case
 * GITHUB_WORKFLOW_REF reflects the caller's workflow ref, not the callee's.
 *
 * @param {string} currentRepo - The value of GITHUB_REPOSITORY (owner/repo)
 * @returns {Promise<{repo: string, ref: string} | null>} Resolved callee repo and ref, or null
 */
async function resolveFromReferencedWorkflows(currentRepo) {
  const rawRunId = process.env.GITHUB_RUN_ID;
  const runId = rawRunId ? parseInt(rawRunId, 10) : typeof context.runId === "number" ? context.runId : NaN;
  if (!Number.isFinite(runId)) {
    core.info("Run ID is unavailable or invalid, cannot perform referenced_workflows lookup");
    return null;
  }

  const [runOwner, runRepo] = currentRepo.split("/");
  try {
    core.info(`Checking for cross-org callee via referenced_workflows API (run ${runId}, repo ${currentRepo})`);
    const runResponse = await github.rest.actions.getWorkflowRun({
      owner: runOwner,
      repo: runRepo,
      run_id: runId,
    });

    const referencedWorkflows = runResponse.data.referenced_workflows || [];
    core.info(`Found ${referencedWorkflows.length} referenced workflow(s) in run`);
    for (const wf of referencedWorkflows) {
      core.info(`  referenced workflow: path=${wf.path} sha=${wf.sha || "(none)"} ref=${wf.ref || "(none)"}`);
    }

    // Collect all referenced workflows from a different repo than the caller.
    // In cross-org workflow_call, the callee (platform) repo is different from currentRepo.
    // If multiple cross-repo candidates are found we cannot safely pick one, so we bail out.
    const crossRepoCandidates = [];
    for (const wf of referencedWorkflows) {
      const pathRepoMatch = wf.path.match(REPO_PREFIX_RE);
      const entryRepo = pathRepoMatch ? pathRepoMatch[1] : "";
      if (entryRepo && entryRepo !== currentRepo) {
        crossRepoCandidates.push({ wf, repo: entryRepo });
      }
    }
    core.info(`Found ${crossRepoCandidates.length} cross-repo candidate(s) (excluding current repo ${currentRepo})`);

    if (crossRepoCandidates.length === 0) {
      core.info("No cross-org callee found in referenced_workflows, using current repo");
      return null;
    }

    if (crossRepoCandidates.length > 1) {
      core.info(`Referenced workflows lookup is ambiguous; found ${crossRepoCandidates.length} cross-repo candidates, not selecting one`);
      for (const candidate of crossRepoCandidates) {
        core.info(`  Candidate referenced workflow path: ${candidate.wf.path}`);
      }
      return null;
    }

    const matchingEntry = crossRepoCandidates[0].wf;
    const calleeRepo = crossRepoCandidates[0].repo;

    // Prefer sha (immutable) over ref (branch/tag can drift) over path-parsed ref.
    const pathRefMatch = matchingEntry.path.match(/@(.+)$/);
    let calleeRefSource;
    if (matchingEntry.sha) {
      calleeRefSource = "sha";
    } else if (matchingEntry.ref) {
      calleeRefSource = "ref";
    } else if (pathRefMatch) {
      calleeRefSource = "path";
    } else {
      calleeRefSource = "none";
    }
    const calleeRef = matchingEntry.sha || matchingEntry.ref || (pathRefMatch ? pathRefMatch[1] : "");
    core.info(`Resolved callee repo from referenced_workflows: ${calleeRepo} @ ${calleeRef || "(default branch)"} (source: ${calleeRefSource})`);
    core.info(`  Referenced workflow path: ${matchingEntry.path}`);
    return { repo: calleeRepo, ref: calleeRef };
  } catch (error) {
    const msg = error instanceof Error ? error.message : String(error);
    core.info(`Could not fetch referenced_workflows from API: ${msg}, using current repo`);
    return null;
  }
}

/**
 * @returns {Promise<void>}
 */
async function main() {
  const workflowRef = process.env.GITHUB_WORKFLOW_REF || "";
  const currentRepo = process.env.GITHUB_REPOSITORY || "";

  core.info(`GITHUB_WORKFLOW_REF: ${workflowRef || "(not set)"}`);
  core.info(`GITHUB_REPOSITORY: ${currentRepo || "(not set)"}`);
  core.info(`GITHUB_RUN_ID: ${process.env.GITHUB_RUN_ID || "(not set)"}`);

  // GITHUB_WORKFLOW_REF format: owner/repo/.github/workflows/file.yml@ref
  // The regex captures everything before the third slash segment (i.e., the owner/repo prefix).
  const repoMatch = workflowRef.match(REPO_PREFIX_RE);
  const workflowRepo = repoMatch ? repoMatch[1] : "";
  core.info(`Parsed workflow repo from GITHUB_WORKFLOW_REF: ${workflowRepo || "(could not parse)"}`);

  // Fall back to currentRepo when GITHUB_WORKFLOW_REF cannot be parsed
  let targetRepo = workflowRepo || currentRepo;

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
  let targetRef = refMatch ? refMatch[1] : "";
  core.info(`Parsed workflow ref from GITHUB_WORKFLOW_REF: ${targetRef || "(none — will use default branch)"}`);

  // Cross-org workflow_call detection: when GITHUB_WORKFLOW_REF points to the same repo as
  // GITHUB_REPOSITORY, it means GITHUB_WORKFLOW_REF is resolving to the caller's workflow
  // (not the callee's). This happens in cross-org workflow_call invocations where GitHub
  // Actions sets GITHUB_WORKFLOW_REF to the top-level caller's workflow ref rather than the
  // reusable workflow being executed. In that case, fall back to the referenced_workflows API
  // to find the actual callee (platform) repo and ref.
  //
  // Note: GITHUB_EVENT_NAME inside a reusable workflow reflects the ORIGINAL trigger event
  // (e.g., "push", "issues"), NOT "workflow_call", so we cannot use event_name to detect
  // this scenario.
  if (workflowRepo && workflowRepo === currentRepo) {
    core.info(`Cross-org workflow_call detected (workflowRepo === currentRepo = ${currentRepo}): falling back to referenced_workflows API`);
    const resolved = await resolveFromReferencedWorkflows(currentRepo);
    if (resolved) {
      targetRepo = resolved.repo;
      targetRef = resolved.ref || targetRef;
    } else {
      core.info("referenced_workflows lookup returned no result; keeping current repo as target");
    }
  } else if (!workflowRepo) {
    core.info("Could not parse workflowRepo from GITHUB_WORKFLOW_REF; falling back to GITHUB_REPOSITORY");
  } else {
    core.info(`Same-org cross-repo invocation: workflowRepo=${workflowRepo}, currentRepo=${currentRepo}`);
  }

  core.info(`Resolved host repo for activation checkout: ${targetRepo}`);
  core.info(`Resolved host ref for activation checkout: ${targetRef || "(default branch)"}`);

  if (targetRepo !== currentRepo && targetRepo !== "") {
    core.info(`Cross-repo invocation detected: platform repo is "${targetRepo}", caller is "${currentRepo}"`);
    await core.summary.addRaw(`**Activation Checkout**: Checking out platform repo \`${targetRepo}\` @ \`${targetRef}\` (caller: \`${currentRepo}\`)`).write();
  } else {
    core.info(`Same-repo invocation: checking out ${targetRepo} @ ${targetRef || "(default branch)"}`);
  }

  // Compute the repository name (without owner prefix) for use cases that require
  // only the repo name, such as actions/create-github-app-token which expects
  // `repositories` to contain repo names only when `owner` is also provided.
  const targetRepoName = targetRepo.split("/").at(-1);
  core.info(`target_repo=${targetRepo} target_repo_name=${targetRepoName} target_ref=${targetRef || "(default branch)"}`);

  core.setOutput("target_repo", targetRepo);
  core.setOutput("target_repo_name", targetRepoName);
  core.setOutput("target_ref", targetRef);
}

module.exports = { main };
