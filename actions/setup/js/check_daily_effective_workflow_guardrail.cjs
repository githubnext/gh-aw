// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");
const os = require("os");
const path = require("path");

const { computeEffectiveTokens } = require("./effective_tokens.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { sanitizeContent } = require("./sanitize_content.cjs");

const TOKEN_USAGE_FILENAME = "token-usage.jsonl";
const TOKEN_USAGE_RELATIVE_PATH = path.join("api-proxy-logs", TOKEN_USAGE_FILENAME);
const PRIMARY_GUARDRAIL_ARTIFACT_NAMES = ["firewall-audit-logs", "agent"];
const DAILY_WORKFLOW_WINDOW_MS = 24 * 60 * 60 * 1000;
const MAX_RECENT_RUNS_IN_ISSUE = 10;
const MAX_WORKFLOW_RUN_PAGES = 10;

/**
 * @returns {Promise<import("@actions/artifact").DefaultArtifactClient>}
 */
async function getArtifactClient() {
  const { DefaultArtifactClient } = await import("@actions/artifact");
  return new DefaultArtifactClient();
}

/**
 * @param {string | undefined} raw
 * @returns {number}
 */
function parsePositiveInt(raw) {
  const trimmed = raw?.trim();
  if (!trimmed || !/^\d+$/.test(trimmed)) {
    return 0;
  }
  const parsed = Number.parseInt(trimmed, 10);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : 0;
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
 * @param {string} root
 * @returns {string}
 */
function findTokenUsageFile(root) {
  const direct = path.join(root, TOKEN_USAGE_RELATIVE_PATH);
  if (fs.existsSync(direct)) {
    return direct;
  }

  /** @type {string[]} */
  const queue = [root];
  while (queue.length > 0) {
    const current = queue.shift();
    if (!current) continue;
    /** @type {fs.Dirent[]} */
    let entries = [];
    try {
      entries = fs.readdirSync(current, { withFileTypes: true });
    } catch {
      continue;
    }
    for (const entry of entries) {
      const fullPath = path.join(current, entry.name);
      if (entry.isDirectory()) {
        queue.push(fullPath);
        continue;
      }
      if (entry.isFile() && entry.name === TOKEN_USAGE_FILENAME) {
        return fullPath;
      }
    }
  }
  return "";
}

/**
 * @param {string} filePath
 * @returns {number}
 */
function sumEffectiveTokensFromTokenUsageFile(filePath) {
  if (!filePath || !fs.existsSync(filePath)) {
    return 0;
  }

  const content = fs.readFileSync(filePath, "utf8");
  if (!content.trim()) {
    return 0;
  }

  let total = 0;
  for (const rawLine of content.split("\n")) {
    const line = rawLine.trim();
    if (!line || line[0] !== "{") {
      continue;
    }

    try {
      const parsed = JSON.parse(line);
      const explicit = Number(parsed?.effective_tokens);
      if (Number.isFinite(explicit) && explicit > 0) {
        total += Math.round(explicit);
        continue;
      }

      const computed = computeEffectiveTokens(
        String(parsed?.model || ""),
        Number(parsed?.input_tokens || 0),
        Number(parsed?.output_tokens || 0),
        Number(parsed?.cache_read_tokens || 0),
        Number(parsed?.cache_write_tokens || 0),
        Number(parsed?.reasoning_tokens || 0)
      );
      if (Number.isFinite(computed) && computed > 0) {
        total += Math.round(computed);
      }
    } catch {
      // Ignore malformed lines.
    }
  }

  return total;
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
 * @param {string} owner
 * @param {string} repo
 * @param {string} workflowName
 * @param {string} workflowID
 * @param {string} runUrl
 * @param {number} totalEffectiveTokens
 * @param {number} threshold
 * @param {Array<{id:number, html_url:string, created_at:string, conclusion:string}>} runs
 * @returns {Promise<string>}
 *
 * Requires the github-script global `github` client provided by setupGlobals().
 */
async function ensureDailyEffectiveWorkflowIssue(owner, repo, workflowName, workflowID, runUrl, totalEffectiveTokens, threshold, runs) {
  const sanitizedWorkflowName = sanitizeContent(workflowName || workflowID || "workflow", { maxLength: 100 });
  const title = `[aw] ${sanitizedWorkflowName} daily ET guardrail exceeded`;
  const searchQuery = `repo:${owner}/${repo} is:issue is:open label:agentic-workflows in:title "${title}"`;

  const search = await github.rest.search.issuesAndPullRequests({
    q: searchQuery,
    per_page: 1,
  });
  if (search.data.total_count > 0) {
    return search.data.items[0]?.html_url || "";
  }

  const runLines = runs
    .slice(0, MAX_RECENT_RUNS_IN_ISSUE)
    .map(run => `- [Run #${run.id}](${run.html_url}) — ${run.created_at} (${run.conclusion || "unknown"})`)
    .join("\n");
  const body = [
    "### Daily Workflow ET Guardrail Exceeded",
    "",
    `**Workflow:** ${workflowName || workflowID}`,
    `**Run:** ${runUrl}`,
    `**24h effective tokens:** ${totalEffectiveTokens}`,
    `**Threshold:** ${threshold}`,
    "",
    "Recent runs counted toward this total:",
    runLines || "- No completed runs with downloadable token-usage artifacts were found.",
    "",
    `<!-- gh-aw-daily-effective-workflow-guardrail: ${workflowID} -->`,
  ].join("\n");

  const created = await github.rest.issues.create({
    owner,
    repo,
    title,
    body,
    labels: ["agentic-workflows"],
  });
  return created.data.html_url || "";
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
  core.setOutput("daily_effective_workflow_issue_url", "");

  const threshold = parsePositiveInt(process.env.GH_AW_MAX_DAILY_EFFECTIVE_WORKFLOW);
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

  const { owner, repo } = context.repo;
  const currentRun = await github.rest.actions.getWorkflowRun({
    owner,
    repo,
    run_id: context.runId,
  });

  const workflowID = process.env.GH_AW_WORKFLOW_ID || "";
  const workflowName = process.env.GH_AW_WORKFLOW_NAME || workflowID || "workflow";
  const runUrl = process.env.GH_AW_RUN_URL || currentRun.data.html_url || "";
  const actorLogin = process.env.GITHUB_TRIGGERING_ACTOR || currentRun.data.triggering_actor?.login || currentRun.data.actor?.login || process.env.GITHUB_ACTOR || "";

  if (!currentRun.data.workflow_id || !actorLogin) {
    core.warning("Skipping daily workflow ET guardrail because the current workflow or actor could not be resolved.");
    return;
  }

  const cutoffMs = Date.now() - DAILY_WORKFLOW_WINDOW_MS;
  /** @type {Array<{id:number, html_url:string, created_at:string, conclusion:string}>} */
  const candidateRuns = [];
  /** @type {Array<any>} */
  let runs = [];
  let page = 1;
  while (page <= MAX_WORKFLOW_RUN_PAGES) {
    const response = await github.rest.actions.listWorkflowRuns({
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
    }
    if (runs.length < 100) {
      break;
    }
    page += 1;
  }

  const artifactClient = await getArtifactClient();
  let totalEffectiveTokens = 0;
  /** @type {Array<{id:number, html_url:string, created_at:string, conclusion:string}>} */
  const countedRuns = [];
  for (const run of candidateRuns) {
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
      });
    } catch (error) {
      core.warning(`Failed to inspect token usage for run ${run.id}: ${getErrorMessage(error)}`);
    }
  }

  core.setOutput("daily_effective_workflow_total_effective_tokens", String(totalEffectiveTokens));
  core.setOutput("daily_effective_workflow_threshold", String(threshold));

  if (totalEffectiveTokens <= threshold) {
    core.info(`Daily workflow ET guardrail not exceeded (${totalEffectiveTokens}/${threshold}).`);
    return;
  }

  core.setOutput("daily_effective_workflow_exceeded", "true");
  const issueUrl = await ensureDailyEffectiveWorkflowIssue(owner, repo, workflowName, workflowID, runUrl, totalEffectiveTokens, threshold, countedRuns);
  if (issueUrl) {
    core.setOutput("daily_effective_workflow_issue_url", issueUrl);
  }
  core.warning(`Daily workflow ET guardrail exceeded for ${workflowName}: ${totalEffectiveTokens}/${threshold}.`);
}

module.exports = {
  main,
  shouldSkipDailyEffectiveWorkflowGuardrail,
  matchesGuardrailArtifactName,
  findTokenUsageFile,
  sumEffectiveTokensFromTokenUsageFile,
};
