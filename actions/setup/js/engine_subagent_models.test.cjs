import { afterEach, describe, it, expect } from "vitest";
import { createRequire } from "module";
import crypto from "crypto";
import fs from "fs";
import os from "os";
import path from "path";

const require = createRequire(import.meta.url);
global.core = { info: () => {}, warning: () => {}, error: () => {}, setFailed: () => {} };
const { applyClaudeSubagentModels, declareCodexSubagentModels, isDetectionPhase, replaceAgentModel } = require("./engine_subagent_models.cjs");
const { extractInlineSubAgents, writeInlineSubAgents } = require("./extract_inline_sub_agents.cjs");
const { extractFrontmatterAndBody } = require("./frontmatter_hash_pure.cjs");
const { parseDocument } = require("./work_queue_yaml.cjs");
const runs = require("./fixtures/claude_codex_subagent_runs.cjs");

const repoRoot = path.resolve(path.dirname(require.resolve("./engine_subagent_models.cjs")), "../../..");
const tempDirs = [];
afterEach(() => {
  for (const dir of tempDirs.splice(0)) fs.rmSync(dir, { recursive: true, force: true });
});

const frontmatterOf = content => parseDocument(extractFrontmatterAndBody(content).frontmatterText).toJS() || {};
const sha = content => crypto.createHash("sha256").update(content).digest("hex");

/**
 * A run layout: the activation artifact under `<tmp>/gh-aw`, restored into the workspace as
 * restore_inline_sub_agents.sh does, and `GH_AW_TMP_DIR` for model-resolution records.
 */
function layout() {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "engine-subagent-models-"));
  tempDirs.push(dir);
  const agentsBaseDir = path.join(dir, "gh-aw");
  const workspaceDir = path.join(dir, "workspace");
  const env = { GH_AW_TMP_DIR: path.join(dir, "records"), GH_AW_AWF_CONFIG_PATH: path.join(dir, "missing-awf-config.json"), AWF_REFLECT_ENABLED: "1" };
  const records = () => {
    const file = path.join(env.GH_AW_TMP_DIR, "agent", "model-resolution.json");
    return fs.existsSync(file) ? JSON.parse(fs.readFileSync(file, "utf8")) : [];
  };
  const logs = [];
  const logger = message => logs.push(message);
  return { dir, agentsBaseDir, workspaceDir, env, records, logs, logger };
}

/** Stage inline agents through the activation writer and restore them into the workspace. */
function stageInline(run, engine, agents) {
  const markdown = ["# Workflow", "", "Main prompt.", ...Object.entries(agents).flatMap(([name, content]) => ["", `## agent: \`${name}\``, content.trimEnd()])].join("\n");
  writeInlineSubAgents(markdown, run.workspaceDir, run.agentsBaseDir, engine);
  const dir = engine === "claude" ? ".claude/agents" : ".codex/agents";
  fs.mkdirSync(path.join(run.workspaceDir, dir), { recursive: true });
  for (const name of fs.readdirSync(path.join(run.agentsBaseDir, dir))) fs.copyFileSync(path.join(run.agentsBaseDir, dir, name), path.join(run.workspaceDir, dir, name));
}

const typistAgent = () => fs.readFileSync(path.join(runs.typistAgentsDir, "duplicate-type-finder.md"), "utf8");
const arxivAgents = () => Object.fromEntries(fs.readdirSync(runs.arxivAgentsDir).map(name => [name.slice(0, -".md".length), fs.readFileSync(path.join(runs.arxivAgentsDir, name), "utf8")]));

