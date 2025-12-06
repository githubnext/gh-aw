/**
 * @file select_model_with_cli.cjs
 * @description Selects a compatible model by calling copilot CLI to get available models.
 * Falls back to hardcoded list if CLI call fails. Supports wildcard patterns including
 * special "*" wildcard meaning "any model, don't specify".
 * 
 * Inputs:
 * - requested_models: JSON string array of model patterns (may include wildcards)
 * 
 * Outputs:
 * - selected_model: The first compatible model found (empty string for "*" wildcard)
 * - matched_pattern: The pattern that matched
 * 
 * Exits with error if no compatible models are found.
 */

const { execSync } = require("child_process");
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

    // Special case: "*" means "any model" or "don't specify a model"
    // Return empty string to indicate no model should be specified
    if (pattern === "*") {
      core.info(`✅ Pattern "*" matches any model - not specifying a model`);
      return { selectedModel: "", matchedPattern: pattern };
    }

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

/**
 * Gets available models by calling copilot CLI
 * @returns {string[]} Array of available model names
 */
function getAvailableModels() {
  let availableModels = [];
  
  try {
    const output = execSync("copilot --list-models", { encoding: "utf8" });
    availableModels = output
      .trim()
      .split("\n")
      .filter((m) => m.trim());
    core.info(`Available models from CLI: ${JSON.stringify(availableModels)}`);
  } catch (error) {
    core.warning(`Failed to get models from CLI: ${error.message}`);
    // Fallback to known models if CLI call fails
    availableModels = [
      "gpt-4",
      "gpt-4-turbo",
      "gpt-4o",
      "gpt-4o-mini",
      "gpt-5",
      "gpt-5-mini",
      "o1",
      "o1-mini",
      "o3",
      "o3-mini",
    ];
    core.info(`Using fallback model list: ${JSON.stringify(availableModels)}`);
  }
  
  return availableModels;
}

async function run() {
  try {
    // Get inputs
    const requestedModelsInput = core.getInput("requested_models", {
      required: true,
    });

    // Parse JSON input
    let requestedModels;
    try {
      requestedModels = JSON.parse(requestedModelsInput);
    } catch (error) {
      core.setFailed(
        `Failed to parse requested_models as JSON: ${error.message}`,
      );
      return;
    }

    // Validate input
    if (!Array.isArray(requestedModels) || requestedModels.length === 0) {
      core.setFailed(
        "requested_models must be a non-empty array of strings",
      );
      return;
    }

    // Get available models from CLI
    const availableModels = getAvailableModels();

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
    if (result.selectedModel === "") {
      core.info(`✅ Pattern '*' matched - not specifying a model`);
    } else {
      core.info(`✅ Selected model: ${result.selectedModel}`);
      if (result.matchedPattern !== result.selectedModel) {
        core.info(`   (matched pattern: ${result.matchedPattern})`);
      }
    }

    // Write to step summary
    await core.summary
      .addHeading("Model Selection", 2)
      .addTable([
        [
          { data: "Field", header: true },
          { data: "Value", header: true },
        ],
        [
          "Selected Model",
          result.selectedModel || "(none - any model)",
        ],
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
