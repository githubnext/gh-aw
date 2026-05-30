// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Generate History Link Module
 *
 * This module provides functions for generating GitHub search URLs that allow
 * users to view all items (issues, pull requests, discussions) created by a
 * specific workflow run. The history link uses the workflow-call-id or
 * workflow-id XML marker to filter results to a specific workflow.
 */

/**
 * @typedef {"issue" | "pull_request" | "discussion" | "comment" | "discussion_comment"} ItemType
 */

/**
 * @typedef {Object} HistoryLinkParams
 * @property {string} owner - Repository owner
 * @property {string} repo - Repository name
 * @property {ItemType} itemType - Type of GitHub item: "issue", "pull_request", "discussion", "comment", or "discussion_comment"
 * @property {string} [workflowCallId] - Caller workflow ID (e.g. "owner/repo/WorkflowName"). Takes precedence over workflowId.
 * @property {string} [workflowId] - Workflow identifier. Used when workflowCallId is not available.
 * @property {string} [serverUrl] - GitHub server URL for enterprise deployments (e.g. "https://github.example.com"). Defaults to "https://github.com".
 */

/**
 * Generate a GitHub search URL for finding all items of a given type created by a workflow.
 *
 * The search URL uses the workflow-call-id marker when available (preferred, as it
 * distinguishes callers that share the same reusable workflow), falling back to
 * the workflow-id marker. Results are filtered to the specified item type but are
 * not filtered by open/closed state.
 *
 * Enterprise GitHub deployments are supported via the serverUrl parameter or the
 * GITHUB_SERVER_URL environment variable.
 *
 * @param {HistoryLinkParams} params
 * @returns {string | null} The search URL, or null if no workflow ID is available
 */
function generateHistoryUrl({ owner, repo, itemType, workflowCallId, workflowId, serverUrl }) {
  // Prefer caller workflow ID for more specific matching; fall back to workflow ID
  const markerId = workflowCallId ? `gh-aw-workflow-call-id: ${workflowCallId}` : workflowId ? `gh-aw-workflow-id: ${workflowId}` : null;

  if (!markerId) {
    return null;
  }

  const server = serverUrl || process.env.GITHUB_SERVER_URL || "https://github.com";

  // Build the search query parts
  const queryParts = [`repo:${owner}/${repo}`];

  // Add item type qualifier (issues use is:issue qualifier; discussions and comments do not)
  if (itemType === "issue") {
    queryParts.push("is:issue");
  }

  queryParts.push(`"${markerId}"`);

  const url = new URL(`${server}/search`);
  url.searchParams.set("q", queryParts.join(" "));

  // Set the type parameter based on itemType for correct GitHub search filtering
  const searchTypeMap = { issue: "issues", pull_request: "pullrequests", discussion: "discussions", comment: "issues", discussion_comment: "discussions" };
  url.searchParams.set("type", searchTypeMap[itemType] ?? "issues");

  return url.toString();
}

/**
 * Generate a markdown history link for use in GitHub item footers.
 *
 * The link opens a GitHub search page filtered to items of the same type created by
 * the same workflow. Returns null if no workflow ID is available (so callers can
 * conditionally include the link).
 *
 * @param {HistoryLinkParams} params
 * @returns {string | null} Markdown link (e.g. "[history](url)"), or null if unavailable
 */
function generateHistoryLink(params) {
  const url = generateHistoryUrl(params);
  if (!url) {
    return null;
  }
  return `[history](${url})`;
}

module.exports = {
  generateHistoryUrl,
  generateHistoryLink,
};
