import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import fs from "fs";

const OUTPUT_DIR = "/tmp/gh-aw";
const OUTPUT_PATH = "/tmp/gh-aw/evals.jsonl";
const MCP_CONFIG_DIR = `${process.env.RUNNER_TEMP || "/tmp"}/gh-aw/mcp-config`;
const GATEWAY_OUTPUT_PATH = `${MCP_CONFIG_DIR}/gateway-output.json`;

const mockCore = {
  info: vi.fn(),
  warning: vi.fn(),
  setFailed: vi.fn(),
};

global.core = mockCore;

describe("redact_evals_results.cjs", () => {
  let module;

  beforeEach(async () => {
    vi.clearAllMocks();
    fs.mkdirSync(OUTPUT_DIR, { recursive: true });
    if (fs.existsSync(OUTPUT_PATH)) {
      fs.unlinkSync(OUTPUT_PATH);
    }
    delete process.env.GH_AW_SECRET_NAMES;
    delete process.env.SECRET_EVALS_SECRET;
    module = await import("./redact_evals_results.cjs");
  });

  afterEach(() => {
    if (fs.existsSync(OUTPUT_PATH)) {
      fs.unlinkSync(OUTPUT_PATH);
    }
    if (fs.existsSync(GATEWAY_OUTPUT_PATH)) {
      fs.unlinkSync(GATEWAY_OUTPUT_PATH);
    }
  });

  it("collects eval secret values from the workflow env", () => {
    process.env.GH_AW_SECRET_NAMES = "EVALS_SECRET, EMPTY_SECRET";
    process.env.SECRET_EVALS_SECRET = "secret-value";
    process.env.SECRET_EMPTY_SECRET = "   ";

    expect(module.getSecretValues()).toContain("secret-value");
    expect(module.getSecretValues()).not.toContain("");
  });

  it("collects MCP gateway tokens from the canonical config path", () => {
    const tokenPart = "gateway-token-abc123";
    const bearerToken = ["Bearer", tokenPart].join(" ");
    fs.mkdirSync(MCP_CONFIG_DIR, { recursive: true });
    fs.writeFileSync(
      GATEWAY_OUTPUT_PATH,
      JSON.stringify({
        mcpServers: {
          github: {
            headers: { Authorization: bearerToken },
          },
        },
      }),
      "utf8"
    );

    expect(module.getSecretValues()).toContain(bearerToken);
    expect(module.getSecretValues()).toContain(tokenPart);
  });

  it("redacts eval output and leaves no lingering secrets", async () => {
    process.env.GH_AW_SECRET_NAMES = "EVALS_SECRET";
    process.env.SECRET_EVALS_SECRET = "super-secret-value";
    fs.writeFileSync(OUTPUT_PATH, `{"question":"Contains super-secret-value"}\n`, "utf8");

    await module.main();

    expect(fs.readFileSync(OUTPUT_PATH, "utf8")).toContain("***REDACTED***");
    expect(mockCore.setFailed).not.toHaveBeenCalled();
  });

  it("fails closed when verification still detects secrets", () => {
    process.env.GH_AW_SECRET_NAMES = "EVALS_SECRET";
    process.env.SECRET_EVALS_SECRET = "super-secret-value";
    fs.writeFileSync(OUTPUT_PATH, `{"question":"Contains super-secret-value"}\n`, "utf8");

    module.verifyRedaction();

    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Secret redaction verification failed"));
  });
});
