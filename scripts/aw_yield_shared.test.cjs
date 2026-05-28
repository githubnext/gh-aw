#!/usr/bin/env node

const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("fs");
const path = require("path");
const { clamp, coerceBool, writeJson, loadJson, FinalizeError } = require("./aw_yield_shared.cjs");

test("shared helpers clamp and coerce booleans", () => {
  assert.equal(clamp(2), 1);
  assert.equal(clamp(-2), 0);
  assert.equal(coerceBool("observed"), true);
  assert.equal(coerceBool("missing"), false);
});

test("shared writeJson/loadJson roundtrip", () => {
  const dir = fs.mkdtempSync(path.join(process.cwd(), "aw-yield-shared-"));
  const file = path.join(dir, "payload.json");
  writeJson(file, { z: 1, a: { b: 2 } });
  const payload = loadJson(file, FinalizeError);
  assert.deepEqual(Object.keys(payload), ["a", "z"]);
});
