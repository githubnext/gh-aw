// @ts-check
/// <reference types="@actions/github-script" />

// extract_inline_sub_agents.cjs
//
// Parses ## agent: name / ## end: name markers from workflow markdown and
// writes each agent block as a separate .md file under .github/agents/.
//
// This step runs AFTER {{#runtime-import}} macros have been fully inlined by
// processRuntimeImports() in interpolate_prompt.cjs, ensuring that any imports
// inside an agent block are resolved before the agent file is written.
//
// Marker syntax
// ─────────────
//   ## agent: name          Opens an agent block.  name must start with a
//                           letter and contain only alphanumeric chars, hyphens,
//                           or underscores (safe for filenames).
//
//   ## end: name            Optionally closes the named agent block.  When the
//                           name matches the currently-open agent the block is
//                           terminated; content after the marker is excluded
//                           from the agent file.  A mismatched name or an end
//                           marker with no open agent is treated as plain text.
//
// If no ## agent: markers are present the content is returned unchanged and no
// files are written.

const fs = require("fs");
const path = require("path");

// Regex for the start marker: ## agent: name
const START_MARKER_RE = /^##[ \t]+agent:[ \t]+([a-zA-Z][a-zA-Z0-9_-]*)[ \t]*$/gm;

// Regex for the optional end marker: ## end: name
const END_MARKER_RE = /^##[ \t]+end:[ \t]+([a-zA-Z][a-zA-Z0-9_-]*)[ \t]*$/gm;

/**
 * Extracts inline sub-agents from markdown content.
 *
 * Returns the main content (everything before the first ## agent: marker, with
 * trailing newlines stripped) and an array of extracted agents.
 *
 * @param {string} content - Markdown with potential inline sub-agent blocks.
 * @returns {{ mainContent: string, agents: Array<{name: string, content: string}> }}
 */
function extractInlineSubAgents(content) {
  /** @type {Array<{kind: "start"|"end", name: string, lineStart: number, lineEnd: number}>} */
  const markers = [];

  for (const m of content.matchAll(START_MARKER_RE)) {
    if (m.index === undefined) continue;
    let lineEnd = m.index + m[0].length;
    if (lineEnd < content.length && content[lineEnd] === "\n") lineEnd++;
    markers.push({ kind: "start", name: m[1], lineStart: m.index, lineEnd });
  }

  for (const m of content.matchAll(END_MARKER_RE)) {
    if (m.index === undefined) continue;
    let lineEnd = m.index + m[0].length;
    if (lineEnd < content.length && content[lineEnd] === "\n") lineEnd++;
    markers.push({ kind: "end", name: m[1], lineStart: m.index, lineEnd });
  }

  // Sort all markers by their position in the document.
  markers.sort((a, b) => a.lineStart - b.lineStart);

  // Find the first start marker.
  const firstStartIdx = markers.findIndex(m => m.kind === "start");
  if (firstStartIdx === -1) {
    return { mainContent: content, agents: [] };
  }

  // Main content is everything before the first start marker (trailing newlines stripped).
  const mainContent = content.slice(0, markers[firstStartIdx].lineStart).replace(/\n+$/, "");

  /** @type {Array<{name: string, content: string}>} */
  const agents = [];
  let currentName = /** @type {string | null} */ null;
  let contentStart = 0;

  for (let i = firstStartIdx; i < markers.length; i++) {
    const m = markers[i];

    if (m.kind === "start") {
      // Close any currently open agent.
      if (currentName !== null) {
        agents.push({ name: currentName, content: content.slice(contentStart, m.lineStart).trim() });
      }
      // Open the new agent.
      currentName = m.name;
      contentStart = m.lineEnd;
    } else if (m.kind === "end" && m.name === currentName) {
      // Matching end marker — close the agent.
      agents.push({ name: currentName, content: content.slice(contentStart, m.lineStart).trim() });
      currentName = null;
    }
    // Mismatched end markers (name doesn't match open agent) are plain text — no action.
  }

  // Close any agent still open at EOF.
  if (currentName !== null) {
    agents.push({ name: currentName, content: content.slice(contentStart).trim() });
  }

  return { mainContent, agents };
}

/**
 * Extracts inline sub-agents from content and writes each one to
 * <workspaceDir>/.github/agents/<name>.md.
 *
 * Returns the main content (before the first ## agent: marker) after stripping
 * all agent blocks.  When no agent markers are found the original content is
 * returned unchanged.
 *
 * @param {string} content - Markdown with potential inline sub-agent blocks.
 * @param {string} workspaceDir - GITHUB_WORKSPACE (repository root).
 * @returns {string} Main content with sub-agent sections removed.
 */
function writeInlineSubAgents(content, workspaceDir) {
  const { mainContent, agents } = extractInlineSubAgents(content);

  if (agents.length === 0) {
    return content;
  }

  const agentsDir = path.join(workspaceDir, ".github", "agents");
  fs.mkdirSync(agentsDir, { recursive: true });

  for (const agent of agents) {
    const agentPath = path.join(agentsDir, agent.name + ".md");
    const agentContent = agent.content.endsWith("\n") ? agent.content : agent.content + "\n";
    fs.writeFileSync(agentPath, agentContent, "utf8");
    core.info(`[extractInlineSubAgents] Written sub-agent: .github/agents/${agent.name}.md`);
  }

  return mainContent;
}

module.exports = { extractInlineSubAgents, writeInlineSubAgents };
