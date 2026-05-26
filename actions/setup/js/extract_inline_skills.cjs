// @ts-check
/// <reference types="@actions/github-script" />

// extract_inline_skills.cjs
//
// Parses ## skill: `name` markers from workflow markdown and writes each skill
// block to the engine-appropriate skills folder.
//
// This step runs AFTER {{#runtime-import}} macros have been fully inlined by
// processRuntimeImports() in interpolate_prompt.cjs, ensuring that any imports
// inside a skill block are resolved before the skill file is written.
//
// Marker syntax
// ─────────────
//   ## skill: `name`       Opens a skill block.  name must start with a
//                          lowercase letter and contain only lowercase letters,
//                          digits, hyphens, or underscores (safe for filenames).
//
// A skill block ends at the next level-2 Markdown heading (## ...) or EOF.
// There is no explicit end marker — any H2 heading closes the skill block.
//
// Supported frontmatter fields (all others are stripped with a warning)
// ─────────────────────────────────────────────────────────────────────
//   description   Human-readable description of the skill's role.
//
// If no ## skill: markers are present the content is returned unchanged and no
// files are written.

const fs = require("fs");
const path = require("path");

// Supported frontmatter fields for inline skills.
// Any other field is stripped with a warning.
const SUPPORTED_FRONTMATTER_FIELDS = ["description"];

// Regex for the start marker: ## skill: `name` (lowercase identifier)
const START_MARKER_RE = /^##[ \t]+skill:[ \t]+`([a-z][a-z0-9_-]*)`[ \t]*$/gm;

// Regex that matches the start of any level-2 Markdown heading (## ).
// Used to find the boundary where each skill block ends.
const H2_HEADING_RE = /^##[ \t]/gm;

/**
 * Filters skill frontmatter to only retain supported fields.
 *
 * Only `description` is valid in a skill frontmatter block. Any other
 * top-level key is stripped and a warning is emitted.
 *
 * When no YAML frontmatter delimiter (`---`) is found at the start of the
 * content, the content is returned unchanged.
 *
 * @param {string} content   - Raw skill block content (frontmatter + prompt).
 * @param {string} skillName - Skill name used in log messages.
 * @returns {string} Content with only supported frontmatter fields retained.
 */
function filterInlineSkillFrontmatter(content, skillName) {
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
    // YAML keys are plain identifiers (no hyphens).
    const keyMatch = line.match(/^([a-z_][a-z0-9_]*)[ \t]*:/);
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
    core.warning(`[extractInlineSkills] skill "${skillName}": unsupported frontmatter field(s) stripped: ${stripped.join(", ")} (only "description" is supported)`);
  }

  // If no supported fields remain, omit the frontmatter block entirely.
  if (kept.length === 0) {
    return body.replace(/^\n/, "");
  }

  return `---\n${kept.join("\n")}\n---${body}`;
}

/**
 * Extracts inline skills from markdown content.
 *
 * Returns the main content (everything before the first ## skill: marker, with
 * trailing newlines stripped) and an array of extracted skills.
 *
 * A skill block extends from its start marker to the next H2 heading or EOF.
 *
 * @param {string} content - Markdown with potential inline skill blocks.
 * @returns {{ mainContent: string, skills: Array<{name: string, content: string}> }}
 */
function extractInlineSkills(content) {
  const startMatches = [...content.matchAll(START_MARKER_RE)];

  if (startMatches.length === 0) {
    return { mainContent: content, skills: [] };
  }

  // Main content is everything before the first start marker (trailing newlines stripped).
  const firstMatch = startMatches[0];
  if (firstMatch.index === undefined) {
    return { mainContent: content, skills: [] };
  }
  const mainContent = content.slice(0, firstMatch.index).replace(/\n+$/, "");

  // Collect all H2 heading positions for block boundary detection.
  const h2Positions = [...content.matchAll(H2_HEADING_RE)].map(m => m.index).filter(i => i !== undefined);

  /** @type {Array<{name: string, content: string}>} */
  const skills = [];

  for (const m of startMatches) {
    if (m.index === undefined) continue;

    const name = m[1];

    // Content starts on the line after the start marker.
    let lineEnd = m.index + m[0].length;
    if (lineEnd < content.length && content[lineEnd] === "\n") lineEnd++;

    // Content ends at the next H2 heading after the start marker line, or EOF.
    const contentEnd = h2Positions.find(pos => pos >= lineEnd) ?? content.length;

    const skillContent = content.slice(lineEnd, contentEnd).trim();
    skills.push({ name, content: skillContent });
  }

  return { mainContent, skills };
}

