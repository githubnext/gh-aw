// @ts-check

const fs = require("fs");
const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Default path for the safe output items manifest file.
 * This file records every item created in GitHub by safe output handlers.
 */
const MANIFEST_FILE_PATH = "/tmp/safe-output-items.jsonl";

/**
 * Safe output types that create items in GitHub.
 * These are the types that should be logged to the manifest.
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
 * @typedef {Object} ManifestEntry
 * @property {string} type - The safe output type (e.g., "create_issue")
 * @property {string} url - URL of the created item in GitHub
 * @property {number} [number] - Issue/PR/discussion number if applicable
 * @property {string} [repo] - Repository slug (owner/repo) if applicable
 * @property {string} [temporaryId] - Temporary ID assigned to this item, if any
 * @property {string} timestamp - ISO 8601 timestamp of creation
 */

/**
 * Create a manifest logger function for recording created items.
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
   * Log a created item to the manifest file.
   * Items without a URL are silently skipped.
   *
   * @param {{type: string, url?: string, number?: number, repo?: string, temporaryId?: string}} item - Created item details
   */
  return function logCreatedItem(item) {
    if (!item || !item.url) return;

    /** @type {ManifestEntry} */
    const entry = {
      type: item.type,
      url: item.url,
      ...(item.number != null ? { number: item.number } : {}),
      ...(item.repo ? { repo: item.repo } : {}),
      ...(item.temporaryId ? { temporaryId: item.temporaryId } : {}),
      timestamp: new Date().toISOString(),
    };

    const jsonLine = JSON.stringify(entry) + "\n";
    try {
      fs.appendFileSync(manifestFile, jsonLine);
    } catch (error) {
      throw new Error(`Failed to write to manifest file: ${getErrorMessage(error)}`);
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
      throw new Error(`Failed to create manifest file: ${getErrorMessage(error)}`);
    }
  }
}

/**
 * Extract created item details from a handler result for manifest logging.
 * Returns null if the result does not represent a created item with a URL,
 * or if the result is from a staged (preview) run where no item was actually created.
 *
 * @param {string} type - The handler type (e.g., "create_issue")
 * @param {any} result - The handler result object
 * @returns {{type: string, url: string, number?: number, repo?: string, temporaryId?: string}|null}
 */
function extractCreatedItemFromResult(type, result) {
  if (!result || !CREATE_ITEM_TYPES.has(type)) return null;

  // In staged mode, no item was actually created in GitHub — skip logging
  if (result.staged === true) return null;

  // Normalize URL from different result shapes
  const url = result.url || result.projectUrl || result.html_url;
  if (!url) return null;

  return {
    type,
    url,
    ...(result.number != null ? { number: result.number } : {}),
    ...(result.repo ? { repo: result.repo } : {}),
    ...(result.temporaryId ? { temporaryId: result.temporaryId } : {}),
  };
}

module.exports = {
  MANIFEST_FILE_PATH,
  CREATE_ITEM_TYPES,
  createManifestLogger,
  ensureManifestExists,
  extractCreatedItemFromResult,
};
