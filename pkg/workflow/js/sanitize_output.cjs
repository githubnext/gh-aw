/**
 * Sanitizes content for safe output in GitHub Actions
 * @param {string} content - The content to sanitize
 * @returns {string} The sanitized content
 */
function sanitizeContent(content) {
  if (!content || typeof content !== "string") {
    return "";
  }

  // Read allowed domains from environment variable
  const allowedDomainsEnv = process.env.GITHUB_AW_ALLOWED_DOMAINS;
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

  // Remove control characters (except newlines and tabs)
  sanitized = sanitized.replace(/[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]/g, "");

  // XML tag neutralization - convert XML tags to parentheses format
  sanitized = convertXmlTagsToParentheses(sanitized);

  // URI filtering - replace non-https protocols with "(redacted)"
  // Step 1: Temporarily mark HTTPS URLs to protect them
  sanitized = sanitizeUrlProtocols(sanitized);

  // Domain filtering for HTTPS URIs
  // Match https:// URIs and check if domain is in allowlist
  sanitized = sanitizeUrlDomains(sanitized);

  // Limit total length to prevent DoS (0.5MB max)
  const maxLength = 524288;
  if (sanitized.length > maxLength) {
    sanitized = sanitized.substring(0, maxLength) + "\n[Content truncated due to length]";
  }

  // Limit number of lines to prevent log flooding (65k max)
  const lines = sanitized.split("\n");
  const maxLines = 65000;
  if (lines.length > maxLines) {
    sanitized = lines.slice(0, maxLines).join("\n") + "\n[Content truncated due to line count]";
  }

  // Remove ANSI escape sequences
  sanitized = sanitized.replace(/\x1b\[[0-9;]*[mGKH]/g, "");

  // Neutralize common bot trigger phrases
  sanitized = neutralizeBotTriggers(sanitized);

  // Neutralize command triggers at the start of text to prevent cycles
  sanitized = neutralizeCommandAtStart(sanitized);

  // Trim excessive whitespace
  return sanitized.trim();

  /**
   * Convert XML tags to parentheses format while preserving non-XML uses of < and >
   * @param {string} s - The string to process
   * @returns {string} The string with XML tags converted to parentheses
   */
  function convertXmlTagsToParentheses(s) {
    if (!s || typeof s !== "string") {
      return s;
    }

    // XML tag patterns that should be converted to parentheses
    return (
      s
        // Standard XML tags: <tag>, <tag attr="value">, <tag/>, </tag>
        .replace(/<\/?[a-zA-Z][a-zA-Z0-9\-_:]*(?:\s[^>]*|\/)?>/g, match => {
          // Extract the tag name and content without < >
          const innerContent = match.slice(1, -1);
          return `(${innerContent})`;
        })
        // XML comments: <!-- comment -->
        .replace(/<!--[\s\S]*?-->/g, match => {
          const innerContent = match.slice(4, -3); // Remove <!-- and -->
          return `(!--${innerContent}--)`;
        })
        // CDATA sections: <![CDATA[content]]>
        .replace(/<!\[CDATA\[[\s\S]*?\]\]>/g, match => {
          const innerContent = match.slice(9, -3); // Remove <![CDATA[ and ]]>
          return `(![CDATA[${innerContent}]])`;
        })
        // XML processing instructions: <?xml ... ?>
        .replace(/<\?[\s\S]*?\?>/g, match => {
          const innerContent = match.slice(2, -2); // Remove <? and ?>
          return `(?${innerContent}?)`;
        })
        // DOCTYPE declarations: <!DOCTYPE ...>
        .replace(/<!DOCTYPE[^>]*>/gi, match => {
          const innerContent = match.slice(9, -1); // Remove <!DOCTYPE and >
          return `(!DOCTYPE${innerContent})`;
        })
    );
  }

  /**
   * Remove unknown domains
   * @param {string} s - The string to process
   * @returns {string} The string with unknown domains redacted
   */
  function sanitizeUrlDomains(s) {
    s = s.replace(/\bhttps:\/\/([^\/\s\])}'"<>&\x00-\x1f]+)/gi, (match, domain) => {
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
   * Neutralizes bot trigger phrases by wrapping them in backticks
   * @param {string} s - The string to process
   * @returns {string} The string with neutralized bot triggers
   */
  function neutralizeBotTriggers(s) {
    // Neutralize common bot trigger phrases like "fixes #123", "closes #asdfs", etc.
    return s.replace(/\b(fixes?|closes?|resolves?|fix|close|resolve)\s+#(\w+)/gi, (match, action, ref) => `\`${action} #${ref}\``);
  }

  /**
   * Neutralizes command triggers at the start of text to prevent cycles
   * @param {string} s - The string to process
   * @returns {string} The string with neutralized command at start
   */
  function neutralizeCommandAtStart(s) {
    // Read command from environment variable
    const command = process.env.GITHUB_AW_COMMAND;
    if (!command) {
      return s; // No command configured, nothing to neutralize
    }

    // Check if text starts with /command
    const trimmedText = s.trim();
    const commandPattern = `/${command}`;

    if (trimmedText.startsWith(commandPattern)) {
      // Neutralize the command at the start by wrapping it in backticks
      // This prevents the output from triggering another workflow run
      return s.replace(
        new RegExp(`^(\\s*)(${commandPattern.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")})`),
        (match, whitespace, cmd) => `${whitespace}\`${cmd}\``
      );
    }

    return s;
  }
}

async function main() {
  const fs = require("fs");
  const outputFile = process.env.GITHUB_AW_SAFE_OUTPUTS;
  if (!outputFile) {
    core.info("GITHUB_AW_SAFE_OUTPUTS not set, no output to collect");
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
