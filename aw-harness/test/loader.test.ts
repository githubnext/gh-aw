import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { writeFileSync, mkdirSync, rmSync } from "node:fs";
import { join } from "node:path";
import { tmpdir } from "node:os";
import { parseArgs, loadInputs, parseConfig, CONFIG_DEFAULTS } from "../src/loader.js";

// ─── parseArgs ───────────────────────────────────────────────────────────────

describe("parseArgs", () => {
  it("parses --config and --prompt flags", () => {
    const result = parseArgs(["--config", "/tmp/config.json", "--prompt", "/tmp/prompt.txt"]);
    expect(result).toEqual({ configPath: "/tmp/config.json", promptPath: "/tmp/prompt.txt" });
  });

  it("parses flags in reverse order", () => {
    const result = parseArgs(["--prompt", "/tmp/prompt.txt", "--config", "/tmp/config.json"]);
    expect(result).toEqual({ configPath: "/tmp/config.json", promptPath: "/tmp/prompt.txt" });
  });

  it("throws when --config is missing", () => {
    expect(() => parseArgs(["--prompt", "/tmp/prompt.txt"])).toThrow("--config");
  });

  it("throws when --prompt is missing", () => {
    expect(() => parseArgs(["--config", "/tmp/config.json"])).toThrow("--prompt");
  });

  it("throws when both flags are missing", () => {
    expect(() => parseArgs([])).toThrow("--config");
  });

  it("ignores unknown flags", () => {
    const result = parseArgs([
      "--verbose",
      "--config",
      "/tmp/config.json",
      "--prompt",
      "/tmp/prompt.txt",
      "--extra",
      "value",
    ]);
    expect(result).toEqual({ configPath: "/tmp/config.json", promptPath: "/tmp/prompt.txt" });
  });
});

// ─── parseConfig ─────────────────────────────────────────────────────────────

describe("parseConfig", () => {
  it("parses a minimal valid config", () => {
    const config = parseConfig({ model: "claude-sonnet-4.6" });
    expect(config.model).toBe("claude-sonnet-4.6");
    expect(config.timeoutMinutes).toBe(CONFIG_DEFAULTS.timeoutMinutes);
    expect(config.context.compaction).toBe("none");
    expect(config.context.compactionThreshold).toBe(0.75);
    expect(config.steering.timeWarningMinutes).toBe(5);
    expect(config.steering.timeCriticalMinutes).toBe(2);
    expect(config.budget).toBeUndefined();
    expect(config.observability).toBeUndefined();
  });

  it("throws when model is missing", () => {
    expect(() => parseConfig({ timeoutMinutes: 30 })).toThrow("model");
  });

  it("throws when model is empty string", () => {
    expect(() => parseConfig({ model: "" })).toThrow("model");
  });

  it("throws on non-object input", () => {
    expect(() => parseConfig("not-an-object")).toThrow();
    expect(() => parseConfig(null)).toThrow();
    expect(() => parseConfig(42)).toThrow();
  });

  it("applies default timeoutMinutes when missing", () => {
    const config = parseConfig({ model: "gpt-4" });
    expect(config.timeoutMinutes).toBe(CONFIG_DEFAULTS.timeoutMinutes);
  });

  it("uses provided timeoutMinutes", () => {
    const config = parseConfig({ model: "gpt-4", timeoutMinutes: 45 });
    expect(config.timeoutMinutes).toBe(45);
  });

  it("parses budget config", () => {
    const config = parseConfig({
      model: "sonnet",
      budget: { maxEffectiveTokens: 100_000 },
    });
    expect(config.budget?.maxEffectiveTokens).toBe(100_000);
  });

  it("ignores invalid budget (negative tokens) and emits a warning to stderr", () => {
    const warnLines: string[] = [];
    const spy = vi.spyOn(process.stderr, "write").mockImplementation((msg: unknown) => {
      warnLines.push(String(msg));
      return true;
    });
    try {
      const config = parseConfig({ model: "sonnet", budget: { maxEffectiveTokens: -1 } });
      expect(config.budget).toBeUndefined();
      expect(warnLines.some((l) => l.includes("Ignoring invalid budget"))).toBe(true);
    } finally {
      spy.mockRestore();
    }
  });

  it("parses context config", () => {
    const config = parseConfig({
      model: "sonnet",
      context: { compaction: "summarize", compactionThreshold: 0.8 },
    });
    expect(config.context.compaction).toBe("summarize");
    expect(config.context.compactionThreshold).toBe(0.8);
  });

  it("clamps compactionThreshold to [0, 1]", () => {
    const configHigh = parseConfig({ model: "m", context: { compactionThreshold: 1.5 } });
    expect(configHigh.context.compactionThreshold).toBe(1);

    const configLow = parseConfig({ model: "m", context: { compactionThreshold: -0.1 } });
    expect(configLow.context.compactionThreshold).toBe(0);
  });

  it("falls back to 'none' for unknown compaction mode", () => {
    const config = parseConfig({ model: "m", context: { compaction: "magic" } });
    expect(config.context.compaction).toBe("none");
  });

  it("parses steering config", () => {
    const config = parseConfig({
      model: "m",
      steering: {
        timeWarningMinutes: 10,
        timeCriticalMinutes: 3,
        budgetWarnPercent: 80,
        budgetCriticalPercent: 95,
      },
    });
    expect(config.steering.timeWarningMinutes).toBe(10);
    expect(config.steering.timeCriticalMinutes).toBe(3);
    expect(config.steering.budgetWarnPercent).toBe(80);
    expect(config.steering.budgetCriticalPercent).toBe(95);
  });

  it("parses observability config", () => {
    const config = parseConfig({
      model: "m",
      observability: { otlp: { endpoint: "https://otlp.example.com", headers: { Authorization: "Bearer tok" } } },
    });
    expect(config.observability?.otlp?.endpoint).toBe("https://otlp.example.com");
    expect(config.observability?.otlp?.headers?.["Authorization"]).toBe("Bearer tok");
  });

  it("ignores observability with missing endpoint", () => {
    const config = parseConfig({ model: "m", observability: { otlp: { noEndpoint: true } } });
    expect(config.observability).toBeUndefined();
  });

  it("parses extensions list", () => {
    const config = parseConfig({
      model: "m",
      extensions: ["./ext.cjs", "@my-org/ext"],
      extensionsRequired: true,
    });
    expect(config.extensions).toEqual(["./ext.cjs", "@my-org/ext"]);
    expect(config.extensionsRequired).toBe(true);
  });

  it("filters non-string entries from extensions", () => {
    const config = parseConfig({ model: "m", extensions: ["./a.cjs", 42, null, "./b.cjs"] });
    expect(config.extensions).toEqual(["./a.cjs", "./b.cjs"]);
  });

  it("parses imports list", () => {
    const config = parseConfig({
      model: "m",
      imports: [{ path: "skills/foo/SKILL.md", content: "# Foo skill" }],
    });
    expect(config.imports).toHaveLength(1);
    expect(config.imports?.[0]?.path).toBe("skills/foo/SKILL.md");
    expect(config.imports?.[0]?.content).toBe("# Foo skill");
  });
});

