// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");

const substitutePlaceholders = async ({ file, substitutions }) => {
  if (!file) throw new Error("file parameter is required");
  if (!substitutions || "object" != typeof substitutions) throw new Error("substitutions parameter must be an object");
  let content;
  try {
    content = fs.readFileSync(file, "utf8");
  } catch (error) {
    throw new Error(`Failed to read file ${file}: ${error.message}`);
  }
  for (const [key, value] of Object.entries(substitutions)) {
    const placeholder = `__${key}__`;
    content = content.split(placeholder).join(value);
  }
  try {
    fs.writeFileSync(file, content, "utf8");
  } catch (error) {
    throw new Error(`Failed to write file ${file}: ${error.message}`);
  }
  return `Successfully substituted ${Object.keys(substitutions).length} placeholder(s) in ${file}`;
};

async function main() {
  // Get the file path from environment
  const file = process.env.GH_AW_PROMPT;
  if (!file) {
    throw new Error("GH_AW_PROMPT environment variable is required");
  }

  // Build substitutions object from environment variables
  const substitutions = {};
  for (const [key, value] of Object.entries(process.env)) {
    if (key.startsWith("GH_AW_") && key !== "GH_AW_PROMPT") {
      substitutions[key] = value;
    }
  }

  // Call the substitution function
  const result = await substitutePlaceholders({ file, substitutions });
  core.info(result);
  return result;
}

module.exports = { main, substitutePlaceholders };
