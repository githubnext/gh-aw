// @ts-check
"use strict";

/**
 * Claude and Codex sub-agent model resolution before the engine CLI starts.
 *
 * Every declared sub-agent model goes through the shared "resolve configured model" step
 * (`resolveEngineModel`) in the engine's provisioned provider scope, then a live check
 * against the configured AWF `/reflect` endpoints of that provider. `/reflect` is refreshed
 * at most once for the whole batch. Each sub-agent with a model gets a `model.resolution`
 * record whose role is the agent name; `inherit` (Claude's native inheritance instruction)
 * means the main model and gets no record.
 *
 * Claude: workflow-declared inline agents fail closed and their generated workspace copy
 * gets the concrete id; repository agents are resolved and recorded, but their files are
 * never changed — a warning says when the resolved id is not applied.
 *
 * Codex: the staged Markdown agents are not loaded by Codex (it discovers custom agents only
 * from TOML role files), so their models are resolved and recorded for observability and a
 * warning says they are not applied; only a reserved name fails closed.
 *
 * The sub-agent name `main` is reserved for both engines: it is the role of the main model's
 * `model.resolution` record, which a sub-agent record with the same role would replace. A
 * workflow-declared `main` stops the engine; a repository `main` is skipped with a warning.
 */

const fs = require("fs");
const os = require("os");
const path = require("path");
const { getErrorMessage } = require("./error_helpers.cjs");
const { normalizeReflectProviderName, REFLECT_PROVIDER_ALIASES } = require("./awf_reflect.cjs");
const { isModelAvailableInReflectData } = require("./awf_model_routing.cjs");
const { MODEL_NOT_SUPPORTED_PATTERN } = require("./detect_agent_errors.cjs");
const { inlineArtifactTarget } = require("./inline_artifact_target.cjs");
const { getEngineSubAgentTarget } = require("./extract_inline_sub_agents.cjs");
const { loadModelAliasMap, ModelAliasResolutionError, recordModelResolution, resolveEngineModel, splitModelIdentifier } = require("./resolve_model_alias.cjs");

/** The `model.resolution` role of the engine's main model; no sub-agent may use it. */
const RESERVED_SUBAGENT_NAME = "main";

/**
 * Log why workflow-declared sub-agents with the reserved name stop the engine.
 * @param {"claude" | "codex"} engine
 * @param {Array<{ path: string }>} agents
 * @param {(msg: string) => void} logger
 * @returns {boolean} true when at least one agent uses the reserved name
 */
function rejectReservedDeclaredNames(engine, agents, logger) {
  for (const agent of agents) {
    logger(
      `${engine} sub-agent file ${agent.path}: the sub-agent name '${RESERVED_SUBAGENT_NAME}' is reserved for the ${engine === "claude" ? "Claude" : "Codex"} main model's resolution record. Rename the agent in the workflow. Refusing to start ${engine === "claude" ? "Claude" : "Codex"}.`
    );
  }
  return agents.length > 0;
}

/**
 * Determine whether the current run phase is threat detection.
 * @param {string | undefined | null} phase
 * @returns {boolean}
 */
function isDetectionPhase(phase) {
  return (
    String(phase || "")
      .trim()
      .toLowerCase() === "detection"
  );
}

/**
 * The `/reflect` provider names an engine provider covers.
 * @param {string} provider
 * @returns {Set<string>}
 */
function reflectProvidersFor(provider) {
  const normalized = normalizeReflectProviderName(provider);
  if (REFLECT_PROVIDER_ALIASES.github.has(normalized)) return REFLECT_PROVIDER_ALIASES.github;
  return Object.prototype.hasOwnProperty.call(REFLECT_PROVIDER_ALIASES, normalized) ? REFLECT_PROVIDER_ALIASES[normalized] : new Set([normalized]);
}

/**
 * Parse the frontmatter of a Markdown agent file.
 * @param {string} content
 * @returns {Record<string, any>}
 */
