// @ts-check
/// <reference types="@actions/github-script" />

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
 * Note: This step is NOT generated when tools.github.app is configured. GitHub App tokens
 * are already scoped to specific repositories, so automatic guard policy detection is
 * unnecessary. It is also NOT generated when both repos and min-integrity are explicitly
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
    const { data: repository } = await github.rest.repos.get({
      owner,
      repo,
    });

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
      core.setOutput("min_integrity", defaultMinIntegrity);
    } else {
      core.info(`min-integrity already configured as '${configuredMinIntegrity}' — not overriding`);
      core.setOutput("min_integrity", configuredMinIntegrity);
    }

    // Set repos if not already configured
    const resolvedRepos = configuredRepos || defaultRepos;
    if (!configuredRepos) {
      core.info(`repos not configured — automatically setting to '${defaultRepos}' for ${visibility} repository`);
      core.setOutput("repos", defaultRepos);
    } else {
      core.info(`repos already configured as '${configuredRepos}' — not overriding`);
      core.setOutput("repos", configuredRepos);
    }

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
    const errorMessage = error instanceof Error ? error.message : String(error);
    core.error(`Failed to determine automatic guard policy: ${errorMessage}`);
    // Default to safe guard policy for public repos on error
    core.setOutput("min_integrity", "approved");
    core.setOutput("repos", "all");
    core.setOutput("visibility", "unknown");
    core.warning("Failed to determine repository visibility. Defaulting to guard policy min-integrity='approved', repos='all' for security.");
  }
}

module.exports = determineAutomaticLockdown;
