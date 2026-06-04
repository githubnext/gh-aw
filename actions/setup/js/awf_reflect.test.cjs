import { afterEach, describe, it, expect, vi } from "vitest";
import { createRequire } from "module";
import fs from "fs";
import os from "os";
import path from "path";

const require = createRequire(import.meta.url);
const {
  AWF_API_PROXY_REFLECT_URL,
  AWF_REFLECT_OUTPUT_PATH,
  AWF_REFLECT_TIMEOUT_MS,
  AWF_MODELS_URL_TIMEOUT_MS,
  AWF_MODELS_URL_MAX_ATTEMPTS,
  AWF_MODELS_URL_RETRY_BASE_MS,
  AWF_MODELS_URL_RETRY_MAX_MS,
  GEMINI_MODEL_NAME_PREFIX,
  enrichReflectModels,
  extractModelIds,
  fetchAWFReflect,
  fetchModelsFromUrl,
  resolveCopilotSDKCustomProviderFromReflect,
} = require("./awf_reflect.cjs");

describe("awf_reflect.cjs", () => {
  describe("constants", () => {
    it("exports expected default values", () => {
      expect(AWF_API_PROXY_REFLECT_URL).toBe("http://api-proxy:10000/reflect");
      expect(AWF_REFLECT_OUTPUT_PATH).toBe("/tmp/gh-aw/sandbox/firewall/awf-reflect.json");
      expect(AWF_REFLECT_TIMEOUT_MS).toBe(60000);
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
          reflectData: expect.objectContaining({ endpoints: expect.any(Array) }),
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

    it("does not throw when the reflect endpoint returns non-ok status", async () => {
      vi.stubGlobal("fetch", vi.fn().mockResolvedValue({ ok: false, status: 503 }));
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
        status: 503,
      });
      expect(logs.some(l => l.includes("unexpected status 503"))).toBe(true);
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
  });

  describe("resolveCopilotSDKCustomProviderFromReflect", () => {
    it("resolves provider baseUrl and model from port when models_url is absent", () => {
      const reflectData = {
        endpoints: [{ provider: "copilot", port: 10002, configured: true, models: ["gpt-5.4", "claude-sonnet-4.6"] }],
      };
      expect(resolveCopilotSDKCustomProviderFromReflect({ reflectData })).toEqual({
        model: "gpt-5.4",
        provider: { type: "openai", baseUrl: "http://api-proxy:10002" },
      });
    });

    it("prefers the endpoint matching the configured model", () => {
      const reflectData = {
        endpoints: [
          { provider: "openai", port: 10001, configured: true, models: ["gpt-4o"] },
          { provider: "anthropic", port: 10002, configured: true, models: ["claude-sonnet-4.6"] },
        ],
      };
      expect(resolveCopilotSDKCustomProviderFromReflect({ reflectData, model: "claude-sonnet-4.6" })).toEqual({
        model: "claude-sonnet-4.6",
        provider: { type: "openai", baseUrl: "http://api-proxy:10002" },
      });
    });

    it("derives baseUrl from models_url origin when available", () => {
      const reflectData = {
        endpoints: [{ provider: "copilot", port: 10002, configured: true, models: ["gpt-4o"], models_url: "http://172.30.0.30:10002/v1/models" }],
      };
      expect(resolveCopilotSDKCustomProviderFromReflect({ reflectData })).toEqual({
        model: "gpt-4o",
        provider: { type: "openai", baseUrl: "http://172.30.0.30:10002" },
      });
    });

    it("returns null when no configured endpoints exist", () => {
      const logs = [];
      const result = resolveCopilotSDKCustomProviderFromReflect({
        reflectData: { endpoints: [{ provider: "copilot", port: 10002, configured: false, models: [] }] },
        logger: msg => logs.push(msg),
      });
      expect(result).toBeNull();
      expect(logs.some(l => l.includes("no configured endpoints"))).toBe(true);
    });

    it("returns null when reflectData is null", () => {
      const logs = [];
      const result = resolveCopilotSDKCustomProviderFromReflect({
        reflectData: null,
        logger: msg => logs.push(msg),
      });
      expect(result).toBeNull();
      expect(logs.some(l => l.includes("no reflect data provided"))).toBe(true);
    });

    it("returns null when reflectData is undefined", () => {
      const logs = [];
      const result = resolveCopilotSDKCustomProviderFromReflect({
        reflectData: undefined,
        logger: msg => logs.push(msg),
      });
      expect(result).toBeNull();
      expect(logs.some(l => l.includes("no reflect data provided"))).toBe(true);
    });
  });
});