function parseAgentFrontmatter(content) {
  const { extractFrontmatterAndBody } = require("./frontmatter_hash_pure.cjs");
  const { parseDocument } = require("./work_queue_yaml.cjs");
  const { frontmatterText } = extractFrontmatterAndBody(content);
  /** @type {any} */
  const document = parseDocument(frontmatterText);
  if (!document || document.errors.length) throw new Error("cannot parse agent frontmatter");
  const frontmatter = document.toJS() || {};
  if (typeof frontmatter !== "object" || Array.isArray(frontmatter)) throw new Error("agent frontmatter must be a mapping");
  return frontmatter;
}

/**
 * Read the Markdown agent files (`*.md`) in one directory.
 * `name` is the trimmed frontmatter `name` string, or null when absent;
 * `model` is the trimmed frontmatter `model` string, when declared.
 * A directory or file that cannot be read or parsed is skipped with a warning, unless
 * `onInvalid` is given: it then receives the path and error instead, so the caller can
 * fail closed.
 * @param {string} directory
 * @param {(msg: string) => void} logger
 * @param {{ onInvalid?: (filePath: string, error: unknown) => void }} [options]
 * @returns {Array<{ fileName: string, path: string, name: string | null, model?: string }>}
 */
function listMarkdownAgentFiles(directory, logger, options = {}) {
  const invalid = options.onInvalid || ((/** @type {string} */ filePath, /** @type {unknown} */ error) => logger(`warning: cannot read custom agent ${filePath}: ${getErrorMessage(error)}`));
  let entries;
  try {
    entries = fs.readdirSync(directory, { withFileTypes: true });
  } catch (error) {
    if (!(error instanceof Error && "code" in error && error.code === "ENOENT")) invalid(directory, error);
    return [];
  }
  const agents = [];
  for (const entry of entries.sort((a, b) => a.name.localeCompare(b.name))) {
    if (!entry.isFile() || !entry.name.endsWith(".md")) continue;
    const agentPath = path.join(directory, entry.name);
    try {
      const frontmatter = parseAgentFrontmatter(fs.readFileSync(agentPath, "utf8"));
      const name = typeof frontmatter.name === "string" && frontmatter.name.trim() ? frontmatter.name.trim() : null;
      agents.push({ fileName: entry.name, path: agentPath, name, ...(typeof frontmatter.model === "string" ? { model: frontmatter.model.trim() } : {}) });
    } catch (error) {
      invalid(agentPath, error);
    }
  }
  return agents;
}

/**
 * Replace the single top-level frontmatter `model:` scalar of an agent file, including any
 * block-scalar continuation lines.
 * @param {string} content
 * @param {string} model
 * @returns {string}
 */
function replaceAgentModel(content, model) {
  const { extractFrontmatterAndBody } = require("./frontmatter_hash_pure.cjs");
  const { parseDocument } = require("./work_queue_yaml.cjs");
  const { frontmatterText } = extractFrontmatterAndBody(content);
  /** @type {any} */
  const document = parseDocument(frontmatterText);
  if (!document || document.errors.length || !document.contents || !Array.isArray(document.contents.items)) {
    throw new Error("cannot parse agent frontmatter");
  }
  const pairs = document.contents.items.filter(pair => pair?.key?.value === "model");
  if (pairs.length !== 1 || typeof pairs[0].value?.value !== "string" || !Array.isArray(pairs[0].value.range)) {
    throw new Error("expected exactly one top-level string 'model:' scalar in the frontmatter");
  }
  const frontmatterStart = content.indexOf("\n") + 1;
  const start = frontmatterStart + pairs[0].value.range[0];
  const end = frontmatterStart + pairs[0].value.range[1];
  const lineEnding = content.slice(start, end).match(/(?:\r?\n)+$/)?.[0] || "";
  return content.slice(0, start) + JSON.stringify(model) + lineEnding + content.slice(end);
}

/**
 * Why a model is not listed for the engine's providers: `catalog_unavailable` when
 * discovery is missing or incomplete for those providers, otherwise `model_unavailable`.
 * @param {any} reflectData
 * @param {Set<string>} providers
 * @returns {"catalog_unavailable" | "model_unavailable"}
 */
