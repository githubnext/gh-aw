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
  extractDomainsFromUrl,
  addRedactedDomain,
} = require("./sanitize_content_core.cjs");

/**
 * @typedef {Object} SanitizeOptions
 * @property {number} [maxLength] - Maximum length of content (default: 524288)
 * @property {string[]} [allowedAliases] - List of aliases (@mentions) that should not be neutralized
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

  if (typeof maxLengthOrOptions === "number") {
    maxLength = maxLengthOrOptions;
  } else if (maxLengthOrOptions && typeof maxLengthOrOptions === "object") {
    maxLength = maxLengthOrOptions.maxLength;
    // Pre-process allowed aliases to lowercase for efficient comparison
    allowedAliasesLowercase = (maxLengthOrOptions.allowedAliases || []).map(alias => alias.toLowerCase());
  }

  // If no allowed aliases specified, use core sanitization (which neutralizes all mentions)
  if (allowedAliasesLowercase.length === 0) {
    return sanitizeContentCore(content, maxLength);
  }

  // If allowed aliases are specified, we need custom mention filtering
  // We'll do most of the sanitization with the core, then apply selective mention filtering

  if (!content || typeof content !== "string") {
    return "";
  }

  // Read allowed domains from environment variable
  const allowedDomainsEnv = process.env.GH_AW_ALLOWED_DOMAINS;
  const defaultAllowedDomains = ["github.com", "github.io", "githubusercontent.com", "githubassets.com", "github.dev", "codespaces.new"];

  let allowedDomains = allowedDomainsEnv
    ? allowedDomainsEnv
        .split(",")
        .map(d => d.trim())
        .filter(d => d)
    : defaultAllowedDomains;

  // Extract and add GitHub domains from GitHub context URLs
  const githubServerUrl = process.env.GITHUB_SERVER_URL;
  const githubApiUrl = process.env.GITHUB_API_URL;

  if (githubServerUrl) {
    const serverDomains = extractDomainsFromUrl(githubServerUrl);
    allowedDomains = allowedDomains.concat(serverDomains);
  }

  if (githubApiUrl) {
    const apiDomains = extractDomainsFromUrl(githubApiUrl);
    allowedDomains = allowedDomains.concat(apiDomains);
  }

  // Remove duplicates
  allowedDomains = [...new Set(allowedDomains)];

  let sanitized = content;

  // Neutralize commands at the start of text
  sanitized = neutralizeCommands(sanitized);

  // Neutralize @mentions with selective filtering
  sanitized = neutralizeMentions(sanitized, allowedAliasesLowercase);

  // Remove XML comments
  sanitized = removeXmlComments(sanitized);

  // Convert XML tags
  sanitized = convertXmlTags(sanitized);

  // Remove ANSI escape sequences
  sanitized = sanitized.replace(/\x1b\[[0-9;]*[mGKH]/g, "");

  // Remove control characters
  sanitized = sanitized.replace(/[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]/g, "");

  // URI filtering
  sanitized = sanitizeUrlProtocols(sanitized);
  sanitized = sanitizeUrlDomains(sanitized, allowedDomains);

  // Truncation
  const lines = sanitized.split("\n");
  const maxLines = 65000;
  maxLength = maxLength || 524288;

  if (lines.length > maxLines) {
    const truncationMsg = "\n[Content truncated due to line count]";
    const truncatedLines = lines.slice(0, maxLines).join("\n") + truncationMsg;

    if (truncatedLines.length > maxLength) {
      sanitized = truncatedLines.substring(0, maxLength - truncationMsg.length) + truncationMsg;
    } else {
      sanitized = truncatedLines;
    }
  } else if (sanitized.length > maxLength) {
    sanitized = sanitized.substring(0, maxLength) + "\n[Content truncated due to length]";
  }

  // Neutralize bot triggers
  sanitized = neutralizeBotTriggers(sanitized);

  return sanitized.trim();

  /**
   * Sanitize URL domains
   * @param {string} s - The string to process
   * @param {string[]} allowed - Allowed domains
   * @returns {string} Sanitized string
   */
  function sanitizeUrlDomains(s, allowed) {
    const httpsUrlRegex = /https:\/\/([\w.-]+(?::\d+)?)(\/[^\s]*)?/gi;

    const result = s.replace(httpsUrlRegex, (match, hostnameWithPort, pathPart) => {
      const hostname = hostnameWithPort.split(":")[0].toLowerCase();
      pathPart = pathPart || "";

      const isAllowed = allowed.some(allowedDomain => {
        const normalizedAllowed = allowedDomain.toLowerCase();

        if (hostname === normalizedAllowed) {
          return true;
        }

        if (normalizedAllowed.startsWith("*.")) {
          const baseDomain = normalizedAllowed.substring(2);
          return hostname.endsWith("." + baseDomain) || hostname === baseDomain;
        }

        return hostname.endsWith("." + normalizedAllowed);
      });

      if (isAllowed) {
        return match;
      } else {
        const truncated = hostname.length > 12 ? hostname.substring(0, 12) + "..." : hostname;
        if (typeof core !== "undefined" && core.info) {
          core.info(`Redacted URL: ${truncated}`);
        }
        if (typeof core !== "undefined" && core.debug) {
          core.debug(`Redacted URL (full): ${match}`);
        }
        addRedactedDomain(hostname);
        return "(redacted)";
      }
    });

    return result;
  }

  /**
   * Sanitize URL protocols
   * @param {string} s - The string to process
   * @returns {string} Sanitized string
   */
  function sanitizeUrlProtocols(s) {
    return s.replace(
      /\b((?:http|ftp|file|ssh|git):\/\/([\w.-]+)(?:[^\s]*)|(?:data|javascript|vbscript|about|mailto|tel):[^\s]+)/gi,
      (match, _fullMatch, domain) => {
        // Extract domain for http/ftp/file/ssh/git protocols
        if (domain) {
          const domainLower = domain.toLowerCase();
          const truncated = domainLower.length > 12 ? domainLower.substring(0, 12) + "..." : domainLower;
          if (typeof core !== "undefined" && core.info) {
            core.info(`Redacted URL: ${truncated}`);
          }
          if (typeof core !== "undefined" && core.debug) {
            core.debug(`Redacted URL (full): ${match}`);
          }
          addRedactedDomain(domainLower);
        } else {
          // For other protocols (data:, javascript:, etc.), track the protocol itself
          const protocolMatch = match.match(/^([^:]+):/);
          if (protocolMatch) {
            const protocol = protocolMatch[1] + ":";
            if (typeof core !== "undefined" && core.info) {
              core.info(`Redacted URL: ${protocol}`);
            }
            if (typeof core !== "undefined" && core.debug) {
              core.debug(`Redacted URL (full): ${match}`);
            }
            addRedactedDomain(protocol);
          }
        }
        return "(redacted)";
      }
    );
  }

  /**
   * Neutralize commands
   * @param {string} s - The string to process
   * @returns {string} Processed string
   */
  function neutralizeCommands(s) {
    const commandName = process.env.GH_AW_COMMAND;
    if (!commandName) {
      return s;
    }

    const escapedCommand = commandName.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
    return s.replace(new RegExp(`^(\\s*)/(${escapedCommand})\\b`, "i"), "$1`/$2`");
  }

  /**
   * Neutralize @mentions with selective filtering
   * @param {string} s - The string to process
   * @param {string[]} allowedLowercase - List of allowed aliases (lowercase)
   * @returns {string} Processed string
   */
  function neutralizeMentions(s, allowedLowercase) {
    return s.replace(/(^|[^\w`])@([A-Za-z0-9](?:[A-Za-z0-9-]{0,37}[A-Za-z0-9])?(?:\/[A-Za-z0-9._-]+)?)/g, (_m, p1, p2) => {
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

  /**
   * Remove XML comments
   * @param {string} s - The string to process
   * @returns {string} Processed string
   */
  function removeXmlComments(s) {
    return s.replace(/<!--[\s\S]*?-->/g, "").replace(/<!--[\s\S]*?--!>/g, "");
  }

  /**
   * Convert XML tags
   * @param {string} s - The string to process
   * @returns {string} Processed string
   */
  function convertXmlTags(s) {
    const allowedTags = [
      "b",
      "blockquote",
      "br",
      "code",
      "details",
      "em",
      "h1",
      "h2",
      "h3",
      "h4",
      "h5",
      "h6",
      "hr",
      "i",
      "li",
      "ol",
      "p",
      "pre",
      "strong",
      "sub",
      "summary",
      "sup",
      "table",
      "tbody",
      "td",
      "th",
      "thead",
      "tr",
      "ul",
    ];

    s = s.replace(/<!\[CDATA\[([\s\S]*?)\]\]>/g, (match, content) => {
      const convertedContent = content.replace(/<(\/?[A-Za-z][A-Za-z0-9]*(?:[^>]*?))>/g, "($1)");
      return `(![CDATA[${convertedContent}]])`;
    });

    return s.replace(/<(\/?[A-Za-z!][^>]*?)>/g, (match, tagContent) => {
      const tagNameMatch = tagContent.match(/^\/?\s*([A-Za-z][A-Za-z0-9]*)/);
      if (tagNameMatch) {
        const tagName = tagNameMatch[1].toLowerCase();
        if (allowedTags.includes(tagName)) {
          return match;
        }
      }
      return `(${tagContent})`;
    });
  }

  /**
   * Neutralize bot triggers
   * @param {string} s - The string to process
   * @returns {string} Processed string
   */
  function neutralizeBotTriggers(s) {
    return s.replace(/\b(fixes?|closes?|resolves?|fix|close|resolve)\s+#(\w+)/gi, (match, action, ref) => `\`${action} #${ref}\``);
  }
}

module.exports = { sanitizeContent, getRedactedDomains, clearRedactedDomains, writeRedactedDomainsLog };
