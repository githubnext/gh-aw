// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");
const zlib = require("zlib");
const { getErrorMessage } = require("./error_helpers.cjs");
const { ERR_API } = require("./error_codes.cjs");
const { sanitizeContent } = require("./sanitize_content.cjs");
const { generateFooterWithExpiration } = require("./ephemerals.cjs");
const { renderTemplateFromFile, getPromptPath } = require("./messages_core.cjs");
const { loadAgentOutput } = require("./load_agent_output.cjs");
const { isStagedMode } = require("./safe_output_helpers.cjs");
const { generateHistoryUrl } = require("./generate_history_link.cjs");
const { formatAIC } = require("./model_costs.cjs");

const NOOP_ROLLUP_ANCHOR = "<!-- gh-aw-noop-runs -->";
const NOOP_ROLLUP_SECTION_START = "<!-- gh-aw-noop-rollup:start -->";
const NOOP_ROLLUP_SECTION_END = "<!-- gh-aw-noop-rollup:end -->";
const NOOP_ROLLUP_STATE_PREFIX = "<!-- gh-aw-noop-rollup-state:";
const NOOP_ROLLUP_SCHEMA_VERSION = 1;
const NOOP_ROLLUP_WORKFLOW_LIMIT = 20;
const NOOP_ROLLUP_BUCKET_LIMIT = 50;
const NOOP_LEGACY_COMMENT_MAX_PAGES = 20;
// Allow extra room for escaping and entity expansion during sanitization before truncating for table cells.
const NOOP_TABLE_SANITIZE_MULTIPLIER = 4;
const NOOP_TRUNCATION_ELLIPSIS_LENGTH = 1;

/**
 * Search for or create the parent issue for all agentic workflow no-op runs
 * @returns {Promise<{number: number, node_id: string}>} Parent issue number and node ID
 */
async function ensureAgentRunsIssue() {
  const { owner, repo } = context.repo;
  const parentTitle = "[aw] No-Op Runs";
  const parentLabel = "agentic-workflows";

  core.info(`Searching for no-op runs issue: "${parentTitle}"`);

  // Search for existing no-op runs issue
  const searchQuery = `repo:${owner}/${repo} is:issue is:open label:${parentLabel} in:title "${parentTitle}"`;

  try {
    const { data } = await github.rest.search.issuesAndPullRequests({
      q: searchQuery,
      per_page: 1,
    });

    if (data.total_count > 0) {
      const existingIssue = data.items[0];
      core.info(`Found existing no-op runs issue #${existingIssue.number}: ${existingIssue.html_url}`);

      return {
        number: existingIssue.number,
        node_id: existingIssue.node_id,
      };
    }
  } catch (error) {
    throw new Error(`${ERR_API}: Failed to search for existing no-op runs issue: ${getErrorMessage(error)}`);
  }

  // Create no-op runs issue if it doesn't exist
  core.info(`No no-op runs issue found, creating one`);

  // Load template from file
  const templatePath = getPromptPath("noop_runs_issue.md");
  const parentBodyContent = fs.readFileSync(templatePath, "utf8");

  const parentBody = generateFooterWithExpiration({
    footerText: parentBodyContent,
    expiresHours: 24 * 30, // 30 days
  });

  const { data: newIssue } = await github.rest.issues.create({
    owner,
    repo,
    title: parentTitle,
    body: parentBody,
    labels: [parentLabel],
  });

  core.info(`✓ Created no-op runs issue #${newIssue.number}: ${newIssue.html_url}`);
  return {
    number: newIssue.number,
    node_id: newIssue.node_id,
  };
}

/**
 * Build the AIC suffix string for use in comment footers.
 * Includes both agent and threat-detection AIC when available.
 * Returns a string like " · 0.001 AIC" or "" when not available.
 * @returns {string}
 */
function buildAICSuffix() {
  const agentRaw = process.env.GH_AW_AIC;
  const detectionRaw = process.env.GH_AW_THREAT_DETECTION_AIC;
  const agentAIC = agentRaw ? Number.parseFloat(agentRaw) : NaN;
  const detectionAIC = detectionRaw ? Number.parseFloat(detectionRaw) : NaN;
  const totalAIC = (Number.isFinite(agentAIC) && agentAIC > 0 ? agentAIC : 0) + (Number.isFinite(detectionAIC) && detectionAIC > 0 ? detectionAIC : 0);
  if (totalAIC <= 0) {
    return "";
  }
  return ` · ${formatAIC(totalAIC)} AIC`;
}