function classifyUnlistedModel(reflectData, providers) {
  const configured = (Array.isArray(reflectData?.endpoints) ? reflectData.endpoints : []).filter(endpoint => endpoint?.configured === true);
  const relevant = configured.filter(endpoint => providers.has(normalizeReflectProviderName(endpoint.provider)));
  const incomplete = !reflectData || reflectData.models_fetch_complete === false || relevant.length === 0 || relevant.some(endpoint => !Array.isArray(endpoint.models) || endpoint.models.length === 0);
  return incomplete ? "catalog_unavailable" : "model_unavailable";
}

/**
 * Resolve the models of Claude or Codex sub-agents in one batch. Agents without a model or
 * with `inherit` are skipped without a record. A `<provider>/` prefix must name the engine's
 * provider; another credential route is never opened. The returned `model` is the bare,
 * engine-normalized id; model parameters are not carried into agent definitions.
 *
 * Failures are recorded as `failed` and returned, never thrown. `fatal` agents log the shared
 * ModelAliasResolutionError diagnostic; other agents get a warning without it, so a failure
 * that does not stop the run is not classified as an unsupported-model run failure.
 *
 * @param {{
 *   engine: "claude" | "codex",
 *   agents: Array<{ name: string, model?: string, path?: string, fatal: boolean, origin: string }>,
 *   provider: string,
 *   normalize: (model: string) => string,
 *   reflectData: any,
 *   refetchReflectData?: () => Promise<any>,
 *   aliasMap?: Record<string, string[]> | null,
 *   nonFatalOutcome: string,
 *   env?: NodeJS.ProcessEnv,
 *   logger: (msg: string) => void,
 * }} options `nonFatalOutcome` says what happens to a non-fatal agent whose model fails.
 * @returns {Promise<{
 *   resolutions: Array<{ name: string, configuredModel: string, model: string, status: string }>,
 *   failures: Array<{ name: string, configuredModel: string, fatal: boolean, failure: InstanceType<typeof ModelAliasResolutionError> }>,
 * }>}
 */
