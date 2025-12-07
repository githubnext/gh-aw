/**
 * @fileoverview Safe template placeholder substitution for GitHub Actions workflows.
 * Replaces __VAR__ placeholders in a file with environment variable values without
 * allowing shell expansion, preventing template injection attacks.
 *
 * @param {object} params - The parameters object
 * @param {string} params.file - Path to the file to process
 * @param {object} params.substitutions - Map of placeholder names to values (without __ prefix/suffix)
 */

const fs = require("fs");

module.exports = async ({ file, substitutions }) => {
  // Validate inputs
  if (!file) {
    throw new Error("file parameter is required");
  }
  if (!substitutions || typeof substitutions !== "object") {
    throw new Error("substitutions parameter must be an object");
  }

  // Read the file content
  let content;
  try {
    content = fs.readFileSync(file, "utf8");
  } catch (error) {
    throw new Error(`Failed to read file ${file}: ${error.message}`);
  }

  // Perform substitutions
  // Each placeholder is in the format __VARIABLE_NAME__
  // We replace it with the corresponding value from the substitutions object
  for (const [key, value] of Object.entries(substitutions)) {
    const placeholder = `__${key}__`;
    // Use a simple string replacement - no regex to avoid any potential issues
    // with special characters in the value
    content = content.split(placeholder).join(value);
  }

  // Write the updated content back to the file
  try {
    fs.writeFileSync(file, content, "utf8");
  } catch (error) {
    throw new Error(`Failed to write file ${file}: ${error.message}`);
  }

  return `Successfully substituted ${Object.keys(substitutions).length} placeholder(s) in ${file}`;
};
