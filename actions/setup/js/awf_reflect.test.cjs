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
  getCatalogModelEntry,
  inferProviderTypeForModel,
  inferWireApiForModel,
  resolveProviderEndpointFromReflect,
  resolveMultiProviderFromReflect,
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

    describe("resolveProviderEndpointFromReflect", () => {
      it("maps github provider to copilot endpoint", () => {
        const resolved = resolveProviderEndpointFromReflect({
          provider: "github",
          reflectData: {
            endpoints: [
              { provider: "openai", configured: true, port: 10000 },
              { provider: "copilot", configured: true, port: 10002, models_url: "http://api-proxy:10002/models" },
            ],
          },
          logger: () => {},
        });
        expect(resolved).toEqual({
          provider: "github",
          endpointProvider: "copilot",
          port: 10002,
          baseUrl: "http://api-proxy:10002",
        });
      });

      it("falls back to first configured endpoint when provider is not found", () => {
        const resolved = resolveProviderEndpointFromReflect({
          provider: "unknown-provider",
          reflectData: {
            endpoints: [{ provider: "openai", configured: true, port: 10000 }],
          },
          logger: () => {},
        });
        expect(resolved).toEqual({
          provider: "unknown-provider",
          endpointProvider: "openai",
          port: 10000,
          baseUrl: "http://api-proxy:10000",
        });
      });
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
      delete process.env.AWF_AUTH_TYPE;
      delete process.env.AWF_MODELS_URL_OIDC_INITIAL_DELAY_MS;
      vi.useRealTimers();
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

    it("delays initial probe for github-oidc auth when probing api-proxy", async () => {
      vi.useFakeTimers();
      process.env.AWF_AUTH_TYPE = "github-oidc";
      process.env.AWF_MODELS_URL_OIDC_INITIAL_DELAY_MS = "5000";

      const fetchMock = vi.fn().mockResolvedValue({ ok: true, status: 200, json: async () => ({ data: [{ id: "gpt-4o" }] }) });
      vi.stubGlobal("fetch", fetchMock);

      const logs = [];
      const run = fetchModelsFromUrl("http://api-proxy:10001/v1/models", 1000, msg => logs.push(msg));

      await vi.advanceTimersByTimeAsync(4999);
      expect(fetchMock).not.toHaveBeenCalled();

      await vi.advanceTimersByTimeAsync(1);
      await run;

      expect(fetchMock).toHaveBeenCalledTimes(1);
      expect(logs.some(l => l.includes("delaying initial models probe"))).toBe(true);
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

  describe("inferProviderTypeForModel", () => {
    it("returns 'anthropic' for anthropic endpoint provider", () => {
      expect(inferProviderTypeForModel("anthropic", "claude-sonnet-4.6", null)).toBe("anthropic");
    });

    it("returns 'azure' for azure endpoint provider", () => {
      expect(inferProviderTypeForModel("azure", "gpt-4o", null)).toBe("azure");
      expect(inferProviderTypeForModel("azure-openai", "gpt-4o", null)).toBe("azure");
    });

    it("returns 'openai' for openai endpoint provider", () => {
      expect(inferProviderTypeForModel("openai", "gpt-4o", null)).toBe("openai");
    });

    it("uses explicit copilot provider mapping (always openai)", () => {
      // GitHub Copilot provider is a multi-model proxy that always uses OpenAI wire protocol,
      // regardless of model name (even for claude/anthropic models)
      expect(inferProviderTypeForModel("copilot", "claude-sonnet-4.6", null)).toBe("openai");
      expect(inferProviderTypeForModel("copilot", "claude-opus-4-5", null)).toBe("openai");
      expect(inferProviderTypeForModel("github-copilot", "claude-haiku-4.5", null)).toBe("openai");
      // Non-copilot providers still use model name heuristics
      expect(inferProviderTypeForModel("", "claude-haiku-4.5", null)).toBe("anthropic");
    });

    it("uses model name heuristic for opus/haiku/sonnet suffix models when provider is not copilot", () => {
      // copilot provider always returns openai
      expect(inferProviderTypeForModel("copilot", "model-opus-4.6", null)).toBe("openai");
      expect(inferProviderTypeForModel("copilot", "model-haiku-4.5", null)).toBe("openai");
      expect(inferProviderTypeForModel("copilot", "model-sonnet-4", null)).toBe("openai");
      // Non-copilot providers use model name heuristics
      expect(inferProviderTypeForModel("", "model-opus-4.6", null)).toBe("anthropic");
      expect(inferProviderTypeForModel("", "model-haiku-4.5", null)).toBe("anthropic");
      expect(inferProviderTypeForModel("", "model-sonnet-4", null)).toBe("anthropic");
    });

    it("uses model name heuristic for gpt-* models", () => {
      expect(inferProviderTypeForModel("copilot", "gpt-5.4", null)).toBe("openai");
      expect(inferProviderTypeForModel("", "gpt-4o", null)).toBe("openai");
    });

    it("uses model name heuristic for o1/o3/o4 models", () => {
      expect(inferProviderTypeForModel("copilot", "o1-mini", null)).toBe("openai");
      expect(inferProviderTypeForModel("copilot", "o3-pro", null)).toBe("openai");
      expect(inferProviderTypeForModel("copilot", "o4-mini", null)).toBe("openai");
    });

    it("copilot provider always returns openai, even for anthropic models in catalog", () => {
      const modelsJson = {
        providers: {
          "github-copilot": {
            models: {
              "raptor-mini": { provider_type: "openai", cost: {} },
              "claude-sonnet-4": { provider_type: "anthropic", cost: {} },
            },
          },
        },
      };
      expect(inferProviderTypeForModel("copilot", "raptor-mini", modelsJson)).toBe("openai");
      // copilot provider mapping takes precedence over catalog provider_type
      expect(inferProviderTypeForModel("copilot", "claude-sonnet-4", modelsJson)).toBe("openai");
    });

    it("copilot provider always returns openai, even for anthropic model name heuristics", () => {
      const modelsJson = { providers: { "github-copilot": { models: {} } } };
      // copilot provider mapping takes precedence over model name heuristics
      expect(inferProviderTypeForModel("copilot", "claude-unknown-model", modelsJson)).toBe("openai");
    });

    it("returns 'openai' by default for unknown models", () => {
      expect(inferProviderTypeForModel("copilot", "gemini-2.5-pro", null)).toBe("openai");
      expect(inferProviderTypeForModel("", "raptor-mini", null)).toBe("openai");
    });
  });

  describe("getCatalogModelEntry", () => {
    it("matches model names case-insensitively", () => {
      const entry = getCatalogModelEntry(
        {
          providers: {
            "github-copilot": { models: { "gpt-5.5": { provider_type: "openai", wire_api: "responses", cost: {} } } },
          },
        },
        "GPT-5.5"
      );
      expect(entry).toEqual({ provider_type: "openai", wire_api: "responses", cost: {} });
    });

    it("uses the requested provider when duplicate model names exist", () => {
      const modelsJson = {
        providers: {
          openai: { models: { "gpt-5.5": { provider_type: "openai", cost: {} } } },
          "github-copilot": { models: { "gpt-5.5": { provider_type: "openai", wire_api: "responses", cost: {} } } },
        },
      };
      expect(getCatalogModelEntry(modelsJson, "gpt-5.5", "github-copilot")).toEqual({
        provider_type: "openai",
        wire_api: "responses",
        cost: {},
      });
      expect(getCatalogModelEntry(modelsJson, "gpt-5.5", "openai")).toEqual({
        provider_type: "openai",
        cost: {},
      });
    });

    it("returns null for invalid catalog entries", () => {
      expect(
        getCatalogModelEntry(
          {
            providers: {
              "github-copilot": { models: { broken: null, arrayish: [] } },
            },
          },
          "broken"
        )
      ).toBeNull();
      expect(
        getCatalogModelEntry(
          {
            providers: {
              "github-copilot": { models: { broken: null, arrayish: [] } },
            },
          },
          "arrayish"
        )
      ).toBeNull();
    });
  });

  describe("inferWireApiForModel", () => {
    it("omits wireApi for Anthropic providers even when the catalog requests one", () => {
      expect(inferWireApiForModel("anthropic", "claude-opus-5", { wire_api: "responses" })).toBeUndefined();
    });

    it("falls back to completions when the catalog value is invalid or absent", () => {
      expect(inferWireApiForModel("openai", "gpt-5.5", { wire_api: "grpc" })).toBe("completions");
      expect(inferWireApiForModel("openai", "gpt-5.5", null)).toBe("completions");
    });
  });

  describe("resolveMultiProviderFromReflect", () => {
    it("returns null when reflectData is null", () => {
      const logs = [];
      const result = resolveMultiProviderFromReflect({ reflectData: null, logger: msg => logs.push(msg) });
      expect(result).toBeNull();
      expect(logs.some(l => l.includes("no reflect data provided"))).toBe(true);
    });

    it("resolves with a single configured endpoint", () => {
      const result = resolveMultiProviderFromReflect({
        reflectData: { endpoints: [{ provider: "copilot", port: 10002, configured: true, models: ["gpt-5.4"] }] },
      });
      expect(result).not.toBeNull();
      expect(result.providers).toHaveLength(1);
      expect(result.model).toBe("gpt-5.4");
    });

    it("returns null when no configured endpoints exist", () => {
      const result = resolveMultiProviderFromReflect({
        reflectData: {
          endpoints: [
            { provider: "openai", port: 10001, configured: false, models: ["gpt-4o"] },
            { provider: "anthropic", port: 10002, configured: false, models: ["claude-sonnet-4.6"] },
          ],
        },
      });
      expect(result).toBeNull();
    });

    it("builds providers and models from two configured endpoints", () => {
      const reflectData = {
        endpoints: [
          { provider: "openai", port: 10001, configured: true, models: ["gpt-4o"] },
          { provider: "anthropic", port: 10002, configured: true, models: ["claude-sonnet-4.6"] },
        ],
      };
      const result = resolveMultiProviderFromReflect({ reflectData });
      expect(result).not.toBeNull();
      expect(result.providers).toHaveLength(2);
      expect(result.models).toHaveLength(2);
      expect(result.providers[0]).toMatchObject({ name: "openai", type: "openai", baseUrl: "http://api-proxy:10001", wireApi: "completions" });
      expect(result.providers[1]).toMatchObject({ name: "anthropic", type: "anthropic", baseUrl: "http://api-proxy:10002" });
      expect(result.providers[1]).not.toHaveProperty("wireApi");
      expect(result.models[0]).toEqual({ id: "gpt-4o", provider: "openai" });
      expect(result.models[1]).toEqual({ id: "claude-sonnet-4.6", provider: "anthropic" });
    });

    it("sets primary model to first model when no configured model provided", () => {
      const reflectData = {
        endpoints: [
          { provider: "openai", port: 10001, configured: true, models: ["gpt-5.4"] },
          { provider: "anthropic", port: 10002, configured: true, models: ["claude-sonnet-4.6"] },
        ],
      };
      const result = resolveMultiProviderFromReflect({ reflectData });
      expect(result.model).toBe("gpt-5.4");
    });

    it("prefers the configured model when it appears in model list", () => {
      const reflectData = {
        endpoints: [
          { provider: "openai", port: 10001, configured: true, models: ["gpt-5.4"] },
          { provider: "anthropic", port: 10002, configured: true, models: ["claude-sonnet-4.6"] },
        ],
      };
      const result = resolveMultiProviderFromReflect({ reflectData, model: "claude-sonnet-4.6" });
      expect(result.model).toBe("claude-sonnet-4.6");
    });

    it("falls back to first model when configured model is not found in list", () => {
      const reflectData = {
        endpoints: [
          { provider: "openai", port: 10001, configured: true, models: ["gpt-5.4"] },
          { provider: "anthropic", port: 10002, configured: true, models: ["claude-sonnet-4.6"] },
        ],
      };
      const result = resolveMultiProviderFromReflect({ reflectData, model: "nonexistent-model" });
      expect(result.model).toBe("gpt-5.4");
    });

    it("derives provider baseUrl from models_url origin when available", () => {
      const reflectData = {
        endpoints: [
          { provider: "openai", port: 10001, configured: true, models: ["gpt-4o"], models_url: "http://172.30.0.10:10001/v1/models" },
          { provider: "anthropic", port: 10002, configured: true, models: ["claude-sonnet-4.6"], models_url: "http://172.30.0.11:10002/v1/models" },
        ],
      };
      const result = resolveMultiProviderFromReflect({ reflectData });
      expect(result.providers[0].baseUrl).toBe("http://172.30.0.10:10001");
      expect(result.providers[1].baseUrl).toBe("http://172.30.0.11:10002");
    });

    it("infers openai wireApi from modelsJson catalog for openai endpoint", () => {
      const reflectData = {
        endpoints: [
          { provider: "copilot", port: 10002, configured: true, models: ["gpt-5.5"] },
          { provider: "anthropic", port: 10003, configured: true, models: ["claude-sonnet-4.6"] },
        ],
      };
      const modelsJson = {
        providers: {
          "github-copilot": { models: { "gpt-5.5": { provider_type: "openai", wire_api: "responses", cost: {} } } },
        },
      };
      const result = resolveMultiProviderFromReflect({ reflectData, modelsJson });
      expect(result.providers[0]).toMatchObject({ name: "copilot", type: "openai", wireApi: "responses" });
      expect(result.providers[1]).toMatchObject({ name: "anthropic", type: "anthropic" });
      expect(result.providers[1]).not.toHaveProperty("wireApi");
    });

    it("handles duplicate provider names by appending a numeric suffix", () => {
      const reflectData = {
        endpoints: [
          { provider: "copilot", port: 10001, configured: true, models: ["gpt-5.4"] },
          { provider: "copilot", port: 10002, configured: true, models: ["gpt-5.5"] },
        ],
      };
      const result = resolveMultiProviderFromReflect({ reflectData });
      expect(result.providers[0].name).toBe("copilot");
      expect(result.providers[1].name).toBe("copilot-1");
      expect(result.models[0]).toEqual({ id: "gpt-5.4", provider: "copilot" });
      expect(result.models[1]).toEqual({ id: "gpt-5.5", provider: "copilot-1" });
    });

    it("skips endpoints with no resolvable baseUrl", () => {
      const logs = [];
      const reflectData = {
        endpoints: [
          { provider: "openai", port: 10001, configured: true, models: ["gpt-4o"] },
          // no port and no models_url — skipped
          { provider: "anthropic", configured: true, models: ["claude-sonnet-4.6"] },
          { provider: "azure", port: 10003, configured: true, models: ["gpt-4o-azure"] },
        ],
      };
      const result = resolveMultiProviderFromReflect({ reflectData, logger: msg => logs.push(msg) });
      expect(result).not.toBeNull();
      expect(result.providers).toHaveLength(2);
      expect(result.providers.map(p => p.name)).toEqual(["openai", "azure"]);
      expect(logs.some(l => l.includes("no resolvable baseUrl"))).toBe(true);
    });

    it("collects all models from all endpoints", () => {
      const reflectData = {
        endpoints: [
          { provider: "openai", port: 10001, configured: true, models: ["gpt-4o", "gpt-5.4"] },
          { provider: "anthropic", port: 10002, configured: true, models: ["claude-sonnet-4.6", "claude-opus-5"] },
        ],
      };
      const result = resolveMultiProviderFromReflect({ reflectData });
      expect(result.models).toHaveLength(4);
      expect(result.models).toContainEqual({ id: "gpt-4o", provider: "openai" });
      expect(result.models).toContainEqual({ id: "gpt-5.4", provider: "openai" });
      expect(result.models).toContainEqual({ id: "claude-sonnet-4.6", provider: "anthropic" });
      expect(result.models).toContainEqual({ id: "claude-opus-5", provider: "anthropic" });
    });
  });
});
