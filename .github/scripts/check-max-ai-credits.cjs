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
  "sandbox/firewall/logs/audit.jsonl",
  "sandbox/firewall/audit/log.jsonl",
];

function verifyCreditLimit(directory) {
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
  for (const relativePath of LOG_PATHS) {
    const filename = path.join(directory, relativePath);
    if (!fs.existsSync(filename)) continue;
    const lines = fs.readFileSync(filename, "utf8").split("\n");
    for (let index = 0; index < lines.length; index++) {
      if (!lines[index].trim()) continue;
      let entry;
      try {
        entry = JSON.parse(lines[index]);
      } catch {
        throw new Error(`Invalid firewall JSONL in ${relativePath} at line ${index + 1}.`);
      }
      if (entry?.event === "ai_credits_limit_exceeded" && entry.max_ai_credits === 1 && typeof entry.total_ai_credits === "number" && Number.isFinite(entry.total_ai_credits) && entry.total_ai_credits >= 1 && entry.hard_cap === false) {
        rejectionFound = true;
      }
    }
  }
  if (!rejectionFound) {
    throw new Error("No AWF per-run credit rejection at cap 1 was recorded. The workload may have ended below the cap or runtime evidence may be missing; enforcement remains unverified.");
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

module.exports = { LOG_PATHS, verifyCreditLimit };
