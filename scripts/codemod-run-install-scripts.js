#!/usr/bin/env node

/**
 * Codemod: move top-level run-install-scripts into runtimes.node
 *
 * The run-install-scripts frontmatter field was previously supported at the
 * top level of the workflow YAML frontmatter block.  It has been moved to be
 * a child of runtimes.node, which is the only runtime that generates npm
 * install commands.
 *
 * This codemod transforms workflow .md files that use the old top-level form:
 *
 *   run-install-scripts: true
 *
 * into the new nested form:
 *
 *   runtimes:
 *     node:
 *       run-install-scripts: true
 *
 * Usage:
 *   node scripts/codemod-run-install-scripts.js [file-or-glob ...]
 *
 * When no arguments are given the script searches for all *.md files under
 * the current directory (excluding node_modules and .git).
 *
 * Exit codes:
 *   0 – success (files may have been modified)
 *   1 – one or more files could not be processed
 */

"use strict";

const fs = require("fs");
const path = require("path");

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/**
 * Collect all *.md files reachable from `dir`, excluding common ignore dirs.
 */
function findMarkdownFiles(dir, results = []) {
  const IGNORE = new Set(["node_modules", ".git", "vendor"]);
  let entries;
  try {
    entries = fs.readdirSync(dir, { withFileTypes: true });
  } catch {
    return results;
  }
  for (const entry of entries) {
    if (IGNORE.has(entry.name)) continue;
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      findMarkdownFiles(full, results);
    } else if (entry.isFile() && entry.name.endsWith(".md")) {
      results.push(full);
    }
  }
  return results;
}

/**
 * Detect and return the indentation string used in the YAML frontmatter (e.g.
 * "  " for two-space indent, "\t" for tab indent).  Defaults to two spaces.
 */
function detectIndent(frontmatter) {
  // Match the first indented line: leading tabs or spaces followed by a non-space char.
  const match = frontmatter.match(/^([ \t]+)\S/m);
  return match ? match[1] : "  ";
}

/**
 * Transform the YAML frontmatter string.
 *
 * Returns `{ changed: boolean, frontmatter: string }`.
 *
 * The transformation:
 *   1. Detects a top-level `run-install-scripts: <bool>` line.
 *   2. Removes that line.
 *   3. Merges the value into the `runtimes.node` block:
 *      - If a `runtimes:` block already exists:
 *          - If a `node:` sub-block already exists inside it, inject
 *            `run-install-scripts` into that block (idempotent if already present).
 *          - Otherwise, prepend a `node:` sub-block with the field.
 *      - Otherwise, append a new `runtimes:` block at the end.
 */
