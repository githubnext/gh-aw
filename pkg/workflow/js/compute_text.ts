/**
 * Sanitizes content for safe output in GitHub Actions
 * @param content - The content to sanitize
 * @returns The sanitized content
 */
function sanitizeContentCompute(content: string): string {
  if (!content || typeof content !== "string") {
    return "";
  }

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
  sanitized = neutralizeMentionsCompute(sanitized);

  // Remove control characters (except newlines and tabs)
  sanitized = sanitized.replace(/[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]/g, "");

  // XML tag neutralization - convert XML tags to parentheses format
  sanitized = convertXmlTagsToParenthesesCompute(sanitized);

  // URI filtering - replace non-https protocols with "(redacted)"
  // Step 1: Temporarily mark HTTPS URLs to protect them
  sanitized = sanitizeUrlProtocolsCompute(sanitized);

  // Domain filtering for HTTPS URIs
  // Match https:// URIs and check if domain is in allowlist
  sanitized = sanitizeUrlDomainsCompute(sanitized);

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
function neutralizeMentionsCompute(content: string): string {
  // Match @mentions (word boundaries, alphanumeric, hyphens, underscores)
  // But exclude email addresses and already backticked mentions
  return content.replace(/(?<!`)@([\w-]+)(?![\w.-]*@[\w.-]+\w)(?!`)/g, "`@$1`");
}

/**
 * Converts XML tags to parentheses format for safety
 * @param content - The content to process
 * @returns Content with XML tags converted to parentheses
 */
function convertXmlTagsToParenthesesCompute(content: string): string {
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
function sanitizeUrlProtocolsCompute(content: string): string {
  // Replace non-https protocols with (redacted)
  // This regex matches protocol:// but excludes https://
  return content.replace(/\b(?!https:\/\/)\w+:\/\/[^\s<>"`]+/gi, "(redacted)");
}

/**
 * Sanitizes URL domains, only allowing domains from allowlist
 * @param content - The content to process
 * @returns Content with non-allowed domains redacted
 */
function sanitizeUrlDomainsCompute(content: string): string {
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

  // Match HTTPS URLs and check domains
  return content.replace(/https:\/\/([^\/\s<>"`]+)/gi, (match, domain) => {
    // Check if domain is in allowlist (case-insensitive)
    const isAllowed = allowedDomains.some(allowed => 
      domain.toLowerCase().endsWith(allowed.toLowerCase())
    );
    
    return isAllowed ? match : "(redacted)";
  });
}

async function computeTextMain(): Promise<void> {
  const fs = require("fs");

  // Read agent output from environment variables
  const agentOutput = process.env.GITHUB_AW_AGENT_OUTPUT;
  const sanitizedOutput = process.env.GITHUB_AW_SANITIZED_OUTPUT;

  core.info(`Agent output content length: ${agentOutput ? agentOutput.length : 0}`);
  core.info(`Sanitized output content length: ${sanitizedOutput ? sanitizedOutput.length : 0}`);

  let textContent = "";

  if (sanitizedOutput) {
    textContent = sanitizedOutput;
    core.info("Using sanitized output content");
  } else if (agentOutput) {
    // Sanitize the agent output if no sanitized version is available
    textContent = sanitizeContentCompute(agentOutput);
    core.info("Sanitized agent output content");
  } else {
    core.info("No agent output content found");
    core.setOutput("computed_text", "");
    return;
  }

  // Extract and compute various text metrics
  const lineCount = textContent.split("\n").length;
  const wordCount = textContent.split(/\s+/).filter(word => word.length > 0).length;
  const charCount = textContent.length;
  const charCountNoSpaces = textContent.replace(/\s/g, "").length;

  // Extract URLs
  const urlMatches = textContent.match(/https?:\/\/[^\s<>"`]+/gi) || [];
  const uniqueUrls = [...new Set(urlMatches)];

  // Extract @mentions (even neutralized ones)
  const mentionMatches = textContent.match(/`?@[\w-]+`?/g) || [];
  const uniqueMentions = [...new Set(mentionMatches)];

  // Extract code blocks
  const codeBlockMatches = textContent.match(/```[\s\S]*?```/g) || [];
  const inlineCodeMatches = textContent.match(/`[^`\n]+`/g) || [];

  // Create summary object
  const computedMetrics = {
    lines: lineCount,
    words: wordCount,
    characters: charCount,
    charactersNoSpaces: charCountNoSpaces,
    urls: uniqueUrls.length,
    mentions: uniqueMentions.length,
    codeBlocks: codeBlockMatches.length,
    inlineCode: inlineCodeMatches.length,
  };

  // Set outputs
  core.setOutput("computed_text", textContent);
  core.setOutput("text_metrics", JSON.stringify(computedMetrics));
  core.setOutput("line_count", lineCount.toString());
  core.setOutput("word_count", wordCount.toString());
  core.setOutput("char_count", charCount.toString());

  // Generate summary report
  let summaryContent = "## 📊 Text Analysis Summary\n\n";
  summaryContent += `**Text Metrics:**\n`;
  summaryContent += `- Lines: ${lineCount.toLocaleString()}\n`;
  summaryContent += `- Words: ${wordCount.toLocaleString()}\n`;
  summaryContent += `- Characters: ${charCount.toLocaleString()}\n`;
  summaryContent += `- Characters (no spaces): ${charCountNoSpaces.toLocaleString()}\n`;
  
  if (uniqueUrls.length > 0) {
    summaryContent += `- URLs found: ${uniqueUrls.length}\n`;
  }
  
  if (uniqueMentions.length > 0) {
    summaryContent += `- Mentions found: ${uniqueMentions.length}\n`;
  }
  
  if (codeBlockMatches.length > 0) {
    summaryContent += `- Code blocks: ${codeBlockMatches.length}\n`;
  }
  
  if (inlineCodeMatches.length > 0) {
    summaryContent += `- Inline code snippets: ${inlineCodeMatches.length}\n`;
  }

  summaryContent += "\n";

  // Write summary
  await core.summary.addRaw(summaryContent).write();

  core.info(`Text analysis complete: ${wordCount} words, ${charCount} characters`);
}

// Export for testing
if (typeof module !== "undefined" && module.exports) {
  module.exports = {
    sanitizeContentCompute,
    neutralizeMentionsCompute,
    convertXmlTagsToParenthesesCompute,
    sanitizeUrlProtocolsCompute,
    sanitizeUrlDomainsCompute,
  };
}

(async () => {
  await computeTextMain();
})();