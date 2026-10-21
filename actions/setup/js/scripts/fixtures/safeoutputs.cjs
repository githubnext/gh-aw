#!/usr/bin/env node
// @ts-check

"use strict";

const fs = require("node:fs");

const [toolName, ...args] = process.argv.slice(2);
if (!["report_incomplete", "report_missing_tool", "missing_tool"].includes(toolName)) {
  throw new Error(`Unsupported test safeoutputs tool: ${toolName}`);
}
const outputPath = process.env.GH_AW_SAFE_OUTPUTS;
if (!outputPath) {
  throw new Error("Test safeoutputs requires GH_AW_SAFE_OUTPUTS");
}
/** @type {Record<string, string>} */
const item = { type: toolName };
for (let i = 0; i < args.length; i += 2) {
  const key = args[i];
  const value = args[i + 1];
  if (!/^--[a-zA-Z0-9_-]+$/.test(key) || value === undefined || key === "--type") {
    throw new Error(`Invalid test safeoutputs argument: ${key}`);
  }
  item[key.slice(2)] = value;
}
fs.appendFileSync(outputPath, JSON.stringify(item) + "\n");
