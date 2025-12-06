/**
 * @file select_model.cjs
 * @description Validates and selects a compatible model from a list of requested models.
 * Supports wildcard patterns (e.g., "sonnet*", "gpt-*-mini").
 * 
 * Inputs:
 * - requestedModels: Array of model strings (may include wildcards)
 * - availableModels: Array of available model strings
 * 
 * Outputs:
 * - selectedModel: The first compatible model found
 * - matchedPattern: The pattern that matched (may be different from selectedModel if wildcard was used)
 * 
 * Exits with error if no compatible models are found.
 */

const core = require("@actions/core");

/**
 * Converts a wildcard pattern to a regular expression
 * @param {string} pattern - Pattern with * wildcards
 * @returns {RegExp} Regular expression for matching
 */
function patternToRegex(pattern) {
  // Escape special regex characters except *
  const escaped = pattern.replace(/[.+?^${}()|[\]\\]/g, "\\$&");
  // Replace * with .*
  const regex = escaped.replace(/\*/g, ".*");
  // Match entire string (^ and $)
  return new RegExp(`^${regex}$`, "i");
}

/**
 * Checks if a model matches a pattern (with wildcard support)
 * @param {string} model - Model name to check
 * @param {string} pattern - Pattern (may include wildcards)
 * @returns {boolean} True if model matches pattern
 */
function matchesPattern(model, pattern) {
  // If no wildcards, do exact match (case-insensitive)
  if (!pattern.includes("*")) {
    return model.toLowerCase() === pattern.toLowerCase();
  }

  // Use regex for wildcard matching
  const regex = patternToRegex(pattern);
  return regex.test(model);
}

/**
 * Selects the first compatible model from the requested list
 * @param {string[]} requestedModels - Array of requested models (may include wildcards)
 * @param {string[]} availableModels - Array of available models
 * @returns {{selectedModel: string, matchedPattern: string} | null} Selected model and pattern, or null if none found
 */
function selectModel(requestedModels, availableModels) {
  core.info(`Requested models: ${JSON.stringify(requestedModels)}`);
  core.info(`Available models: ${JSON.stringify(availableModels)}`);

  // Try each requested model/pattern in order
  for (const pattern of requestedModels) {
    core.info(`Checking pattern: ${pattern}`);

    // Find first available model that matches this pattern
    for (const model of availableModels) {
      if (matchesPattern(model, pattern)) {
        core.info(`✅ Found match: ${model} (pattern: ${pattern})`);
        return { selectedModel: model, matchedPattern: pattern };
      }
    }

    core.info(`❌ No match found for pattern: ${pattern}`);
  }

  return null;
}

async function run() {
  try {
    // Get inputs
    const requestedModelsInput = core.getInput("requested_models", {
      required: true,
    });
    const availableModelsInput = core.getInput("available_models", {
      required: true,
    });

    // Parse JSON inputs
    let requestedModels;
    let availableModels;

    try {
      requestedModels = JSON.parse(requestedModelsInput);
    } catch (error) {
      core.setFailed(
        `Failed to parse requested_models as JSON: ${error.message}`,
      );
      return;
    }

    try {
      availableModels = JSON.parse(availableModelsInput);
    } catch (error) {
      core.setFailed(
        `Failed to parse available_models as JSON: ${error.message}`,
      );
      return;
    }

    // Validate inputs
    if (!Array.isArray(requestedModels) || requestedModels.length === 0) {
      core.setFailed(
        "requested_models must be a non-empty array of strings",
      );
      return;
    }

    if (!Array.isArray(availableModels) || availableModels.length === 0) {
      core.setFailed(
        "available_models must be a non-empty array of strings",
      );
      return;
    }

    // Select model
    const result = selectModel(requestedModels, availableModels);

    if (!result) {
      // No compatible model found - exit with error
      core.setFailed(
        `No compatible model found.\n` +
          `Requested: ${requestedModels.join(", ")}\n` +
          `Available: ${availableModels.join(", ")}\n` +
          `\n` +
          `Please update your workflow configuration to use a supported model.\n` +
          `You can specify multiple models in priority order using an array:\n` +
          `  model: ["preferred-model", "fallback-model", "gpt-*"]`,
      );
      return;
    }

    // Set outputs
    core.setOutput("selected_model", result.selectedModel);
    core.setOutput("matched_pattern", result.matchedPattern);

    // Log success
    core.info(`✅ Selected model: ${result.selectedModel}`);
    if (result.matchedPattern !== result.selectedModel) {
      core.info(`   (matched pattern: ${result.matchedPattern})`);
    }

    // Write to step summary
    await core.summary
      .addHeading("Model Selection", 2)
      .addTable([
        [
          { data: "Field", header: true },
          { data: "Value", header: true },
        ],
        ["Selected Model", result.selectedModel],
        ["Matched Pattern", result.matchedPattern],
        [
          "Requested Models",
          requestedModels.map((m) => `\`${m}\``).join(", "),
        ],
      ])
      .write();
  } catch (error) {
    core.setFailed(`Model selection failed: ${error.message}`);
  }
}

run();
