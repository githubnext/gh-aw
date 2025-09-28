const fs = require("fs");
const path = require("path");
const { execSync } = require("child_process");

/**
 * Sanitizes markdown content for wiki pages
 * @param {string} content - The content to sanitize
 * @returns {string} The sanitized content
 */
function sanitizeMarkdown(content) {
  if (!content || typeof content !== "string") {
    return "";
  }

  let sanitized = content;

  // Remove script tags and their content
  sanitized = sanitized.replace(/<script\b[^<]*(?:(?!<\/script>)<[^<]*)*<\/script>/gi, "");

  // Remove potentially dangerous HTML attributes
  sanitized = sanitized.replace(/\s*(?:on\w+|javascript:|data:)\s*=\s*["'][^"']*["']/gi, "");

  // Remove form elements that could be used maliciously
  sanitized = sanitized.replace(/<\s*(?:form|input|textarea|select|option|button)\b[^>]*>/gi, "");
  sanitized = sanitized.replace(/<\/\s*(?:form|input|textarea|select|option|button)\s*>/gi, "");

  // Remove potentially dangerous protocols (keep only http, https, mailto)
  sanitized = sanitized.replace(/\bhref\s*=\s*["'](?!(?:https?|mailto):)[^"']*["']/gi, 'href="#"');
  sanitized = sanitized.replace(/\bsrc\s*=\s*["'](?!https?:)[^"']*["']/gi, 'src="#"');

  // Normalize line endings
  sanitized = sanitized.replace(/\r\n/g, "\n").replace(/\r/g, "\n");

  // Limit content length (max 500KB for wiki pages)
  const maxLength = 500 * 1024;
  if (sanitized.length > maxLength) {
    sanitized = sanitized.substring(0, maxLength) + "\n\n*Content truncated due to length limits*";
  }

  return sanitized;
}

/**
 * Normalizes wiki file path for proper GitHub wiki structure
 * @param {string} inputPath - The input path from user
 * @returns {string} Normalized path with .md extension
 */
function normalizeWikiFilePath(inputPath) {
  if (!inputPath || typeof inputPath !== "string") {
    throw new Error("Wiki path must be a non-empty string");
  }

  // Enforce relative path - remove leading slashes
  let normalizedPath = inputPath.replace(/^\/+/, "");

  // Remove trailing slashes
  normalizedPath = normalizedPath.replace(/\/+$/, "");

  // Ensure .md extension
  if (!normalizedPath.endsWith(".md")) {
    normalizedPath += ".md";
  }

  // Return the path without forcing pages/ directory
  const fullPath = normalizedPath;

  return fullPath;
}

/**
 * Validates wiki page path against allowed patterns
 * @param {string} pagePath - The page path to validate (before normalization)
 * @param {string[]} allowedPaths - Array of allowed path patterns
 * @param {string} workflowName - Default workflow name for path restriction
 * @returns {boolean} Whether the path is allowed
 */
function validatePagePath(pagePath, allowedPaths, workflowName) {
  if (!pagePath || typeof pagePath !== "string") {
    return false;
  }

  // Remove leading/trailing slashes and normalize for validation
  const normalizedPath = pagePath.replace(/^\/+|\/+$/g, "").replace(/\/+/g, "/");

  // If no allowed paths configured, default to workflow name folder
  const pathsToCheck = allowedPaths && allowedPaths.length > 0 ? allowedPaths : [workflowName];

  for (const allowedPath of pathsToCheck) {
    const normalizedAllowed = allowedPath.replace(/^\/+|\/+$/g, "").replace(/\/+/g, "/");

    // Allow exact match or path starting with allowed pattern
    if (normalizedPath === normalizedAllowed || normalizedPath.startsWith(normalizedAllowed + "/")) {
      return true;
    }
  }

  return false;
}

