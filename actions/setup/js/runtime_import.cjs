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
 * Processes a URL import and returns content with sanitization
 * @param {string} url - The URL to fetch
 * @param {boolean} optional - Whether the import is optional
 * @param {number} [startLine] - Optional start line (1-indexed, inclusive)
 * @param {number} [endLine] - Optional end line (1-indexed, inclusive)
 * @returns {Promise<string>} - The processed URL content
 * @throws {Error} - If URL fetch fails or content is invalid
 */
async function processUrlImport(url, optional, startLine, endLine) {
  const cacheDir = "/tmp/gh-aw/url-cache";

  // Fetch URL content (with caching)
  let content;
  try {
    content = await fetchUrlContent(url, cacheDir);
  } catch (error) {
    if (optional) {
      const errorMessage = getErrorMessage(error);
      core.warning(`Optional runtime import URL failed: ${url}: ${errorMessage}`);
      return "";
    }
    throw error;
  }

  // If line range is specified, extract those lines first (before other processing)
  if (startLine !== undefined || endLine !== undefined) {
    const lines = content.split("\n");
    const totalLines = lines.length;

    // Validate line numbers (1-indexed)
    const start = startLine !== undefined ? startLine : 1;
    const end = endLine !== undefined ? endLine : totalLines;

    if (start < 1 || start > totalLines) {
      throw new Error(`Invalid start line ${start} for URL ${url} (total lines: ${totalLines})`);
    }
    if (end < 1 || end > totalLines) {
      throw new Error(`Invalid end line ${end} for URL ${url} (total lines: ${totalLines})`);
    }
    if (start > end) {
      throw new Error(`Start line ${start} cannot be greater than end line ${end} for URL ${url}`);
    }

    // Extract lines (convert to 0-indexed)
    content = lines.slice(start - 1, end).join("\n");
  }

  // Check for front matter and warn
  if (hasFrontMatter(content)) {
    core.warning(`URL ${url} contains front matter which will be ignored in runtime import`);
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
    throw new Error(`URL ${url} contains GitHub Actions macros (\${{ ... }}) which are not allowed in runtime imports`);
  }

  return content;
}

/**
 * Reads and processes a file or URL for runtime import
 * @param {string} filepathOrUrl - The path to the file (relative to GITHUB_WORKSPACE) or URL to import
 * @param {boolean} optional - Whether the import is optional (true for {{#runtime-import? filepath}})
 * @param {string} workspaceDir - The GITHUB_WORKSPACE directory path
 * @param {number} [startLine] - Optional start line (1-indexed, inclusive)
 * @param {number} [endLine] - Optional end line (1-indexed, inclusive)
 * @returns {Promise<string>} - The processed file or URL content, or empty string if optional and file not found
 * @throws {Error} - If file/URL is not found and import is not optional, or if GitHub Actions macros are detected
 */
async function processRuntimeImport(filepathOrUrl, optional, workspaceDir, startLine, endLine) {
  // Check if this is a URL
  if (/^https?:\/\//i.test(filepathOrUrl)) {
    return await processUrlImport(filepathOrUrl, optional, startLine, endLine);
  }

  // Otherwise, process as a file
  let filepath = filepathOrUrl;

  // Trim .github/ prefix if provided (support both .github/file and file)
  // This allows users to use either format
  if (filepath.startsWith(".github/")) {
    filepath = filepath.substring(8); // Remove ".github/"
  } else if (filepath.startsWith(".github\\")) {
    filepath = filepath.substring(8); // Remove ".github\" (Windows)
  }

  // Remove leading ./ or ../ if present
  if (filepath.startsWith("./")) {
    filepath = filepath.substring(2);
  } else if (filepath.startsWith(".\\")) {
    filepath = filepath.substring(2);
  }
  // Note: We don't allow ../ paths as they would escape .github folder

  // Construct the path within .github folder
  const githubFolder = path.join(workspaceDir, ".github");
  const absolutePath = path.resolve(githubFolder, filepath);
  const normalizedPath = path.normalize(absolutePath);
  const normalizedGithubFolder = path.normalize(githubFolder);

  // Security check: ensure the resolved path is within the .github folder
  // Use path.relative to check if the path escapes the .github folder
  const relativePath = path.relative(normalizedGithubFolder, normalizedPath);
  if (relativePath.startsWith("..") || path.isAbsolute(relativePath)) {
    throw new Error(`Security: Path ${filepathOrUrl} must be within .github folder (resolves to: ${relativePath})`);
  }

  // Check if file exists
  if (!fs.existsSync(normalizedPath)) {
    if (optional) {
      core.warning(`Optional runtime import file not found: ${filepath}`);
      return "";
    }
    throw new Error(`Runtime import file not found: ${filepath}`);
  }

  // Read the file
  let content = fs.readFileSync(normalizedPath, "utf8");

  // If line range is specified, extract those lines first (before other processing)
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
    throw new Error(`File ${filepath} contains GitHub Actions macros (\${{ ... }}) which are not allowed in runtime imports`);
  }

  return content;
}

/**
 * Processes all runtime-import macros in the content
 * @param {string} content - The markdown content containing runtime-import macros
 * @param {string} workspaceDir - The GITHUB_WORKSPACE directory path
 * @returns {Promise<string>} - Content with runtime-import macros replaced by file/URL contents
 */
