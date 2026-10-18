"use strict";

const fs = require("fs");
const { runBuiltinGrader } = require("./trace_graders.cjs");

const input = JSON.parse(fs.readFileSync(0, "utf8"));
const grader = input.grader;
if (grader.source !== "builtin" || Object.hasOwn(grader, "script") || Object.hasOwn(grader, "run")) {
  throw new Error("custom graders are not supported");
}
process.stdout.write(`${JSON.stringify(runBuiltinGrader(grader.id, input.payload, grader), null, 2)}\n`);
