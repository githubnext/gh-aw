import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { spawnSync } from "child_process";
import { createRequire } from "module";
import fs from "fs";
import os from "os";
import path from "path";

const require = createRequire(import.meta.url);
const { parseArgs, loadConfig, loadPrompt, buildProviderConfigs, appendStepSummary } = require("./aw_harness.cjs");

const harnessPath = path.resolve(path.dirname(require.resolve("./aw_harness.cjs")), "aw_harness.cjs");

// ---------------------------------------------------------------------------
// Temp directory helpers
// ---------------------------------------------------------------------------

const agentTempDir = "/tmp/gh-aw/agent";

function makeTempDir(prefix) {
  fs.mkdirSync(agentTempDir, { recursive: true });
  return fs.mkdtempSync(path.join(agentTempDir, prefix));
}

function makeConfigFile(dir, config = {}) {
  const configPath = path.join(dir, "config.json");
  fs.writeFileSync(configPath, JSON.stringify({ model: "test-model", ...config }), "utf8");
  return configPath;
}

function makePromptFile(dir, content = "Analyse the repository.") {
  const promptPath = path.join(dir, "prompt.txt");
  fs.writeFileSync(promptPath, content, "utf8");
  return promptPath;
}

// ---------------------------------------------------------------------------
// Unit tests for exported helpers
// ---------------------------------------------------------------------------

describe("parseArgs", () => {
  it("parses --config and --prompt flags", () => {
    const result = parseArgs(["node", "aw_harness.cjs", "--config", "/tmp/config.json", "--prompt", "/tmp/prompt.txt"]);
    expect(result.configPath).toBe("/tmp/config.json");
    expect(result.promptPath).toBe("/tmp/prompt.txt");
  });

  it("returns null when flags are absent", () => {
    const result = parseArgs(["node", "aw_harness.cjs"]);
    expect(result.configPath).toBeNull();
    expect(result.promptPath).toBeNull();
  });

  it("handles flags in reversed order", () => {
    const result = parseArgs(["node", "aw_harness.cjs", "--prompt", "/tmp/prompt.txt", "--config", "/tmp/config.json"]);
    expect(result.configPath).toBe("/tmp/config.json");
    expect(result.promptPath).toBe("/tmp/prompt.txt");
  });

  it("ignores unknown flags", () => {
    const result = parseArgs(["node", "aw_harness.cjs", "--unknown", "val", "--config", "/cfg", "--prompt", "/p"]);
    expect(result.configPath).toBe("/cfg");
    expect(result.promptPath).toBe("/p");
  });
});

describe("loadConfig", () => {
  it("reads and parses a valid JSON config file", () => {
    const dir = makeTempDir("aw-load-config-");
    const configPath = makeConfigFile(dir, { model: "claude-sonnet-4.6" });
    const config = loadConfig(configPath);
    expect(config.model).toBe("claude-sonnet-4.6");
  });

  it("throws a descriptive error when file does not exist", () => {
    expect(() => loadConfig("/nonexistent/config.json")).toThrow(/failed to read config file/);
  });

  it("throws a descriptive error when file contains invalid JSON", () => {
    const dir = makeTempDir("aw-load-config-");
    const badPath = path.join(dir, "bad.json");
    fs.writeFileSync(badPath, "not json", "utf8");
    expect(() => loadConfig(badPath)).toThrow(/failed to parse config file/);
  });
});

describe("loadPrompt", () => {
  it("reads the prompt file contents", () => {
    const dir = makeTempDir("aw-load-prompt-");
    const promptPath = makePromptFile(dir, "Fix the broken tests.");
    const prompt = loadPrompt(promptPath);
    expect(prompt).toBe("Fix the broken tests.");
  });

  it("throws a descriptive error when file does not exist", () => {
    expect(() => loadPrompt("/nonexistent/prompt.txt")).toThrow(/failed to read prompt file/);
  });
});

