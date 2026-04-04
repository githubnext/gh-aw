// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Resolves the item type, item number, and comment id from the GitHub Actions
 * event payload, covering issues, pull requests, discussions, check runs,
 * check suites, PR reviews, and comment variants.
 *
 * | Event family                              | item_type     | item_number              | comment_id              |
 * |-------------------------------------------|---------------|--------------------------|-------------------------|
 * | issues, issue_comment (on issue)          | issue         | payload.issue.number     | payload.comment.id      |
 * | issue_comment (on PR), pull_request,      | pull_request  | payload.pull_request.    | payload.review.id or    |
 * | pull_request_review, pull_request_review_ |               | number or                | payload.comment.id      |
 * | comment                                   |               | payload.issue.number     |                         |
 * | discussion, discussion_comment            | discussion    | payload.discussion.      | payload.comment.id      |
 * |                                           |               | number                   |                         |
 * | check_run                                 | check_run     | payload.check_run.id     |                         |
 * | check_suite                               | check_suite   | payload.check_suite.id   |                         |
 * | push, workflow_dispatch, …                | (empty)       | (empty)                  |                         |
 *
 * Note: for `issue_comment` events GitHub places the PR data in `payload.issue`
 * with a `payload.issue.pull_request` marker.  Those events are classified as
 * `pull_request` rather than `issue`.
 *
 * @param {object | null | undefined} payload - GitHub Actions context.payload
 * @returns {{ item_type: string, item_number: string, comment_id: string, comment_node_id: string }}
 *   comment_node_id is only populated for discussion/discussion_comment events where
 *   payload.comment.node_id is present (GraphQL node ID needed for reply threading).
 *   It is intentionally empty for all other event types (issues, PRs, checks).
 */
function resolveItemContext(payload) {
  if (payload?.issue != null) {
    // GitHub sends `issue_comment` events for PR comments with the PR data in
    // `payload.issue` and a `payload.issue.pull_request` marker.  Detect this
    // case and classify as pull_request so callers get the correct item type.
    if (payload.issue.pull_request != null) {
      return {
        item_type: "pull_request",
        item_number: payload.issue.number != null ? String(payload.issue.number) : "",
        comment_id: payload.comment?.id != null ? String(payload.comment.id) : "",
        comment_node_id: "",
      };
    }
    return {
      item_type: "issue",
      item_number: payload.issue.number != null ? String(payload.issue.number) : "",
      comment_id: payload.comment?.id != null ? String(payload.comment.id) : "",
      comment_node_id: "",
    };
  }
  if (payload?.pull_request != null) {
    return {
      item_type: "pull_request",
      item_number: payload.pull_request.number != null ? String(payload.pull_request.number) : "",
      // pull_request_review events carry a review object; pull_request_review_comment
      // events carry a comment object.  Both are reported as comment_id.
      comment_id: payload.comment?.id != null ? String(payload.comment.id) : payload.review?.id != null ? String(payload.review.id) : "",
      comment_node_id: "",
    };
  }
  if (payload?.discussion != null) {
    return {
      item_type: "discussion",
      item_number: payload.discussion.number != null ? String(payload.discussion.number) : "",
      comment_id: payload.comment?.id != null ? String(payload.comment.id) : "",
      // comment_node_id is the GraphQL node ID of the triggering discussion comment.
      // It can be used as reply_to_id in add_comment to thread responses under
      // the triggering comment when dispatching specialist workflows.
      comment_node_id: payload.comment?.node_id != null ? String(payload.comment.node_id) : "",
    };
  }
  if (payload?.check_run != null) {
    return {
      item_type: "check_run",
      item_number: payload.check_run.id != null ? String(payload.check_run.id) : "",
      comment_id: "",
      comment_node_id: "",
    };
  }
  if (payload?.check_suite != null) {
    return {
      item_type: "check_suite",
      item_number: payload.check_suite.id != null ? String(payload.check_suite.id) : "",
      comment_id: "",
      comment_node_id: "",
    };
  }
  return { item_type: "", item_number: "", comment_id: "", comment_node_id: "" };
}

/**
 * Builds the aw_context object that identifies the calling workflow run.
 * This metadata is injected into dispatched workflows that declare an
 * aw_context input, allowing them to trace back to their caller and
 * resolve the current item (issue, pull request, discussion, check, etc.)
 * that triggered the calling workflow.
 *
 * @returns {{
 *   repo: string,
 *   run_id: string,
 *   workflow_id: string,
 *   workflow_call_id: string,
 *   time: string,
 *   actor: string,
 *   event_type: string,
 *   item_type: string,
 *   item_number: string,
 *   comment_id: string,
 *   comment_node_id: string,
 *   otel_trace_id: string
 * }}
 * Properties:
 *   - item_type: Kind of entity that triggered the workflow (issue, pull_request,
 *     discussion, check_run, check_suite). Empty string for events with no item
 *     (e.g. push, workflow_dispatch).
 *   - item_number: Sequential number of the item (issue/PR/discussion) or database
 *     id (check_run/check_suite). Empty string when item_type is empty.
 *   - comment_id: ID of the triggering comment or review. Empty string when the
 *     event is not a comment/review event.
 *   - comment_node_id: GraphQL node ID of the triggering discussion comment.
 *     Only populated for discussion/discussion_comment events. Can be passed
 *     as reply_to_id in add_comment to thread responses under the triggering
 *     comment when a dispatched specialist workflow replies to a discussion.
 *   - otel_trace_id: OTLP trace ID from the parent workflow's setup span.
 *     Empty string when OTLP is not configured or the parent setup step has
 *     not yet run.  Used by child workflow setup steps to continue the same
 *     trace as the parent (composite-action trace propagation).
 */
function buildAwContext() {
  const { item_type, item_number, comment_id, comment_node_id } = resolveItemContext(context.payload);

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
    item_type,
    item_number,
    comment_id,
    comment_node_id,
    // Propagate the current OTLP trace ID to dispatched child workflows so that
    // composite actions share the same trace as their parent.  Empty string when
    // OTLP is not configured or the parent setup step has not run yet.
    otel_trace_id: process.env.GITHUB_AW_OTEL_TRACE_ID || "",
  };
}

module.exports = { buildAwContext, resolveItemContext };
