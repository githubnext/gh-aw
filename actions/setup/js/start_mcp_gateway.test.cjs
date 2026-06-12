import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { applyOTLPIgnoreIfMissing, detectEngineType, getOTLPIfMissingMode, hasNonEmptyOTLPHeaders, resolveCopilotConfigPaths } from "./start_mcp_gateway.cjs";

describe("start_mcp_gateway OTLP if-missing helpers", () => {
  let originalWarning;

  beforeEach(() => {
    originalWarning = global.core.warning;
    global.core.warning = vi.fn();
  });

  afterEach(() => {
    delete process.env.GH_AW_OTLP_IF_MISSING;
    global.core.warning = originalWarning;
  });

  it("normalizes if-missing mode", () => {
    expect(getOTLPIfMissingMode(undefined)).toBe("error");
    expect(getOTLPIfMissingMode(" warn ")).toBe("warn");
    expect(getOTLPIfMissingMode("ignore")).toBe("ignore");
    expect(getOTLPIfMissingMode("invalid")).toBe("error");
  });

  it("detects non-empty OTLP headers for string/map/array forms", () => {
    expect(hasNonEmptyOTLPHeaders("")).toBe(false);
    expect(hasNonEmptyOTLPHeaders("Authorization=Bearer token")).toBe(true);
    expect(hasNonEmptyOTLPHeaders({ Authorization: "" })).toBe(false);
    expect(hasNonEmptyOTLPHeaders({ Authorization: "Bearer token" })).toBe(true);
    expect(hasNonEmptyOTLPHeaders(["", "  "])).toBe(false);
    expect(hasNonEmptyOTLPHeaders(["", "token"])).toBe(true);
  });

  it("is a no-op when if-missing mode is unset/error", () => {
    const config = {
      gateway: {
        opentelemetry: {
          endpoint: "   ",
          headers: "",
        },
      },
    };
    applyOTLPIgnoreIfMissing(config);
    expect(config.gateway.opentelemetry).toEqual({
      endpoint: "   ",
      headers: "",
    });
  });

  it("removes opentelemetry when endpoint is empty for warn mode and emits a warning", () => {
    const warningSpy = vi.fn();
    global.core.warning = warningSpy;
    process.env.GH_AW_OTLP_IF_MISSING = "warn";

    const config = {
      gateway: {
        opentelemetry: {
          endpoint: "   ",
          headers: { Authorization: "" },
        },
      },
    };

    applyOTLPIgnoreIfMissing(config);

    expect(config.gateway.opentelemetry).toBeUndefined();
    expect(warningSpy).toHaveBeenCalledOnce();
    expect(warningSpy).toHaveBeenCalledWith(expect.stringContaining("OTLP endpoint is missing/empty"));
  });

  it("removes empty headers object for warn mode and emits a warning", () => {
    const warningSpy = vi.fn();
    global.core.warning = warningSpy;
    process.env.GH_AW_OTLP_IF_MISSING = "warn";

    const config = {
      gateway: {
        opentelemetry: {
          endpoint: "https://collector.example/v1/traces",
          headers: { Authorization: "", "X-Tenant": "   " },
        },
      },
    };

    applyOTLPIgnoreIfMissing(config);

    expect(config.gateway.opentelemetry.headers).toBeUndefined();
    expect(warningSpy).toHaveBeenCalledOnce();
    expect(warningSpy).toHaveBeenCalledWith(expect.stringContaining("OTLP headers are missing/empty"));
  });

  it("removes empty headers object for ignore mode without warning", () => {
    const warningSpy = vi.fn();
    global.core.warning = warningSpy;
    process.env.GH_AW_OTLP_IF_MISSING = "ignore";

    const config = {
      gateway: {
        opentelemetry: {
          endpoint: "https://collector.example/v1/traces",
          headers: { Authorization: "" },
        },
      },
    };

    applyOTLPIgnoreIfMissing(config);

    expect(config.gateway.opentelemetry.headers).toBeUndefined();
    expect(warningSpy).not.toHaveBeenCalled();
  });
});

// -----------------------------------------------------------------------------
// resolveCopilotConfigPaths — guards against the regression where /home/runner
// was hard-coded and broke self-hosted runners with HOME != /home/runner.
// -----------------------------------------------------------------------------
describe("start_mcp_gateway resolveCopilotConfigPaths", () => {
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

  it("resolves the Copilot config dir under the runtime $HOME", () => {
    process.env.HOME = "/home/runner";
    expect(resolveCopilotConfigPaths()).toEqual({
      dir: "/home/runner/.copilot",
      file: "/home/runner/.copilot/mcp-config.json",
    });
  });

  it("respects a self-hosted runner HOME (not /home/runner)", () => {
    process.env.HOME = "/home/actions";
    expect(resolveCopilotConfigPaths()).toEqual({
      dir: "/home/actions/.copilot",
      file: "/home/actions/.copilot/mcp-config.json",
    });
  });

  it("respects a containerized HOME (/root)", () => {
    process.env.HOME = "/root";
    expect(resolveCopilotConfigPaths()).toEqual({
      dir: "/root/.copilot",
      file: "/root/.copilot/mcp-config.json",
    });
  });

  it("handles HOME with spaces and special characters via path.join", () => {
    process.env.HOME = "/var/lib/actions runner";
    expect(resolveCopilotConfigPaths()).toEqual({
      dir: "/var/lib/actions runner/.copilot",
      file: "/var/lib/actions runner/.copilot/mcp-config.json",
    });
  });

  it("throws (not exits) when HOME is unset so tests can exercise the branch", () => {
    delete process.env.HOME;
    expect(() => resolveCopilotConfigPaths()).toThrow(/HOME environment variable is not set/);
  });

  it("throws when HOME is empty string", () => {
    process.env.HOME = "";
    expect(() => resolveCopilotConfigPaths()).toThrow(/HOME environment variable is not set/);
  });

  it("never returns a path containing the literal /home/runner when HOME is different", () => {
    process.env.HOME = "/opt/actions/home";
    const { dir, file } = resolveCopilotConfigPaths();
    expect(dir).not.toContain("/home/runner");
    expect(file).not.toContain("/home/runner");
  });
});

describe("start_mcp_gateway detectEngineType", () => {
  const configDir = "/tmp/gh-aw/mcp-config";

  it("does not require HOME for an explicit non-copilot engine", () => {
    expect(detectEngineType(configDir, { GH_AW_ENGINE: "codex" }, () => false)).toBe("codex");
  });

  it("does not require HOME when auto-detecting codex", () => {
    const existsSync = vi.fn(p => p === `${configDir}/config.toml`);
    expect(detectEngineType(configDir, {}, existsSync)).toBe("codex");
    expect(existsSync).not.toHaveBeenCalledWith("/.copilot");
  });

  it("auto-detects copilot from the HOME-scoped config directory", () => {
    const env = { HOME: "/var/lib/actions runner" };
    const existsSync = vi.fn(p => p === "/var/lib/actions runner/.copilot");
    expect(detectEngineType(configDir, env, existsSync)).toBe("copilot");
  });
});
