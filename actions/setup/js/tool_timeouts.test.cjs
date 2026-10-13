import fs from "node:fs";
import path from "node:path";
import Ajv from "ajv";
import { describe, expect, it } from "vitest";
const { timeoutMilliseconds, applyClaudeRuntimeTimeouts, applyGatewayRuntimeTimeouts } = require("./tool_timeouts.cjs");

const schemaPath = path.resolve(__dirname, "../../../docs/public/schemas/mcp-gateway-config.schema.json");
const schema = JSON.parse(fs.readFileSync(schemaPath, "utf8"));
const ajv = new Ajv({ allErrors: true, strict: false, formats: { uri: true } });
const validateGatewayConfig = ajv.compile(schema);

describe("runtime tool timeouts", () => {
  it("converts evaluated workflow inputs into every Claude native timeout", () => {
    const env = { GH_AW_STARTUP_TIMEOUT: "180", GH_AW_TOOL_TIMEOUT: "90", MCP_TIMEOUT: "120000", MCP_TOOL_TIMEOUT: "60000" };
    applyClaudeRuntimeTimeouts(env);
    expect(env).toMatchObject({ MCP_TIMEOUT: "180000", MCP_TOOL_TIMEOUT: "90000", BASH_DEFAULT_TIMEOUT_MS: "90000", BASH_MAX_TIMEOUT_MS: "90000" });
  });
  it("preserves defaults and explicit native values when no workflow value is provided", () => {
    const env = { MCP_TIMEOUT: "42" };
    applyClaudeRuntimeTimeouts(env);
    expect(env).toEqual({ MCP_TIMEOUT: "42" });
  });
  it.each(["0", "-1", "abc", "${{ inputs.timeout }}", "1.5", "2147484"])("rejects invalid resolved timeout %s", raw => {
    expect(() => timeoutMilliseconds(raw, "tools.timeout")).toThrow("tools.timeout");
  });
  it("configures the gateway and its health budget from evaluated input", () => {
    const env = { GH_AW_STARTUP_TIMEOUT: "180", GH_AW_TOOL_TIMEOUT: "90" };
    const config = { gateway: { startupTimeout: 120 } };
    applyGatewayRuntimeTimeouts(config, env);
    expect(config.gateway).toEqual({ startupTimeout: 180, toolTimeout: 90 });
    expect(env.GH_AW_MCP_GATEWAY_BACKEND_STARTUP_TIMEOUT_MS).toBe("180000");
  });
  it("sets gateway.toolTimeout as an integer number when GH_AW_TOOL_TIMEOUT is provided", () => {
    const config = { gateway: {} };
    applyGatewayRuntimeTimeouts(config, { GH_AW_TOOL_TIMEOUT: "900" });
    expect(typeof config.gateway.toolTimeout).toBe("number");
    expect(config.gateway.toolTimeout).toBe(900);
  });
  it("preserves an existing engine.mcp.tool-timeout value", () => {
    const config = { gateway: { toolTimeout: 120 } };
    applyGatewayRuntimeTimeouts(config, { GH_AW_TOOL_TIMEOUT: "900" });
    expect(config.gateway.toolTimeout).toBe(120);
  });
  it("preserves the more specific gateway tool timeout", () => {
    const config = { gateway: { toolTimeout: "2m" } };
    applyGatewayRuntimeTimeouts(config, { GH_AW_TOOL_TIMEOUT: "90" });
    expect(config.gateway.toolTimeout).toBe("2m");
  });
  it("validates the patched config against mcp-gateway-config.schema.json", () => {
    const config = {
      mcpServers: {},
      gateway: {
        port: 8080,
        domain: "localhost",
        agentId: "test-agent",
      },
    };
    applyGatewayRuntimeTimeouts(config, { GH_AW_STARTUP_TIMEOUT: "180", GH_AW_TOOL_TIMEOUT: "900" });
    const valid = validateGatewayConfig(config);
    expect(validateGatewayConfig.errors).toBeNull();
    expect(valid).toBe(true);
    expect(typeof config.gateway.toolTimeout).toBe("number");
    expect(config.gateway.toolTimeout).toBe(900);

    const invalidConfig = {
      mcpServers: {},
      gateway: {
        port: 8080,
        domain: "localhost",
        agentId: "test-agent",
        toolTimeout: "900s",
      },
    };
    expect(validateGatewayConfig(invalidConfig)).toBe(false);
    expect(validateGatewayConfig.errors).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          instancePath: "/gateway/toolTimeout",
          keyword: "type",
          params: { type: "integer" },
        }),
      ])
    );
  });
});
