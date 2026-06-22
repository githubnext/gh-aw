#!/usr/bin/env node

const crypto = require("crypto");
const fs = require("fs");
const path = require("path");
const { execFileSync } = require("child_process");
const {
  evaluateItem,
  normalizeOutcome,
  readJSONL,
} = require("../actions/setup/js/evaluate_outcomes.cjs");

const DATA_DIR = "/tmp/gh-aw/agent/objective-impact-report";
const RUNS_DIR = path.join(DATA_DIR, "safe-output-runs");
const OUTPUT_JSONL = path.join(DATA_DIR, "safe-output-issue-evaluations.jsonl");
const OUTPUT_SUMMARY = path.join(DATA_DIR, "safe-output-issue-summary.json");

function readJSON(filePath, fallback) {
  try {
    return JSON.parse(fs.readFileSync(filePath, "utf8"));
  } catch {
    return fallback;
  }
}

// Atomically write content to a file using a temp-file-then-rename pattern.
// Using O_EXCL with a cryptographically random suffix prevents TOCTOU and
// symlink attacks (CWE-377, CWE-378). crypto.randomBytes is used instead of
// process.pid to make the temp file name unpredictable.
function writeFileAtomic(filePath, content) {
  const tmp = filePath + "." + crypto.randomBytes(8).toString("hex") + ".tmp";
  let fd;
  try {
    fd = fs.openSync(tmp, fs.constants.O_WRONLY | fs.constants.O_CREAT | fs.constants.O_EXCL, 0o666);
    fs.writeFileSync(fd, content);
    fs.closeSync(fd);
    fd = undefined;
    fs.renameSync(tmp, filePath);
  } catch (err) {
    if (typeof fd === "number") {
      try { fs.closeSync(fd); } catch {}
    }
    try { fs.unlinkSync(tmp); } catch {}
    throw err;
  }
}

function writeJSON(filePath, value) {
  writeFileAtomic(filePath, JSON.stringify(value, null, 2) + "\n");
}

function gh(args) {
  try {
    return execFileSync("gh", args, { encoding: "utf8", stdio: ["pipe", "pipe", "pipe"] }).trim();
  } catch {
    return null;
  }
}

function ensureIssueURL(item, repo) {
  if (item.url || typeof item.number !== "number" || !repo) {
    return item;
  }
  return {
    ...item,
    url: `https://github.com/${repo}/issues/${item.number}`,
  };
}

function loadRuns() {
  const workflowLogs = readJSON(path.join(DATA_DIR, "workflow-logs.json"), {});
  const runs = Array.isArray(workflowLogs.runs) ? workflowLogs.runs : [];
  return runs
    .map(run => ({
      id: Number(run.id ?? run.databaseId ?? 0),
      workflow_name: run.workflow_name || run.workflowName || "",
      aic: run.aic ?? null,
      created_at: run.created_at || run.createdAt || "",
      status: run.status || "",
      conclusion: run.conclusion || "",
      url: run.html_url || run.url || "",
    }))
    .filter(run => Number.isInteger(run.id) && run.id > 0);
}

function loadManifest(runDir) {
  const manifestPath = path.join(runDir, "safe-output-items.jsonl");
  if (!fs.existsSync(manifestPath)) return [];
  return readJSONL(manifestPath);
}

function downloadManifest(repo, runId, runDir) {
  fs.mkdirSync(runDir, { recursive: true });
  const manifestPath = path.join(runDir, "safe-output-items.jsonl");
  if (fs.existsSync(manifestPath) && fs.statSync(manifestPath).size > 0) {
    return true;
  }
  const result = gh(["run", "download", String(runId), "--repo", repo, "--name", "safe-outputs-items", "--dir", runDir]);
  return result !== null && fs.existsSync(manifestPath) && fs.statSync(manifestPath).size > 0;
}

function main() {
  const repo = process.env.EXPR_GITHUB_REPOSITORY || process.env.GITHUB_REPOSITORY || "";
  if (!repo) {
    console.error("EXPR_GITHUB_REPOSITORY or GITHUB_REPOSITORY is required");
    process.exit(1);
  }

  fs.mkdirSync(DATA_DIR, { recursive: true });
  fs.mkdirSync(RUNS_DIR, { recursive: true });

  const runs = loadRuns();
  /** @type {any[]} */
  const rows = [];

  for (const run of runs) {
    const runDir = path.join(RUNS_DIR, `run-${run.id}`);
    if (!downloadManifest(repo, run.id, runDir)) {
      continue;
    }

    const items = loadManifest(runDir)
      .filter(item => item && (item.type === "create_issue" || item.type === "close_issue"))
      .map(item => ensureIssueURL(item, item.repo || repo));

    for (const item of items) {
      const evalResult = evaluateItem(item, repo);
      const normalized = normalizeOutcome(evalResult.result, evalResult.detail);
      rows.push({
        run_id: run.id,
        workflow_name: run.workflow_name,
        workflow_aic: run.aic,
        workflow_run_created_at: run.created_at,
        workflow_run_url: run.url,
        type: item.type,
        repo: item.repo || repo,
        number: typeof item.number === "number" ? item.number : null,
        url: item.url || "",
        timestamp: item.timestamp || "",
        result: evalResult.result,
        detail: evalResult.detail,
        outcome_status: normalized.outcome_status,
        evidence_strength: normalized.evidence_strength,
        signal: normalized.signal,
        resolution_sec: evalResult.resolution_sec,
        pending_age_sec: evalResult.pending_age_sec,
        comments: evalResult.comments,
        reactions_total: evalResult.reactions_total,
        reactions_positive: evalResult.reactions_positive,
        reactions_negative: evalResult.reactions_negative,
        zero_touch: evalResult.zero_touch,
      });
    }
  }

  writeFileAtomic(OUTPUT_JSONL, rows.map(row => JSON.stringify(row)).join("\n") + (rows.length > 0 ? "\n" : ""));

  const summary = {
    total_issue_outcomes: rows.length,
    create_issue_count: rows.filter(row => row.type === "create_issue").length,
    close_issue_count: rows.filter(row => row.type === "close_issue").length,
    accepted_count: rows.filter(row => row.outcome_status === "accepted").length,
    rejected_count: rows.filter(row => row.outcome_status === "rejected").length,
    pending_count: rows.filter(row => row.outcome_status === "pending").length,
    ignored_count: rows.filter(row => row.outcome_status === "ignored").length,
    unknown_count: rows.filter(row => row.outcome_status === "unknown").length,
    distinct_workflows: [...new Set(rows.map(row => row.workflow_name).filter(Boolean))].length,
    distinct_runs_with_issue_outcomes: [...new Set(rows.map(row => row.run_id))].length,
  };
  writeJSON(OUTPUT_SUMMARY, summary);
}

main();