/**
 * Build the ambient context suffix string for use in comment footers.
 * Returns a string like " · ⊞ 1.2K" or "" when not available.
 * @returns {string}
 */
function buildAmbientContextSuffix() {
  const raw = process.env.GH_AW_AMBIENT_CONTEXT;
  const parsed = raw ? Number.parseInt(raw, 10) : NaN;
  if (!Number.isFinite(parsed) || parsed <= 0) {
    return "";
  }
  // Format compact integer values: e.g. 1200 → "1.2K"
  const formatted = parsed >= 1000 ? `${(parsed / 1000).toFixed(1)}K` : String(parsed);
  return ` · ⊞ ${formatted}`;
}

/**
 * Build a markdown history link for use in comment footers.
 * Returns a string like " · [◷](url)" or "" when not available.
 * @returns {string}
 */
function buildHistoryLink() {
  const workflowId = process.env.GH_AW_WORKFLOW_ID || "";
  if (!workflowId) {
    return "";
  }
  const { owner, repo } = context.repo;
  const historyUrl = generateHistoryUrl({
    owner,
    repo,
    itemType: "comment",
    workflowId,
    serverUrl: context.serverUrl,
  });
  return historyUrl ? ` · [◷](${historyUrl})` : "";
}

/**
 * @returns {{schemaVersion: number, totalRuns: number, updatedAt: string, latestRunUrl: string, buckets: Array<{workflowName: string, message: string, count: number, lastSeenAt: string, lastRunUrl: string}>}}
 */
function createEmptyNoopRollupState() {
  return {
    schemaVersion: NOOP_ROLLUP_SCHEMA_VERSION,
    totalRuns: 0,
    updatedAt: "",
    latestRunUrl: "",
    buckets: [],
  };
}

/**
 * @param {string} value
 * @returns {string}
 */
function normalizeNoopText(value) {
  return String(value || "")
    .replace(/\r\n/g, "\n")
    .replace(/\s+/g, " ")
    .trim();
}

/**
 * @param {string} left
 * @param {string} right
 * @returns {number}
 */
function compareNoopTimestamps(left, right) {
  const leftTime = Date.parse(left || "");
  const rightTime = Date.parse(right || "");
  const leftValid = Number.isFinite(leftTime);
  const rightValid = Number.isFinite(rightTime);
  if (leftValid && rightValid) {
    return leftTime - rightTime;
  }
  if (leftValid) {
    return 1;
  }
  if (rightValid) {
    return -1;
  }
  return String(left || "").localeCompare(String(right || ""));
}

/**
 * @param {string} body
 * @returns {{schemaVersion: number, totalRuns: number, updatedAt: string, latestRunUrl: string, buckets: Array<{workflowName: string, message: string, count: number, lastSeenAt: string, lastRunUrl: string}>} | null}
 */
function parseNoopRollupState(body) {
  if (typeof body !== "string") {
    return null;
  }

  const match = body.match(/<!-- gh-aw-noop-rollup-state:([A-Za-z0-9+/=]+) -->/);
  if (!match) {
    return null;
  }

  try {
    const decoded = zlib.gunzipSync(Buffer.from(match[1], "base64")).toString("utf8");
    const parsed = JSON.parse(decoded);
    const buckets = Array.isArray(parsed?.buckets)
      ? parsed.buckets
          .map(bucket => ({
            workflowName: normalizeNoopText(bucket?.workflowName),
            message: normalizeNoopText(bucket?.message),
            count: Number.isFinite(bucket?.count) ? Math.max(0, Math.trunc(bucket.count)) : 0,
            lastSeenAt: typeof bucket?.lastSeenAt === "string" ? bucket.lastSeenAt : "",
            lastRunUrl: typeof bucket?.lastRunUrl === "string" ? bucket.lastRunUrl : "",
          }))
          .filter(bucket => bucket.workflowName && bucket.message && bucket.count > 0)
      : [];

    const derivedTotalRuns = buckets.reduce((sum, bucket) => sum + bucket.count, 0);

    return {
      schemaVersion: NOOP_ROLLUP_SCHEMA_VERSION,
      totalRuns: Number.isFinite(parsed?.totalRuns) ? Math.max(0, Math.trunc(parsed.totalRuns)) : derivedTotalRuns,
      updatedAt: typeof parsed?.updatedAt === "string" ? parsed.updatedAt : "",
      latestRunUrl: typeof parsed?.latestRunUrl === "string" ? parsed.latestRunUrl : "",
      buckets,
    };
  } catch {
    return null;
  }
}

