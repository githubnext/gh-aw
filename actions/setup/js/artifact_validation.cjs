// @ts-check
"use strict";

const { prepare, finalize, readApproval } = require("./verified_artifact.cjs");
const fs = require("fs");
const path = require("path");
const { downloadArtifactValidationInput, hasCheckedArtifactImport } = require("./artifact_archive.cjs");
const { artifactError } = require("./resolve_transport_paths.cjs");

/** Download and prepare in one invocation: checked-import provenance is process-local. */
/** @param {any} [options] */
async function prepareArtifactValidation(options = {}) {
  const importOptions = {
    stagingRoot: process.env.GH_AW_ARTIFACT_STAGING_ROOT || "/tmp/gh-aw/artifact-validation/input",
    preparedRoot: process.env.GH_AW_ARTIFACT_PREPARED_ROOT || "/tmp/gh-aw/artifact-validation/prepared",
    approvedRoot: process.env.GH_AW_ARTIFACT_APPROVED_ROOT || "/tmp/gh-aw/artifact-validation/approved",
    ...options,
    discoverOutput: true,
    sameRepoOnly: true,
    rejectClaims: true,
    rejectContentReferences: true,
  };
  if (!fs.existsSync(importOptions.stagingRoot)) await downloadArtifactValidationInput(importOptions);
  if (!hasCheckedArtifactImport(importOptions.stagingRoot)) throw artifactError("artifact-validation requires checked archive import into a fresh staging root");
  return prepare(importOptions);
}

/** Compiler invokes only after every configured artifact-validation step succeeds. */
/** @param {any} [options] */
function finalizeArtifactValidation(options = {}) {
  return finalize({
    preparedRoot: process.env.GH_AW_ARTIFACT_PREPARED_ROOT || "/tmp/gh-aw/artifact-validation/prepared",
    approvedRoot: process.env.GH_AW_ARTIFACT_APPROVED_ROOT || "/tmp/gh-aw/artifact-validation/approved",
    ...options,
  });
}

/** Verify the entire package before exposing any publisher handler input. */
/** @param {any} [options] */
function initializeArtifactPublication(options = {}) {
  const coreApi = options.core;
  const approvalRoot = process.env.GH_AW_ARTIFACT_APPROVAL_ROOT;
  if (process.env.GH_AW_ARTIFACT_APPROVAL_REQUIRED !== "true" || !approvalRoot) throw artifactError("protected publication requires an approved artifact root");
  if (typeof coreApi?.setOutput !== "function" || typeof coreApi?.exportVariable !== "function") throw artifactError("publication initialization requires the trusted Actions core API");
  const { manifest, root } = readApproval({ verifyArtifacts: true });
  const outputPath = path.join(root, "agent_output.json");
  const manifestPath = path.join(root, "manifest.json");
  coreApi.exportVariable("GH_AW_AGENT_OUTPUT", outputPath);
  coreApi.exportVariable("GH_AW_ARTIFACT_MANIFEST", manifestPath);
  coreApi.exportVariable("GH_AW_RESOLVED_ARTIFACT_MANIFEST", manifestPath);
  coreApi.exportVariable("GH_AW_ARTIFACT_APPROVAL_MANIFEST", manifestPath);
  coreApi.setOutput("GH_AW_AGENT_OUTPUT", outputPath);
  return manifest;
}

/** @param {any} [options] */
async function downloadApprovedArtifact(options = {}) {
  const artifactId = options.artifactId || process.env.GH_AW_ARTIFACT_APPROVAL_ARTIFACT_ID;
  const stagingRoot = options.stagingRoot || process.env.GH_AW_ARTIFACT_APPROVAL_ROOT;
  if (!artifactId || !stagingRoot) throw artifactError("trusted approved artifact ID and extraction root are required");
  return downloadArtifactValidationInput({ ...options, artifactId, stagingRoot });
}

module.exports = { prepareArtifactValidation, finalizeArtifactValidation, initializeArtifactPublication, downloadArtifactValidationInput, downloadApprovedArtifact };
