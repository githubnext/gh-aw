#!/usr/bin/env bash
set -euo pipefail

if (( $# < 1 || $# > 2 )); then
    printf 'error: usage: verify-operational-value-evaluator.sh <evaluator.sh> [fixtures.json]\n' >&2
    exit 1
fi

skill_dir=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
repo_root=$(CDPATH='' cd -- "$skill_dir/../../.." && pwd)

node - "$repo_root" "$@" <<'NODE'
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const [root, evaluator, fixtures] = process.argv.slice(2);
const { executeOperationalValueEvaluator } = require(path.join(root, "actions/setup/js/operational_value_grader.cjs"));
const source = fs.readFileSync(evaluator, "utf8");
assert.match(source, /^#!.*\bbash\b/, "evaluator requires a Bash shebang");
assert.ok(Buffer.byteLength(source) <= 65536, "evaluator exceeds 64 KiB");
const run = (session, config) => {
  assert.ok(Array.isArray(session), "fixture session must be a unified-event array");
  for (const event of session) {
    assert.ok(event && typeof event.type === "string" && event.type &&
      event.data && typeof event.data === "object" && !Array.isArray(event.data) &&
      event.provenance && typeof event.provenance === "object" && !Array.isArray(event.provenance),
      "fixture event requires type, data, and provenance");
  }
  return executeOperationalValueEvaluator(source, session, config);
};
if (!fixtures) {
  run([], {});
} else {
  const cases = JSON.parse(fs.readFileSync(fixtures, "utf8"));
  assert.ok(Array.isArray(cases) && cases.length >= 4, "fixtures require attained, missed, unavailable, and malformed cases");
  const names = cases.map(test => test.name);
  assert.equal(new Set(names).size, names.length, "fixture names must be unique");
  for (const name of ["attained", "missed", "unavailable", "malformed"]) {
    assert.ok(names.includes(name), `missing ${name} fixture`);
  }
  let metricIDs;
  const primary = new Map();
  for (const test of cases) {
    assert.deepEqual(Object.keys(test).sort(), ["config", "expected", "name", "session"], "fixture fields");
    assert.ok(typeof test.name === "string" && test.name.trim(), "fixture name is required");
    assert.ok(test.config && typeof test.config === "object" && !Array.isArray(test.config), "fixture config must be an object");
    const actual = run(test.session, test.config);
    assert.deepEqual(actual, test.expected, `${test.name}: metrics differ`);
    assert.deepEqual(run(test.session, test.config), actual, `${test.name}: nondeterministic output`);
    const ids = actual.map(metric => metric.id);
    if (metricIDs) assert.deepEqual(ids, metricIDs, `${test.name}: metric IDs or order changed`);
    metricIDs = ids;
    primary.set(test.name, actual[0].value);
  }
  assert.equal(typeof primary.get("attained"), "number");
  assert.equal(typeof primary.get("missed"), "number");
  assert.notEqual(primary.get("attained"), primary.get("missed"), "attained and missed primary values must differ");
  assert.equal(primary.get("unavailable"), null, "unavailable primary must be null");
  assert.equal(primary.get("malformed"), null, "malformed evidence primary must be null");
}
console.log(`verified ${evaluator}`);
NODE
