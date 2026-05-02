/**
 * Prompt assembly with imports.
 *
 * Prepends resolved import file contents (from imports: frontmatter key) to the
 * prompt body text. The assembled string is passed as the single user prompt to
 * the Pi AgentSession.
 *
 * Per spec §6.4: every item in the session prompt MUST come from an explicitly
 * declared origin. No ambient files (AGENTS.md, skills directories, etc.) are
 * auto-loaded. Skills must be listed individually under imports:.
 */

import type { ImportEntry } from "./types.js";

// ─── assemblePrompt ──────────────────────────────────────────────────────────

/**
 * Assemble the final prompt string from imports and the prompt body.
 *
 * Import contents are prepended in declaration order, each wrapped in a
 * fenced block with the source path as a header, so the agent can identify
 * which resource each block came from.
 *
 * @param imports - Resolved import entries (may be empty)
 * @param promptBody - Raw prompt body from prompt.txt
 * @returns Assembled prompt string
 */
export function assemblePrompt(imports: ImportEntry[], promptBody: string): string {
  if (imports.length === 0) {
    return promptBody;
  }

  const sections: string[] = [];

  for (const entry of imports) {
    sections.push(
      [
        `<!-- import: ${entry.path} -->`,
        entry.content.trimEnd(),
        `<!-- /import: ${entry.path} -->`,
      ].join("\n"),
    );
  }

  sections.push(promptBody);
  return sections.join("\n\n");
}
