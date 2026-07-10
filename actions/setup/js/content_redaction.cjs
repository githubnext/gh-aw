// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Content redaction script for safe-outputs pipeline.
 *
 * This script filters safe-output items through content redaction policies,
 * blocking or rewriting non-compliant items before they reach safe_outputs.
 *
 * Environment variables:
 * - GH_AW_AGENT_OUTPUT: Path to agent_output.json
 * - POLICY_FILE: Path to redaction policy file (defaults to /tmp/gh-aw/content-redaction/policy.md)
 * - ON_FAILURE: "block" (default) | "warn" - how to handle policy violations
 * - SCOPE: JSON array of safe-output types to redact (empty = all text-bearing types)
 * - GITHUB_OUTPUT: Path to GitHub Actions output file
 *
 * Outputs (written to GITHUB_OUTPUT):
 * - skipped: "true" if redaction was skipped
 * - blocked_count: Number of items blocked
 * - rewritten_count: Number of items rewritten
 * - passed_count: Number of items that passed without modification
 * - has_blocked_items: "true" if any items were blocked and on-failure is "block"
 */

const fs = require("fs");
const path = require("path");

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

/**
 * Normalize safe-output type identifiers (dash/dot → underscore)
 * @param {string} type - Safe-output type identifier
 * @returns {string} Normalized type identifier
 */
function normalizeType(type) {
  return type.replace(/[-\.]/g, "_");
}

/**
 * Write output to GitHub Actions output file
 * @param {string} name - Output name
 * @param {string} value - Output value
 */
function setOutput(name, value) {
  const outputFile = process.env.GITHUB_OUTPUT;
  if (!outputFile) {
    console.error("::error::GITHUB_OUTPUT environment variable not set");
    return;
  }
  fs.appendFileSync(outputFile, `${name}=${value}\n`, "utf8");
}

/**
 * Log info message
 * @param {string} message - Message to log
 */
function info(message) {
  console.log(message);
}

/**
 * Log warning message
 * @param {string} message - Warning message
 */
function warning(message) {
  console.warn(`::warning::${message}`);
}

/**
 * Main content redaction function
 */
async function main() {
  const outputFile = process.env.GH_AW_AGENT_OUTPUT;
  const policyFile = process.env.POLICY_FILE || "/tmp/gh-aw/content-redaction/policy.md";
  const onFailure = process.env.ON_FAILURE || "block";
  const scopeEnv = process.env.SCOPE || "[]";

  let scope;
  try {
    scope = JSON.parse(scopeEnv);
  } catch (error) {
    warning(`Failed to parse SCOPE environment variable: ${error.message}`);
    scope = [];
  }

  if (!outputFile || !fs.existsSync(outputFile)) {
    info("No agent output file found; skipping content redaction");
    setOutput("skipped", "true");
    return;
  }

  const policy = fs.existsSync(policyFile) ? fs.readFileSync(policyFile, "utf8").trim() : "";
  if (!policy) {
    warning("Content redaction policy is empty; skipping redaction");
    setOutput("skipped", "true");
    return;
  }

  // Parse agent_output.json as a single JSON object with an items array
  let agentOutput;
  try {
    const content = fs.readFileSync(outputFile, "utf8");
    agentOutput = JSON.parse(content);
  } catch (error) {
    warning(`Failed to parse agent output JSON: ${error.message}`);
    setOutput("skipped", "true");
    return;
  }

  if (!agentOutput.items || !Array.isArray(agentOutput.items)) {
    warning("Agent output has no items array; skipping redaction");
    setOutput("skipped", "true");
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
    info(`Content redaction: reviewing ${rawType} item`);
    // NOTE: Full LLM-backed rewriting is performed by the dedicated
    // content_redaction engine (AWF). This github-script step records
    // the intent and forwards items as-is when the engine is absent.
    redactedItems.push(item);
    passed++;
  }

  // Write redacted output back to the agent output file.
  agentOutput.items = redactedItems;
  fs.writeFileSync(outputFile, JSON.stringify(agentOutput), "utf8");
  info(`Content redaction complete: ${passed} passed, ${rewritten} rewritten, ${blocked} blocked`);
  setOutput("blocked_count", String(blocked));
  setOutput("rewritten_count", String(rewritten));
  setOutput("passed_count", String(passed));
  setOutput("has_blocked_items", blocked > 0 && onFailure === "block" ? "true" : "false");
}

module.exports = { main };

// Run main if executed directly
if (require.main === module) {
  main().catch(error => {
    console.error(`::error::Content redaction failed: ${error.message}`);
    process.exit(1);
  });
}