async function main() {
  // Check if we're in staged mode
  const isStaged = process.env.GITHUB_AW_SAFE_OUTPUTS_STAGED === "true";

  // Read the validated output content from environment variable
  const outputContent = process.env.GITHUB_AW_AGENT_OUTPUT;
  if (!outputContent) {
    core.info("No GITHUB_AW_AGENT_OUTPUT environment variable found");
    return;
  }

  if (outputContent.trim() === "") {
    core.info("Agent output content is empty");
    return;
  }

  core.info(`Agent output content length: ${outputContent.length}`);

  // Parse the validated output JSON
  let validatedOutput;
  try {
    validatedOutput = JSON.parse(outputContent);
  } catch (error) {
    core.setFailed(`Error parsing agent output JSON: ${error instanceof Error ? error.message : String(error)}`);
    return;
  }

  if (!validatedOutput.items || !Array.isArray(validatedOutput.items)) {
    core.info("No valid items found in agent output");
    return;
  }

  // Find all edit-wiki-page items
  const wikiItems = validatedOutput.items.filter(/** @param {any} item */ item => item.type === "edit-wiki-page");
  if (wikiItems.length === 0) {
    core.info("No edit-wiki-page items found in agent output");
    return;
  }

  core.info(`Found ${wikiItems.length} edit-wiki item(s)`);

  // Get configuration from environment variables
  const workflowName = process.env.GITHUB_WORKFLOW_NAME || "workflow";
  const allowedPathsEnv = process.env.GITHUB_AW_WIKI_ALLOWED_PATHS;
  const allowedPaths = allowedPathsEnv ? allowedPathsEnv.split(",").map(p => p.trim()) : [];
  const maxEdits = process.env.GITHUB_AW_WIKI_MAX ? parseInt(process.env.GITHUB_AW_WIKI_MAX) : 1;

  core.info(`Workflow name: ${workflowName}`);
  core.info(`Allowed paths: ${allowedPaths.length > 0 ? allowedPaths.join(", ") : "default to workflow name"}`);
  core.info(`Max edits: ${maxEdits}`);

  // If in staged mode, emit step summary instead of editing wiki
  if (isStaged) {
    let summaryContent = "## 🎭 Staged Mode: Edit Wiki Pages Preview\n\n";
    summaryContent += "The following wiki pages would be edited if staged mode was disabled:\n\n";

    for (let i = 0; i < Math.min(wikiItems.length, maxEdits); i++) {
      const item = wikiItems[i];
      const isPathValid = validatePagePath(item.path, allowedPaths, workflowName);

      let normalizedPath = "N/A";
      try {
        normalizedPath = normalizeWikiFilePath(item.path);
      } catch (error) {
        // Keep N/A if path normalization fails
      }

      summaryContent += `### Wiki Page ${i + 1}\n`;
      summaryContent += `**Path:** ${item.path || "No path provided"}\n\n`;
      summaryContent += `**Normalized Path:** ${normalizedPath}\n\n`;
      summaryContent += `**Path Valid:** ${isPathValid ? "✅ Yes" : "❌ No (restricted)"}\n\n`;
      summaryContent += `**Content Length:** ${item.content ? item.content.length : 0} characters\n\n`;

      if (item.content && item.content.length > 0) {
        const preview = item.content.length > 200 ? item.content.substring(0, 200) + "..." : item.content;
        summaryContent += `**Content Preview:**\n\`\`\`markdown\n${preview}\n\`\`\`\n\n`;
      }

      summaryContent += "---\n\n";
    }

    if (wikiItems.length > maxEdits) {
      summaryContent += `*Note: ${wikiItems.length - maxEdits} additional wiki page(s) would be ignored due to max limit of ${maxEdits}*\n\n`;
    }

    // Write to step summary
    await core.summary.addRaw(summaryContent).write();
    core.info("📝 Wiki edit preview written to step summary");
    return;
  }

  // Validate and process wiki edits
  const validWikiEdits = [];

  for (let i = 0; i < Math.min(wikiItems.length, maxEdits); i++) {
    const item = wikiItems[i];

    if (!item.path) {
      core.warning(`Wiki edit ${i + 1}: Missing path, skipping`);
      continue;
    }

    if (!item.content) {
      core.warning(`Wiki edit ${i + 1}: Missing content, skipping`);
      continue;
    }

    // Validate path against allowed patterns
    if (!validatePagePath(item.path, allowedPaths, workflowName)) {
      core.warning(`Wiki edit ${i + 1}: Path '${item.path}' not allowed, skipping`);
      continue;
    }

    // Sanitize content
    const sanitizedContent = sanitizeMarkdown(item.content);
    if (!sanitizedContent.trim()) {
      core.warning(`Wiki edit ${i + 1}: Content became empty after sanitization, skipping`);
      continue;
    }

    validWikiEdits.push({
      path: item.path.trim(),
      content: sanitizedContent,
      message: item.message || `Update ${item.path} via GitHub Agentic Workflow`,
    });
  }

  if (validWikiEdits.length === 0) {
    core.info("No valid wiki edits found after validation");
    return;
  }

  core.info(`Processing ${validWikiEdits.length} valid wiki edit(s)`);

  // Get repository information
  const owner = context.repo.owner;
  const repo = context.repo.repo;

  core.info(`Repository: ${owner}/${repo}`);

  // Process each wiki edit
  let successCount = 0;
  let errorCount = 0;

  for (let i = 0; i < validWikiEdits.length; i++) {
    const wikiEdit = validWikiEdits[i];

    try {
      core.info(`Editing wiki page ${i + 1}/${validWikiEdits.length}: ${wikiEdit.path}`);

      // Clone or update the wiki repository
      const wikiUrl = `https://github.com/${owner}/${repo}.wiki.git`;
      const wikiDir = `/tmp/wiki-${Date.now()}-${i}`;

      // Clone the wiki repository (it might not exist yet)

      try {
        core.info(`Cloning wiki repository: ${wikiUrl}`);
        execSync(`git clone ${wikiUrl} ${wikiDir}`, { stdio: "pipe" });
      } catch (cloneError) {
        // Wiki doesn't exist yet, create it
        core.info("Wiki repository doesn't exist, creating new one");
        execSync(`mkdir -p ${wikiDir}`, { stdio: "pipe" });
        // Initialize git repository in the created directory
        execSync(`git init`, { cwd: wikiDir, stdio: "pipe" });
        execSync(`git remote add origin ${wikiUrl}`, { cwd: wikiDir, stdio: "pipe" });
      }

      // Configure git user (required for commits)
      execSync(`git config user.name "github-actions[bot]"`, { cwd: wikiDir, stdio: "pipe" });
      execSync(`git config user.email "github-actions[bot]@users.noreply.github.com"`, { cwd: wikiDir, stdio: "pipe" });

      // Normalize the wiki file path (adds .md extension and ensures relative path)
      const normalizedFilePath = normalizeWikiFilePath(wikiEdit.path);
      core.info(`Normalized wiki file path: ${normalizedFilePath}`);

      // Create directory structure if needed
      const pageDir = path.dirname(normalizedFilePath);
      if (pageDir !== ".") {
        const fullPageDir = path.join(wikiDir, pageDir);
        execSync(`mkdir -p "${fullPageDir}"`, { stdio: "pipe" });
      }

      // Write the content to the wiki page
      const wikiPagePath = path.join(wikiDir, normalizedFilePath);
      fs.writeFileSync(wikiPagePath, wikiEdit.content, "utf8");

      // Commit and push the changes
      execSync(`git add .`, { cwd: wikiDir, stdio: "pipe" });

      // Check if there are changes to commit
      const status = execSync(`git status --porcelain`, { cwd: wikiDir, encoding: "utf8" });
      if (!status.trim()) {
        core.info(`No changes detected for wiki page: ${wikiEdit.path}`);
        continue;
      }

      execSync(`git commit -m "${wikiEdit.message}"`, { cwd: wikiDir, stdio: "pipe" });
      execSync(`git push origin HEAD:master`, { cwd: wikiDir, stdio: "pipe" });

      // Clean up temporary directory
      execSync(`rm -rf ${wikiDir}`, { stdio: "pipe" });

      core.info(`✅ Successfully edited wiki page: ${wikiEdit.path}`);
      successCount++;
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      core.error(`❌ Failed to edit wiki page ${wikiEdit.path}: ${errorMessage}`);
      errorCount++;
    }
  }

  // Report final results
  core.info(`Wiki editing completed: ${successCount} successful, ${errorCount} failed`);

  if (errorCount > 0) {
    core.setFailed(`Failed to edit ${errorCount} wiki page(s). See logs for details.`);
  } else {
    core.info("🎉 All wiki pages edited successfully!");
  }
}

await main();
