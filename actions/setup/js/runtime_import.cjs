// @ts-check
/// <reference types="@actions/github-script" />

// runtime_import.cjs
// Processes {{#runtime-import filepath}} and {{#runtime-import? filepath}} macros
// at runtime to import markdown file contents dynamically.
// Also processes inline @path and @url references.

const { getErrorMessage } = require("./error_helpers.cjs");

const fs = require("fs");
const path = require("path");
const https = require("https");
const http = require("http");

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
  // Apply repeatedly to handle nested/overlapping patterns that could reintroduce comment markers
  let previous;
  do {
    previous = content;
    content = content.replace(/<!--[\s\S]*?-->/g, "");
  } while (content !== previous);
  return content;
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
      const errorMessage = getErrorMessage(error);
      throw new Error(`Failed to process runtime import for ${filepath}: ${errorMessage}`);
    }
  }

  return processedContent;
}

/**
 * Processes a file inline and returns content with sanitization
 * @param {string} filepath - The path to the file (relative to GITHUB_WORKSPACE)
 * @param {string} workspaceDir - The GITHUB_WORKSPACE directory path
 * @param {number} [startLine] - Optional start line (1-indexed, inclusive)
 * @param {number} [endLine] - Optional end line (1-indexed, inclusive)
 * @returns {string} - The processed file content
 * @throws {Error} - If file is not found or line ranges are invalid
 */
function processFileInline(filepath, workspaceDir, startLine, endLine) {
  // Resolve the absolute path
  const absolutePath = path.resolve(workspaceDir, filepath);

  // Check if file exists
  if (!fs.existsSync(absolutePath)) {
    throw new Error(`File not found for inline: ${filepath}`);
  }

  // Read the file
  let content = fs.readFileSync(absolutePath, "utf8");

  // If line range is specified, extract those lines
  if (startLine !== undefined || endLine !== undefined) {
    const lines = content.split("\n");
    const totalLines = lines.length;

    // Validate line numbers (1-indexed)
    const start = startLine !== undefined ? startLine : 1;
    const end = endLine !== undefined ? endLine : totalLines;

    if (start < 1 || start > totalLines) {
      throw new Error(`Invalid start line ${start} for file ${filepath} (total lines: ${totalLines})`);
    }
    if (end < 1 || end > totalLines) {
      throw new Error(`Invalid end line ${end} for file ${filepath} (total lines: ${totalLines})`);
    }
    if (start > end) {
      throw new Error(`Start line ${start} cannot be greater than end line ${end} for file ${filepath}`);
    }

    // Extract lines (convert to 0-indexed)
    content = lines.slice(start - 1, end).join("\n");
  }

  // Check for front matter and warn
  if (hasFrontMatter(content)) {
    core.warning(`File ${filepath} contains front matter which will be ignored in inline`);
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
    throw new Error(`File ${filepath} contains GitHub Actions macros ($\{{ ... }}) which are not allowed in inline content`);
  }

  return content;
}

/**
 * Processes all @path and @path:line-line inline references in the content
 * @param {string} content - The markdown content containing @path references
 * @param {string} workspaceDir - The GITHUB_WORKSPACE directory path
 * @returns {string} - Content with @path references replaced by file contents
 */
function processFileInlines(content, workspaceDir) {
  // Pattern to match @filepath or @filepath:startline-endline
  // This pattern matches:
  // - @path/to/file.ext
  // - @file.ext
  // - @path/to/file.ext:10-20
  // But NOT email addresses like user@example.com
  // We require the path to contain at least one of: /, ., - or alphanumeric followed by /, ., -
  // This ensures we match file paths but not bare domain names in emails
  const pattern = /@([a-zA-Z0-9_\-./]+[a-zA-Z0-9_])(?::(\d+)-(\d+))?/g;

  let processedContent = content;
  let match;

  // Reset regex state
  pattern.lastIndex = 0;

  while ((match = pattern.exec(content)) !== null) {
    const filepath = match[1];
    const startLine = match[2] ? parseInt(match[2], 10) : undefined;
    const endLine = match[3] ? parseInt(match[3], 10) : undefined;
    const fullMatch = match[0];

    // Skip if this looks like part of an email address
    // Check if there's an alphanumeric character immediately before the @
    const matchIndex = match.index;
    if (matchIndex > 0) {
      const charBefore = content[matchIndex - 1];
      if (/[a-zA-Z0-9_]/.test(charBefore)) {
        // This is likely an email address, skip it
        continue;
      }
    }

    try {
      const inlinedContent = processFileInline(filepath, workspaceDir, startLine, endLine);
      // Replace the @path reference with the inlined content
      processedContent = processedContent.replace(fullMatch, inlinedContent);
    } catch (error) {
      const errorMessage = getErrorMessage(error);
      throw new Error(`Failed to process inline for ${fullMatch}: ${errorMessage}`);
    }
  }

  return processedContent;
}

