// @ts-check
/// <reference types="@actions/github-script" />

// extract_inline_sub_agents.cjs
//
// Parses ## agent: `name` markers from workflow markdown and writes each agent
// block as a separate .agent.md file under .agents/agents/.
//
// This step runs AFTER {{#runtime-import}} macros have been fully inlined by
// processRuntimeImports() in interpolate_prompt.cjs, ensuring that any imports
// inside an agent block are resolved before the agent file is written.
//
// Marker syntax
// ─────────────
//   ## agent: `name`       Opens an agent block.  name must start with a
//                          lowercase letter and contain only lowercase letters,
//                          digits, hyphens, or underscores (safe for filenames).
//
// An agent block ends at the next level-2 Markdown heading (## ...) or EOF.
// There is no explicit end marker — any H2 heading closes the agent block.
//
// If no ## agent: markers are present the content is returned unchanged and no
// files are written.

const fs = require("fs");
const path = require("path");

// Regex for the start marker: ## agent: `name` (lowercase identifier)
const START_MARKER_RE = /^##[ \t]+agent:[ \t]+`([a-z][a-z0-9_-]*)`[ \t]*$/gm;

// Regex that matches the start of any level-2 Markdown heading (## ).
// Used to find the boundary where each agent block ends.
const H2_HEADING_RE = /^##[ \t]/gm;

/**
 * Extracts inline sub-agents from markdown content.
 *
 * Returns the main content (everything before the first ## agent: marker, with
 * trailing newlines stripped) and an array of extracted agents.
 *
 * An agent block extends from its start marker to the next H2 heading or EOF.
 *
 * @param {string} content - Markdown with potential inline sub-agent blocks.
 * @returns {{ mainContent: string, agents: Array<{name: string, content: string}> }}
 */
function extractInlineSubAgents(content) {
  const startMatches = [...content.matchAll(START_MARKER_RE)];

  if (startMatches.length === 0) {
    return { mainContent: content, agents: [] };
  }

  // Main content is everything before the first start marker (trailing newlines stripped).
  const firstMatch = startMatches[0];
  if (firstMatch.index === undefined) {
    return { mainContent: content, agents: [] };
  }
  const mainContent = content.slice(0, firstMatch.index).replace(/\n+$/, "");

  // Collect all H2 heading positions for block boundary detection.
  const h2Positions = [...content.matchAll(H2_HEADING_RE)].map(m => m.index).filter(i => i !== undefined);

  /** @type {Array<{name: string, content: string}>} */
  const agents = [];

  for (const m of startMatches) {
    if (m.index === undefined) continue;

    const name = m[1];

    // Content starts on the line after the start marker.
    let lineEnd = m.index + m[0].length;
    if (lineEnd < content.length && content[lineEnd] === "\n") lineEnd++;

    // Content ends at the next H2 heading after the start marker line, or EOF.
    const contentEnd = h2Positions.find(pos => pos >= lineEnd) ?? content.length;

    const agentContent = content.slice(lineEnd, contentEnd).trim();
    agents.push({ name, content: agentContent });
  }

  return { mainContent, agents };
}

/**
 * Extracts inline sub-agents from content and writes each one to
 * <workspaceDir>/.agents/agents/<name>.agent.md.
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

  const agentsDir = path.join(workspaceDir, ".agents", "agents");
  fs.mkdirSync(agentsDir, { recursive: true });

  for (const agent of agents) {
    const agentPath = path.join(agentsDir, agent.name + ".agent.md");
    const agentContent = agent.content.endsWith("\n") ? agent.content : agent.content + "\n";
    fs.writeFileSync(agentPath, agentContent, "utf8");
    core.info(`[extractInlineSubAgents] Written sub-agent: .agents/agents/${agent.name}.agent.md`);
  }

  return mainContent;
}

module.exports = { extractInlineSubAgents, writeInlineSubAgents };
