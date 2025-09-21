function sanitizeContent(content) {
    if (!content || typeof content !== "string") {
        return "";
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
    const allowedDomains = allowedDomainsEnv
        ? allowedDomainsEnv
            .split(",")
            .map(d => d.trim())
            .filter(d => d)
        : defaultAllowedDomains;
    let sanitized = content;
    sanitized = neutralizeMentions(sanitized);
    sanitized = sanitized.replace(/[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]/g, "");
    sanitized = convertXmlTagsToParentheses(sanitized);
    sanitized = sanitizeUrlProtocols(sanitized);
    sanitized = sanitizeUrlDomains(sanitized);
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
function neutralizeMentions(content) {
    return content.replace(/(?<!`)@([\w-]+)(?![\w.-]*@[\w.-]+\w)(?!`)/g, "`@$1`");
}
function convertXmlTagsToParentheses(content) {
    let result = content.replace(/<([^<>/\s]+)>/g, "($1)");
    result = result.replace(/<\/([^<>/\s]+)>/g, "(/$1)");
    result = result.replace(/<([^<>/\s]+)\/>/g, "($1/)");
    return result;
}
function sanitizeUrlProtocols(content) {
    return content.replace(/\b(?!https:\/\/)\w+:\/\/[^\s<>"`]+/gi, "(redacted)");
}
function sanitizeUrlDomains(content) {
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
    return content.replace(/https:\/\/([^\/\s<>"`]+)/gi, (match, domain) => {
        const isAllowed = allowedDomains.some(allowed => domain.toLowerCase().endsWith(allowed.toLowerCase()));
        return isAllowed ? match : "(redacted)";
    });
}
async function sanitizeOutputMain() {
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
    const sanitizedContent = sanitizeContent(outputContent);
    core.info(`Sanitized content length: ${sanitizedContent.length}`);
    core.setOutput("sanitized_content", sanitizedContent);
    if (sanitizedContent !== outputContent) {
        const sizeDiff = outputContent.length - sanitizedContent.length;
        const reductionPercent = ((sizeDiff / outputContent.length) * 100).toFixed(1);
        core.info(`Content sanitized: ${sizeDiff} characters removed (${reductionPercent}% reduction)`);
        await core.summary
            .addRaw("## Content Sanitization Summary\n")
            .addRaw(`- **Original size**: ${outputContent.length.toLocaleString()} characters\n`)
            .addRaw(`- **Sanitized size**: ${sanitizedContent.length.toLocaleString()} characters\n`)
            .addRaw(`- **Reduction**: ${sizeDiff.toLocaleString()} characters (${reductionPercent}%)\n`)
            .write();
    }
    else {
        core.info("No sanitization changes were needed");
    }
}
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
