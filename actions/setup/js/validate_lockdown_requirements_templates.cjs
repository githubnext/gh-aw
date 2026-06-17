// @ts-check

const { renderTemplate } = require("./messages_core.cjs");

const LOCKDOWN_TOKEN_ERROR_TEMPLATE = `Lockdown mode is enabled (lockdown: true) but no custom GitHub token is configured.

Please configure one of the following as a repository secret:
  - GH_AW_GITHUB_TOKEN (recommended)
  - GH_AW_GITHUB_MCP_SERVER_TOKEN (alternative)
  - Custom github-token in your workflow frontmatter

See: {auth_docs_url}

To set a token:
  gh aw secrets set GH_AW_GITHUB_TOKEN --value "YOUR_FINE_GRAINED_PAT"`;

const PUBLIC_STRICT_MODE_ERROR_TEMPLATE = `This workflow is running on a public repository but was not compiled with strict mode.

Public repository workflows must be compiled with strict mode enabled to meet
the security requirements for public exposure.

To fix this, recompile the workflow with strict mode:
  {strict_compile_command}

See: {security_docs_url}`;

const PULL_REQUEST_TARGET_ERROR_TEMPLATE = `This workflow is triggered by the pull_request_target event on a public repository.

The pull_request_target event is not allowed on public repositories because it runs
workflows with access to repository secrets even when triggered from a fork, which
creates a significant security risk (known as a "pwn request").

To fix this, use the pull_request event instead, or migrate to a private repository.

See: {security_docs_url}`;

const TEMPLATE_CONTEXT = {
  auth_docs_url: "https://github.com/github/gh-aw/blob/main/docs/src/content/docs/reference/auth.mdx",
  security_docs_url: "https://github.com/github/gh-aw/blob/main/docs/src/content/docs/reference/security.mdx",
  strict_compile_command: "gh aw compile --strict",
};

function renderLockdownTokenErrorMessage() {
  return renderTemplate(LOCKDOWN_TOKEN_ERROR_TEMPLATE, TEMPLATE_CONTEXT);
}

function renderPublicStrictModeErrorMessage() {
  return renderTemplate(PUBLIC_STRICT_MODE_ERROR_TEMPLATE, TEMPLATE_CONTEXT);
}

function renderPullRequestTargetErrorMessage() {
  return renderTemplate(PULL_REQUEST_TARGET_ERROR_TEMPLATE, TEMPLATE_CONTEXT);
}

module.exports = {
  renderLockdownTokenErrorMessage,
  renderPublicStrictModeErrorMessage,
  renderPullRequestTargetErrorMessage,
};