/**
 * @param {{schemaVersion: number, totalRuns: number, updatedAt: string, latestRunUrl: string, buckets: Array<{workflowName: string, message: string, count: number, lastSeenAt: string, lastRunUrl: string}>}} state
 * @returns {string}
 */
function encodeNoopRollupState(state) {
  return zlib.gzipSync(Buffer.from(JSON.stringify(state), "utf8")).toString("base64");
}

/**
 * @param {string} body
 * @returns {{workflowName: string, message: string, runUrl: string} | null}
 */
function parseLegacyNoopComment(body) {
  if (typeof body !== "string" || body.includes(NOOP_ROLLUP_SECTION_START)) {
    return null;
  }

  const lines = body.trim().split(/\r?\n/);
  if (!lines[0]?.startsWith("### ")) {
    return null;
  }

  const workflowName = normalizeNoopText(lines[0].slice(4));
  const footerStart = lines.findIndex(line => line.startsWith("> Generated from ["));
  if (!workflowName || footerStart === -1) {
    return null;
  }

  const messageLines = lines.slice(1, footerStart);
  while (messageLines[0] === "") {
    messageLines.shift();
  }
  while (messageLines[messageLines.length - 1] === "") {
    messageLines.pop();
  }

  const message = normalizeNoopText(messageLines.join("\n"));
  if (!message) {
    return null;
  }

  const runUrlMatch = body.match(/\> Generated from \[[^\]]+\]\(([^)]+)\)/);
  return {
    workflowName,
    message,
    runUrl: runUrlMatch?.[1] || "",
  };
}

/**
 * @param {{schemaVersion: number, totalRuns: number, updatedAt: string, latestRunUrl: string, buckets: Array<{workflowName: string, message: string, count: number, lastSeenAt: string, lastRunUrl: string}>}} state
 * @param {{workflowName: string, message: string, runUrl: string, seenAt: string}} entry
 * @returns {void}
 */
function addNoopRollupEntry(state, entry) {
  const workflowName = normalizeNoopText(entry.workflowName);
  const message = normalizeNoopText(entry.message);
  if (!workflowName || !message) {
    return;
  }

  const existing = state.buckets.find(bucket => bucket.workflowName === workflowName && bucket.message === message);
  const seenAt = entry.seenAt || "";
  const runUrl = entry.runUrl || "";

  if (existing) {
    existing.count += 1;
    if (!existing.lastSeenAt || (seenAt && compareNoopTimestamps(seenAt, existing.lastSeenAt) >= 0)) {
      existing.lastSeenAt = seenAt;
      existing.lastRunUrl = runUrl;
    }
  } else {
    state.buckets.push({
      workflowName,
      message,
      count: 1,
      lastSeenAt: seenAt,
      lastRunUrl: runUrl,
    });
  }

  state.totalRuns += 1;
  if (!state.updatedAt || (seenAt && compareNoopTimestamps(seenAt, state.updatedAt) >= 0)) {
    state.updatedAt = seenAt;
    state.latestRunUrl = runUrl;
  }
}

/**
 * @param {string} value
 * @param {number} maxLength
 * @returns {string}
 */
function formatNoopTableCell(value, maxLength) {
  const sanitized = sanitizeContent(String(value || ""), { maxLength: maxLength * NOOP_TABLE_SANITIZE_MULTIPLIER })
    .replace(/\r?\n+/g, " ")
    .replace(/\|/g, "\\|")
    .trim();
  if (!sanitized) {
    return "—";
  }
  if (sanitized.length <= maxLength) {
    return sanitized;
  }
  return `${sanitized.slice(0, Math.max(0, maxLength - NOOP_TRUNCATION_ELLIPSIS_LENGTH)).trimEnd()}…`;
}

/**
 * @param {string} value
 * @returns {string}
 */
function formatNoopTimestamp(value) {
  if (!value) {
    return "—";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "—";
  }
  return date.toISOString().replace(".000Z", "Z").replace("T", " ");
}

/**
 * @param {string} runUrl
 * @param {string} timestamp
 * @returns {string}
 */
function formatRunReference(runUrl, timestamp) {
  const label = formatNoopTimestamp(timestamp);
  return runUrl ? `[${label}](${runUrl})` : label;
}

/**
 * @param {{schemaVersion: number, totalRuns: number, updatedAt: string, latestRunUrl: string, buckets: Array<{workflowName: string, message: string, count: number, lastSeenAt: string, lastRunUrl: string}>}} state
 * @param {string} [latestFooterLine]
 * @returns {string}
 */
