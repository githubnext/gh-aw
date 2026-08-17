#!/usr/bin/env node

/**
 * llms.txt Generator
 *
 * Generates llms.txt at the repository root from the agent prompt files in
 * .github/aw/*.md. Mirrors the logic in docs/src/pages/llms.txt.ts so that
 * https://github.com/<org>/gh-aw/llms.txt resolves to the same content that
 * the docs site serves at /llms.txt.
 *
 * Usage:
 *   node scripts/generate-llms-txt.js
 */

import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const ROOT = path.resolve(__dirname, "..");
const AW_DIR = path.join(ROOT, ".github", "aw");
const OUT_FILE = path.join(ROOT, "llms.txt");
const RAW_BASE = "https://raw.githubusercontent.com/github/gh-aw/main/.github/aw";

function parseFrontmatterDescription(content) {
  const match = content.match(/^---[\r\n]+([\s\S]*?)[\r\n]+---/);
  if (!match) return "";
  const descMatch = match[1].match(/^description:\s*(.+)$/m);
  return descMatch ? descMatch[1].trim() : "";
}

const files = fs
  .readdirSync(AW_DIR)
  .filter(f => f.endsWith(".md"))
  .sort();

const prompts = files.map(file => {
  const content = fs.readFileSync(path.join(AW_DIR, file), "utf-8");
  return {
    file,
    description: parseFrontmatterDescription(content),
    rawUrl: `${RAW_BASE}/${file}`,
  };
});

const lines = [
  "# GitHub Agentic Workflows",
  "",
  "GitHub Agentic Workflows (gh-aw) is a GitHub CLI extension for creating, compiling, and running AI-powered workflows in GitHub Actions.",
  "",
  "> Use this index to find the project documentation, guides, reference material, and agent-optimised prompt files.",
  "",
  "## Documentation",
  "",
  "- [Documentation home](https://github.github.com/gh-aw/): Learn about GitHub Agentic Workflows.",
  "- [Quickstart](https://github.github.com/gh-aw/setup/quick-start/): Install gh-aw and run an agentic workflow.",
  "- [Create a workflow](https://github.github.com/gh-aw/setup/creating-workflows/): Author and compile a workflow.",
  "",
  "## Guides",
  "",
  "- [Agentic authoring](https://github.github.com/gh-aw/guides/agentic-authoring/): Use agents to create workflows.",
  "- [Reusing workflows](https://github.github.com/gh-aw/guides/reusing-workflows/): Share and update reusable workflows.",
  "- [Using gh-aw at scale](https://github.github.com/gh-aw/guides/using-at-scale/): Manage agentic workflows across repositories.",
  "",
  "## Reference",
  "",
  "- [Authentication](https://github.github.com/gh-aw/reference/auth/): Configure tokens, secrets, and GitHub Apps.",
  "- [Editors](https://github.github.com/gh-aw/reference/editors/): Configure editor support for workflow authoring.",
  "- [Triggering CI](https://github.github.com/gh-aw/reference/triggering-ci/): Trigger CI safely from agentic workflows.",
  "",
  "## Agent Prompts",
  "",
  ...prompts.map(({ file, description, rawUrl }) => {
    const label = file.replace(/\.md$/, "");
    return description ? `- [${label}](${rawUrl}): ${description}` : `- [${label}](${rawUrl})`;
  }),
];

fs.writeFileSync(OUT_FILE, lines.join("\n") + "\n", "utf-8");
console.log(`Written ${OUT_FILE} (${prompts.length} prompts)`);
