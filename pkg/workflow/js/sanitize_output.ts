/**
 * Sanitizes content for safe output in GitHub Actions
 * @param content - The content to sanitize
 * @returns The sanitized content
 */
function sanitizeContent(content: string): string {
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
    sanitized = sanitized.substring(0, maxLength) + "\n\n[Content truncated for safety]";
  }

  // Limit number of lines (65k max)
  const maxLines = 65536;
  const lines = sanitized.split("\n");
  if (lines.length > maxLines) {
    sanitized = lines.slice(0, maxLines).join("\n") + "\n\n[Content truncated for safety - too many lines]";
  }

  return sanitized;
}

/**
 * Neutralizes @mentions by wrapping them in backticks
 * @param content - The content to process
 * @returns Content with neutralized @mentions
 */
function neutralizeMentions(content: string): string {
  // Match @mentions (word boundaries, alphanumeric, hyphens, underscores)
  // But exclude email addresses and already backticked mentions
  return content.replace(/(?<!`)@([\w-]+)(?![\w.-]*@[\w.-]+\w)(?!`)/g, "`@$1`");
}

/**
 * Converts XML tags to parentheses format for safety
 * @param content - The content to process
 * @returns Content with XML tags converted to parentheses
 */
function convertXmlTagsToParentheses(content: string): string {
  // Convert opening XML tags like <tag> to (tag)
  let result = content.replace(/<([^<>/\s]+)>/g, "($1)");

  // Convert closing XML tags like </tag> to (/tag)
  result = result.replace(/<\/([^<>/\s]+)>/g, "(/$1)");

  // Convert self-closing XML tags like <tag/> to (tag/)
  result = result.replace(/<([^<>/\s]+)\/>/g, "($1/)");

  return result;
}

/**
 * Sanitizes URL protocols, only allowing https://
 * @param content - The content to process
 * @returns Content with non-HTTPS URLs redacted
 */
function sanitizeUrlProtocols(content: string): string {
  // Replace non-https protocols with (redacted)
  // This regex matches protocol:// but excludes https://
  return content.replace(/\b(?!https:\/\/)\w+:\/\/[^\s<>"`]+/gi, "(redacted)");
}

/**
 * Sanitizes URL domains, only allowing domains from allowlist
 * @param content - The content to process
 * @returns Content with non-allowed domains redacted
 */
function sanitizeUrlDomains(content: string): string {
  const allowedDomainsEnv = process.env.GITHUB_AW_ALLOWED_DOMAINS;
  const defaultAllowedDomains = ["github.com", "github.io", "githubusercontent.com", "githubassets.com", "github.dev", "codespaces.new"];

  const allowedDomains = allowedDomainsEnv
    ? allowedDomainsEnv
        .split(",")
        .map(d => d.trim())
        .filter(d => d)
    : defaultAllowedDomains;

  // Match HTTPS URLs and check domains
  return content.replace(/https:\/\/([^\/\s<>"`]+)/gi, (match, domain) => {
    // Check if domain is in allowlist (case-insensitive)
    const isAllowed = allowedDomains.some(allowed => domain.toLowerCase().endsWith(allowed.toLowerCase()));

    return isAllowed ? match : "(redacted)";
  });
}

async function sanitizeOutputMain(): Promise<void> {
  // Read the agent output content from environment variable
  const outputContent = process.env.GITHUB_AW_AGENT_OUTPUT;
  if (!outputContent) {
    core.info("No GITHUB_AW_AGENT_OUTPUT environment variable found");
    core.setOutput("sanitized_content", "");
    return;
  }

  if (outputContent.trim() === "") {
    core.info("Agent output content is empty");
    core.setOutput("sanitized_content", "");
    return;
  }

  core.info(`Original content length: ${outputContent.length}`);

  // Sanitize the content
  const sanitizedContent = sanitizeContent(outputContent);

  core.info(`Sanitized content length: ${sanitizedContent.length}`);

  // Set the sanitized content as output
  core.setOutput("sanitized_content", sanitizedContent);

  // If content was modified, log a summary
  if (sanitizedContent !== outputContent) {
    const sizeDiff = outputContent.length - sanitizedContent.length;
    const reductionPercent = ((sizeDiff / outputContent.length) * 100).toFixed(1);

    core.info(`Content sanitized: ${sizeDiff} characters removed (${reductionPercent}% reduction)`);

    // Write summary
    await core.summary
      .addRaw("## Content Sanitization Summary\n")
      .addRaw(`- **Original size**: ${outputContent.length.toLocaleString()} characters\n`)
      .addRaw(`- **Sanitized size**: ${sanitizedContent.length.toLocaleString()} characters\n`)
      .addRaw(`- **Reduction**: ${sizeDiff.toLocaleString()} characters (${reductionPercent}%)\n`)
      .write();
  } else {
    core.info("No sanitization changes were needed");
  }
}

// Export for testing
if (typeof module !== "undefined" && module.exports) {
  module.exports = {
    sanitizeContent,
    neutralizeMentions,
    convertXmlTagsToParentheses,
    sanitizeUrlProtocols,
    sanitizeUrlDomains,
  };
}

(async () => {
  await sanitizeOutputMain();
})();
