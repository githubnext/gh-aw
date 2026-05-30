// @ts-check
/// <reference types="@actions/github-script" />

/**
 * setup_globals.cjs
 * Helper function to store GitHub Actions builtin objects in the global scope
 * This allows required modules to access these objects without needing to pass them as parameters
 */

const { createRateLimitAwareGithub } = require("./github_rate_limit_logger.cjs");

/**
 * Stores GitHub Actions builtin objects (core, github, context, exec, io, getOctokit) in the global scope
 * This must be called before requiring any script that depends on these globals
 *
 * The github object is wrapped with a rate-limit-aware proxy so that every
 * github.rest.*.*() call automatically logs rate-limit headers to
 * /tmp/gh-aw/github_rate_limits.jsonl for post-run observability.
 *
 * @param {typeof core} coreModule - The @actions/core module
 * @param {any} githubModule - The @actions/github module
 * @param {typeof context} contextModule - The GitHub context object
 * @param {typeof exec} execModule - The @actions/exec module
 * @param {typeof io} ioModule - The @actions/io module
 * @param {typeof getOctokit} getOctokitFn - The getOctokit function (builtin in actions/github-script@v9)
 */
function setupGlobals(coreModule, githubModule, contextModule, execModule, ioModule, getOctokitFn) {
  global.core = coreModule;
  // Inject X-GitHub-Api-Version header on every request to suppress the
  // "@octokit/request: endpoint is deprecated" warning that fires when the
  // unversioned GitHub REST API is used.
  githubModule.hook.before("request", options => {
    if (!options.headers["X-GitHub-Api-Version"]) {
      options.headers["X-GitHub-Api-Version"] = "2022-11-28";
    }
  });
  // @ts-expect-error - Assigning to global properties that are declared as const
  // Wrap the github object so every github.rest.*.*() call automatically logs
  // x-ratelimit-* headers to github_rate_limits.jsonl for observability.
  global.github = createRateLimitAwareGithub(githubModule);
  global.context = contextModule;
  // @ts-expect-error - Assigning to global properties that are declared as const
  global.exec = execModule;
  // @ts-expect-error - Assigning to global properties that are declared as const
  global.io = ioModule;
  // Wrap getOctokit so every client created via global.getOctokit(token) also
  // carries X-GitHub-Api-Version, suppressing the deprecation warning for
  // per-handler authenticated clients (cross-repo PAT operations, etc.).
  global.getOctokit = (token, options = {}) =>
    getOctokitFn(token, {
      ...options,
      headers: {
        "X-GitHub-Api-Version": "2022-11-28",
        ...(options.headers || {}),
      },
    });
}

module.exports = { setupGlobals };
