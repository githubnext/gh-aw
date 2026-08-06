// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Safe Output Summary Generator
 *
 * This module provides functionality to generate step summaries for safe-output messages.
 * Each processed safe-output generates a summary enclosed in a <details> section.
 */

const { displayFileContent } = require("./display_file_helpers.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { computeSafeOutputsStatus } = require("./safe_outputs_status.cjs");
const ERROR_CODES = require("./error_codes.cjs");
const { redactStepSummaryContent } = require("./redact_secrets.cjs");

/**
 * Error codes that may be rendered verbatim in a step summary.
 * Handler errors are built from caught exception messages, which can embed request
 * URLs, payloads, or credentials, so only the machine-readable code prefix is shown.
 * @type {Set<string>}
 */
const SUMMARY_SAFE_ERROR_CODES = new Set(Object.values(ERROR_CODES));

/** @type {string} Rendered when an error carries no allowlisted code prefix */
const UNCLASSIFIED_ERROR_CODE = "UNCLASSIFIED";

/**
 * Reduces an error message to an allowlisted error code so that raw exception text
 * (which may contain secrets) never reaches the step summary.
 * @param {any} error - The error message produced by a safe-output handler
 * @returns {string} An allowlisted error code
 */
function toSummarySafeErrorCode(error) {
  const text = typeof error === "string" ? error : String(error ?? "");
  const match = text.match(/^\s*([A-Z][A-Z0-9_]*)\s*:/);
  if (match && SUMMARY_SAFE_ERROR_CODES.has(match[1])) {
    return match[1];
  }
  return UNCLASSIFIED_ERROR_CODE;
}

/**
 * Generate a step summary for a single safe-output message
 * @param {Object} options - Summary generation options
 * @param {string} options.type - The safe-output type (e.g., "create_issue", "create_project")
 * @param {number} options.messageIndex - The message index (1-based)
 * @param {boolean} options.success - Whether the message was processed successfully
 * @param {any} options.result - The result from the handler
 * @param {any} options.message - The original message
 * @param {string} [options.error] - Error message if processing failed
 * @returns {string} - Markdown content for the step summary
 */
function generateSafeOutputSummary(options) {
  const { type, messageIndex, success, result, message, error } = options;

  // Format the type for display (e.g., "create_issue" -> "Create Issue")
  const displayType = type
    .split("_")
    .map(word => word.charAt(0).toUpperCase() + word.slice(1))
    .join(" ");

  // Detect fallback outcomes for code-push types.
  // Prefer explicit fallback_type when available; infer only for backward compatibility.
  const isDuplicateDrop = success && result && result.dropped_duplicate === true;
  const isFallback = success && result && result.fallback_used === true;
  const inferredFallbackType = isFallback && (result.pull_request_url || result.pull_request_number != null) ? "pull_request" : "issue";
  const fallbackType = isFallback && result?.fallback_type ? result.fallback_type : inferredFallbackType;

  // Choose emoji and status based on success and fallback
  const emoji = isDuplicateDrop ? "⚠️" : isFallback ? "⚠️" : success ? "✅" : "❌";
  const status = isDuplicateDrop ? "Duplicate Dropped" : isFallback ? (fallbackType === "pull_request" ? "Fallback Pull Request Created" : "Fallback Issue Created") : success ? "Success" : "Failed";

  // Start building the summary
  let summary = `<details>\n<summary>${emoji} ${displayType} - ${status} (Message ${messageIndex})</summary>\n\n`;

  // Add message details
  const sectionTitle = isFallback ? `### ${displayType} — ${fallbackType === "pull_request" ? "Fallback Pull Request" : "Fallback Issue"}\n\n` : `### ${displayType}\n\n`;
  summary += sectionTitle;

  if (isDuplicateDrop) {
    summary += `> ℹ️ Duplicate issue title was dropped by title-based deduplication.\n\n`;
    if (result.title || message?.title) {
      summary += `**Title:** ${result.title || message?.title}\n\n`;
    }
    if (result.duplicate_of_title) {
      summary += `**Matched Existing Title:** ${result.duplicate_of_title}\n\n`;
    }
    if (result.duplicate_distance !== undefined) {
      summary += `**Levenshtein Distance:** ${result.duplicate_distance}\n\n`;
    }
    if (result.dedup_source) {
      summary += `**Dedup Source:** ${result.dedup_source}\n\n`;
    }
  } else if (isFallback) {
    // Explain why the fallback occurred and show the created fallback target
    if (fallbackType === "pull_request") {
      summary += `> ℹ️ Direct push to the original pull request branch was not possible (diverged/non-fast-forward). A fallback pull request was created instead.\n\n`;
      if (result.pull_request_url) {
        summary += `**Fallback Pull Request:** ${result.pull_request_url}\n\n`;
      }
      if (result.pull_request_number != null && result.repo) {
        summary += `**Location:** ${result.repo}#${result.pull_request_number}\n\n`;
      }
    } else {
      summary += `> ℹ️ Pull request creation was blocked due to protected file changes. A review issue was created instead.\n\n`;
      if (result.issue_url) {
        summary += `**Fallback Issue:** ${result.issue_url}\n\n`;
      }
      if (result.issue_number != null && result.repo) {
        summary += `**Location:** ${result.repo}#${result.issue_number}\n\n`;
      }
    }
    if (result.branch_name) {
      summary += `**Branch:** \`${result.branch_name}\`\n\n`;
    }

    // Add original message details if available
    if (message) {
      if (message.title) {
        summary += `**Title:** ${message.title}\n\n`;
      }
    }
  } else if (success && result) {
    // Add result-specific information based on type
    if (result.url) {
      summary += `**URL:** ${result.url}\n\n`;
    }
    if (result.repo && result.number) {
      summary += `**Location:** ${result.repo}#${result.number}\n\n`;
    }
    if (result.projectUrl) {
      summary += `**Project URL:** ${result.projectUrl}\n\n`;
    }
    if (result.temporaryId) {
      summary += `**Temporary ID:** \`${result.temporaryId}\`\n\n`;
    }

    // Add original message details if available
    if (message) {
      if (message.title) {
        summary += `**Title:** ${message.title}\n\n`;
      }
      if (message.labels && Array.isArray(message.labels)) {
        summary += `**Labels:** ${message.labels.join(", ")}\n\n`;
      }
    }
  } else if (error) {
    // Show only an allowlisted error code; raw exception text and message content are
    // omitted because they can embed URLs, payloads, or credentials.
    summary += `**Error:** \`${toSummarySafeErrorCode(error)}\` (see the job logs for details)\n\n`;
  }

  // Display secrecy and integrity security metadata fields if present in the message.
  // secrecy indicates the confidentiality level of the message content.
  // integrity indicates the trustworthiness level of the message source.
  // message.data is intentionally omitted to prevent secret leakage into step summaries.
  if (message) {
    if (message.secrecy !== undefined && message.secrecy !== null) {
      summary += `**Secrecy:** \`${message.secrecy}\`\n\n`;
    }
    if (message.integrity !== undefined && message.integrity !== null) {
      summary += `**Integrity:** \`${message.integrity}\`\n\n`;
    }
  }

  summary += `</details>\n\n`;

  return summary;
}

/**
 * Write safe-output summaries to the GitHub Actions step summary
 * @param {Array<Object>} results - Array of processing results
 * @param {Array<Object>} messages - Array of original messages
 * @returns {Promise<void>}
 */
async function writeSafeOutputSummaries(results, messages) {
  if (!results || results.length === 0) {
    return;
  }

  // Log the raw .jsonl content from the safe outputs file
  const safeOutputsFile = process.env.GH_AW_SAFE_OUTPUTS;
  if (safeOutputsFile) {
    const fs = require("fs");
    if (fs.existsSync(safeOutputsFile)) {
      try {
        const content = fs.readFileSync(safeOutputsFile, "utf8");
        if (content.trim()) {
          // Use displayFileContent helper to show file with truncation and collapsible group
          // Pass a filename with .jsonl extension so it's recognized as displayable
          displayFileContent(safeOutputsFile, "safe-outputs.jsonl", 5000);
        }
      } catch (error) {
        core.debug(`Could not read raw safe-output file: ${getErrorMessage(error)}`);
      }
    }
  }

  let summaryContent = `## Safe Output Processing Summary\n\n`;
  summaryContent += `Processed ${results.length} safe-output message(s).\n\n`;
  const status = computeSafeOutputsStatus(results);
  summaryContent += `Status: **${status.status}**\n\n`;
  summaryContent += `Items succeeded: **${status.itemsSucceeded}**\n\n`;
  summaryContent += `Items failed: **${status.itemsFailed}**\n\n`;

  // Generate summary for each result
  for (const result of results) {
    // Skip only if this was explicitly delegated to a standalone step or custom safe output job.
    // `result.reason` is set (e.g. "Handled by standalone step") only when processMessages
    // decides that a different step is responsible for the message; it is NOT set when a
    // handler itself returns { success: false, skipped: true } for a handler-side condition
    // (e.g. "no issue fields available"). Handler-returned skips still appear in the summary
    // so their diagnostic signal is preserved without the job failing.
    if (result.skipped && result.reason) {
      continue;
    }

    // Get the original message
    const message = messages[result.messageIndex];

    summaryContent += generateSafeOutputSummary({
      type: result.type,
      messageIndex: result.messageIndex + 1, // Convert to 1-based
      success: result.success,
      result: result.result,
      message: message,
      error: result.error,
    });
  }

  try {
    await core.summary.addRaw(redactStepSummaryContent(summaryContent)).write();
    core.info(`📝 Safe output summaries written to step summary`);
  } catch (error) {
    core.warning(`Failed to write safe output summaries: ${getErrorMessage(error)}`);
  }
}

module.exports = {
  generateSafeOutputSummary,
  writeSafeOutputSummaries,
};