describe("Claude sub-agent models (Typist run 39415087125)", () => {
  it("claude_declared_small_resolves_before_spawn", async () => {
    const run = layout();
    stageInline(run, "claude", { "duplicate-type-finder": typistAgent() });

    const outcome = await applyClaudeSubagentModels({ reflectData: runs.typistReflect(), provider: "anthropic", workspaceDir: run.workspaceDir, agentsBaseDir: run.agentsBaseDir, env: run.env, logger: run.logger });

    expect(outcome.ok).toBe(true);
    expect(outcome.resolutions).toEqual([{ name: "duplicate-type-finder", configuredModel: "small", model: "claude-haiku-5-5", status: "resolved" }]);
    const workspaceCopy = fs.readFileSync(path.join(run.workspaceDir, ".claude/agents/duplicate-type-finder.md"), "utf8");
    expect(frontmatterOf(workspaceCopy)).toMatchObject({ name: runs.typistAgentToolUse.input.subagent_type, model: "claude-haiku-5-5" });
    expect(extractFrontmatterAndBody(workspaceCopy).markdown).toBe(extractFrontmatterAndBody(typistAgent()).markdown);
    expect(frontmatterOf(fs.readFileSync(path.join(run.agentsBaseDir, ".claude/agents/duplicate-type-finder.md"), "utf8")).model).toBe("small");
    expect(run.records()).toEqual([{ engine: "claude", role: "duplicate-type-finder", configured_model: "small", resolved_model: "claude-haiku-5-5", provider: "anthropic", status: "resolved", cause: null }]);
  });

  it.each([
    // Fixture mutation: the recorded catalogue with claude-haiku-5-5 removed.
    ["a requested id absent from the catalogue", "claude-haiku-5-5", reflect => reflect.endpoints.forEach(endpoint => endpoint.models && (endpoint.models = endpoint.models.filter(model => model !== "claude-haiku-5-5")))],
    // Fixture mutation: an injected unknown selector.
    ["an unknown selector", "claude-nonexistent-9", () => {}],
  ])("claude_declared_dead_or_unknown_model_fails_before_spawn: %s", async (_name, model, mutate) => {
    const run = layout();
    stageInline(run, "claude", { "duplicate-type-finder": typistAgent().replace("model: small", `model: ${model}`) });
    const reflectData = runs.typistReflect();
    mutate(reflectData);
    let refreshes = 0;
    const before = fs.readFileSync(path.join(run.workspaceDir, ".claude/agents/duplicate-type-finder.md"), "utf8");

    const outcome = await applyClaudeSubagentModels({
      reflectData,
      refetchReflectData: async () => (refreshes++, reflectData),
      provider: "anthropic",
      workspaceDir: run.workspaceDir,
      agentsBaseDir: run.agentsBaseDir,
      env: run.env,
      logger: run.logger,
    });

    expect(outcome.ok).toBe(false);
    expect(refreshes).toBe(1);
    expect(run.records()).toEqual([{ engine: "claude", role: "duplicate-type-finder", configured_model: model, resolved_model: null, provider: "anthropic", status: "failed", cause: "model_unavailable" }]);
    expect(run.logs).toContainEqual(expect.stringContaining(`claude sub-agent 'duplicate-type-finder': The requested model is not supported: '${model}' is not available in the workflow's live model catalog.`));
    expect(run.logs).toContainEqual(expect.stringContaining("Refusing to start Claude with a different sub-agent model."));
    expect(fs.readFileSync(path.join(run.workspaceDir, ".claude/agents/duplicate-type-finder.md"), "utf8")).toBe(before);
  });

  it("records catalog_unavailable when the provider's catalogue is missing", async () => {
    const run = layout();
    stageInline(run, "claude", { "duplicate-type-finder": typistAgent().replace("model: small", "model: claude-haiku-5-5") });
    // Fixture mutation: model discovery incomplete with no Anthropic models.
    const reflectData = runs.typistReflect();
    reflectData.models_fetch_complete = false;
    reflectData.endpoints.forEach(endpoint => (endpoint.models = null));

    const outcome = await applyClaudeSubagentModels({ reflectData, provider: "anthropic", workspaceDir: run.workspaceDir, agentsBaseDir: run.agentsBaseDir, env: run.env, logger: run.logger });

    expect(outcome.ok).toBe(false);
    expect(run.records()).toEqual([expect.objectContaining({ role: "duplicate-type-finder", status: "failed", cause: "catalog_unavailable" })]);
  });

  it("claude_inherit_uses_main_without_subagent_record", async () => {
    const run = layout();
    const source = fs.readFileSync(path.join(repoRoot, ".github/workflows/agent-persona-explorer.md"), "utf8");
    const persona = extractInlineSubAgents(extractFrontmatterAndBody(source).markdown).agents.find(agent => agent.name === "persona-evaluator");
    expect(frontmatterOf(persona.content).model).toBe("inherit");
    stageInline(run, "claude", { "persona-evaluator": persona.content });
    const before = fs.readFileSync(path.join(run.workspaceDir, ".claude/agents/persona-evaluator.md"), "utf8");

    const outcome = await applyClaudeSubagentModels({ reflectData: runs.typistReflect(), provider: "anthropic", workspaceDir: run.workspaceDir, agentsBaseDir: run.agentsBaseDir, env: run.env, logger: run.logger });

    expect(outcome).toEqual({ ok: true, resolutions: [] });
    expect(run.records()).toEqual([]);
    expect(fs.readFileSync(path.join(run.workspaceDir, ".claude/agents/persona-evaluator.md"), "utf8")).toBe(before);
    expect(frontmatterOf(before).model).toBe("inherit");
  });

  it("claude_native_aliases_use_shared_resolution", async () => {
    const run = layout();
    const agent = model => typistAgent().replace("model: small", `model: ${model}`);
    stageInline(run, "claude", { "haiku-worker": agent("haiku"), "sonnet-worker": agent("sonnet"), "opus-worker": agent("opus") });

    const outcome = await applyClaudeSubagentModels({ reflectData: runs.typistReflect(), provider: "anthropic", workspaceDir: run.workspaceDir, agentsBaseDir: run.agentsBaseDir, env: run.env, logger: run.logger });

    expect(outcome.ok).toBe(true);
    expect(Object.fromEntries(outcome.resolutions.map(resolution => [resolution.name, [resolution.model, resolution.status]]))).toEqual({
      "haiku-worker": ["claude-haiku-5-5", "resolved"],
      "opus-worker": ["claude-opus-5-5", "resolved"],
      "sonnet-worker": ["claude-sonnet-5-5", "resolved"],
    });
    for (const name of ["haiku-worker", "sonnet-worker", "opus-worker"]) {
      const record = run.records().find(entry => entry.role === name);
      expect(record).toMatchObject({ engine: "claude", status: "resolved", provider: "anthropic" });
      expect(frontmatterOf(fs.readFileSync(path.join(run.workspaceDir, ".claude/agents", `${name}.md`), "utf8")).model).toBe(record.resolved_model);
    }
  });

  it("repository_unresolvable_model_warns_records_and_continues", async () => {
    const run = layout();
    stageInline(run, "claude", { "duplicate-type-finder": typistAgent() });
    // Fixture mutation: a repository agent with an unknown model.
    const repositoryAgent = "---\nname: repo-reviewer\ndescription: Reviews code\nmodel: claude-nonexistent-9\ntools: Read, Grep\n---\nReview.\n";
    fs.writeFileSync(path.join(run.workspaceDir, ".claude/agents/repo-reviewer.md"), repositoryAgent);

    const outcome = await applyClaudeSubagentModels({ reflectData: runs.typistReflect(), provider: "anthropic", workspaceDir: run.workspaceDir, agentsBaseDir: run.agentsBaseDir, env: run.env, logger: run.logger });

    expect(outcome.ok).toBe(true);
    expect(sha(fs.readFileSync(path.join(run.workspaceDir, ".claude/agents/repo-reviewer.md"), "utf8"))).toBe(sha(repositoryAgent));
    expect(run.records()).toContainEqual({ engine: "claude", role: "repo-reviewer", configured_model: "claude-nonexistent-9", resolved_model: null, provider: "anthropic", status: "failed", cause: "model_unavailable" });
    expect(run.logs).toContainEqual(expect.stringContaining("warning: repository agent 'repo-reviewer'"));
    expect(run.logs.join("\n")).not.toMatch(/The requested model is not supported/);
  });

  it("claude_repository_override_respects_native_precedence", async () => {
    const run = layout();
    const home = path.join(run.dir, "home");
    // Claude Code ranks project agents above user agents, so a user-level copy cannot override
    // a project agent; no tested session-local override exists. The project file stays as is.
    const repositoryAgent = "---\nname: repo-reviewer\ndescription: Reviews code\nmodel: small\ntools: Read, Grep\npermissionMode: plan\n---\nReview.\n";
    fs.mkdirSync(path.join(run.workspaceDir, ".claude/agents"), { recursive: true });
    fs.writeFileSync(path.join(run.workspaceDir, ".claude/agents/repo-reviewer.md"), repositoryAgent);

    const outcome = await applyClaudeSubagentModels({
      reflectData: runs.typistReflect(),
      provider: "anthropic",
      workspaceDir: run.workspaceDir,
      agentsBaseDir: run.agentsBaseDir,
      env: { ...run.env, HOME: home },
      logger: run.logger,
    });

    expect(outcome.ok).toBe(true);
    expect(outcome.resolutions).toEqual([{ name: "repo-reviewer", configuredModel: "small", model: "claude-haiku-5-5", status: "resolved" }]);
    expect(sha(fs.readFileSync(path.join(run.workspaceDir, ".claude/agents/repo-reviewer.md"), "utf8"))).toBe(sha(repositoryAgent));
    expect(fs.existsSync(path.join(home, ".claude"))).toBe(false);
    expect(run.records()).toEqual([expect.objectContaining({ role: "repo-reviewer", resolved_model: "claude-haiku-5-5", status: "resolved" })]);
    expect(run.logs).toContainEqual(expect.stringContaining("resolved to 'claude-haiku-5-5', but the resolved id is not applied"));
  });

  it("skips repository agents without a name, which Claude Code does not register", async () => {
    const run = layout();
    fs.mkdirSync(path.join(run.workspaceDir, ".claude/agents"), { recursive: true });
    fs.writeFileSync(path.join(run.workspaceDir, ".claude/agents/unnamed.md"), "---\nmodel: claude-nonexistent-9\n---\nBody\n");

    const outcome = await applyClaudeSubagentModels({ reflectData: runs.typistReflect(), provider: "anthropic", workspaceDir: run.workspaceDir, agentsBaseDir: run.agentsBaseDir, env: run.env, logger: run.logger });

    expect(outcome).toEqual({ ok: true, resolutions: [] });
    expect(run.records()).toEqual([]);
    expect(run.logs).toContainEqual(expect.stringContaining("has no frontmatter 'name'"));
  });

  it("rejects a provider prefix the engine is not provisioned for", async () => {
    const run = layout();
    stageInline(run, "claude", { "duplicate-type-finder": typistAgent().replace("model: small", "model: openai/gpt-5.4-mini") });

    const outcome = await applyClaudeSubagentModels({ reflectData: runs.typistReflect(), provider: "anthropic", workspaceDir: run.workspaceDir, agentsBaseDir: run.agentsBaseDir, env: run.env, logger: run.logger });

    expect(outcome.ok).toBe(false);
    expect(run.records()).toEqual([expect.objectContaining({ role: "duplicate-type-finder", status: "failed", cause: "model_unavailable", provider: "openai" })]);
  });

  it("replaces only the model scalar", () => {
    expect(replaceAgentModel('---\nname: "a"\nmodel: small\ntools: Read\n---\nBody model: small\n', "claude-haiku-5-5")).toBe('---\nname: "a"\nmodel: "claude-haiku-5-5"\ntools: Read\n---\nBody model: small\n');
  });
});

