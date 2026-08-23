// @ts-check
/// <reference types="@actions/github-script" />

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

const { isStagedMode } = require("./safe_output_helpers.cjs");
const { logStagedPreviewInfo } = require("./staged_preview.cjs");

/** @type {string} Safe output type handled by this module */
const HANDLER_TYPE = "upload_code_coverage";

/**
 * Main handler factory for upload_code_coverage.
 *
 * This handler does not touch the coverage report file's bytes: the file is staged by the
 * agent (or auto-copied by the upload_code_coverage MCP tool handler) into the
 * upload-code-coverage staging directory, uploaded as an artifact by the agent job, and later
 * downloaded and passed to actions/upload-code-coverage by the dedicated upload_code_coverage
 * job. This handler only validates the file/language/label metadata and exposes it via
 * step outputs so the downstream job can pass them to that action.
 *
 * Returns a message handler function that processes individual upload_code_coverage messages.
 * @type {HandlerFactoryFunction}
 */
async function main(config = {}) {
  const maxCount = config.max || 1;
  const isStaged = isStagedMode(config);

  core.info(`Upload code coverage configuration: max=${maxCount}`);

  // Track how many items we've processed for max limit
  let processedCount = 0;

  /**
   * Message handler function that processes a single upload_code_coverage message
   * @param {Object} message - The upload_code_coverage message to process
   * @param {Object} resolvedTemporaryIds - Map of temporary IDs to {repo, number}
   * @returns {Promise<Object>} Result with success/error status
   */
  return async function handleUploadCodeCoverage(message, resolvedTemporaryIds) {
    if (processedCount >= maxCount) {
      core.warning(`Skipping upload_code_coverage: max count of ${maxCount} reached`);
      return {
        success: false,
        error: `Max count of ${maxCount} reached`,
      };
    }

    const item = message;

    // Validate required fields
    if (!item.file || typeof item.file !== "string") {
      core.warning('Missing or invalid required field "file" in upload_code_coverage item');
      return {
        success: false,
        error: 'Missing or invalid required field "file"',
      };
    }

    if (!item.language || typeof item.language !== "string") {
      core.warning('Missing or invalid required field "language" in upload_code_coverage item');
      return {
        success: false,
        error: 'Missing or invalid required field "language"',
      };
    }

    if (!item.label || typeof item.label !== "string") {
      core.warning('Missing or invalid required field "label" in upload_code_coverage item');
      return {
        success: false,
        error: 'Missing or invalid required field "label"',
      };
    }

    const file = item.file.trim();
    const language = item.language.trim();
    const label = item.label.trim();

    processedCount++;

    core.info(`Processing upload_code_coverage: file=${file}, language=${language}, label=${label}`);

    if (isStaged) {
      logStagedPreviewInfo(`Would upload code coverage report: file=${file}, language=${language}, label=${label}`);
      return {
        success: true,
        staged: true,
        file,
        language,
        label,
      };
    }

    // Expose the metadata via step outputs for the downstream upload_code_coverage job.
    core.setOutput("upload_code_coverage_file", file);
    core.setOutput("upload_code_coverage_language", language);
    core.setOutput("upload_code_coverage_label", label);

    core.info(`✓ Recorded upload_code_coverage metadata for downstream upload: ${file}`);

    return {
      success: true,
      file,
      language,
      label,
    };
  };
}

module.exports = { main };
