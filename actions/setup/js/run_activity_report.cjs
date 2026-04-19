// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage, isRateLimitError } = require("./error_helpers.cjs");
const { resolveExecutionOwnerRepo } = require("./repo_helpers.cjs");
const { sanitizeContent } = require("./sanitize_content.cjs");

const ISSUE_TITLE = "[aw] agentic status report";

/** @typedef {{ key: string, heading: string, startDate: string, optionalOnRateLimit: boolean }} ActivityRange */

/** @type {ActivityRange[]} */
const REPORT_RANGES = [
  { key: "24h", heading: "Last 24 hours", startDate: "-1d", optionalOnRateLimit: false },
  { key: "7d", heading: "Last 7 days", startDate: "-1w", optionalOnRateLimit: false },
  { key: "30d", heading: "Last 30 days", startDate: "-1mo", optionalOnRateLimit: true },
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
 * @returns {Promise<{ heading: string, body: string }>}
 */
async function runRangeReport(bin, prefixArgs, repoSlug, range) {
  const args = [...prefixArgs, "logs", "--repo", repoSlug, "--start-date", range.startDate, "--format", "markdown"];
  core.info(`Running: ${bin} ${args.join(" ")}`);

  try {
    const result = await exec.getExecOutput(bin, args, { ignoreReturnCode: true });
    const output = `${result.stdout || ""}\n${result.stderr || ""}`.trim();
    const rateLimited = hasRateLimitText(output);

    if (result.exitCode === 0 && result.stdout.trim()) {
      return {
        heading: range.heading,
        body: sanitizeContent(result.stdout.trim()),
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
 * Generate an agentic workflow activity report issue.
 * @returns {Promise<void>}
 */
async function main() {
  const cmdPrefixStr = process.env.GH_AW_CMD_PREFIX || "gh aw";
  const [bin, ...prefixArgs] = cmdPrefixStr.split(" ").filter(Boolean);
  const { owner, repo } = resolveExecutionOwnerRepo();
  const repoSlug = `${owner}/${repo}`;

  core.info(`Generating agentic workflow activity report for ${repoSlug}`);

  const sections = [];
  for (const range of REPORT_RANGES) {
    sections.push(await runRangeReport(bin, prefixArgs, repoSlug, range));
  }

  const headerLines = ["## Agentic workflow activity report", "", `Repository: \`${repoSlug}\``, `Generated at: ${new Date().toISOString()}`, ""];
  const sectionLines = sections.flatMap(section => ["---", "", `## ${section.heading}`, "", section.body, ""]);
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

module.exports = { main, hasRateLimitText, runRangeReport };
