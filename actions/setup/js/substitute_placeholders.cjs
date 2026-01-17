const fs = require("fs");
const { getErrorMessage } = require("./error_helpers.cjs");

const substitutePlaceholders = async ({ file, substitutions }) => {
  if (!file) throw new Error("file parameter is required");
  if (!substitutions || "object" != typeof substitutions) throw new Error("substitutions parameter must be an object");
  let content;
  try {
    content = fs.readFileSync(file, "utf8");
  } catch (error) {
    const errorMessage = getErrorMessage(error);
    throw new Error(`Failed to read file ${file}: ${errorMessage}`);
  }
  for (const [key, value] of Object.entries(substitutions)) {
    const placeholder = `__${key}__`;
    // Convert undefined/null to empty string to avoid leaving "undefined" or "null" in the output
    const safeValue = value === undefined || value === null ? "" : value;
    content = content.split(placeholder).join(safeValue);
  }
  try {
    fs.writeFileSync(file, content, "utf8");
  } catch (error) {
    const errorMessage = getErrorMessage(error);
    throw new Error(`Failed to write file ${file}: ${errorMessage}`);
  }
  return `Successfully substituted ${Object.keys(substitutions).length} placeholder(s) in ${file}`;
};
module.exports = substitutePlaceholders;
