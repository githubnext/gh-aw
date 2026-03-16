// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Determines automatic guard policy for GitHub MCP server based on repository visibility.
 *
 * For public repositories, this step automatically configures guard policies on the
 * GitHub MCP server to restrict repository access and enforce content integrity:
 *   - If `min_integrity` is not already configured, it is automatically set to `approved`.
 *   - If `repos` is not already configured, it is automatically set to `all`.
 *
 * Whether a field is "already configured" is determined by the environment variables
 * GH_AW_GITHUB_MIN_INTEGRITY and GH_AW_GITHUB_REPOS, which are set at compile time
 * from the workflow's tools.github guard policy configuration.
 *
 * For private/internal repositories, no guard policy is automatically applied.
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

    if (isPrivate) {
      core.info("Private/internal repository — no automatic guard policy applied");
      return;
    }

    // Public repository: check whether guard policy fields are already configured
    const configuredMinIntegrity = process.env.GH_AW_GITHUB_MIN_INTEGRITY || "";
    const configuredRepos = process.env.GH_AW_GITHUB_REPOS || "";

    core.info(`Configured min-integrity: ${configuredMinIntegrity || "(not set)"}`);
    core.info(`Configured repos: ${configuredRepos || "(not set)"}`);

    // Set min_integrity to "approved" if not already configured
    const resolvedMinIntegrity = configuredMinIntegrity || "approved";
    if (!configuredMinIntegrity) {
      core.info("min-integrity not configured — automatically setting to 'approved' for public repository");
      core.setOutput("min_integrity", "approved");
    } else {
      core.info(`min-integrity already configured as '${configuredMinIntegrity}' — not overriding`);
    }

    // Set repos to "all" if not already configured
    const resolvedRepos = configuredRepos || "all";
    if (!configuredRepos) {
      core.info("repos not configured — automatically setting to 'all' for public repository");
      core.setOutput("repos", "all");
    } else {
      core.info(`repos already configured as '${configuredRepos}' — not overriding`);
    }

    core.info("Automatic guard policy determination complete for public repository");
    core.warning("GitHub MCP guard policy automatically applied for public repository. " + "min-integrity='approved' and repos='all' ensure only approved-integrity content is accessible.");

    // Write resolved guard policy values to the step summary
    await core.summary
      .addHeading("GitHub MCP Guard Policy", 3)
      .addTable([
        [
          { data: "Field", header: true },
          { data: "Value", header: true },
          { data: "Source", header: true },
        ],
        ["min-integrity", resolvedMinIntegrity, configuredMinIntegrity ? "workflow config" : "automatic (public repo)"],
        ["repos", resolvedRepos, configuredRepos ? "workflow config" : "automatic (public repo)"],
      ])
      .write();
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
