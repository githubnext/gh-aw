import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { providerSetupExtension } from "../../src/extensions/provider-setup.js";
import type { ExtensionAPI } from "@mariozechner/pi-coding-agent";

function buildMockPi(): {
  pi: ExtensionAPI;
  registered: Array<{ name: string; config: Record<string, unknown> }>;
} {
  const registered: Array<{ name: string; config: Record<string, unknown> }> = [];
  const pi = {
    on: vi.fn(),
    registerProvider: vi.fn((name: unknown, config: unknown) => {
      registered.push({ name: name as string, config: config as Record<string, unknown> });
    }),
  } as unknown as ExtensionAPI;
  return { pi, registered };
}

describe("providerSetupExtension", () => {
  const origEnv = { ...process.env };

  beforeEach(() => {
    // Clear provider keys before each test
    delete process.env["ANTHROPIC_API_KEY"];
    delete process.env["OPENAI_API_KEY"];
    delete process.env["GITHUB_TOKEN"];
    delete process.env["ANTHROPIC_BASE_URL"];
    delete process.env["OPENAI_BASE_URL"];
    delete process.env["COPILOT_BASE_URL"];
    vi.spyOn(process.stderr, "write").mockImplementation(() => true);
  });

  afterEach(() => {
    // Restore env
    Object.assign(process.env, origEnv);
    vi.restoreAllMocks();
  });

  it("throws when no provider credentials are set", () => {
    const { pi } = buildMockPi();
    expect(() => providerSetupExtension(pi)).toThrow(/No LLM provider credentials/);
  });

  it("registers anthropic provider when ANTHROPIC_API_KEY is set", () => {
    process.env["ANTHROPIC_API_KEY"] = "sk-ant-test";
    const { pi, registered } = buildMockPi();
    providerSetupExtension(pi);

    expect(registered.some((r) => r.name === "anthropic")).toBe(true);
    const anthropic = registered.find((r) => r.name === "anthropic");
    expect(anthropic?.config?.["apiKey"]).toBe("sk-ant-test");
    expect(anthropic?.config?.["api"]).toBe("anthropic-messages");
  });

  it("includes baseUrl when ANTHROPIC_BASE_URL is set", () => {
    process.env["ANTHROPIC_API_KEY"] = "sk-ant-test";
    process.env["ANTHROPIC_BASE_URL"] = "https://proxy.example.com";
    const { pi, registered } = buildMockPi();
    providerSetupExtension(pi);

    const anthropic = registered.find((r) => r.name === "anthropic");
    expect(anthropic?.config?.["baseUrl"]).toBe("https://proxy.example.com");
  });

  it("omits baseUrl when ANTHROPIC_BASE_URL is not set", () => {
    process.env["ANTHROPIC_API_KEY"] = "sk-ant-test";
    const { pi, registered } = buildMockPi();
    providerSetupExtension(pi);

    const anthropic = registered.find((r) => r.name === "anthropic");
    expect(anthropic?.config?.["baseUrl"]).toBeUndefined();
  });

  it("registers openai provider when OPENAI_API_KEY is set", () => {
    process.env["OPENAI_API_KEY"] = "sk-openai-test";
    const { pi, registered } = buildMockPi();
    providerSetupExtension(pi);

    const openai = registered.find((r) => r.name === "openai");
    expect(openai).toBeDefined();
    expect(openai?.config?.["apiKey"]).toBe("sk-openai-test");
  });

  it("registers copilot provider when GITHUB_TOKEN is set", () => {
    process.env["GITHUB_TOKEN"] = "ghs_test";
    const { pi, registered } = buildMockPi();
    providerSetupExtension(pi);

    const copilot = registered.find((r) => r.name === "copilot");
    expect(copilot).toBeDefined();
    expect(copilot?.config?.["apiKey"]).toBe("ghs_test");
  });

  it("registers multiple providers when multiple keys are set", () => {
    process.env["ANTHROPIC_API_KEY"] = "sk-ant";
    process.env["OPENAI_API_KEY"] = "sk-oai";
    const { pi, registered } = buildMockPi();
    providerSetupExtension(pi);

    expect(registered.length).toBeGreaterThanOrEqual(2);
    expect(registered.some((r) => r.name === "anthropic")).toBe(true);
    expect(registered.some((r) => r.name === "openai")).toBe(true);
  });
});
