/**
 * Sanitizes content for safe output in GitHub Actions
 * @param {string} content - The content to sanitize
 * @param {Array<object>?} auditLog - Optional audit log array to track changes
 * @returns {string} The sanitized content
 */
function sanitizeContent(content, auditLog = null) {
  if (!content || typeof content !== "string") {
    return "";
  }

  // Initialize audit logging if provided
  const audit = auditLog ? 
    (/** @type {string} */ type, /** @type {string} */ original, /** @type {string} */ sanitized, /** @type {number} */ lineNumber, /** @type {string} */ description) => {
      auditLog.push({
        type,
        original,
        sanitized,
        line_number: lineNumber,
        description,
        context: getContext(content, original, lineNumber)
      });
    } : 
    () => {}; // No-op function if no audit log provided

  // Read allowed domains from environment variable
  const allowedDomainsEnv = process.env.GITHUB_AW_ALLOWED_DOMAINS;
  const defaultAllowedDomains = [
    "github.com",
    "github.io",
    "githubusercontent.com",
    "githubassets.com",
    "github.dev",
    "codespaces.new",
  ];

  const allowedDomains = allowedDomainsEnv
    ? allowedDomainsEnv
        .split(",")
        .map(d => d.trim())
        .filter(d => d)
    : defaultAllowedDomains;

  let sanitized = content;

  // Neutralize @mentions to prevent unintended notifications
  sanitized = neutralizeMentions(sanitized, audit);

  // Remove control characters (except newlines and tabs)
  const beforeControlChars = sanitized;
  sanitized = sanitized.replace(/[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]/g, "");
  if (beforeControlChars !== sanitized) {
    const removedChars = beforeControlChars.replace(/[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]/g, '');
    audit("control_char", beforeControlChars, sanitized, getLineNumber(content, beforeControlChars), "Control characters removed for safety");
  }

  // XML character escaping
  const beforeXML = sanitized;
  sanitized = sanitized
    .replace(/&/g, "&amp;") // Must be first to avoid double-escaping
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&apos;");
  if (beforeXML !== sanitized) {
    audit("xml_escape", beforeXML, sanitized, getLineNumber(content, beforeXML), "XML characters escaped for safety");
  }

  // URI filtering - replace non-https protocols with "(redacted)"
  // Step 1: Temporarily mark HTTPS URLs to protect them
  sanitized = sanitizeUrlProtocols(sanitized, audit, content);

  // Domain filtering for HTTPS URIs
  // Match https:// URIs and check if domain is in allowlist
  sanitized = sanitizeUrlDomains(sanitized, audit, content);

  // Limit total length to prevent DoS (0.5MB max)
  const maxLength = 524288;
  if (sanitized.length > maxLength) {
    const original = sanitized;
    sanitized = sanitized.substring(0, maxLength) + "\n[Content truncated due to length]";
    audit("truncation", original, sanitized, -1, `Content truncated: ${original.length} chars → ${maxLength} chars max`);
  }

  // Limit number of lines to prevent log flooding (65k max)
  const lines = sanitized.split("\n");
  const maxLines = 65000;
  if (lines.length > maxLines) {
    const original = sanitized;
    sanitized = lines.slice(0, maxLines).join("\n") + "\n[Content truncated due to line count]";
    audit("truncation", original, sanitized, -1, `Content truncated: ${lines.length} lines → ${maxLines} lines max`);
  }

  // Remove ANSI escape sequences
  const beforeANSI = sanitized;
  sanitized = sanitized.replace(/\x1b\[[0-9;]*[mGKH]/g, "");
  if (beforeANSI !== sanitized) {
    audit("ansi_removal", beforeANSI, sanitized, getLineNumber(content, beforeANSI), "ANSI escape sequences removed");
  }

  // Neutralize common bot trigger phrases
  sanitized = neutralizeBotTriggers(sanitized, audit);

  // Trim excessive whitespace
  return sanitized.trim();

  /**
   * Get line number for a given substring in the original content
   * @param {string} originalContent - The original content
   * @param {string} substring - The substring to find
   * @returns {number} Line number (1-indexed) or -1 if not found
   */
  function getLineNumber(originalContent, substring) {
    const index = originalContent.indexOf(substring);
    if (index === -1) return -1;
    return originalContent.substring(0, index).split('\n').length;
  }

  /**
   * Get context around a change for audit logging
   * @param {string} content - The content to extract context from
   * @param {string} original - The original text that was changed
   * @param {number} lineNumber - Line number where change occurred
   * @returns {string} Context string
   */
  function getContext(content, original, lineNumber) {
    if (!content || !original) return "";
    
    const lines = content.split('\n');
    if (lineNumber < 1 || lineNumber > lines.length) return "";
    
    const startLine = Math.max(0, lineNumber - 2);
    const endLine = Math.min(lines.length, lineNumber + 1);
    const contextLines = lines.slice(startLine, endLine);
    
    return contextLines.join('\n');
  }

  /**
   * Remove unknown domains
   * @param {string} s - The string to process
   * @param {Function} audit - Audit logging function
   * @param {string} originalContent - Original content for line number tracking
   * @returns {string} The string with unknown domains redacted
   */
  function sanitizeUrlDomains(s, audit, originalContent) {
    s = s.replace(
      /\bhttps:\/\/([^\/\s\])}'"<>&\x00-\x1f]+)/gi,
      (match, domain) => {
        // Extract the hostname part (before first slash, colon, or other delimiter)
        const hostname = domain.split(/[\/:\?#]/)[0].toLowerCase();

        // Check if this domain or any parent domain is in the allowlist
        const isAllowed = allowedDomains.some(allowedDomain => {
          const normalizedAllowed = allowedDomain.toLowerCase();
          return (
            hostname === normalizedAllowed ||
            hostname.endsWith("." + normalizedAllowed)
          );
        });

        if (!isAllowed) {
          const lineNumber = getLineNumber(originalContent, match);
          audit("url", match, "(redacted)", lineNumber, `HTTPS URL redacted due to disallowed domain: ${hostname}`);
          return "(redacted)";
        }

        return match;
      }
    );

    return s;
  }

  /**
   * Remove unknown protocols except https
   * @param {string} s - The string to process
   * @param {Function} audit - Audit logging function
   * @param {string} originalContent - Original content for line number tracking
   * @returns {string} The string with non-https protocols redacted
   */
  function sanitizeUrlProtocols(s, audit, originalContent) {
    // Match both protocol:// and protocol: patterns
    // This covers URLs like https://example.com, javascript:alert(), mailto:user@domain.com, etc.
    return s.replace(
      /\b(\w+):(?:\/\/)?[^\s\])}'"<>&\x00-\x1f]+/gi,
      (match, protocol) => {
        // Allow https (case insensitive), redact everything else
        if (protocol.toLowerCase() !== "https") {
          const lineNumber = getLineNumber(originalContent, match);
          audit("url", match, "(redacted)", lineNumber, `URL redacted due to unsafe protocol: ${protocol}`);
          return "(redacted)";
        }
        return match;
      }
    );
  }

  /**
   * Neutralizes @mentions by wrapping them in backticks
   * @param {string} s - The string to process
   * @param {Function} audit - Audit logging function
   * @returns {string} The string with neutralized mentions
   */
  function neutralizeMentions(s, audit) {
    // Replace @name or @org/team outside code with `@name`
    return s.replace(
      /(^|[^\w`])@([A-Za-z0-9](?:[A-Za-z0-9-]{0,37}[A-Za-z0-9])?(?:\/[A-Za-z0-9._-]+)?)/g,
      (match, p1, p2, offset) => {
        const original = `@${p2}`;
        const sanitized = `\`@${p2}\``;
        const lineNumber = getLineNumber(s, original);
        audit("mention", original, sanitized, lineNumber, "@mention neutralized to prevent unintended notifications");
        return `${p1}${sanitized}`;
      }
    );
  }

  /**
   * Neutralizes bot trigger phrases by wrapping them in backticks
   * @param {string} s - The string to process
   * @param {Function} audit - Audit logging function
   * @returns {string} The string with neutralized bot triggers
   */
  function neutralizeBotTriggers(s, audit) {
    // Neutralize common bot trigger phrases like "fixes #123", "closes #asdfs", etc.
    return s.replace(
      /\b(fixes?|closes?|resolves?|fix|close|resolve)\s+#(\w+)/gi,
      (match, action, ref) => {
        const original = `${action} #${ref}`;
        const sanitized = `\`${action} #${ref}\``;
        const lineNumber = getLineNumber(s, original);
        audit("bot_trigger", original, sanitized, lineNumber, "Bot trigger phrase neutralized to prevent unintended actions");
        return sanitized;
      }
    );
  }
}

async function main() {
  const fs = require("fs");
  const path = require("path");
  const outputFile = process.env.GITHUB_AW_SAFE_OUTPUTS;
  if (!outputFile) {
    console.log("GITHUB_AW_SAFE_OUTPUTS not set, no output to collect");
    core.setOutput("output", "");
    return;
  }

  if (!fs.existsSync(outputFile)) {
    console.log("Output file does not exist:", outputFile);
    core.setOutput("output", "");
    return;
  }

  const outputContent = fs.readFileSync(outputFile, "utf8");
  if (outputContent.trim() === "") {
    console.log("Output file is empty");
    core.setOutput("output", "");
  } else {
    // Create audit log to track sanitization changes
    /** @type {Array<object>} */ const auditLog = [];
    
    const sanitizedContent = sanitizeContent(outputContent, auditLog);
    console.log(
      "Collected agentic output (sanitized):",
      sanitizedContent.substring(0, 200) +
        (sanitizedContent.length > 200 ? "..." : "")
    );
    core.setOutput("output", sanitizedContent);

    // Save audit log if there were any changes
    if (auditLog.length > 0) {
      try {
        // Create audit log in the same directory as the safe outputs
        const outputDir = path.dirname(outputFile);
        const auditLogPath = path.join(outputDir, "sanitization_audit.json");
        
        const auditData = {
          timestamp: new Date().toISOString(),
          total_changes: auditLog.length,
          changes_by_type: auditLog.reduce((/** @type {Record<string, number>} */ acc, /** @type {any} */ change) => {
            acc[change.type] = (acc[change.type] || 0) + 1;
            return acc;
          }, /** @type {Record<string, number>} */ ({})),
          changes: auditLog
        };

        fs.writeFileSync(auditLogPath, JSON.stringify(auditData, null, 2));
        console.log(`Sanitization audit log written to: ${auditLogPath}`);
        console.log(`Total sanitization changes: ${auditLog.length}`);
      } catch (error) {
        console.warn("Failed to write sanitization audit log:", /** @type {Error} */ (error).message);
      }
    } else {
      console.log("No sanitization changes detected");
    }
  }
}

await main();