function transformFrontmatter(fm) {
  // Match exactly a top-level (no leading spaces) run-install-scripts line.
  const risRegex = /^(run-install-scripts\s*:\s*(true|false)[ \t]*)$/m;
  const risMatch = fm.match(risRegex);
  if (!risMatch) {
    return { changed: false, frontmatter: fm };
  }

  const risValue = risMatch[2].trim(); // "true" or "false"
  const indent = detectIndent(fm);

  // Remove the top-level line (and any immediately-following blank lines).
  let updated = fm.replace(/^run-install-scripts\s*:[^\n]*\n(\s*\n)*/m, "");

  // Check whether `runtimes.node.run-install-scripts` already exists so we
  // can be idempotent.  We look specifically inside the node sub-block to
  // avoid falsely matching the field under a different runtime (e.g. python).
  const nodeBlockPattern = new RegExp(
    `^[ \\t]+node\\s*:[\\s\\S]*?^[ \\t]+run-install-scripts\\s*:`,
    "m"
  );
  const alreadyPresent = nodeBlockPattern.test(updated);
  if (alreadyPresent) {
    // The per-node field is already there; we only removed the top-level line.
    return { changed: true, frontmatter: updated };
  }

  // -------------------------------------------------------------------------
  // Determine where to inject the nested field.
  // -------------------------------------------------------------------------

  // Check for an existing `runtimes:` block (top-level key, no leading spaces).
  const runtimesBlockRegex = /^(runtimes\s*:[ \t]*\n)((?:[ \t]+[^\n]*\n|[ \t]*\n)*)/m;
  const runtimesMatch = updated.match(runtimesBlockRegex);

  if (runtimesMatch) {
    // A `runtimes:` block exists.
    const runtimesHeader = runtimesMatch[1]; // e.g. "runtimes:\n"
    const runtimesBody = runtimesMatch[2];   // the indented lines that follow

    // Check for an existing `node:` sub-block within the runtimes block.
    const nodeBlockRegex = new RegExp(
      `^(${indent}node\\s*:[ \\t]*\\n)((?:${indent}${indent}[^\\n]*\\n|[ \\t]*\\n)*)`,
      "m"
    );
    const nodeMatch = runtimesBody.match(nodeBlockRegex);

    if (nodeMatch) {
      // Inject `run-install-scripts` at the start of the node block.
      const nodeHeader = nodeMatch[1];
      const nodeBody = nodeMatch[2];
      const newNodeBody =
        `${indent}${indent}run-install-scripts: ${risValue}\n` + nodeBody;
      const newRuntimesBody = runtimesBody.replace(
        nodeMatch[0],
        nodeHeader + newNodeBody
      );
      updated = updated.replace(
        runtimesMatch[0],
        runtimesHeader + newRuntimesBody
      );
    } else {
      // Prepend a new `node:` sub-block inside the existing `runtimes:` block.
      const nodeBlock =
        `${indent}node:\n${indent}${indent}run-install-scripts: ${risValue}\n`;
      const newRuntimesBody = nodeBlock + runtimesBody;
      updated = updated.replace(
        runtimesMatch[0],
        runtimesHeader + newRuntimesBody
      );
    }
  } else {
    // No `runtimes:` block exists – append one at the end of the frontmatter.
    const runtimesBlock =
      `runtimes:\n${indent}node:\n${indent}${indent}run-install-scripts: ${risValue}\n`;
    // Ensure there is a trailing newline before we append.
    if (!updated.endsWith("\n")) updated += "\n";
    updated += runtimesBlock;
  }

  return { changed: true, frontmatter: updated };
}

/**
 * Process a single file.  Returns `true` when the file was modified.
 */
function processFile(filePath) {
  let content;
  try {
    content = fs.readFileSync(filePath, "utf8");
  } catch (err) {
    console.error(`ERROR: cannot read ${filePath}: ${err.message}`);
    return false;
  }

  // Only process files that have YAML frontmatter (opening --- at line 1).
  if (!content.startsWith("---")) {
    return false;
  }

  // Locate the closing --- of the frontmatter.
  const afterOpen = content.indexOf("\n") + 1; // skip first ---\n
  const closeIdx = content.indexOf("\n---", afterOpen);
  if (closeIdx === -1) {
    // Malformed – no closing fence; skip.
    return false;
  }

  const frontmatter = content.slice(afterOpen, closeIdx + 1); // includes trailing \n
  const rest = content.slice(closeIdx + 1); // starts with \n---

  // Quick check before doing real work.
  if (!/^run-install-scripts\s*:/m.test(frontmatter)) {
    return false;
  }

  const { changed, frontmatter: newFrontmatter } = transformFrontmatter(frontmatter);
  if (!changed) {
    return false;
  }

  const newContent = "---\n" + newFrontmatter + rest;
  try {
    fs.writeFileSync(filePath, newContent, "utf8");
  } catch (err) {
    console.error(`ERROR: cannot write ${filePath}: ${err.message}`);
    return false;
  }

  return true;
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

function main() {
  const args = process.argv.slice(2);

  let files;
  if (args.length === 0) {
    console.log("No paths provided – scanning current directory for *.md files…");
    files = findMarkdownFiles(process.cwd());
  } else {
    files = args.flatMap((arg) => {
      // Accept plain file paths; glob expansion is left to the caller's shell.
      if (fs.existsSync(arg)) {
        const stat = fs.statSync(arg);
        if (stat.isDirectory()) {
          return findMarkdownFiles(arg);
        }
        return [arg];
      }
      console.warn(`WARNING: path not found: ${arg}`);
      return [];
    });
  }

  let modified = 0;
  let skipped = 0;

  for (const file of files) {
    const changed = processFile(file);
    if (changed) {
      console.log(`  updated: ${file}`);
      modified++;
    } else {
      skipped++;
    }
  }

  console.log(
    `\nDone. ${modified} file(s) updated, ${skipped} file(s) skipped.`
  );
}

main();
