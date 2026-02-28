// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Action outputs for safe-output results
 *
 * Emits individual named step outputs for the first successful result of each
 * safe output type. These outputs allow workflow_call callers to access specific
 * results without having to parse the temporary_id_map.
 */

/**
 * @typedef {Object} ProcessingResultItem
 * @property {boolean} success
 * @property {string} type
 * @property {any} result
 */

/**
 * @typedef {Object} ProcessingResult
 * @property {ProcessingResultItem[]} results
 */

/**
 * Emit individual named step outputs for the first successful result of each
 * safe output type in processingResult.
 *
 * Output names:
 *   create_issue              → created_issue_number, created_issue_url
 *   create_pull_request       → created_pr_number, created_pr_url
 *   add_comment               → comment_id, comment_url
 *   push_to_pull_request_branch → push_commit_sha, push_commit_url
 *
 * @param {ProcessingResult} processingResult - Result from processMessages()
 */
function emitSafeOutputActionOutputs(processingResult) {
  const successfulResults = processingResult.results.filter(r => r.success && r.result);

  // create_issue: created_issue_number, created_issue_url
  const firstIssueResult = successfulResults.find(r => r.type === "create_issue");
  if (firstIssueResult?.result && !Array.isArray(firstIssueResult.result)) {
    const r = firstIssueResult.result;
    if (r.number != null) {
      core.setOutput("created_issue_number", String(r.number));
      core.info(`Exported created_issue_number: ${r.number}`);
    }
    if (r.url) {
      core.setOutput("created_issue_url", r.url);
      core.info(`Exported created_issue_url: ${r.url}`);
    }
  }

  // create_pull_request: created_pr_number, created_pr_url
  const firstPRResult = successfulResults.find(r => r.type === "create_pull_request");
  if (firstPRResult?.result && !Array.isArray(firstPRResult.result)) {
    const r = firstPRResult.result;
    if (r.pull_request_number != null) {
      core.setOutput("created_pr_number", String(r.pull_request_number));
      core.info(`Exported created_pr_number: ${r.pull_request_number}`);
    }
    if (r.pull_request_url) {
      core.setOutput("created_pr_url", r.pull_request_url);
      core.info(`Exported created_pr_url: ${r.pull_request_url}`);
    }
  }

  // add_comment: comment_id, comment_url
  // add_comment handlers may return an array when multiple comments were queued for the same
  // message (e.g., fallback retries). We take the first element to get the primary comment.
  const firstCommentResult = successfulResults.find(r => r.type === "add_comment");
  if (firstCommentResult?.result) {
    const r = Array.isArray(firstCommentResult.result) ? firstCommentResult.result[0] : firstCommentResult.result;
    if (r?.commentId != null) {
      core.setOutput("comment_id", String(r.commentId));
      core.info(`Exported comment_id: ${r.commentId}`);
    }
    if (r?.url) {
      core.setOutput("comment_url", r.url);
      core.info(`Exported comment_url: ${r.url}`);
    }
  }

  // push_to_pull_request_branch: push_commit_sha, push_commit_url
  const firstPushResult = successfulResults.find(r => r.type === "push_to_pull_request_branch");
  if (firstPushResult?.result && !Array.isArray(firstPushResult.result)) {
    const r = firstPushResult.result;
    if (r.commit_sha) {
      core.setOutput("push_commit_sha", r.commit_sha);
      core.info(`Exported push_commit_sha: ${r.commit_sha}`);
    }
    if (r.commit_url) {
      core.setOutput("push_commit_url", r.commit_url);
      core.info(`Exported push_commit_url: ${r.commit_url}`);
    }
  }
}

module.exports = { emitSafeOutputActionOutputs };
