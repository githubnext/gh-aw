#!/usr/bin/env node
/**
 * Model Tables Documentation Generator
 *
 * Reads the built-in model alias map and model multipliers from JSON data files
 * and generates an intuitively readable reference page with markdown tables.
 *
 * Usage:
 *   node scripts/generate-model-tables.js
 *
 * Inputs:
 *   pkg/cli/data/model_aliases.json     – Built-in alias → pattern mappings
 *   pkg/cli/data/model_multipliers.json – Per-model legacy Effective Token multipliers
 *
 * Output:
 *   docs/src/content/docs/reference/model-tables.md
 */

import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const ROOT = path.resolve(__dirname, "..");
const ALIASES_PATH = path.join(ROOT, "pkg/workflow/data/model_aliases.json");
const MULTIPLIERS_PATH = path.join(ROOT, "pkg/cli/data/model_multipliers.json");
const OUTPUT_PATH = path.join(ROOT, "docs/src/content/docs/reference/model-tables.md");

// ---------------------------------------------------------------------------
// Load data
// ---------------------------------------------------------------------------

/**
 * Read and parse a JSON file, with a clear error message on failure.
 * @param {string} filePath
 * @returns {any}
 */
function readJSON(filePath) {
  let raw;
  try {
    raw = fs.readFileSync(filePath, "utf-8");
  } catch (err) {
    console.error(`Error: could not read ${filePath}: ${err.message}`);
    process.exit(1);
  }
  try {
    return JSON.parse(raw);
  } catch (err) {
    console.error(`Error: invalid JSON in ${filePath}: ${err.message}`);
    process.exit(1);
  }
}

const aliasesData = readJSON(ALIASES_PATH);
const multipliersData = readJSON(MULTIPLIERS_PATH);

const aliases = aliasesData.aliases;
const multipliers = multipliersData.multipliers;
const tokenClassWeights = multipliersData.token_class_weights;
const referenceModel = multipliersData.reference_model;

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/**
 * Group aliases into "vendor" aliases (patterns using glob wildcards) and
 * "meta" aliases (patterns that reference other aliases by name, with no slash).
 */
function classifyAliases(aliasMap) {
  const vendor = [];
  const meta = [];
  for (const [alias, patterns] of Object.entries(aliasMap)) {
    const isMetaOnly = patterns.every(p => !p.includes("/"));
    if (isMetaOnly) {
      meta.push({ alias, resolves: patterns });
    } else {
      vendor.push({ alias, patterns });
    }
  }
  return { vendor, meta };
}

/**
 * Group multipliers into provider sections based on model name prefix.
 */
function groupMultipliers(mults) {
  const groups = {
    Anthropic: [],
    OpenAI: [],
    "OpenAI Reasoning": [],
    Google: [],
    Other: [],
  };

  for (const [model, value] of Object.entries(mults)) {
    if (model.startsWith("claude-")) {
      groups["Anthropic"].push({ model, value });
    } else if (/^o[0-9]/.test(model)) {
      groups["OpenAI Reasoning"].push({ model, value });
    } else if (model.startsWith("gpt-")) {
      groups["OpenAI"].push({ model, value });
    } else if (model.startsWith("gemini-")) {
      groups["Google"].push({ model, value });
    } else {
      groups["Other"].push({ model, value });
    }
  }

  // Remove empty groups
  return Object.fromEntries(Object.entries(groups).filter(([, rows]) => rows.length > 0));
}

// ---------------------------------------------------------------------------
// Markdown generators
// ---------------------------------------------------------------------------

function generateAliasTable(vendorAliases) {
  const lines = [];
  lines.push("| Alias | Fallback patterns (tried in order) |");
  lines.push("|-------|-------------------------------------|");
  for (const { alias, patterns } of vendorAliases) {
    const formattedPatterns = patterns.map(p => `\`${p}\``).join(", ");
    lines.push(`| \`${alias}\` | ${formattedPatterns} |`);
  }
  return lines.join("\n");
}

function generateMetaAliasTable(metaAliases) {
  const lines = [];
  lines.push("| Meta-alias | Expands to |");
  lines.push("|------------|------------|");
  for (const { alias, resolves } of metaAliases) {
    const formattedResolves = resolves.map(r => `\`${r}\``).join(" → ");
    lines.push(`| \`${alias}\` | ${formattedResolves} |`);
  }
  return lines.join("\n");
}

function generateMultiplierSection(groupName, rows) {
  const lines = [];
  lines.push(`### ${groupName}`);
  lines.push("");
  lines.push("| Model | Multiplier |");
  lines.push("|-------|-----------|");
  for (const { model, value } of rows) {
    lines.push(`| \`${model}\` | ${value} |`);
  }
  return lines.join("\n");
}

