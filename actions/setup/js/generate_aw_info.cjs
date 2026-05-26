// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");
const path = require("path");
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

  const frontmatterSource = process.env.GH_AW_INFO_FRONTMATTER_SOURCE || "";
  if (frontmatterSource) {
    awInfo.frontmatter_source = frontmatterSource;
  }

  const frontmatterEmoji = process.env.GH_AW_INFO_FRONTMATTER_EMOJI || "";
  if (frontmatterEmoji) {
    awInfo.frontmatter_emoji = frontmatterEmoji;
  }

  const bodyModified = process.env.GH_AW_INFO_BODY_MODIFIED;
  if (bodyModified === "true" || bodyModified === "false") {
    awInfo.body_modified = bodyModified === "true";
  }

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

  const tokenWeights = parseTokenWeightsFromEnv(core);
  if (tokenWeights) {
    awInfo.token_weights = tokenWeights;
  }

  // Include aw_context when the workflow was triggered by a caller that relayed
  // orchestration context via workflow inputs or repository_dispatch client payload.
  // Validates JSON format and structure before populating the context key in aw_info.json.
  const awContextRaw = ctx.payload?.inputs?.aw_context ?? ctx.payload?.client_payload?.aw_context;
  if (awContextRaw != null) {
    try {
      const parsed = typeof awContextRaw === "string" ? JSON.parse(awContextRaw) : awContextRaw;

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
      core.warning(`Failed to parse aw_context input as JSON: ${String(awContextRaw)}`);
    }
  }

  // Write to /tmp/gh-aw directory to avoid inclusion in PR
  fs.mkdirSync(TMP_GH_AW_PATH, { recursive: true });
  writeMergedModelMultipliers(core, tokenWeights);
  const tmpPath = TMP_GH_AW_PATH + "/aw_info.json";
  fs.writeFileSync(tmpPath, JSON.stringify(awInfo, null, 2));

  if (awInfo.staged) {
    logStagedPreviewInfo("Generating workflow info in staged mode — no changes applied");
  }

  /**
   * Parse optional custom token weights from GH_AW_INFO_TOKEN_WEIGHTS.
   * @param {typeof import('@actions/core')} core
   * @returns {Record<string, unknown> | null}
   */
  function parseTokenWeightsFromEnv(core) {
    const tokenWeightsEnv = process.env.GH_AW_INFO_TOKEN_WEIGHTS;
    if (!tokenWeightsEnv) {
      return null;
    }
    try {
      const tokenWeights = JSON.parse(tokenWeightsEnv);
      if (tokenWeights !== null && typeof tokenWeights === "object" && !Array.isArray(tokenWeights)) {
        return tokenWeights;
      }
      core.warning(`GH_AW_INFO_TOKEN_WEIGHTS must be a JSON object, ignoring`);
      return null;
    } catch {
      core.warning(`Failed to parse GH_AW_INFO_TOKEN_WEIGHTS: ${tokenWeightsEnv}`);
      return null;
    }
  }

  /**
   * Load and merge model multiplier data, then persist it to /tmp/gh-aw for downstream jobs.
   * Base data is read from the setup action directory; custom workflow overrides (if any)
   * are merged on top.
   * @param {typeof import('@actions/core')} core
   * @param {Record<string, unknown> | null} tokenWeights
   */
  function writeMergedModelMultipliers(core, tokenWeights) {
    const builtInPath = path.join(__dirname, "model_multipliers.json");
    /** @type {Record<string, unknown>} */
    let builtIn = {};
    if (fs.existsSync(builtInPath)) {
      try {
        const parsed = JSON.parse(fs.readFileSync(builtInPath, "utf8"));
        if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
          builtIn = parsed;
        } else {
          core.warning(`Built-in model_multipliers.json is not a JSON object: ${builtInPath}`);
        }
      } catch (error) {
        core.warning(`Failed to parse built-in model_multipliers.json: ${String(error)}`);
      }
    } else {
      core.warning(`Built-in model_multipliers.json not found at ${builtInPath}`);
    }

    const builtInTokenClassWeights = getPlainObjectOrEmpty(builtIn.token_class_weights);
    const builtInMultipliers = getPlainObjectOrEmpty(builtIn.multipliers);

    const customTokenClassWeights = getPlainObjectOrEmpty(tokenWeights?.token_class_weights);
    const customMultipliers = getPlainObjectOrEmpty(tokenWeights?.multipliers);

    const merged = {
      ...builtIn,
      token_class_weights: { ...builtInTokenClassWeights, ...customTokenClassWeights },
      multipliers: { ...builtInMultipliers, ...customMultipliers },
    };

    const mergedPath = `${TMP_GH_AW_PATH}/model_multipliers.json`;
    fs.writeFileSync(mergedPath, JSON.stringify(merged, null, 2));
    core.info(`Generated merged model multipliers at: ${mergedPath}`);
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

/**
 * @param {unknown} value
 * @returns {Record<string, unknown>}
 */
function getPlainObjectOrEmpty(value) {
  if (isPlainObject(value)) {
    return value;
  }
  return {};
}

/**
 * @param {unknown} value
 * @returns {value is Record<string, unknown>}
 */
function isPlainObject(value) {
  return Boolean(value) && typeof value === "object" && !Array.isArray(value);
}
