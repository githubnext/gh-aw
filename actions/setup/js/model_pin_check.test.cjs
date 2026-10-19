// @ts-check
import { afterEach, describe, expect, it } from "vitest";
import fs from "fs";
import os from "os";
import path from "path";
import { createRequire } from "module";

const require = createRequire(import.meta.url);
const { buildModelPinReport, collectModelPins, main, renderModelPinReport, renderModelPinSummary } = require("./model_pin_check.cjs");
// Recorded /reflect from Design Decision Gate run 38945917145 (40 Copilot models; no claude-opus-5).
const { ddgReflect, incompleteDiscovery } = require("./fixtures/model_resolution_reflect.cjs");

/** @type {string[]} */
const tempDirs = [];

afterEach(() => {
  for (const dir of tempDirs.splice(0)) fs.rmSync(dir, { recursive: true, force: true });
});

/**
 * Write small fixture workflows into a temporary repository.
 * @param {Record<string, string>} files repository-relative path → content
 * @returns {string} repository root
 */
function fixtureRepo(files) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "model-pin-check-"));
  tempDirs.push(root);
  fs.mkdirSync(path.join(root, ".github", "workflows"), { recursive: true });
  for (const [rel, content] of Object.entries(files)) {
    const file = path.join(root, rel);
    fs.mkdirSync(path.dirname(file), { recursive: true });
    fs.writeFileSync(file, content);
  }
  return root;
}

/**
 * @param {Record<string, string>} files
 * @param {any} [reflectData]
 */
function check(files, reflectData = ddgReflect()) {
  const collected = collectModelPins({ repoRoot: fixtureRepo(files) });
  return buildModelPinReport({ ...collected, reflectData });
}

/**
 * @param {ReturnType<typeof buildModelPinReport>} report
 * @param {string} field
 */
function pin(report, field) {
  const found = report.pins.find(result => result.field === field);
  expect(found, `pin ${field}`).toBeDefined();
  return /** @type {NonNullable<typeof found>} */ found;
}

describe("model_pin_check acceptance cases", () => {
  it("1. a retired copilot/claude-opus-5 pin is unresolvable and listed under Action required", () => {
    const report = check({ ".github/workflows/gate.md": "---\nengine: pi\nmodel: copilot/claude-opus-5\n---\n# Gate\n" });
    const result = pin(report, "model");
    expect(result).toMatchObject({ workflow: ".github/workflows/gate.md", configured: "copilot/claude-opus-5", status: "unresolvable", cause: "model_unavailable" });
    expect(report.action_required).toEqual([result]);
    const summary = renderModelPinSummary(report);
    expect(summary.startsWith("### Action required")).toBe(true);
    expect(summary).toContain("| `.github/workflows/gate.md` | `model` | `copilot/claude-opus-5` | `model_unavailable` |");
  });

  it("2. copilot/opus resolves to the newest live Opus", () => {
    const report = check({ ".github/workflows/opus.md": "---\nengine: copilot\nmodel: copilot/opus\n---\n" });
    expect(pin(report, "model")).toMatchObject({ status: "alias", resolved: "copilot/claude-opus-5.5" });
    expect(renderModelPinReport(report)).toContain("| alias resolves | → `copilot/claude-opus-5.5` |");
  });

  it("3. gpt-5-mini (an alias name and a live id) is exact", () => {
    const report = check({ ".github/workflows/mini.md": "---\nengine: copilot\nmodel: gpt-5-mini\n---\n" });
    expect(pin(report, "model")).toMatchObject({ status: "exact", resolved: "gpt-5-mini", cause: null });
  });

  it("4. an expression pin is not checked with reason expression", () => {
    const report = check({ ".github/workflows/expr.md": "---\nengine:\n  id: copilot\n  model: ${{ vars.GH_AW_MODEL_AGENT_COPILOT }}\n---\n" });
    expect(pin(report, "engine.model")).toMatchObject({ status: "not_checked", reason: "expression" });
  });

  it("5. a Claude workflow on the Anthropic provider is not checked", () => {
    const report = check({ ".github/workflows/claude.md": "---\nengine: claude\nmodel: claude-sonnet-4-6\n---\n" });
    expect(pin(report, "model")).toMatchObject({ provider: "anthropic", status: "not_checked", reason: "provider not in the Copilot catalogue" });
  });

  it("6. a workflow models: alias whose every candidate is gone is unresolvable", () => {
    const report = check({ ".github/workflows/alias.md": "---\nengine: copilot\nmodel: retired\nmodels:\n  retired:\n    - copilot/claude-opus-5\n    - copilot/claude-opus-4.8\n---\n" });
    expect(pin(report, "models.retired")).toMatchObject({ status: "unresolvable", cause: "model_unavailable" });
    expect(pin(report, "model")).toMatchObject({ status: "unresolvable", cause: "model_unavailable" });
    expect(report.action_required).toHaveLength(2);
  });

  it("7. no pins gives an empty table and no Action required section", () => {
    const report = check({ ".github/workflows/plain.md": "---\non: push\n---\n# Plain\n" });
    expect(report.pins).toEqual([]);
    expect(report.counts).toEqual({ total: 0, exact: 0, alias: 0, unresolvable: 0, not_checked: 0 });
    const full = renderModelPinReport(report);
    expect(full).not.toContain("Action required");
    expect(full).toMatch(/\| Workflow \| Field \| Configured \| Provider \| Result \| Detail \|\n\|---\|---\|---\|---\|---\|---\|\n$/);
  });
});

