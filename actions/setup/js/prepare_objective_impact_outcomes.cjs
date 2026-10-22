// @ts-check

const fs = require("node:fs");
const path = require("node:path");
const { execFileSync } = require("node:child_process");
const { getSetupTimeoutMs, MAX_SETUP_TIMEOUT_MS } = require("./child_process_timeouts.cjs");
const { ERR_API, ERR_CONFIG, ERR_PARSE, ERR_SYSTEM, ERR_VALIDATION } = require("./error_codes.cjs");

/** @param {string} filePath */
function readJSON(filePath) {
  let content;
  try {
    content = fs.readFileSync(filePath, "utf8");
  } catch (error) {
    throw new Error(`${ERR_SYSTEM}: Failed to read impact report input ${filePath}`, { cause: error });
  }
  try {
    return JSON.parse(content);
  } catch (error) {
    throw new Error(`${ERR_PARSE}: Invalid impact report JSON in ${filePath}`, { cause: error });
  }
}

/** @param {string} filePath @param {string} content */
function writeFileAtomic(filePath, content) {
  const temporary = `${filePath}.tmp`;
  try {
    fs.writeFileSync(temporary, content);
    fs.renameSync(temporary, filePath);
  } catch (error) {
    throw new Error(`${ERR_SYSTEM}: Failed to write impact report output ${filePath}`, { cause: error });
  }
}

/** @param {string} repo @param {string} runsDir @param {number[]} ids */
function collectOutcomeReports(repo, runsDir, ids) {
  const reports = new Map();
  if (ids.length === 0) return reports;
  const timeout = Math.min(MAX_SETUP_TIMEOUT_MS, getSetupTimeoutMs("outcomeGh") * ids.length);
  let raw;
  try {
    raw = execFileSync("gh", ["aw", "logs", "--stdin", "--outcomes", "--repo", repo, "--output", runsDir, "--timeout", String(Math.ceil(timeout / 60_000)), "--json"], {
      input: ids.join("\n") + "\n",
      encoding: "utf8",
      stdio: ["pipe", "pipe", "inherit"],
      timeout,
      maxBuffer: 64 * 1024 * 1024,
    });
  } catch (error) {
    throw new Error(`${ERR_API}: Failed to evaluate outcomes for selected runs`, { cause: error });
  }
  let logs;
  try {
    logs = JSON.parse(raw);
  } catch (error) {
    throw new Error(`${ERR_PARSE}: Invalid gh aw logs --outcomes JSON`, { cause: error });
  }
  if (!Array.isArray(logs?.outcomes)) throw new Error(`${ERR_VALIDATION}: Invalid gh aw logs --outcomes report`);
  for (const report of logs.outcomes) {
    if (!Number.isSafeInteger(report?.run_id) || report.run_id <= 0 || reports.has(report.run_id)) throw new Error(`${ERR_VALIDATION}: Invalid gh aw logs --outcomes report`);
    reports.set(report.run_id, report);
  }
  return reports;
}

/** @param {string} dataDir @param {string} repo */
function prepareIssueOutcomes(dataDir, repo) {
  if (!repo) throw new Error(`${ERR_CONFIG}: EXPR_GITHUB_REPOSITORY or GITHUB_REPOSITORY is required`);
  const workflowLogs = readJSON(path.join(dataDir, "workflow-logs.json"));
  if (!workflowLogs || !Array.isArray(workflowLogs.runs)) throw new Error(`${ERR_VALIDATION}: Impact report input must contain a runs array`);
  const runs = new Map();
  for (const run of workflowLogs.runs) {
    const id = Number(run?.run_id ?? run?.id ?? run?.databaseId);
    if (!Number.isSafeInteger(id) || id <= 0) throw new Error(`${ERR_VALIDATION}: Impact report contains an invalid run ID`);
    if (!runs.has(id)) runs.set(id, run);
  }
  const reports = collectOutcomeReports(repo, path.join(dataDir, "safe-output-runs"), [...runs.keys()]);
  const rows = [];
  for (const [id, run] of runs) {
    const report = reports.get(id);
    if (!report || report.eval_error || report.run_id !== id || !Array.isArray(report.items) || report.items.some(item => !item || typeof item.type !== "string" || typeof item.outcome_status !== "string")) {
      throw new Error(`${ERR_VALIDATION}: Invalid gh aw logs --outcomes report for run ${id}`);
    }
    for (const item of report.items) {
      if (item.type !== "create_issue" && item.type !== "close_issue") continue;
      rows.push({
        ...item,
        run_id: id,
        workflow_name: report.workflow || run.workflow_name || run.workflowName || "",
        workflow_aic: run.aic ?? null,
        workflow_run_created_at: run.created_at || run.createdAt || "",
        workflow_run_url: run.html_url || run.url || "",
        number: item.object_number ?? null,
        url: item.object_url || "",
        timestamp: item.created_at || "",
      });
    }
  }

  const summary = {
    total_issue_outcomes: rows.length,
    create_issue_count: rows.filter(row => row.type === "create_issue").length,
    close_issue_count: rows.filter(row => row.type === "close_issue").length,
    accepted_count: rows.filter(row => row.outcome_status === "accepted").length,
    rejected_count: rows.filter(row => row.outcome_status === "rejected").length,
    pending_count: rows.filter(row => row.outcome_status === "pending").length,
    ignored_count: rows.filter(row => row.outcome_status === "ignored").length,
    unknown_count: rows.filter(row => row.outcome_status === "unknown").length,
    errors_count: rows.filter(row => row.outcome_status === "error").length,
    lifecycle_count: rows.filter(row => ["lifecycle", "lifecycle_close"].includes(row.outcome_status)).length,
    distinct_workflows: new Set(rows.map(row => row.workflow_name).filter(Boolean)).size,
    distinct_runs_with_issue_outcomes: new Set(rows.map(row => row.run_id)).size,
  };
  writeFileAtomic(path.join(dataDir, "safe-output-issue-evaluations.jsonl"), rows.map(row => JSON.stringify(row)).join("\n") + (rows.length ? "\n" : ""));
  writeFileAtomic(path.join(dataDir, "safe-output-issue-summary.json"), JSON.stringify(summary, null, 2) + "\n");
}

if (require.main === module) {
  prepareIssueOutcomes("/tmp/gh-aw/agent/objective-impact-report", process.env.EXPR_GITHUB_REPOSITORY || process.env.GITHUB_REPOSITORY || "");
}

module.exports = { prepareIssueOutcomes };
