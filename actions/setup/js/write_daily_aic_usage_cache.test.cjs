// @ts-check
import fs from "fs";
import os from "os";
import path from "path";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

let exports;

describe("write_daily_aic_usage_cache", () => {
  let tmpDir;
  let cacheFile;
  let usageDir;

  beforeEach(async () => {
    vi.resetModules();
    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "write-aic-cache-test-"));
    cacheFile = path.join(tmpDir, "agentic-workflow-usage-cache.jsonl");
    usageDir = path.join(tmpDir, "usage");
    fs.mkdirSync(usageDir, { recursive: true });

    global.core = { info: vi.fn(), warning: vi.fn(), error: vi.fn(), setFailed: vi.fn() };
    process.env.GITHUB_RUN_ID = "12345";

    const mod = await import("./write_daily_aic_usage_cache.cjs");
    exports = mod.default || mod;
  });

  afterEach(() => {
    fs.rmSync(tmpDir, { recursive: true, force: true });
    delete global.core;
    delete process.env.GITHUB_RUN_ID;
  });

  /**
   * Writes a usage JSONL file in the temp usage directory so that sumAICFromUsageJSONLFiles
   * can pick it up. The entry uses the `ai_credits` field for simplicity.
   *
   * @param {number} aiCredits
   */
  function writeUsageFile(aiCredits) {
    fs.writeFileSync(path.join(usageDir, "agent_usage.jsonl"), JSON.stringify({ ai_credits: aiCredits }) + "\n", "utf8");
  }

  /**
   * Invoke the main() function after patching the module-level paths to point to the
   * temp directory. We call the function directly on the in-process module instance.
   *
   * @returns {Promise<void>}
   */
  async function runMain() {
    // Patch the module-level path constants by calling main() on the already-imported
    // module; to make the paths configurable we call the exported helper that accepts
    // explicit path arguments (defined below), falling back to swapping env vars.
    await exports.mainWithPaths(cacheFile, usageDir);
  }

  it("writes a new entry with run_id, aic, and a timestamp when no cache file exists", async () => {
    writeUsageFile(7.5);
    await runMain();

    const content = fs.readFileSync(cacheFile, "utf8").trim();
    const entry = JSON.parse(content);
    expect(entry.run_id).toBe(12345);
    expect(entry.aic).toBe(7.5);
    expect(typeof entry.timestamp).toBe("string");
    // Timestamp should be a valid ISO 8601 date within the last minute.
    const ts = Date.parse(entry.timestamp);
    expect(Number.isFinite(ts)).toBe(true);
    expect(ts).toBeLessThanOrEqual(Date.now());
    expect(ts).toBeGreaterThan(Date.now() - 60_000);
  });

  it("appends to an existing cache file and preserves entries within 48 h", async () => {
    const recentTs = new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString(); // 2 hours ago
    fs.writeFileSync(cacheFile, JSON.stringify({ run_id: 9999, aic: 3.0, timestamp: recentTs }) + "\n", "utf8");

    writeUsageFile(5.0);
    await runMain();

    const lines = fs.readFileSync(cacheFile, "utf8").trim().split("\n");
    expect(lines).toHaveLength(2);
    const first = JSON.parse(lines[0]);
    expect(first.run_id).toBe(9999);
    const second = JSON.parse(lines[1]);
    expect(second.run_id).toBe(12345);
    expect(second.aic).toBe(5.0);
  });

  it("prunes existing entries whose timestamp is older than 48 h", async () => {
    const staleTs = new Date(Date.now() - 50 * 60 * 60 * 1000).toISOString(); // 50 hours ago
    const recentTs = new Date(Date.now() - 1 * 60 * 60 * 1000).toISOString(); // 1 hour ago
    fs.writeFileSync(cacheFile, [JSON.stringify({ run_id: 1001, aic: 1.0, timestamp: staleTs }), JSON.stringify({ run_id: 1002, aic: 2.0, timestamp: recentTs })].join("\n") + "\n", "utf8");

    writeUsageFile(4.0);
    await runMain();

    const lines = fs.readFileSync(cacheFile, "utf8").trim().split("\n");
    const runIds = lines.map(line => JSON.parse(line).run_id);
    // Stale entry 1001 must be pruned; recent 1002 and new 12345 kept.
    expect(runIds).not.toContain(1001);
    expect(runIds).toContain(1002);
    expect(runIds).toContain(12345);
  });

  it("preserves entries without a timestamp (backward compatibility)", async () => {
    fs.writeFileSync(cacheFile, JSON.stringify({ run_id: 2001, aic: 8.0 }) + "\n", "utf8");

    writeUsageFile(1.0);
    await runMain();

    const lines = fs.readFileSync(cacheFile, "utf8").trim().split("\n");
    const runIds = lines.map(line => JSON.parse(line).run_id);
    expect(runIds).toContain(2001);
    expect(runIds).toContain(12345);
  });

  it("skips writing when GITHUB_RUN_ID is not set", async () => {
    delete process.env.GITHUB_RUN_ID;
    writeUsageFile(5.0);
    await runMain();
    expect(fs.existsSync(cacheFile)).toBe(false);
  });
});
