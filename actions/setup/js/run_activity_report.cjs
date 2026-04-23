// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage, isRateLimitError } = require("./error_helpers.cjs");
const { resolveExecutionOwnerRepo } = require("./repo_helpers.cjs");
const { sanitizeContent } = require("./sanitize_content.cjs");

const ISSUE_TITLE = "[aw] agentic status report";
const REPORT_COUNT = 1000;
const HEADING_DEMOTION_LEVELS = 2;
const DEFAULT_REPORT_OUTPUT_DIR = "./.cache/gh-aw/activity-report-logs";

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
async function runRangeReport(bin, prefixArgs, repoSlug, range, outputDir) {
  const args = [...prefixArgs, "logs", "--repo", repoSlug, "--start-date", range.startDate, "--count", String(REPORT_COUNT), "--output", outputDir, "--format", "markdown"];
  core.info(`Running: ${bin} ${args.join(" ")}`);

  try {
    const result = await exec.getExecOutput(bin, args, { ignoreReturnCode: true });
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
  }
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
async function main() {
  const cmdPrefixStr = process.env.GH_AW_CMD_PREFIX || "gh aw";
  const reportOutputDir = process.env.GH_AW_ACTIVITY_REPORT_OUTPUT_DIR || DEFAULT_REPORT_OUTPUT_DIR;
  const [bin, ...prefixArgs] = cmdPrefixStr.split(" ").filter(Boolean);
  const { owner, repo } = resolveExecutionOwnerRepo();
  const repoSlug = `${owner}/${repo}`;

  core.info(`Generating agentic workflow activity report for ${repoSlug}`);

  const sections = [];
  for (const range of REPORT_RANGES) {
    sections.push(await runRangeReport(bin, prefixArgs, repoSlug, range, reportOutputDir));
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

module.exports = { main, hasRateLimitText, runRangeReport, normalizeReportMarkdown };
