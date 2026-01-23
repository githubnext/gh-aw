// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Constants
 *
 * This module provides shared constants used across JavaScript actions.
 * These constants should be kept in sync with the constants in pkg/constants/constants.go
 */

/**
 * AgentOutputFilename is the filename of the agent output JSON file
 * @type {string}
 */
const AGENT_OUTPUT_FILENAME = "agent_output.json";

/**
 * Base path for temporary gh-aw files
 * @type {string}
 */
const TMP_GH_AW_PATH = "/tmp/gh-aw";

module.exports = {
  AGENT_OUTPUT_FILENAME,
  TMP_GH_AW_PATH,
};