function generateTokenWeightsTable(weights) {
  const lines = [];
  lines.push("| Token class | Default weight |");
  lines.push("|-------------|---------------|");
  for (const [cls, weight] of Object.entries(weights)) {
    const label = cls.replace(/_/g, " ").replace(/\b\w/g, c => c.toUpperCase());
    lines.push(`| ${label} | ${weight} |`);
  }
  return lines.join("\n");
}

// ---------------------------------------------------------------------------
// Build the full document
// ---------------------------------------------------------------------------

function generateMarkdown() {
  const { vendor, meta } = classifyAliases(aliases);
  const multiplierGroups = groupMultipliers(multipliers);

  const lines = [];

  // Frontmatter
  lines.push("---");
  lines.push("title: Model Aliases & Multipliers");
  lines.push("description: Reference tables for the built-in model alias map and legacy per-model Effective Token multipliers used by GitHub Agentic Workflows.");
  lines.push("sidebar:");
  lines.push("  order: 297");
  lines.push("---");
  lines.push("");

  lines.push("This page lists the built-in model aliases and the legacy per-model Effective Token (ET) multipliers used by GitHub Agentic Workflows. AI Credits (AIC) is the primary cost metric.");
  lines.push("");

  // Approximation callout
  lines.push("> [!CAUTION]");
  lines.push("> The multiplier values shown on this page are **approximations** for legacy ET normalization only. gh-aw uses AI Credits (AIC) as the primary billing metric. Do not use ET multipliers for billing or financial calculations.");
  lines.push("");

  // -------------------------------------------------------------------------
  // Model Aliases
  // -------------------------------------------------------------------------
  lines.push("## Model Aliases");
  lines.push("");
  lines.push(
    "Model aliases let you write `engine: copilot` with a human-friendly model name such as `sonnet` or `mini`, and gh-aw resolves it to the best available concrete model at compile time. Each alias holds an ordered list of patterns; the first pattern that matches an available model wins."
  );
  lines.push("");
  lines.push("For details on the alias syntax, fallback resolution algorithm, and how to define your own aliases in workflow frontmatter, see the [Model Alias Format Specification](/gh-aw/specs/model-alias-specification/).");
  lines.push("");

  lines.push("### Vendor Aliases");
  lines.push("");
  lines.push("Vendor aliases map a short name to one or more provider-scoped glob patterns. The Copilot gateway is always tried first.");
  lines.push("");
  lines.push(generateAliasTable(vendor));
  lines.push("");

  lines.push("### Meta-Aliases");
  lines.push("");
  lines.push("Meta-aliases reference other aliases by name. They are resolved recursively until a concrete pattern is reached.");
  lines.push("");
  lines.push(generateMetaAliasTable(meta));
  lines.push("");

  // -------------------------------------------------------------------------
  // Model Multipliers
  // -------------------------------------------------------------------------
  lines.push("## Model Multipliers");
  lines.push("");
  lines.push(
    `Legacy Effective Token multipliers scale the weighted token total for each model relative to the reference model (\`${referenceModel}\`, multiplier = 1.0). A multiplier of 5.0 means that a run on that model counts as five times as many Effective Tokens as the same run on the reference model.`
  );
  lines.push("");
  lines.push("See the [AI Credits Specification](/gh-aw/specs/ai-credits-specification/) for primary billing details and the [Effective Tokens Specification](/gh-aw/specs/effective-tokens-specification/) for legacy ET formulas.");
  lines.push("");

  lines.push("### Token Class Weights");
  lines.push("");
  lines.push("Before per-model multipliers are applied, raw token counts are weighted by token class:");
  lines.push("");
  lines.push(generateTokenWeightsTable(tokenClassWeights));
  lines.push("");

  lines.push("### Per-Model Multipliers");
  lines.push("");
  for (const [groupName, rows] of Object.entries(multiplierGroups)) {
    lines.push(generateMultiplierSection(groupName, rows));
    lines.push("");
  }

  return lines.join("\n");
}

// ---------------------------------------------------------------------------
// Write output
// ---------------------------------------------------------------------------
console.log("Generating model tables documentation...");

const markdown = generateMarkdown();

const outputDir = path.dirname(OUTPUT_PATH);
if (!fs.existsSync(outputDir)) {
  fs.mkdirSync(outputDir, { recursive: true });
}

fs.writeFileSync(OUTPUT_PATH, markdown, "utf-8");
console.log(`✓ Generated: ${OUTPUT_PATH}`);
