// @ts-check
/// <reference types="@actions/github-script" />

/**
 * @param {any} error
 * @returns {string}
 */
function getErrorMessage(error) {
  return error instanceof Error ? error.message : String(error);
}

/**
 * @param {any} error
 * @returns {boolean}
 */
function isInstallationRateLimitError(error) {
  const message = getErrorMessage(error).toLowerCase();
  const status = error?.status ?? error?.response?.status;

  if (message.includes("api rate limit exceeded for installation")) {
    return true;
  }

  if ((status === 403 || status === 429) && message.includes("rate limit")) {
    return true;
  }

  return false;
}

/**
 * @param {any} error
 * @returns {number}
 */
function getRetryDelayMs(error) {
  const retryAfterHeader = error?.response?.headers?.["retry-after"];
  const retryAfterSeconds = Number.parseInt(String(retryAfterHeader || ""), 10);
  if (!Number.isNaN(retryAfterSeconds) && retryAfterSeconds > 0) {
    return retryAfterSeconds * 1000;
  }

  const resetHeader = error?.response?.headers?.["x-ratelimit-reset"];
  const resetEpochSeconds = Number.parseInt(String(resetHeader || ""), 10);
  if (!Number.isNaN(resetEpochSeconds) && resetEpochSeconds > 0) {
    const waitMs = resetEpochSeconds * 1000 - Date.now();
    if (waitMs > 0) {
      return waitMs;
    }
  }

  return 0;
}

/**
 * @param {number} ms
 * @returns {Promise<void>}
 */
function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

/**
 * @param {any} github
 * @param {string} owner
 * @param {string} repo
 * @param {any} core
 * @returns {Promise<any>}
 */
async function getRepositoryWithRetry(github, owner, repo, core) {
  const maxAttempts = 3;
  let delayMs = 2000;

  for (let attempt = 1; attempt <= maxAttempts; attempt++) {
    try {
      const { data: repository } = await github.rest.repos.get({
        owner,
        repo,
      });
      return repository;
    } catch (error) {
      if (!isInstallationRateLimitError(error) || attempt === maxAttempts) {
        throw error;
      }

      const headerDelayMs = getRetryDelayMs(error);
      const waitMs = headerDelayMs > 0 ? headerDelayMs : delayMs;
      core.warning(`GitHub App installation rate limit hit while determining guard policy (attempt ${attempt}/${maxAttempts}). Retrying in ${Math.ceil(waitMs / 1000)}s.`);
      await sleep(waitMs);

      if (headerDelayMs <= 0) {
        delayMs = Math.min(delayMs * 2, 8000);
      }
    }
  }

  throw new Error("failed to fetch repository");
}

/**
 * Determines automatic guard policy for GitHub MCP server based on repository visibility.
 *
 * This step always sets `min_integrity` and `repos` outputs so that the GitHub MCP
 * `guard-policies` block is never populated with empty values:
 *
 * - Public repositories: defaults to `min_integrity=approved`, `repos=all`
 * - Private/internal repositories: defaults to `min_integrity=none`, `repos=all`
 *
 * Whether a field is "already configured" is determined by the environment variables
 * GH_AW_GITHUB_MIN_INTEGRITY and GH_AW_GITHUB_REPOS, which are set at compile time
 * from the workflow's tools.github guard policy configuration. Pre-configured values
 * are never overridden.
 *
 * Note: This step is NOT generated when both repos and min-integrity are explicitly
 * configured in the workflow.
 *
 * @param {any} github - GitHub API client
 * @param {any} context - GitHub context
 * @param {any} core - GitHub Actions core library
 * @returns {Promise<void>}
 */
async function determineAutomaticLockdown(github, context, core) {
  try {
    core.info("Determining automatic guard policy for GitHub MCP server");

    const { owner, repo } = context.repo;
    core.info(`Checking repository: ${owner}/${repo}`);

    // Fetch repository information
    const repository = await getRepositoryWithRetry(github, owner, repo, core);

    const isPrivate = repository.private;
    const visibility = repository.visibility || (isPrivate ? "private" : "public");

    core.info(`Repository visibility: ${visibility}`);
    core.info(`Repository is private: ${isPrivate}`);

    core.setOutput("visibility", visibility);

    // Check whether guard policy fields are already configured at compile time
    const configuredMinIntegrity = process.env.GH_AW_GITHUB_MIN_INTEGRITY || "";
    const configuredRepos = process.env.GH_AW_GITHUB_REPOS || "";

    core.info(`Configured min-integrity: ${configuredMinIntegrity || "(not set)"}`);
    core.info(`Configured repos: ${configuredRepos || "(not set)"}`);

    // Private/internal repos default to min_integrity=none; public repos to approved.
    // Either way, always emit outputs so guard-policies values are never empty.
    const defaultMinIntegrity = isPrivate ? "none" : "approved";
    const defaultRepos = "all";

    // Set min_integrity if not already configured
    const resolvedMinIntegrity = configuredMinIntegrity || defaultMinIntegrity;
    if (!configuredMinIntegrity) {
      core.info(`min-integrity not configured — automatically setting to '${defaultMinIntegrity}' for ${visibility} repository`);
    } else {
      core.info(`min-integrity already configured as '${configuredMinIntegrity}' — not overriding`);
    }
    core.setOutput("min_integrity", resolvedMinIntegrity);

    // Set repos if not already configured
    const resolvedRepos = configuredRepos || defaultRepos;
    if (!configuredRepos) {
      core.info(`repos not configured — automatically setting to '${defaultRepos}' for ${visibility} repository`);
    } else {
      core.info(`repos already configured as '${configuredRepos}' — not overriding`);
    }
    core.setOutput("repos", resolvedRepos);

    if (isPrivate) {
      core.info("Automatic guard policy determination complete for private/internal repository");
    } else {
      core.info("Automatic guard policy determination complete for public repository");
      core.warning("GitHub MCP guard policy automatically applied for public repository. " + "min-integrity='approved' and repos='all' ensure only approved-integrity content is accessible.");
    }

    // Write resolved guard policy values to the step summary
    const autoLabel = isPrivate ? "automatic (private repo)" : "automatic (public repo)";
    const minIntegritySource = configuredMinIntegrity ? "workflow config" : autoLabel;
    const reposSource = configuredRepos ? "workflow config" : autoLabel;

    /**
     * Escapes a value for safe embedding in a markdown table cell.
     * Replaces HTML-special characters and pipe characters that would break the table.
     * @param {string} value
     * @returns {string}
     */
    const escapeCell = value => value.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/\|/g, "\\|").replace(/\n/g, " ");

    const tableRows = [
      "| Field | Value | Source |",
      "|-------|-------|--------|",
      `| min-integrity | ${escapeCell(resolvedMinIntegrity)} | ${escapeCell(minIntegritySource)} |`,
      `| repos | ${escapeCell(resolvedRepos)} | ${escapeCell(reposSource)} |`,
    ].join("\n");
    const details = `<details>\n<summary>GitHub MCP Guard Policy</summary>\n\n${tableRows}\n\n</details>\n`;
    await core.summary.addRaw(details).write();
  } catch (error) {
    const errorMessage = getErrorMessage(error);
    core.error(`Failed to determine automatic guard policy: ${errorMessage}`);
    // Default to safe guard policy for public repos on error
    core.setOutput("min_integrity", "approved");
    core.setOutput("repos", "all");
    core.setOutput("visibility", "unknown");
    core.warning("Failed to determine repository visibility. Defaulting to guard policy min-integrity='approved', repos='all' for security.");
  }
}

module.exports = determineAutomaticLockdown;