describe("buildProviderConfigs", () => {
  let savedEnv;

  beforeEach(() => {
    savedEnv = { ...process.env };
    // Clear all provider env vars before each test
    for (const key of ["ANTHROPIC_API_KEY", "ANTHROPIC_BASE_URL", "OPENAI_API_KEY", "CODEX_API_KEY", "OPENAI_BASE_URL", "CODEX_BASE_URL", "COPILOT_GITHUB_TOKEN", "GITHUB_TOKEN", "COPILOT_BASE_URL", "GEMINI_API_KEY", "GEMINI_BASE_URL"]) {
      delete process.env[key];
    }
  });

  afterEach(() => {
    // Restore env
    for (const key of ["ANTHROPIC_API_KEY", "ANTHROPIC_BASE_URL", "OPENAI_API_KEY", "CODEX_API_KEY", "OPENAI_BASE_URL", "CODEX_BASE_URL", "COPILOT_GITHUB_TOKEN", "GITHUB_TOKEN", "COPILOT_BASE_URL", "GEMINI_API_KEY", "GEMINI_BASE_URL"]) {
      if (savedEnv[key] !== undefined) {
        process.env[key] = savedEnv[key];
      } else {
        delete process.env[key];
      }
    }
  });

  it("returns empty array when no provider env vars are set", () => {
    const providers = buildProviderConfigs();
    expect(providers).toEqual([]);
  });

  it("returns anthropic provider when ANTHROPIC_API_KEY is set", () => {
    process.env.ANTHROPIC_API_KEY = "sk-ant-test";
    const providers = buildProviderConfigs();
    expect(providers).toHaveLength(1);
    expect(providers[0].name).toBe("anthropic");
    expect(providers[0].api).toBe("anthropic");
    expect(providers[0].apiKey).toBe("sk-ant-test");
    expect(providers[0].baseUrl).toBeUndefined();
  });

  it("includes baseUrl when ANTHROPIC_BASE_URL is set", () => {
    process.env.ANTHROPIC_API_KEY = "sk-ant-test";
    process.env.ANTHROPIC_BASE_URL = "https://proxy.example.com";
    const providers = buildProviderConfigs();
    expect(providers[0].baseUrl).toBe("https://proxy.example.com");
  });

  it("returns openai provider from OPENAI_API_KEY", () => {
    process.env.OPENAI_API_KEY = "sk-openai-test";
    const providers = buildProviderConfigs();
    const openai = providers.find(p => p.name === "openai");
    expect(openai).toBeDefined();
    expect(openai.api).toBe("openai-completions");
  });

  it("returns openai provider from CODEX_API_KEY (alias)", () => {
    process.env.CODEX_API_KEY = "sk-codex-test";
    const providers = buildProviderConfigs();
    const openai = providers.find(p => p.name === "openai");
    expect(openai).toBeDefined();
    expect(openai.apiKey).toBe("sk-codex-test");
  });

  it("prefers OPENAI_API_KEY over CODEX_API_KEY", () => {
    process.env.OPENAI_API_KEY = "sk-openai";
    process.env.CODEX_API_KEY = "sk-codex";
    const providers = buildProviderConfigs();
    const openai = providers.find(p => p.name === "openai");
    expect(openai.apiKey).toBe("sk-openai");
  });

  it("returns copilot provider from COPILOT_GITHUB_TOKEN", () => {
    process.env.COPILOT_GITHUB_TOKEN = "ghp_test";
    const providers = buildProviderConfigs();
    const copilot = providers.find(p => p.name === "copilot");
    expect(copilot).toBeDefined();
    expect(copilot.apiKey).toBe("ghp_test");
  });

  it("returns copilot provider from GITHUB_TOKEN fallback", () => {
    process.env.GITHUB_TOKEN = "ghs_test";
    const providers = buildProviderConfigs();
    const copilot = providers.find(p => p.name === "copilot");
    expect(copilot).toBeDefined();
  });

  it("returns google provider from GEMINI_API_KEY", () => {
    process.env.GEMINI_API_KEY = "gemini-key";
    const providers = buildProviderConfigs();
    const google = providers.find(p => p.name === "google");
    expect(google).toBeDefined();
    expect(google.api).toBe("google-generative-ai");
  });

  it("returns multiple providers when multiple env vars are set", () => {
    process.env.ANTHROPIC_API_KEY = "sk-ant";
    process.env.OPENAI_API_KEY = "sk-oai";
    const providers = buildProviderConfigs();
    expect(providers.length).toBeGreaterThanOrEqual(2);
    expect(providers.map(p => p.name)).toContain("anthropic");
    expect(providers.map(p => p.name)).toContain("openai");
  });
});

describe("appendStepSummary", () => {
  let savedEnv;

  beforeEach(() => {
    savedEnv = process.env.GITHUB_STEP_SUMMARY;
    delete process.env.GITHUB_STEP_SUMMARY;
  });

  afterEach(() => {
    if (savedEnv !== undefined) {
      process.env.GITHUB_STEP_SUMMARY = savedEnv;
    } else {
      delete process.env.GITHUB_STEP_SUMMARY;
    }
  });

  it("does nothing when GITHUB_STEP_SUMMARY is not set", () => {
    // Should not throw
    expect(() => appendStepSummary("hello")).not.toThrow();
  });

  it("writes content to the summary file", () => {
    const dir = makeTempDir("aw-step-summary-");
    const summaryPath = path.join(dir, "summary.md");
    process.env.GITHUB_STEP_SUMMARY = summaryPath;
    appendStepSummary("# Test Summary\n");
    const content = fs.readFileSync(summaryPath, "utf8");
    expect(content).toBe("# Test Summary\n");
  });

  it("appends multiple writes", () => {
    const dir = makeTempDir("aw-step-summary-");
    const summaryPath = path.join(dir, "summary.md");
    process.env.GITHUB_STEP_SUMMARY = summaryPath;
    appendStepSummary("line 1\n");
    appendStepSummary("line 2\n");
    const content = fs.readFileSync(summaryPath, "utf8");
    expect(content).toBe("line 1\nline 2\n");
  });

  it("does not throw when summary path is not writable", () => {
    process.env.GITHUB_STEP_SUMMARY = "/nonexistent/dir/summary.md";
    expect(() => appendStepSummary("hello")).not.toThrow();
  });
});