async function processRuntimeImports(content, workspaceDir) {
  // Pattern to match {{#runtime-import filepath}} or {{#runtime-import? filepath}}
  // Captures: optional flag (?), whitespace, filepath/URL (which may include :startline-endline)
  const pattern = /\{\{#runtime-import(\?)?[ \t]+([^\}]+?)\}\}/g;

  let processedContent = content;
  const matches = [];
  let match;

  // Reset regex state and collect all matches
  pattern.lastIndex = 0;

  while ((match = pattern.exec(content)) !== null) {
    const optional = match[1] === "?";
    const filepathWithRange = match[2].trim();
    const fullMatch = match[0];

    // Parse filepath/URL and optional line range (filepath:startline-endline)
    const rangeMatch = filepathWithRange.match(/^(.+?):(\d+)-(\d+)$/);
    let filepathOrUrl, startLine, endLine;

    if (rangeMatch) {
      filepathOrUrl = rangeMatch[1];
      startLine = parseInt(rangeMatch[2], 10);
      endLine = parseInt(rangeMatch[3], 10);
    } else {
      filepathOrUrl = filepathWithRange;
      startLine = undefined;
      endLine = undefined;
    }

    matches.push({
      fullMatch,
      filepathOrUrl,
      optional,
      startLine,
      endLine,
      filepathWithRange,
    });
  }

  // Process all imports sequentially (to handle async URLs)
  const importedFiles = new Set();

  for (const matchData of matches) {
    const { fullMatch, filepathOrUrl, optional, startLine, endLine, filepathWithRange } = matchData;

    // Check for circular/duplicate imports
    if (importedFiles.has(filepathWithRange)) {
      core.warning(`File/URL ${filepathWithRange} is imported multiple times, which may indicate a circular reference`);
    }
    importedFiles.add(filepathWithRange);

    try {
      const importedContent = await processRuntimeImport(filepathOrUrl, optional, workspaceDir, startLine, endLine);
      // Replace the macro with the imported content
      processedContent = processedContent.replace(fullMatch, importedContent);
    } catch (error) {
      const errorMessage = getErrorMessage(error);
      throw new Error(`Failed to process runtime import for ${filepathWithRange}: ${errorMessage}`);
    }
  }

  return processedContent;
}

/**
 * Converts inline syntax to runtime-import macros
 * - File paths: `@./path` or `@../path` (must start with ./ or ../)
 * - URLs: `@https://...` or `@http://...`
 * @param {string} content - The markdown content containing inline references
 * @returns {string} - Content with inline references converted to runtime-import macros
 */
function convertInlinesToMacros(content) {
  let processedContent = content;

  // First, process URL patterns (@https://... or @http://...)
  const urlPattern = /@(https?:\/\/[^\s]+?)(?::(\d+)-(\d+))?(?=[\s\n]|$)/g;
  let match;

  urlPattern.lastIndex = 0;
  while ((match = urlPattern.exec(content)) !== null) {
    const url = match[1];
    const startLine = match[2];
    const endLine = match[3];
    const fullMatch = match[0];

    // Skip if this looks like part of an email address
    const matchIndex = match.index;
    if (matchIndex > 0) {
      const charBefore = content[matchIndex - 1];
      if (/[a-zA-Z0-9_]/.test(charBefore)) {
        continue;
      }
    }

    // Convert to {{#runtime-import URL}} or {{#runtime-import URL:start-end}}
    let macro;
    if (startLine && endLine) {
      macro = `{{#runtime-import ${url}:${startLine}-${endLine}}}`;
    } else {
      macro = `{{#runtime-import ${url}}}`;
    }

    processedContent = processedContent.replace(fullMatch, macro);
  }

  // Then, process file path patterns (@./path or @../path or @./path:line-line)
  // This pattern matches ONLY relative paths starting with ./ or ../
  // - @./file.ext
  // - @./path/to/file.ext
  // - @../path/to/file.ext:10-20
  // But NOT:
  // - @path (without ./ or ../)
  // - email addresses like user@example.com
  // - URLs (already processed)
  const filePattern = /@(\.\.?\/[a-zA-Z0-9_\-./]+)(?::(\d+)-(\d+))?/g;

  filePattern.lastIndex = 0;
  while ((match = filePattern.exec(processedContent)) !== null) {
    const filepath = match[1];
    const startLine = match[2];
    const endLine = match[3];
    const fullMatch = match[0];

    // Skip if this looks like part of an email address
    const matchIndex = match.index;
    if (matchIndex > 0) {
      const charBefore = processedContent[matchIndex - 1];
      if (/[a-zA-Z0-9_]/.test(charBefore)) {
        continue;
      }
    }

    // Convert to {{#runtime-import filepath}} or {{#runtime-import filepath:start-end}}
    let macro;
    if (startLine && endLine) {
      macro = `{{#runtime-import ${filepath}:${startLine}-${endLine}}}`;
    } else {
      macro = `{{#runtime-import ${filepath}}}`;
    }

    processedContent = processedContent.replace(fullMatch, macro);
  }

  return processedContent;
}

module.exports = {
  processRuntimeImports,
  processRuntimeImport,
  convertInlinesToMacros,
  hasFrontMatter,
  removeXMLComments,
  hasGitHubActionsMacros,
};
