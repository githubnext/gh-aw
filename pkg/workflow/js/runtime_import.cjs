// @ts-check
/// <reference types="@actions/github-script" />

// runtime_import.cjs
// Processes {{#runtime-import filepath}} and {{#runtime-import? filepath}} macros
// at runtime to import markdown file contents dynamically.

const fs = require("fs");
const path = require("path");

/**
 * Checks if a file starts with front matter (---\n)
 * @param {string} content - The file content to check
 * @returns {boolean} - True if content starts with front matter
 */
function hasFrontMatter(content) {
  return content.trimStart().startsWith("---\n") || content.trimStart().startsWith("---\r\n");
}

/**
 * Removes XML comments from content
 * @param {string} content - The content to process
 * @returns {string} - Content with XML comments removed
 */
function removeXMLComments(content) {
  // Remove XML/HTML comments: <!-- ... -->
  return content.replace(/<!--[\s\S]*?-->/g, "");
}

/**
 * Checks if content contains GitHub Actions macros (${{ ... }})
 * @param {string} content - The content to check
 * @returns {boolean} - True if GitHub Actions macros are found
 */
function hasGitHubActionsMacros(content) {
  return /\$\{\{[\s\S]*?\}\}/.test(content);
}

/**
 * Reads and processes a file for runtime import
 * @param {string} filepath - The path to the file to import (relative to GITHUB_WORKSPACE)
 * @param {boolean} optional - Whether the import is optional (true for {{#runtime-import? filepath}})
 * @param {string} workspaceDir - The GITHUB_WORKSPACE directory path
 * @returns {string} - The processed file content, or empty string if optional and file not found
 * @throws {Error} - If file is not found and import is not optional, or if GitHub Actions macros are detected
 */
function processRuntimeImport(filepath, optional, workspaceDir) {
  // Resolve the absolute path
  const absolutePath = path.resolve(workspaceDir, filepath);

  // Check if file exists
  if (!fs.existsSync(absolutePath)) {
    if (optional) {
      core.warning(`Optional runtime import file not found: ${filepath}`);
      return "";
    }
    throw new Error(`Runtime import file not found: ${filepath}`);
  }

  // Read the file
  let content = fs.readFileSync(absolutePath, "utf8");

  // Check for front matter and warn
  if (hasFrontMatter(content)) {
    core.warning(`File ${filepath} contains front matter which will be ignored in runtime import`);
    // Remove front matter (everything between first --- and second ---)
    const lines = content.split("\n");
    let inFrontMatter = false;
    let frontMatterCount = 0;
    const processedLines = [];

    for (const line of lines) {
      if (line.trim() === "---" || line.trim() === "---\r") {
        frontMatterCount++;
        if (frontMatterCount === 1) {
          inFrontMatter = true;
          continue;
        } else if (frontMatterCount === 2) {
          inFrontMatter = false;
          continue;
        }
      }
      if (!inFrontMatter && frontMatterCount >= 2) {
        processedLines.push(line);
      }
    }
    content = processedLines.join("\n");
  }

  // Remove XML comments
  content = removeXMLComments(content);

  // Check for GitHub Actions macros and error if found
  if (hasGitHubActionsMacros(content)) {
    throw new Error(`File ${filepath} contains GitHub Actions macros ($\{{ ... }}) which are not allowed in runtime imports`);
  }

  return content;
}

/**
 * Processes all runtime-import macros in the content
 * @param {string} content - The markdown content containing runtime-import macros
 * @param {string} workspaceDir - The GITHUB_WORKSPACE directory path
 * @returns {string} - Content with runtime-import macros replaced by file contents
 */
function processRuntimeImports(content, workspaceDir) {
  // Pattern to match {{#runtime-import filepath}} or {{#runtime-import? filepath}}
  // Captures: optional flag (?), whitespace, filepath
  const pattern = /\{\{#runtime-import(\?)?[ \t]+([^\}]+?)\}\}/g;

  let processedContent = content;
  let match;
  const importedFiles = new Set();

  // Reset regex state
  pattern.lastIndex = 0;

  while ((match = pattern.exec(content)) !== null) {
    const optional = match[1] === "?";
    const filepath = match[2].trim();
    const fullMatch = match[0];

    // Check for circular/duplicate imports
    if (importedFiles.has(filepath)) {
      core.warning(`File ${filepath} is imported multiple times, which may indicate a circular reference`);
    }
    importedFiles.add(filepath);

    try {
      const importedContent = processRuntimeImport(filepath, optional, workspaceDir);
      // Replace the macro with the imported content
      processedContent = processedContent.replace(fullMatch, importedContent);
    } catch (error) {
      throw new Error(`Failed to process runtime import for ${filepath}: ${error.message}`);
    }
  }

  return processedContent;
}

module.exports = {
  processRuntimeImports,
  processRuntimeImport,
  hasFrontMatter,
  removeXMLComments,
  hasGitHubActionsMacros,
};
