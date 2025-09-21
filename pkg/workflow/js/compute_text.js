function sanitizeContentCompute(content) {
  if (!content || typeof content !== "string") {
    return "";
  }
  const allowedDomainsEnv = process.env.GITHUB_AW_ALLOWED_DOMAINS;
  const defaultAllowedDomains = ["github.com", "github.io", "githubusercontent.com", "githubassets.com", "github.dev", "codespaces.new"];
  const allowedDomains = allowedDomainsEnv
    ? allowedDomainsEnv
        .split(",")
        .map(d => d.trim())
        .filter(d => d)
    : defaultAllowedDomains;
  let sanitized = content;
  sanitized = neutralizeMentionsCompute(sanitized);
  sanitized = sanitized.replace(/[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]/g, "");
  sanitized = convertXmlTagsToParenthesesCompute(sanitized);
  sanitized = sanitizeUrlProtocolsCompute(sanitized);
  sanitized = sanitizeUrlDomainsCompute(sanitized);
  const maxLength = 524288;
  if (sanitized.length > maxLength) {
    sanitized = sanitized.substring(0, maxLength) + "\n\n[Content truncated for safety]";
  }
  const maxLines = 65536;
  const lines = sanitized.split("\n");
  if (lines.length > maxLines) {
    sanitized = lines.slice(0, maxLines).join("\n") + "\n\n[Content truncated for safety - too many lines]";
  }
  return sanitized;
}
function neutralizeMentionsCompute(content) {
  return content.replace(/(?<!`)@([\w-]+)(?![\w.-]*@[\w.-]+\w)(?!`)/g, "`@$1`");
}
function convertXmlTagsToParenthesesCompute(content) {
  let result = content.replace(/<([^<>/\s]+)>/g, "($1)");
  result = result.replace(/<\/([^<>/\s]+)>/g, "(/$1)");
  result = result.replace(/<([^<>/\s]+)\/>/g, "($1/)");
  return result;
}
function sanitizeUrlProtocolsCompute(content) {
  return content.replace(/\b(?!https:\/\/)\w+:\/\/[^\s<>"`]+/gi, "(redacted)");
}
function sanitizeUrlDomainsCompute(content) {
  const allowedDomainsEnv = process.env.GITHUB_AW_ALLOWED_DOMAINS;
  const defaultAllowedDomains = ["github.com", "github.io", "githubusercontent.com", "githubassets.com", "github.dev", "codespaces.new"];
  const allowedDomains = allowedDomainsEnv
    ? allowedDomainsEnv
        .split(",")
        .map(d => d.trim())
        .filter(d => d)
    : defaultAllowedDomains;
  return content.replace(/https:\/\/([^\/\s<>"`]+)/gi, (match, domain) => {
    const isAllowed = allowedDomains.some(allowed => domain.toLowerCase().endsWith(allowed.toLowerCase()));
    return isAllowed ? match : "(redacted)";
  });
}
async function computeTextMain() {
  const fs = require("fs");
  const agentOutput = process.env.GITHUB_AW_AGENT_OUTPUT;
  const sanitizedOutput = process.env.GITHUB_AW_SANITIZED_OUTPUT;
  core.info(`Agent output content length: ${agentOutput ? agentOutput.length : 0}`);
  core.info(`Sanitized output content length: ${sanitizedOutput ? sanitizedOutput.length : 0}`);
  let textContent = "";
  if (sanitizedOutput) {
    textContent = sanitizedOutput;
    core.info("Using sanitized output content");
  } else if (agentOutput) {
    textContent = sanitizeContentCompute(agentOutput);
    core.info("Sanitized agent output content");
  } else {
    core.info("No agent output content found");
    core.setOutput("computed_text", "");
    return;
  }
  const lineCount = textContent.split("\n").length;
  const wordCount = textContent.split(/\s+/).filter(word => word.length > 0).length;
  const charCount = textContent.length;
  const charCountNoSpaces = textContent.replace(/\s/g, "").length;
  const urlMatches = textContent.match(/https?:\/\/[^\s<>"`]+/gi) || [];
  const uniqueUrls = [...new Set(urlMatches)];
  const mentionMatches = textContent.match(/`?@[\w-]+`?/g) || [];
  const uniqueMentions = [...new Set(mentionMatches)];
  const codeBlockMatches = textContent.match(/```[\s\S]*?```/g) || [];
  const inlineCodeMatches = textContent.match(/`[^`\n]+`/g) || [];
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
  core.setOutput("computed_text", textContent);
  core.setOutput("text_metrics", JSON.stringify(computedMetrics));
  core.setOutput("line_count", lineCount.toString());
  core.setOutput("word_count", wordCount.toString());
  core.setOutput("char_count", charCount.toString());
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
  await core.summary.addRaw(summaryContent).write();
  core.info(`Text analysis complete: ${wordCount} words, ${charCount} characters`);
}
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