// ─── loadInputs ──────────────────────────────────────────────────────────────

describe("loadInputs", () => {
  let tmpDir: string;

  beforeEach(() => {
    tmpDir = join(tmpdir(), `aw-harness-test-${Date.now()}`);
    mkdirSync(tmpDir, { recursive: true });
  });

  afterEach(() => {
    rmSync(tmpDir, { recursive: true, force: true });
  });

  it("loads valid config.json and prompt.txt", () => {
    const configPath = join(tmpDir, "config.json");
    const promptPath = join(tmpDir, "prompt.txt");

    writeFileSync(configPath, JSON.stringify({ model: "claude-sonnet-4.6", timeoutMinutes: 30 }));
    writeFileSync(promptPath, "Review the changes and create an issue.");

    const { config, promptBody } = loadInputs(configPath, promptPath);
    expect(config.model).toBe("claude-sonnet-4.6");
    expect(config.timeoutMinutes).toBe(30);
    expect(promptBody).toBe("Review the changes and create an issue.");
  });

  it("throws when config.json is missing", () => {
    const promptPath = join(tmpDir, "prompt.txt");
    writeFileSync(promptPath, "hello");

    expect(() => loadInputs(join(tmpDir, "nonexistent.json"), promptPath)).toThrow("config.json");
  });

  it("throws when prompt.txt is missing", () => {
    const configPath = join(tmpDir, "config.json");
    writeFileSync(configPath, JSON.stringify({ model: "gpt-4" }));

    expect(() => loadInputs(configPath, join(tmpDir, "nonexistent.txt"))).toThrow("prompt.txt");
  });

  it("throws when config.json contains invalid JSON", () => {
    const configPath = join(tmpDir, "config.json");
    const promptPath = join(tmpDir, "prompt.txt");

    writeFileSync(configPath, "{ not valid json");
    writeFileSync(promptPath, "hello");

    expect(() => loadInputs(configPath, promptPath)).toThrow("config.json");
  });

  it("throws when config.json is missing the model field", () => {
    const configPath = join(tmpDir, "config.json");
    const promptPath = join(tmpDir, "prompt.txt");

    writeFileSync(configPath, JSON.stringify({ timeoutMinutes: 30 }));
    writeFileSync(promptPath, "hello");

    expect(() => loadInputs(configPath, promptPath)).toThrow("model");
  });
});
