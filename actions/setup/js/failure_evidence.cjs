// @ts-check
"use strict";

const { readSmokeFailure } = require("./smoke_evidence_diagnostics.cjs");

// Only built-in readers may interpret untrusted artifacts. Each reader owns its
// schema, bounded catalogue, and failed-step attribution.
const evidenceReaders = [readSmokeFailure];

/**
 * Collect validated artifact context for a failed workflow step.
 * @param {string} failingStep
 * @param {string} [rootDir]
 * @returns {string}
 */
function readFailureEvidence(failingStep, rootDir = "/tmp/gh-aw") {
  return evidenceReaders.map(read => read(failingStep, rootDir)).join("");
}

module.exports = { readFailureEvidence };