async function resolveEngineSubagentModels(options) {
  const { engine, logger } = options;
  const env = options.env || process.env;
  const providers = reflectProvidersFor(options.provider);
  const withModel = options.agents.filter(agent => typeof agent.model === "string" && agent.model.trim() && agent.model.trim().toLowerCase() !== "inherit");
  /** @type {Awaited<ReturnType<typeof resolveEngineSubagentModels>>} */
  const outcome = { resolutions: [], failures: [] };
  if (withModel.length === 0) return outcome;
  const aliasMap = options.aliasMap !== undefined ? options.aliasMap : loadModelAliasMap({ reflectData: options.reflectData, env });
  let reflectData = options.reflectData;
  /** @type {Promise<any> | null} */
  let refresh = null;
  const refreshOnce = () => {
    // A failed refresh counts as no refreshed catalogue, so every agent is still resolved,
    // recorded and classified (catalog_unavailable when the catalogue stays incomplete).
    refresh ??= Promise.resolve()
      .then(() => (options.refetchReflectData ? options.refetchReflectData() : null))
      .then(
        refreshed => {
          if (refreshed) reflectData = refreshed;
          return refreshed || null;
        },
        error => {
          logger(`warning: ${engine} sub-agent model resolution: cannot refresh the awf-reflect catalogue: ${getErrorMessage(error)}`);
          return null;
        }
      );
    return refresh;
  };
  const liveCheck = env.AWF_REFLECT_ENABLED === "1" || Boolean(options.reflectData);
  const stripProvider = (/** @type {string} */ model) => {
    const { base } = splitModelIdentifier(model);
    const slash = base.indexOf("/");
    return slash > 0 && providers.has(normalizeReflectProviderName(base.slice(0, slash))) ? base.slice(slash + 1) : base;
  };

  for (const agent of withModel) {
    const configuredModel = String(agent.model).trim();
    const agentLogger = agent.fatal ? logger : (/** @type {string} */ message) => (MODEL_NOT_SUPPORTED_PATTERN.test(message) ? undefined : logger(message));
    const fail = (/** @type {InstanceType<typeof ModelAliasResolutionError>} */ failure, /** @type {string} */ detail) => {
      recordModelResolution({ engine, role: agent.name, configured_model: configuredModel, resolved_model: null, provider: failure.provider || null, status: "failed", cause: failure.reason }, env);
      outcome.failures.push({ name: agent.name, configuredModel, fatal: agent.fatal, failure });
      if (agent.fatal) {
        logger(`${engine} sub-agent '${agent.name}': ${failure.message}${detail ? ` ${detail}` : ""} Refusing to start ${engine === "claude" ? "Claude" : "Codex"} with a different sub-agent model.`);
      } else {
        logger(`warning: ${agent.origin} '${agent.name}' (${agent.path || "unknown file"}) declares model '${configuredModel}', which could not be resolved (cause: ${failure.reason}); ${options.nonFatalOutcome}.`);
      }
    };

    const { base } = splitModelIdentifier(configuredModel);
    const slash = base.indexOf("/");
    const prefix = slash > 0 ? base.slice(0, slash) : "";
    if (prefix && ![...reflectProvidersFor(prefix)].some(name => providers.has(name))) {
      fail(new ModelAliasResolutionError(configuredModel, "model_unavailable", { provider: prefix }), `Sub-agent model '${configuredModel}' names provider '${prefix}', but ${engine} is provisioned for '${options.provider}'.`);
      continue;
    }

    let result;
    try {
      result = await resolveEngineModel({
        engine,
        role: agent.name,
        configuredModel,
        aliasMap,
        reflectData,
        provider: options.provider,
        refetchReflectData: refreshOnce,
        logPrefix: `${engine} sub-agent '${agent.name}' model alias resolution`,
        normalize: resolution => stripProvider(options.normalize(resolution.resolvedModel)),
        env,
        logger: agentLogger,
      });
    } catch (error) {
      if (error instanceof ModelAliasResolutionError) {
        fail(error, "");
        continue;
      }
      fail(new ModelAliasResolutionError(configuredModel, "model_unavailable", { provider: options.provider }), getErrorMessage(error));
      continue;
    }
    const model = result.engineModel;
    const resolvedNote = model !== configuredModel ? ` (resolved to '${model}')` : "";
    if (liveCheck && !isModelAvailableInReflectData(model, reflectData, providers)) {
      const initialReason = classifyUnlistedModel(reflectData, providers);
      const refreshed = await refreshOnce();
      if (!refreshed || !isModelAvailableInReflectData(model, refreshed, providers)) {
        // A complete catalogue without the model already proves model_unavailable; only an
        // incomplete one is reclassified, and only when a refreshed catalogue exists.
        const reason = initialReason === "catalog_unavailable" && refreshed ? classifyUnlistedModel(refreshed, providers) : initialReason;
        fail(new ModelAliasResolutionError(configuredModel, reason, { provider: result.provider || options.provider }), `Sub-agent model '${configuredModel}'${resolvedNote} is not listed on the configured awf-reflect endpoints.`);
        continue;
      }
    }
    const { params } = splitModelIdentifier(configuredModel);
    if ([...params.keys()].length > 0) logger(`${engine} sub-agent '${agent.name}': model parameters in '${configuredModel}' are not applied to the agent definition`);
    logger(`${engine} sub-agent '${agent.name}': model '${configuredModel}' -> '${model}'`);
    outcome.resolutions.push({ name: agent.name, configuredModel, model, status: result.status });
  }
  return outcome;
}

