// @ts-check

const { renderLockdownTokenErrorMessage, renderPublicStrictModeErrorMessage, renderPullRequestTargetErrorMessage } = require("./validate_lockdown_requirements_templates.cjs");

/**
 * Validates that lockdown mode requirements are met at runtime.
 *
 * When lockdown mode is explicitly enabled in the workflow configuration,
 * at least one custom GitHub token must be configured (GH_AW_GITHUB_TOKEN,
 * GH_AW_GITHUB_MCP_SERVER_TOKEN, or custom github-token). Without any custom token,
 * the workflow will fail with a clear error message.
 *
 * Additionally, workflows running on public repositories must be compiled with
 * strict mode enabled (GH_AW_COMPILED_STRICT=true). This ensures that public
 * repository workflows meet the security requirements enforced by strict mode.
 *
 * Finally, the pull_request_target event is disallowed on public repositories
 * to prevent "pwn request" attacks where a fork can trigger workflows with access
 * to repository secrets.
 *
 * This validation runs at the start of the workflow to fail fast if requirements
 * are not met, providing clear guidance to the user.
 *
 * @param {any} core - GitHub Actions core library
 * @returns {void}
 */
function validateLockdownRequirements(core) {
  /**
   * @param {string} message
   * @returns {never}
   */
  function failWithError(message) {
    core.setOutput("lockdown_check_failed", "true");
    core.setFailed(message);
    throw new Error(message);
  }

  // Check if lockdown mode is explicitly enabled (set to "true" in frontmatter)
  const lockdownEnabled = process.env.GITHUB_MCP_LOCKDOWN_EXPLICIT === "true";

  if (!lockdownEnabled) {
    // Lockdown not explicitly enabled, no validation needed
    core.info("Lockdown mode not explicitly enabled, skipping validation");
  } else {
    core.info("Lockdown mode is explicitly enabled, validating requirements...");

    // Check if any custom GitHub token is configured
    // This matches the token selection logic used by the MCP gateway:
    // GH_AW_GITHUB_MCP_SERVER_TOKEN || GH_AW_GITHUB_TOKEN || custom github-token
    const hasGhAwToken = !!process.env.GH_AW_GITHUB_TOKEN;
    const hasGhAwMcpToken = !!process.env.GH_AW_GITHUB_MCP_SERVER_TOKEN;
    const hasCustomToken = !!process.env.CUSTOM_GITHUB_TOKEN;
    const hasAnyCustomToken = hasGhAwToken || hasGhAwMcpToken || hasCustomToken;

    core.info(`GH_AW_GITHUB_TOKEN configured: ${hasGhAwToken}`);
    core.info(`GH_AW_GITHUB_MCP_SERVER_TOKEN configured: ${hasGhAwMcpToken}`);
    core.info(`Custom github-token configured: ${hasCustomToken}`);

    if (!hasAnyCustomToken) {
      failWithError(renderLockdownTokenErrorMessage());
    }

    core.info("✓ Lockdown mode requirements validated: Custom GitHub token is configured");
  }

  // Enforce strict mode for public repositories.
  // Workflows compiled without strict mode must not run on public repositories,
  // as strict mode enforces important security constraints for public exposure.
  const isPublic = process.env.GITHUB_REPOSITORY_VISIBILITY === "public";
  const isStrict = process.env.GH_AW_COMPILED_STRICT === "true";

  core.info(`Repository visibility: ${process.env.GITHUB_REPOSITORY_VISIBILITY || "unknown"}`);
  core.info(`Compiled with strict mode: ${isStrict}`);

  if (isPublic && !isStrict) {
    failWithError(renderPublicStrictModeErrorMessage());
  }

  if (isPublic && isStrict) {
    core.info("✓ Strict mode requirements validated: Public repository compiled with strict mode");
  }

  // Disallow pull_request_target event in public repositories.
  // The pull_request_target event runs workflows in the context of the base repository
  // with access to secrets, even when triggered from a fork. This creates a significant
  // security risk in public repositories where anyone can open a pull request from a fork
  // and potentially exfiltrate secrets or cause unintended side effects.
  const eventName = process.env.GITHUB_EVENT_NAME;
  if (isPublic && eventName === "pull_request_target") {
    failWithError(renderPullRequestTargetErrorMessage());
  }
}

module.exports = validateLockdownRequirements;
