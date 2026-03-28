// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Builds the aw_context object that identifies the calling workflow run.
 * This metadata is injected into dispatched workflows that declare an
 * aw_context input, allowing them to trace back to their caller and
 * resolve the current item (issue, pull request, or discussion) that
 * triggered the calling workflow.
 *
 * @returns {{
 *   repo: string,
 *   run_id: string,
 *   workflow_id: string,
 *   workflow_call_id: string,
 *   time: string,
 *   actor: string,
 *   event_type: string,
 *   item_number: string,
 *   comment_id: string
 * }}
 * Properties:
 *   - item_number: Number of the triggering issue, pull request, or discussion.
 *     Empty string for events with no associated item (e.g. push, workflow_dispatch).
 *   - comment_id: ID of the triggering comment for comment events (issue_comment,
 *     pull_request_review_comment, discussion_comment). Empty string otherwise.
 */
function buildAwContext() {
  // Resolve the current item number from the event payload.
  // Covers issues events (payload.issue.number), pull_request events
  // (payload.pull_request.number), discussion events
  // (payload.discussion.number), and comment events where the parent
  // entity number is available on the same paths.
  // Empty string for events with no associated item (e.g. push, workflow_dispatch).
  const itemNumber = context.payload?.issue?.number ?? context.payload?.pull_request?.number ?? context.payload?.discussion?.number;

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
    // item_number identifies the issue, pull request, or discussion that triggered
    // the calling workflow, enabling dispatched workflows to resolve the current item.
    item_number: itemNumber != null ? String(itemNumber) : "",
    // comment_id identifies the specific comment that triggered the calling workflow,
    // when the event was a comment on an issue, pull request, or discussion.
    comment_id: context.payload?.comment?.id != null ? String(context.payload.comment.id) : "",
  };
}

module.exports = { buildAwContext };