/**
 * Claude: resolve the models of the sub-agents Claude Code loads from the project before it
 * starts. Workflow-declared inline agents are the files staged by the activation job in
 * `<agentsBaseDir>/.claude/agents`; their generated workspace copy (restored by
 * `restore_inline_sub_agents.sh`) is rewritten with the resolved concrete id. When one of them
 * cannot be read, parsed, resolved or written, `ok` is false and Claude must not start. Repository agents
 * in `<workingDirectory>/.claude/agents` are resolved and recorded, but their files are left
 * unchanged: Claude Code ranks project agents above user agents, so no file-based override
 * exists without editing tracked content. Agent files without a `name` are skipped because
 * Claude Code does not register them.
 *
 * With `engine.bare: true`, Claude loads the workflow agents through the temporary plugin in
 * `pluginDir`, whose `agents` entry links to the staged directory. That link is replaced by a
 * run-local copy of the staged definitions carrying the resolved ids, so the staged files and
 * repository files stay unchanged and the copy is removed with the plugin.
 *
 * @param {{
 *   reflectData: any,
 *   refetchReflectData?: () => Promise<any>,
 *   provider: string,
 *   pluginDir?: string,
 *   workspaceDir?: string,
 *   workingDirectory?: string,
 *   agentsBaseDir?: string,
 *   aliasMap?: Record<string, string[]> | null,
 *   env?: NodeJS.ProcessEnv,
 *   logger: (msg: string) => void,
 * }} options
 * @returns {Promise<{ ok: boolean, resolutions: Array<{ name: string, configuredModel: string, model: string, status: string }> }>}
 */
