// @ts-check
/**
 * Full sanitization utilities with mention filtering support
 * This module provides the complete sanitization with selective mention filtering.
 * For incoming text that doesn't need mention filtering, use sanitize_incoming_text.cjs instead.
 */

const {
  sanitizeContentCore,
  getRedactedDomains,
  clearRedactedDomains,
  writeRedactedDomainsLog,
  buildAllowedDomains,
  buildAllowedGitHubReferences,
  getCurrentRepoSlug,
  sanitizeUrlProtocols,
  sanitizeUrlDomains,
  neutralizeCommands,
  neutralizeGitHubReferences,
  removeXmlComments,
  neutralizeMarkdownLinkTitles,
  convertXmlTags,
  applyToNonCodeRegions,
  neutralizeBotTriggers,
  neutralizeTemplateDelimiters,
  applyTruncation,
  hardenUnicodeText,
} = require("./sanitize_content_core.cjs");

const { balanceCodeRegions } = require("./markdown_code_region_balancer.cjs");

/**
 * @typedef {Object} SanitizeOptions
 * @property {number} [maxLength] - Maximum length of content (default: 524288)
 * @property {string[]} [allowedAliases] - List of aliases (@mentions) that should not be neutralized
 * @property {number} [maxBotMentions] - Maximum bot trigger references before filtering (default: 10)
 */

/**
 * Sanitizes content for safe output in GitHub Actions with optional mention filtering
 * @param {string} content - The content to sanitize
 * @param {number | SanitizeOptions} [maxLengthOrOptions] - Maximum length of content (default: 524288) or options object
 * @returns {string} The sanitized content
 */
function sanitizeContent(content, maxLengthOrOptions) {
  // Handle both old signature (maxLength) and new signature (options object)
  /** @type {number | undefined} */
  let maxLength;
  /** @type {string[]} */
  let allowedAliasesLowercase = [];
  /** @type {number | undefined} */
  let maxBotMentions;

  if (typeof maxLengthOrOptions === "number") {
    maxLength = maxLengthOrOptions;
  } else if (maxLengthOrOptions && typeof maxLengthOrOptions === "object") {
    maxLength = maxLengthOrOptions.maxLength;
    // Pre-process allowed aliases to lowercase for efficient comparison
    allowedAliasesLowercase = (maxLengthOrOptions.allowedAliases || []).map(alias => alias.toLowerCase());
    maxBotMentions = maxLengthOrOptions.maxBotMentions;
  }

  // If no allowed aliases specified, use core sanitization (which neutralizes all mentions)
  if (allowedAliasesLowercase.length === 0) {
    return sanitizeContentCore(content, maxLength, maxBotMentions);
  }

  // If allowed aliases are specified, we need custom mention filtering
  // We'll apply the same sanitization pipeline but with selective mention filtering

  if (!content || typeof content !== "string") {
    return "";
  }

  // Build list of allowed domains (shared with core)
  const allowedDomains = buildAllowedDomains();

  // Build list of allowed GitHub references from environment
  const allowedGitHubRefs = buildAllowedGitHubReferences();

  let sanitized = content;

  // Apply Unicode hardening first to normalize text representation
  sanitized = hardenUnicodeText(sanitized);

  // Remove ANSI escape sequences and control characters early
  sanitized = sanitized.replace(/\x1b\[[0-9;]*[mGKH]/g, "");
  sanitized = sanitized.replace(/[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]/g, "");

  // Neutralize commands at the start of text
  sanitized = neutralizeCommands(sanitized);

  // Remove XML comments before mention neutralization to prevent bypass: if removeXmlComments
  // ran after neutralizeMentions, a comment like <!-- @user payload --> would first become
  // <!-- `@user` payload --> and applyFnOutsideInlineCode would split at the backtick boundary,
  // preventing the full <!--...--> pattern from being matched.
  sanitized = applyToNonCodeRegions(sanitized, removeXmlComments);

  // Neutralize markdown link titles as a hidden/steganographic injection channel analogous to
  // HTML comments: inline-link titles are made visible in link text, while reference-style
  // titles are stripped. Quoted title text ([text](url "TITLE") and [ref]: url "TITLE") is
  // invisible in GitHub's rendered markdown (shown only as hover-tooltips) but reaches the AI
  // model verbatim. Must run before mention neutralization for the same ordering reason as
  // removeXmlComments.
  sanitized = applyToNonCodeRegions(sanitized, neutralizeMarkdownLinkTitles);

  // Neutralize @mentions with selective filtering (custom logic for allowed aliases)
  sanitized = neutralizeMentions(sanitized, allowedAliasesLowercase);

  // Convert XML tags – skip code blocks and inline code
  sanitized = applyToNonCodeRegions(sanitized, convertXmlTags);

  // URI filtering (shared with core)
  sanitized = sanitizeUrlProtocols(sanitized);
  sanitized = sanitizeUrlDomains(sanitized, allowedDomains);

  // Apply truncation limits (shared with core)
  sanitized = applyTruncation(sanitized, maxLength);

  // Neutralize GitHub references if restrictions are configured
  sanitized = neutralizeGitHubReferences(sanitized, allowedGitHubRefs);

  // Neutralize bot triggers
  sanitized = neutralizeBotTriggers(sanitized, maxBotMentions);

  // Neutralize template syntax delimiters (defense-in-depth)
  // This prevents potential issues if content is processed by downstream template engines
  sanitized = neutralizeTemplateDelimiters(sanitized);

  // Balance markdown code regions to fix improperly nested fences
  // This repairs markdown where AI models generate nested code blocks at the same indentation
  sanitized = balanceCodeRegions(sanitized);

  return sanitized.trim();

  /**
   * Neutralize @mentions with selective filtering
   * @param {string} s - The string to process
   * @param {string[]} allowedLowercase - List of allowed aliases (lowercase)
   * @returns {string} Processed string
   */
  function neutralizeMentions(s, allowedLowercase) {
    return s.replace(/(^|[^\w`])@([A-Za-z0-9](?:[A-Za-z0-9_-]{0,37}[A-Za-z0-9])?(?:\/[A-Za-z0-9._-]+)?)/g, (_m, p1, p2) => {
      // Check if this mention is in the allowed aliases list (case-insensitive)
      const isAllowed = allowedLowercase.includes(p2.toLowerCase());
      if (isAllowed) {
        return `${p1}@${p2}`; // Keep the original mention
      }
      // Log when a mention is escaped
      if (typeof core !== "undefined" && core.info) {
        core.info(`Escaped mention: @${p2} (not in allowed list)`);
      }
      return `${p1}\`@${p2}\``; // Neutralize the mention
    });
  }
}

module.exports = { sanitizeContent, getRedactedDomains, clearRedactedDomains, writeRedactedDomainsLog };
