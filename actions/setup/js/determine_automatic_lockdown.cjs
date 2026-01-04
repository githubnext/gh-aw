// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Determines automatic lockdown mode for GitHub MCP server based on repository visibility.
 *
 * This function only applies when a custom GitHub MCP server token is defined
 * (GH_AW_GITHUB_MCP_SERVER_TOKEN) and for public repositories.
 *
 * For public repositories, lockdown mode should be enabled (true) to prevent
 * the GitHub token from accessing private repositories, which could leak
 * sensitive information.
 *
 * For private repositories, lockdown mode is not necessary (false) as there
 * is no risk of exposing private repository access.
 *
 * @param {any} github - GitHub API client
 * @param {any} context - GitHub context
 * @param {any} core - GitHub Actions core library
 * @returns {Promise<void>}
 */
async function determineAutomaticLockdown(github, context, core) {
  try {
    core.info("Determining automatic lockdown mode for GitHub MCP server");

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

    // Set lockdown based on visibility
    // Public repos should have lockdown enabled to prevent token from accessing private repos
    const shouldLockdown = !isPrivate;

    core.info(`Automatic lockdown mode determined: ${shouldLockdown}`);
    core.setOutput("lockdown", shouldLockdown.toString());
    core.setOutput("visibility", visibility);

    if (shouldLockdown) {
      core.info("Automatic lockdown mode enabled for public repository");
      core.warning("GitHub MCP lockdown mode enabled for public repository. " + "This prevents the GitHub token from accessing private repositories.");
    } else {
      core.info("Automatic lockdown mode disabled for private/internal repository");
    }
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    core.error(`Failed to determine automatic lockdown mode: ${errorMessage}`);
    // Default to lockdown mode for safety
    core.setOutput("lockdown", "true");
    core.setOutput("visibility", "unknown");
    core.warning("Failed to determine repository visibility. Defaulting to lockdown mode for security.");
  }
}

module.exports = determineAutomaticLockdown;