function buildNoopRollupSection(state, latestFooterLine = "") {
  const buckets = [...state.buckets].sort((a, b) => {
    if (b.count !== a.count) {
      return b.count - a.count;
    }
    if (b.lastSeenAt !== a.lastSeenAt) {
      return b.lastSeenAt.localeCompare(a.lastSeenAt);
    }
    if (a.workflowName !== b.workflowName) {
      return a.workflowName.localeCompare(b.workflowName);
    }
    return a.message.localeCompare(b.message);
  });

  const workflows = [
    ...buckets
      .reduce((acc, bucket) => {
        const existing = acc.get(bucket.workflowName) || {
          workflowName: bucket.workflowName,
          count: 0,
          lastSeenAt: "",
          lastRunUrl: "",
        };
        existing.count += bucket.count;
        if (!existing.lastSeenAt || compareNoopTimestamps(bucket.lastSeenAt, existing.lastSeenAt) >= 0) {
          existing.lastSeenAt = bucket.lastSeenAt;
          existing.lastRunUrl = bucket.lastRunUrl;
        }
        acc.set(bucket.workflowName, existing);
        return acc;
      }, new Map())
      .values(),
  ].sort((a, b) => {
    if (b.count !== a.count) {
      return b.count - a.count;
    }
    if (b.lastSeenAt !== a.lastSeenAt) {
      return b.lastSeenAt.localeCompare(a.lastSeenAt);
    }
    return a.workflowName.localeCompare(b.workflowName);
  });

  const lines = [
    "## No-Op Rollup",
    "",
    "This summary is updated in place so recurring no-op runs do not add a new comment each time.",
    "",
    `Last updated: ${formatRunReference(state.latestRunUrl, state.updatedAt)}`,
    ...(latestFooterLine ? ["", latestFooterLine] : []),
    "",
    `Tracked no-op runs: **${state.totalRuns}** across **${workflows.length}** workflows and **${buckets.length}** workflow/root-cause buckets.`,
    "",
    "### By workflow",
    "",
    "| Workflow | Count | Latest |",
    "| --- | ---: | --- |",
    ...workflows.slice(0, NOOP_ROLLUP_WORKFLOW_LIMIT).map(workflow => {
      return `| ${formatNoopTableCell(workflow.workflowName, 60)} | ${workflow.count} | ${formatRunReference(workflow.lastRunUrl, workflow.lastSeenAt)} |`;
    }),
  ];

  if (workflows.length > NOOP_ROLLUP_WORKFLOW_LIMIT) {
    lines.push("", `Showing the top ${NOOP_ROLLUP_WORKFLOW_LIMIT} workflows by count.`);
  }

  lines.push(
    "",
    "<details>",
    `<summary>By workflow and root cause (${buckets.length} buckets)</summary>`,
    "",
    "| Workflow | Root cause | Count | Latest |",
    "| --- | --- | ---: | --- |",
    ...buckets.slice(0, NOOP_ROLLUP_BUCKET_LIMIT).map(bucket => {
      return `| ${formatNoopTableCell(bucket.workflowName, 50)} | ${formatNoopTableCell(bucket.message, 120)} | ${bucket.count} | ${formatRunReference(bucket.lastRunUrl, bucket.lastSeenAt)} |`;
    })
  );

  if (buckets.length > NOOP_ROLLUP_BUCKET_LIMIT) {
    lines.push("", `Showing the top ${NOOP_ROLLUP_BUCKET_LIMIT} workflow/root-cause buckets by count.`);
  }

  lines.push("");
  lines.push("</details>");
  lines.push("");
  lines.push("> Historical per-run comments remain for older runs. New runs update this rollup instead.");
  lines.push("");
  lines.push(`${NOOP_ROLLUP_STATE_PREFIX}${encodeNoopRollupState(state)} -->`);

  return lines.join("\n");
}

/**
 * @param {string} issueBody
 * @param {string} rollupSection
 * @returns {string}
 */