async function applyClaudeSubagentModels(options) {
  const { logger } = options;
  const env = options.env || process.env;
  const { normalizeClaudeModel } = require("./model_fallback.cjs");
  const workspaceDir = options.workspaceDir || env.GITHUB_WORKSPACE || process.cwd();
  const workingDirectory = options.workingDirectory || path.resolve(workspaceDir, env.GH_AW_ENGINE_CWD || ".");
  const agentsBaseDir = options.agentsBaseDir || path.join(os.tmpdir(), "gh-aw");
  const target = inlineArtifactTarget("SUB_AGENT", getEngineSubAgentTarget("claude"));
  /** @type {string[]} */
  const invalidDeclared = [];
  const stagedDir = path.resolve(agentsBaseDir, target.dir);
  const declared = listMarkdownAgentFiles(stagedDir, logger, {
    onInvalid: (filePath, error) => {
      invalidDeclared.push(filePath);
      logger(`claude sub-agent file ${filePath}: cannot read or parse the staged workflow sub-agent: ${getErrorMessage(error)}.`);
    },
  }).map(agent => ({
    ...agent,
    name: agent.name || agent.fileName.slice(0, -".md".length),
    workspacePath: path.resolve(workspaceDir, target.dir, agent.fileName),
  }));
  if (invalidDeclared.length > 0) {
    logger(`Refusing to start Claude: ${invalidDeclared.length} workflow-declared sub-agent file(s) cannot be read or parsed, so their models cannot be resolved.`);
    return { ok: false, resolutions: [] };
  }
  if (
    rejectReservedDeclaredNames(
      "claude",
      declared.filter(agent => agent.name === RESERVED_SUBAGENT_NAME),
      logger
    )
  ) {
    return { ok: false, resolutions: [] };
  }
  const declaredPaths = new Set(declared.map(agent => agent.workspacePath));
  const declaredNames = new Set(declared.map(agent => agent.name));
  const repository = [];
  for (const agent of listMarkdownAgentFiles(path.resolve(workingDirectory, ".claude", "agents"), logger)) {
    if (declaredPaths.has(path.resolve(agent.path))) continue;
    if (!agent.name) {
      if (agent.model) logger(`warning: repository agent ${agent.path} has no frontmatter 'name', so Claude Code does not register it; its model '${agent.model}' is not resolved.`);
      continue;
    }
    if (agent.name === RESERVED_SUBAGENT_NAME) {
      logger(`warning: repository agent ${agent.path} uses the reserved name '${RESERVED_SUBAGENT_NAME}' (the role of the Claude main model's resolution record); its model is not resolved or recorded.`);
      continue;
    }
    if (declaredNames.has(agent.name)) continue;
    repository.push({ ...agent, name: agent.name });
  }
  const { resolutions, failures } = await resolveEngineSubagentModels({
    engine: "claude",
    agents: [
      ...declared.map(agent => ({ name: agent.name, model: agent.model, path: agent.path, fatal: true, origin: "workflow sub-agent" })),
      ...repository.map(agent => ({ name: agent.name, model: agent.model, path: agent.path, fatal: false, origin: "repository agent" })),
    ],
    provider: options.provider,
    normalize: model => normalizeClaudeModel(model, options.provider, env, { reflectData: options.reflectData, logger }),
    reflectData: options.reflectData,
    refetchReflectData: options.refetchReflectData,
    aliasMap: options.aliasMap,
    nonFatalOutcome: "Claude Code uses the agent file unchanged",
    env,
    logger,
  });
  if (failures.some(failure => failure.fatal)) return { ok: false, resolutions };

  /** @type {Map<string, string>} */
  const resolvedDefinitions = new Map();
  for (const resolution of resolutions) {
    const agent = declared.find(candidate => candidate.name === resolution.name);
    if (!agent) {
      const repositoryAgent = repository.find(candidate => candidate.name === resolution.name);
      if (resolution.model !== resolution.configuredModel) {
        logger(
          `warning: repository agent '${resolution.name}' (${repositoryAgent?.path || "unknown file"}) declares model '${resolution.configuredModel}', resolved to '${resolution.model}', but the resolved id is not applied: the agent file is left unchanged and Claude Code receives '${resolution.configuredModel}'.`
        );
      }
      continue;
    }
    try {
      const staged = fs.readFileSync(agent.path, "utf8");
      const content = resolution.model === resolution.configuredModel ? staged : replaceAgentModel(staged, resolution.model);
      resolvedDefinitions.set(agent.fileName, content);
      const existing = fs.existsSync(agent.workspacePath) ? fs.readFileSync(agent.workspacePath, "utf8") : null;
      if (existing !== content) {
        fs.mkdirSync(path.dirname(agent.workspacePath), { recursive: true });
        fs.writeFileSync(agent.workspacePath, content);
        logger(`claude sub-agent '${resolution.name}': wrote model '${resolution.model}' to ${agent.workspacePath}`);
      }
    } catch (error) {
      logger(`claude sub-agent '${resolution.name}': cannot write the generated agent copy ${agent.workspacePath}: ${getErrorMessage(error)}. Refusing to start Claude with an unresolved sub-agent model.`);
      return { ok: false, resolutions };
    }
  }
  if (options.pluginDir) {
    try {
      writePluginAgents(options.pluginDir, stagedDir, resolvedDefinitions, logger);
    } catch (error) {
      logger(`claude sub-agents: cannot write the resolved agent definitions into the bare-mode plugin ${options.pluginDir}: ${getErrorMessage(error)}. Refusing to start Claude with unresolved sub-agent models.`);
      return { ok: false, resolutions };
    }
  }
  return { ok: true, resolutions };
}

/**
 * Replace the bare-mode plugin's `agents` link to the staged directory with a run-local copy
 * of the staged definitions, using the resolved content where a model was resolved. Nothing
 * happens when the plugin has no `agents` entry.
 * @param {string} pluginDir
 * @param {string} stagedDir
 * @param {Map<string, string>} resolvedDefinitions file name -> resolved definition
 * @param {(msg: string) => void} logger
 */
