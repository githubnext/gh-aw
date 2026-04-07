// @ts-check
/**
 * Core sanitization utilities without mention filtering
 * This module provides the base sanitization functions that don't require
 * mention resolution or filtering. It's designed to be imported by both
 * sanitize_content.cjs (full version) and sanitize_incoming_text.cjs (minimal version).
 */

const { isRepoAllowed } = require("./repo_helpers.cjs");

/**
 * Module-level set to collect redacted URL domains across sanitization calls.
 * @type {string[]}
 */
const redactedDomains = [];

/**
 * Gets the list of redacted URL domains collected during sanitization.
 * @returns {string[]} Array of redacted domain strings
 */
function getRedactedDomains() {
  return [...redactedDomains];
}

/**
 * Adds a domain to the redacted domains list
 * @param {string} domain - Domain to add
 */
function addRedactedDomain(domain) {
  redactedDomains.push(domain);
}

/**
 * Clears the list of redacted URL domains.
 * Useful for testing or resetting state between operations.
 */
function clearRedactedDomains() {
  redactedDomains.length = 0;
}

/**
 * Writes the collected redacted URL domains to a log file.
 * Only creates the file if there are redacted domains.
 * @param {string} [filePath] - Path to write the log file. Defaults to /tmp/gh-aw/redacted-urls.log
 * @returns {string|null} The file path if written, null if no domains to write
 */
function writeRedactedDomainsLog(filePath) {
  if (redactedDomains.length === 0) {
    return null;
  }

  const fs = require("fs");
  const path = require("path");
  const targetPath = filePath || "/tmp/gh-aw/redacted-urls.log";

  // Ensure directory exists
  const dir = path.dirname(targetPath);
  if (!fs.existsSync(dir)) {
    fs.mkdirSync(dir, { recursive: true });
  }

  // Write domains to file, one per line
  fs.writeFileSync(targetPath, redactedDomains.join("\n") + "\n");

  return targetPath;
}

/**
 * Extract domains from a URL and return an array of domain variations
 * @param {string} url - The URL to extract domains from
 * @returns {string[]} Array of domain variations
 */
function extractDomainsFromUrl(url) {
  if (!url || typeof url !== "string") {
    return [];
  }

  try {
    // Parse the URL
    const urlObj = new URL(url);
    const hostname = urlObj.hostname.toLowerCase();

    // Return both the exact hostname and common variations
    const domains = [hostname];

    // For github.com, add api and raw content domain variations
    if (hostname === "github.com") {
      domains.push("api.github.com");
      domains.push("raw.githubusercontent.com");
      domains.push("*.githubusercontent.com");
    }
    // For custom GitHub Enterprise domains, add api. prefix and raw content variations
    else if (!hostname.startsWith("api.")) {
      domains.push("api." + hostname);
      // For GitHub Enterprise, raw content is typically served from raw.hostname
      domains.push("raw." + hostname);
    }

    return domains;
  } catch (e) {
    // Invalid URL, return empty array
    return [];
  }
}

/**
 * Build the list of allowed domains from environment variables and GitHub context
 * @returns {string[]} Array of allowed domains
 */
function buildAllowedDomains() {
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
  return [...new Set(allowedDomains)];
}

/**
 * Sanitize a domain name to only include alphanumeric characters and dots,
 * keeping up to 3 domain parts (e.g., sub.example.com).
 * If more than 3 parts exist, truncates with "..."
 * @param {string} domain - The domain to sanitize
 * @returns {string} The sanitized domain
 */
function sanitizeDomainName(domain) {
  if (!domain || typeof domain !== "string") {
    return "";
  }

  // Split domain into parts
  const parts = domain.split(".");

  // Keep only alphanumeric characters in each part
  const sanitizedParts = parts.map(part => part.replace(/[^a-zA-Z0-9]/g, ""));

  // Filter out empty parts
  const nonEmptyParts = sanitizedParts.filter(part => part.length > 0);

  // Join the parts back together
  const joined = nonEmptyParts.join(".");

  // If the domain is longer than 48 characters, truncate to show first 24 and last 24
  if (joined.length > 48) {
    const first24 = joined.substring(0, 24);
    const last24 = joined.substring(joined.length - 24);
    return first24 + "…" + last24;
  }

  return joined;
}

/**
 * Sanitize URL protocols - replace non-https with <sanitized-domain>/redacted
 * @param {string} s - The string to process
 * @returns {string} The string with non-https protocols redacted
 */
