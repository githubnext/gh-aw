// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("node:fs");
const { getPromptPath, renderTemplateFromFile } = require("./messages_core.cjs");

const FORECAST_REPORT_PATH = "./.cache/gh-aw/forecast/report.json";
const FORECAST_ERROR_PATH = "./.cache/gh-aw/forecast/error.json";
const FORECAST_ISSUE_TITLE = "[aw] workflow forecast report";
const FORECAST_ERROR_ISSUE_TITLE = "[aw] workflow forecast report (error)";
const FORECAST_ISSUE_TEMPLATE = "forecast_issue.md";

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
 * @param {Record<string, any>|null} report
 * @param {{owner: string, repo: string, serverUrl: string, runID?: string, generatedAtISO?: string, outcome?: string, errorMessage?: string}} options
 * @returns {string}
 */
function buildForecastIssueBody(report, options) {
  const workflows = Array.isArray(report?.workflows) ? report.workflows : [];
  const rows = workflows.map(workflow => {
    const p50 = workflow?.monte_carlo?.p50_projected_effective_tokens ?? workflow?.projected_effective_tokens ?? 0;
    return [escapeCell(workflow.workflow_id), workflow.sampled_runs ?? 0, Number(p50)];
  });

  const allProjectedZero = rows.length > 0 && rows.every(([, , p50]) => Number(p50) === 0);
  const zeroProjectedWithSamples = rows.filter(([, sampledRuns, p50]) => Number(sampledRuns) > 0 && Number(p50) === 0).length;
  const zeroWorkflowWord = zeroProjectedWithSamples === 1 ? "workflow" : "workflows";
  const zeroWorkflowVerb = zeroProjectedWithSamples === 1 ? "has" : "have";
  const reportTable = rows.length > 0
    ? ["| Workflow | Sampled runs | Forecast ET (P50) |", "| --- | ---: | ---: |", ...rows.map(([workflowID, sampledRuns, p50]) => `| ${workflowID} | ${sampledRuns} | ${formatET(p50)} |`)].join("\n")
    : "_No forecast rows were produced._";

  const repoSlug = `${options.owner}/${options.repo}`;
  const period = report?.period || "month";
  const runID = options.runID || "";
  const runURL = runID ? `${options.serverUrl}/${repoSlug}/actions/runs/${runID}` : "";
  const outcome = (options.outcome || "success").toLowerCase();

  const allProjectedZeroNote = allProjectedZero
    ? ["> [!NOTE]", "> All projected ET values are 0 even after cache warm-up. This usually means cached run summaries do not include token usage for sampled runs.", "> Verify gh aw logs fetched recent runs and that run_summary.json files include token usage.", ""].join("\n")
    : "";
  const zeroProjectedTip = zeroProjectedWithSamples > 0
    ? ["> [!TIP]", `> ${zeroProjectedWithSamples} ${zeroWorkflowWord} ${zeroWorkflowVerb} sampled runs but forecast ET is 0. This usually indicates missing token usage in cached run summaries for sampled runs.`, "> Increase the warm-up scope with `gh aw logs --start-date -30d --count <larger value>` if this persists.", ""].join("\n")
    : "";
  const sourceRunLine = runURL ? `_Forecast source run: [#${runID}](${runURL})._` : "";
  const errorSection = outcome === "success"
    ? ""
    : ["> [!WARNING]", `> Forecast outcome: ${outcome}.`, `> ${options.errorMessage || "Forecast computation did not complete successfully."}`].join("\n");

  return renderTemplateFromFile(getPromptPath(FORECAST_ISSUE_TEMPLATE), {
    repository: repoSlug,
    generated_at: options.generatedAtISO || new Date().toISOString(),
    period,
    report_table: reportTable,
    all_projected_zero_note: allProjectedZeroNote,
    zero_projected_tip: zeroProjectedTip,
    error_section: errorSection,
    source_run_line: sourceRunLine,
  }).trim();
}

/**
 * @returns {Promise<void>}
 */
async function main() {
  /** @type {Record<string, any>|null} */
  let report = null;
  let outcome = "success";
  let errorMessage = "";

  if (fs.existsSync(FORECAST_REPORT_PATH)) {
    let reportBody = "";
    try {
      reportBody = fs.readFileSync(FORECAST_REPORT_PATH, "utf8").trim();
    } catch (error) {
      outcome = "error";
      errorMessage = `Failed to read forecast report JSON at ${FORECAST_REPORT_PATH}: ${error.message}`;
      core.warning(errorMessage);
    }

    if (reportBody) {
      try {
        report = JSON.parse(reportBody);
      } catch (error) {
        outcome = "error";
        errorMessage = `Failed to parse forecast report JSON at ${FORECAST_REPORT_PATH}: ${error.message}`;
        core.warning(errorMessage);
      }
    } else if (!errorMessage) {
      outcome = "error";
      errorMessage = `Forecast report JSON is empty at ${FORECAST_REPORT_PATH}.`;
      core.warning(errorMessage);
    }
  } else {
    outcome = "error";
    errorMessage = `Forecast report JSON not found at ${FORECAST_REPORT_PATH}.`;
    core.warning(errorMessage);
  }

  if (fs.existsSync(FORECAST_ERROR_PATH)) {
    try {
      const errorPayload = JSON.parse(fs.readFileSync(FORECAST_ERROR_PATH, "utf8"));
      outcome = String(errorPayload?.outcome || outcome).toLowerCase();
      if (typeof errorPayload?.message === "string" && errorPayload.message.trim() !== "") {
        errorMessage = errorPayload.message.trim();
      }
    } catch (error) {
      core.warning(`Failed to parse forecast error JSON at ${FORECAST_ERROR_PATH}: ${error.message}`);
    }
  }

  if (process.env.FORECAST_STEP_OUTCOME && outcome === "success") {
    const stepOutcome = process.env.FORECAST_STEP_OUTCOME.toLowerCase();
    if (stepOutcome !== "success") {
      outcome = stepOutcome;
      errorMessage = errorMessage || `Forecast step finished with outcome: ${stepOutcome}.`;
    }
  }

  const isErrorOutcome = outcome !== "success";

  const body = buildForecastIssueBody(report, {
    owner: context.repo.owner,
    repo: context.repo.repo,
    serverUrl: context.serverUrl,
    runID: process.env.GITHUB_RUN_ID || "",
    outcome,
    errorMessage,
  });

  const createdIssue = await github.rest.issues.create({
    owner: context.repo.owner,
    repo: context.repo.repo,
    title: isErrorOutcome ? FORECAST_ERROR_ISSUE_TITLE : FORECAST_ISSUE_TITLE,
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
  FORECAST_ERROR_PATH,
  FORECAST_ISSUE_TITLE,
  FORECAST_ERROR_ISSUE_TITLE,
  FORECAST_ISSUE_TEMPLATE,
};
