// @ts-check

/**
 * Parse JSONL content into an array of entries.
 * Malformed lines are ignored so callers can safely consume partially written logs.
 *
 * @param {string} content
 * @returns {unknown[]}
 */
function parseJsonlContent(content) {
  if (typeof content !== "string" || content.length === 0) {
    return [];
  }

  /** @type {unknown[]} */
  const entries = [];
  for (const rawLine of content.split("\n")) {
    const line = rawLine.trim();
    if (!line) {
      continue;
    }
    try {
      entries.push(JSON.parse(line));
    } catch {
      // Ignore malformed JSONL lines.
    }
  }
  return entries;
}

module.exports = {
  parseJsonlContent,
};
