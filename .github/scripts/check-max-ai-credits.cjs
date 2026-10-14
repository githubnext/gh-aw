"use strict";

const fs = require("node:fs");
const path = require("node:path");

const LOG_PATHS = [
  "sandbox/firewall/logs/api-proxy-logs/event-logs.jsonl",
  "sandbox/firewall/logs/api-proxy-logs/events.jsonl",
  "sandbox/firewall/audit/api-proxy-logs/event-logs.jsonl",
  "sandbox/firewall/audit/api-proxy-logs/events.jsonl",
  "sandbox/firewall-audit-logs/api-proxy-logs/event-logs.jsonl",
  "sandbox/firewall-audit-logs/api-proxy-logs/events.jsonl",
  "sandbox/firewall/logs/api-proxy-logs/token-tracker-audit.jsonl",
  "sandbox/firewall/audit/api-proxy-logs/token-tracker-audit.jsonl",
  "sandbox/firewall/logs/audit.jsonl",
  "sandbox/firewall/audit/log.jsonl",
];
const USAGE_PATH = "sandbox/firewall/logs/api-proxy-logs/token-usage.jsonl";

function readJSON(directory, name) {
  const filename = path.join(directory, name);
  if (!fs.existsSync(filename)) throw new Error(`Missing ${name}; enforcement remains unverified.`);
  try {
    return JSON.parse(fs.readFileSync(filename, "utf8"));
  } catch {
    throw new Error(`Invalid JSON in ${name}; enforcement remains unverified.`);
  }
}

function readJSONL(directory, name) {
  const filename = path.join(directory, name);
  if (!fs.existsSync(filename)) return null;
  return fs
    .readFileSync(filename, "utf8")
    .split("\n")
    .flatMap((line, index) => {
      if (!line.trim()) return [];
      try {
        return [JSON.parse(line)];
      } catch {
        throw new Error(`Invalid firewall JSONL in ${name} at line ${index + 1}.`);
      }
    });
}

function verifyCreditLimit(directory, expected = { runId: process.env.GITHUB_RUN_ID, runAttempt: process.env.GITHUB_RUN_ATTEMPT, repository: process.env.GITHUB_REPOSITORY }) {
  if (!/^[1-9]\d*$/.test(expected.runId || "") || !/^[1-9]\d*$/.test(expected.runAttempt || "") || !expected.repository) {
    throw new Error("Expected repository, run ID and run attempt are required; enforcement remains unverified.");
  }
  const info = readJSON(directory, "aw_info.json");
  if (String(info?.run_id) !== String(expected.runId) || String(info?.run_attempt) !== String(expected.runAttempt) || info?.repository !== expected.repository) {
    throw new Error("Agent artifact repository/run/attempt does not match this verification job; enforcement remains unverified.");
  }
  const configText = fs.readFileSync(path.join(directory, "awf-config.json"), "utf8");
  let config;
  try {
    config = JSON.parse(configText);
  } catch {
    throw new Error("Invalid AWF configuration JSON; enforcement remains unverified.");
  }
  if (config?.apiProxy?.maxAiCredits !== 1) {
    throw new Error("Expected the compiled AWF per-run credit cap to be exactly 1.");
  }

  let rejectionFound = false;
  let observedFiles = 0;
  let observedRecords = 0;
  let mismatchedRejections = 0;
  for (const relativePath of LOG_PATHS) {
    const entries = readJSONL(directory, relativePath);
    if (entries === null) continue;
    observedFiles++;
    observedRecords += entries.length;
    for (const entry of entries) {
      if (entry?.event !== "ai_credits_limit_exceeded") continue;
      if (
        (entry.run_id !== undefined && String(entry.run_id) !== String(expected.runId)) ||
        (entry.run_attempt !== undefined && String(entry.run_attempt) !== String(expected.runAttempt)) ||
        (entry.repository !== undefined && entry.repository !== expected.repository)
      ) {
        throw new Error("Firewall rejection repository/run/attempt does not match this verification job; enforcement remains unverified.");
      }
      if (entry?.event === "ai_credits_limit_exceeded" && entry.max_ai_credits === 1 && typeof entry.total_ai_credits === "number" && Number.isFinite(entry.total_ai_credits) && entry.total_ai_credits >= 1 && entry.hard_cap === false) {
        rejectionFound = true;
      } else {
        mismatchedRejections++;
      }
    }
  }
  const usage = readJSONL(directory, USAGE_PATH);
  if (!rejectionFound) {
    const diagnostics = [`Inspected ${observedRecords} records in ${observedFiles} candidate firewall files.`];
    if (observedFiles === 0) diagnostics.push("Runtime evidence may be missing: no candidate firewall event/audit/tracker file was collected.");
    else diagnostics.push("Collected firewall evidence contains no qualifying structured rejection; check AWF rejection-event emission and artifact collection, not just workload duration.");
    if (mismatchedRejections) diagnostics.push(`${mismatchedRejections} credit rejection records had invalid cap, usage or hard_cap fields.`);
    const totals = (usage || []).filter(entry => entry?.event === "token_usage" && typeof entry.ai_credits_total === "number" && Number.isFinite(entry.ai_credits_total) && entry.ai_credits_total >= 0);
    if (totals.length) {
      const total = totals[totals.length - 1].ai_credits_total;
      diagnostics.push(`Last AWF token_usage total: ${total} (${total >= 1 ? "at/above" : "below"} cap 1); usage is not rejection evidence.`);
    }
    const stdioPath = path.join(directory, "agent-stdio.log");
    if (info.engine_id === "codex" && fs.existsSync(stdioPath)) {
      for (const line of fs.readFileSync(stdioPath, "utf8").split("\n")) {
        if (!line.startsWith("{")) continue;
        let entry;
        try {
          entry = JSON.parse(line);
        } catch {
          continue; // Mixed engine output is diagnostic-only, never passing evidence.
        }
        if (entry?.type !== "error" || typeof entry.message !== "string") continue;
        const match = /^unexpected status 403 Forbidden: Maximum AI credits exceeded \((\d+(?:\.\d+)?) \/ (\d+(?:\.\d+)?)\)\./.exec(entry.message);
        if (match && Number.isFinite(Number(match[1])) && Number.isFinite(Number(match[2]))) {
          diagnostics.push(`Codex terminal API error reports credit rejection (${Number(match[1])} / ${Number(match[2])}); agent text cannot substitute for a structured AWF rejection event.`);
          break;
        }
      }
    }
    throw new Error(`No AWF per-run credit rejection at cap 1 was recorded. ${diagnostics.join(" ")} Per-run enforcement remains unverified.`);
  }
  return "Passed: AWF recorded a per-run AI credit rejection at cap 1.";
}

if (require.main === module) {
  try {
    if (process.argv.length !== 3) throw new Error("Usage: node check-max-ai-credits.cjs AGENT_ARTIFACT_DIRECTORY");
    const verdict = verifyCreditLimit(process.argv[2]);
    console.log(verdict);
    if (process.env.GITHUB_STEP_SUMMARY) fs.appendFileSync(process.env.GITHUB_STEP_SUMMARY, `### Per-run AI credit guardrail\n\n${verdict}\n`);
  } catch (error) {
    console.error(error.message);
    process.exitCode = 1;
  }
}

module.exports = { LOG_PATHS, USAGE_PATH, verifyCreditLimit };