function sanitizeUrlProtocols(s) {
  // Normalize percent-encoded colons before applying the protocol blocklist.
  // This prevents bypasses via javascript%3Aalert(1) (single-encoded),
  // javascript%253Aalert(1) (double-encoded), or deeper nesting.
  // Strategy: iteratively decode %25 -> % (up to 4 passes, which handles
  // encodings up to 5 levels deep) until stable, then decode %3A -> :
  // so the blocklist regex always sees literal colons.
  let normalized = s;
  // Iteratively decode %25XX (percent-encoded percent signs) one level at a
  // time. 4 passes handles up to 5 encoding levels, which is far beyond the
  // 1-2 levels a browser uses during URL parsing. Any input requiring more
  // passes almost certainly has no browser-decoded equivalent that would
  // produce a dangerous protocol. The early-exit break keeps this O(n) for
  // typical non-malicious input.
  for (let i = 0; i < 4; i++) {
    // Replace %25XX (percent-encoded percent sign) with %XX one level at a time.
    const next = normalized.replace(/%25([0-9A-Fa-f]{2})/gi, "%$1");
    if (next === normalized) break;
    normalized = next;
  }
  normalized = normalized.replace(/%3[Aa]/gi, ":"); // decode %3A -> :

  // Match common non-https protocols
  // This regex matches: protocol://domain or protocol:path or incomplete protocol://
  // Examples: http://, ftp://, file://, data:, javascript:, mailto:, tel:, ssh://, git://
  // The regex also matches incomplete protocols like "http://" or "ftp://" without a domain
  // Note: No word boundary check to catch protocols even when preceded by word characters
  return normalized.replace(/((?:http|ftp|file|ssh|git):\/\/([\w.-]*)(?:[^\s]*)|(?:data|javascript|vbscript|about|mailto|tel):[^\s]+)/gi, (match, _fullMatch, domain) => {
    // Extract domain for http/ftp/file/ssh/git protocols
    if (domain) {
      const domainLower = domain.toLowerCase();
      const sanitized = sanitizeDomainName(domainLower);
      const truncated = domainLower.length > 12 ? domainLower.substring(0, 12) + "..." : domainLower;
      if (typeof core !== "undefined" && core.info) {
        core.info(`Redacted URL: ${truncated}`);
      }
      if (typeof core !== "undefined" && core.debug) {
        core.debug(`Redacted URL (full): ${match}`);
      }
      addRedactedDomain(domainLower);
      // Return sanitized domain format
      return sanitized ? `(${sanitized}/redacted)` : "(redacted)";
    } else {
      // For other protocols (data:, javascript:, etc.), track the protocol itself
      const protocolMatch = match.match(/^([^:]+):/);
      if (protocolMatch) {
        const protocol = protocolMatch[1] + ":";
        // Truncate the matched URL for logging (keep first 12 chars + "...")
        const truncated = match.length > 12 ? match.substring(0, 12) + "..." : match;
        if (typeof core !== "undefined" && core.info) {
          core.info(`Redacted URL: ${truncated}`);
        }
        if (typeof core !== "undefined" && core.debug) {
          core.debug(`Redacted URL (full): ${match}`);
        }
        addRedactedDomain(protocol);
      }
      return "(redacted)";
    }
  });
}

/**
 * Remove unknown domains
 * @param {string} s - The string to process
 * @param {string[]} allowed - List of allowed domains
 * @returns {string} The string with unknown domains redacted
 */
