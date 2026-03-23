// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Builds the aw_context object that identifies the calling workflow run.
 * This metadata is injected into dispatched workflows that declare an
 * aw_context input, allowing them to trace back to their caller.
 *
 * @returns {{ repo: string, run_id: string, workflow_id: string, workflow_call_id: string, time: string, actor: string, event_type: string }}
 */
function buildAwContext() {
  return {
    repo: `${context.repo.owner}/${context.repo.repo}`,
    run_id: String(context.runId ?? ""),
    // GITHUB_WORKFLOW_REF provides the full workflow file path including the ref,
    // e.g. "owner/repo/.github/workflows/dispatcher.yml@refs/heads/main"
    workflow_id: process.env.GITHUB_WORKFLOW_REF ?? "",
    // workflow_call_id uniquely identifies this specific call attempt:
    // combine run_id with run_attempt (GITHUB_RUN_ATTEMPT) so re-runs produce different IDs.
    workflow_call_id: `${process.env.GITHUB_RUN_ID ?? context.runId ?? ""}-${process.env.GITHUB_RUN_ATTEMPT ?? "1"}`,
    time: new Date().toISOString(),
    actor: context.actor ?? "",
    event_type: context.eventName ?? "",
  };
}

module.exports = { buildAwContext };