describe("Codex sub-agent models (daily-arxiv-researcher run 39394965127)", () => {
  it("codex_live_prefixed_model_is_exact", async () => {
    const run = layout();
    stageInline(run, "codex", arxivAgents());

    const outcome = await declareCodexSubagentModels({ reflectData: runs.arxivReflect(), provider: "openai", agentsBaseDir: run.agentsBaseDir, env: run.env, logger: run.logger });

    expect(outcome.failures).toEqual([]);
    expect(outcome.resolutions.map(resolution => [resolution.name, resolution.configuredModel, resolution.model, resolution.status])).toEqual([
      ["opportunity-extractor", "openai/gpt-5.6-sol", "gpt-5.6-sol", "exact"],
      ["paper-screener", "openai/gpt-5.4-mini", "gpt-5.4-mini", "exact"],
      ["preliminary-investigator", "openai/gpt-5.4-mini", "gpt-5.4-mini", "exact"],
      ["relevance-ranker", "openai/gpt-5.4-mini", "gpt-5.4-mini", "exact"],
    ]);
    // Same record fields as the run's recorded main-role model.resolution.
    for (const record of run.records()) {
      expect(Object.keys(record)).toEqual(Object.keys(runs.arxivMainModelResolution.data));
      expect(record).toMatchObject({ engine: "codex", provider: runs.arxivMainModelResolution.data.provider, status: "exact", cause: null });
      expect(record.configured_model).toMatch(/^openai\//);
    }
    expect(run.records().map(record => record.role)).toEqual(["opportunity-extractor", "paper-screener", "preliminary-investigator", "relevance-ranker"]);
  });

  it("codex_dead_model_fails_closed_when_native_agent_is_active: declare-and-warn branch (Markdown agents are not native)", async () => {
    const run = layout();
    stageInline(run, "codex", arxivAgents());
    // Fixture mutation: the recorded catalogue with gpt-5.6-sol removed.
    const reflectData = runs.arxivReflect();
    reflectData.endpoints.forEach(endpoint => endpoint.models && (endpoint.models = endpoint.models.filter(model => model !== "gpt-5.6-sol")));

    const outcome = await declareCodexSubagentModels({ reflectData, refetchReflectData: async () => reflectData, provider: "openai", agentsBaseDir: run.agentsBaseDir, env: run.env, logger: run.logger });

    expect(outcome.failures.map(failure => failure.name)).toEqual(["opportunity-extractor"]);
    expect(run.records()).toContainEqual({ engine: "codex", role: "opportunity-extractor", configured_model: "openai/gpt-5.6-sol", resolved_model: null, provider: "openai", status: "failed", cause: "model_unavailable" });
    expect(run.logs).toContainEqual(expect.stringContaining("warning: workflow sub-agent 'opportunity-extractor'"));
    expect(run.logs.join("\n")).not.toMatch(/The requested model is not supported|Refusing to start/);
  });

  it("codex_markdown_agents_are_not_loaded_and_warn", async () => {
    const run = layout();
    stageInline(run, "codex", arxivAgents());
    const staged = Object.fromEntries(fs.readdirSync(path.join(run.agentsBaseDir, ".codex/agents")).map(name => [name, fs.readFileSync(path.join(run.agentsBaseDir, ".codex/agents", name), "utf8")]));
    // The staged files are the recorded Markdown (Codex 0.163.0 discovers `.toml` only).
    expect(Object.keys(staged).every(name => name.endsWith(".md"))).toBe(true);
    expect(staged).toEqual(Object.fromEntries(Object.entries(arxivAgents()).map(([name, content]) => [`${name}.md`, content.endsWith("\n") ? content : `${content}\n`])));

    const outcome = await declareCodexSubagentModels({ reflectData: runs.arxivReflect(), provider: "openai", agentsBaseDir: run.agentsBaseDir, env: run.env, logger: run.logger });

    expect(outcome.agents).toEqual(["opportunity-extractor", "paper-screener", "preliminary-investigator", "relevance-ranker"]);
    expect(run.logs).toContainEqual(
      expect.stringMatching(
        /^warning: Codex discovers custom agents only from TOML role files \(verified for Codex 0\.163\.0\); the 4 workflow-declared Markdown sub-agent\(s\) .* are not loaded by Codex, and their declared models are recorded but not applied\.$/
      )
    );
    for (const [name, content] of Object.entries(staged)) {
      expect(fs.readFileSync(path.join(run.agentsBaseDir, ".codex/agents", name), "utf8")).toBe(content);
      expect(fs.readFileSync(path.join(run.workspaceDir, ".codex/agents", name), "utf8")).toBe(content);
    }
    expect(fs.readdirSync(path.join(run.workspaceDir, ".codex/agents")).some(name => name.endsWith(".toml"))).toBe(false);
  });

  it("does nothing without staged Codex agents", async () => {
    const run = layout();
    const outcome = await declareCodexSubagentModels({ reflectData: runs.arxivReflect(), provider: "openai", agentsBaseDir: run.agentsBaseDir, env: run.env, logger: run.logger });
    expect(outcome).toEqual({ agents: [], resolutions: [], failures: [] });
    expect(run.logs).toEqual([]);
  });
});

describe("isDetectionPhase", () => {
  it.each([
    ["detection", true],
    [" Detection ", true],
    ["agent", false],
    ["", false],
    [undefined, false],
  ])("%s -> %s", (phase, expected) => {
    expect(isDetectionPhase(phase)).toBe(expected);
  });
});