function sanitizeUrlDomains(s, allowed) {
  // Match HTTPS URLs with optional port and path
  // This regex is designed to:
  // 1. Match https:// URIs with explicit protocol
  // 2. Capture the hostname/domain
  // 3. Allow optional port (:8080)
  // 4. Allow optional path and query string (but not trailing commas/periods)
  // 5. Stop before another https:// URL in query params (using negative lookahead)
  const httpsUrlRegex = /https:\/\/([\w.-]+(?::\d+)?)(\/(?:(?!https:\/\/)[^\s,])*)?/gi;

  /**
   * Shared domain-allowlist check and redaction logic.
   * @param {string} match - The full matched URL string
   * @param {string} hostnameWithPort - The hostname (and optional port) portion
   * @returns {string} The original match if allowed, or a redacted replacement
   */
  function applyDomainFilter(match, hostnameWithPort) {
    // Extract just the hostname (remove port if present)
    const hostname = hostnameWithPort.split(":")[0].toLowerCase();

    // Check if domain is in the allowed list or is a subdomain of an allowed domain
    const isAllowed = allowed.some(allowedDomain => {
      const normalizedAllowed = allowedDomain.toLowerCase();

      // Exact match
      if (hostname === normalizedAllowed) {
        return true;
      }

      // Wildcard match (*.example.com matches subdomain.example.com)
      if (normalizedAllowed.startsWith("*.")) {
        const baseDomain = normalizedAllowed.substring(2); // Remove *.
        return hostname.endsWith("." + baseDomain) || hostname === baseDomain;
      }

      // Subdomain match (example.com matches subdomain.example.com)
      return hostname.endsWith("." + normalizedAllowed);
    });

    if (isAllowed) {
      return match; // Keep the full URL as-is
    } else {
      // Redact the domain but preserve the protocol and structure for debugging
      const sanitized = sanitizeDomainName(hostname);
      const truncated = hostname.length > 12 ? hostname.substring(0, 12) + "..." : hostname;
      if (typeof core !== "undefined" && core.info) {
        core.info(`Redacted URL: ${truncated}`);
      }
      if (typeof core !== "undefined" && core.debug) {
        core.debug(`Redacted URL (full): ${match}`);
      }
      addRedactedDomain(hostname);
      // Return sanitized domain format
      return sanitized ? `(${sanitized}/redacted)` : "(redacted)";
    }
  }

  // First pass: handle explicit https:// URLs
  s = s.replace(httpsUrlRegex, (match, hostnameWithPort) => applyDomainFilter(match, hostnameWithPort));

  // Second pass: handle protocol-relative URLs (//hostname/path).
  // Browsers on HTTPS pages resolve these to https://, so they must be subject
  // to the same domain allowlist check as explicit https:// URLs.
  // We only treat // as a protocol-relative URL when it appears at the start of
  // the string or immediately after a clear delimiter (whitespace, brackets,
  // or quotes). This avoids matching // segments inside the path of an allowed
  // https:// URL, such as "https://github.com//issues".
  // The path stop-condition (?!\/\/) stops before the next protocol-relative URL
  // (analogous to how the httpsUrlRegex stops before the next https:// URL).
  // Capture groups:
  //   1: prefix (start-of-string or delimiter)
  //   2: full protocol-relative URL (starting with //)
  //   3: hostname (and optional port)
  //   4: optional path
  const protoRelativeUrlRegex = /(^|[\s([{"'])(\/\/([\w.-]+(?::\d+)?)(\/(?:(?!\/\/)[^\s,])*)?)/gi;

  s = s.replace(protoRelativeUrlRegex, (match, prefix, url, hostnameWithPort) => prefix + applyDomainFilter(url, hostnameWithPort));

  return s;
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
 * Neutralizes ALL @mentions by wrapping them in backticks
 * This is the core version without any filtering
 * @param {string} s - The string to process
 * @returns {string} The string with neutralized mentions
 */
function neutralizeAllMentions(s) {
  // Replace @name or @org/team outside code with `@name`
  // No filtering - all mentions are neutralized
  // Changed [^\w`] to [^A-Za-z0-9`] to include underscore as a valid preceding character
  // This prevents bypass patterns like "test_@user" from escaping sanitization
  return s.replace(/(^|[^A-Za-z0-9`])@([A-Za-z0-9](?:[A-Za-z0-9_-]{0,37}[A-Za-z0-9])?(?:\/[A-Za-z0-9._-]+)?)/g, (m, p1, p2) => {
    // Log when a mention is escaped to help debug issues
    if (typeof core !== "undefined" && core.info) {
      core.info(`Escaped mention: @${p2} (not in allowed list)`);
    }
    return `${p1}\`@${p2}\``;
  });
}

/**
 * Returns the character ranges [start, end) of fenced code blocks in markdown content.
 * Fenced code blocks are delimited by lines starting with 3+ backticks or 3+ tildes.
 * The returned ranges span from the first character of the opening fence line through
 * the last character of the closing fence line (inclusive of any trailing newline).
 *
 * @param {string} s - Markdown content to scan
 * @returns {Array<[number, number]>} Array of [start, end) character positions
 */
function getFencedCodeRanges(s) {
  /** @type {Array<[number, number]>} */
  const ranges = [];
  const lines = s.split("\n");
  let pos = 0;
  let inBlock = false;
  let blockStart = -1;
  let fenceChar = "";
  let fenceLen = 0;

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    const trimmed = line.trim();
    // Character position of the end of this line's content (not including the newline separator)
    const lineContentEnd = pos + line.length;
    // Character position after the newline separator (or same as lineContentEnd for the last line)
    const lineEnd = i < lines.length - 1 ? lineContentEnd + 1 : lineContentEnd;

    if (!inBlock) {
      const m = trimmed.match(/^(`{3,}|~{3,})/);
      if (m) {
        inBlock = true;
        blockStart = pos;
        fenceChar = m[1][0];
        fenceLen = m[1].length;
      }
    } else {
      // A closing fence: same character, at least as long, only whitespace after
      const fc = fenceChar === "`" ? "\\`" : "~";
      const closingRegex = new RegExp(`^[${fc}]{${fenceLen},}\\s*$`);
      if (closingRegex.test(trimmed)) {
        ranges.push([blockStart, lineEnd]);
        inBlock = false;
        blockStart = -1;
        fenceChar = "";
        fenceLen = 0;
      }
    }

    pos = lineEnd;
  }

  // Unclosed fence – treat the rest as code (safer fallback)
  if (inBlock && blockStart !== -1) {
    ranges.push([blockStart, s.length]);
  }

  return ranges;
}

/**
 * Applies a transformation function to a text segment while skipping inline code spans
 * (backtick-delimited sequences).  The transformation is applied to each run of
 * non-code text; inline code spans are preserved verbatim.
 *
 * @param {string} text - The text to process (should not contain fenced code blocks)
 * @param {function(string): string} fn - Transformation to apply to non-code portions
 * @returns {string} The processed text
 */
function applyFnOutsideInlineCode(text, fn) {
  if (!text) return fn(text || "");

  const parts = [];
  let i = 0;
  let textStart = 0;

  while (i < text.length) {
    if (text[i] !== "`") {
      i++;
      continue;
    }

    // Count consecutive backticks at the current position
    const btStart = i;
    let btCount = 0;
    while (i < text.length && text[i] === "`") {
      btCount++;
      i++;
    }
    // i is now past the opening backtick sequence

    // Look for the matching closing sequence of exactly btCount backticks
    let closeIdx = -1;
    let j = i;
    while (j < text.length) {
      if (text[j] === "`") {
        let closeCount = 0;
        const jStart = j;
        while (j < text.length && text[j] === "`") {
          closeCount++;
          j++;
        }
        if (closeCount === btCount) {
          closeIdx = jStart;
          break;
        }
        // Different length – keep scanning (j already advanced past these backticks)
      } else {
        j++;
      }
    }

    if (closeIdx !== -1) {
      // Valid inline code span found: apply fn to the text before it, then keep the code span
      if (textStart < btStart) {
        parts.push(fn(text.slice(textStart, btStart)));
      }
      parts.push(text.slice(btStart, closeIdx + btCount));
      textStart = closeIdx + btCount;
      i = textStart;
    }
    // If no matching close was found, the backticks are treated as regular text (i already advanced)
  }

  // Apply fn to any remaining non-code text
  if (textStart < text.length) {
    parts.push(fn(text.slice(textStart)));
  }

  return parts.join("");
}

/**
 * Applies a transformation function only to the non-code regions of markdown content.
 * Skips both fenced code blocks (``` / ~~~ delimited) and inline code spans (backtick
 * delimited) so that the transformation is not applied to code content.
 *
 * Falls back to applying fn to the entire string if any parsing error occurs.
 *
 * @param {string} s - Markdown content to process
 * @param {function(string): string} fn - Transformation to apply outside code regions
 * @returns {string} The content with the transformation applied only outside code regions
 */
function applyToNonCodeRegions(s, fn) {
  if (!s || typeof s !== "string") {
    return s || "";
  }

  try {
    const codeRanges = getFencedCodeRanges(s);

    if (codeRanges.length === 0) {
      // No fenced code blocks – still protect inline code spans
      return applyFnOutsideInlineCode(s, fn);
    }

    const parts = [];
    let pos = 0;

    for (const [start, end] of codeRanges) {
      if (pos < start) {
        // Non-code text before this code block: protect inline code spans
        parts.push(applyFnOutsideInlineCode(s.slice(pos, start), fn));
      }
      // Fenced code block: preserve verbatim
      parts.push(s.slice(start, end));
      pos = end;
    }

    // Non-code text after the last code block
    if (pos < s.length) {
      parts.push(applyFnOutsideInlineCode(s.slice(pos), fn));
    }

    return parts.join("");
  } catch (_e) {
    // Fallback: apply fn to the entire string (conservative – redacts more, never less)
    return fn(s);
  }
}

/**
 * Removes XML comments from content
 * @param {string} s - The string to process
 * @returns {string} The string with XML comments removed
 */
function removeXmlComments(s) {
  // Remove <!-- comment --> and malformed <!--! comment --!>
  // Consolidated into single atomic regex to prevent intermediate state vulnerabilities
  // The pattern <!--[\s\S]*?--!?> matches both <!-- ... --> and <!-- ... --!>
  // Apply repeatedly to handle nested/overlapping patterns that could reintroduce comment markers
  let previous;
  do {
    previous = s;
    s = s.replace(/<!--[\s\S]*?--!?>/g, "");
  } while (s !== previous);
  return s;
}

/**
 * Converts XML/HTML tags to parentheses format to prevent injection
 * @param {string} s - The string to process
 * @returns {string} The string with XML tags converted to parentheses
 */
function convertXmlTags(s) {
  // Allow safe HTML tags supported by GitHub Flavored Markdown:
  // b, blockquote, br, code, details, em, h1–h6, hr, i, li, ol, p, pre, strong, sub, summary, sup, table, tbody, td, th, thead, tr, ul
  // Plus GFM inline tags: abbr, del, ins, kbd, mark, s, span
  const allowedTags = [
    "abbr",
    "b",
    "blockquote",
    "br",
    "code",
    "del",
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
    "ins",
    "kbd",
    "li",
    "mark",
    "ol",
    "p",
    "pre",
    "s",
    "span",
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

  // First, process CDATA sections specially - convert tags inside them and the CDATA markers
  s = s.replace(/<!\[CDATA\[([\s\S]*?)\]\]>/g, (match, content) => {
    // Convert tags inside CDATA content
    const convertedContent = content.replace(/<(\/?[A-Za-z][A-Za-z0-9]*(?:[^>]*?))>/g, "($1)");
    // Return with CDATA markers also converted to parentheses
    return `(![CDATA[${convertedContent}]])`;
  });

  /**
   * Strips dangerous HTML attributes from an allowed tag's content string.
   * Removes on* event handler attributes (e.g. onclick, ontoggle) and style
   * attributes in all quoting forms (double-quoted, single-quoted, unquoted, bare).
   * Safe attributes such as title, class, open, lang, id, etc. are preserved.
   *
   * Note: `\s+` (requiring at least one whitespace before the attribute name) is
   * intentional — HTML attributes are always separated from the tag name and from
   * each other by at least one whitespace character. Using `\s*` would risk false
   * matches inside tag names (e.g. matching "ong" inside "strong").
   *
   * @param {string} tagContent - Tag content without surrounding angle brackets
   * @returns {string} Tag content with dangerous attributes removed
   */
  function stripDangerousAttributes(tagContent) {
    // Match: one-or-more whitespace + (on* | style) + optional =value
    // Value forms: "...", '...', or unquoted (no whitespace / > / quote chars), or bare (no =)
    // The unquoted form excludes >, whitespace, and all quote characters (', ", `) so it
    // cannot consume the closing > of the tag or straddle other attribute values.
    return tagContent.replace(/\s+(?:on\w+|style)(?:\s*=\s*(?:"[^"]*"|'[^']*'|[^\s>"'`]*))?/gi, "");
  }

  // Convert opening tags: <tag> or <tag attr="value"> to (tag) or (tag attr="value")
  // Convert closing tags: </tag> to (/tag)
  // Convert self-closing tags: <tag/> or <tag /> to (tag/) or (tag /)
  // But preserve allowed safe tags (with dangerous attributes stripped)
  return s.replace(/<(\/?[A-Za-z!][^>]*?)>/g, (match, tagContent) => {
    // Extract tag name from the content (handle closing tags and attributes)
    const tagNameMatch = tagContent.match(/^\/?\s*([A-Za-z][A-Za-z0-9]*)/);
    if (tagNameMatch) {
      const tagName = tagNameMatch[1].toLowerCase();
      if (allowedTags.includes(tagName)) {
        // Strip dangerous attributes (on* event handlers and style) before preserving
        const sanitizedContent = stripDangerousAttributes(tagContent);
        return `<${sanitizedContent}>`;
      }
    }
    return `(${tagContent})`; // Convert other tags to parentheses
  });
}

/**
 * Maximum number of bot trigger references allowed before filtering is applied.
 */
const MAX_BOT_TRIGGER_REFERENCES = 10;

/**
 * Neutralizes bot trigger phrases by wrapping them in backticks.
 * The first `maxBotMentions` unquoted trigger references are left unchanged;
 * any occurrences beyond that threshold are wrapped in backticks.
 * Already-quoted entries are never re-quoted.
 * @param {string} s - The string to process
 * @param {number} [maxBotMentions] - Number of occurrences to allow before escaping (default: MAX_BOT_TRIGGER_REFERENCES)
 * @returns {string} The string with excess bot triggers neutralized
 */
function neutralizeBotTriggers(s, maxBotMentions = MAX_BOT_TRIGGER_REFERENCES) {
  // Match unquoted bot trigger phrases like "fixes #123", "closes #asdfs", etc.
  // The negative lookbehind (?<!`) skips already-quoted entries.
  const pattern = /(?<!`)\b(fixes?|closes?|resolves?|fix|close|resolve)\s+#(\w+)/gi;
  const matches = s.match(pattern);
  if (!matches || matches.length <= maxBotMentions) {
    return s;
  }
  let count = 0;
  return s.replace(pattern, (match, action, ref) => {
    count++;
    if (count <= maxBotMentions) {
      return match;
    }
    return `\`${action} #${ref}\``;
  });
}

/**
 * Neutralizes template syntax delimiters to prevent potential template injection
 * if content is processed by downstream template engines.
 *
 * This is a defense-in-depth measure. GitHub's markdown rendering doesn't evaluate
 * template syntax, but this prevents issues if content is later processed by
 * template engines (Jinja2, Liquid, ERB, JavaScript template literals).
 *
 * @param {string} s - The string to process
 * @returns {string} The string with escaped template delimiters
 */
function neutralizeTemplateDelimiters(s) {
  if (!s || typeof s !== "string") {
    return "";
  }

  let result = s;
  let templatesDetected = false;

  // Escape Jinja2/Liquid double curly braces: {{ ... }}
  // Replace {{ with \{\{ to prevent template evaluation
  if (/\{\{/.test(result)) {
    templatesDetected = true;
    if (typeof core !== "undefined" && core.info) {
      core.info("Template syntax detected: Jinja2/Liquid double braces {{");
    }
    result = result.replace(/\{\{/g, "\\{\\{");
  }

  // Escape ERB delimiters: <%= ... %>
  // Replace <%= with \<%= to prevent ERB evaluation
  if (/<%=/.test(result)) {
    templatesDetected = true;
    if (typeof core !== "undefined" && core.info) {
      core.info("Template syntax detected: ERB delimiter <%=");
    }
    result = result.replace(/<%=/g, "\\<%=");
  }

  // Escape JavaScript template literal delimiters: ${ ... }
  // Replace ${ with \$\{ to prevent template literal evaluation
  if (/\$\{/.test(result)) {
    templatesDetected = true;
    if (typeof core !== "undefined" && core.info) {
      core.info("Template syntax detected: JavaScript template literal ${");
    }
    result = result.replace(/\$\{/g, "\\$\\{");
  }

  // Escape Jinja2 comment delimiters: {# ... #}
  // Replace {# with \{\# to prevent Jinja2 comment evaluation
  if (/\{#/.test(result)) {
    templatesDetected = true;
    if (typeof core !== "undefined" && core.info) {
      core.info("Template syntax detected: Jinja2 comment {#");
    }
    result = result.replace(/\{#/g, "\\{\\#");
  }

  // Escape Jekyll raw blocks: {% raw %} and {% endraw %}
  // Replace {% with \{\% to prevent Jekyll directive evaluation
  if (/\{%/.test(result)) {
    templatesDetected = true;
    if (typeof core !== "undefined" && core.info) {
      core.info("Template syntax detected: Jekyll/Liquid directive {%");
    }
    result = result.replace(/\{%/g, "\\{\\%");
  }

  // Log a summary warning if any template patterns were detected
  if (templatesDetected && typeof core !== "undefined" && core.warning) {
    core.warning(
      "Template-like syntax detected and escaped. " +
        "This is a defense-in-depth measure to prevent potential template injection " +
        "if content is processed by downstream template engines. " +
        "GitHub's markdown rendering does not evaluate template syntax."
    );
  }

  return result;
}

/**
 * Builds the list of allowed repositories for GitHub reference filtering
 * Returns null if all references should be allowed (default behavior)
 * Returns empty array if no references should be allowed (escape all)
 * @returns {string[]|null} Array of allowed repository slugs or null if all allowed
 */
function buildAllowedGitHubReferences() {
  const allowedRefsEnv = process.env.GH_AW_ALLOWED_GITHUB_REFS;
  if (allowedRefsEnv === undefined) {
    return null; // All references allowed by default (env var not set)
  }

  if (allowedRefsEnv === "") {
    if (typeof core !== "undefined" && core.info) {
      core.info("GitHub reference filtering: all references will be escaped (GH_AW_ALLOWED_GITHUB_REFS is empty)");
    }
    return []; // Empty array means escape all references
  }

  const refs = allowedRefsEnv
    .split(",")
    .map(ref => ref.trim().toLowerCase())
    .filter(ref => ref);
  if (typeof core !== "undefined" && core.info) {
    core.info(`GitHub reference filtering: allowed repos = ${refs.join(", ")}`);
  }
  return refs;
}

/**
 * Gets the current repository slug from GitHub context
 * @returns {string} Repository slug in "owner/repo" format, or empty string if not available
 */
function getCurrentRepoSlug() {
  // Try to get from GITHUB_REPOSITORY env var
  const repoSlug = process.env.GITHUB_REPOSITORY;
  if (repoSlug) {
    return repoSlug.toLowerCase();
  }
  return "";
}

/**
 * Neutralizes GitHub references (#123 or owner/repo#456) by wrapping them in backticks
 * if they reference repositories not in the allowed list.
 * Supports wildcard patterns (e.g., "myorg/*", "*") via isRepoAllowed().
 * @param {string} s - The string to process
 * @param {string[]|null} allowedRepos - List of allowed repository slugs (lowercase), or null to allow all
 * @returns {string} The string with unauthorized references neutralized
 */
function neutralizeGitHubReferences(s, allowedRepos) {
  // If no restrictions configured (null), allow all references
  if (allowedRepos === null) {
    return s;
  }

  const currentRepo = getCurrentRepoSlug();

  // Expand the special "repo" keyword to the current repo slug and build a Set for isRepoAllowed()
  const allowedSet = new Set(allowedRepos.map(r => (r === "repo" ? currentRepo : r)));

  // Match GitHub references:
  // - #123 (current repo reference)
  // - owner/repo#456 (cross-repo reference)
  // - GH-123 (GitHub shorthand)
  // Must not be inside backticks or code blocks
  return s.replace(/(^|[^\w`])(?:([a-z0-9](?:[a-z0-9-]{0,38}[a-z0-9])?)\/([a-z0-9._-]+))?#(\w+)/gi, (match, prefix, owner, repo, issueNum) => {
    let targetRepo;

    if (owner && repo) {
      // Cross-repo reference: owner/repo#123
      targetRepo = `${owner}/${repo}`.toLowerCase();
    } else {
      // Current repo reference: #123
      targetRepo = currentRepo;
    }

    // Check if this repo is allowed using isRepoAllowed (supports wildcard patterns)
    if (isRepoAllowed(targetRepo, allowedSet)) {
      return match; // Keep the original reference
    } else {
      // Escape the reference
      const refText = owner && repo ? `${owner}/${repo}#${issueNum}` : `#${issueNum}`;

      // Log when a reference is escaped
      if (typeof core !== "undefined" && core.info) {
        core.info(`Escaped GitHub reference: ${refText} (not in allowed list)`);
      }

      return `${prefix}\`${refText}\``;
    }
  });
}

/**
 * Apply truncation limits to content
 * @param {string} content - The content to truncate
 * @param {number} [maxLength] - Maximum length of content (default: 524288)
 * @returns {string} The truncated content
 */
function applyTruncation(content, maxLength) {
  maxLength = maxLength || 524288;
  const lines = content.split("\n");
  const maxLines = 65000;

  // If content has too many lines, truncate by lines (primary limit)
  if (lines.length > maxLines) {
    const truncationMsg = "\n[Content truncated due to line count]";
    const truncatedLines = lines.slice(0, maxLines).join("\n") + truncationMsg;

    // If still too long after line truncation, shorten but keep the line count message
    if (truncatedLines.length > maxLength) {
      return truncatedLines.substring(0, maxLength - truncationMsg.length) + truncationMsg;
    } else {
      return truncatedLines;
    }
  } else if (content.length > maxLength) {
    return content.substring(0, maxLength) + "\n[Content truncated due to length]";
  }

  return content;
}

/**
 * Decodes HTML entities to prevent bypass of @mention detection and to ensure
 * HTML-encoded characters do not persist in sanitized output (e.g. &gt; in titles).
 * Handles named entities (e.g., &commat;, &gt;, &lt;, &amp;), decimal entities (e.g., &#64;),
 * and hex entities (e.g., &#x40;), including double-encoded variants (e.g., &amp;commat;).
 *
 * @param {string} text - Input text that may contain HTML entities
 * @returns {string} Text with HTML entities decoded
 */
function decodeHtmlEntities(text) {
  if (!text || typeof text !== "string") {
    return "";
  }

  let result = text;

  // Decode named entity for @ symbol (including double-encoded variants)
  // &commat; and &amp;commat; → @
  result = result.replace(/&(?:amp;)?commat;/gi, "@");

  // Decode common named HTML entities (including double-encoded variants)
  // These prevent HTML-encoded characters from persisting as literal entities
  // in sanitized output (e.g. a title containing &gt; instead of >).
  // &gt; and &amp;gt; → >
  result = result.replace(/&(?:amp;)?gt;/gi, ">");
  // &lt; and &amp;lt; → < (convertXmlTags will then neutralise any resulting tags)
  result = result.replace(/&(?:amp;)?lt;/gi, "<");
  // &amp; and &amp;amp; → & (decoded after gt/lt so &amp;gt; is already handled above)
  result = result.replace(/&(?:amp;)?amp;/gi, "&");

  // Decode decimal entities (including double-encoded variants)
  // &#64; and &amp;#64; → @
  // &#NNN; and &amp;#NNN; → corresponding character
  result = result.replace(/&(?:amp;)?#(\d+);/g, (match, code) => {
    const codePoint = parseInt(code, 10);
    // Validate code point is in valid Unicode range
    if (codePoint >= 0 && codePoint <= 0x10ffff) {
      return String.fromCodePoint(codePoint);
    }
    // Return original match if invalid
    return match;
  });

  // Decode hex entities (including double-encoded variants)
  // &#x40;, &#X40;, &amp;#x40;, &amp;#X40; → @
  // &#xHHH;, &#XHHH;, &amp;#xHHH;, &amp;#XHHH; → corresponding character
  result = result.replace(/&(?:amp;)?#[xX]([0-9a-fA-F]+);/g, (match, code) => {
    const codePoint = parseInt(code, 16);
    // Validate code point is in valid Unicode range
    if (codePoint >= 0 && codePoint <= 0x10ffff) {
      return String.fromCodePoint(codePoint);
    }
    // Return original match if invalid
    return match;
  });

  return result;
}

/**
 * Performs text hardening to protect against Unicode-based attacks.
 * This applies multiple layers of character normalization and filtering
 * to ensure consistent text processing and prevent visual spoofing.
 *
 * @param {string} text - Input text to harden
 * @returns {string} Hardened text with Unicode security applied
 */
function hardenUnicodeText(text) {
  if (!text || typeof text !== "string") {
    return "";
  }

  let result = text;

  // Step 1: Normalize Unicode to canonical composition (NFC)
  // This ensures consistent character representation across different encodings
  result = result.normalize("NFC");

  // Step 2: Decode HTML entities to prevent @mention bypass
  // This MUST happen early, before any other processing, to ensure entities
  // are converted to their actual characters for proper sanitization
  result = decodeHtmlEntities(result);

  // Step 3: Strip invisible zero-width characters that can hide content
  // These include: zero-width space, zero-width non-joiner, zero-width joiner,
  // left-to-right mark (U+200E), right-to-left mark (U+200F),
  // soft hyphen (U+00AD), combining grapheme joiner (U+034F),
  // word joiner, and byte order mark
  result = result.replace(/[\u00AD\u034F\u200B\u200C\u200D\u200E\u200F\u2060\uFEFF]/g, "");

  // Step 4: Remove bidirectional text override controls
  // These can be used to reverse text direction and create visual spoofs
  result = result.replace(/[\u202A\u202B\u202C\u202D\u202E\u2066\u2067\u2068\u2069]/g, "");

  // Step 5: Convert full-width ASCII characters to standard ASCII
  // Full-width characters (U+FF01-FF5E) can be used to bypass filters
  result = result.replace(/[\uFF01-\uFF5E]/g, char => {
    const code = char.charCodeAt(0);
    // Map full-width to half-width by subtracting offset
    const standardCode = code - 0xfee0;
    return String.fromCharCode(standardCode);
  });

  return result;
}

/**
 * Core sanitization function without mention filtering
 * @param {string} content - The content to sanitize
 * @param {number} [maxLength] - Maximum length of content (default: 524288)
 * @param {number} [maxBotMentions] - Max bot trigger references before filtering (default: MAX_BOT_TRIGGER_REFERENCES)
 * @returns {string} The sanitized content
 */
function sanitizeContentCore(content, maxLength, maxBotMentions) {
  if (!content || typeof content !== "string") {
    return "";
  }

  // Build list of allowed domains from environment and GitHub context
  const allowedDomains = buildAllowedDomains();

  // Build list of allowed GitHub references from environment
  const allowedGitHubRefs = buildAllowedGitHubReferences();

  let sanitized = content;

  // Apply Unicode hardening first to normalize text representation
  // This prevents Unicode-based attacks and ensures consistent processing
  sanitized = hardenUnicodeText(sanitized);

  // Remove ANSI escape sequences and control characters early
  // This must happen before mention neutralization to avoid creating bare mentions
  // when control characters are removed between @ and username
  sanitized = sanitized.replace(/\x1b\[[0-9;]*[mGKH]/g, "");
  // Remove control characters except newlines (\n), tabs (\t), and carriage returns (\r)
  sanitized = sanitized.replace(/[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]/g, "");

  // Neutralize commands at the start of text (e.g., /bot-name)
  sanitized = neutralizeCommands(sanitized);

  // Neutralize ALL @mentions (no filtering in core version)
  sanitized = neutralizeAllMentions(sanitized);

  // Remove XML comments – skip code blocks and inline code to avoid altering code content
  sanitized = applyToNonCodeRegions(sanitized, removeXmlComments);

  // Convert XML tags to parentheses format – skip code blocks and inline code so that
  // type parameters (e.g. VBuffer<float32>) and code containing angle brackets are preserved
  sanitized = applyToNonCodeRegions(sanitized, convertXmlTags);

  // URI filtering - replace non-https protocols with "(redacted)"
  sanitized = sanitizeUrlProtocols(sanitized);

  // Domain filtering for HTTPS URIs
  sanitized = sanitizeUrlDomains(sanitized, allowedDomains);

  // Apply truncation limits
  sanitized = applyTruncation(sanitized, maxLength);

  // Neutralize GitHub references if restrictions are configured
  sanitized = neutralizeGitHubReferences(sanitized, allowedGitHubRefs);

  // Neutralize common bot trigger phrases
  sanitized = neutralizeBotTriggers(sanitized, maxBotMentions);

  // Neutralize template syntax delimiters (defense-in-depth)
  // This prevents potential issues if content is processed by downstream template engines
  sanitized = neutralizeTemplateDelimiters(sanitized);

  // Balance markdown code regions to fix improperly nested fences
  // This repairs markdown where AI models generate nested code blocks at the same indentation
  const { balanceCodeRegions } = require("./markdown_code_region_balancer.cjs");
  sanitized = balanceCodeRegions(sanitized);

  // Trim excessive whitespace
  return sanitized.trim();
}

module.exports = {
  sanitizeContentCore,
  getRedactedDomains,
  addRedactedDomain,
  clearRedactedDomains,
  writeRedactedDomainsLog,
  extractDomainsFromUrl,
  buildAllowedDomains,
  buildAllowedGitHubReferences,
  getCurrentRepoSlug,
  sanitizeDomainName,
  sanitizeUrlProtocols,
  sanitizeUrlDomains,
  neutralizeCommands,
  neutralizeGitHubReferences,
  removeXmlComments,
  convertXmlTags,
  applyToNonCodeRegions,
  neutralizeBotTriggers,
  MAX_BOT_TRIGGER_REFERENCES,
  neutralizeTemplateDelimiters,
  applyTruncation,
  hardenUnicodeText,
  decodeHtmlEntities,
};
