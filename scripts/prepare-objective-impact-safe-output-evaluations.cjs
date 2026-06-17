#!/usr/bin/env node

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

function writeJSON(filePath, value) {
  fs.writeFileSync(filePath, JSON.stringify(value, null, 2) + "\n");
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

/**
 * Load the objective mapping from the precomputed file.
 * Falls back to an empty mapping if the file is missing or invalid.
 * @param {string} filePath
 * @returns {{ label_to_value: Record<string, number>, multi_label_logic: string }}
 */
function loadObjectiveMapping(filePath) {
  const data = readJSON(filePath, {});
  const labelToValue = data.label_to_value || {};
  const logic = typeof data.multi_label_logic === "string" ? data.multi_label_logic : "max";
  return { label_to_value: labelToValue, multi_label_logic: logic };
}

/**
 * Compute the objective value for a set of issue labels using the mapping.
 * Mirrors the logic in pkg/github/label_objective_mapping.go ComputeObjectiveValue.
 * @param {string[]} labels
 * @param {{ label_to_value: Record<string, number>, multi_label_logic: string }} mapping
 * @returns {number}
 */
function computeObjectiveValue(labels, mapping) {
  if (!Array.isArray(labels) || labels.length === 0) return 0;
  const lv = mapping.label_to_value || {};
  const matchingValues = [];
  for (const label of labels) {
    const normalized = String(label).toLowerCase().trim();
    if (Object.prototype.hasOwnProperty.call(lv, normalized)) {
      matchingValues.push(lv[normalized]);
    }
  }
  if (matchingValues.length === 0) return 0;
  const logic = mapping.multi_label_logic || "max";
  if (logic === "sum") return matchingValues.reduce((a, b) => a + b, 0);
  return Math.max(...matchingValues);
}

/**
 * Return the subset of labels that have objective values in the mapping.
 * Mirrors the logic in pkg/github/label_objective_mapping.go GetObjectiveLabels.
 * @param {string[]} labels
 * @param {{ label_to_value: Record<string, number> }} mapping
 * @returns {string[]}
 */
function getObjectiveLabels(labels, mapping) {
  if (!Array.isArray(labels) || labels.length === 0) return [];
  const lv = mapping.label_to_value || {};
  return labels.filter(label => Object.prototype.hasOwnProperty.call(lv, String(label).toLowerCase().trim()));
}

/**
 * Parse an issue number from a GitHub issue URL.
 * @param {string} url
 * @returns {number | null}
 */
function parseIssueNumberFromURL(url) {
  const match = String(url || "").match(/\/(?:issues|pull)\/(\d+)/);
  if (!match) return null;
  const num = Number.parseInt(match[1], 10);
  return Number.isInteger(num) && num > 0 ? num : null;
}

/**
 * Fetch labels for a GitHub issue using the root-level resolver approach:
 * use the issue's own labels directly (these are the "root" labels for
 * safe-output issue outcomes).
 * @param {string} itemUrl
 * @param {string} itemRepo
 * @returns {string[]}
 */
function fetchIssueLabelsByURL(itemUrl, itemRepo) {
  const num = parseIssueNumberFromURL(itemUrl);
  if (!num || !itemRepo) return [];
  const raw = gh(["api", `repos/${itemRepo}/issues/${num}`, "--jq", ".labels[].name"]);
  if (!raw) return [];
  return raw.split("\n").map(l => l.trim()).filter(Boolean);
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

  const mapping = loadObjectiveMapping(path.join(DATA_DIR, "objective-mapping.json"));
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
      const itemRepo = item.repo || repo;
      const itemUrl = item.url || "";

      // Use the root-level resolver: fetch the issue's own labels from GitHub to compute
      // objective value. For safe-output issues, the issue itself is the root object.
      const labels = itemUrl ? fetchIssueLabelsByURL(itemUrl, itemRepo) : [];
      const objectiveValue = computeObjectiveValue(labels, mapping);
      const objectiveLabels = getObjectiveLabels(labels, mapping);

      rows.push({
        run_id: run.id,
        workflow_name: run.workflow_name,
        workflow_aic: run.aic,
        workflow_run_created_at: run.created_at,
        workflow_run_url: run.url,
        type: item.type,
        repo: itemRepo,
        number: typeof item.number === "number" ? item.number : null,
        url: itemUrl,
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
        labels,
        objective_value: objectiveValue,
        objective_labels: objectiveLabels,
      });
    }
  }

  fs.writeFileSync(OUTPUT_JSONL, rows.map(row => JSON.stringify(row)).join("\n") + (rows.length > 0 ? "\n" : ""));

  const acceptedRows = rows.filter(row => row.outcome_status === "accepted");
  const summary = {
    total_issue_outcomes: rows.length,
    create_issue_count: rows.filter(row => row.type === "create_issue").length,
    close_issue_count: rows.filter(row => row.type === "close_issue").length,
    accepted_count: acceptedRows.length,
    rejected_count: rows.filter(row => row.outcome_status === "rejected").length,
    pending_count: rows.filter(row => row.outcome_status === "pending").length,
    ignored_count: rows.filter(row => row.outcome_status === "ignored").length,
    unknown_count: rows.filter(row => row.outcome_status === "unknown").length,
    distinct_workflows: [...new Set(rows.map(row => row.workflow_name).filter(Boolean))].length,
    distinct_runs_with_issue_outcomes: [...new Set(rows.map(row => row.run_id))].length,
    total_objective_value: rows.reduce((sum, row) => sum + (row.objective_value || 0), 0),
    accepted_objective_value: acceptedRows.reduce((sum, row) => sum + (row.objective_value || 0), 0),
    mapped_count: rows.filter(row => (row.objective_value || 0) > 0).length,
    unmapped_count: rows.filter(row => (row.objective_value || 0) === 0).length,
  };
  writeJSON(OUTPUT_SUMMARY, summary);
}

main();