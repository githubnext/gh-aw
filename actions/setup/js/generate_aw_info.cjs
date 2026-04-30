// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");
const { TMP_GH_AW_PATH } = require("./constants.cjs");
const { generateWorkflowOverview } = require("./generate_workflow_overview.cjs");
const { logStagedPreviewInfo } = require("./staged_preview.cjs");
const { validateContextVariables } = require("./validate_context_variables.cjs");
const validateLockdownRequirements = require("./validate_lockdown_requirements.cjs");

/**
 * Generate aw_info.json with workflow run metadata.
 * Reads compile-time values from environment variables (GH_AW_INFO_*) and
 * runtime values from the GitHub Actions context. Validates required context
 * variables, writes to /tmp/gh-aw/aw_info.json, sets the model output, and
 * prints the agent overview in the step summary.
 *
 * SEC-005: The `target_repo` field written to aw_info.json is compile-time
 * metadata sourced from GH_AW_INFO_TARGET_REPO. It is not used for cross-repository
 * API calls in this handler; no validateTargetRepo allowlist check is required here.
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
    target_repo: process.env.GH_AW_INFO_TARGET_REPO || "",
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

  // Include deployment_state when triggered by a deployment_status event.
  // This makes the deployment state available to the agent without requiring it to
  // read the raw event payload, and is propagated to child workflows via aw_context.
  const deploymentState = ctx.payload?.deployment_status?.state;
  if (deploymentState && typeof deploymentState === "string") {
    awInfo.deployment_state = deploymentState;
  }

  // Include workflow_run_conclusion when triggered by a workflow_run event.
  // This makes the triggering run conclusion available to the agent without requiring it
  // to read the raw event payload, and is propagated to child workflows via aw_context.
  const workflowRunConclusion = ctx.payload?.workflow_run?.conclusion;
  if (workflowRunConclusion && typeof workflowRunConclusion === "string") {
    awInfo.workflow_run_conclusion = workflowRunConclusion;
  }

  // Include custom token weights when set (engine.token-weights in workflow frontmatter).
  // Deep structure validation is intentionally minimal here: the JSON schema and Go parser
  // already validate the structure at compile time. We only verify the top-level type to
  // guard against unexpected env-var values at runtime.
  const tokenWeightsEnv = process.env.GH_AW_INFO_TOKEN_WEIGHTS;
  if (tokenWeightsEnv) {
    try {
      const tokenWeights = JSON.parse(tokenWeightsEnv);
      if (tokenWeights !== null && typeof tokenWeights === "object" && !Array.isArray(tokenWeights)) {
        awInfo.token_weights = tokenWeights;
      } else {
        core.warning(`GH_AW_INFO_TOKEN_WEIGHTS must be a JSON object, ignoring`);
      }
    } catch {
      core.warning(`Failed to parse GH_AW_INFO_TOKEN_WEIGHTS: ${tokenWeightsEnv}`);
    }
  }

  // Include aw_context when the workflow was triggered via workflow_dispatch with
  // the aw_context input set by a calling agentic workflow's dispatch_workflow handler.
  // Validates JSON format and structure before populating the context key in aw_info.json.
  const awContextRaw = ctx.payload?.inputs?.aw_context;
  if (awContextRaw && typeof awContextRaw === "string" && awContextRaw.trim() !== "") {
    try {
      const parsed = JSON.parse(awContextRaw);

      // Validate: must be a plain non-null object (not an array or primitive)
      if (parsed === null || typeof parsed !== "object" || Array.isArray(parsed)) {
        core.warning(`aw_context must be a JSON object, got: ${typeof parsed}`);
      } else {
        // Validate: no nested objects (all values must be primitives)
        const nestedKeys = Object.entries(parsed)
          .filter(([, v]) => v !== null && typeof v === "object")
          .map(([k]) => k);
        if (nestedKeys.length > 0) {
          core.warning(`aw_context contains nested objects for keys: ${nestedKeys.join(", ")}. Ignoring aw_context.`);
        } else {
          // Validate: required fields must be present
          const requiredFields = ["run_id", "repo", "workflow_id"];
          const missingFields = requiredFields.filter(f => !(f in parsed));
          if (missingFields.length > 0) {
            core.warning(`aw_context is missing required fields: ${missingFields.join(", ")}. Ignoring aw_context.`);
          } else {
            awInfo.context = parsed;
          }
        }
      }
    } catch {
      core.warning(`Failed to parse aw_context input as JSON: ${awContextRaw}`);
    }
  }

  // Write to /tmp/gh-aw directory to avoid inclusion in PR
  fs.mkdirSync(TMP_GH_AW_PATH, { recursive: true });
  const tmpPath = TMP_GH_AW_PATH + "/aw_info.json";
  fs.writeFileSync(tmpPath, JSON.stringify(awInfo, null, 2));

  if (awInfo.staged) {
    logStagedPreviewInfo("Generating workflow info in staged mode — no changes applied");
  }

  core.info("Generated aw_info.json at: " + tmpPath);
  core.info(JSON.stringify(awInfo, null, 2));

  // Set model and engine_id as outputs for reuse in other steps/jobs
  core.setOutput("model", awInfo.model);
  core.setOutput("engine_id", awInfo.engine_id);

  // Generate workflow overview and write to step summary
  await generateWorkflowOverview(core);
}

module.exports = { main };
