// @ts-check

/**
 * Payload Helpers
 *
 * Utilities for rendering and parsing the structured payload block that agents
 * can attach to safe-output body fields (add_comment, create_issue, etc.).
 *
 * The payload channel solves the problem of HTML comments being stripped by the
 * safe-output sanitizer: instead of embedding structured data as <!-- ... --> markers
 * (which removeXmlComments() removes), agents use the `payload` field and let the
 * runtime render it as a fenced code block that survives all sanitization passes.
 *
 * Format written into the GitHub body:
 *
 *   ```json gh-aw-payload
 *   {"verdict":"APPROVE","criteria_passed":5}
 *   ```
 *
 * The `json` language tag enables syntax highlighting; the `gh-aw-payload` info
 * token makes the block uniquely identifiable for programmatic parsing by
 * downstream workflows.
 */

/**
 * The fenced code block info string used to mark payload blocks.
 * Format: "<language> <identifier>" — `json` for GitHub syntax highlighting,
 * `gh-aw-payload` as the machine-readable discriminator.
 */
const PAYLOAD_FENCE_INFO = "json gh-aw-payload";

/**
 * Render a validated payload object as a collapsible details section containing
 * a pretty-printed fenced JSON code block.
 * The object is always round-tripped through JSON.parse/JSON.stringify to normalize it
 * (removes undefined values, coerces types, etc.) before rendering.
 * Returns an empty string when payload is absent or empty so callers can
 * unconditionally append the result without producing stray blank lines.
 *
 * @param {Record<string, string|number|boolean|null>|null|undefined} payload - The payload object to render
 * @returns {string} Markdown details section with fenced code block, or empty string if no payload
 */
function renderPayloadBlock(payload) {
  if (!payload || typeof payload !== "object" || Array.isArray(payload)) {
    return "";
  }
  if (Object.keys(payload).length === 0) {
    return "";
  }

  // Always roundtrip through JSON to normalize the object (removes undefined values,
  // coerces types, and ensures the result is serializable)
  let normalized;
  try {
    normalized = JSON.parse(JSON.stringify(payload));
  } catch {
    return "";
  }

  if (!normalized || typeof normalized !== "object" || Array.isArray(normalized)) {
    return "";
  }

  const json = JSON.stringify(normalized, null, 2);

  // Verify the pretty-printed JSON round-trips cleanly.
  // Silently return empty string on failure: callers treat "" as "no payload"
  // and the earlier JSON.parse normalization makes this path unreachable in practice.
  try {
    JSON.parse(json);
  } catch {
    return "";
  }

  return `<details>\n<summary>payload</summary>\n\n\`\`\`${PAYLOAD_FENCE_INFO}\n${json}\n\`\`\`\n\n</details>`;
}

/**
 * Parse the payload object out of a body string that was previously rendered
 * with renderPayloadBlock().  Returns null when no payload block is present
 * or when the embedded JSON is malformed.
 * Handles both LF and CRLF line endings.
 *
 * @param {string} body - The body string (e.g. a GitHub comment body)
 * @returns {Record<string, string|number|boolean|null>|null} Parsed payload, or null
 */
function parsePayloadFromBody(body) {
  if (!body || typeof body !== "string") {
    return null;
  }

  // Match the fenced code block: ```json gh-aw-payload\r?\n...\r?\n```
  // \r? handles both LF and CRLF line endings
  const pattern = new RegExp("```" + PAYLOAD_FENCE_INFO.replace(/[-/\\^$*+?.()|[\]{}]/g, "\\$&") + "\\r?\\n([\\s\\S]*?)\\r?\\n```", "m");
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
  PAYLOAD_FENCE_INFO,
  renderPayloadBlock,
  parsePayloadFromBody,
};
