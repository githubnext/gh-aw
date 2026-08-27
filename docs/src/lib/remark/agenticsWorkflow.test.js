#!/usr/bin/env node
// @ts-check

import { mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import remarkAgenticsWorkflow from "./agenticsWorkflow.js";

const sourceDirectory = mkdtempSync(join(tmpdir(), "agentics-workflow-"));
const workflow = "---\ndescription: Test workflow\non: issues\n---\n\n# Test\n\nInvestigate the issue.\n";

try {
  writeFileSync(join(sourceDirectory, "test-workflow.md"), workflow);

  const directive = {
    type: "html",
    value: "<!-- agentics-workflow: test-workflow.md -->",
  };
  const tree = { type: "root", children: [directive] };

  const transform = remarkAgenticsWorkflow({ sourceDirectory });
  transform(tree);

  assertEqual(directive.type, "code", "directive becomes a code node");
  assertEqual(directive.lang, "aw", "workflow uses aw syntax highlighting");
  assertEqual(directive.meta, 'wrap title=".github/workflows/test-workflow.md"', "workflow code block has a destination filename");
  assertEqual(directive.value, workflow, "workflow content is preserved exactly");

  console.log("4 passed, 0 failed");
} finally {
  rmSync(sourceDirectory, { recursive: true, force: true });
}

function assertEqual(actual, expected, label) {
  if (actual !== expected) {
    console.error(`FAILED: ${label}`);
    console.error(`expected: ${JSON.stringify(expected)}`);
    console.error(`actual:   ${JSON.stringify(actual)}`);
    process.exitCode = 1;
    throw new Error(label);
  }

  console.log(`PASS: ${label}`);
}
