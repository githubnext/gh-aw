// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Sanitizes content for safe output in GitHub Actions
 * @param {string} content - The content to sanitize
 * @returns {string} The sanitized content
 */
// === Inlined from ./lib/sanitize.cjs ===
// @ts-check
/**
 * Shared sanitization utilities for GitHub Actions output
 * This module provides functions for sanitizing content to prevent security issues
 * and unintended side effects in GitHub Actions workflows.
 */

/**
 * Sanitizes content for safe output in GitHub Actions
 * @param {string} content - The content to sanitize
 * @param {number} [maxLength] - Maximum length of content (default: 524288)
 * @returns {string} The sanitized content
 */
function sanitizeContent(content, maxLength) {
  if (!content || typeof content !== "string") {
    return "";
  }

  // Read allowed domains from environment variable
  const allowedDomainsEnv = process.env.GH_AW_ALLOWED_DOMAINS;
  const defaultAllowedDomains = ["github.com", "github.io", "githubusercontent.com", "githubassets.com", "github.dev", "codespaces.new"];

  const allowedDomains = allowedDomainsEnv
    ? allowedDomainsEnv
        .split(",")
        .map(d => d.trim())
        .filter(d => d)
    : defaultAllowedDomains;

  let sanitized = content;

  // Neutralize @mentions to prevent unintended notifications
  sanitized = neutralizeMentions(sanitized);

  // Remove XML comments first
  sanitized = removeXmlComments(sanitized);

  // Remove ANSI escape sequences
  sanitized = sanitized.replace(/\x1b\[[0-9;]*[mGKH]/g, "");

  // Remove control characters (except newlines and tabs)
  sanitized = sanitized.replace(/[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]/g, "");

  // URI filtering - replace non-https protocols with "(redacted)"
  sanitized = sanitizeUrlProtocols(sanitized);

  // Domain filtering for HTTPS URIs
  sanitized = sanitizeUrlDomains(sanitized);

  // Check line count before length to provide more specific truncation message
  const lines = sanitized.split("\n");
  const maxLines = 65000;
  maxLength = maxLength || 524288;

  // If content has too many lines, truncate by lines (primary limit)
  if (lines.length > maxLines) {
    const truncationMsg = "\n[Content truncated due to line count]";
    const truncatedLines = lines.slice(0, maxLines).join("\n") + truncationMsg;

    // If still too long after line truncation, shorten but keep the line count message
    if (truncatedLines.length > maxLength) {
      sanitized = truncatedLines.substring(0, maxLength - truncationMsg.length) + truncationMsg;
    } else {
      sanitized = truncatedLines;
    }
  } else if (sanitized.length > maxLength) {
    sanitized = sanitized.substring(0, maxLength) + "\n[Content truncated due to length]";
  }

  // Neutralize common bot trigger phrases
  sanitized = neutralizeBotTriggers(sanitized);

  // Trim excessive whitespace
  return sanitized.trim();

  /**
   * Remove unknown domains
   * @param {string} s - The string to process
   * @returns {string} The string with unknown domains redacted
   */
  function sanitizeUrlDomains(s) {
    s = s.replace(/\bhttps:\/\/([^\/\s\])}'"<>&\x00-\x1f,;]+)/gi, (match, domain) => {
      // Extract the hostname part (before first slash, colon, or other delimiter)
      const hostname = domain.split(/[\/:\?#]/)[0].toLowerCase();

      // Check if this domain or any parent domain is in the allowlist
      const isAllowed = allowedDomains.some(allowedDomain => {
        const normalizedAllowed = allowedDomain.toLowerCase();
        return hostname === normalizedAllowed || hostname.endsWith("." + normalizedAllowed);
      });

      return isAllowed ? match : "(redacted)";
    });

    return s;
  }

  /**
   * Remove unknown protocols except https
   * @param {string} s - The string to process
   * @returns {string} The string with non-https protocols redacted
   */
  function sanitizeUrlProtocols(s) {
    // Match both protocol:// and protocol: patterns
    // This covers URLs like https://example.com, javascript:alert(), mailto:user@domain.com, etc.
    return s.replace(/\b(\w+):(?:\/\/)?[^\s\])}'"<>&\x00-\x1f]+/gi, (match, protocol) => {
      // Allow https (case insensitive), redact everything else
      return protocol.toLowerCase() === "https" ? match : "(redacted)";
    });
  }

  /**
   * Neutralizes @mentions by wrapping them in backticks
   * @param {string} s - The string to process
   * @returns {string} The string with neutralized mentions
   */
  function neutralizeMentions(s) {
    // Replace @name or @org/team outside code with `@name`
    return s.replace(
      /(^|[^\w`])@([A-Za-z0-9](?:[A-Za-z0-9-]{0,37}[A-Za-z0-9])?(?:\/[A-Za-z0-9._-]+)?)/g,
      (_m, p1, p2) => `${p1}\`@${p2}\``
    );
  }

  /**
   * Removes XML comments from content
   * @param {string} s - The string to process
   * @returns {string} The string with XML comments removed
   */
  function removeXmlComments(s) {
    // Remove <!-- comment --> and malformed <!--! comment --!>
    return s.replace(/<!--[\s\S]*?-->/g, "").replace(/<!--[\s\S]*?--!>/g, "");
  }

  /**
   * Neutralizes bot trigger phrases by wrapping them in backticks
   * @param {string} s - The string to process
   * @returns {string} The string with neutralized bot triggers
   */
  function neutralizeBotTriggers(s) {
    // Neutralize common bot trigger phrases like "fixes #123", "closes #asdfs", etc.
    return s.replace(/\b(fixes?|closes?|resolves?|fix|close|resolve)\s+#(\w+)/gi, (match, action, ref) => `\`${action} #${ref}\``);
  }
}

// === End of ./lib/sanitize.cjs ===

async function main() {
  const fs = require("fs");
  const outputFile = process.env.GH_AW_SAFE_OUTPUTS;
  if (!outputFile) {
    core.info("GH_AW_SAFE_OUTPUTS not set, no output to collect");
    core.setOutput("output", "");
    return;
  }

  if (!fs.existsSync(outputFile)) {
    core.info(`Output file does not exist: ${outputFile}`);
    core.setOutput("output", "");
    return;
  }

  const outputContent = fs.readFileSync(outputFile, "utf8");
  if (outputContent.trim() === "") {
    core.info("Output file is empty");
    core.setOutput("output", "");
  } else {
    const sanitizedContent = sanitizeContent(outputContent);
    core.info(`Collected agentic output (sanitized): ${sanitizedContent.substring(0, 200)}${sanitizedContent.length > 200 ? "..." : ""}`);
    core.setOutput("output", sanitizedContent);
  }
}

await main();