/**
 * Fetches content from a URL with caching
 * @param {string} url - The URL to fetch
 * @param {string} cacheDir - Directory to store cached URL content
 * @returns {Promise<string>} - The fetched content
 * @throws {Error} - If URL fetch fails
 */
async function fetchUrlContent(url, cacheDir) {
  // Create cache directory if it doesn't exist
  if (!fs.existsSync(cacheDir)) {
    fs.mkdirSync(cacheDir, { recursive: true });
  }

  // Generate cache filename from URL (hash it for safety)
  const crypto = require("crypto");
  const urlHash = crypto.createHash("sha256").update(url).digest("hex");
  const cacheFile = path.join(cacheDir, `url-${urlHash}.cache`);

  // Check if cached version exists and is recent (less than 1 hour old)
  if (fs.existsSync(cacheFile)) {
    const stats = fs.statSync(cacheFile);
    const ageInMs = Date.now() - stats.mtimeMs;
    const oneHourInMs = 60 * 60 * 1000;

    if (ageInMs < oneHourInMs) {
      core.info(`Using cached content for URL: ${url}`);
      return fs.readFileSync(cacheFile, "utf8");
    }
  }

  // Fetch URL content
  core.info(`Fetching content from URL: ${url}`);

  return new Promise((resolve, reject) => {
    const protocol = url.startsWith("https") ? https : http;

    protocol
      .get(url, res => {
        if (res.statusCode !== 200) {
          reject(new Error(`Failed to fetch URL ${url}: HTTP ${res.statusCode}`));
          return;
        }

        let data = "";
        res.on("data", chunk => {
          data += chunk;
        });

        res.on("end", () => {
          // Cache the content
          fs.writeFileSync(cacheFile, data, "utf8");
          resolve(data);
        });
      })
      .on("error", err => {
        reject(new Error(`Failed to fetch URL ${url}: ${err.message}`));
      });
  });
}

/**
 * Processes a URL inline and returns content with sanitization
 * @param {string} url - The URL to fetch
 * @param {string} cacheDir - Directory to store cached URL content
 * @returns {Promise<string>} - The processed URL content
 * @throws {Error} - If URL fetch fails or content is invalid
 */
async function processUrlInline(url, cacheDir) {
  // Fetch URL content (with caching)
  let content = await fetchUrlContent(url, cacheDir);

  // Check for front matter and warn
  if (hasFrontMatter(content)) {
    core.warning(`URL ${url} contains front matter which will be ignored in inline`);
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
    throw new Error(`URL ${url} contains GitHub Actions macros ($\{{ ... }}) which are not allowed in inline content`);
  }

  return content;
}

/**
 * Processes all @url inline references in the content
 * @param {string} content - The markdown content containing @url references
 * @param {string} cacheDir - Directory to store cached URL content
 * @returns {Promise<string>} - Content with @url references replaced by URL contents
 */
async function processUrlInlines(content, cacheDir) {
  // Pattern to match @https://... or @http://...
  const pattern = /@(https?:\/\/[^\s]+)/g;

  let processedContent = content;
  const matches = [];

  // Collect all matches first
  let match;
  pattern.lastIndex = 0;
  while ((match = pattern.exec(content)) !== null) {
    matches.push({
      fullMatch: match[0],
      url: match[1],
    });
  }

  // Process each match sequentially (to handle async)
  for (const { fullMatch, url } of matches) {
    try {
      const inlinedContent = await processUrlInline(url, cacheDir);
      // Replace the @url reference with the inlined content
      processedContent = processedContent.replace(fullMatch, inlinedContent);
    } catch (error) {
      const errorMessage = getErrorMessage(error);
      throw new Error(`Failed to process URL inline for ${fullMatch}: ${errorMessage}`);
    }
  }

  return processedContent;
}

module.exports = {
  processRuntimeImports,
  processRuntimeImport,
  processFileInline,
  processFileInlines,
  processUrlInline,
  processUrlInlines,
  hasFrontMatter,
  removeXMLComments,
  hasGitHubActionsMacros,
};
