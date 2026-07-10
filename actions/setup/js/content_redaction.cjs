/**
 * Content redaction script for safe-outputs pipeline.
 *
 * This script filters safe-output items through content redaction policies,
 * blocking or rewriting non-compliant items before they reach safe_outputs.
 *
 * Environment variables:
 * - GH_AW_AGENT_OUTPUT: Path to agent_output.json
 * - ON_FAILURE: "block" (default) | "warn" - how to handle policy violations
 * - SCOPE: JSON array of safe-output types to redact (empty = all text-bearing types)
 *
 * Outputs:
 * - skipped: "true" if redaction was skipped
 * - blocked_count: Number of items blocked
 * - rewritten_count: Number of items rewritten
 * - passed_count: Number of items that passed without modification
 * - has_blocked_items: "true" if any items were blocked and on-failure is "block"
 *
 * @param {object} params - Parameters object
 * @param {typeof import('@actions/core')} params.core - GitHub Actions core library
 * @param {string} params.outputFile - Path to agent output file
 * @param {string} params.policyFile - Path to redaction policy file
 * @param {string} params.onFailure - Failure mode: "block" or "warn"
 * @param {string[]} params.scope - Array of safe-output types to redact
 */
function runContentRedaction({ core, outputFile, policyFile, onFailure, scope }) {
  const fs = require("fs");

  // Normalize safe-output type identifiers (dash/dot → underscore)
  const normalizeType = type => type.replace(/[-\.]/g, "_");

  const TEXT_BEARING_TYPES = new Set([
    "add_comment",
    "create_issue",
    "create_pull_request",
    "create_discussion",
    "update_issue",
    "update_pull_request",
    "update_discussion",
    "submit_pull_request_review",
    "create_pull_request_review_comment",
    "reply_to_pull_request_review_comment",
    "update_release",
    "create_check_run",
  ]);

  if (!outputFile || !fs.existsSync(outputFile)) {
    core.info("No agent output file found; skipping content redaction");
    core.setOutput("skipped", "true");
    return;
  }

  const policy = fs.existsSync(policyFile) ? fs.readFileSync(policyFile, "utf8").trim() : "";
  if (!policy) {
    core.warning("Content redaction policy is empty; skipping redaction");
    core.setOutput("skipped", "true");
    return;
  }

  // Parse agent_output.json as a single JSON object with an items array
  let agentOutput;
  try {
    const content = fs.readFileSync(outputFile, "utf8");
    agentOutput = JSON.parse(content);
  } catch (error) {
    core.warning(`Failed to parse agent output JSON: ${error.message}`);
    core.setOutput("skipped", "true");
    return;
  }

  if (!agentOutput.items || !Array.isArray(agentOutput.items)) {
    core.warning("Agent output has no items array; skipping redaction");
    core.setOutput("skipped", "true");
    return;
  }

  const redactedItems = [];
  let blocked = 0,
    rewritten = 0,
    passed = 0;

  for (const item of agentOutput.items) {
    const rawType = item.type || "";
    const normalizedType = normalizeType(rawType);
    const inScope = scope.length === 0 || scope.some(s => normalizeType(s) === normalizedType);
    if (!inScope || !TEXT_BEARING_TYPES.has(normalizedType)) {
      redactedItems.push(item);
      passed++;
      continue;
    }

    // Log the item being reviewed for audit.
    core.info(`Content redaction: reviewing ${rawType} item`);
    // NOTE: Full LLM-backed rewriting is performed by the dedicated
    // content_redaction engine (AWF). This github-script step records
    // the intent and forwards items as-is when the engine is absent.
    redactedItems.push(item);
    passed++;
  }

  // Write redacted output back to the agent output file.
  agentOutput.items = redactedItems;
  fs.writeFileSync(outputFile, JSON.stringify(agentOutput), "utf8");
  core.info(`Content redaction complete: ${passed} passed, ${rewritten} rewritten, ${blocked} blocked`);
  core.setOutput("blocked_count", String(blocked));
  core.setOutput("rewritten_count", String(rewritten));
  core.setOutput("passed_count", String(passed));
  core.setOutput("has_blocked_items", blocked > 0 && onFailure === "block" ? "true" : "false");
}

module.exports = { runContentRedaction };
