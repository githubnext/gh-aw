// @ts-check

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
 * This validation runs at the start of the workflow to fail fast if requirements
 * are not met, providing clear guidance to the user.
 *
 * @param {any} core - GitHub Actions core library
 * @returns {void}
 */
const { ERR_VALIDATION } = require("./error_codes.cjs");
function validateLockdownRequirements(core) {
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
      const errorMessage =
        "Lockdown mode is enabled (lockdown: true) but no custom GitHub token is configured.\\n" +
        "\\n" +
        "Please configure one of the following as a repository secret:\\n" +
        "  - GH_AW_GITHUB_TOKEN (recommended)\\n" +
        "  - GH_AW_GITHUB_MCP_SERVER_TOKEN (alternative)\\n" +
        "  - Custom github-token in your workflow frontmatter\\n" +
        "\\n" +
        "See: https://github.com/github/gh-aw/blob/main/docs/src/content/docs/reference/auth.mdx\\n" +
        "\\n" +
        "To set a token:\\n" +
        '  gh aw secrets set GH_AW_GITHUB_TOKEN --value "YOUR_FINE_GRAINED_PAT"';

      core.setOutput("lockdown_check_failed", "true");
      core.setFailed(errorMessage);
      throw new Error(errorMessage);
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
    const errorMessage =
      "This workflow is running on a public repository but was not compiled with strict mode.\\n" +
      "\\n" +
      "Public repository workflows must be compiled with strict mode enabled to meet\\n" +
      "the security requirements for public exposure.\\n" +
      "\\n" +
      "To fix this, recompile the workflow with strict mode:\\n" +
      "  gh aw compile --strict\\n" +
      "\\n" +
      "See: https://github.com/github/gh-aw/blob/main/docs/src/content/docs/reference/security.mdx";

    core.setOutput("lockdown_check_failed", "true");
    core.setFailed(errorMessage);
    throw new Error(errorMessage);
  }

  if (isPublic && isStrict) {
    core.info("✓ Strict mode requirements validated: Public repository compiled with strict mode");
  }
}

module.exports = validateLockdownRequirements;
