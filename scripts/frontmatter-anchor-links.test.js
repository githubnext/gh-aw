#!/usr/bin/env node

import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const schemaPath = path.join(__dirname, "../pkg/parser/schemas/main_workflow_schema.json");
const frontmatterPath = path.join(__dirname, "../docs/src/content/docs/reference/frontmatter.md");

const schema = JSON.parse(fs.readFileSync(schemaPath, "utf-8"));
const frontmatter = fs.readFileSync(frontmatterPath, "utf-8");

function slugifyHeading(heading) {
  return heading
    .trim()
    .toLowerCase()
    .replace(/[`()[\]:]/g, "")
    .replace(/[^a-z0-9 -]/g, "")
    .replace(/\s+/g, "-")
    .replace(/-+/g, "-");
}

const headingAnchors = new Set(
  Array.from(frontmatter.matchAll(/^###\s+(.+)$/gm), match => slugifyHeading(match[1])),
);

const fieldToExpectedAnchor = {
  strict: "strict-mode-strict",
  "check-for-updates": "check-for-updates",
  "run-install-scripts": "run-install-scripts",
};

let allPassed = true;

for (const [field, expectedAnchor] of Object.entries(fieldToExpectedAnchor)) {
  const description = schema.properties?.[field]?.description ?? "";
  const urlMatch = description.match(/#([a-z0-9-]+)$/i);

  if (!urlMatch) {
    console.error(`❌ FAIL: ${field} schema description is missing a documentation anchor`);
    allPassed = false;
    continue;
  }

  const actualAnchor = urlMatch[1];
  if (actualAnchor !== expectedAnchor) {
    console.error(`❌ FAIL: ${field} schema description points to #${actualAnchor}, expected #${expectedAnchor}`);
    allPassed = false;
  } else {
    console.log(`✓ PASS: ${field} schema description points to #${expectedAnchor}`);
  }

  if (!headingAnchors.has(expectedAnchor)) {
    console.error(`❌ FAIL: docs/src/content/docs/reference/frontmatter.md is missing heading anchor #${expectedAnchor}`);
    allPassed = false;
  } else {
    console.log(`✓ PASS: frontmatter.md includes heading anchor #${expectedAnchor}`);
  }
}

if (!allPassed) {
  process.exit(1);
}

console.log("✅ All frontmatter schema anchors are valid.");