describe("model_pin_check collection and rules", () => {
  it("collects engine, detection, evals, sub-agent and imported pins with file and field", () => {
    const report = check({
      ".github/workflows/full.md": [
        "---",
        "engine:",
        "  id: copilot",
        "  model: gpt-5-mini",
        "  fallback-models: [copilot/claude-opus-5]",
        "imports:",
        "  - shared/models.md",
        "  - ../agents/helper.md",
        "safe-outputs:",
        "  threat-detection:",
        "    model: sonnet",
        "evals:",
        "  - question: ok?",
        "    model: mini",
        "permissions:",
        "  models: read",
        "---",
        "# Full",
        "",
        "## agent: `nano`",
        "---",
        "model: inherited",
        "---",
        "Help.",
      ].join("\n"),
      ".github/workflows/shared/models.md": "---\nmodels:\n  providers:\n    copilot:\n      models: {}\n  team-fast:\n    - copilot/claude-haiku-4.5\n---\n",
      ".github/agents/helper.md": "---\nname: helper\nmodel: copilot/claude-opus-4.8\n---\nHelper.\n",
    });
    const rows = report.pins.map(result => [result.source, result.field, result.status]);
    expect(rows).toEqual(
      expect.arrayContaining([
        [".github/workflows/full.md", "engine.model", "exact"],
        [".github/workflows/full.md", "engine.fallback-models[0]", "unresolvable"],
        [".github/workflows/full.md", "safe-outputs.threat-detection.model", "alias"],
        [".github/workflows/full.md", "evals[0].model", "alias"],
        [".github/workflows/full.md", "## agent: 'nano' model", "not_checked"],
        [".github/agents/helper.md", "model (imported agent file)", "unresolvable"],
        [".github/workflows/shared/models.md", "models.team-fast", "alias"],
      ])
    );
    expect(report.pins.some(result => result.field.includes("providers"))).toBe(false);
    expect(pin(report, "## agent: 'nano' model").reason).toMatch(/^sentinel `inherited`/);
  });

  it("resolves nested imports like the compiler (shared/ paths from the workflow directory)", () => {
    const report = check({
      ".github/workflows/nested.md": "---\nengine: copilot\nimports:\n  - shared/a.md\n---\n",
      ".github/workflows/shared/a.md": "---\nimports:\n  - shared/b.md\n  - ./c.md\n---\n",
      ".github/workflows/shared/b.md": "---\nmodels:\n  b-alias: [copilot/gpt-5-mini]\n---\n",
      ".github/workflows/shared/c.md": "---\nmodels:\n  c-alias: [copilot/gpt-5-mini]\n---\n",
    });
    expect(report.errors).toEqual([]);
    expect(pin(report, "models.b-alias").source).toBe(".github/workflows/shared/b.md");
    expect(pin(report, "models.c-alias").source).toBe(".github/workflows/shared/c.md");
  });

  it("does not check Copilot BYOK pins", () => {
    const report = check({ ".github/workflows/byok.md": "---\nengine:\n  id: copilot\n  env:\n    COPILOT_PROVIDER_BASE_URL: https://example.invalid/openai/v1\nmodel: o4-mini-aw\n---\n" });
    expect(pin(report, "model")).toMatchObject({ status: "not_checked", reason: expect.stringContaining("BYOK") });
  });

  it("honours a copilot/ prefix over the engine provider and keeps Claude copilot/auto as a sentinel", () => {
    const report = check({ ".github/workflows/claude.md": "---\nengine:\n  id: claude\n  model: copilot/auto\n  fallback-models: [copilot/claude-sonnet-4.6, claude-sonnet-4-6]\n---\n" });
    expect(pin(report, "engine.fallback-models[0]")).toMatchObject({ provider: "copilot", status: "unresolvable", cause: "model_unavailable" });
    expect(pin(report, "engine.model")).toMatchObject({ status: "not_checked", reason: expect.stringMatching(/^sentinel `auto`/) });
    // A bare fallback follows the engine provider, which a copilot/ main model selects.
    expect(pin(report, "engine.fallback-models[1]")).toMatchObject({ provider: "copilot", status: "unresolvable", cause: "model_unavailable" });
  });

  it("reports catalog_unavailable when /reflect has no Copilot model list", () => {
    const report = check({ ".github/workflows/mini.md": "---\nengine: copilot\nmodel: gpt-5-mini\n---\n" }, incompleteDiscovery());
    expect(pin(report, "model")).toMatchObject({ status: "unresolvable", cause: "catalog_unavailable" });
    const summary = renderModelPinSummary(report);
    expect(summary.startsWith("### Action required")).toBe(true);
    expect(summary).toContain("1 Copilot workflow model pin(s) could not be checked");
    expect(summary).not.toContain("| Workflow | Field | Configured | Cause |");
    expect(summary).toContain("Live Copilot catalogue: **unavailable**");
  });

  it("main writes the JSON, full and summary reports", () => {
    const root = fixtureRepo({ ".github/workflows/gate.md": "---\nengine: pi\nmodel: copilot/claude-opus-5\n---\n" });
    const reflectFile = path.join(root, "reflect.json");
    fs.writeFileSync(reflectFile, JSON.stringify(ddgReflect()));
    const outDir = path.join(root, "out");
    const write = process.stdout.write;
    /** @type {string[]} */
    const printed = [];
    process.stdout.write = /** @type {any} */ chunk => printed.push(String(chunk));
    try {
      main(["--reflect", reflectFile, "--out-dir", outDir, "--repo", root]);
    } finally {
      process.stdout.write = write;
    }
    const json = JSON.parse(fs.readFileSync(path.join(outDir, "model-pins.json"), "utf8"));
    expect(json.counts.unresolvable).toBe(1);
    expect(fs.readFileSync(path.join(outDir, "model-pins.md"), "utf8")).toContain("### All Workflow Model Pins");
    expect(fs.readFileSync(path.join(outDir, "model-pins-summary.md"), "utf8")).toBe(printed.join(""));
  });
});
