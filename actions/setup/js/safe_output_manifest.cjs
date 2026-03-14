// @ts-check

const fs = require("fs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { ERR_SYSTEM } = require("./error_codes.cjs");

/**
 * Default path for the safe output items manifest file.
 * This file records every item created in GitHub by safe output handlers.
 */
const MANIFEST_FILE_PATH = "/tmp/safe-output-items.jsonl";

/**
 * Safe output types that create new items in GitHub (these typically return a URL,
 * but the URL may be omitted in some cases).
 * This is a subset of LOGGED_TYPES kept for backward compatibility.
 * @type {Set<string>}
 */
const CREATE_ITEM_TYPES = new Set([
  "create_issue",
  "add_comment",
  "create_discussion",
  "create_pull_request",
  "create_project",
  "create_project_status_update",
  "create_pull_request_review_comment",
  "submit_pull_request_review",
  "reply_to_pull_request_review_comment",
  "create_code_scanning_alert",
  "autofix_code_scanning_alert",
]);

/**
 * All safe output types that should be logged to the manifest, including both
 * creation operations (which typically return a URL, but may omit it) and
 * modification operations (which operate on existing GitHub items and may not
 * return a URL).
 *
 * Excludes purely internal types that do not represent a GitHub state change:
 * - noop: no-op, produces no GitHub side effects
 * - missing_tool / missing_data: meta-operations that signal missing capabilities
 * @type {Set<string>}
 */
const LOGGED_TYPES = new Set([
  // Creation types (also in CREATE_ITEM_TYPES)
  "create_issue",
  "add_comment",
  "create_discussion",
  "create_pull_request",
  "create_project",
  "create_project_status_update",
  "create_pull_request_review_comment",
  "submit_pull_request_review",
  "reply_to_pull_request_review_comment",
  "create_code_scanning_alert",
  "autofix_code_scanning_alert",
  "create_missing_tool_issue",
  "create_missing_data_issue",
  // Modification/action types
  "close_issue",
  "close_discussion",
  "add_labels",
  "remove_labels",
  "update_issue",
  "update_discussion",
  "link_sub_issue",
  "update_release",
  "resolve_pull_request_review_thread",
  "push_to_pull_request_branch",
  "update_pull_request",
  "close_pull_request",
  "mark_pull_request_as_ready_for_review",
  "hide_comment",
  "set_issue_type",
  "add_reviewer",
  "assign_milestone",
  "assign_to_user",
  "unassign_from_user",
  "dispatch_workflow",
  "update_project",
]);

/**
 * @typedef {Object} ManifestEntry
 * @property {string} type - The safe output type (e.g., "create_issue")
 * @property {string} [url] - URL of the affected item in GitHub (present for creation types; omitted for modification types that don't return a URL)
 * @property {number} [number] - Issue/PR/discussion number if applicable
 * @property {string} [repo] - Repository slug (owner/repo) if applicable
 * @property {string} [temporaryId] - Temporary ID assigned to this item, if any
 * @property {string} timestamp - ISO 8601 timestamp of creation
 */

/**
 * Create a manifest logger function for recording executed safe output items.
 *
 * The logger writes JSONL entries to the specified manifest file.
 * It is designed to be easily testable by accepting the file path as a parameter.
 *
 * @param {string} [manifestFile] - Path to the manifest file (defaults to MANIFEST_FILE_PATH)
 * @returns {(item: {type: string, url?: string, number?: number, repo?: string, temporaryId?: string}) => void} Logger function
 */
function createManifestLogger(manifestFile = MANIFEST_FILE_PATH) {
  // Touch the file immediately so it exists for artifact upload
  // even if no items are created during this run.
  ensureManifestExists(manifestFile);

  /**
   * Log an executed safe output item to the manifest file.
   *
   * @param {{type: string, url?: string, number?: number, repo?: string, temporaryId?: string}} item - Executed item details
   */
  return function logCreatedItem(item) {
    if (!item) return;

    /** @type {ManifestEntry} */
    const entry = {
      type: item.type,
      ...(item.url ? { url: item.url } : {}),
      ...(item.number != null ? { number: item.number } : {}),
      ...(item.repo ? { repo: item.repo } : {}),
      ...(item.temporaryId ? { temporaryId: item.temporaryId } : {}),
      timestamp: new Date().toISOString(),
    };

    const jsonLine = JSON.stringify(entry) + "\n";
    try {
      fs.appendFileSync(manifestFile, jsonLine);
    } catch (error) {
      throw new Error(`${ERR_SYSTEM}: Failed to write to manifest file: ${getErrorMessage(error)}`);
    }
  };
}

/**
 * Ensure the manifest file exists, creating an empty file if it does not.
 * This should be called at the end of safe output processing to guarantee
 * the artifact upload always has a file to upload.
 *
 * @param {string} [manifestFile] - Path to the manifest file (defaults to MANIFEST_FILE_PATH)
 */
function ensureManifestExists(manifestFile = MANIFEST_FILE_PATH) {
  if (!fs.existsSync(manifestFile)) {
    try {
      fs.writeFileSync(manifestFile, "");
    } catch (error) {
      throw new Error(`${ERR_SYSTEM}: Failed to create manifest file: ${getErrorMessage(error)}`);
    }
  }
}

/**
 * Extract executed item details from a handler result for manifest logging.
 * Returns null if the type is not tracked (not in LOGGED_TYPES) or if the
 * result is from a staged (preview) run where no item was actually modified.
 *
 * For creation types (CREATE_ITEM_TYPES), the result URL is included when present.
 * For modification types (e.g. add_labels, close_issue), the URL is optional.
 *
 * @param {string} type - The handler type (e.g., "create_issue")
 * @param {any} result - The handler result object
 * @returns {{type: string, url?: string, number?: number, repo?: string, temporaryId?: string}|null}
 */
function extractCreatedItemFromResult(type, result) {
  if (!result || !LOGGED_TYPES.has(type)) return null;

  // In staged mode (🎭 Staged Mode Preview), no item was actually modified in GitHub — skip logging
  if (result.staged === true) return null;

  // Normalize URL from different result shapes (present for creation types)
  const url = result.url || result.projectUrl || result.html_url;

  return {
    type,
    ...(url ? { url } : {}),
    ...(result.number != null ? { number: result.number } : {}),
    ...(result.repo ? { repo: result.repo } : {}),
    ...(result.temporaryId ? { temporaryId: result.temporaryId } : {}),
  };
}

module.exports = {
  MANIFEST_FILE_PATH,
  CREATE_ITEM_TYPES,
  LOGGED_TYPES,
  createManifestLogger,
  ensureManifestExists,
  extractCreatedItemFromResult,
};
