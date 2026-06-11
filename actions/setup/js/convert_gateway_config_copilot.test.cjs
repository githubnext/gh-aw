import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { resolveCopilotConfigOutputPath, transformCopilotEntry } from "./convert_gateway_config_copilot.cjs";

// -----------------------------------------------------------------------------
// resolveCopilotConfigOutputPath — guards against the regression where
// /home/runner was hard-coded and broke self-hosted runners with HOME set to
// anything other than /home/runner.
// -----------------------------------------------------------------------------
describe("convert_gateway_config_copilot resolveCopilotConfigOutputPath", () => {
  let originalHome;

  beforeEach(() => {
    originalHome = process.env.HOME;
  });

  afterEach(() => {
    if (originalHome === undefined) {
      delete process.env.HOME;
    } else {
      process.env.HOME = originalHome;
    }
  });

  it("resolves the Copilot MCP config path under the runtime $HOME", () => {
    process.env.HOME = "/home/runner";
    expect(resolveCopilotConfigOutputPath()).toBe("/home/runner/.copilot/mcp-config.json");
  });

  it("respects a self-hosted runner HOME (not /home/runner)", () => {
    process.env.HOME = "/home/actions";
    expect(resolveCopilotConfigOutputPath()).toBe("/home/actions/.copilot/mcp-config.json");
  });

  it("respects a containerized HOME (/root)", () => {
    process.env.HOME = "/root";
    expect(resolveCopilotConfigOutputPath()).toBe("/root/.copilot/mcp-config.json");
  });

  it("handles HOME with spaces and special characters via path.join", () => {
    process.env.HOME = "/var/lib/actions runner";
    expect(resolveCopilotConfigOutputPath()).toBe("/var/lib/actions runner/.copilot/mcp-config.json");
  });

  it("throws (not exits) when HOME is unset so tests can exercise the branch", () => {
    delete process.env.HOME;
    expect(() => resolveCopilotConfigOutputPath()).toThrow(/HOME environment variable is not set/);
  });

  it("throws when HOME is empty string", () => {
    process.env.HOME = "";
    expect(() => resolveCopilotConfigOutputPath()).toThrow(/HOME environment variable is not set/);
  });

  it("never returns a path containing the literal /home/runner when HOME is different", () => {
    process.env.HOME = "/opt/actions/home";
    const out = resolveCopilotConfigOutputPath();
    expect(out).not.toContain("/home/runner");
    expect(out).toBe("/opt/actions/home/.copilot/mcp-config.json");
  });
});

// -----------------------------------------------------------------------------
// transformCopilotEntry — pre-existing behavior, included for safety so the
// Copilot-specific transform stays covered when we touch this file.
// -----------------------------------------------------------------------------
describe("convert_gateway_config_copilot transformCopilotEntry", () => {
  it("adds tools:['*'] when not present", () => {
    const entry = { type: "http", url: "http://1.2.3.4:80/mcp/foo" };
    const out = transformCopilotEntry(entry, "http://host.docker.internal:8080");
    expect(out.tools).toEqual(["*"]);
  });

  it("preserves an existing tools field", () => {
    const entry = { type: "http", url: "http://1.2.3.4:80/mcp/foo", tools: ["read"] };
    const out = transformCopilotEntry(entry, "http://host.docker.internal:8080");
    expect(out.tools).toEqual(["read"]);
  });

  it("rewrites a string URL to the target prefix", () => {
    const entry = { type: "http", url: "http://0.0.0.0:80/mcp/foo" };
    const out = transformCopilotEntry(entry, "http://host.docker.internal:8080");
    expect(out.url).toBe("http://host.docker.internal:8080/mcp/foo");
  });

  it("leaves a non-string url alone", () => {
    const entry = { type: "stdio", command: "echo" };
    const out = transformCopilotEntry(entry, "http://host.docker.internal:8080");
    expect(out.command).toBe("echo");
    expect(out.url).toBeUndefined();
  });

  it("does not mutate the input entry", () => {
    const entry = { type: "http", url: "http://0.0.0.0:80/mcp/foo" };
    const snapshot = JSON.parse(JSON.stringify(entry));
    transformCopilotEntry(entry, "http://host.docker.internal:8080");
    expect(entry).toEqual(snapshot);
  });
});