function upsertNoopRollupIntoIssueBody(issueBody, rollupSection) {
  const currentBody = typeof issueBody === "string" ? issueBody : "";
  const wrappedSection = `${NOOP_ROLLUP_SECTION_START}\n${rollupSection}\n${NOOP_ROLLUP_SECTION_END}`;
  const sectionRegex = /<!-- gh-aw-noop-rollup:start -->[\s\S]*?<!-- gh-aw-noop-rollup:end -->/;

  if (sectionRegex.test(currentBody)) {
    return currentBody.replace(sectionRegex, wrappedSection);
  }

  if (currentBody.includes(NOOP_ROLLUP_ANCHOR)) {
    return currentBody.replace(NOOP_ROLLUP_ANCHOR, `${NOOP_ROLLUP_ANCHOR}\n\n${wrappedSection}`);
  }

  return `${currentBody.trimEnd()}\n\n${wrappedSection}\n`;
}

/**
 * @param {any} githubClient
 * @param {string} owner
 * @param {string} repo
 * @param {number} issueNumber
 * @returns {Promise<Array<{workflowName: string, message: string, runUrl: string, seenAt: string}>>}
 */
async function loadLegacyNoopEntries(githubClient, owner, repo, issueNumber) {
  const entries = [];
  let page = 1;
  const perPage = 100;

  while (page <= NOOP_LEGACY_COMMENT_MAX_PAGES) {
    const { data } = await githubClient.rest.issues.listComments({
      owner,
      repo,
      issue_number: issueNumber,
      per_page: perPage,
      page,
    });

    if (!Array.isArray(data) || data.length === 0) {
      break;
    }

    for (const comment of data) {
      const parsed = parseLegacyNoopComment(comment.body);
      if (!parsed) {
        continue;
      }

      entries.push({
        workflowName: parsed.workflowName,
        message: parsed.message,
        runUrl: parsed.runUrl,
        seenAt: comment.updated_at || comment.created_at || "",
      });
    }

    if (data.length < perPage) {
      break;
    }
    page += 1;
  }

  if (page > NOOP_LEGACY_COMMENT_MAX_PAGES) {
    core.warning(`Stopped legacy no-op comment scan after ${NOOP_LEGACY_COMMENT_MAX_PAGES} pages`);
  }

  return entries;
}

/**
 * Process no-op safe outputs and optionally post to the no-op runs issue.
 * This merged step replaces the separate "Process no-op messages" + "Handle No-Op Message"
 * steps, eliminating the cross-step output dependency on GH_AW_NOOP_MESSAGE.
 *
 * Behaviour:
 * 1. Load noop items directly from the agent output artifact.
 * 2. In staged mode: write a summary preview and exit without posting.
 * 3. Otherwise: write a summary, set the `noop_message` step output, then post to the
 *    "[aw] No-Op Runs" tracking issue when the agent produced only noop outputs.
 */
