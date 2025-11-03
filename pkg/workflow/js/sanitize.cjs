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

  // Neutralize commands at the start of text (e.g., /bot-name)
  sanitized = neutralizeCommands(sanitized);

  // Neutralize @mentions to prevent unintended notifications
  sanitized = neutralizeMentions(sanitized);

  // Remove XML comments first
  sanitized = removeXmlComments(sanitized);

  // Convert XML tags to parentheses format to prevent injection
  sanitized = convertXmlTags(sanitized);

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
    // First pass: match all HTTPS URLs and process them
    // We need to handle URLs that might contain other URLs in query parameters
    s = s.replace(/\bhttps:\/\/([^\s\])}'"<>&\x00-\x1f,;]+)/gi, (match, rest) => {
      // Extract the hostname part (before first slash, colon, or other delimiter)
      const hostname = rest.split(/[\/:\?#]/)[0].toLowerCase();

      // Check if this domain or any parent domain is in the allowlist
      const isAllowed = allowedDomains.some(allowedDomain => {
        const normalizedAllowed = allowedDomain.toLowerCase();
        return hostname === normalizedAllowed || hostname.endsWith("." + normalizedAllowed);
      });

      if (isAllowed) {
        return match; // Keep allowed URLs as-is
      }

      // Log the redaction
      if (typeof core !== "undefined") {
        const domain = hostname;
        const truncated = domain.length > 12 ? domain.substring(0, 12) + "..." : domain;
        if (core.info) {
          core.info(`Redacted URL: ${truncated}`);
        }
        if (core.debug) {
          core.debug(`Redacted URL (full): ${match}`);
        }
      }

      // For disallowed URLs, check if there are any allowed URLs in the query/fragment
      // and preserve those while redacting the main URL
      const urlParts = match.split(/([?&#])/);
      let result = "(redacted)"; // Redact the main domain

      // Process query/fragment parts to preserve any allowed URLs within them
      for (let i = 1; i < urlParts.length; i++) {
        if (urlParts[i].match(/^[?&#]$/)) {
          result += urlParts[i]; // Keep separators
        } else {
          // Recursively process this part to preserve any allowed URLs
          result += sanitizeUrlDomains(urlParts[i]);
        }
      }

      return result;
    });

    return s;
  }

  /**
   * Remove unknown protocols except https
   * @param {string} s - The string to process
   * @returns {string} The string with non-https protocols redacted
   */
  function sanitizeUrlProtocols(s) {
    // Match protocol patterns but avoid command-line flags, file paths, and namespaces
    // Protocol patterns typically have :// or are well-known schemes followed by :
    // Use negative lookbehind to exclude patterns preceded by - (command flags)
    // Match only patterns that look like actual protocols
    return s.replace(/(?<![-\/\w])([A-Za-z][A-Za-z0-9+.-]*):(?:\/\/|(?=[^\s:]))[^\s\])}'"<>&\x00-\x1f]+/g, (match, protocol) => {
      // Allow https (case insensitive), redact everything else
      // But only if it looks like a URL (has :// or is followed by non-colon content)
      if (protocol.toLowerCase() === "https") {
        return match;
      }

      // Allow if it looks like a file path or namespace (::)
      if (match.includes("::")) {
        return match;
      }

      // Redact if it has :// (definite protocol)
      if (match.includes("://")) {
        // Log the redaction
        if (typeof core !== "undefined") {
          // Extract domain from URL
          const domainMatch = match.match(/^[^:]+:\/\/([^\/\s?#]+)/);
          const domain = domainMatch ? domainMatch[1] : match;
          const truncated = domain.length > 12 ? domain.substring(0, 12) + "..." : domain;
          if (core.info) {
            core.info(`Redacted URL: ${truncated}`);
          }
          if (core.debug) {
            core.debug(`Redacted URL (full): ${match}`);
          }
        }
        return "(redacted)";
      }

      // Redact well-known dangerous protocols like javascript:, data:, etc.
      const dangerousProtocols = ["javascript", "data", "vbscript", "file", "about", "mailto", "tel", "ssh", "ftp"];
      if (dangerousProtocols.includes(protocol.toLowerCase())) {
        // Log the redaction
        if (typeof core !== "undefined") {
          // For dangerous protocols without ://, show protocol and beginning of content
          const truncated = match.length > 12 ? match.substring(0, 12) + "..." : match;
          if (core.info) {
            core.info(`Redacted URL: ${truncated}`);
          }
          if (core.debug) {
            core.debug(`Redacted URL (full): ${match}`);
          }
        }
        return "(redacted)";
      }

      // Otherwise preserve (could be file:path, namespace:thing, etc.)
      return match;
    });
  }

  /**
   * Neutralizes commands at the start of text by wrapping them in backticks
   * @param {string} s - The string to process
   * @returns {string} The string with neutralized commands
   */
  function neutralizeCommands(s) {
    const commandName = process.env.GH_AW_COMMAND;
    if (!commandName) {
      return s;
    }

    // Escape special regex characters in command name
    const escapedCommand = commandName.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");

    // Neutralize /command at the start of text (with optional leading whitespace)
    // Only match at the start of the string or after leading whitespace
    return s.replace(new RegExp(`^(\\s*)/(${escapedCommand})\\b`, "i"), "$1`/$2`");
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
   * Converts XML/HTML tags to parentheses format to prevent injection
   * @param {string} s - The string to process
   * @returns {string} The string with XML tags converted to parentheses
   */
  function convertXmlTags(s) {
    // Allow safe HTML tags: details, summary, code, em, b
    const allowedTags = ["details", "summary", "code", "em", "b"];

    // First, process CDATA sections specially - convert tags inside them and the CDATA markers
    s = s.replace(/<!\[CDATA\[([\s\S]*?)\]\]>/g, (match, content) => {
      // Convert tags inside CDATA content
      const convertedContent = content.replace(/<(\/?[A-Za-z][A-Za-z0-9]*(?:[^>]*?))>/g, "($1)");
      // Return with CDATA markers also converted to parentheses
      return `(![CDATA[${convertedContent}]])`;
    });

    // Convert opening tags: <tag> or <tag attr="value"> to (tag) or (tag attr="value")
    // Convert closing tags: </tag> to (/tag)
    // Convert self-closing tags: <tag/> or <tag /> to (tag/) or (tag /)
    // But preserve allowed safe tags
    return s.replace(/<(\/?[A-Za-z!][^>]*?)>/g, (match, tagContent) => {
      // Extract tag name from the content (handle closing tags and attributes)
      const tagNameMatch = tagContent.match(/^\/?\s*([A-Za-z][A-Za-z0-9]*)/);
      if (tagNameMatch) {
        const tagName = tagNameMatch[1].toLowerCase();
        if (allowedTags.includes(tagName)) {
          return match; // Preserve allowed tags
        }
      }
      return `(${tagContent})`; // Convert other tags to parentheses
    });
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

module.exports = { sanitizeContent };
