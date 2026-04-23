// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage, isRateLimitError } = require("./error_helpers.cjs");
const { resolveExecutionOwnerRepo } = require("./repo_helpers.cjs");
const { sanitizeContent } = require("./sanitize_content.cjs");
const { spawn } = require("node:child_process");

const ISSUE_TITLE = "[aw] agentic status report";
const REPORT_COUNT = 1000;
const HEADING_DEMOTION_LEVELS = 2;
const DEFAULT_REPORT_OUTPUT_DIR = "./.cache/gh-aw/activity-report-logs";
const LOG_DOWNLOAD_TIMEOUT_MS = 20 * 60 * 1000;
const POST_DOWNLOAD_SETTLE_DELAY_MS = 10 * 1000;

/** @typedef {{ key: string, heading: string, startDate: string, optionalOnRateLimit: boolean }} ActivityRange */

/** @type {ActivityRange[]} */
const REPORT_RANGES = [
  { key: "24h", heading: "Last 24 hours", startDate: "-1d", optionalOnRateLimit: false },
  { key: "7d", heading: "Last 7 days", startDate: "-1w", optionalOnRateLimit: false },
];

/**
 * @param {string} text
 * @returns {boolean}
 */
function hasRateLimitText(text) {
  return /\bapi rate limit\b|\brate limit exceeded\b|\bsecondary rate limit\b|\b429\b/i.test(text);
}

/**
 * Run the logs command for a configured report range.
 *
 * @param {string} bin
 * @param {string[]} prefixArgs
 * @param {string} repoSlug
 * @param {ActivityRange} range
 * @param {string} outputDir
 * @returns {Promise<{ heading: string, body: string }>}
 */
async function runRangeReport(bin, prefixArgs, repoSlug, range, outputDir, options = {}) {
  const commandRunner = options.commandRunner || runCommandWithTimeout;
  const sleepFn = options.sleepFn || sleep;
  const timeoutMs = options.timeoutMs || LOG_DOWNLOAD_TIMEOUT_MS;
  const settleDelayMs = options.settleDelayMs || POST_DOWNLOAD_SETTLE_DELAY_MS;
  const args = [...prefixArgs, "logs", "--repo", repoSlug, "--start-date", range.startDate, "--count", String(REPORT_COUNT), "--output", outputDir, "--format", "markdown"];
  core.info(`Running: ${bin} ${args.join(" ")}`);

  try {
    const result = await commandRunner(bin, args, timeoutMs);
    const output = `${result.stdout || ""}\n${result.stderr || ""}`.trim();
    const rateLimited = hasRateLimitText(output);

    if (result.exitCode === 0 && result.stdout.trim()) {
      return {
        heading: range.heading,
        body: normalizeReportMarkdown(sanitizeContent(result.stdout.trim())),
      };
    }

    if (rateLimited && range.optionalOnRateLimit) {
      core.warning(`Skipping ${range.heading} report due to GitHub API rate limiting`);
      return {
        heading: range.heading,
        body: "_Skipped due to GitHub API rate limiting._",
      };
    }

    if (rateLimited) {
      return {
        heading: range.heading,
        body: "_Could not generate this section due to GitHub API rate limiting._",
      };
    }

    return {
      heading: range.heading,
      body: `_Report command failed (exit code ${result.exitCode})._\n\n\`\`\`\n${sanitizeContent(output || "No command output was captured.")}\n\`\`\``,
    };
  } catch (error) {
    const errorMessage = getErrorMessage(error);
    const rateLimited = isRateLimitError(error) || hasRateLimitText(errorMessage);

    if (rateLimited && range.optionalOnRateLimit) {
      core.warning(`Skipping ${range.heading} report due to GitHub API rate limiting`);
      return {
        heading: range.heading,
        body: "_Skipped due to GitHub API rate limiting._",
      };
    }

    if (rateLimited) {
      return {
        heading: range.heading,
        body: "_Could not generate this section due to GitHub API rate limiting._",
      };
    }

    return {
      heading: range.heading,
      body: `_Report command failed: ${sanitizeContent(errorMessage)}_`,
    };
  } finally {
    core.info(`Waiting ${Math.floor(settleDelayMs / 1000)}s for log copy operations to settle`);
    await sleepFn(settleDelayMs);
  }
}

/**
 * Execute command with timeout and process lifecycle controls.
 *
 * @param {string} command
 * @param {string[]} args
 * @param {number} timeoutMs
 * @returns {Promise<{ exitCode: number, stdout: string, stderr: string }>}
 */
