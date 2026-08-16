import fs from "fs";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  applyOTLPIgnoreIfMissing,
  detectEngineType,
  extractOptionalServerNames,
  getJSONParseErrorContext,
  getOTLPIfMissingMode,
  hasNonEmptyOTLPHeaders,
  injectCustomGatewayEnvArgs,
  normalizeSinkVisibilityEncoding,
  resolveCopilotConfigPaths,
} from "./start_mcp_gateway.cjs";

describe("start_mcp_gateway logging", () => {
  it("does not create the legacy MCP gateway stderr log", () => {
    const source = fs.readFileSync(new URL("./start_mcp_gateway.cjs", import.meta.url), "utf8");
    expect(source).not.toContain("/tmp/gh-aw/mcp-logs/stderr.log");
  });

  it("does not create the MCP gateway startup log", () => {
    const source = fs.readFileSync(new URL("./start_mcp_gateway.cjs", import.meta.url), "utf8");
    expect(source).not.toContain("/tmp/gh-aw/mcp-logs/start-gateway.log");
  });

  it("discards the gateway child process stderr", () => {
    const source = fs.readFileSync(new URL("./start_mcp_gateway.cjs", import.meta.url), "utf8");
    expect(source).toContain(`stdio: ["pipe", outputFd, "ignore"]`);
  });
});

describe("start_mcp_gateway custom environment arguments", () => {
  const marker = "__GH_AW_MCP_GATEWAY_CUSTOM_ENV__";

  it("passes hostile values as one atomic Docker argument", () => {
    const hostileValue = `x" --privileged -v /workspace/evil:/evil --entrypoint /evil -e X="x`;
    const args = injectCustomGatewayEnvArgs(["run", "--rm", marker, "gateway-image"], {
      GH_AW_MCP_GATEWAY_CUSTOM_ENV_NAMES: '["BASH_ENV"]',
      GH_AW_MCP_GATEWAY_ENV_0: hostileValue,
    });

    expect(args).toEqual(["run", "--rm", "-e", `BASH_ENV=${hostileValue}`, "gateway-image"]);
  });

  it("preserves sorted multi-value index mapping and empty values", () => {
    const args = injectCustomGatewayEnvArgs(["run", marker, "gateway-image"], {
      GH_AW_MCP_GATEWAY_CUSTOM_ENV_NAMES: '["ALPHA","EMPTY","OMEGA"]',
      GH_AW_MCP_GATEWAY_ENV_0: "first",
      GH_AW_MCP_GATEWAY_ENV_1: "",
      GH_AW_MCP_GATEWAY_ENV_2: "last\nline",
    });

    expect(args).toEqual(["run", "-e", "ALPHA=first", "-e", "EMPTY=", "-e", "OMEGA=last\nline", "gateway-image"]);
  });

  it("uses an empty value when transport metadata is missing", () => {
    const args = injectCustomGatewayEnvArgs(["run", marker, "gateway-image"], {
      GH_AW_MCP_GATEWAY_CUSTOM_ENV_NAMES: '["PRESENT","MISSING"]',
      GH_AW_MCP_GATEWAY_ENV_0: "value",
    });

    expect(args).toEqual(["run", "-e", "PRESENT=value", "-e", "MISSING=", "gateway-image"]);
  });

  it("leaves commands without the marker unchanged", () => {
    const args = ["run", "--rm", "gateway-image"];
    expect(injectCustomGatewayEnvArgs(args, { GH_AW_MCP_GATEWAY_CUSTOM_ENV_NAMES: "not-json" })).toBe(args);
  });

  it("rejects malformed JSON metadata", () => {
    expect(() =>
      injectCustomGatewayEnvArgs(["run", marker, "gateway-image"], {
        GH_AW_MCP_GATEWAY_CUSTOM_ENV_NAMES: "not-json",
      })
    ).toThrow(/must be valid JSON/);
  });

  it("rejects malformed or unsafe environment variable names", () => {
    expect(() =>
      injectCustomGatewayEnvArgs(["run", marker, "gateway-image"], {
        GH_AW_MCP_GATEWAY_CUSTOM_ENV_NAMES: '["BAD-NAME"]',
      })
    ).toThrow(/valid environment variable names/);
  });

  it.each([["GH_AW_MCP_GATEWAY_ENV_0"], ["GH_AW_MCP_GATEWAY_CUSTOM_ENV_NAMES"]])("rejects the reserved transport name %s", reservedName => {
    expect(() =>
      injectCustomGatewayEnvArgs(["run", marker, "gateway-image"], {
        GH_AW_MCP_GATEWAY_CUSTOM_ENV_NAMES: JSON.stringify([reservedName]),
      })
    ).toThrow(/reserved/);
  });

  it("rejects duplicate environment variable names", () => {
    expect(() =>
      injectCustomGatewayEnvArgs(["run", marker, "gateway-image"], {
        GH_AW_MCP_GATEWAY_CUSTOM_ENV_NAMES: '["API_TOKEN","API_TOKEN"]',
      })
    ).toThrow(/duplicate/);
  });
});

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

describe("start_mcp_gateway getJSONParseErrorContext", () => {
  it("extracts line/column and key for invalid escape values", () => {
    const invalidConfig = `{
  "mcpServers": {
    "github": {
      "env": {
        "GITHUB_HOST": "\\https://github.com"
      }
    }
  }
}`;
    let parseErrorMessage = "";
    try {
      JSON.parse(invalidConfig);
    } catch (err) {
      parseErrorMessage = /** @type {Error} */ err.message;
    }
    const context = getJSONParseErrorContext(invalidConfig, parseErrorMessage);
    expect(context).toBeTruthy();
    expect(context?.key).toBe("GITHUB_HOST");
    expect(context?.lineText).toContain(`"GITHUB_HOST"`);
  });
});

describe("start_mcp_gateway normalizeSinkVisibilityEncoding", () => {
  it("normalizes double-encoded sink visibility values", () => {
    const invalidConfig = `{
  "guard-policies": {
    "write-sink": {
      "sink-visibility": ""public""
    }
  }
}`;
    expect(normalizeSinkVisibilityEncoding(invalidConfig)).toContain(`"sink-visibility": "public"`);
  });

  it("leaves correctly encoded sink visibility values unchanged", () => {
    const validConfig = `{
  "guard-policies": {
    "write-sink": {
      "sink-visibility": "public"
    }
  }
}`;
    expect(normalizeSinkVisibilityEncoding(validConfig)).toBe(validConfig);
  });
});

describe("start_mcp_gateway extractOptionalServerNames", () => {
  it("collects servers declared with required: false and strips the flag from every server", () => {
    const configObj = {
      mcpServers: {
        datadog: { type: "http", url: "https://example.com/mcp", required: false },
        grafana: { type: "http", url: "https://example.com/grafana" },
        sentry: { type: "http", url: "https://example.com/sentry", required: true },
      },
    };

    expect(extractOptionalServerNames(configObj)).toEqual(["datadog"]);
    // The gateway configuration specification has no `required` field, so it is
    // removed for every server regardless of its value.
    expect(configObj.mcpServers.datadog).not.toHaveProperty("required");
    expect(configObj.mcpServers.sentry).not.toHaveProperty("required");
    expect(configObj.mcpServers.grafana).not.toHaveProperty("required");
  });

  it("returns an empty list when no servers are configured", () => {
    expect(extractOptionalServerNames({})).toEqual([]);
    expect(extractOptionalServerNames({ mcpServers: null })).toEqual([]);
  });
});
