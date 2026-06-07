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
function formatAIC(value) {
  const n = Number(value ?? 0);
  if (!Number.isFinite(n) || n <= 0) {
    return "0";
  }
  return new Intl.NumberFormat("en-US", { maximumFractionDigits: 0 }).format(n);
}

/**
 * @param {Record<string, any>} workflow
 * @param {{owner: string, repo: string, serverUrl: string}} options
 * @returns {string}
 */
function renderWorkflowLink(workflow, options) {
  const label = escapeCell(workflow?.workflow_id ?? "");
  const workflowPath = typeof workflow?.workflow_path === "string" ? workflow.workflow_path.trim() : "";
  if (!workflowPath) {
    return label;
  }
  const repoSlug = `${options.owner}/${options.repo}`;
  return `[${label}](${options.serverUrl}/${repoSlug}/actions/workflows/${encodeURIComponent(workflowPath)})`;
}

/**
 * @param {Record<string, any>} workflow
 * @returns {number}
 */
function monthlyCost(workflow) {
  return Number(workflow?.monthly_monte_carlo?.p50_projected_aic ?? workflow?.monthly_projected_aic ?? 0);
}

/**
 * @param {Record<string, any>} workflow
 * @returns {string}
 */
function formatEngineList(workflow) {
  const engines = Array.isArray(workflow?.engines) ? workflow.engines.filter(Boolean) : [];
  if (engines.length === 0) return "-";
  return escapeCell(engines.join(", "));
}

/**
 * @param {Record<string, any>|null} report
 * @param {{owner: string, repo: string, serverUrl: string, runID?: string, generatedAtISO?: string, outcome?: string, errorMessage?: string}} options
 * @returns {string}
 */
function buildForecastIssueBody(report, options) {
  const workflows = Array.isArray(report?.workflows) ? [...report.workflows] : [];
  workflows.sort((a, b) => monthlyCost(b) - monthlyCost(a));

  // Build the summary table with per-run P50/P95 and weekly/monthly projected totals.
  const tableRows = workflows.map(workflow => {
    const p50PerRun = workflow?.p50_aic_per_run ?? 0;
    const p95PerRun = workflow?.p95_aic_per_run ?? 0;
    const weeklyP50 = workflow?.weekly_monte_carlo?.p50_projected_aic ?? workflow?.weekly_projected_aic ?? 0;
    const monthlyP50 = workflow?.monthly_monte_carlo?.p50_projected_aic ?? workflow?.monthly_projected_aic ?? 0;
    return [renderWorkflowLink(workflow, options), formatEngineList(workflow), workflow.sampled_runs ?? 0, Number(p50PerRun), Number(p95PerRun), Number(weeklyP50), Number(monthlyP50)];
  });

  // Legacy fallback: derive weekly/monthly from the configured-period P50 when new fields are absent.
  const hasNewFields = workflows.some(w => w?.p50_aic_per_run != null || w?.weekly_projected_aic != null);
  const legacyRows = hasNewFields
    ? null
    : workflows.map(workflow => {
        const p50 = workflow?.monte_carlo?.p50_projected_aic ?? workflow?.projected_aic ?? workflow?.monte_carlo?.p50_projected_effective_tokens ?? workflow?.projected_effective_tokens ?? 0;
        return [escapeCell(workflow.workflow_id), workflow.sampled_runs ?? 0, Number(p50)];
      });

  const allWeeklyZero = tableRows.length > 0 && tableRows.every(([, , , , , weekly]) => Number(weekly) === 0);
  const allMonthlyZero = tableRows.length > 0 && tableRows.every(([, , , , , , monthly]) => Number(monthly) === 0);
  const allProjectedZero = legacyRows ? legacyRows.length > 0 && legacyRows.every(([, , p50]) => Number(p50) === 0) : allWeeklyZero && allMonthlyZero;

  let reportTable;
  if (legacyRows) {
    reportTable =
      legacyRows.length > 0
        ? ["| Workflow | Sampled runs | Forecast AIC (P50) |", "| --- | ---: | ---: |", ...legacyRows.map(([workflowID, sampledRuns, p50]) => `| ${workflowID} | ${sampledRuns} | ${formatAIC(p50)} |`)].join("\n")
        : "_No forecast rows were produced._";
  } else {
    if (tableRows.length === 0) {
      reportTable = "_No forecast rows were produced._";
    } else {
      const totalWeekly = tableRows.reduce((s, [, , , , , w]) => s + Number(w), 0);
      const totalMonthly = tableRows.reduce((s, [, , , , , , m]) => s + Number(m), 0);
      const dataRows = tableRows.map(
        ([workflowID, engines, sampledRuns, p50Run, p95Run, weekly, monthly]) => `| ${workflowID} | ${engines} | ${sampledRuns} | ${formatAIC(p50Run)} | ${formatAIC(p95Run)} | ${formatAIC(weekly)} | ${formatAIC(monthly)} |`
      );
      if (tableRows.length > 1) {
        dataRows.push(`| **TOTAL** | | | | | **${formatAIC(totalWeekly)}** | **${formatAIC(totalMonthly)}** |`);
      }
      reportTable = ["| Workflow | Engines | Runs | P50/Run | P95/Run | Weekly (P50) | Monthly (P50) |", "| --- | --- | ---: | ---: | ---: | ---: | ---: |", ...dataRows].join("\n");
    }
  }

  const repoSlug = `${options.owner}/${options.repo}`;
  const period = report?.period || "month";
  const runID = options.runID || "";
  const runURL = runID ? `${options.serverUrl}/${repoSlug}/actions/runs/${runID}` : "";
  const outcome = (options.outcome || "success").toLowerCase();

  const allProjectedZeroNote = allProjectedZero
    ? [
        "> [!NOTE]",
        "> All projected AIC values are 0 even after cache warm-up. This usually means cached run summaries do not include token usage for sampled runs.",
        "> Verify gh aw logs fetched recent runs and that run_summary.json files include token usage.",
        "",
      ].join("\n")
    : "";
  const sourceRunLine = runURL ? `_Forecast source run: [#${runID}](${runURL})._` : "";
  const errorSection = outcome === "success" ? "" : ["> [!WARNING]", `> Forecast outcome: ${outcome}.`, `> ${options.errorMessage || "Forecast computation did not complete successfully."}`].join("\n");

  return renderTemplateFromFile(getPromptPath(FORECAST_ISSUE_TEMPLATE), {
    repository: repoSlug,
    generated_at: options.generatedAtISO || new Date().toISOString(),
    period,
    report_table: reportTable,
    all_projected_zero_note: allProjectedZeroNote,
    run_samples_section: "",
    error_section: errorSection,
    source_run_line: sourceRunLine,
  }).trim();
}

