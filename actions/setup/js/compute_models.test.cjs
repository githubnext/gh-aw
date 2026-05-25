import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import os from "os";
import path from "path";
import { loadBuiltinAliases, loadUserAliases, mergeAliases, renderSummary, main } from "./compute_models.cjs";

const mockCore = {
  debug: vi.fn(),
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(undefined),
  },
};

global.core = mockCore;

describe("compute_models.cjs", () => {
  let tmpDir;
  let origRunnerTemp;
  let builtinModelsPath;

  beforeEach(() => {
    vi.clearAllMocks();

    // Create isolated tmp directory for each test
    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "compute-models-test-"));
    origRunnerTemp = process.env.RUNNER_TEMP;
    process.env.RUNNER_TEMP = tmpDir;

    // Create the expected actions destination directory and models.json
    const actionsDir = path.join(tmpDir, "gh-aw", "actions");
    fs.mkdirSync(actionsDir, { recursive: true });
    builtinModelsPath = path.join(actionsDir, "models.json");

    // Ensure /tmp/gh-aw exists for output
    fs.mkdirSync("/tmp/gh-aw", { recursive: true });

    // Clear GH_AW_INFO_MODEL_ALIASES by default
    delete process.env.GH_AW_INFO_MODEL_ALIASES;
  });

  afterEach(() => {
    // Restore RUNNER_TEMP
    if (origRunnerTemp === undefined) {
      delete process.env.RUNNER_TEMP;
    } else {
      process.env.RUNNER_TEMP = origRunnerTemp;
    }
    // Remove test tmp dir
    fs.rmSync(tmpDir, { recursive: true, force: true });
  });

  // -------------------------------------------------------------------------
  // loadBuiltinAliases
  // -------------------------------------------------------------------------
  describe("loadBuiltinAliases", () => {
    it("returns the aliases from models.json", () => {
      const aliases = { sonnet: ["copilot/*sonnet*"], haiku: ["copilot/*haiku*"] };
      fs.writeFileSync(builtinModelsPath, JSON.stringify({ version: "1", aliases }));

      const result = loadBuiltinAliases();
      expect(result).toEqual(aliases);
    });

    it("returns empty object and warns when file is missing", () => {
      // Do not write models.json
      const result = loadBuiltinAliases();
      expect(result).toEqual({});
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("not found"));
    });

    it("returns empty object and warns when aliases key is missing", () => {
      fs.writeFileSync(builtinModelsPath, JSON.stringify({ version: "1" }));
      const result = loadBuiltinAliases();
      expect(result).toEqual({});
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining('missing "aliases"'));
    });

    it("returns empty object and warns on invalid JSON", () => {
      fs.writeFileSync(builtinModelsPath, "not json");
      const result = loadBuiltinAliases();
      expect(result).toEqual({});
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Failed to parse builtin"));
    });
  });

  // -------------------------------------------------------------------------
  // loadUserAliases
  // -------------------------------------------------------------------------
  describe("loadUserAliases", () => {
    it("returns empty object when GH_AW_INFO_MODEL_ALIASES is not set", () => {
      delete process.env.GH_AW_INFO_MODEL_ALIASES;
      expect(loadUserAliases()).toEqual({});
    });

    it("returns empty object when GH_AW_INFO_MODEL_ALIASES is empty JSON object", () => {
      process.env.GH_AW_INFO_MODEL_ALIASES = "{}";
      expect(loadUserAliases()).toEqual({});
    });

    it("parses valid JSON object", () => {
      const overrides = { "my-model": ["openai/my-model-v1"] };
      process.env.GH_AW_INFO_MODEL_ALIASES = JSON.stringify(overrides);
      expect(loadUserAliases()).toEqual(overrides);
    });

    it("returns empty object and warns on invalid JSON", () => {
      process.env.GH_AW_INFO_MODEL_ALIASES = "not json";
      expect(loadUserAliases()).toEqual({});
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Failed to parse GH_AW_INFO_MODEL_ALIASES"));
    });

    it("returns empty object and warns when value is not an object", () => {
      process.env.GH_AW_INFO_MODEL_ALIASES = '"just-a-string"';
      expect(loadUserAliases()).toEqual({});
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("not a JSON object"));
    });
  });

  // -------------------------------------------------------------------------
  // mergeAliases
  // -------------------------------------------------------------------------
  describe("mergeAliases", () => {
    it("returns builtins when overrides is empty", () => {
      const builtins = { sonnet: ["copilot/*sonnet*"] };
      expect(mergeAliases(builtins, {})).toEqual(builtins);
    });

    it("overrides win over builtins for same key", () => {
      const builtins = { sonnet: ["copilot/*sonnet*"] };
      const overrides = { sonnet: ["openai/my-sonnet"] };
      const result = mergeAliases(builtins, overrides);
      expect(result.sonnet).toEqual(["openai/my-sonnet"]);
    });

    it("adds new user-defined aliases that are not in builtins", () => {
      const builtins = { sonnet: ["copilot/*sonnet*"] };
      const overrides = { "my-alias": ["openai/my-model-v1"] };
      const result = mergeAliases(builtins, overrides);
      expect(result).toEqual({ sonnet: ["copilot/*sonnet*"], "my-alias": ["openai/my-model-v1"] });
    });

    it("does not mutate builtins", () => {
      const builtins = { sonnet: ["copilot/*sonnet*"] };
      const overrides = { sonnet: ["openai/new-sonnet"] };
      mergeAliases(builtins, overrides);
      expect(builtins.sonnet).toEqual(["copilot/*sonnet*"]);
    });
  });

  // -------------------------------------------------------------------------
  // renderSummary
  // -------------------------------------------------------------------------
  describe("renderSummary", () => {
    it("includes alias count and user count in summary label", () => {
      const aliases = { sonnet: ["copilot/*sonnet*"], haiku: ["copilot/*haiku*"] };
      const markdown = renderSummary(aliases, 1);
      expect(markdown).toContain("2 total, 1 user-defined");
    });

    it("renders alias names as code spans", () => {
      const aliases = { sonnet: ["copilot/*sonnet*"] };
      const markdown = renderSummary(aliases, 0);
      expect(markdown).toContain("`sonnet`");
    });

    it("renders empty-string alias key as _default_", () => {
      const aliases = { "": ["sonnet", "haiku"] };
      const markdown = renderSummary(aliases, 0);
      expect(markdown).toContain("_default_");
    });

    it("renders targets as code spans", () => {
      const aliases = { sonnet: ["copilot/*sonnet*", "anthropic/*sonnet*"] };
      const markdown = renderSummary(aliases, 0);
      expect(markdown).toContain("`copilot/*sonnet*`");
      expect(markdown).toContain("`anthropic/*sonnet*`");
    });

    it("wraps output in a details/summary block", () => {
      const aliases = { a: ["b"] };
      const markdown = renderSummary(aliases, 0);
      expect(markdown).toContain("<details>");
      expect(markdown).toContain("</details>");
      expect(markdown).toContain("<summary>");
    });
  });

  // -------------------------------------------------------------------------
  // main integration
  // -------------------------------------------------------------------------
  describe("main", () => {
    it("writes merged models.json to /tmp/gh-aw/models.json", async () => {
      const aliases = { sonnet: ["copilot/*sonnet*"], haiku: ["copilot/*haiku*"] };
      fs.writeFileSync(builtinModelsPath, JSON.stringify({ version: "1", aliases }));

      process.env.GH_AW_INFO_MODEL_ALIASES = JSON.stringify({ "my-alias": ["openai/my-model"] });

      await main();

      const written = JSON.parse(fs.readFileSync("/tmp/gh-aw/models.json", "utf8"));
      expect(written.aliases.sonnet).toEqual(["copilot/*sonnet*"]);
      expect(written.aliases["my-alias"]).toEqual(["openai/my-model"]);
    });

    it("user overrides win over builtins in main", async () => {
      const aliases = { sonnet: ["copilot/*sonnet*"] };
      fs.writeFileSync(builtinModelsPath, JSON.stringify({ version: "1", aliases }));

      process.env.GH_AW_INFO_MODEL_ALIASES = JSON.stringify({ sonnet: ["openai/new-sonnet"] });

      await main();

      const written = JSON.parse(fs.readFileSync("/tmp/gh-aw/models.json", "utf8"));
      expect(written.aliases.sonnet).toEqual(["openai/new-sonnet"]);
    });

    it("writes models.json even when no user overrides", async () => {
      const aliases = { sonnet: ["copilot/*sonnet*"] };
      fs.writeFileSync(builtinModelsPath, JSON.stringify({ version: "1", aliases }));
      delete process.env.GH_AW_INFO_MODEL_ALIASES;

      await main();

      const written = JSON.parse(fs.readFileSync("/tmp/gh-aw/models.json", "utf8"));
      expect(written.aliases.sonnet).toEqual(["copilot/*sonnet*"]);
    });

    it("writes step summary", async () => {
      fs.writeFileSync(builtinModelsPath, JSON.stringify({ version: "1", aliases: { a: ["b"] } }));

      await main();

      expect(mockCore.summary.addRaw).toHaveBeenCalled();
      expect(mockCore.summary.write).toHaveBeenCalled();
    });
  });
});