function writePluginAgents(pluginDir, stagedDir, resolvedDefinitions, logger) {
  const pluginAgents = path.join(pluginDir, "agents");
  try {
    fs.lstatSync(pluginAgents);
  } catch (error) {
    if (error instanceof Error && "code" in error && error.code === "ENOENT") return;
    throw error;
  }
  /** @type {string | undefined} */
  let copy;
  try {
    copy = fs.mkdtempSync(path.join(pluginDir, ".agents-"));
    fs.cpSync(stagedDir, copy, { recursive: true, dereference: true });
    for (const [fileName, content] of resolvedDefinitions) fs.writeFileSync(path.join(copy, fileName), content, { mode: 0o600 });
    fs.rmSync(pluginAgents, { recursive: true, force: true });
    fs.renameSync(copy, pluginAgents);
  } catch (error) {
    try {
      if (copy) fs.rmSync(copy, { recursive: true, force: true });
    } catch (cleanupError) {
      logger(`warning: cannot remove the partial bare-mode agents copy ${copy}: ${getErrorMessage(cleanupError)}; the plugin directory is removed with the plugin.`);
    }
    throw error;
  }
  logger(`claude sub-agents: the bare-mode plugin loads a run-local copy of the resolved definitions from ${pluginAgents}`);
}

/**
 * Codex: declare and warn only. Codex discovers custom agents only from TOML role files
 * (verified for Codex 0.163.0), so the workflow-declared Markdown agents staged in
 * `<agentsBaseDir>/.codex/agents` are not loaded and their models are not applied. Their
 * models are still resolved and recorded so audits see the declarations; a model failure is
 * a warning and never stops Codex. An agent named `main` makes `ok` false, because its
 * record would replace the main model's record; Codex must then not start.
 *
 * @param {{
 *   reflectData: any,
 *   refetchReflectData?: () => Promise<any>,
 *   provider: string,
 *   agentsBaseDir?: string,
 *   aliasMap?: Record<string, string[]> | null,
 *   env?: NodeJS.ProcessEnv,
 *   logger: (msg: string) => void,
 * }} options
 * @returns {Promise<{ ok: boolean, agents: string[], resolutions: Array<{ name: string, configuredModel: string, model: string, status: string }>, failures: Array<{ name: string }> }>}
 */
async function declareCodexSubagentModels(options) {
  const { logger } = options;
  const env = options.env || process.env;
  const { normalizeCodexModel } = require("./model_fallback.cjs");
  const agentsBaseDir = options.agentsBaseDir || path.join(os.tmpdir(), "gh-aw");
  const directory = path.resolve(agentsBaseDir, inlineArtifactTarget("SUB_AGENT", getEngineSubAgentTarget("codex")).dir);
  const agents = listMarkdownAgentFiles(directory, logger).map(agent => ({ ...agent, name: agent.name || agent.fileName.slice(0, -".md".length) }));
  if (agents.length === 0) return { ok: true, agents: [], resolutions: [], failures: [] };
  if (
    rejectReservedDeclaredNames(
      "codex",
      agents.filter(agent => agent.name === RESERVED_SUBAGENT_NAME),
      logger
    )
  ) {
    return { ok: false, agents: agents.map(agent => agent.name), resolutions: [], failures: [] };
  }
  logger(
    `warning: Codex discovers custom agents only from TOML role files (verified for Codex 0.163.0); the ${agents.length} workflow-declared Markdown sub-agent(s) in ${directory} (${agents.map(agent => agent.name).join(", ")}) are not loaded by Codex, and their declared models are recorded but not applied.`
  );
  const { resolutions, failures } = await resolveEngineSubagentModels({
    engine: "codex",
    agents: agents.map(agent => ({ name: agent.name, model: agent.model, path: agent.path, fatal: false, origin: "workflow sub-agent" })),
    provider: options.provider,
    normalize: model => normalizeCodexModel(model, options.provider, { env, logger }),
    reflectData: options.reflectData,
    refetchReflectData: options.refetchReflectData,
    aliasMap: options.aliasMap,
    nonFatalOutcome: "Codex does not load the Markdown agent, so the run continues",
    env,
    logger,
  });
  return { ok: true, agents: agents.map(agent => agent.name), resolutions, failures };
}

module.exports = {
  applyClaudeSubagentModels,
  declareCodexSubagentModels,
  isDetectionPhase,
  listMarkdownAgentFiles,
  replaceAgentModel,
  resolveEngineSubagentModels,
};
