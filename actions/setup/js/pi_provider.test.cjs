import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

describe("pi_provider.cjs", () => {
  let module;
  let originalEnv;
  let originalFetch;
  let stderrOutput;

  beforeEach(async () => {
    originalEnv = { ...process.env };
    originalFetch = global.fetch;
    stderrOutput = [];
    vi.spyOn(process.stderr, "write").mockImplementation(msg => {
      stderrOutput.push(String(msg));
      return true;
    });
    module = await import("./pi_provider.cjs?" + Date.now());
  });

  afterEach(() => {
    process.env = originalEnv;
    global.fetch = originalFetch;
    vi.restoreAllMocks();
  });

  it("prefers GH_AW_PI_MODEL over PI_MODEL", () => {
    process.env.GH_AW_PI_MODEL = "copilot/claude-sonnet-4";
    process.env.PI_MODEL = "anthropic/claude-opus-4";

    expect(module.getConfiguredModel()).toBe("copilot/claude-sonnet-4");
  });

  it("registers configured providers and aliases from the environment", () => {
    process.env.COPILOT_GITHUB_TOKEN = "copilot-token";
    process.env.GITHUB_COPILOT_BASE_URL = "https://copilot.example.test";
    process.env.ANTHROPIC_API_KEY = "anthropic-token";
    process.env.ANTHROPIC_BASE_URL = "https://anthropic.example.test";
    process.env.CODEX_API_KEY = "codex-token";
    process.env.OPENAI_BASE_URL = "https://openai.example.test";

    const calls = [];
    const pi = {
      registerProvider: vi.fn((name, config) => {
        calls.push([name, config]);
      }),
      on: vi.fn(),
    };

    const count = module.registerConfiguredProviders(pi, () => {});

    expect(count).toBe(5);
    expect(calls).toEqual([
      ["github-copilot", { apiKey: "copilot-token", api: "openai-completions", baseUrl: "https://copilot.example.test" }],
      ["copilot", { apiKey: "copilot-token", api: "openai-completions", baseUrl: "https://copilot.example.test" }],
      ["anthropic", { apiKey: "anthropic-token", api: "anthropic", baseUrl: "https://anthropic.example.test" }],
      ["openai", { apiKey: "codex-token", api: "openai-completions", baseUrl: "https://openai.example.test" }],
      ["codex", { apiKey: "codex-token", api: "openai-completions", baseUrl: "https://openai.example.test" }],
    ]);
  });

  it("logs the configured provider using GH_AW_PI_MODEL during agent_start", async () => {
    process.env.GH_AW_PI_MODEL = "copilot/claude-sonnet-4";
    global.fetch = vi.fn().mockRejectedValue(new Error("network disabled"));

    const handlers = {};
    const pi = {
      registerProvider: vi.fn(),
      on: vi.fn((event, handler) => {
        handlers[event] = handler;
      }),
    };

    module.default(pi);
    await handlers.agent_start();

    expect(stderrOutput.some(line => line.includes("provider=copilot model=copilot/claude-sonnet-4"))).toBe(true);
  });

  it("calls /reflect on the Copilot gateway port (10002) for a copilot model", async () => {
    process.env.GH_AW_PI_MODEL = "copilot/claude-sonnet-4";
    const fetchedUrls = [];
    global.fetch = vi.fn().mockImplementation(url => {
      fetchedUrls.push(url);
      return Promise.reject(new Error("network disabled"));
    });

    const handlers = {};
    const pi = {
      registerProvider: vi.fn(),
      on: vi.fn((event, handler) => {
        handlers[event] = handler;
      }),
    };

    module.default(pi);
    await handlers.agent_start();
    await handlers.agent_end();

    expect(fetchedUrls.every(url => url === "http://api-proxy:10002/reflect")).toBe(true);
    expect(fetchedUrls.length).toBe(2);
  });

  it("calls /reflect on the Anthropic gateway port (10001) for an anthropic model", async () => {
    process.env.GH_AW_PI_MODEL = "anthropic/claude-opus-4";
    const fetchedUrls = [];
    global.fetch = vi.fn().mockImplementation(url => {
      fetchedUrls.push(url);
      return Promise.reject(new Error("network disabled"));
    });

    const handlers = {};
    const pi = {
      registerProvider: vi.fn(),
      on: vi.fn((event, handler) => {
        handlers[event] = handler;
      }),
    };

    module.default(pi);
    await handlers.agent_start();

    expect(fetchedUrls[0]).toBe("http://api-proxy:10001/reflect");
  });

  it("defaults to Copilot gateway port (10002) when no provider prefix is set", async () => {
    process.env.GH_AW_PI_MODEL = "my-custom-model";
    const fetchedUrls = [];
    global.fetch = vi.fn().mockImplementation(url => {
      fetchedUrls.push(url);
      return Promise.reject(new Error("network disabled"));
    });

    const handlers = {};
    const pi = {
      registerProvider: vi.fn(),
      on: vi.fn((event, handler) => {
        handlers[event] = handler;
      }),
    };

    module.default(pi);
    await handlers.agent_start();

    expect(fetchedUrls[0]).toBe("http://api-proxy:10002/reflect");
  });

  it("falls back to Copilot gateway port (10002) and logs a warning for an unknown provider", async () => {
    process.env.GH_AW_PI_MODEL = "unknown-provider/my-model";
    const fetchedUrls = [];
    global.fetch = vi.fn().mockImplementation(url => {
      fetchedUrls.push(url);
      return Promise.reject(new Error("network disabled"));
    });

    const handlers = {};
    const pi = {
      registerProvider: vi.fn(),
      on: vi.fn((event, handler) => {
        handlers[event] = handler;
      }),
    };

    module.default(pi);
    await handlers.agent_start();

    // Falls back to Copilot gateway (port 10002) and emits a warning at load time.
    expect(fetchedUrls[0]).toBe("http://api-proxy:10002/reflect");
    expect(stderrOutput.some(line => line.includes("no known AWF gateway port") && line.includes("falling back to Copilot gateway"))).toBe(true);
  });
});
