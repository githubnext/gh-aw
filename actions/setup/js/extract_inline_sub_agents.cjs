// @ts-check
/// <reference types="@actions/github-script" />

// extract_inline_sub_agents.cjs
//
// Parses ## agent: `name` markers from workflow markdown and writes each agent
// block as a separate .agent.md file under .github/agents/ (Copilot) or the
// engine-appropriate directory.
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
// Supported frontmatter fields (all others are stripped with a warning)
// ─────────────────────────────────────────────────────────────────────
//   description   Human-readable description of the sub-agent's role.
//   model         AI model to use.  Default is "inherited" (uses the parent
//                 workflow's model when not set).
//
// If no ## agent: markers are present the content is returned unchanged and no
// files are written.

const fs = require("fs");
const path = require("path");

// Supported frontmatter fields for inline sub-agents.
// Any other field is stripped with a warning.
const SUPPORTED_FRONTMATTER_FIELDS = ["description", "model"];

// Regex for the start marker: ## agent: `name` (lowercase identifier)
const START_MARKER_RE = /^##[ \t]+agent:[ \t]+`([a-z][a-z0-9_-]*)`[ \t]*$/gm;

// Regex that matches the start of any level-2 Markdown heading (## ).
// Used to find the boundary where each agent block ends.
const H2_HEADING_RE = /^##[ \t]/gm;

/**
 * Filters sub-agent frontmatter to only retain supported fields.
 *
 * Only `description` and `model` are valid fields in a sub-agent frontmatter
 * block.  Any other top-level key is stripped and a warning is emitted.
 * If `model` is not present its implicit default is "inherited" (the sub-agent
 * uses the parent workflow's model), but the key is NOT written unless the
 * workflow author explicitly sets it.
 *
 * When no YAML frontmatter delimiter (`---`) is found at the start of the
 * content, the content is returned unchanged.
 *
 * @param {string} content   - Raw agent block content (frontmatter + prompt).
 * @param {string} agentName - Agent name used in log messages.
 * @returns {string} Content with only supported frontmatter fields retained.
 */
function filterSubAgentFrontmatter(content, agentName) {
  // A YAML frontmatter block must start immediately at the beginning of the
  // content (after trimming performed by the caller).
  if (!content.startsWith("---\n")) {
    return content;
  }

  // Locate the closing delimiter.  We search for "\n---" starting after the
  // complete opening "---\n" (offset 4) to avoid matching the opening itself.
  const closeIdx = content.indexOf("\n---", 4);
  if (closeIdx === -1) {
    return content;
  }

  // Lines between the opening and closing "---".
  const fmLines = content.slice(4, closeIdx).split("\n");
  // Everything after the closing "\n---" (including the optional newline).
  const body = content.slice(closeIdx + 4);

  const kept = [];
  const stripped = [];

  for (const line of fmLines) {
    // Match a simple scalar YAML key at the start of the line.
    // YAML keys for description and model are plain identifiers (no hyphens).
    const keyMatch = line.match(/^([a-zA-Z_][a-zA-Z0-9_]*)[ \t]*:/);
    if (keyMatch) {
      const key = keyMatch[1];
      if (SUPPORTED_FRONTMATTER_FIELDS.includes(key)) {
        kept.push(line);
      } else {
        stripped.push(key);
      }
    } else {
      // Continuation / comment / blank line — keep only when at least one
      // supported key has already been accepted, so multi-line values (e.g.
      // `description: |`) are preserved correctly.
      if (kept.length > 0) {
        kept.push(line);
      }
    }
  }

  if (stripped.length > 0) {
    core.warning(`[extractInlineSubAgents] sub-agent "${agentName}": unsupported frontmatter field(s) stripped: ${stripped.join(", ")} (only "description" and "model" are supported)`);
  }

  // If no supported fields remain, omit the frontmatter block entirely.
  if (kept.length === 0) {
    return body.replace(/^\n/, "");
  }

  return `---\n${kept.join("\n")}\n---${body}`;
}

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
 * Returns the target directory (relative to agentsBaseDir) and filename extension
 * for inline sub-agent files based on the engine ID.
 *
 * Each AI engine stores its sub-agent definitions in a different location:
 *   claude   → .claude/agents/<name>.md
 *   codex    → .codex/agents/<name>.md
 *   gemini   → .gemini/agents/<name>.md
 *   copilot  → .github/agents/<name>.agent.md  (default)
 *   others   → .github/agents/<name>.agent.md  (fallback)
 *
 * @param {string} [engineId] - The engine identifier (e.g. "claude", "copilot").
 * @returns {{ dir: string, ext: string }}
 */
