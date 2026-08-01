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
  "> Agent-optimised prompt files for GitHub Agentic Workflows (gh-aw).",
  "> These markdown files are designed for AI agents and LLMs, not human readers.",
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
