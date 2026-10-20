// @ts-check
/// <reference types="@actions/github-script" />

const { setupGlobals } = require("./setup_globals.cjs");
const { main: checkAdmission } = require("./check_admission.cjs");

/**
 * @param {typeof core} coreModule
 * @param {any} githubModule
 * @param {typeof context} contextModule
 * @param {typeof exec} execModule
 * @param {typeof io} ioModule
 * @param {typeof getOctokit} getOctokitFn
 */
async function main(coreModule, githubModule, contextModule, execModule, ioModule, getOctokitFn) {
  setupGlobals(coreModule, githubModule, contextModule, execModule, ioModule, getOctokitFn);
  await checkAdmission(coreModule);
}

module.exports = { main };
