// @ts-check
/// <reference types="@actions/github-script" />

/**
 * setup_globals.cjs
 * Helper function to store GitHub Actions builtin objects in the global scope
 * This allows required modules to access these objects without needing to pass them as parameters
 */

/**
 * Stores GitHub Actions builtin objects (core, github, context, exec, io) in the global scope
 * This must be called before requiring any script that depends on these globals
 *
 * @param {typeof core} coreModule - The @actions/core module
 * @param {typeof github} githubModule - The @actions/github module
 * @param {typeof context} contextModule - The GitHub context object
 * @param {typeof exec} execModule - The @actions/exec module
 * @param {typeof io} ioModule - The @actions/io module
 */
function setupGlobals(coreModule, githubModule, contextModule, execModule, ioModule) {
  global.core = coreModule;
  global.github = githubModule;
  global.context = contextModule;
  global.exec = execModule;
  global.io = ioModule;
}

module.exports = { setupGlobals };
