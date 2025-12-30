// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Detects repository visibility and sets lockdown mode for GitHub MCP server.
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
async function detectRepoVisibility(github, context, core) {
  try {
    core.info("Detecting repository visibility for GitHub MCP lockdown configuration");

    const { owner, repo } = context.repo;
    core.info(`Checking visibility for repository: ${owner}/${repo}`);

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

    core.info(`Setting GitHub MCP lockdown: ${shouldLockdown}`);
    core.setOutput("lockdown", shouldLockdown.toString());
    core.setOutput("visibility", visibility);

    if (shouldLockdown) {
      core.warning("GitHub MCP lockdown mode enabled for public repository. " + "This prevents the GitHub token from accessing private repositories.");
    }
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    core.error(`Failed to detect repository visibility: ${errorMessage}`);
    // Default to lockdown mode for safety
    core.setOutput("lockdown", "true");
    core.setOutput("visibility", "unknown");
    core.warning("Failed to detect repository visibility. Defaulting to lockdown mode for security.");
  }
}

module.exports = detectRepoVisibility;