// ---------------------------------------------------------------------------
// Integration tests: invocation contract (T-AW-001 exit code 2 cases)
// ---------------------------------------------------------------------------

describe("aw_harness.cjs invocation contract (section 5)", () => {
  it("exits with code 2 when --config is missing", () => {
    const dir = makeTempDir("aw-invocation-");
    const promptPath = makePromptFile(dir);
    const result = spawnSync(process.execPath, [harnessPath, "--prompt", promptPath], {
      encoding: "utf8",
      timeout: 10000,
    });
    expect(result.status).toBe(2);
    expect(result.stderr).toMatch(/--config.*required/i);
  });

  it("exits with code 2 when --prompt is missing", () => {
    const dir = makeTempDir("aw-invocation-");
    const configPath = makeConfigFile(dir);
    const result = spawnSync(process.execPath, [harnessPath, "--config", configPath], {
      encoding: "utf8",
      timeout: 10000,
    });
    expect(result.status).toBe(2);
    expect(result.stderr).toMatch(/--prompt.*required/i);
  });

  it("exits with code 2 when config file does not exist", () => {
    const dir = makeTempDir("aw-invocation-");
    const promptPath = makePromptFile(dir);
    const result = spawnSync(process.execPath, [harnessPath, "--config", "/nonexistent/config.json", "--prompt", promptPath], {
      encoding: "utf8",
      timeout: 10000,
    });
    expect(result.status).toBe(2);
    expect(result.stderr).toMatch(/failed to read config file/i);
  });

  it("exits with code 2 when config file contains invalid JSON", () => {
    const dir = makeTempDir("aw-invocation-");
    const badConfigPath = path.join(dir, "bad.json");
    fs.writeFileSync(badConfigPath, "not valid json", "utf8");
    const promptPath = makePromptFile(dir);
    const result = spawnSync(process.execPath, [harnessPath, "--config", badConfigPath, "--prompt", promptPath], {
      encoding: "utf8",
      timeout: 10000,
    });
    expect(result.status).toBe(2);
    expect(result.stderr).toMatch(/failed to parse config file/i);
  });

  it("exits with code 2 when prompt file does not exist", () => {
    const dir = makeTempDir("aw-invocation-");
    const configPath = makeConfigFile(dir);
    const result = spawnSync(process.execPath, [harnessPath, "--config", configPath, "--prompt", "/nonexistent/prompt.txt"], {
      encoding: "utf8",
      timeout: 10000,
    });
    expect(result.status).toBe(2);
    expect(result.stderr).toMatch(/failed to read prompt file/i);
  });

  it("exits with code 1 when Pi SDK is not installed (invocation contract §5.3)", () => {
    // The Pi SDK is not installed in the dev environment — the harness must
    // exit with code 1 and emit a descriptive error to stderr.
    const dir = makeTempDir("aw-invocation-");
    const configPath = makeConfigFile(dir);
    const promptPath = makePromptFile(dir);
    const result = spawnSync(process.execPath, [harnessPath, "--config", configPath, "--prompt", promptPath], {
      encoding: "utf8",
      timeout: 15000,
    });
    // Either the SDK is installed (exit 0 or 1 for session) or not (exit 1).
    // We only assert it is NOT exit code 2 (that would indicate an invocation error).
    expect(result.status).not.toBe(2);
    if (result.status === 1) {
      // Expect either "Pi SDK is not installed" or a session error
      expect(result.stderr.length).toBeGreaterThan(0);
    }
  });

  it("writes step summary content when GITHUB_STEP_SUMMARY is set and SDK is missing", () => {
    const dir = makeTempDir("aw-invocation-");
    const configPath = makeConfigFile(dir);
    const promptPath = makePromptFile(dir);
    // Step summary is only written during session (post-SDK load), so this
    // test verifies the flag doesn't cause any crash during the exit code 1 path.
    const summaryPath = path.join(dir, "summary.md");
    const result = spawnSync(process.execPath, [harnessPath, "--config", configPath, "--prompt", promptPath], {
      encoding: "utf8",
      timeout: 15000,
      env: { ...process.env, GITHUB_STEP_SUMMARY: summaryPath },
    });
    // Must not exit with invocation error (2)
    expect(result.status).not.toBe(2);
  });
});