function runCommandWithTimeout(command, args, timeoutMs) {
  return new Promise((resolve, reject) => {
    const child = spawn(command, args, {
      stdio: ["ignore", "pipe", "pipe"],
      shell: false,
      env: process.env,
    });

    let stdout = "";
    let stderr = "";
    let timedOut = false;
    let settled = false;
    const childPid = child.pid;
    core.info(`Started log download process with PID ${childPid || "unknown"}`);

    const killTimer = setTimeout(() => {
      timedOut = true;

      if (!childPid) {
        core.warning(`Log download exceeded ${Math.floor(timeoutMs / 1000)}s timeout and has no PID to kill`);
        return;
      }

      core.warning(`Log download exceeded ${Math.floor(timeoutMs / 1000)}s timeout; sending SIGTERM to gh aw PID ${childPid}`);
      try {
        process.kill(childPid, "SIGTERM");
      } catch (error) {
        core.warning(`Could not SIGTERM PID ${childPid}: ${getErrorMessage(error)}`);
      }

      setTimeout(() => {
        if (settled) {
          return;
        }

        core.warning(`gh aw PID ${childPid} still running; sending SIGKILL`);
        try {
          process.kill(childPid, "SIGKILL");
        } catch (error) {
          core.warning(`Could not SIGKILL PID ${childPid}: ${getErrorMessage(error)}`);
        }
      }, 5000);
    }, timeoutMs);

    child.stdout.on("data", chunk => {
      stdout += chunk.toString();
    });
    child.stderr.on("data", chunk => {
      stderr += chunk.toString();
    });
    child.on("error", error => {
      if (settled) {
        return;
      }
      settled = true;
      clearTimeout(killTimer);
      reject(error);
    });
    child.on("close", code => {
      if (settled) {
        return;
      }
      settled = true;
      clearTimeout(killTimer);
      resolve({
        exitCode: typeof code === "number" ? code : 1,
        stdout,
        stderr: timedOut ? `${stderr.trim()}\nProcess timed out after ${Math.floor(timeoutMs / 1000)}s and was terminated.`.trim() : stderr.trim(),
      });
    });
  });
}

/**
 * @param {number} ms
 * @returns {Promise<void>}
 */
function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

/**
 * Normalize report markdown for issue rendering.
 * Demotes headings so top-level report headings start at H3.
 *
 * @param {string} markdown
 * @returns {string}
 */
function normalizeReportMarkdown(markdown) {
  return markdown.replace(/^(#{1,6})\s+/gm, (_, hashes) => {
    const headingLevel = hashes.length;
    const demotedHeadingLevel = Math.min(6, headingLevel + HEADING_DEMOTION_LEVELS);
    return `${"#".repeat(demotedHeadingLevel)} `;
  });
}

/**
 * Generate an agentic workflow activity report issue.
 * @returns {Promise<void>}
 */
async function main(options = {}) {
  const cmdPrefixStr = process.env.GH_AW_CMD_PREFIX || "gh aw";
  const reportOutputDir = process.env.GH_AW_ACTIVITY_REPORT_OUTPUT_DIR || DEFAULT_REPORT_OUTPUT_DIR;
  const [bin, ...prefixArgs] = cmdPrefixStr.split(" ").filter(Boolean);
  const { owner, repo } = resolveExecutionOwnerRepo();
  const repoSlug = `${owner}/${repo}`;

  core.info(`Generating agentic workflow activity report for ${repoSlug}`);

  const sections = [];
  for (const range of REPORT_RANGES) {
    sections.push(
      await runRangeReport(bin, prefixArgs, repoSlug, range, reportOutputDir, {
        commandRunner: options.commandRunner,
        sleepFn: options.sleepFn,
        timeoutMs: options.timeoutMs,
        settleDelayMs: options.settleDelayMs,
      })
    );
  }

  const headerLines = ["### Agentic workflow activity report", "", `Repository: \`${repoSlug}\``, `Generated at: ${new Date().toISOString()}`, ""];
  const sectionLines = sections.flatMap(section => ["<details>", `<summary>${section.heading}</summary>`, "", section.body, "", "</details>", ""]);
  const body = [...headerLines, ...sectionLines].join("\n");

  const createdIssue = await github.rest.issues.create({
    owner,
    repo,
    title: ISSUE_TITLE,
    body,
    labels: ["agentic-workflows"],
  });

  core.info(`Created issue #${createdIssue.data.number}: ${createdIssue.data.html_url}`);
}

module.exports = { main, hasRateLimitText, runRangeReport, normalizeReportMarkdown, runCommandWithTimeout, sleep };
