// @ts-check
/// <reference types="@actions/github-script" />

// interpolate_prompt.cjs
// Interpolates GitHub Actions expressions in the prompt file using github-script.
// This replaces the previous approach of using shell variable expansion.

const fs = require("fs");

/**
 * Interpolates variables in the prompt content
 * @param {string} content - The prompt content with ${GH_AW_EXPR_*} placeholders
 * @param {Record<string, string>} variables - Map of variable names to their values
 * @returns {string} - The interpolated content
 */
function interpolateVariables(content, variables) {
  let result = content;
  
  // Replace each ${VAR_NAME} with its corresponding value
  for (const [varName, value] of Object.entries(variables)) {
    const pattern = new RegExp(`\\$\\{${varName}\\}`, 'g');
    result = result.replace(pattern, value);
  }
  
  return result;
}

/**
 * Main function for prompt variable interpolation
 */
async function main() {
  try {
    const promptPath = process.env.GH_AW_PROMPT;
    if (!promptPath) {
      core.setFailed("GH_AW_PROMPT environment variable is not set");
      return;
    }

    // Read the prompt file
    let content = fs.readFileSync(promptPath, "utf8");

    // Collect all GH_AW_EXPR_* environment variables
    const variables = {};
    for (const [key, value] of Object.entries(process.env)) {
      if (key.startsWith("GH_AW_EXPR_")) {
        variables[key] = value || "";
      }
    }

    // Log the number of variables found
    const varCount = Object.keys(variables).length;
    core.info(`Found ${varCount} expression variable(s) to interpolate`);

    if (varCount === 0) {
      core.info("No expression variables found, skipping interpolation");
      return;
    }

    // Interpolate variables
    const interpolated = interpolateVariables(content, variables);

    // Write back to the same file
    fs.writeFileSync(promptPath, interpolated, "utf8");

    core.info(`Successfully interpolated ${varCount} variable(s) in prompt`);
  } catch (error) {
    core.setFailed(error instanceof Error ? error.message : String(error));
  }
}

// Execute main function
main();
