const fs = require("fs");
const path = require("path");
const child_process = require("child_process");

/**
 * Run a shell command synchronously while logging the command and its output.
 * Keeps behavior similar to child_process.execSync but logs stdout/stderr
 * and rethrows errors after logging useful information.
 *
 * @param {string} cmd - The shell command to run
 * @param {object} [options] - Options passed to execSync
 * @returns {string|Buffer} The command stdout (string when encoding present)
 */
function runCmd(cmd, options = {}) {
  // Prefer the existing `core` logger if available (this module runs inside GH Actions),
  // otherwise fall back to console.
  const info = typeof core !== "undefined" && core && core.info ? core.info.bind(core) : console.log.bind(console);
  const error = typeof core !== "undefined" && core && core.error ? core.error.bind(core) : console.error.bind(console);

  try {
    info(`$ ${cmd}`);

    const result = child_process.execSync(cmd, options);

    // Normalize Buffer -> string for easier logging/consumption.
    let out = result;
    if (Buffer.isBuffer(result)) {
      const enc = options && options.encoding ? options.encoding : "utf8";
      out = result.toString(enc);
    }

    if (out && String(out).trim()) {
      info(`stdout: ${String(out).trim()}`);
    }

    return out;
  } catch (err) {
    // err may contain stdout/stderr buffers on execSync failures
    const stdout = err && err.stdout ? (Buffer.isBuffer(err.stdout) ? err.stdout.toString("utf8") : String(err.stdout)) : "";
    const stderr = err && err.stderr ? (Buffer.isBuffer(err.stderr) ? err.stderr.toString("utf8") : String(err.stderr)) : "";

    if (stdout && stdout.trim()) {
      error(`stdout: ${stdout.trim()}`);
    }
    if (stderr && stderr.trim()) {
      error(`stderr: ${stderr.trim()}`);
    }

    const status = err && typeof err.status === "number" ? err.status : "unknown";
    error(`Command failed: ${cmd}`);
    error(`Exit code: ${status}`);

    // Rethrow so existing callers' error handling still works
    throw err;
  }
}

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

  let successCount = 0;
  let errorCount = 0;

  const wikiDir = `/tmp/wiki`;
  const serverHostname = new URL(process.env.GITHUB_SERVER_URL || "https://github.com").hostname;
  const baseWikiUrl = `${serverHostname}/${owner}/${repo}.wiki.git`;
  const token = process.env.GITHUB_TOKEN || process.env.INPUT_GITHUB_TOKEN || process.env.INPUT_TOKEN;

  // Clone (with token if available) or init new repo if wiki doesn't exist
  try {
    // Avoid leaking the token in logs
    core.info(`Cloning wiki repository (token auth): ${baseWikiUrl}`);
    runCmd(`git clone https://x-access-token:${token}@${serverHostname}/${owner}/${repo}.wiki.git ${wikiDir}`, {
      stdio: "pipe",
    });
  } catch (fatalClone) {
    core.setFailed(`Failed to prepare wiki repository: ${fatalClone instanceof Error ? fatalClone.message : String(fatalClone)}`);
    return;
  }

  // Configure git user once
  try {
    runCmd(`git config user.name "github-actions[bot]"`, { cwd: wikiDir, stdio: "pipe" });
    runCmd(`git config user.email "github-actions[bot]@users.noreply.github.com"`, { cwd: wikiDir, stdio: "pipe" });
  } catch (gitConfigErr) {
    core.setFailed(`Failed to configure git user: ${gitConfigErr instanceof Error ? gitConfigErr.message : String(gitConfigErr)}`);
    return;
  }

  // Write each page (staging all changes first)
  for (let i = 0; i < validWikiEdits.length; i++) {
    const wikiEdit = validWikiEdits[i];
    try {
      core.info(`Writing wiki page ${i + 1}/${validWikiEdits.length}: ${wikiEdit.path}`);
      const normalizedFilePath = normalizeWikiFilePath(wikiEdit.path);
      core.info(`Normalized wiki file path: ${normalizedFilePath}`);
      const pageDir = path.dirname(normalizedFilePath);
      if (pageDir !== ".") {
        const fullPageDir = path.join(wikiDir, pageDir);
        runCmd(`mkdir -p "${fullPageDir}"`, { stdio: "pipe" });
      }
      const wikiPagePath = path.join(wikiDir, normalizedFilePath);
      fs.writeFileSync(wikiPagePath, wikiEdit.content, "utf8");
      runCmd(`git add "${normalizedFilePath}"`, { cwd: wikiDir, stdio: "pipe" });
      successCount++;
    } catch (writeErr) {
      const msg = writeErr instanceof Error ? writeErr.message : String(writeErr);
      core.error(`❌ Failed to stage wiki page ${wikiEdit.path}: ${msg}`);
      errorCount++;
    }
  }

  // Check if there is anything to commit
  let statusOutput = "";
  try {
    statusOutput = runCmd(`git status --porcelain`, { cwd: wikiDir, encoding: "utf8" });
  } catch (statusErr) {
    core.setFailed(`Failed to check git status: ${statusErr instanceof Error ? statusErr.message : String(statusErr)}`);
    return;
  }

  if (statusOutput.trim()) {
    // Build a consolidated commit message
    const uniqueMessages = [...new Set(validWikiEdits.map(e => e.message).filter(Boolean))];
    let commitMessage;
    if (uniqueMessages.length === 1) {
      commitMessage = uniqueMessages[0];
    } else {
      commitMessage = `Update ${validWikiEdits.length} wiki pages`;
      if (uniqueMessages.length > 1) {
        commitMessage +=
          `\n\n` +
          uniqueMessages
            .slice(0, 20)
            .map(m => `- ${m}`)
            .join("\n");
        if (uniqueMessages.length > 20) {
          commitMessage += `\n- ...and ${uniqueMessages.length - 20} more messages`;
        }
      }
    }

    // Write commit message to a temp file to avoid shell quoting issues
    const commitMsgFile = path.join(wikiDir, `.commit-message.txt`);
    fs.writeFileSync(commitMsgFile, commitMessage, "utf8");

    try {
      runCmd(`git commit -F .commit-message.txt`, { cwd: wikiDir, stdio: "pipe" });
    } catch (commitErr) {
      core.setFailed(`Failed to commit wiki changes: ${commitErr instanceof Error ? commitErr.message : String(commitErr)}`);
      return;
    }

    // Push changes (token embedded in remote URL already if provided)
    try {
      runCmd(`git push origin HEAD:master`, { cwd: wikiDir, stdio: "pipe" });
    } catch (pushErr) {
      core.setFailed(`Failed to push wiki changes: ${pushErr instanceof Error ? pushErr.message : String(pushErr)}`);
      return;
    }
  } else {
    core.info("No wiki changes to commit");
  }

  core.info(`Wiki editing completed: ${successCount} processed, ${errorCount} failed to stage`);
  if (errorCount > 0 && successCount === 0) {
    core.setFailed(`All wiki edits failed (${errorCount})`);
  } else if (errorCount > 0) {
    core.warning(`Some wiki edits failed: ${errorCount}`);
  } else {
    core.info("🎉 All wiki pages edited successfully!");
  }
}

await main();
