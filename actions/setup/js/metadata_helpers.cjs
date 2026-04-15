// @ts-check

/**
 * Metadata Helpers
 *
 * Utilities for rendering and parsing the structured metadata block that agents
 * can attach to safe-output body fields (add_comment, create_issue, etc.).
 *
 * The metadata channel solves the problem of HTML comments being stripped by the
 * safe-output sanitizer: instead of embedding structured data as <!-- ... --> markers
 * (which removeXmlComments() removes), agents use the `metadata` field and let the
 * runtime render it as a fenced code block that survives all sanitization passes.
 *
 * Format written into the GitHub body:
 *
 *   ```aw-metadata
 *   {"verdict":"APPROVE","criteria_passed":5}
 *   ```
 *
 * The `aw-metadata` language tag makes the block identifiable for programmatic
 * parsing by downstream workflows.
 */

/**
 * Language tag used for metadata fenced code blocks.
 * Must be a unique identifier so downstream scripts can reliably extract the block.
 */
const METADATA_FENCE_LANG = "aw-metadata";

/**
 * Render a validated metadata object as a fenced code block.
 * Returns an empty string when metadata is absent or empty so callers can
 * unconditionally append the result without producing stray blank lines.
 *
 * @param {Record<string, string|number|boolean|null>|null|undefined} metadata - The metadata object to render
 * @returns {string} Markdown fenced code block, or empty string if no metadata
 */
function renderMetadataBlock(metadata) {
  if (!metadata || typeof metadata !== "object" || Array.isArray(metadata)) {
    return "";
  }
  if (Object.keys(metadata).length === 0) {
    return "";
  }

  const json = JSON.stringify(metadata);
  return `\`\`\`${METADATA_FENCE_LANG}\n${json}\n\`\`\``;
}

/**
 * Parse the metadata object out of a body string that was previously rendered
 * with renderMetadataBlock().  Returns null when no metadata block is present
 * or when the embedded JSON is malformed.
 *
 * @param {string} body - The body string (e.g. a GitHub comment body)
 * @returns {Record<string, string|number|boolean|null>|null} Parsed metadata, or null
 */
function parseMetadataFromBody(body) {
  if (!body || typeof body !== "string") {
    return null;
  }

  // Match the fenced code block: ```aw-metadata\n...\n```
  const pattern = new RegExp("```" + METADATA_FENCE_LANG + "\\n([\\s\\S]*?)\\n```", "m");
  const match = body.match(pattern);
  if (!match) {
    return null;
  }

  try {
    const parsed = JSON.parse(match[1]);
    if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
      return parsed;
    }
    return null;
  } catch {
    return null;
  }
}

module.exports = {
  METADATA_FENCE_LANG,
  renderMetadataBlock,
  parseMetadataFromBody,
};
