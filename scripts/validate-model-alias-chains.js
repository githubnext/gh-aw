#!/usr/bin/env node

import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const ROOT = path.resolve(__dirname, "..");
const ALIASES_PATH = path.join(ROOT, "pkg/workflow/data/model_aliases.json");

function readAliases() {
  const raw = fs.readFileSync(ALIASES_PATH, "utf8");
  const data = JSON.parse(raw);
  if (!data || typeof data !== "object" || !data.aliases || typeof data.aliases !== "object") {
    throw new Error(`Invalid aliases JSON structure in ${ALIASES_PATH}`);
  }
  return data.aliases;
}

function aliasLabel(alias) {
  return alias === "" ? '"" (default)' : alias;
}

function validateAliasChains(aliasMap) {
  const errors = [];
  const resolving = new Set();
  const memo = new Map();

  const entriesFor = alias => (Array.isArray(aliasMap[alias]) ? aliasMap[alias] : []);

  function dfs(alias, chain) {
    if (memo.has(alias)) {
      return memo.get(alias);
    }
    if (resolving.has(alias)) {
      const start = chain.indexOf(alias);
      const cycle = [...chain.slice(start), alias];
      errors.push(`circular alias chain: ${cycle.map(aliasLabel).join(" -> ")}`);
      memo.set(alias, false);
      return false;
    }

    resolving.add(alias);
    const entries = entriesFor(alias);
    let resolvesToProviderScopedPattern = false;

    for (const entry of entries) {
      const base = String(entry).split("?", 1)[0].trim();

      if (base.includes("/")) {
        resolvesToProviderScopedPattern = true;
        continue;
      }

      if (!Object.prototype.hasOwnProperty.call(aliasMap, base)) {
        errors.push(`unresolved alias reference: ${aliasLabel(alias)} -> ${JSON.stringify(entry)}`);
        continue;
      }

      if (dfs(base, [...chain, alias])) {
        resolvesToProviderScopedPattern = true;
      }
    }

    if (!resolvesToProviderScopedPattern) {
      errors.push(`alias does not resolve to a provider-scoped model: ${aliasLabel(alias)}`);
    }

    resolving.delete(alias);
    memo.set(alias, resolvesToProviderScopedPattern);
    return resolvesToProviderScopedPattern;
  }

  for (const alias of Object.keys(aliasMap)) {
    dfs(alias, []);
  }

  return errors;
}

function main() {
  const aliases = readAliases();
  const errors = validateAliasChains(aliases);

  if (errors.length > 0) {
    console.error(`✗ Model alias chain validation failed (${errors.length} issue(s))`);
    for (const err of errors) {
      console.error(`  - ${err}`);
    }
    process.exit(1);
  }

  console.log(`✓ Model alias chains validated (${Object.keys(aliases).length} alias(es))`);
}

main();
