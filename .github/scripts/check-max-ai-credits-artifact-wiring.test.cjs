"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const { test } = require("node:test");
const { parseDocument } = require("../../actions/setup/js/work_queue_yaml.cjs");

const workflowBase = path.join(__dirname, "../workflows/daily-max-ai-credits-test");
const evidenceDirectory = "/tmp/gh-aw/";

function readWorkflow(extension) {
  let text = fs.readFileSync(`${workflowBase}.${extension}`, "utf8");
  if (extension === "md") {
    const frontmatter = /^---\r?\n([\s\S]*?)\r?\n---(?:\r?\n|$)/.exec(text);
    assert.ok(frontmatter, "workflow Markdown must have YAML frontmatter");
    text = frontmatter[1];
  }
  const document = parseDocument(text);
  assert.deepEqual(document.errors, [], "workflow YAML must parse without errors");
  return document.toJS();
}

function assertVerifierDownloads(workflow) {
  const job = workflow.jobs.verify_credit_limit;
  assert.equal(job.needs, "agent");
  assert.equal(job.if, "always() && !cancelled() && needs.agent.result != 'skipped'");
  assert.equal(job.permissions.actions, "read");
  const steps = job.steps;
  const verifierIndex = steps.findIndex(step => step.run === `node .github/scripts/check-max-ai-credits.cjs ${evidenceDirectory}`);
  assert.ok(verifierIndex >= 0, "verifier must read the artifact extraction directory");
  for (const name of ["agent", "info"]) {
    const downloads = steps.filter(step => step.uses?.startsWith("actions/download-artifact@") && step.with?.name === name);
    assert.equal(downloads.length, 1, `expected exactly one ${name} download`);
    const download = downloads[0];
    assert.deepEqual(download.with, { name, path: evidenceDirectory }, `${name} must use current-run, same-repository download defaults`);
    assert.ok(steps.indexOf(download) < verifierIndex, `${name} must be downloaded before verification`);
    assert.equal(download.if, undefined, `${name} download must not be skipped`);
    assert.equal(download["continue-on-error"], undefined, `${name} download failure must fail the job`);
  }
}

test("authored workflow downloads agent evidence and current-run identity before verification", () => {
  assertVerifierDownloads(readWorkflow("md"));
});

test("compiled workflow preserves the current-run artifact downloads", () => {
  const workflow = readWorkflow("lock.yml");
  assertVerifierDownloads(workflow);
  const sourceSteps = readWorkflow("md").jobs.verify_credit_limit.steps;
  for (const name of ["agent", "info"]) {
    const sourceDownload = sourceSteps.find(step => step.with?.name === name);
    const compiledDownload = workflow.jobs.verify_credit_limit.steps.find(step => step.with?.name === name);
    assert.equal(sourceDownload.uses, "actions/download-artifact@v8.0.2");
    assert.match(compiledDownload.uses, /^actions\/download-artifact@[a-f0-9]{40}$/);
    assert.deepEqual(compiledDownload.with, sourceDownload.with);
  }
});

test("info producer packages aw_info.json at the extraction root consumed by the verifier", () => {
  const workflow = readWorkflow("lock.yml");
  const producer = workflow.jobs.activation;
  const uploads = producer.steps.filter(step => step.uses?.startsWith("actions/upload-artifact@") && step.with?.name === "info");
  assert.equal(uploads.length, 1);
  const upload = uploads[0];
  assert.equal(upload.if, "success() || failure()");
  assert.equal(upload.with.path, "/tmp/gh-aw/aw_info.json");
  assert.equal(workflow.jobs.agent.needs, "activation", "identity producer must precede the agent");

  const download = workflow.jobs.verify_credit_limit.steps.find(step => step.with?.name === "info");
  assert.ok(download, "verifier must download the separate info artifact");
  // A single-file upload is archived relative to its parent directory.
  const archiveEntry = path.posix.relative(path.posix.dirname(upload.with.path), upload.with.path);
  assert.equal(archiveEntry, "aw_info.json");
  assert.equal(path.posix.join(download.with.path, archiveEntry), "/tmp/gh-aw/aw_info.json");

  const agentUpload = workflow.jobs.agent.steps.find(step => step.uses?.startsWith("actions/upload-artifact@") && step.with?.name === "agent");
  assert.ok(agentUpload);
  const agentPaths = agentUpload.with.path.trim().split(/\s+/);
  assert.ok(!agentPaths.includes(upload.with.path), "run identity is a separate artifact, not part of agent evidence");
});

test("artifact contract rejects missing, foreign-run, misplaced and late identity downloads", () => {
  for (const mutate of [
    steps =>
      steps.splice(
        steps.findIndex(step => step.with?.name === "info"),
        1
      ),
    steps => (steps.find(step => step.with?.name === "info").with["run-id"] = "another-run"),
    steps => (steps.find(step => step.with?.name === "info").with.path = "/tmp/gh-aw/info/"),
    steps =>
      steps.push(
        steps.splice(
          steps.findIndex(step => step.with?.name === "info"),
          1
        )[0]
      ),
  ]) {
    const workflow = readWorkflow("md");
    mutate(workflow.jobs.verify_credit_limit.steps);
    assert.throws(() => assertVerifierDownloads(workflow), assert.AssertionError);
  }
});
