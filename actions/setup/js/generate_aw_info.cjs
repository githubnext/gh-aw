// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");
const { TMP_GH_AW_PATH } = require("./constants.cjs");
const { generateWorkflowOverview } = require("./generate_workflow_overview.cjs");
const { validateContextVariables } = require("./validate_context_variables.cjs");
const validateLockdownRequirements = require("./validate_lockdown_requirements.cjs");

/**
 * Generate aw_info.json with workflow run metadata.
 * Reads compile-time values from environment variables (GH_AW_INFO_*) and
 * runtime values from the GitHub Actions context. Validates required context
 * variables, writes to /tmp/gh-aw/aw_info.json, sets the model output, and
 * prints the agent overview in the step summary.
 *
 * @param {typeof import('@actions/core')} core - GitHub Actions core library
 * @param {object} ctx - GitHub Actions context object
 * @returns {Promise<void>}
 */
async function main(core, ctx) {
  // Validate numeric context variables before processing run info.
  // This prevents malicious payloads from hiding special text or code in numeric fields.
  await validateContextVariables(core, ctx);

  // Validate lockdown mode requirements if lockdown is explicitly enabled.
  // This fails fast if lockdown: true is set but no custom GitHub token is configured.
  validateLockdownRequirements(core);

  // Validate required context variables
  const requiredContextFields = ["runId", "runNumber", "sha", "ref", "actor", "eventName", "repo"];
  for (const field of requiredContextFields) {
    if (ctx[field] == null) {
      core.warning(`GitHub Actions context.${field} is not set`);
    }
  }

  // Parse allowed domains from JSON env var
  let allowedDomains = [];
  const allowedDomainsEnv = process.env.GH_AW_INFO_ALLOWED_DOMAINS || "[]";
  try {
    allowedDomains = JSON.parse(allowedDomainsEnv);
  } catch {
    core.warning(`Failed to parse GH_AW_INFO_ALLOWED_DOMAINS: ${allowedDomainsEnv}`);
  }

  // Build awInfo from env vars (compile-time) + context (runtime)
  /** @type {Record<string, unknown>} */
  const awInfo = {
    engine_id: process.env.GH_AW_INFO_ENGINE_ID || "",
    engine_name: process.env.GH_AW_INFO_ENGINE_NAME || "",
    model: process.env.GH_AW_INFO_MODEL || "",
    version: process.env.GH_AW_INFO_VERSION || "",
    agent_version: process.env.GH_AW_INFO_AGENT_VERSION || "",
    workflow_name: process.env.GH_AW_INFO_WORKFLOW_NAME || "",
    experimental: process.env.GH_AW_INFO_EXPERIMENTAL === "true",
    supports_tools_allowlist: process.env.GH_AW_INFO_SUPPORTS_TOOLS_ALLOWLIST === "true",
    run_id: ctx.runId,
    run_number: ctx.runNumber,
    run_attempt: process.env.GITHUB_RUN_ATTEMPT,
    repository: ctx.repo ? ctx.repo.owner + "/" + ctx.repo.repo : "",
    ref: ctx.ref,
    sha: ctx.sha,
    actor: ctx.actor,
    event_name: ctx.eventName,
    staged: process.env.GH_AW_INFO_STAGED === "true",
    allowed_domains: allowedDomains,
    firewall_enabled: process.env.GH_AW_INFO_FIREWALL_ENABLED === "true",
    awf_version: process.env.GH_AW_INFO_AWF_VERSION || "",
    awmg_version: process.env.GH_AW_INFO_AWMG_VERSION || "",
    steps: {
      firewall: process.env.GH_AW_INFO_FIREWALL_TYPE || "",
    },
    created_at: new Date().toISOString(),
  };

  // Include cli_version only when set (released builds only)
  const cliVersion = process.env.GH_AW_INFO_CLI_VERSION;
  if (cliVersion) {
    awInfo.cli_version = cliVersion;
  }

  // Write to /tmp/gh-aw directory to avoid inclusion in PR
  fs.mkdirSync(TMP_GH_AW_PATH, { recursive: true });
  const tmpPath = TMP_GH_AW_PATH + "/aw_info.json";
  fs.writeFileSync(tmpPath, JSON.stringify(awInfo, null, 2));
  core.info("Generated aw_info.json at: " + tmpPath);
  core.info(JSON.stringify(awInfo, null, 2));

  // Set model as output for reuse in other steps/jobs
  core.setOutput("model", awInfo.model);

  // Generate workflow overview and write to step summary
  await generateWorkflowOverview(core);
}

module.exports = { main };
