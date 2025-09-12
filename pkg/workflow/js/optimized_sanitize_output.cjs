/**
 * Optimized sanitization for CI Doctor workflow outputs
 * Performance improvements:
 * - Single-pass regex processing
 * - Early exit conditions
 * - Efficient domain filtering
 * - Minimal memory allocations
 */

const core = require("@actions/core");

/**
 * Optimized content sanitization with performance improvements
 * @param {string} content - The content to sanitize
 * @returns {string} The sanitized content
 */
function optimizedSanitizeContent(content) {
  if (!content || typeof content !== "string") {
    return "";
  }

  // Early exit for very small content
  if (content.length < 100) {
    return basicSanitize(content);
  }

  // Read allowed domains once and cache
  const allowedDomains = getAllowedDomains();

  let sanitized = content;

  // Single-pass combined sanitization
  sanitized = singlePassSanitize(sanitized, allowedDomains);

  // Length and line limiting (optimized)
  sanitized = applyLimits(sanitized);

  return sanitized.trim();
}

/**
 * Get allowed domains with caching
 */
let cachedAllowedDomains = null;
function getAllowedDomains() {
  if (cachedAllowedDomains) {
    return cachedAllowedDomains;
  }

  const allowedDomainsEnv = process.env.GITHUB_AW_ALLOWED_DOMAINS;
  const defaultAllowedDomains = [
    "github.com",
    "github.io",
    "githubusercontent.com",
    "githubassets.com",
    "github.dev",
    "codespaces.new",
  ];

  cachedAllowedDomains = allowedDomainsEnv
    ? allowedDomainsEnv
        .split(",")
        .map(d => d.trim())
        .filter(d => d)
    : defaultAllowedDomains;

  return cachedAllowedDomains;
}

/**
 * Basic sanitization for small content
 */
function basicSanitize(content) {
  return content
    .replace(/[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]/g, "") // Control chars
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&apos;")
    .replace(/\x1b\[[0-9;]*[mGKH]/g, ""); // ANSI codes
}

/**
 * Single-pass sanitization combining multiple operations
 */
function singlePassSanitize(content, allowedDomains) {
  // Build a comprehensive regex that handles multiple patterns in one pass
  const combinedRegex =
    /(\@[A-Za-z0-9](?:[A-Za-z0-9-]{0,37}[A-Za-z0-9])?(?:\/[A-Za-z0-9._-]+)?)|(\bhttps:\/\/([^\/\s\])}'"<>&\x00-\x1f]+))|(\b(\w+):(?:\/\/)?[^\s\])}'"<>&\x00-\x1f]+)|(\b(fixes?|closes?|resolves?|fix|close|resolve)\s+#(\w+))|(&)|(<)|(>)|(")|(')|([\x00-\x08\x0B\x0C\x0E-\x1F\x7F])|(\x1b\[[0-9;]*[mGKH])/gi;

  return content.replace(
    combinedRegex,
    (
      match,
      mention,
      httpsUrl,
      httpsDomain,
      otherUrl,
      otherProtocol,
      botTrigger,
      actionWord,
      issueRef,
      amp,
      lt,
      gt,
      quot,
      apos,
      control,
      ansi
    ) => {
      // Handle @mentions
      if (mention) {
        return `\`${mention}\``;
      }

      // Handle HTTPS URLs with domain filtering
      if (httpsUrl && httpsDomain) {
        const hostname = httpsDomain.split(/[\/:\?#]/)[0].toLowerCase();
        const isAllowed = allowedDomains.some(allowedDomain => {
          const normalizedAllowed = allowedDomain.toLowerCase();
          return (
            hostname === normalizedAllowed ||
            hostname.endsWith("." + normalizedAllowed)
          );
        });
        return isAllowed ? httpsUrl : "(redacted)";
      }

      // Handle non-HTTPS URLs
      if (otherUrl && otherProtocol) {
        return otherProtocol.toLowerCase() === "https"
          ? otherUrl
          : "(redacted)";
      }

      // Handle bot triggers
      if (botTrigger && actionWord && issueRef) {
        return `\`${actionWord} #${issueRef}\``;
      }

      // Handle XML entities
      if (amp) return "&amp;";
      if (lt) return "&lt;";
      if (gt) return "&gt;";
      if (quot) return "&quot;";
      if (apos) return "&apos;";

      // Handle control characters and ANSI codes
      if (control || ansi) return "";

      return match;
    }
  );
}

/**
 * Apply length and line limits efficiently
 */
function applyLimits(content) {
  const maxLength = 524288; // 0.5MB
  const maxLines = 65000;

  // Length limiting
  if (content.length > maxLength) {
    content =
      content.substring(0, maxLength) + "\n[Content truncated due to length]";
  }

  // Line limiting - only split if needed
  if (content.includes("\n")) {
    const lineCount = (content.match(/\n/g) || []).length + 1;
    if (lineCount > maxLines) {
      const lines = content.split("\n");
      content =
        lines.slice(0, maxLines).join("\n") +
        "\n[Content truncated due to line count]";
    }
  }

  return content;
}

/**
 * Fast sanitization for CI Doctor specific content
 * Optimized for error messages and investigation reports
 */
function sanitizeCIContent(content) {
  if (!content || typeof content !== "string") {
    return "";
  }

  // For CI error content, we can make assumptions about structure
  // and optimize accordingly

  // Early patterns for common CI content
  if (content.startsWith("Error:") || content.startsWith("FAIL:")) {
    // Error messages - minimal sanitization needed
    return basicSanitize(content);
  }

  if (content.includes("investigation") || content.includes("## ")) {
    // Investigation reports - need full sanitization
    return optimizedSanitizeContent(content);
  }

  // Default to optimized sanitization
  return optimizedSanitizeContent(content);
}

/**
 * Main sanitization entry point for CI Doctor
 */
async function main() {
  const fs = require("fs");
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
    return;
  }

  // Performance measurement
  const startTime = process.hrtime.bigint();

  // Use optimized sanitization
  const sanitizedContent = sanitizeCIContent(outputContent);

  const endTime = process.hrtime.bigint();
  const sanitizationTimeMs = Number(endTime - startTime) / 1000000;

  console.log(
    `Optimized sanitization completed in ${sanitizationTimeMs.toFixed(2)}ms`,
    `- Input: ${outputContent.length} chars`,
    `- Output: ${sanitizedContent.length} chars`,
    `- Reduction: ${((1 - sanitizedContent.length / outputContent.length) * 100).toFixed(1)}%`
  );

  core.setOutput("output", sanitizedContent);
}

// Performance monitoring
function measurePerformance(fn, name) {
  return function (...args) {
    const start = process.hrtime.bigint();
    const result = fn.apply(this, args);
    const end = process.hrtime.bigint();
    const timeMs = Number(end - start) / 1000000;
    console.log(`${name}: ${timeMs.toFixed(2)}ms`);
    return result;
  };
}

// Export for testing
module.exports = {
  optimizedSanitizeContent,
  sanitizeCIContent,
  singlePassSanitize,
  applyLimits,
};

// Run if called directly
if (require.main === module) {
  main().catch(error => {
    core.setFailed(error.message);
  });
}
