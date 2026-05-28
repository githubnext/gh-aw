import { afterEach, describe, it, expect, vi } from "vitest";
import { createRequire } from "module";
import fs from "fs";
import os from "os";
import path from "path";
import childProcess from "child_process";

const require = createRequire(import.meta.url);
const {
  AWF_API_PROXY_REFLECT_URL,
  AWF_REFLECT_OUTPUT_PATH,
  AWF_REFLECT_TIMEOUT_MS,
  AWF_REFLECT_MAX_ATTEMPTS,
  AWF_REFLECT_RETRY_BASE_MS,
  AWF_REFLECT_RETRY_MAX_MS,
  AWF_MODELS_URL_TIMEOUT_MS,
  AWF_MODELS_URL_MAX_ATTEMPTS,
  AWF_MODELS_URL_RETRY_BASE_MS,
  AWF_MODELS_URL_RETRY_MAX_MS,
  GEMINI_MODEL_NAME_PREFIX,
  enrichReflectModels,
  extractModelIds,
  fetchAWFReflect,
  fetchModelsFromUrl,
} = require("./awf_reflect.cjs");

describe("awf_reflect.cjs", () => {
  describe("constants", () => {
    it("exports expected default values", () => {
      expect(AWF_API_PROXY_REFLECT_URL).toBe("http://api-proxy:10000/reflect");
      expect(AWF_REFLECT_OUTPUT_PATH).toBe("/tmp/gh-aw/sandbox/firewall/awf-reflect.json");
      expect(AWF_REFLECT_TIMEOUT_MS).toBe(60000);
      expect(AWF_REFLECT_MAX_ATTEMPTS).toBe(5);
      expect(AWF_REFLECT_RETRY_BASE_MS).toBe(500);
      expect(AWF_REFLECT_RETRY_MAX_MS).toBe(5000);
      expect(AWF_MODELS_URL_TIMEOUT_MS).toBe(3000);
      expect(AWF_MODELS_URL_MAX_ATTEMPTS).toBe(5);
      expect(AWF_MODELS_URL_RETRY_BASE_MS).toBe(250);
      expect(AWF_MODELS_URL_RETRY_MAX_MS).toBe(2000);
      expect(GEMINI_MODEL_NAME_PREFIX).toBe("models/");
    });
  });

  describe("extractModelIds", () => {
    it("returns null for null input", () => {
      expect(extractModelIds(null)).toBeNull();
    });

    it("returns null for empty object", () => {
      expect(extractModelIds({})).toBeNull();
    });

    it("returns null for empty data array", () => {
      expect(extractModelIds({ data: [] })).toBeNull();
    });

    it("extracts ids from OpenAI format", () => {
      const json = { data: [{ id: "gpt-4o" }, { id: "gpt-4o-mini" }] };
      expect(extractModelIds(json)).toEqual(["gpt-4o", "gpt-4o-mini"]);
    });

    it("falls back to name when id is absent in OpenAI format", () => {
      const json = { data: [{ name: "model-a" }, { id: "model-b" }] };
      expect(extractModelIds(json)).toEqual(["model-a", "model-b"]);
    });

    it("extracts ids from Gemini format, stripping prefix", () => {
      const json = {
        models: [{ name: "models/gemini-1.5-pro" }, { name: "models/gemini-1.0-pro" }],
      };
      expect(extractModelIds(json)).toEqual(["gemini-1.0-pro", "gemini-1.5-pro"]);
    });

    it("handles Gemini entries without the prefix", () => {
      const json = { models: [{ name: "custom-model" }] };
      expect(extractModelIds(json)).toEqual(["custom-model"]);
    });

    it("returns sorted results", () => {
      const json = { data: [{ id: "z-model" }, { id: "a-model" }, { id: "m-model" }] };
      expect(extractModelIds(json)).toEqual(["a-model", "m-model", "z-model"]);
    });

    it("returns null for empty models array", () => {
      expect(extractModelIds({ models: [] })).toBeNull();
    });
  });

  describe("enrichReflectModels", () => {
    afterEach(() => {
      vi.unstubAllGlobals();
    });

    it("does nothing when all configured endpoints already have models", async () => {
      const reflectData = {
        endpoints: [{ provider: "openai", configured: true, models: ["gpt-4o"], models_url: "http://api-proxy:10000/v1/models" }],
      };
      const logger = () => {};
      await enrichReflectModels(reflectData, 1000, logger);
      expect(reflectData.endpoints[0].models).toEqual(["gpt-4o"]);
    });

    it("does nothing for unconfigured endpoints with null models", async () => {
      const reflectData = {
        endpoints: [{ provider: "anthropic", configured: false, models: null, models_url: "http://api-proxy:10001/v1/models" }],
      };
      const logger = () => {};
      await enrichReflectModels(reflectData, 1000, logger);
      expect(reflectData.endpoints[0].models).toBeNull();
    });

    it("does nothing when models_url is null", async () => {
      const reflectData = {
        endpoints: [{ provider: "opencode", configured: true, models: null, models_url: null }],
      };
      const logger = () => {};
      await enrichReflectModels(reflectData, 1000, logger);
      expect(reflectData.endpoints[0].models).toBeNull();
    });

    it("fetches models from models_url for configured endpoints with null models", async () => {
      const modelResponse = { data: [{ id: "claude-sonnet-4.6" }, { id: "gpt-4o" }] };
      vi.stubGlobal("fetch", vi.fn().mockResolvedValue({ ok: true, status: 200, json: async () => modelResponse }));

      const reflectData = {
        endpoints: [{ provider: "copilot", configured: true, models: null, models_url: "http://api-proxy:10002/models" }],
      };
      const logs = [];
      await enrichReflectModels(reflectData, 3000, msg => logs.push(msg));

      expect(reflectData.endpoints[0].models).toEqual(["claude-sonnet-4.6", "gpt-4o"]);
      expect(logs.some(l => l.includes("fetched 2 model(s)"))).toBe(true);
    });

    it("leaves models null when models_url fetch fails", async () => {
      vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new Error("ECONNREFUSED")));

      const reflectData = {
        endpoints: [{ provider: "openai", configured: true, models: null, models_url: "http://api-proxy:10000/v1/models" }],
      };
      const logs = [];
      await enrichReflectModels(reflectData, 500, msg => logs.push(msg));
      expect(reflectData.endpoints[0].models).toBeNull();
      expect(logs.some(l => l.includes("models fetch error"))).toBe(true);
    });

    it("handles empty endpoints array", async () => {
      const reflectData = { endpoints: [] };
      const logger = () => {};
      await enrichReflectModels(reflectData, 1000, logger);
      expect(reflectData.endpoints).toEqual([]);
    });
  });

  describe("fetchModelsFromUrl", () => {
    afterEach(() => {
      vi.unstubAllGlobals();
    });

    it("returns model IDs on successful fetch", async () => {
      const modelData = { data: [{ id: "gpt-4o" }] };
      vi.stubGlobal("fetch", vi.fn().mockResolvedValue({ ok: true, status: 200, json: async () => modelData }));

      const logs = [];
      const result = await fetchModelsFromUrl("http://api-proxy:10000/v1/models", 1000, msg => logs.push(msg));
      expect(result).toEqual(["gpt-4o"]);
      expect(logs.some(l => l.includes("fetched 1 model(s)"))).toBe(true);
    });

    it("returns null on non-ok HTTP status", async () => {
      vi.stubGlobal("fetch", vi.fn().mockResolvedValue({ ok: false, status: 403 }));

      const logs = [];
      const result = await fetchModelsFromUrl("http://api-proxy:10000/v1/models", 1000, msg => logs.push(msg));
      expect(result).toBeNull();
      expect(logs.some(l => l.includes("models fetch returned 403"))).toBe(true);
    });

    it("returns null on network error", async () => {
      vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new Error("ECONNREFUSED")));

      const logs = [];
      const result = await fetchModelsFromUrl("http://api-proxy:10000/v1/models", 1000, msg => logs.push(msg));
      expect(result).toBeNull();
      expect(logs.some(l => l.includes("models fetch error"))).toBe(true);
    });

    it("retries on 503 and eventually succeeds", async () => {
      vi.stubGlobal(
        "fetch",
        vi
          .fn()
          .mockResolvedValueOnce({ ok: false, status: 503 })
          .mockResolvedValueOnce({ ok: false, status: 503 })
          .mockResolvedValueOnce({ ok: true, status: 200, json: async () => ({ data: [{ id: "gpt-4o" }] }) })
      );

      const logs = [];
      const result = await fetchModelsFromUrl("http://api-proxy:10000/v1/models", 1000, msg => logs.push(msg));
      expect(result).toEqual(["gpt-4o"]);
      expect(logs.filter(l => l.includes("retrying (attempt")).length).toBe(2);
      expect(logs.some(l => l.includes("fetched 1 model(s)"))).toBe(true);
    });

    it("stops retrying after max attempts on repeated 503 responses", async () => {
      vi.stubGlobal("fetch", vi.fn().mockResolvedValue({ ok: false, status: 503 }));

      const logs = [];
      const result = await fetchModelsFromUrl("http://api-proxy:10000/v1/models", 1000, msg => logs.push(msg));
      expect(result).toBeNull();
      expect(logs.filter(l => l.includes("retrying (attempt")).length).toBe(AWF_MODELS_URL_MAX_ATTEMPTS - 1);
      expect(logs.some(l => l.includes("models fetch returned 503"))).toBe(true);
    });
  });

  describe("fetchAWFReflect", () => {
    afterEach(() => {
      vi.unstubAllGlobals();
      vi.restoreAllMocks();
    });

    it("saves enriched reflect data when api-proxy returns null models for configured provider", async () => {
      const modelData = { data: [{ id: "gpt-4o" }, { id: "gpt-4o-mini" }] };
      const reflectPayload = {
        endpoints: [{ provider: "openai", port: 10000, configured: true, models: null, models_url: "http://api-proxy:10000/v1/models" }],
        models_fetch_complete: true,
      };

      vi.stubGlobal(
        "fetch",
        vi.fn().mockImplementation(url => {
          const body = String(url).includes("/reflect") ? reflectPayload : modelData;
          return Promise.resolve({ ok: true, status: 200, json: async () => body });
        })
      );

      const outputDir = fs.mkdtempSync(path.join(os.tmpdir(), "awf-reflect-test-"));
      const outputPath = path.join(outputDir, "awf-reflect.json");
      const logs = [];

      try {
        const result = await fetchAWFReflect({
          reflectUrl: "http://api-proxy:10000/reflect",
          outputPath,
          timeoutMs: 3000,
          modelsTimeoutMs: 1000,
          logger: msg => logs.push(msg),
        });

        expect(result).toEqual({
          ok: true,
          reflectUrl: "http://api-proxy:10000/reflect",
          outputPath,
          bytesWritten: expect.any(Number),
        });
        const saved = JSON.parse(fs.readFileSync(outputPath, "utf8"));
        expect(saved.endpoints[0].models).toEqual(["gpt-4o", "gpt-4o-mini"]);
        expect(logs.some(l => l.includes("saved "))).toBe(true);
      } finally {
        fs.rmSync(outputDir, { recursive: true, force: true });
      }
    });

    it("does not throw when the reflect endpoint is unreachable", async () => {
      vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new Error("ECONNREFUSED")));
      const logs = [];
      await expect(
        fetchAWFReflect({
          reflectUrl: "http://api-proxy:10000/reflect",
          outputPath: "/tmp/gh-aw-test-noop.json",
          timeoutMs: 500,
          logger: msg => logs.push(msg),
        })
      ).resolves.toEqual({
        ok: false,
        reflectUrl: "http://api-proxy:10000/reflect",
        outputPath: "/tmp/gh-aw-test-noop.json",
        reason: "request_failed",
        error: "ECONNREFUSED",
      });
      expect(logs.some(l => l.includes("request failed"))).toBe(true);
    });

    it("does not throw when the reflect endpoint returns non-retryable non-ok status", async () => {
      vi.stubGlobal("fetch", vi.fn().mockResolvedValue({ ok: false, status: 403 }));
      const logs = [];
      await expect(
        fetchAWFReflect({
          reflectUrl: "http://api-proxy:10000/reflect",
          outputPath: "/tmp/gh-aw-test-noop.json",
          timeoutMs: 500,
          logger: msg => logs.push(msg),
        })
      ).resolves.toEqual({
        ok: false,
        reflectUrl: "http://api-proxy:10000/reflect",
        outputPath: "/tmp/gh-aw-test-noop.json",
        reason: "unexpected_status",
        status: 403,
      });
      expect(logs.some(l => l.includes("unexpected status 403"))).toBe(true);
    });

    it("retries transient reflect status failures and eventually succeeds", async () => {
      const reflectPayload = {
        endpoints: [{ provider: "openai", port: 10000, configured: true, models: ["gpt-4o"], models_url: "http://api-proxy:10000/v1/models" }],
        models_fetch_complete: true,
      };

      vi.stubGlobal(
        "fetch",
        vi
          .fn()
          .mockResolvedValueOnce({ ok: false, status: 503 })
          .mockResolvedValueOnce({ ok: false, status: 503 })
          .mockResolvedValue({ ok: true, status: 200, json: async () => reflectPayload })
      );

      const outputDir = fs.mkdtempSync(path.join(os.tmpdir(), "awf-reflect-retry-test-"));
      const outputPath = path.join(outputDir, "awf-reflect.json");

      try {
        const result = await fetchAWFReflect({
          reflectUrl: "http://api-proxy:10000/reflect",
          outputPath,
          timeoutMs: 1000,
          maxAttempts: 3,
          retryBaseMs: 1,
          retryMaxMs: 2,
          logger: () => {},
        });

        expect(result.ok).toBe(true);
        expect(fs.existsSync(outputPath)).toBe(true);
      } finally {
        fs.rmSync(outputDir, { recursive: true, force: true });
      }
    });

    it("retries fetch failed when the nested cause is transient and eventually succeeds", async () => {
      const reflectPayload = {
        endpoints: [{ provider: "openai", port: 10000, configured: true, models: ["gpt-4o"], models_url: "http://api-proxy:10000/v1/models" }],
        models_fetch_complete: true,
      };
      const fetchFailedWithTransientCause = Object.assign(new Error("fetch failed"), { cause: { code: "ECONNREFUSED" } });

      vi.stubGlobal(
        "fetch",
        vi
          .fn()
          .mockRejectedValueOnce(fetchFailedWithTransientCause)
          .mockResolvedValue({ ok: true, status: 200, json: async () => reflectPayload })
      );

      const outputDir = fs.mkdtempSync(path.join(os.tmpdir(), "awf-reflect-retry-fetch-failed-test-"));
      const outputPath = path.join(outputDir, "awf-reflect.json");

      try {
        const result = await fetchAWFReflect({
          reflectUrl: "http://api-proxy:10000/reflect",
          outputPath,
          timeoutMs: 1000,
          maxAttempts: 3,
          retryBaseMs: 1,
          retryMaxMs: 2,
          logger: () => {},
        });

        expect(result.ok).toBe(true);
        expect(fs.existsSync(outputPath)).toBe(true);
      } finally {
        fs.rmSync(outputDir, { recursive: true, force: true });
      }
    });

    it("uses the caller-supplied logger for all messages", async () => {
      vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new Error("ECONNREFUSED")));
      const collected = [];
      await fetchAWFReflect({
        reflectUrl: "http://api-proxy:10000/reflect",
        outputPath: "/tmp/gh-aw-test-noop.json",
        timeoutMs: 500,
        logger: msg => collected.push(msg),
      });
      expect(collected.length).toBeGreaterThan(0);
    });

    it("runs a curl probe when fetch fails with a Node.js fetch error", async () => {
      vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new TypeError("fetch failed")));
      const spawnSyncSpy = vi.spyOn(childProcess, "spawnSync").mockReturnValue({
        status: 0,
        stdout: "200",
        stderr: "",
      });
      const logs = [];

      await expect(
        fetchAWFReflect({
          reflectUrl: "http://api-proxy:10000/reflect",
          outputPath: "/tmp/gh-aw-test-noop.json",
          timeoutMs: 500,
          maxAttempts: 1,
          logger: msg => logs.push(msg),
        })
      ).resolves.toEqual({
        ok: false,
        reflectUrl: "http://api-proxy:10000/reflect",
        outputPath: "/tmp/gh-aw-test-noop.json",
        reason: "request_failed",
        error: "fetch failed",
      });

      expect(spawnSyncSpy).toHaveBeenCalledOnce();
      expect(spawnSyncSpy.mock.calls[0][0]).toBe("curl");
      expect(spawnSyncSpy.mock.calls[0][1]).toContain("http://api-proxy:10000/reflect");
      expect(logs.some(l => l.includes("running curl probe"))).toBe(true);
      expect(logs.some(l => l.includes("curl probe exit=0 http_status=200"))).toBe(true);
    });

    it("logs proxy environment variables before reflect fetch", async () => {
      vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new Error("ECONNREFUSED")));
      const originalProxyEnvs = Object.fromEntries(["HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY", "ALL_PROXY", "http_proxy", "https_proxy", "no_proxy", "all_proxy"].map(name => [name, process.env[name]]));
      process.env.HTTP_PROXY = "http://proxy.internal:3128";
      process.env.HTTPS_PROXY = "http://proxy-secure.internal:3129";
      process.env.NO_PROXY = "localhost,127.0.0.1,api-proxy";
      process.env.ALL_PROXY = "socks5://proxy-all.internal:1080";
      process.env.http_proxy = "http://proxy-lower.internal:3130";
      process.env.https_proxy = "http://proxy-lower-secure.internal:3131";
      process.env.no_proxy = "localhost";
      process.env.all_proxy = "socks5://proxy-lower-all.internal:1081";
      const logs = [];

      try {
        await fetchAWFReflect({
          reflectUrl: "http://api-proxy:10000/reflect",
          outputPath: "/tmp/gh-aw-test-noop.json",
          timeoutMs: 500,
          maxAttempts: 1,
          logger: msg => logs.push(msg),
        });
      } finally {
        for (const [name, value] of Object.entries(originalProxyEnvs)) {
          if (value === undefined) delete process.env[name];
          else process.env[name] = value;
        }
      }

      const proxyLogLine = logs.find(l => l.includes("awf-reflect: proxy env "));
      expect(proxyLogLine).toBeTruthy();
      expect(proxyLogLine).toContain('HTTP_PROXY="http://proxy.internal:3128"');
      expect(proxyLogLine).toContain('HTTPS_PROXY="http://proxy-secure.internal:3129"');
      expect(proxyLogLine).toContain('NO_PROXY="localhost,127.0.0.1,api-proxy"');
      expect(proxyLogLine).toContain('ALL_PROXY="socks5://proxy-all.internal:1080"');
      expect(proxyLogLine).toContain('http_proxy="http://proxy-lower.internal:3130"');
      expect(proxyLogLine).toContain('https_proxy="http://proxy-lower-secure.internal:3131"');
      expect(proxyLogLine).toContain('no_proxy="localhost"');
      expect(proxyLogLine).toContain('all_proxy="socks5://proxy-lower-all.internal:1081"');
    });
  });
});