async function main() {
  try {
    // --- Load and filter noop items from agent output ---
    const result = loadAgentOutput();
    if (!result.success) {
      core.info("Could not load agent output, skipping");
      return;
    }

    const maxCount = parseInt(process.env.GH_AW_NOOP_MAX || "0", 10);
    const allNoopItems = (result.items || []).filter(/** @param {any} item */ item => item.type === "noop");
    const noopItems = maxCount > 0 ? allNoopItems.slice(0, maxCount) : allNoopItems;

    if (noopItems.length === 0) {
      core.info("No noop items found in agent output");
      return;
    }

    core.info(`Found ${noopItems.length} noop item(s)`);
    const noopMessage = noopItems[0].message;

    // --- Staged mode: preview only, do not post ---
    if (isStagedMode()) {
      let summaryContent = "## 🎭 Staged Mode: No-Op Messages Preview\n\n";
      summaryContent += "The following messages would be logged if staged mode was disabled:\n\n";
      for (let i = 0; i < noopItems.length; i++) {
        const item = noopItems[i];
        summaryContent += `### Message ${i + 1}\n`;
        summaryContent += `${item.message}\n\n`;
        summaryContent += "---\n\n";
      }
      await core.summary.addRaw(summaryContent).write();
      core.info("📝 No-op message preview written to step summary");
      return;
    }

    // --- Write step summary ---
    let summaryContent = "\n\n## No-Op Messages\n\n";
    summaryContent += "The following messages were logged for transparency:\n\n";
    for (let i = 0; i < noopItems.length; i++) {
      const item = noopItems[i];
      core.info(`No-op message ${i + 1}: ${item.message}`);
      summaryContent += `- ${item.message}\n`;
    }
    await core.summary.addRaw(summaryContent).write();

    // Export for downstream steps/jobs
    core.setOutput("noop_message", noopMessage);
    core.info(`Successfully processed ${noopItems.length} noop message(s)`);

    // --- Post to no-op runs issue ---
    const workflowName = process.env.GH_AW_WORKFLOW_NAME || "unknown";
    const runUrl = process.env.GH_AW_RUN_URL || "";
    const agentConclusion = process.env.GH_AW_AGENT_CONCLUSION || "";
    const reportAsIssue = process.env.GH_AW_NOOP_REPORT_AS_ISSUE !== "false"; // Default to true

    core.info(`Workflow name: ${workflowName}`);
    core.info(`Run URL: ${runUrl}`);
    core.info(`Agent conclusion: ${agentConclusion}`);
    core.info(`Report as issue: ${reportAsIssue}`);

    if (!reportAsIssue) {
      core.info("report-as-issue is disabled (set to false), skipping no-op message posting to issue");
      return;
    }

    // Only post to "agent runs" issue if:
    // 1. The agent succeeded (agentConclusion === "success"), OR
    // 2. The agent failed but produced only noop outputs, which indicates a transient AI model
    //    error after the meaningful work (noop) was already captured. Skipped/cancelled runs
    //    and other non-success/non-failure conclusions are always skipped.
    if (agentConclusion !== "success" && agentConclusion !== "failure") {
      core.info(`Agent did not succeed (conclusion: ${agentConclusion}), skipping no-op message posting`);
      return;
    }

    // Skip posting when there are non-noop outputs (agent did real work)
    const nonNoopItems = result.items.filter(/** @param {any} item */ ({ type }) => type !== "noop");
    if (nonNoopItems.length > 0) {
      core.info(`Found ${nonNoopItems.length} non-noop output(s), skipping no-op message posting`);
      return;
    }

    if (agentConclusion === "failure") {
      core.info("Agent failed but produced only noop outputs (transient AI model error after noop was captured) - posting noop message");
    } else {
      core.info("Agent succeeded with only noop outputs - posting to no-op runs issue");
    }

    const { owner, repo } = context.repo;

    // Ensure no-op runs issue exists
    let noopRunsIssue;
    try {
      noopRunsIssue = await ensureAgentRunsIssue();
    } catch (error) {
      core.warning(`Could not create no-op runs issue: ${getErrorMessage(error)}`);
      // Don't fail the workflow if we can't create the issue
      return;
    }

    try {
      const { data: issue } = await github.rest.issues.get({
        owner,
        repo,
        issue_number: noopRunsIssue.number,
      });

      let state = parseNoopRollupState(issue.body) || createEmptyNoopRollupState();
      if (state.totalRuns === 0 && state.buckets.length === 0) {
        core.info(`No existing no-op rollup state found on issue #${noopRunsIssue.number}; seeding from historical comments`);
        const legacyEntries = await loadLegacyNoopEntries(github, owner, repo, noopRunsIssue.number);
        for (const entry of legacyEntries) {
          addNoopRollupEntry(state, entry);
        }
      }

      addNoopRollupEntry(state, {
        workflowName,
        message: noopMessage,
        runUrl,
        seenAt: new Date().toISOString(),
      });

      const footerTemplatePath = getPromptPath("noop_comment.md");
      const footerPreview = renderTemplateFromFile(footerTemplatePath, {
        workflow_name: workflowName,
        message: noopMessage,
        run_url: runUrl,
        aic_suffix: buildAICSuffix(),
        ambient_context_suffix: buildAmbientContextSuffix(),
        history_link: buildHistoryLink(),
      });
      const latestFooterLine = footerPreview
        .split("\n")
        .map(line => line.trim())
        .find(line => line.startsWith("> Generated from"));
      core.info(`Updating no-op rollup with latest entry: ${footerPreview.split("\n")[0]}`);

      const updatedBody = upsertNoopRollupIntoIssueBody(issue.body || "", buildNoopRollupSection(state, sanitizeContent(latestFooterLine || "", { maxLength: 1000 })));

      await github.rest.issues.update({
        owner,
        repo,
        issue_number: noopRunsIssue.number,
        body: updatedBody,
      });

      core.info(`✓ Updated no-op rollup on issue #${noopRunsIssue.number}`);
    } catch (error) {
      core.warning(`Failed to update no-op rollup issue: ${getErrorMessage(error)}`);
      // Don't fail the workflow
    }
  } catch (error) {
    core.warning(`Error in handle_noop_message: ${getErrorMessage(error)}`);
    // Don't fail the workflow
  }
}

module.exports = { main, ensureAgentRunsIssue, buildNoopRollupSection, parseNoopRollupState, upsertNoopRollupIntoIssueBody };