function getEngineSubAgentTarget(engineId) {
  switch ((engineId || "").toLowerCase()) {
    case "claude":
      return { dir: ".claude/agents", ext: ".md" };
    case "codex":
      return { dir: ".codex/agents", ext: ".md" };
    case "gemini":
      return { dir: ".gemini/agents", ext: ".md" };
    default:
      return { dir: ".github/agents", ext: ".agent.md" };
  }
}

/**
 * Extracts inline sub-agents from content and writes each one to the
 * engine-appropriate location under agentsBaseDir.
 *
 * The target directory and filename extension are determined by engineId:
 *   - claude  → <base>/.claude/agents/<name>.md
 *   - codex   → <base>/.codex/agents/<name>.md
 *   - gemini  → <base>/.gemini/agents/<name>.md
 *   - default → <base>/.github/agents/<name>.agent.md
 *
 * Returns the main content (before the first ## agent: marker) after stripping
 * all agent blocks.  When no agent markers are found the original content is
 * returned unchanged.
 *
 * Agent files are written relative to `agentsBaseDir` (defaults to `workspaceDir`).
 * Pass the gh-aw tmp directory (`/tmp/gh-aw`) as `agentsBaseDir` in production so
 * the files land under `/tmp/gh-aw/<engine-dir>/` — which is included in the
 * activation artifact and therefore available to the downstream agent job.
 *
 * @param {string} content - Markdown with potential inline sub-agent blocks.
 * @param {string} workspaceDir - GITHUB_WORKSPACE (repository root).
 * @param {string} [agentsBaseDir] - Root directory for agent output.
 *   Defaults to `workspaceDir` when omitted (for tests and legacy callers).
 * @param {string} [engineId] - The engine ID (e.g. "claude", "copilot").
 *   Defaults to "copilot" behavior when omitted.
 * @returns {string} Main content with sub-agent sections removed.
 */
function writeInlineSubAgents(content, workspaceDir, agentsBaseDir, engineId) {
  const { mainContent, agents } = extractInlineSubAgents(content);

  if (agents.length === 0) {
    return content;
  }

  const baseDir = agentsBaseDir || workspaceDir;
  const { dir, ext } = getEngineSubAgentTarget(engineId);
  const agentsDir = path.join(baseDir, dir);
  core.info(`[extractInlineSubAgents] Engine: "${engineId || "(default)"}" → dir="${dir}" ext="${ext}"`);
  core.info(`[extractInlineSubAgents] Writing ${agents.length} sub-agent(s) to: ${agentsDir}`);
  fs.mkdirSync(agentsDir, { recursive: true });

  for (const agent of agents) {
    const agentPath = path.join(agentsDir, agent.name + ext);
    const filteredContent = filterSubAgentFrontmatter(agent.content, agent.name);
    const agentContent = filteredContent.endsWith("\n") ? filteredContent : filteredContent + "\n";
    fs.writeFileSync(agentPath, agentContent, "utf8");
    core.info(`[extractInlineSubAgents] Written sub-agent: ${agentPath} (${agentContent.length} bytes)`);
  }

  core.info(`[extractInlineSubAgents] Done — ${agents.length} file(s) written to ${agentsDir}`);
  return mainContent;
}

module.exports = { extractInlineSubAgents, writeInlineSubAgents, getEngineSubAgentTarget, filterSubAgentFrontmatter };
