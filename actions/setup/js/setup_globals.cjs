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
 * @param {typeof github} githubModule - The @actions/github module
 * @param {typeof context} contextModule - The GitHub context object
 * @param {typeof exec} execModule - The @actions/exec module
 * @param {typeof io} ioModule - The @actions/io module
 * @param {typeof getOctokit} getOctokitFn - The getOctokit function (builtin in actions/github-script@v9)
 */
function setupGlobals(coreModule, githubModule, contextModule, execModule, ioModule, getOctokitFn) {
  // @ts-expect-error - Assigning to global properties that are declared as const
  global.core = coreModule;
  // @ts-expect-error - Assigning to global properties that are declared as const
  // Wrap the github object so every github.rest.*.*() call automatically logs
  // x-ratelimit-* headers to github_rate_limits.jsonl for observability.
  global.github = createRateLimitAwareGithub(githubModule);
  // @ts-expect-error - Assigning to global properties that are declared as const
  global.context = contextModule;
  // @ts-expect-error - Assigning to global properties that are declared as const
  global.exec = execModule;
  // @ts-expect-error - Assigning to global properties that are declared as const
  global.io = ioModule;
  global.getOctokit = getOctokitFn;
}

module.exports = { setupGlobals };