/**
 * @param {Record<string, any>|null} report
 * @param {{owner: string, repo: string, serverUrl: string, generatedAtISO?: string}} options
 * @returns {string}
 */
function buildForecastStepSummary(report, options) {
  const workflows = Array.isArray(report?.workflows) ? [...report.workflows] : [];
  workflows.sort((a, b) => monthlyCost(b) - monthlyCost(a));
  const samplesSection = buildRunSamplesSection(workflows, options);
  if (!samplesSection) {
    return "";
  }

  return ["### Workflow run samples", "", samplesSection.trim(), ""].join("\n");
}

/**
 * Builds a collapsed <details> block listing every sampled run used in the forecast.
 * Returns an empty string when no workflow has run samples.
 * @param {Array<Record<string, any>>} workflows
 * @param {{owner: string, repo: string, serverUrl: string}} options
 * @returns {string}
 */
function buildRunSamplesSection(workflows, options) {
  const hasAny = workflows.some(w => Array.isArray(w?.run_samples) && w.run_samples.length > 0);
  if (!hasAny) return "";

  const lines = ["<details>", "<summary>Sampled runs used in computation</summary>", "", "| Workflow | Run ID | Date | AIC |", "| --- | ---: | --- | ---: |"];
  for (const wf of workflows) {
    const samples = Array.isArray(wf?.run_samples) ? wf.run_samples : [];
    const workflowLabel = renderWorkflowLink(wf, options);
    for (const s of samples) {
      const runID = s?.run_id ?? "";
      const date = s?.date ?? "";
      const aic = formatAIC(s?.aic ?? 0);
      const runURL = typeof s?.run_url === "string" && s.run_url !== "" ? `[#${runID}](${s.run_url})` : `#${runID}`;
      lines.push(`| ${workflowLabel} | ${runURL} | ${date} | ${aic} |`);
    }
  }
  lines.push("", "</details>", "");
  return lines.join("\n");
}

/**
 * @returns {Promise<void>}
 */
async function main() {
  /** @type {Record<string, any>|null} */
  let report = null;
  let outcome = "success";
  let errorMessage = "";
  const stepOutcome = String(process.env.FORECAST_STEP_OUTCOME || "").toLowerCase();
  const forecastStepFailed = stepOutcome !== "" && stepOutcome !== "success";

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
      if (forecastStepFailed) {
        outcome = stepOutcome;
        core.info(`${errorMessage} Forecast step outcome was ${stepOutcome}.`);
      } else {
        core.warning(errorMessage);
      }
    }
  } else {
    outcome = "error";
    errorMessage = `Forecast report JSON not found at ${FORECAST_REPORT_PATH}.`;
    if (forecastStepFailed) {
      outcome = stepOutcome;
      core.info(`${errorMessage} Forecast step outcome was ${stepOutcome}.`);
    } else {
      core.warning(errorMessage);
    }
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

  if (stepOutcome && outcome === "success") {
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
  const summary = buildForecastStepSummary(report, {
    owner: context.repo.owner,
    repo: context.repo.repo,
    serverUrl: context.serverUrl,
  });
  if (summary) {
    await core.summary.addRaw(summary).write();
  }

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
  buildForecastStepSummary,
  buildRunSamplesSection,
  formatAIC,
  escapeCell,
  FORECAST_REPORT_PATH,
  FORECAST_ERROR_PATH,
  FORECAST_ISSUE_TITLE,
  FORECAST_ERROR_ISSUE_TITLE,
  FORECAST_ISSUE_TEMPLATE,
};