/**
 * Returns the target directory (relative to skillsBaseDir) and filename extension
 * for inline skill files based on the engine ID.
 *
 * Each AI engine stores its skill definitions in a different location:
 *   claude   → .claude/skills/<name>.md
 *   codex    → .codex/skills/<name>.md
 *   gemini   → .gemini/skills/<name>.md
 *   copilot  → .github/skills/<name>/SKILL.md  (default)
 *   others   → .github/skills/<name>/SKILL.md  (fallback)
 *
 * @param {string} [engineId] - The engine identifier (e.g. "claude", "copilot").
 * @returns {{ dir: string, ext: string }}
 */
function getEngineSkillTarget(engineId) {
  switch ((engineId || "").toLowerCase()) {
    case "claude":
      return { dir: ".claude/skills", ext: ".md" };
    case "codex":
      return { dir: ".codex/skills", ext: ".md" };
    case "gemini":
      return { dir: ".gemini/skills", ext: ".md" };
    default:
      return { dir: ".github/skills", ext: "/SKILL.md" };
  }
}

/**
 * Extracts inline skills from content and writes each one to the
 * engine-appropriate location under skillsBaseDir.
 *
 * The target directory and filename extension are determined by engineId:
 *   - claude  → <base>/.claude/skills/<name>.md
 *   - codex   → <base>/.codex/skills/<name>.md
 *   - gemini  → <base>/.gemini/skills/<name>.md
 *   - default → <base>/.github/skills/<name>/SKILL.md
 *
 * Returns the main content (before the first ## skill: marker) after stripping
 * all skill blocks.  When no skill markers are found the original content is
 * returned unchanged.
 *
 * Skill files are written relative to `skillsBaseDir` (defaults to `workspaceDir`).
 * Pass the gh-aw tmp directory (`/tmp/gh-aw`) as `agentsBaseDir` in production so
 * the files land under `/tmp/gh-aw/<engine-dir>/` — which is included in the
 * activation artifact and therefore available to the downstream agent job.
 *
 * @param {string} content - Markdown with potential inline skill blocks.
 * @param {string} workspaceDir - GITHUB_WORKSPACE (repository root).
 * @param {string} [skillsBaseDir] - Root directory for skill output.
 *   Defaults to `workspaceDir` when omitted (for tests and legacy callers).
 * @param {string} [engineId] - The engine ID (e.g. "claude", "copilot").
 *   Defaults to "copilot" behavior when omitted.
 * @returns {string} Main content with skill sections removed.
 */
function writeInlineSkills(content, workspaceDir, skillsBaseDir, engineId) {
  const { mainContent, skills } = extractInlineSkills(content);

  if (skills.length === 0) {
    return content;
  }

  const baseDir = skillsBaseDir || workspaceDir;
  const { dir, ext } = getEngineSkillTarget(engineId);
  const skillsDir = path.join(baseDir, dir);
  core.info(`[extractInlineSkills] Engine: "${engineId || "(default)"}" → dir="${dir}" ext="${ext}"`);
  core.info(`[extractInlineSkills] Writing ${skills.length} skill(s) to: ${skillsDir}`);
  fs.mkdirSync(skillsDir, { recursive: true });

  for (const skill of skills) {
    const skillPath = path.join(skillsDir, skill.name + ext);
    fs.mkdirSync(path.dirname(skillPath), { recursive: true });
    const filteredContent = filterInlineSkillFrontmatter(skill.content, skill.name);
    const skillContent = filteredContent.endsWith("\n") ? filteredContent : filteredContent + "\n";
    fs.writeFileSync(skillPath, skillContent, "utf8");
    core.info(`[extractInlineSkills] Written skill: ${skillPath} (${skillContent.length} bytes)`);
  }

  core.info(`[extractInlineSkills] Done — ${skills.length} file(s) written to ${skillsDir}`);
  return mainContent;
}

module.exports = { extractInlineSkills, writeInlineSkills, getEngineSkillTarget, filterInlineSkillFrontmatter };
