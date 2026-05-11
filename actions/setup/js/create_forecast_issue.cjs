// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("node:fs");

const FORECAST_REPORT_PATH = "./.cache/gh-aw/forecast/report.json";
const FORECAST_ISSUE_TITLE = "[aw] workflow forecast report";

/**
 * @param {unknown} value
 * @returns {string}
 */
function escapeCell(value) {
  return String(value ?? "").replaceAll("|", "\\|");
}

/**
 * @param {unknown} value
 * @returns {string}
 */
function formatET(value) {
  const n = Number(value ?? 0);
  if (!Number.isFinite(n) || n <= 0) {
    return "0";
  }
  return new Intl.NumberFormat("en-US", { maximumFractionDigits: 0 }).format(n);
}

/**
 * @param {Record<string, any>} report
 * @param {{owner: string, repo: string, serverUrl: string, runID?: string, generatedAtISO?: string}} options
 * @returns {string}
 */
function buildForecastIssueBody(report, options) {
  const workflows = Array.isArray(report.workflows) ? report.workflows : [];
  const rows = workflows.map(workflow => {
    const p50 = workflow?.monte_carlo?.p50_projected_effective_tokens ?? workflow?.projected_effective_tokens ?? 0;
    return [escapeCell(workflow.workflow_id), workflow.sampled_runs ?? 0, Number(p50)];
  });

  const allProjectedZero = rows.every(([, , p50]) => Number(p50) === 0);
  const zeroProjectedWithSamples = rows.filter(([, sampledRuns, p50]) => Number(sampledRuns) > 0 && Number(p50) === 0).length;
  const zeroWorkflowWord = zeroProjectedWithSamples === 1 ? "workflow" : "workflows";
  const zeroWorkflowVerb = zeroProjectedWithSamples === 1 ? "has" : "have";
  const reportTable = ["| Workflow | Sampled runs | Forecast ET (P50) |", "| --- | ---: | ---: |", ...rows.map(([workflowID, sampledRuns, p50]) => `| ${workflowID} | ${sampledRuns} | ${formatET(p50)} |`)].join("\n");

  const repoSlug = `${options.owner}/${options.repo}`;
  const period = report.period || "month";
  const runID = options.runID || "";
  const runURL = runID ? `${options.serverUrl}/${repoSlug}/actions/runs/${runID}` : "";

  return [
    "### Agentic workflow forecast report",
    "",
    `Repository: ${repoSlug}`,
    `Generated at: ${options.generatedAtISO || new Date().toISOString()}`,
    `Period: ${period}`,
    "",
    reportTable,
    "",
    ...(allProjectedZero
      ? [
          "> [!NOTE]",
          "> All projected ET values are 0 even after cache warm-up. This usually means cached run summaries do not include token usage for sampled runs.",
          "> Verify gh aw logs fetched recent runs and that run_summary.json files include token usage.",
          "",
        ]
      : []),
    ...(zeroProjectedWithSamples > 0
      ? [
          "> [!TIP]",
          `> ${zeroProjectedWithSamples} ${zeroWorkflowWord} ${zeroWorkflowVerb} sampled runs but forecast ET is 0. This usually indicates missing token usage in cached run summaries for sampled runs.`,
          "> Increase the warm-up scope with `gh aw logs --start-date -30d --count <larger value>` if this persists.",
          "",
        ]
      : []),
    ...(runURL ? [`_Forecast source run: [#${runID}](${runURL})._`] : []),
  ].join("\n");
}

/**
 * @returns {Promise<void>}
 */
async function main() {
  if (!fs.existsSync(FORECAST_REPORT_PATH)) {
    core.warning(`Forecast report JSON not found at ${FORECAST_REPORT_PATH}; skipping issue creation.`);
    return;
  }

  let reportBody = "";
  try {
    reportBody = fs.readFileSync(FORECAST_REPORT_PATH, "utf8").trim();
  } catch (error) {
    core.warning(`Failed to read forecast report JSON at ${FORECAST_REPORT_PATH}: ${error.message}`);
    return;
  }

  if (!reportBody) {
    core.warning(`Forecast report JSON is empty at ${FORECAST_REPORT_PATH}; skipping issue creation.`);
    return;
  }

  /** @type {Record<string, any>} */
  let report = {};
  try {
    report = JSON.parse(reportBody);
  } catch (error) {
    core.warning(`Failed to parse forecast report JSON at ${FORECAST_REPORT_PATH}: ${error.message}`);
    return;
  }

  if (!Array.isArray(report.workflows) || report.workflows.length === 0) {
    core.warning("Forecast report contains no workflows; skipping issue creation.");
    return;
  }

  const body = buildForecastIssueBody(report, {
    owner: context.repo.owner,
    repo: context.repo.repo,
    serverUrl: context.serverUrl,
    runID: process.env.GITHUB_RUN_ID || "",
  });

  const createdIssue = await github.rest.issues.create({
    owner: context.repo.owner,
    repo: context.repo.repo,
    title: FORECAST_ISSUE_TITLE,
    body,
    labels: ["agentic-workflows"],
  });

  core.info(`Created issue #${createdIssue.data.number}: ${createdIssue.data.html_url}`);
}

module.exports = {
  main,
  buildForecastIssueBody,
  formatET,
  escapeCell,
  FORECAST_REPORT_PATH,
  FORECAST_ISSUE_TITLE,
};
