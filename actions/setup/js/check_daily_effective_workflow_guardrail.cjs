// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");
const os = require("os");
const path = require("path");

const { calculateDailyEffectiveWorkflowStats, findTokenUsageFile, formatEffectiveTokens, sumEffectiveTokensFromTokenUsageFile } = require("./daily_effective_workflow_helpers.cjs");
const { parsePositiveEffectiveTokenLimitNumber } = require("./effective_token_limits.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { createRateLimitAwareGithub } = require("./github_rate_limit_logger.cjs");

const PRIMARY_GUARDRAIL_ARTIFACT_NAMES = ["firewall-audit-logs", "agent"];
const DAILY_WORKFLOW_WINDOW_MS = 24 * 60 * 60 * 1000;
const MAX_WORKFLOW_RUN_PAGES = 10;
const RATE_LIMIT_RESERVE = 100;
const REQUEST_OVERHEAD_BUDGET = MAX_WORKFLOW_RUN_PAGES + 4;
const ESTIMATED_API_OPERATIONS_PER_RUN = 2;
const INTEGER_FORMATTER = new Intl.NumberFormat("en-US");

/**
 * @returns {Promise<import("@actions/artifact").DefaultArtifactClient>}
 */
async function getArtifactClient() {
  const { DefaultArtifactClient } = await import("@actions/artifact");
  return new DefaultArtifactClient();
}

/**
 * @returns {boolean}
 */
function shouldSkipDailyEffectiveWorkflowGuardrail() {
  const eventName = process.env.GITHUB_EVENT_NAME || "";
  if (eventName === "workflow_call" || eventName === "repository_dispatch") {
    return true;
  }
  return eventName === "workflow_dispatch" && (process.env.GH_AW_WORKFLOW_DISPATCH_AW_CONTEXT || "").trim() !== "";
}

/**
 * @param {string} artifactName
 * @returns {boolean}
 */
function matchesGuardrailArtifactName(artifactName) {
  if (!artifactName) {
    return false;
  }
  return PRIMARY_GUARDRAIL_ARTIFACT_NAMES.some(name => artifactName === name || artifactName.endsWith(`-${name}`));
}

/**
 * @param {import("@actions/artifact").DefaultArtifactClient} artifactClient
 * @param {number} runId
 * @param {string} token
 * @param {string} owner
 * @param {string} repo
 * @returns {Promise<number>}
 */
async function getRunEffectiveTokens(artifactClient, runId, token, owner, repo) {
  const { artifacts } = await artifactClient.listArtifacts({
    latest: true,
    findBy: {
      token,
      workflowRunId: runId,
      repositoryOwner: owner,
      repositoryName: repo,
    },
  });

  const artifact = artifacts.find(item => matchesGuardrailArtifactName(item.name));
  if (!artifact) {
    return 0;
  }

  const downloadRoot = fs.mkdtempSync(path.join(os.tmpdir(), `gh-aw-daily-guardrail-${runId}-`));
  const download = await artifactClient.downloadArtifact(artifact.id, {
    path: downloadRoot,
    findBy: {
      token,
      workflowRunId: runId,
      repositoryOwner: owner,
      repositoryName: repo,
    },
  });

  const tokenUsageFile = findTokenUsageFile(download.downloadPath || downloadRoot);
  return sumEffectiveTokensFromTokenUsageFile(tokenUsageFile);
}

/**
 * @param {number | undefined} value
 * @returns {string}
 */
function formatInteger(value) {
  const safeValue = Number.isFinite(value) ? Math.round(value || 0) : 0;
  return INTEGER_FORMATTER.format(safeValue);
}

/**
 * @param {string} raw
 * @returns {string}
 */
function escapeMarkdownCell(raw) {
  return String(raw || "")
    .replace(/\|/g, "\\|")
    .replace(/\n/g, " ");
}

/**
 * @param {number} remaining
 * @returns {number}
 */
function computeMaxInspectableRuns(remaining) {
  if (!Number.isFinite(remaining) || remaining <= 0) {
    return 0;
  }
  // Reserve headroom for the workflow-run listing overhead plus a conservative
  // estimate of two API operations per inspected run (artifact lookup and
  // artifact download). Adjust ESTIMATED_API_OPERATIONS_PER_RUN if observed
  // usage changes.
  return Math.max(0, Math.floor((remaining - RATE_LIMIT_RESERVE - REQUEST_OVERHEAD_BUDGET) / ESTIMATED_API_OPERATIONS_PER_RUN));
}

/**
 * @param {any} githubClient
 * @returns {Promise<{remaining:number,limit:number,used:number,reset:string}>}
 */
async function getCoreRateLimitSnapshot(githubClient) {
  const response = await githubClient.rest.rateLimit.get();
  const coreRate = response?.data?.resources?.core || response?.data?.rate || {};
  const reset = coreRate?.reset ? new Date(coreRate.reset * 1000).toISOString() : "";
  return {
    remaining: Number(coreRate?.remaining || 0),
    limit: Number(coreRate?.limit || 0),
    used: Number(coreRate?.used || 0),
    reset,
  };
}

/**
 * @param {string} workflowName
 * @param {string} actorLogin
 * @param {number} threshold
 * @param {Array<{id:number, html_url:string, created_at:string, conclusion:string, effective_tokens:number}>} countedRuns
 * @param {{remaining:number,limit:number,used:number,reset:string}} rateLimit
 * @param {{candidateRunsCount:number,inspectedRunsCount:number,truncatedByRateLimit:boolean}} meta
 * @returns {string}
 */
function renderDailyEffectiveWorkflowSummary(workflowName, actorLogin, threshold, countedRuns, rateLimit, meta) {
  const stats = calculateDailyEffectiveWorkflowStats(countedRuns);
  const remainingBudget = Math.max(0, threshold - stats.total);
  const usagePercent = threshold > 0 ? ((stats.total / threshold) * 100).toFixed(2) : "0.00";
  const runRows =
    countedRuns.length > 0
      ? countedRuns
          .slice()
          .sort((a, b) => Date.parse(b.created_at || "") - Date.parse(a.created_at || ""))
          .map(run => `| [#${run.id}](${run.html_url || ""}) | ${escapeMarkdownCell(run.created_at || "")} | ${escapeMarkdownCell(run.conclusion || "unknown")} | ${formatEffectiveTokens(run.effective_tokens)} |`)
          .join("\n")
      : "| _none_ | — | — | 0 |";

  const noteLines = [];
  if (meta.truncatedByRateLimit) {
    noteLines.push(`- Stopped early to preserve GitHub API rate limit headroom (${rateLimit.remaining} remaining, reserve ${RATE_LIMIT_RESERVE}).`);
  }
  if (meta.candidateRunsCount > meta.inspectedRunsCount) {
    noteLines.push(`- Considered ${meta.candidateRunsCount} prior runs in the 24h window and inspected ${meta.inspectedRunsCount}.`);
  }
  return [
    `**Workflow:** ${workflowName || "workflow"}`,
    `**Actor:** ${actorLogin || "unknown"}`,
    "",
    "| Statistic | Value |",
    "| --- | ---: |",
    `| 24h total ET | ${formatEffectiveTokens(stats.total)} |`,
    `| Threshold | ${formatEffectiveTokens(threshold)} |`,
    `| Threshold used | ${usagePercent}% |`,
    `| Remaining headroom | ${formatEffectiveTokens(remainingBudget)} |`,
    `| Runs counted | ${formatInteger(stats.count)} |`,
    `| Avg ET / run | ${formatEffectiveTokens(stats.average)} |`,
    `| Std dev ET | ${formatEffectiveTokens(stats.stddev)} |`,
    `| Min / Max ET | ${formatEffectiveTokens(stats.min)} / ${formatEffectiveTokens(stats.max)} |`,
    `| API remaining | ${formatInteger(rateLimit.remaining)} / ${formatInteger(rateLimit.limit)} |`,
    `| API used | ${formatInteger(rateLimit.used)} |`,
    `| API reset | ${rateLimit.reset || "unknown"} |`,
    "",
    "Previous runs counted in the last 24 hours:",
    "",
    "| Run | Created | Conclusion | ET |",
    "| --- | --- | --- | ---: |",
    runRows,
    ...(noteLines.length > 0 ? ["", ...noteLines] : []),
  ].join("\n");
}

/**
 * @param {string} workflowName
 * @param {string} actorLogin
 * @param {number} threshold
 * @param {Array<{id:number, html_url:string, created_at:string, conclusion:string, effective_tokens:number}>} countedRuns
 * @param {{remaining:number,limit:number,used:number,reset:string}} rateLimit
 * @param {{candidateRunsCount:number,inspectedRunsCount:number,truncatedByRateLimit:boolean}} meta
 * @returns {Promise<void>}
 */
async function appendDailyEffectiveWorkflowSummary(workflowName, actorLogin, threshold, countedRuns, rateLimit, meta) {
  const markdown = renderDailyEffectiveWorkflowSummary(workflowName, actorLogin, threshold, countedRuns, rateLimit, meta);
  core.summary.addDetails("Daily Effective Token Usage (24h)", "\n\n" + markdown);
  await core.summary.write();
}

/**
 * @returns {Promise<void>}
 *
 * Requires github-script globals (`core`, `github`, `context`) provided by setupGlobals().
 */
async function main() {
  core.setOutput("daily_effective_workflow_exceeded", "false");
  core.setOutput("daily_effective_workflow_total_effective_tokens", "");
  core.setOutput("daily_effective_workflow_threshold", "");
  const threshold = parsePositiveEffectiveTokenLimitNumber(process.env.GH_AW_MAX_DAILY_EFFECTIVE_TOKENS);
  if (threshold <= 0) {
    return;
  }
  if (shouldSkipDailyEffectiveWorkflowGuardrail()) {
    core.info("Skipping daily workflow ET guardrail for workflow_call, repository_dispatch, or workflow_dispatch with aw_context.");
    return;
  }

  const token = process.env.GH_AW_GITHUB_TOKEN || process.env.GITHUB_TOKEN || process.env.GH_TOKEN || "";
  if (!token) {
    core.warning("Skipping daily workflow ET guardrail because no GitHub token was available for artifact lookup.");
    return;
  }

  const githubClient = createRateLimitAwareGithub(github);
  const { owner, repo } = context.repo;
  const currentRun = await githubClient.rest.actions.getWorkflowRun({
    owner,
    repo,
    run_id: context.runId,
  });
  const rateLimit = await getCoreRateLimitSnapshot(githubClient);

  const workflowID = process.env.GH_AW_WORKFLOW_ID || "";
  const workflowName = process.env.GH_AW_WORKFLOW_NAME || workflowID || "workflow";
  const runUrl = process.env.GH_AW_RUN_URL || currentRun.data.html_url || "";
  const actorLogin = process.env.GITHUB_TRIGGERING_ACTOR || currentRun.data.triggering_actor?.login || currentRun.data.actor?.login || process.env.GITHUB_ACTOR || "";

  if (!currentRun.data.workflow_id || !actorLogin) {
    core.warning("Skipping daily workflow ET guardrail because the current workflow or actor could not be resolved.");
    return;
  }

  const maxInspectableRuns = computeMaxInspectableRuns(rateLimit.remaining);
  if (maxInspectableRuns <= 0) {
    core.warning(`Skipping daily workflow ET guardrail because the GitHub API rate limit is too low (${rateLimit.remaining} remaining, reserve ${RATE_LIMIT_RESERVE}).`);
    return;
  }

  const cutoffMs = Date.now() - DAILY_WORKFLOW_WINDOW_MS;
  /** @type {Array<{id:number, html_url:string, created_at:string, conclusion:string}>} */
  const candidateRuns = [];
  /** @type {Array<any>} */
  let runs = [];
  let page = 1;
  let truncatedByRateLimit = false;
  while (page <= MAX_WORKFLOW_RUN_PAGES) {
    const response = await githubClient.rest.actions.listWorkflowRuns({
      owner,
      repo,
      workflow_id: currentRun.data.workflow_id,
      actor: actorLogin,
      status: "completed",
      per_page: 100,
      page,
    });
    runs = response.data.workflow_runs || [];
    if (runs.length === 0) {
      break;
    }
    for (const run of runs) {
      if (!run || run.id === context.runId) {
        continue;
      }
      const createdAtMs = Date.parse(run.created_at || "");
      if (!Number.isFinite(createdAtMs) || createdAtMs < cutoffMs) {
        continue;
      }
      candidateRuns.push(run);
      if (candidateRuns.length >= maxInspectableRuns) {
        truncatedByRateLimit = true;
        break;
      }
    }
    if (candidateRuns.length >= maxInspectableRuns || runs.length < 100) {
      break;
    }
    page += 1;
  }

  const artifactClient = await getArtifactClient();
  let totalEffectiveTokens = 0;
  /** @type {Array<{id:number, html_url:string, created_at:string, conclusion:string, effective_tokens:number}>} */
  const countedRuns = [];
  for (const run of candidateRuns) {
    if (countedRuns.length >= maxInspectableRuns) {
      truncatedByRateLimit = true;
      break;
    }
    try {
      const runEffectiveTokens = await getRunEffectiveTokens(artifactClient, run.id, token, owner, repo);
      if (runEffectiveTokens <= 0) {
        continue;
      }
      totalEffectiveTokens += runEffectiveTokens;
      countedRuns.push({
        id: run.id,
        html_url: run.html_url || "",
        created_at: run.created_at || "",
        conclusion: run.conclusion || "",
        effective_tokens: runEffectiveTokens,
      });
    } catch (error) {
      core.warning(`Failed to inspect token usage for run ${run.id}: ${getErrorMessage(error)}`);
    }
  }

  core.setOutput("daily_effective_workflow_total_effective_tokens", String(totalEffectiveTokens));
  core.setOutput("daily_effective_workflow_threshold", String(threshold));

  /** @type {{candidateRunsCount:number,inspectedRunsCount:number,truncatedByRateLimit:boolean}} */
  const summaryMeta = {
    candidateRunsCount: candidateRuns.length,
    inspectedRunsCount: countedRuns.length,
    truncatedByRateLimit,
  };

  if (totalEffectiveTokens <= threshold) {
    await appendDailyEffectiveWorkflowSummary(workflowName, actorLogin, threshold, countedRuns, rateLimit, summaryMeta);
    core.info(`Daily workflow ET guardrail not exceeded (${totalEffectiveTokens}/${threshold}).`);
    return;
  }

  core.setOutput("daily_effective_workflow_exceeded", "true");
  await appendDailyEffectiveWorkflowSummary(workflowName, actorLogin, threshold, countedRuns, rateLimit, summaryMeta);
  core.warning(`Daily workflow ET guardrail exceeded for ${workflowName}: ${totalEffectiveTokens}/${threshold}.`);
}

module.exports = {
  main,
  shouldSkipDailyEffectiveWorkflowGuardrail,
  matchesGuardrailArtifactName,
  findTokenUsageFile,
  sumEffectiveTokensFromTokenUsageFile,
  calculateDailyEffectiveWorkflowStats,
  computeMaxInspectableRuns,
  renderDailyEffectiveWorkflowSummary,
};
