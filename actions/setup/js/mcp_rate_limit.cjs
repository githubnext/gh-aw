// @ts-check

/**
 * Extract only the status and retry duration, never query text or URLs.
 * @param {string} message
 * @returns {string|null}
 */
function getMcpRateLimitHint(message) {
  // Echoed queries can contain unescaped quotes, so use their final closing delimiter.
  const unquotedMessage = message.replace(/\bquery\s+(['"])[\s\S]*\1(?=\s*(?::|$))/gi, match => " ".repeat(match.length)).replace(/"(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*'/g, match => " ".repeat(match.length));
  const requestStatuses = Array.from(unquotedMessage.matchAll(/\b(?:GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS)\s+https?:\/\/\S*?:\s+([45]\d{2})\b/gi)).map(match => ({ index: match.index + match[0].lastIndexOf(match[1]), code: match[1] }));
  const diagnosticMessage = unquotedMessage.replace(/https?:\/\/\S+/gi, match => " ".repeat(match.length));
  const statuses = Array.from(diagnosticMessage.matchAll(/(?:^|\bHTTP(?:\/[\d.]+)?\s+|\bstatus(?:\s+code)?\s*[:=]?\s*|:\s+)([45]\d{2})\b/gi))
    .filter(match => !/\bretry-after\s*$/i.test(diagnosticMessage.slice(0, match.index)))
    .map(match => ({ index: match.index + match[0].lastIndexOf(match[1]), code: match[1] }));
  const status = [...statuses, ...requestStatuses].sort((first, second) => first.index - second.index).at(-1);
  if (!status || status.code !== "429") return null;
  const diagnostic = diagnosticMessage.slice(status.index);
  const duration = diagnostic.match(/\btry again in\s+((?:\d+(?:\.\d+)?(?:ns|µs|μs|us|ms|s|m|h))+)/i);
  if (duration) return `try again in ${duration[1]}`;
  const retryAfter = diagnostic.match(/\bretry-after\s*[:=]\s*(\d+(?:\.\d+)?|(?:Mon|Tue|Wed|Thu|Fri|Sat|Sun),\s+\d{1,2}\s+(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s+\d{4}\s+\d{2}:\d{2}:\d{2}\s+GMT)\b/i);
  if (!retryAfter) return "No retry hint provided";
  return /^\d/.test(retryAfter[1]) ? `Retry-After: ${retryAfter[1]}s` : `Retry-After: ${retryAfter[1]}`;
}

/**
 * @param {Object} entry
 * @returns {string|null}
 */
function getRpcRateLimitHint(entry) {
  const payload = entry.payload;
  if (payload?.error && typeof payload.error.message === "string") {
    return getMcpRateLimitHint(payload.error.message);
  }
  if (payload?.result?.isError === true && Array.isArray(payload.result.content)) {
    for (const block of payload.result.content) {
      if (block?.type === "text" && typeof block.text === "string") {
        const hint = getMcpRateLimitHint(block.text);
        if (hint !== null) return hint;
      }
    }
  }
  return null;
}

module.exports = { getMcpRateLimitHint, getRpcRateLimitHint };
